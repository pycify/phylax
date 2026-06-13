package models

import (
	"time"

	"github.com/pycify/phylax/internal/app/types/enum"
	"gorm.io/gorm"
)

type ProjectApp struct {
	ID                 string              `gorm:"primaryKey;column:id" json:"id"`
	ProjectID          string              `gorm:"column:project_id;not null;index" json:"project_id"`
	Name               string              `gorm:"column:name;not null" json:"name"`
	Platform           enum.AppPlatform    `gorm:"column:platform;not null;index" json:"platform"`
	Provider           enum.AppProvider    `gorm:"column:provider;not null;index" json:"provider"`
	Environment        enum.AppEnvironment `gorm:"column:environment;not null;index" json:"environment"`
	PackageName        string              `gorm:"column:package_name;index" json:"package_name,omitempty"`
	BundleID           string              `gorm:"column:bundle_id;index" json:"bundle_id,omitempty"`
	CloudProjectNumber string              `gorm:"column:cloud_project_number" json:"cloud_project_number,omitempty"`
	Enabled            bool                `gorm:"column:enabled;not null;default:true" json:"enabled"`
	CreatedAt          time.Time           `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time           `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt          gorm.DeletedAt      `gorm:"column:deleted_at;index" json:"-"`
}

func (ProjectApp) TableName() string {
	return "project_apps"
}
