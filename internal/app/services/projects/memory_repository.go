package projects

import (
	"context"
	"sync"
	"time"

	projectmodels "github.com/pycify/phylax/internal/app/models/projects"
	"github.com/pycify/phylax/internal/app/types/enum"
)

type MemoryRepository struct {
	mu          sync.RWMutex
	projects    map[string]projectmodels.Project
	apiKeys     map[string]projectmodels.ProjectAPIKey
	apps        map[string]projectmodels.ProjectApp
	credentials map[string]projectmodels.AppCredential
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		projects:    map[string]projectmodels.Project{},
		apiKeys:     map[string]projectmodels.ProjectAPIKey{},
		apps:        map[string]projectmodels.ProjectApp{},
		credentials: map[string]projectmodels.AppCredential{},
	}
}

func (r *MemoryRepository) CreateProject(_ context.Context, project projectmodels.Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.projects[project.ID] = project
	return nil
}

func (r *MemoryRepository) CreateProjectWithAPIKey(_ context.Context, project projectmodels.Project, key projectmodels.ProjectAPIKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.projects[project.ID] = project
	r.apiKeys[key.ID] = key
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
	for keyID, key := range r.apiKeys {
		if key.ProjectID == projectID && key.DeletedAt.Time.IsZero() {
			key.Active = false
			key.RevokedAt = &now
			key.UpdatedAt = now
			key.DeletedAt.Time = now
			key.DeletedAt.Valid = true
			r.apiKeys[keyID] = key
		}
	}
	return nil
}

func (r *MemoryRepository) FindActiveProjectAPIKey(_ context.Context, ownerUserID string, projectID string) (*projectmodels.ProjectAPIKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	project, ok := r.projects[projectID]
	if !ok || project.OwnerUserID != ownerUserID || !project.DeletedAt.Time.IsZero() {
		return nil, ErrProjectNotFound
	}
	for _, key := range r.apiKeys {
		if key.ProjectID == projectID && key.Active && key.RevokedAt == nil && key.DeletedAt.Time.IsZero() {
			return &key, nil
		}
	}
	return nil, ErrAPIKeyNotFound
}

func (r *MemoryRepository) RotateProjectAPIKey(_ context.Context, ownerUserID string, projectID string, key projectmodels.ProjectAPIKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	project, ok := r.projects[projectID]
	if !ok || project.OwnerUserID != ownerUserID || !project.DeletedAt.Time.IsZero() {
		return ErrProjectNotFound
	}
	now := time.Now().UTC()
	for keyID, existing := range r.apiKeys {
		if existing.ProjectID == projectID && existing.Active && existing.DeletedAt.Time.IsZero() {
			existing.Active = false
			existing.RevokedAt = &now
			existing.UpdatedAt = now
			r.apiKeys[keyID] = existing
		}
	}
	r.apiKeys[key.ID] = key
	return nil
}

func (r *MemoryRepository) RevokeProjectAPIKey(_ context.Context, ownerUserID string, projectID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	project, ok := r.projects[projectID]
	if !ok || project.OwnerUserID != ownerUserID || !project.DeletedAt.Time.IsZero() {
		return ErrProjectNotFound
	}
	now := time.Now().UTC()
	revoked := false
	for keyID, key := range r.apiKeys {
		if key.ProjectID == projectID && key.Active && key.RevokedAt == nil && key.DeletedAt.Time.IsZero() {
			key.Active = false
			key.RevokedAt = &now
			key.UpdatedAt = now
			r.apiKeys[keyID] = key
			revoked = true
		}
	}
	if !revoked {
		return ErrAPIKeyNotFound
	}
	return nil
}

func (r *MemoryRepository) FindActiveProjectAPIKeyByHash(_ context.Context, keyHash string) (*projectmodels.ProjectAPIKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, key := range r.apiKeys {
		if key.KeyHash == keyHash && key.Active && key.RevokedAt == nil && key.DeletedAt.Time.IsZero() {
			project, ok := r.projects[key.ProjectID]
			if !ok || !project.DeletedAt.Time.IsZero() {
				return nil, ErrAPIKeyNotFound
			}
			return &key, nil
		}
	}
	return nil, ErrAPIKeyNotFound
}

func (r *MemoryRepository) MarkProjectAPIKeyUsed(_ context.Context, keyID string, usedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key, ok := r.apiKeys[keyID]
	if !ok {
		return ErrAPIKeyNotFound
	}
	key.LastUsedAt = &usedAt
	key.UpdatedAt = usedAt
	r.apiKeys[keyID] = key
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

func (r *MemoryRepository) UpsertAppCredential(_ context.Context, credential projectmodels.AppCredential) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.apps[credential.AppID]; !ok {
		return ErrAppNotFound
	}
	for id, existing := range r.credentials {
		if existing.AppID == credential.AppID && existing.Provider == credential.Provider && existing.CredentialType == credential.CredentialType && existing.Active && existing.DeletedAt.Time.IsZero() {
			existing.Active = false
			existing.UpdatedAt = time.Now().UTC()
			r.credentials[id] = existing
		}
	}
	r.credentials[credential.ID] = credential
	return nil
}

func (r *MemoryRepository) FindActiveAppCredential(_ context.Context, appID string, provider enum.AppProvider, credentialType string) (*projectmodels.AppCredential, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, credential := range r.credentials {
		if credential.AppID == appID && credential.Provider == provider && credential.CredentialType == credentialType && credential.Active && credential.DeletedAt.Time.IsZero() {
			return &credential, nil
		}
	}
	return nil, ErrAppCredentialNotFound
}
