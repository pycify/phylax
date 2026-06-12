package phylax

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"

	"github.com/pycify/phylax/internal/config"
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
		Platform:    PlatformAndroid,
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
		Platform:    PlatformAndroid,
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
		Apps: []config.AppConfig{{
			ID:   "demo",
			Name: "Demo",
			Android: config.AndroidConfig{
				Enabled:               true,
				PackageName:           "com.example.demo",
				AllowMockVerification: true,
			},
		}},
	}, newMemoryTestStore(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return service
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
