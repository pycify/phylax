package store

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("record not found")

type Platform string

const (
	PlatformAndroid  Platform = "android"
	PlatformApple    Platform = "apple"
	PlatformFirebase Platform = "firebase"
)

type Challenge struct {
	ID         string    `json:"id"`
	AppID      string    `json:"app_id"`
	Platform   Platform  `json:"platform"`
	Action     string    `json:"action"`
	InstallID  string    `json:"install_id"`
	Nonce      string    `json:"nonce"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
	ConsumedAt time.Time `json:"consumed_at,omitempty"`
}

type AttestationSession struct {
	ID        string    `json:"id"`
	AppID     string    `json:"app_id"`
	Platform  Platform  `json:"platform"`
	Action    string    `json:"action"`
	InstallID string    `json:"install_id"`
	Subject   string    `json:"subject,omitempty"`
	RiskLevel string    `json:"risk_level"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
	RevokedAt time.Time `json:"revoked_at,omitempty"`
}

type Store interface {
	CreateChallenge(ctx context.Context, challenge Challenge) error
	ConsumeChallenge(ctx context.Context, id string) (*Challenge, error)
	SaveSession(ctx context.Context, session AttestationSession) error
	FindSession(ctx context.Context, id string) (*AttestationSession, error)
	RevokeSession(ctx context.Context, id string, revokedAt time.Time) error
}
