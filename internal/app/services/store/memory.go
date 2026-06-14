package store

import (
	"context"
	"sync"
	"time"
)

type MemoryStore struct {
	mu         sync.RWMutex
	challenges map[string]Challenge
	sessions   map[string]AttestationSession
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		challenges: map[string]Challenge{},
		sessions:   map[string]AttestationSession{},
	}
}

func (s *MemoryStore) CreateChallenge(_ context.Context, challenge Challenge) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.challenges[challenge.ID] = challenge
	return nil
}

func (s *MemoryStore) ConsumeChallenge(_ context.Context, id string) (*Challenge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	challenge, ok := s.challenges[id]
	if !ok {
		return nil, ErrNotFound
	}
	if !challenge.ConsumedAt.IsZero() {
		return nil, ErrNotFound
	}
	challenge.ConsumedAt = time.Now().UTC()
	s.challenges[id] = challenge
	return &challenge, nil
}

func (s *MemoryStore) SaveSession(_ context.Context, session AttestationSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
	return nil
}

func (s *MemoryStore) FindSession(_ context.Context, id string) (*AttestationSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &session, nil
}
