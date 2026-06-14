package attestation

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pycify/phylax/internal/config"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/api/playintegrity/v1"
)

type PlayIntegrityVerifier interface {
	Decode(ctx context.Context, app config.AppConfig, integrityToken string) (*playintegrity.TokenPayloadExternal, error)
}

type GooglePlayIntegrityVerifier struct{}

func (GooglePlayIntegrityVerifier) Decode(ctx context.Context, app config.AppConfig, integrityToken string) (*playintegrity.TokenPayloadExternal, error) {
	credentialsJSON := strings.TrimSpace(app.Android.ServiceAccountJSON)
	if credentialsJSON == "" {
		return nil, errors.New("android service account JSON is required for Play Integrity verification")
	}

	creds, err := google.CredentialsFromJSONWithTypeAndParams(
		ctx,
		[]byte(credentialsJSON),
		google.ServiceAccount,
		google.CredentialsParams{Scopes: []string{playintegrity.PlayintegrityScope}},
	)
	if err != nil {
		return nil, err
	}

	service, err := playintegrity.NewService(ctx,
		option.WithCredentials(creds),
	)
	if err != nil {
		return nil, err
	}

	response, err := service.V1.DecodeIntegrityToken(app.Android.PackageName, &playintegrity.DecodeIntegrityTokenRequest{
		IntegrityToken: integrityToken,
	}).Context(ctx).Do()
	if err != nil {
		var gerr *googleapi.Error
		if errors.As(err, &gerr) && gerr.Code == 403 {
			if strings.Contains(strings.ToLower(gerr.Message), "not authorized to decode") {
				return nil, fmt.Errorf("play integrity decode forbidden: ensure the app is linked to the same Google Cloud project number, Play Integrity API is enabled in that project, and the configured service account has permission to decode tokens")
			}
		}
		return nil, err
	}
	if response == nil || response.TokenPayloadExternal == nil {
		return nil, errors.New("play integrity response did not include a token payload")
	}
	return response.TokenPayloadExternal, nil
}

type AndroidProvider struct {
	verifier PlayIntegrityVerifier
}

func (AndroidProvider) Platform() Platform { return PlatformAndroid }
func (AndroidProvider) Name() string       { return "play_integrity" }
func (AndroidProvider) Status() ProviderStatus {
	return ProviderStatus{Platform: string(PlatformAndroid), Provider: "play_integrity"}
}

func (p AndroidProvider) Verify(ctx context.Context, app config.AppConfig, challenge Challenge, req VerifyAttestationRequest) (Verdict, error) {
	if err := ensurePlatformEnabled(app, PlatformAndroid); err != nil {
		return Verdict{}, err
	}

	if app.Android.AllowMockVerification {
		return Verdict{
			Platform:       PlatformAndroid,
			AppID:          app.ID,
			InstallID:      req.InstallID,
			Action:         challenge.Action,
			RiskLevel:      "low",
			Provider:       "play_integrity_mock",
			ChallengeID:    challenge.ID,
			ChallengeNonce: challenge.Nonce,
			AttestedAt:     time.Now().UTC(),
			Claims: map[string]any{
				"package_name": app.Android.PackageName,
				"mock":         true,
			},
		}, nil
	}

	if req.Token == "" {
		return Verdict{}, errors.New("android integrity token is required")
	}

	verifier := p.verifier
	if verifier == nil {
		verifier = GooglePlayIntegrityVerifier{}
	}
	payload, err := verifier.Decode(ctx, app, req.Token)
	if err != nil {
		return Verdict{}, err
	}
	if err := validatePlayIntegrityPayload(app, challenge, payload); err != nil {
		return Verdict{}, err
	}

	return playIntegrityVerdict(app, challenge, req, payload), nil
}

