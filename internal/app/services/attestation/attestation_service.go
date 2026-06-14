package attestation

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	attestationstore "github.com/pycify/phylax/internal/app/services/store"
	"github.com/pycify/phylax/internal/config"
)

var ErrNotFound = attestationstore.ErrNotFound

type AppResolver interface {
	ResolveApp(ctx context.Context, appID string) (config.AppConfig, error)
}

type Provider interface {
	Platform() Platform
	Name() string
	Status() ProviderStatus
	Verify(ctx context.Context, app config.AppConfig, challenge Challenge, req VerifyAttestationRequest) (Verdict, error)
}

type Service struct {
	cfg       config.Config
	store     attestationstore.Store
	logger    *slog.Logger
	apps      AppResolver
	providers map[Platform]Provider
}

func NewService(cfg config.Config, st attestationstore.Store, logger *slog.Logger, apps AppResolver) (*Service, error) {
	return &Service{
		cfg:    cfg,
		store:  st,
		logger: logger,
		apps:   apps,
		providers: map[Platform]Provider{
			PlatformAndroid:  AndroidProvider{},
			PlatformApple:    AppleProvider{},
			PlatformFirebase: FirebaseProvider{},
		},
	}, nil
}

func (s *Service) CreateChallenge(ctx context.Context, req CreateChallengeRequest) (CreateChallengeResponse, error) {
	if req.AppID == "" || req.Action == "" || req.InstallID == "" {
		return CreateChallengeResponse{}, errors.New("app_id, action, and install_id are required")
	}
	app, err := s.resolveApp(ctx, req.AppID)
	if err != nil {
		return CreateChallengeResponse{}, err
	}
	req.Platform, err = inferPlatform(app)
	if err != nil {
		return CreateChallengeResponse{}, err
	}
	if err := ensurePlatformEnabled(app, req.Platform); err != nil {
		return CreateChallengeResponse{}, err
	}

	now := time.Now().UTC()
	challenge := Challenge{
		ID:        uuid.NewString(),
		AppID:     req.AppID,
		Platform:  req.Platform,
		Action:    req.Action,
		InstallID: req.InstallID,
		Nonce:     randomNonce(32),
		CreatedAt: now,
		ExpiresAt: now.Add(time.Duration(s.cfg.Challenges.TTLSeconds) * time.Second),
	}
	if err := s.store.CreateChallenge(ctx, challengeToStore(challenge)); err != nil {
		return CreateChallengeResponse{}, err
	}
	return CreateChallengeResponse{
		ChallengeID:        challenge.ID,
		AppID:              challenge.AppID,
		Platform:           challenge.Platform,
		Action:             challenge.Action,
		Nonce:              challenge.Nonce,
		CloudProjectNumber: app.Android.CloudProjectNumber,
		ExpiresAt:          challenge.ExpiresAt,
	}, nil
}

func (s *Service) VerifyAttestation(ctx context.Context, req VerifyAttestationRequest) (VerifyAttestationResponse, error) {
	if req.AppID == "" || req.ChallengeID == "" || req.InstallID == "" {
		return VerifyAttestationResponse{}, errors.New("app_id, challenge_id, and install_id are required")
	}
	return s.verifyAndIssueSession(ctx, req, "")
}

func (s *Service) RefreshAttestation(ctx context.Context, req RefreshAttestationRequest) (VerifyAttestationResponse, error) {
	if req.AppID == "" || req.CurrentToken == "" || req.ChallengeID == "" || req.InstallID == "" {
		return VerifyAttestationResponse{}, errors.New("app_id, current_token, challenge_id, and install_id are required")
	}

	claims, err := parseSessionToken(s.cfg.Token.Secret, s.cfg.Token.Issuer, req.CurrentToken)
	if err != nil {
		return VerifyAttestationResponse{}, errors.New("current token is invalid")
	}

	storedCurrentSession, err := s.store.FindSession(ctx, claims.SessionID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return VerifyAttestationResponse{}, errors.New("current session not found")
		}
		return VerifyAttestationResponse{}, err
	}
	currentSession := sessionFromStore(*storedCurrentSession)
	if time.Now().UTC().After(currentSession.ExpiresAt) {
		return VerifyAttestationResponse{}, errors.New("current session expired")
	}
	if currentSession.AppID != req.AppID || currentSession.InstallID != req.InstallID {
		return VerifyAttestationResponse{}, errors.New("current session does not match app or install")
	}

	verifyRequest := VerifyAttestationRequest{
		AppID:       req.AppID,
		ChallengeID: req.ChallengeID,
		InstallID:   req.InstallID,
		Subject:     req.Subject,
		Token:       req.Token,
		KeyID:       req.KeyID,
		Proof:       req.Proof,
	}
	if verifyRequest.Subject == "" {
		verifyRequest.Subject = currentSession.Subject
	}

	return s.verifyAndIssueSession(ctx, verifyRequest, currentSession.ID)
}

