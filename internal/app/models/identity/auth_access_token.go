package models

import "time"

type AuthAccessToken struct {
	ID         string     `gorm:"primaryKey;column:id" json:"id"`
	UserID     string     `gorm:"column:user_id;not null;index" json:"user_id"`
	Name       string     `gorm:"column:name;not null" json:"name"`
	Hash       string     `gorm:"column:hash;not null;uniqueIndex" json:"-"`
	Abilities  string     `gorm:"column:abilities;not null" json:"abilities"`
	CreatedAt  time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	LastUsedAt *time.Time `gorm:"column:last_used_at" json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `gorm:"column:expires_at" json:"expires_at,omitempty"`
	RevokedAt  *time.Time `gorm:"column:revoked_at;index" json:"revoked_at,omitempty"`
}

func (AuthAccessToken) TableName() string {
	return "auth_access_tokens"
}