func validatePlayIntegrityPayload(app config.AppConfig, challenge Challenge, payload *playintegrity.TokenPayloadExternal) error {
	if payload == nil {
		return errors.New("play integrity token payload is required")
	}
	if payload.RequestDetails == nil {
		return errors.New("play integrity request details are required")
	}
	if payload.RequestDetails.RequestPackageName != app.Android.PackageName {
		return fmt.Errorf("play integrity request package %q does not match configured package %q", payload.RequestDetails.RequestPackageName, app.Android.PackageName)
	}
	if payload.RequestDetails.Nonce != "" {
		if !equivalentNonce(payload.RequestDetails.Nonce, challenge.Nonce) {
			return errors.New("play integrity nonce does not match challenge")
		}
	} else if payload.RequestDetails.RequestHash != "" {
		if !equivalentNonce(payload.RequestDetails.RequestHash, challenge.Nonce) {
			return errors.New("play integrity request_hash does not match challenge")
		}
	} else {
		return errors.New("play integrity request details must include nonce or request_hash")
	}
	if err := validatePlayIntegrityTimestamp(challenge, payload.RequestDetails.TimestampMillis); err != nil {
		return err
	}

	if payload.AppIntegrity == nil {
		return errors.New("play integrity app integrity verdict is required")
	}
	if payload.AppIntegrity.PackageName != "" && payload.AppIntegrity.PackageName != app.Android.PackageName {
		return fmt.Errorf("play integrity app package %q does not match configured package %q", payload.AppIntegrity.PackageName, app.Android.PackageName)
	}
	// TODO
	// if app.Android.RequirePlayRecognized && payload.AppIntegrity.AppRecognitionVerdict != "PLAY_RECOGNIZED" {
	// 	return fmt.Errorf("play integrity app recognition verdict %q is not PLAY_RECOGNIZED", payload.AppIntegrity.AppRecognitionVerdict)
	// }
	if len(app.Android.ExpectedCertificateSHA256) > 0 && !hasAnyString(payload.AppIntegrity.CertificateSha256Digest, app.Android.ExpectedCertificateSHA256) {
		return errors.New("play integrity certificate digest does not match configured certificates")
	}
	if len(app.Android.AllowedVersionCodePrefixes) > 0 && !hasAnyPrefix(strconv.FormatInt(payload.AppIntegrity.VersionCode, 10), app.Android.AllowedVersionCodePrefixes) {
		return fmt.Errorf("play integrity app version_code %d is not allowed", payload.AppIntegrity.VersionCode)
	}

	if len(app.Android.RequiredDeviceIntegrity) > 0 {
		if payload.DeviceIntegrity == nil {
			return errors.New("play integrity device integrity verdict is required")
		}
		for _, required := range app.Android.RequiredDeviceIntegrity {
			if !hasString(payload.DeviceIntegrity.DeviceRecognitionVerdict, required) {
				return fmt.Errorf("play integrity device verdict does not include %q", required)
			}
		}
	}
	return nil
}

func validatePlayIntegrityTimestamp(challenge Challenge, timestampMillis int64) error {
	if timestampMillis == 0 {
		return errors.New("play integrity request timestamp is required")
	}
	requestedAt := time.UnixMilli(timestampMillis).UTC()
	now := time.Now().UTC()
	if requestedAt.Before(challenge.CreatedAt.Add(-2 * time.Minute)) {
		return errors.New("play integrity token was requested before challenge creation")
	}
	if requestedAt.After(now.Add(2 * time.Minute)) {
		return errors.New("play integrity token timestamp is in the future")
	}
	return nil
}

func playIntegrityVerdict(app config.AppConfig, challenge Challenge, req VerifyAttestationRequest, payload *playintegrity.TokenPayloadExternal) Verdict {
	claims := map[string]any{
		"package_name":            payload.AppIntegrity.PackageName,
		"app_recognition_verdict": payload.AppIntegrity.AppRecognitionVerdict,
		"certificate_sha256":      payload.AppIntegrity.CertificateSha256Digest,
		"version_code":            payload.AppIntegrity.VersionCode,
		"request_package_name":    payload.RequestDetails.RequestPackageName,
		"timestamp_millis":        payload.RequestDetails.TimestampMillis,
	}
	if payload.RequestDetails.Nonce != "" {
		claims["nonce"] = payload.RequestDetails.Nonce
	}
	if payload.RequestDetails.RequestHash != "" {
		claims["request_hash"] = payload.RequestDetails.RequestHash
	}
	if payload.DeviceIntegrity != nil {
		claims["device_recognition_verdict"] = payload.DeviceIntegrity.DeviceRecognitionVerdict
	}
	if payload.AccountDetails != nil {
		claims["app_licensing_verdict"] = payload.AccountDetails.AppLicensingVerdict
	}
	if payload.TestingDetails != nil {
		claims["testing_response"] = payload.TestingDetails.IsTestingResponse
	}

	return Verdict{
		Platform:       PlatformAndroid,
		AppID:          app.ID,
		InstallID:      req.InstallID,
		Action:         challenge.Action,
		RiskLevel:      "low",
		Provider:       "play_integrity",
		Claims:         claims,
		ChallengeID:    challenge.ID,
		ChallengeNonce: challenge.Nonce,
		AttestedAt:     time.Now().UTC(),
	}
}

