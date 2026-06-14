package store

import (
	"context"
	"errors"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PostgresStore struct {
	db *gorm.DB
}

type challengeModel struct {
	ID         string `gorm:"primaryKey;column:id"`
	AppID      string
	Platform   string
	Action     string
	InstallID  string
	Nonce      string
	ExpiresAt  time.Time
	CreatedAt  time.Time
	ConsumedAt *time.Time
}

func (challengeModel) TableName() string {
	return "phylax_challenges"
}

type sessionModel struct {
	ID        string `gorm:"primaryKey;column:id"`
	AppID     string
	Platform  string
	Action    string
	InstallID string
	Subject   *string
	RiskLevel string
	IssuedAt  time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
}

func (sessionModel) TableName() string {
	return "phylax_sessions"
}

func NewPostgresStore(databaseURL string) (*PostgresStore, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return nil, err
	}
	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) CreateChallenge(ctx context.Context, challenge Challenge) error {
	return s.db.WithContext(ctx).Create(challengeToModel(challenge)).Error
}

func (s *PostgresStore) ConsumeChallenge(ctx context.Context, id string) (*Challenge, error) {
	var consumed Challenge
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var model challengeModel
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&model, "id = ?", id).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if model.ConsumedAt != nil {
			return ErrNotFound
		}

		now := time.Now().UTC()
		if err := tx.Model(&model).Update("consumed_at", now).Error; err != nil {
			return err
		}
		model.ConsumedAt = &now
		consumed = challengeFromModel(model)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &consumed, nil
}

func (s *PostgresStore) SaveSession(ctx context.Context, session AttestationSession) error {
	return s.db.WithContext(ctx).Create(sessionToModel(session)).Error
}

func (s *PostgresStore) FindSession(ctx context.Context, id string) (*AttestationSession, error) {
	var model sessionModel
	err := s.db.WithContext(ctx).First(&model, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	session := sessionFromModel(model)
	return &session, nil
}

func challengeToModel(challenge Challenge) *challengeModel {
	var consumedAt *time.Time
	if !challenge.ConsumedAt.IsZero() {
		consumedAt = &challenge.ConsumedAt
	}
	return &challengeModel{
		ID:         challenge.ID,
		AppID:      challenge.AppID,
		Platform:   string(challenge.Platform),
		Action:     challenge.Action,
		InstallID:  challenge.InstallID,
		Nonce:      challenge.Nonce,
		ExpiresAt:  challenge.ExpiresAt,
		CreatedAt:  challenge.CreatedAt,
		ConsumedAt: consumedAt,
	}
}

func challengeFromModel(model challengeModel) Challenge {
	challenge := Challenge{
		ID:        model.ID,
		AppID:     model.AppID,
		Platform:  Platform(model.Platform),
		Action:    model.Action,
		InstallID: model.InstallID,
		Nonce:     model.Nonce,
		ExpiresAt: model.ExpiresAt,
		CreatedAt: model.CreatedAt,
	}
	if model.ConsumedAt != nil {
		challenge.ConsumedAt = *model.ConsumedAt
	}
	return challenge
}

func sessionToModel(session AttestationSession) *sessionModel {
	var subject *string
	if session.Subject != "" {
		subject = &session.Subject
	}
	return &sessionModel{
		ID:        session.ID,
		AppID:     session.AppID,
		Platform:  string(session.Platform),
		Action:    session.Action,
		InstallID: session.InstallID,
		Subject:   subject,
		RiskLevel: session.RiskLevel,
		IssuedAt:  session.IssuedAt,
		ExpiresAt: session.ExpiresAt,
	}
}

func sessionFromModel(model sessionModel) AttestationSession {
	session := AttestationSession{
		ID:        model.ID,
		AppID:     model.AppID,
		Platform:  Platform(model.Platform),
		Action:    model.Action,
		InstallID: model.InstallID,
		RiskLevel: model.RiskLevel,
		IssuedAt:  model.IssuedAt,
		ExpiresAt: model.ExpiresAt,
	}
	if model.Subject != nil {
		session.Subject = *model.Subject
	}
	return session
}
