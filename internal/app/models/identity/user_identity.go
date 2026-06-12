package models

import (
	"time"

	"github.com/pycify/phylax/internal/app/types/enum"
	"gorm.io/gorm"
)

type UserIdentity struct {
	ID             string            `gorm:"primaryKey;column:id" json:"id"`
	UserID         string            `gorm:"column:user_id;not null;index" json:"user_id"`
	Provider       enum.AuthProvider `gorm:"column:provider;not null;index" json:"provider"`
	ProviderUserID string            `gorm:"column:provider_user_id;not null;index" json:"provider_user_id"`
	Email          string            `gorm:"column:email;not null" json:"email"`
	EmailVerified  bool              `gorm:"column:email_verified;not null" json:"email_verified"`
	CreatedAt      time.Time         `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time         `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt      gorm.DeletedAt    `gorm:"column:deleted_at;index" json:"-"`
}

func (UserIdentity) TableName() string {
	return "user_identities"
}
