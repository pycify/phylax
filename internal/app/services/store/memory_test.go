package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pycify/phylax/internal/config"
)

func TestMemoryStoreChallengeLifecycle(t *testing.T) {
	st := NewMemoryStore()
	challenge := Challenge{
		ID:        "chl_123",
		AppID:     "demo",
		Platform:  PlatformAndroid,
		Action:    "ai.ask",
		InstallID: "install_123",
		Nonce:     "nonce",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(time.Minute),
	}

	if err := st.CreateChallenge(context.Background(), challenge); err != nil {
		t.Fatal(err)
	}
	consumed, err := st.ConsumeChallenge(context.Background(), challenge.ID)
	if err != nil {
		t.Fatal(err)
	}
	if consumed.ID != challenge.ID {
		t.Fatalf("expected challenge %q, got %q", challenge.ID, consumed.ID)
	}
	if _, err := st.ConsumeChallenge(context.Background(), challenge.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected reused challenge not found, got %v", err)
	}
}

func TestMemoryStoreSessionLifecycle(t *testing.T) {
	st := NewMemoryStore()
	session := AttestationSession{
		ID:        "ses_123",
		AppID:     "demo",
		Platform:  PlatformAndroid,
		Action:    "ai.ask",
		InstallID: "install_123",
		RiskLevel: "low",
		IssuedAt:  time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(time.Minute),
	}
	if err := st.SaveSession(context.Background(), session); err != nil {
		t.Fatal(err)
	}
	found, err := st.FindSession(context.Background(), session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.ID != session.ID {
		t.Fatalf("expected session %q, got %q", session.ID, found.ID)
	}
}

func TestFactoryCreatesMemoryStore(t *testing.T) {
	st, err := New(config.StorageConfig{Driver: "memory"})
	if err != nil {
		t.Fatal(err)
	}
	if st == nil {
		t.Fatal("expected store")
	}
}

func TestFactoryRejectsUnknownDriver(t *testing.T) {
	if _, err := New(config.StorageConfig{Driver: "unknown"}); err == nil {
		t.Fatal("expected unknown driver error")
	}
}
