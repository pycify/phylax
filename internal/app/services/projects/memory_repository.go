package projects

import (
	"context"
	"sync"
	"time"

	projectmodels "github.com/pycify/phylax/internal/app/models/projects"
)

type MemoryRepository struct {
	mu       sync.RWMutex
	projects map[string]projectmodels.Project
	apps     map[string]projectmodels.ProjectApp
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		projects: map[string]projectmodels.Project{},
		apps:     map[string]projectmodels.ProjectApp{},
	}
}

func (r *MemoryRepository) CreateProject(_ context.Context, project projectmodels.Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.projects[project.ID] = project
	return nil
}

func (r *MemoryRepository) ListProjects(_ context.Context, ownerUserID string) ([]projectmodels.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	projects := []projectmodels.Project{}
	for _, project := range r.projects {
		if project.OwnerUserID == ownerUserID && project.DeletedAt.Time.IsZero() {
			projects = append(projects, project)
		}
	}
	return projects, nil
}

func (r *MemoryRepository) FindProject(_ context.Context, ownerUserID string, projectID string) (*projectmodels.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	project, ok := r.projects[projectID]
	if !ok || project.OwnerUserID != ownerUserID || !project.DeletedAt.Time.IsZero() {
		return nil, ErrProjectNotFound
	}
	return &project, nil
}

func (r *MemoryRepository) UpdateProject(_ context.Context, project projectmodels.Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.projects[project.ID]; !ok {
		return ErrProjectNotFound
	}
	r.projects[project.ID] = project
	return nil
}

func (r *MemoryRepository) DeleteProject(_ context.Context, ownerUserID string, projectID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	project, ok := r.projects[projectID]
	if !ok || project.OwnerUserID != ownerUserID || !project.DeletedAt.Time.IsZero() {
		return ErrProjectNotFound
	}
	now := time.Now().UTC()
	project.DeletedAt.Time = now
	project.DeletedAt.Valid = true
	r.projects[projectID] = project
	for appID, app := range r.apps {
		if app.ProjectID == projectID && app.DeletedAt.Time.IsZero() {
			app.DeletedAt.Time = now
			app.DeletedAt.Valid = true
			r.apps[appID] = app
		}
	}
	return nil
}

func (r *MemoryRepository) CreateApp(_ context.Context, app projectmodels.ProjectApp) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.apps[app.ID] = app
	return nil
}

func (r *MemoryRepository) ListApps(_ context.Context, projectID string) ([]projectmodels.ProjectApp, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	apps := []projectmodels.ProjectApp{}
	for _, app := range r.apps {
		if app.ProjectID == projectID && app.DeletedAt.Time.IsZero() {
			apps = append(apps, app)
		}
	}
	return apps, nil
}

func (r *MemoryRepository) FindApp(_ context.Context, projectID string, appID string) (*projectmodels.ProjectApp, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	app, ok := r.apps[appID]
	if !ok || app.ProjectID != projectID || !app.DeletedAt.Time.IsZero() {
		return nil, ErrAppNotFound
	}
	return &app, nil
}

func (r *MemoryRepository) FindAppByID(_ context.Context, appID string) (*projectmodels.ProjectApp, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	app, ok := r.apps[appID]
	if !ok || !app.DeletedAt.Time.IsZero() {
		return nil, ErrAppNotFound
	}
	project, ok := r.projects[app.ProjectID]
	if !ok || !project.DeletedAt.Time.IsZero() {
		return nil, ErrAppNotFound
	}
	return &app, nil
}

func (r *MemoryRepository) FindAppForOwner(_ context.Context, ownerUserID string, appID string) (*projectmodels.ProjectApp, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	app, ok := r.apps[appID]
	if !ok || !app.DeletedAt.Time.IsZero() {
		return nil, ErrAppNotFound
	}
	project, ok := r.projects[app.ProjectID]
	if !ok || project.OwnerUserID != ownerUserID || !project.DeletedAt.Time.IsZero() {
		return nil, ErrAppNotFound
	}
	return &app, nil
}

func (r *MemoryRepository) UpdateApp(_ context.Context, app projectmodels.ProjectApp) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.apps[app.ID]; !ok {
		return ErrAppNotFound
	}
	r.apps[app.ID] = app
	return nil
}

func (r *MemoryRepository) DeleteApp(_ context.Context, projectID string, appID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	app, ok := r.apps[appID]
	if !ok || app.ProjectID != projectID || !app.DeletedAt.Time.IsZero() {
		return ErrAppNotFound
	}
	now := time.Now().UTC()
	app.DeletedAt.Time = now
	app.DeletedAt.Valid = true
	r.apps[appID] = app
	return nil
}

func (r *MemoryRepository) DeleteAppForOwner(_ context.Context, ownerUserID string, appID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	app, ok := r.apps[appID]
	if !ok || !app.DeletedAt.Time.IsZero() {
		return ErrAppNotFound
	}
	project, ok := r.projects[app.ProjectID]
	if !ok || project.OwnerUserID != ownerUserID || !project.DeletedAt.Time.IsZero() {
		return ErrAppNotFound
	}
	now := time.Now().UTC()
	app.DeletedAt.Time = now
	app.DeletedAt.Valid = true
	r.apps[appID] = app
	return nil
}
