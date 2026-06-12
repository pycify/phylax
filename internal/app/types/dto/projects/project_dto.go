package projects

import (
	"time"

	projectmodels "github.com/pycify/phylax/internal/app/models/projects"
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
	ID          string              `json:"id"`
	ProjectID   string              `json:"project_id"`
	Name        string              `json:"name"`
	Platform    enum.AppPlatform    `json:"platform"`
	Provider    enum.AppProvider    `json:"provider"`
	Environment enum.AppEnvironment `json:"environment"`
	PackageName string              `json:"package_name,omitempty"`
	BundleID    string              `json:"bundle_id,omitempty"`
	Enabled     bool                `json:"enabled"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

func NewProjectAppDTO(app projectmodels.ProjectApp) ProjectAppDTO {
	return ProjectAppDTO{
		ID:          app.ID,
		ProjectID:   app.ProjectID,
		Name:        app.Name,
		Platform:    app.Platform,
		Provider:    app.Provider,
		Environment: app.Environment,
		PackageName: app.PackageName,
		BundleID:    app.BundleID,
		Enabled:     app.Enabled,
		CreatedAt:   app.CreatedAt,
		UpdatedAt:   app.UpdatedAt,
	}
}

type ProjectResponse struct {
	Project ProjectDTO `json:"project"`
}

type ProjectsResponse struct {
	Projects []ProjectDTO `json:"projects"`
}

type ProjectAppResponse struct {
	App ProjectAppDTO `json:"app"`
}

type ProjectAppsResponse struct {
	Apps []ProjectAppDTO `json:"apps"`
}
