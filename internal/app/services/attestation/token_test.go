package attestation

import (
	"strings"
	"testing"
	"time"
)

func TestSignAndParseSessionToken(t *testing.T) {
	session := AttestationSession{
		ID:        "ses_123",
		AppID:     "demo",
		Platform:  PlatformAndroid,
		Action:    "ai.ask",
		InstallID: "install_123",
		Subject:   "user_123",
		IssuedAt:  time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(time.Minute),
	}

	token, err := signSessionToken("secret-with-more-than-32-bytes", "phylax-test", session)
	if err != nil {
		t.Fatal(err)
	}

	claims, err := parseSessionToken("secret-with-more-than-32-bytes", "phylax-test", token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.SessionID != session.ID {
		t.Fatalf("expected session id %q, got %q", session.ID, claims.SessionID)
	}
}

func TestParseSessionTokenRejectsTampering(t *testing.T) {
	session := AttestationSession{
		ID:        "ses_123",
		AppID:     "demo",
		Platform:  PlatformAndroid,
		Action:    "ai.ask",
		InstallID: "install_123",
		IssuedAt:  time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(time.Minute),
	}
	token, err := signSessionToken("secret-with-more-than-32-bytes", "phylax-test", session)
	if err != nil {
		t.Fatal(err)
	}

	parts := strings.Split(token, ".")
	parts[1] = "tampered"
	if _, err := parseSessionToken("secret-with-more-than-32-bytes", "phylax-test", strings.Join(parts, ".")); err == nil {
		t.Fatal("expected tampered token to be rejected")
	}
}
