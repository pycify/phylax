package phylax

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/pycify/phylax/internal/config"
)

type AndroidProvider struct{}

func (AndroidProvider) Platform() Platform { return PlatformAndroid }
func (AndroidProvider) Name() string       { return "play_integrity" }
func (AndroidProvider) Status() ProviderStatus {
	return ProviderStatus{Platform: string(PlatformAndroid), Provider: "play_integrity"}
}

func (AndroidProvider) Verify(_ context.Context, app config.AppConfig, challenge Challenge, req VerifyAttestationRequest) (Verdict, error) {
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

	return Verdict{}, fmt.Errorf("play integrity verification is not implemented yet for app %q", app.ID)
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