func (s *Service) verifyAndIssueSession(ctx context.Context, req VerifyAttestationRequest, previousSessionID string) (VerifyAttestationResponse, error) {
	app, err := s.resolveApp(ctx, req.AppID)
	if err != nil {
		return VerifyAttestationResponse{}, err
	}
	req.Platform, err = inferPlatform(app)
	if err != nil {
		return VerifyAttestationResponse{}, err
	}
	provider, ok := s.providers[req.Platform]
	if !ok {
		return VerifyAttestationResponse{}, fmt.Errorf("platform %q is not supported", req.Platform)
	}

	storedChallenge, err := s.store.ConsumeChallenge(ctx, req.ChallengeID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return VerifyAttestationResponse{}, errors.New("challenge not found or already consumed")
		}
		return VerifyAttestationResponse{}, err
	}
	challenge := challengeFromStore(*storedChallenge)
	if time.Now().UTC().After(challenge.ExpiresAt) {
		return VerifyAttestationResponse{}, errors.New("challenge expired")
	}
	if challenge.AppID != req.AppID || challenge.Platform != req.Platform || challenge.InstallID != req.InstallID {
		return VerifyAttestationResponse{}, errors.New("challenge does not match request")
	}

	verdict, err := provider.Verify(ctx, app, challenge, req)
	if err != nil {
		return VerifyAttestationResponse{}, err
	}

	now := time.Now().UTC()
	session := AttestationSession{
		ID:        uuid.NewString(),
		AppID:     req.AppID,
		Platform:  req.Platform,
		Action:    challenge.Action,
		InstallID: req.InstallID,
		Subject:   req.Subject,
		RiskLevel: verdict.RiskLevel,
		IssuedAt:  now,
		ExpiresAt: now.Add(time.Duration(s.cfg.Token.TTLSeconds) * time.Second),
	}
	if err := s.store.SaveSession(ctx, sessionToStore(session)); err != nil {
		return VerifyAttestationResponse{}, err
	}

	token, err := signSessionToken(s.cfg.Token.Secret, s.cfg.Token.Issuer, session)
	if err != nil {
		return VerifyAttestationResponse{}, err
	}
	if previousSessionID != "" {
		if err := s.store.RevokeSession(ctx, previousSessionID, now); err != nil {
			return VerifyAttestationResponse{}, err
		}
	}

	return VerifyAttestationResponse{
		Token:     token,
		Session:   session,
		Verdict:   verdict,
		ExpiresAt: session.ExpiresAt,
	}, nil
}

func (s *Service) resolveApp(ctx context.Context, appID string) (config.AppConfig, error) {
	if s.apps == nil {
		return config.AppConfig{}, errors.New("app resolver is not configured")
	}
	app, err := s.apps.ResolveApp(ctx, appID)
	if err != nil {
		return config.AppConfig{}, err
	}
	return app, nil
}

func inferPlatform(app config.AppConfig) (Platform, error) {
	if app.Firebase.Enabled {
		return PlatformFirebase, nil
	}
	if app.Android.Enabled {
		return PlatformAndroid, nil
	}
	if app.Apple.Enabled {
		return PlatformApple, nil
	}
	return "", fmt.Errorf("app %q has no enabled attestation provider", app.ID)
}

func (s *Service) Introspect(ctx context.Context, token string) IntrospectResponse {
	claims, err := parseSessionToken(s.cfg.Token.Secret, s.cfg.Token.Issuer, token)
	if err != nil {
		return IntrospectResponse{Active: false, Error: err.Error()}
	}

	storedSession, err := s.store.FindSession(ctx, claims.SessionID)
	if err != nil {
		return IntrospectResponse{Active: false, Error: "session not found"}
	}
	session := sessionFromStore(*storedSession)
	if time.Now().UTC().After(session.ExpiresAt) {
		return IntrospectResponse{Active: false, Error: "session expired"}
	}
	return IntrospectResponse{Active: true, Session: &session}
}

func ensurePlatformEnabled(app config.AppConfig, platform Platform) error {
	switch platform {
	case PlatformAndroid:
		if !app.Android.Enabled {
			return errors.New("android attestation is not enabled for this app")
		}
	case PlatformApple:
		if !app.Apple.Enabled {
			return errors.New("apple attestation is not enabled for this app")
		}
	case PlatformFirebase:
		if !app.Firebase.Enabled {
			return errors.New("firebase app check is not enabled for this app")
		}
	default:
		return fmt.Errorf("platform %q is not supported", platform)
	}
	return nil
}

func randomNonce(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

func challengeToStore(challenge Challenge) attestationstore.Challenge {
	return attestationstore.Challenge{
		ID:         challenge.ID,
		AppID:      challenge.AppID,
		Platform:   attestationstore.Platform(challenge.Platform),
		Action:     challenge.Action,
		InstallID:  challenge.InstallID,
		Nonce:      challenge.Nonce,
		ExpiresAt:  challenge.ExpiresAt,
		CreatedAt:  challenge.CreatedAt,
		ConsumedAt: challenge.ConsumedAt,
	}
}

func challengeFromStore(challenge attestationstore.Challenge) Challenge {
	return Challenge{
		ID:         challenge.ID,
		AppID:      challenge.AppID,
		Platform:   Platform(challenge.Platform),
		Action:     challenge.Action,
		InstallID:  challenge.InstallID,
		Nonce:      challenge.Nonce,
		ExpiresAt:  challenge.ExpiresAt,
		CreatedAt:  challenge.CreatedAt,
		ConsumedAt: challenge.ConsumedAt,
	}
}

func sessionToStore(session AttestationSession) attestationstore.AttestationSession {
	return attestationstore.AttestationSession{
		ID:        session.ID,
		AppID:     session.AppID,
		Platform:  attestationstore.Platform(session.Platform),
		Action:    session.Action,
		InstallID: session.InstallID,
		Subject:   session.Subject,
		RiskLevel: session.RiskLevel,
		IssuedAt:  session.IssuedAt,
		ExpiresAt: session.ExpiresAt,
		RevokedAt: session.RevokedAt,
	}
}

func sessionFromStore(session attestationstore.AttestationSession) AttestationSession {
	return AttestationSession{
		ID:        session.ID,
		AppID:     session.AppID,
		Platform:  Platform(session.Platform),
		Action:    session.Action,
		InstallID: session.InstallID,
		Subject:   session.Subject,
		RiskLevel: session.RiskLevel,
		IssuedAt:  session.IssuedAt,
		ExpiresAt: session.ExpiresAt,
		RevokedAt: session.RevokedAt,
	}
}
