package models

import (
	"time"

	"gorm.io/gorm"
)

type ProjectAPIKey struct {
	ID         string         `gorm:"primaryKey;column:id" json:"id"`
	ProjectID  string         `gorm:"column:project_id;not null;index" json:"project_id"`
	Name       string         `gorm:"column:name;not null" json:"name"`
	Prefix     string         `gorm:"column:prefix;not null;index" json:"prefix"`
	KeyHash    string         `gorm:"column:key_hash;not null;uniqueIndex" json:"-"`
	Active     bool           `gorm:"column:active;not null;default:true" json:"active"`
	LastUsedAt *time.Time     `gorm:"column:last_used_at" json:"last_used_at,omitempty"`
	CreatedAt  time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	RevokedAt  *time.Time     `gorm:"column:revoked_at" json:"revoked_at,omitempty"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (ProjectAPIKey) TableName() string {
	return "project_api_keys"
}
