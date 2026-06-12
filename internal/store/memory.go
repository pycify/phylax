package store

import (
	"context"
	"sync"
	"time"

	"github.com/pycify/phylax/internal/phylax"
)

type MemoryStore struct {
	mu         sync.RWMutex
	challenges map[string]phylax.Challenge
	sessions   map[string]phylax.AttestationSession
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		challenges: map[string]phylax.Challenge{},
		sessions:   map[string]phylax.AttestationSession{},
	}
}

func (s *MemoryStore) CreateChallenge(_ context.Context, challenge phylax.Challenge) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.challenges[challenge.ID] = challenge
	return nil
}

func (s *MemoryStore) ConsumeChallenge(_ context.Context, id string) (*phylax.Challenge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	challenge, ok := s.challenges[id]
	if !ok {
		return nil, phylax.ErrNotFound
	}
	if !challenge.ConsumedAt.IsZero() {
		return nil, phylax.ErrNotFound
	}
	challenge.ConsumedAt = time.Now().UTC()
	s.challenges[id] = challenge
	return &challenge, nil
}

func (s *MemoryStore) SaveSession(_ context.Context, session phylax.AttestationSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
	return nil
}

func (s *MemoryStore) FindSession(_ context.Context, id string) (*phylax.AttestationSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[id]
	if !ok {
		return nil, phylax.ErrNotFound
	}
	return &session, nil
}