func hasString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func hasAnyString(values []string, expected []string) bool {
	for _, value := range values {
		for _, candidate := range expected {
			if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(candidate)) {
				return true
			}
		}
	}
	return false
}

func hasAnyPrefix(value string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, strings.TrimSpace(prefix)) {
			return true
		}
	}
	return false
}

func equivalentNonce(actual string, expected string) bool {
	if actual == expected {
		return true
	}
	actualNormalized, okActual := normalizeNonce(actual)
	expectedNormalized, okExpected := normalizeNonce(expected)
	if !okActual || !okExpected {
		return false
	}
	return actualNormalized == expectedNormalized
}

func normalizeNonce(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	if decoded, err := base64.RawURLEncoding.DecodeString(trimmed); err == nil {
		return base64.RawURLEncoding.EncodeToString(decoded), true
	}
	if decoded, err := base64.URLEncoding.DecodeString(trimmed); err == nil {
		return base64.RawURLEncoding.EncodeToString(decoded), true
	}
	if decoded, err := base64.StdEncoding.DecodeString(trimmed); err == nil {
		return base64.RawURLEncoding.EncodeToString(decoded), true
	}
	return "", false
}

func maskValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) <= 8 {
		return "***"
	}
	return trimmed[:4] + "..." + trimmed[len(trimmed)-4:]
}

type AppleProvider struct{}

func (AppleProvider) Platform() Platform { return PlatformApple }
func (AppleProvider) Name() string       { return "app_attest" }
func (AppleProvider) Status() ProviderStatus {
	return ProviderStatus{Platform: string(PlatformApple), Provider: "app_attest"}
}

func (AppleProvider) Verify(_ context.Context, app config.AppConfig, challenge Challenge, req VerifyAttestationRequest) (Verdict, error) {
	if err := ensurePlatformEnabled(app, PlatformApple); err != nil {
		return Verdict{}, err
	}

	if app.Apple.AllowMockVerification {
		return Verdict{
			Platform:       PlatformApple,
			AppID:          app.ID,
			InstallID:      req.InstallID,
			Action:         challenge.Action,
			RiskLevel:      "low",
			Provider:       "app_attest_mock",
			ChallengeID:    challenge.ID,
			ChallengeNonce: challenge.Nonce,
			AttestedAt:     time.Now().UTC(),
			Claims: map[string]any{
				"bundle_id": app.Apple.BundleID,
				"team_id":   app.Apple.TeamID,
				"mock":      true,
			},
		}, nil
	}

	if req.KeyID == "" || req.Token == "" {
		return Verdict{}, errors.New("apple key_id and assertion token are required")
	}

	return Verdict{}, fmt.Errorf("apple app attest verification is not implemented yet for app %q", app.ID)
}

type FirebaseProvider struct{}

func (FirebaseProvider) Platform() Platform { return PlatformFirebase }
func (FirebaseProvider) Name() string       { return "firebase_app_check" }
func (FirebaseProvider) Status() ProviderStatus {
	return ProviderStatus{Platform: string(PlatformFirebase), Provider: "firebase_app_check"}
}

func (FirebaseProvider) Verify(_ context.Context, app config.AppConfig, challenge Challenge, req VerifyAttestationRequest) (Verdict, error) {
	if err := ensurePlatformEnabled(app, PlatformFirebase); err != nil {
		return Verdict{}, err
	}
	if req.Token == "" {
		return Verdict{}, errors.New("firebase app check token is required")
	}
	return Verdict{}, fmt.Errorf("firebase app check verification is not implemented yet for app %q", app.ID)
}
