package store

import (
	"context"

	"github.com/pycify/phylax/internal/phylax"
)

type Store interface {
	CreateChallenge(ctx context.Context, challenge phylax.Challenge) error
	ConsumeChallenge(ctx context.Context, id string) (*phylax.Challenge, error)
	SaveSession(ctx context.Context, session phylax.AttestationSession) error
	FindSession(ctx context.Context, id string) (*phylax.AttestationSession, error)
}
