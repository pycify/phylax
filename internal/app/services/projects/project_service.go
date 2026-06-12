package projects

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	projectmodels "github.com/pycify/phylax/internal/app/models/projects"
	"github.com/pycify/phylax/internal/app/types/enum"
	projectvalidators "github.com/pycify/phylax/internal/app/validators/projects"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	ErrProjectNotFound = errors.New("project not found")
	ErrAppNotFound     = errors.New("app not found")
	ErrInvalidApp      = errors.New("invalid app")
)

type Repository interface {
	CreateProject(ctx context.Context, project projectmodels.Project) error
	ListProjects(ctx context.Context, ownerUserID string) ([]projectmodels.Project, error)
	FindProject(ctx context.Context, ownerUserID string, projectID string) (*projectmodels.Project, error)
	UpdateProject(ctx context.Context, project projectmodels.Project) error
	DeleteProject(ctx context.Context, ownerUserID string, projectID string) error

	CreateApp(ctx context.Context, app projectmodels.ProjectApp) error
	ListApps(ctx context.Context, projectID string) ([]projectmodels.ProjectApp, error)
	FindApp(ctx context.Context, projectID string, appID string) (*projectmodels.ProjectApp, error)
	UpdateApp(ctx context.Context, app projectmodels.ProjectApp) error
	DeleteApp(ctx context.Context, projectID string, appID string) error
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) CreateProject(ctx context.Context, ownerUserID string, payload projectvalidators.CreateProjectRequest) (projectmodels.Project, error) {
	now := time.Now().UTC()
	project := projectmodels.Project{
		ID:          uuid.NewString(),
		OwnerUserID: ownerUserID,
		Name:        strings.TrimSpace(payload.Name),
		Slug:        strings.TrimSpace(payload.Slug),
		Description: strings.TrimSpace(payload.Description),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	return project, s.repository.CreateProject(ctx, project)
}

func (s *Service) ListProjects(ctx context.Context, ownerUserID string) ([]projectmodels.Project, error) {
	return s.repository.ListProjects(ctx, ownerUserID)
}

func (s *Service) GetProject(ctx context.Context, ownerUserID string, projectID string) (*projectmodels.Project, error) {
	return s.repository.FindProject(ctx, ownerUserID, projectID)
}

func (s *Service) UpdateProject(ctx context.Context, ownerUserID string, projectID string, payload projectvalidators.UpdateProjectRequest) (*projectmodels.Project, error) {
	project, err := s.repository.FindProject(ctx, ownerUserID, projectID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(payload.Name) != "" {
		project.Name = strings.TrimSpace(payload.Name)
	}
	if strings.TrimSpace(payload.Slug) != "" {
		project.Slug = strings.TrimSpace(payload.Slug)
	}
	project.Description = strings.TrimSpace(payload.Description)
	project.UpdatedAt = time.Now().UTC()
	if err := s.repository.UpdateProject(ctx, *project); err != nil {
		return nil, err
	}
	return project, nil
}

func (s *Service) DeleteProject(ctx context.Context, ownerUserID string, projectID string) error {
	return s.repository.DeleteProject(ctx, ownerUserID, projectID)
}

func (s *Service) CreateApp(ctx context.Context, ownerUserID string, projectID string, payload projectvalidators.CreateProjectAppRequest) (projectmodels.ProjectApp, error) {
	if _, err := s.repository.FindProject(ctx, ownerUserID, projectID); err != nil {
		return projectmodels.ProjectApp{}, err
	}
	if err := validateAppIdentifiers(payload.Platform, payload.PackageName, payload.BundleID); err != nil {
		return projectmodels.ProjectApp{}, err
	}

	enabled := true
	if payload.Enabled != nil {
		enabled = *payload.Enabled
	}
	now := time.Now().UTC()
	app := projectmodels.ProjectApp{
		ID:          uuid.NewString(),
		ProjectID:   projectID,
		Name:        strings.TrimSpace(payload.Name),
		Platform:    payload.Platform,
		Provider:    payload.Provider,
		Environment: payload.Environment,
		PackageName: strings.TrimSpace(payload.PackageName),
		BundleID:    strings.TrimSpace(payload.BundleID),
		Enabled:     enabled,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	return app, s.repository.CreateApp(ctx, app)
}

func (s *Service) ListApps(ctx context.Context, ownerUserID string, projectID string) ([]projectmodels.ProjectApp, error) {
	if _, err := s.repository.FindProject(ctx, ownerUserID, projectID); err != nil {
		return nil, err
	}
	return s.repository.ListApps(ctx, projectID)
}

func (s *Service) GetApp(ctx context.Context, ownerUserID string, projectID string, appID string) (*projectmodels.ProjectApp, error) {
	if _, err := s.repository.FindProject(ctx, ownerUserID, projectID); err != nil {
		return nil, err
	}
	return s.repository.FindApp(ctx, projectID, appID)
}

func (s *Service) UpdateApp(ctx context.Context, ownerUserID string, projectID string, appID string, payload projectvalidators.UpdateProjectAppRequest) (*projectmodels.ProjectApp, error) {
	if _, err := s.repository.FindProject(ctx, ownerUserID, projectID); err != nil {
		return nil, err
	}
	app, err := s.repository.FindApp(ctx, projectID, appID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(payload.Name) != "" {
		app.Name = strings.TrimSpace(payload.Name)
	}
	if payload.Provider != "" {
		app.Provider = payload.Provider
	}
	if payload.Environment != "" {
		app.Environment = payload.Environment
	}
	if strings.TrimSpace(payload.PackageName) != "" {
		app.PackageName = strings.TrimSpace(payload.PackageName)
	}
	if strings.TrimSpace(payload.BundleID) != "" {
		app.BundleID = strings.TrimSpace(payload.BundleID)
	}
	if payload.Enabled != nil {
		app.Enabled = *payload.Enabled
	}
	if err := validateAppIdentifiers(app.Platform, app.PackageName, app.BundleID); err != nil {
		return nil, err
	}
	app.UpdatedAt = time.Now().UTC()
	if err := s.repository.UpdateApp(ctx, *app); err != nil {
		return nil, err
	}
	return app, nil
}

func (s *Service) DeleteApp(ctx context.Context, ownerUserID string, projectID string, appID string) error {
	if _, err := s.repository.FindProject(ctx, ownerUserID, projectID); err != nil {
		return err
	}
	return s.repository.DeleteApp(ctx, projectID, appID)
}

func validateAppIdentifiers(platform enum.AppPlatform, packageName string, bundleID string) error {
	packageName = strings.TrimSpace(packageName)
	bundleID = strings.TrimSpace(bundleID)
	switch platform {
	case enum.AppPlatformAndroid:
		if packageName == "" || bundleID != "" {
			return fmt.Errorf("%w: android apps require package_name and must not include bundle_id", ErrInvalidApp)
		}
	case enum.AppPlatformIOS:
		if bundleID == "" || packageName != "" {
			return fmt.Errorf("%w: iOS apps require bundle_id and must not include package_name", ErrInvalidApp)
		}
	default:
		return fmt.Errorf("%w: unsupported platform", ErrInvalidApp)
	}
	return nil
}

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(databaseURL string) (*GormRepository, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return nil, err
	}
	return &GormRepository{db: db}, nil
}

func (r *GormRepository) CreateProject(ctx context.Context, project projectmodels.Project) error {
	return r.db.WithContext(ctx).Create(&project).Error
}

func (r *GormRepository) ListProjects(ctx context.Context, ownerUserID string) ([]projectmodels.Project, error) {
	var projects []projectmodels.Project
	err := r.db.WithContext(ctx).Where("owner_user_id = ?", ownerUserID).Order("created_at desc").Find(&projects).Error
	return projects, err
}

func (r *GormRepository) FindProject(ctx context.Context, ownerUserID string, projectID string) (*projectmodels.Project, error) {
	var project projectmodels.Project
	err := r.db.WithContext(ctx).Where("owner_user_id = ? and id = ?", ownerUserID, projectID).First(&project).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrProjectNotFound
	}
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *GormRepository) UpdateProject(ctx context.Context, project projectmodels.Project) error {
	return r.db.WithContext(ctx).Save(&project).Error
}

func (r *GormRepository) DeleteProject(ctx context.Context, ownerUserID string, projectID string) error {
	result := r.db.WithContext(ctx).Where("owner_user_id = ? and id = ?", ownerUserID, projectID).Delete(&projectmodels.Project{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProjectNotFound
	}
	return nil
}

func (r *GormRepository) CreateApp(ctx context.Context, app projectmodels.ProjectApp) error {
	return r.db.WithContext(ctx).Create(&app).Error
}

func (r *GormRepository) ListApps(ctx context.Context, projectID string) ([]projectmodels.ProjectApp, error) {
	var apps []projectmodels.ProjectApp
	err := r.db.WithContext(ctx).Where("project_id = ?", projectID).Order("created_at desc").Find(&apps).Error
	return apps, err
}

func (r *GormRepository) FindApp(ctx context.Context, projectID string, appID string) (*projectmodels.ProjectApp, error) {
	var app projectmodels.ProjectApp
	err := r.db.WithContext(ctx).Where("project_id = ? and id = ?", projectID, appID).First(&app).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrAppNotFound
	}
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (r *GormRepository) UpdateApp(ctx context.Context, app projectmodels.ProjectApp) error {
	return r.db.WithContext(ctx).Save(&app).Error
}

func (r *GormRepository) DeleteApp(ctx context.Context, projectID string, appID string) error {
	result := r.db.WithContext(ctx).Where("project_id = ? and id = ?", projectID, appID).Delete(&projectmodels.ProjectApp{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrAppNotFound
	}
	return nil
}
