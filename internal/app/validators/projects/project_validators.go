package projects

import "github.com/pycify/phylax/internal/app/types/enum"

type CreateProjectRequest struct {
	Name        string `json:"name" validate:"required,min=2,max=120"`
	Slug        string `json:"slug" validate:"required,min=2,max=120,lowercase"`
	Description string `json:"description" validate:"max=500"`
}

type UpdateProjectRequest struct {
	Name        string `json:"name" validate:"omitempty,min=2,max=120"`
	Slug        string `json:"slug" validate:"omitempty,min=2,max=120,lowercase"`
	Description string `json:"description" validate:"max=500"`
}

type CreateProjectAppRequest struct {
	Name               string              `json:"name" validate:"required,min=2,max=120"`
	Platform           enum.AppPlatform    `json:"platform" validate:"required,oneof=android ios"`
	Provider           enum.AppProvider    `json:"provider" validate:"required,oneof=play_integrity app_attest firebase_app_check"`
	Environment        enum.AppEnvironment `json:"environment" validate:"required,oneof=development staging production"`
	PackageName        string              `json:"package_name" validate:"omitempty,min=3,max=255"`
	BundleID           string              `json:"bundle_id" validate:"omitempty,min=3,max=255"`
	CloudProjectNumber string              `json:"cloud_project_number" validate:"required_if=Provider play_integrity,omitempty,numeric,min=6,max=32"`
	Enabled            *bool               `json:"enabled"`
}

type UpdateProjectAppRequest struct {
	Name               string              `json:"name" validate:"omitempty,min=2,max=120"`
	Provider           enum.AppProvider    `json:"provider" validate:"omitempty,oneof=play_integrity app_attest firebase_app_check"`
	Environment        enum.AppEnvironment `json:"environment" validate:"omitempty,oneof=development staging production"`
	PackageName        string              `json:"package_name" validate:"omitempty,min=3,max=255"`
	BundleID           string              `json:"bundle_id" validate:"omitempty,min=3,max=255"`
	CloudProjectNumber string              `json:"cloud_project_number" validate:"omitempty,numeric,min=6,max=32"`
	Enabled            *bool               `json:"enabled"`
}

type StoreAppCredentialRequest struct {
	Provider       enum.AppProvider `json:"provider" validate:"required,oneof=play_integrity app_attest firebase_app_check"`
	CredentialType string           `json:"credential_type" validate:"required,oneof=google_service_account firebase_service_account apple_team"`
	Secret         string           `json:"secret" validate:"required,min=8"`
}
