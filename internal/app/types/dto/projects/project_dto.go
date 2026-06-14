package projects

import (
	"time"

	projectmodels "github.com/pycify/phylax/internal/app/models/projects"
	projectservice "github.com/pycify/phylax/internal/app/services/projects"
	"github.com/pycify/phylax/internal/app/types/enum"
)

type ProjectDTO struct {
	ID          string    `json:"id"`
	OwnerUserID string    `json:"owner_user_id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func NewProjectDTO(project projectmodels.Project) ProjectDTO {
	return ProjectDTO{
		ID:          project.ID,
		OwnerUserID: project.OwnerUserID,
		Name:        project.Name,
		Slug:        project.Slug,
		Description: project.Description,
		CreatedAt:   project.CreatedAt,
		UpdatedAt:   project.UpdatedAt,
	}
}

type ProjectAppDTO struct {
	ID                 string              `json:"id"`
	ProjectID          string              `json:"project_id"`
	Name               string              `json:"name"`
	Platform           enum.AppPlatform    `json:"platform"`
	Provider           enum.AppProvider    `json:"provider"`
	Environment        enum.AppEnvironment `json:"environment"`
	PackageName        string              `json:"package_name,omitempty"`
	BundleID           string              `json:"bundle_id,omitempty"`
	CloudProjectNumber string              `json:"cloud_project_number,omitempty"`
	Enabled            bool                `json:"enabled"`
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at"`
}

type AppCredentialDTO struct {
	ID             string           `json:"id"`
	AppID          string           `json:"app_id"`
	Provider       enum.AppProvider `json:"provider"`
	CredentialType string           `json:"credential_type"`
	SecretSHA256   string           `json:"secret_sha256"`
	Active         bool             `json:"active"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

type ProjectAPIKeyDTO struct {
	ID         string     `json:"id"`
	ProjectID  string     `json:"project_id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	Key        string     `json:"key,omitempty"`
	Active     bool       `json:"active"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
}

func NewProjectAppDTO(app projectmodels.ProjectApp) ProjectAppDTO {
	return ProjectAppDTO{
		ID:                 app.ID,
		ProjectID:          app.ProjectID,
		Name:               app.Name,
		Platform:           app.Platform,
		Provider:           app.Provider,
		Environment:        app.Environment,
		PackageName:        app.PackageName,
		BundleID:           app.BundleID,
		CloudProjectNumber: app.CloudProjectNumber,
		Enabled:            app.Enabled,
		CreatedAt:          app.CreatedAt,
		UpdatedAt:          app.UpdatedAt,
	}
}

func NewAppCredentialDTO(credential projectmodels.AppCredential) AppCredentialDTO {
	return AppCredentialDTO{
		ID:             credential.ID,
		AppID:          credential.AppID,
		Provider:       credential.Provider,
		CredentialType: credential.CredentialType,
		SecretSHA256:   credential.SecretSHA256,
		Active:         credential.Active,
		CreatedAt:      credential.CreatedAt,
		UpdatedAt:      credential.UpdatedAt,
	}
}

func NewProjectAPIKeyDTO(key projectservice.ProjectAPIKey) ProjectAPIKeyDTO {
	return ProjectAPIKeyDTO{
		ID:         key.ID,
		ProjectID:  key.ProjectID,
		Name:       key.Name,
		Prefix:     key.Prefix,
		Key:        key.Plaintext,
		Active:     key.Active,
		LastUsedAt: key.LastUsedAt,
		CreatedAt:  key.CreatedAt,
		UpdatedAt:  key.UpdatedAt,
		RevokedAt:  key.RevokedAt,
	}
}

type ProjectResponse struct {
	Project ProjectDTO `json:"project"`
}

type ProjectCreatedResponse struct {
	Project ProjectDTO       `json:"project"`
	APIKey  ProjectAPIKeyDTO `json:"api_key"`
}

type ProjectsResponse struct {
	Projects []ProjectDTO `json:"projects"`
}

type ProjectAppResponse struct {
	App ProjectAppDTO `json:"app"`
}

type AppCredentialResponse struct {
	Credential AppCredentialDTO `json:"credential"`
}

type ProjectAPIKeyResponse struct {
	APIKey ProjectAPIKeyDTO `json:"api_key"`
}

type ProjectAppsResponse struct {
	Apps []ProjectAppDTO `json:"apps"`
}

type AppAttestationStatusDTO struct {
	AppID                        string            `json:"app_id"`
	Platform                     enum.AppPlatform  `json:"platform"`
	Provider                     enum.AppProvider  `json:"provider"`
	Enabled                      bool              `json:"enabled"`
	Configured                   bool              `json:"configured"`
	Identifier                   string            `json:"identifier"`
	ExternalConfigurationCheck   string            `json:"external_configuration_check"`
	RequiresDeveloperCredentials bool              `json:"requires_developer_credentials"`
	Issues                       []string          `json:"issues,omitempty"`
	NextSteps                    []string          `json:"next_steps,omitempty"`
	SetupSteps                   []AppSetupStepDTO `json:"setup_steps"`
}

type AppSetupStepDTO struct {
	Key                string `json:"key"`
	Title              string `json:"title"`
	Status             string `json:"status"`
	VerificationMethod string `json:"verification_method"`
	Details            string `json:"details"`
	Action             string `json:"action,omitempty"`
	DocsURL            string `json:"docs_url,omitempty"`
}

func NewAppAttestationStatusDTO(status projectservice.AppAttestationStatus) AppAttestationStatusDTO {
	steps := make([]AppSetupStepDTO, 0, len(status.SetupSteps))
	for _, step := range status.SetupSteps {
		steps = append(steps, AppSetupStepDTO{
			Key:                step.Key,
			Title:              step.Title,
			Status:             step.Status,
			VerificationMethod: step.VerificationMethod,
			Details:            step.Details,
			Action:             step.Action,
			DocsURL:            step.DocsURL,
		})
	}

	return AppAttestationStatusDTO{
		AppID:                        status.AppID,
		Platform:                     status.Platform,
		Provider:                     status.Provider,
		Enabled:                      status.Enabled,
		Configured:                   status.Configured,
		Identifier:                   status.Identifier,
		ExternalConfigurationCheck:   status.ExternalConfigurationCheck,
		RequiresDeveloperCredentials: status.RequiresDeveloperCredentials,
		Issues:                       status.Issues,
		NextSteps:                    status.NextSteps,
		SetupSteps:                   steps,
	}
}

type AppAttestationStatusResponse struct {
	Status AppAttestationStatusDTO `json:"status"`
}
