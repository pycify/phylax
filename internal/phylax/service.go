package phylax

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/pycify/phylax/internal/config"
)

var ErrNotFound = errors.New("record not found")

type Store interface {
	CreateChallenge(ctx context.Context, challenge Challenge) error
	ConsumeChallenge(ctx context.Context, id string) (*Challenge, error)
	SaveSession(ctx context.Context, session AttestationSession) error
	FindSession(ctx context.Context, id string) (*AttestationSession, error)
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
	apps      map[string]config.AppConfig
	providers map[Platform]Provider
}

func NewService(cfg config.Config, st Store, logger *slog.Logger) (*Service, error) {
	apps := map[string]config.AppConfig{}
	for _, app := range cfg.Apps {
		apps[app.ID] = app
	}

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

func (s *Service) Apps() []config.AppConfig {
	apps := make([]config.AppConfig, 0, len(s.cfg.Apps))
	apps = append(apps, s.cfg.Apps...)
	return apps
}

func (s *Service) ProviderStatuses(appID string) ([]ProviderStatus, error) {
	app, ok := s.apps[appID]
	if !ok {
		return nil, fmt.Errorf("app %q not found", appID)
	}

	statuses := []ProviderStatus{}
	if provider, ok := s.providers[PlatformAndroid]; ok {
		status := provider.Status()
		status.Enabled = app.Android.Enabled
		status.Ready = app.Android.Enabled && app.Android.PackageName != ""
		if app.Android.Enabled && app.Android.ServiceAccountJSONEnv != "" && os.Getenv(app.Android.ServiceAccountJSONEnv) == "" {
			status.Ready = false
			status.Issues = append(status.Issues, "service account env is not set")
		}
		statuses = append(statuses, status)
	}
	if provider, ok := s.providers[PlatformApple]; ok {
		status := provider.Status()
		status.Enabled = app.Apple.Enabled
		status.Ready = app.Apple.Enabled && app.Apple.BundleID != "" && app.Apple.TeamID != ""
		statuses = append(statuses, status)
	}
	if provider, ok := s.providers[PlatformFirebase]; ok {
		status := provider.Status()
		status.Enabled = app.Firebase.Enabled
		status.Ready = app.Firebase.Enabled && app.Firebase.ProjectID != ""
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func (s *Service) CreateChallenge(ctx context.Context, req CreateChallengeRequest) (CreateChallengeResponse, error) {
	if req.AppID == "" || req.Platform == "" || req.Action == "" || req.InstallID == "" {
		return CreateChallengeResponse{}, errors.New("app_id, platform, action, and install_id are required")
	}
	app, ok := s.apps[req.AppID]
	if !ok {
		return CreateChallengeResponse{}, fmt.Errorf("app %q not found", req.AppID)
	}
	if err := ensurePlatformEnabled(app, req.Platform); err != nil {
		return CreateChallengeResponse{}, err
	}

	now := time.Now().UTC()
	challenge := Challenge{
		ID:        randomID("chl"),
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
		ChallengeID: challenge.ID,
		AppID:       challenge.AppID,
		Platform:    challenge.Platform,
		Action:      challenge.Action,
		Nonce:       challenge.Nonce,
		ExpiresAt:   challenge.ExpiresAt,
	}, nil
}

func (s *Service) VerifyAttestation(ctx context.Context, req VerifyAttestationRequest) (VerifyAttestationResponse, error) {
	app, ok := s.apps[req.AppID]
	if !ok {
		return VerifyAttestationResponse{}, fmt.Errorf("app %q not found", req.AppID)
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

	now := time.Now().UTC()
	session := AttestationSession{
		ID:        randomID("ses"),
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

func randomID(prefix string) string {
	return prefix + "_" + randomNonce(18)
}

func RandomID(prefix string) string {
	return randomID(prefix)
}

func randomNonce(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}
