package models

import (
	"time"

	"github.com/pycify/phylax/internal/app/types/enum"
	"gorm.io/gorm"
)

type AppCredential struct {
	ID              string           `gorm:"primaryKey;column:id" json:"id"`
	AppID           string           `gorm:"column:app_id;not null;index" json:"app_id"`
	Provider        enum.AppProvider `gorm:"column:provider;not null;index" json:"provider"`
	CredentialType  string           `gorm:"column:credential_type;not null;index" json:"credential_type"`
	EncryptedSecret string           `gorm:"column:encrypted_secret;not null" json:"-"`
	SecretNonce     string           `gorm:"column:secret_nonce;not null" json:"-"`
	SecretSHA256    string           `gorm:"column:secret_sha256;not null" json:"secret_sha256"`
	Active          bool             `gorm:"column:active;not null;default:true" json:"active"`
	CreatedAt       time.Time        `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time        `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt       gorm.DeletedAt   `gorm:"column:deleted_at;index" json:"-"`
}

func (AppCredential) TableName() string {
	return "app_credentials"
}
