package phylax

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/pycify/phylax/internal/config"
)

var ErrNotFound = errors.New("record not found")

type Store interface {
	CreateChallenge(ctx context.Context, challenge Challenge) error
	ConsumeChallenge(ctx context.Context, id string) (*Challenge, error)
	SaveSession(ctx context.Context, session AttestationSession) error
	FindSession(ctx context.Context, id string) (*AttestationSession, error)
}

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
	store     Store
	logger    *slog.Logger
	apps      AppResolver
	providers map[Platform]Provider
}

func NewService(cfg config.Config, st Store, logger *slog.Logger, apps AppResolver) (*Service, error) {
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
	if err := s.store.CreateChallenge(ctx, challenge); err != nil {
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

	challenge, err := s.store.ConsumeChallenge(ctx, req.ChallengeID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return VerifyAttestationResponse{}, errors.New("challenge not found or already consumed")
		}
		return VerifyAttestationResponse{}, err
	}
	if time.Now().UTC().After(challenge.ExpiresAt) {
		return VerifyAttestationResponse{}, errors.New("challenge expired")
	}
	if challenge.AppID != req.AppID || challenge.Platform != req.Platform || challenge.InstallID != req.InstallID {
		return VerifyAttestationResponse{}, errors.New("challenge does not match request")
	}

	verdict, err := provider.Verify(ctx, app, *challenge, req)
	if err != nil {
		return VerifyAttestationResponse{}, err
	}

	fmt.Printf("Attestation verification Verdict: %+v\n", verdict)

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
	if err := s.store.SaveSession(ctx, session); err != nil {
		return VerifyAttestationResponse{}, err
	}

	token, err := signSessionToken(s.cfg.Token.Secret, s.cfg.Token.Issuer, session)
	if err != nil {
		return VerifyAttestationResponse{}, err
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

	session, err := s.store.FindSession(ctx, claims.SessionID)
	if err != nil {
		return IntrospectResponse{Active: false, Error: "session not found"}
	}
	if time.Now().UTC().After(session.ExpiresAt) {
		return IntrospectResponse{Active: false, Error: "session expired"}
	}
	return IntrospectResponse{Active: true, Session: session}
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
