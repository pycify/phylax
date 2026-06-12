package models

import (
	"time"

	"github.com/pycify/phylax/internal/app/types/enum"
	"gorm.io/gorm"
)

type Project struct {
	ID          string         `gorm:"primaryKey;column:id" json:"id"`
	OwnerUserID string         `gorm:"column:owner_user_id;not null;index" json:"owner_user_id"`
	Name        string         `gorm:"column:name;not null" json:"name"`
	Slug        string         `gorm:"column:slug;not null;uniqueIndex" json:"slug"`
	Description string         `gorm:"column:description" json:"description"`
	CreatedAt   time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (Project) TableName() string {
	return "projects"
}

type ProjectMember struct {
	ID        string                   `gorm:"primaryKey;column:id" json:"id"`
	ProjectID string                   `gorm:"column:project_id;not null;index" json:"project_id"`
	UserID    string                   `gorm:"column:user_id;not null;index" json:"user_id"`
	Role      enum.ProjectRole         `gorm:"column:role;not null;index" json:"role"`
	Status    enum.ProjectMemberStatus `gorm:"column:status;not null;index" json:"status"`
	CreatedAt time.Time                `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time                `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt           `gorm:"column:deleted_at;index" json:"-"`
}

func (ProjectMember) TableName() string {
	return "project_members"
}
