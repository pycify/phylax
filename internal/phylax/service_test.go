package phylax

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/pycify/phylax/internal/config"
	"google.golang.org/api/playintegrity/v1"
)

func TestServiceCreatesChallengeAndVerifiesMockAndroidAttestation(t *testing.T) {
	service := newTestService(t)

	challenge, err := service.CreateChallenge(context.Background(), CreateChallengeRequest{
		AppID:     "demo",
		Platform:  PlatformAndroid,
		Action:    "document_ai.ask_question",
		InstallID: "install_123",
	})
	if err != nil {
		t.Fatal(err)
	}

	verification, err := service.VerifyAttestation(context.Background(), VerifyAttestationRequest{
		AppID:       "demo",
		ChallengeID: challenge.ChallengeID,
		InstallID:   "install_123",
		Subject:     "user_123",
		Token:       "mock",
	})
	if err != nil {
		t.Fatal(err)
	}
	if verification.Token == "" {
		t.Fatal("expected token")
	}

	result := service.Introspect(context.Background(), verification.Token)
	if !result.Active {
		t.Fatalf("expected token to be active: %s", result.Error)
	}
	if result.Session.Subject != "user_123" {
		t.Fatalf("expected subject user_123, got %q", result.Session.Subject)
	}
}

func TestServiceRejectsReusedChallenge(t *testing.T) {
	service := newTestService(t)
	challenge, err := service.CreateChallenge(context.Background(), CreateChallengeRequest{
		AppID:     "demo",
		Platform:  PlatformAndroid,
		Action:    "document_ai.ask_question",
		InstallID: "install_123",
	})
	if err != nil {
		t.Fatal(err)
	}
	request := VerifyAttestationRequest{
		AppID:       "demo",
		ChallengeID: challenge.ChallengeID,
		InstallID:   "install_123",
		Token:       "mock",
	}
	if _, err := service.VerifyAttestation(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if _, err := service.VerifyAttestation(context.Background(), request); err == nil {
		t.Fatal("expected reused challenge to be rejected")
	}
}

func TestServiceVerifiesPlayIntegrityVerdict(t *testing.T) {
	service := newRealAndroidTestService(t, fakePlayIntegrityVerifier{})
	challenge, err := service.CreateChallenge(context.Background(), CreateChallengeRequest{
		AppID:     "demo",
		Platform:  PlatformAndroid,
		Action:    "document_ai.ask_question",
		InstallID: "install_123",
	})
	if err != nil {
		t.Fatal(err)
	}

	verification, err := service.VerifyAttestation(context.Background(), VerifyAttestationRequest{
		AppID:       "demo",
		ChallengeID: challenge.ChallengeID,
		InstallID:   "install_123",
		Token:       challenge.Nonce,
	})
	if err != nil {
		t.Fatal(err)
	}
	if verification.Verdict.Provider != "play_integrity" {
		t.Fatalf("expected play_integrity provider, got %q", verification.Verdict.Provider)
	}
	if verification.Verdict.Claims["app_recognition_verdict"] != "PLAY_RECOGNIZED" {
		t.Fatalf("expected PLAY_RECOGNIZED claim, got %#v", verification.Verdict.Claims["app_recognition_verdict"])
	}
}

func TestServiceRejectsPlayIntegrityPackageMismatch(t *testing.T) {
	service := newRealAndroidTestService(t, fakePlayIntegrityVerifier{requestPackageName: "com.attacker.app"})
	challenge, err := service.CreateChallenge(context.Background(), CreateChallengeRequest{
		AppID:     "demo",
		Platform:  PlatformAndroid,
		Action:    "document_ai.ask_question",
		InstallID: "install_123",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.VerifyAttestation(context.Background(), VerifyAttestationRequest{
		AppID:       "demo",
		ChallengeID: challenge.ChallengeID,
		InstallID:   "install_123",
		Token:       challenge.Nonce,
	})
	if err == nil {
		t.Fatal("expected package mismatch to be rejected")
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	service, err := NewService(config.Config{
		Environment: "development",
		Server:      config.ServerConfig{Addr: ":8080"},
		Storage:     config.StorageConfig{Driver: "memory"},
		Token: config.TokenConfig{
			Issuer:     "phylax-test",
			Secret:     "test-secret-with-more-than-32-bytes",
			TTLSeconds: 900,
		},
		Challenges: config.ChallengeConfig{TTLSeconds: 300},
	}, newMemoryTestStore(), slog.New(slog.NewTextHandler(io.Discard, nil)), staticAppResolver{apps: map[string]config.AppConfig{
		"demo": {
			ID:   "demo",
			Name: "Demo",
			Android: config.AndroidConfig{
				Enabled:               true,
				PackageName:           "com.example.demo",
				AllowMockVerification: true,
			},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func newRealAndroidTestService(t *testing.T, verifier PlayIntegrityVerifier) *Service {
	t.Helper()
	service, err := NewService(config.Config{
		Environment: "development",
		Server:      config.ServerConfig{Addr: ":8080"},
		Storage:     config.StorageConfig{Driver: "memory"},
		Token: config.TokenConfig{
			Issuer:     "phylax-test",
			Secret:     "test-secret-with-more-than-32-bytes",
			TTLSeconds: 900,
		},
		Challenges: config.ChallengeConfig{TTLSeconds: 300},
	}, newMemoryTestStore(), slog.New(slog.NewTextHandler(io.Discard, nil)), staticAppResolver{apps: map[string]config.AppConfig{
		"demo": {
			ID:   "demo",
			Name: "Demo",
			Android: config.AndroidConfig{
				Enabled:                   true,
				PackageName:               "com.example.demo",
				RequirePlayRecognized:     true,
				RequiredDeviceIntegrity:   []string{"MEETS_DEVICE_INTEGRITY"},
				ExpectedCertificateSHA256: []string{"cert-digest"},
			},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	service.providers[PlatformAndroid] = AndroidProvider{verifier: verifier}
	return service
}

type fakePlayIntegrityVerifier struct {
	requestPackageName string
}

type staticAppResolver struct {
	apps map[string]config.AppConfig
}

func (r staticAppResolver) ResolveApp(_ context.Context, appID string) (config.AppConfig, error) {
	app, ok := r.apps[appID]
	if !ok {
		return config.AppConfig{}, errors.New("app not found")
	}
	return app, nil
}

func (v fakePlayIntegrityVerifier) Decode(_ context.Context, app config.AppConfig, integrityToken string) (*playintegrity.TokenPayloadExternal, error) {
	requestPackageName := v.requestPackageName
	if requestPackageName == "" {
		requestPackageName = app.Android.PackageName
	}
	return &playintegrity.TokenPayloadExternal{
		RequestDetails: &playintegrity.RequestDetails{
			RequestPackageName: requestPackageName,
			Nonce:              integrityToken,
			TimestampMillis:    time.Now().UTC().UnixMilli(),
		},
		AppIntegrity: &playintegrity.AppIntegrity{
			AppRecognitionVerdict:   "PLAY_RECOGNIZED",
			PackageName:             app.Android.PackageName,
			CertificateSha256Digest: []string{"cert-digest"},
			VersionCode:             42,
		},
		DeviceIntegrity: &playintegrity.DeviceIntegrity{
			DeviceRecognitionVerdict: []string{"MEETS_DEVICE_INTEGRITY"},
		},
	}, nil
}

type memoryTestStore struct {
	mu         sync.Mutex
	challenges map[string]Challenge
	sessions   map[string]AttestationSession
}

func newMemoryTestStore() *memoryTestStore {
	return &memoryTestStore{
		challenges: map[string]Challenge{},
		sessions:   map[string]AttestationSession{},
	}
}

func (s *memoryTestStore) CreateChallenge(_ context.Context, challenge Challenge) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.challenges[challenge.ID] = challenge
	return nil
}

func (s *memoryTestStore) ConsumeChallenge(_ context.Context, id string) (*Challenge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	challenge, ok := s.challenges[id]
	if !ok || !challenge.ConsumedAt.IsZero() {
		return nil, ErrNotFound
	}
	challenge.ConsumedAt = challenge.CreatedAt
	s.challenges[id] = challenge
	return &challenge, nil
}

func (s *memoryTestStore) SaveSession(_ context.Context, session AttestationSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
	return nil
}

func (s *memoryTestStore) FindSession(_ context.Context, id string) (*AttestationSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return &session, nil
}
