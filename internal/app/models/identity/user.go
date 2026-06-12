package models

import (
	"time"

	"github.com/pycify/phylax/internal/app/types/enum"
	"gorm.io/gorm"
)

type User struct {
	ID              string          `gorm:"primaryKey;column:id" json:"id"`
	Email           string          `gorm:"column:email;not null;index" json:"email"`
	FullName        string          `gorm:"column:full_name" json:"full_name"`
	PasswordHash    string          `gorm:"column:password_hash;not null" json:"-"`
	UserType        enum.UserType   `gorm:"column:user_type;not null;index" json:"user_type"`
	Status          enum.UserStatus `gorm:"column:status;not null;index" json:"status"`
	EmailVerifiedAt *time.Time      `gorm:"column:email_verified_at" json:"email_verified_at,omitempty"`
	CreatedAt       time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt       gorm.DeletedAt  `gorm:"column:deleted_at;index" json:"-"`
}

func (User) TableName() string {
	return "users"
}
