package models

import "time"

type Role struct {
	ID          string    `gorm:"primaryKey;column:id" json:"id"`
	Name        string    `gorm:"column:name;not null;uniqueIndex" json:"name"`
	Description string    `gorm:"column:description" json:"description"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Role) TableName() string {
	return "roles"
}

type UserRole struct {
	UserID    string    `gorm:"primaryKey;column:user_id" json:"user_id"`
	RoleID    string    `gorm:"primaryKey;column:role_id" json:"role_id"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`

	User User `gorm:"foreignKey:UserID" json:"-"`
	Role Role `gorm:"foreignKey:RoleID" json:"-"`
}

func (UserRole) TableName() string {
	return "user_roles"
}
