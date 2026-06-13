package projects

import (
	"context"
	"errors"
	"testing"

	"github.com/pycify/phylax/internal/app/types/enum"
	projectvalidators "github.com/pycify/phylax/internal/app/validators/projects"
	"github.com/stretchr/testify/require"
)

func TestProjectServiceCreatesProjectAndNestedApps(t *testing.T) {
	service := NewService(NewMemoryRepository())
	ctx := context.Background()

	project, err := service.CreateProject(ctx, "user_1", projectvalidators.CreateProjectRequest{
		Name: "Acme",
		Slug: "acme",
	})
	require.NoError(t, err)
	require.Equal(t, "user_1", project.OwnerUserID)

	app, err := service.CreateApp(ctx, "user_1", project.ID, projectvalidators.CreateProjectAppRequest{
		Name:        "Acme Android",
		Platform:    enum.AppPlatformAndroid,
		Provider:    enum.AppProviderPlayIntegrity,
		Environment: enum.AppEnvironmentDevelopment,
		PackageName: "com.acme.app",
	})
	require.NoError(t, err)
	require.Equal(t, project.ID, app.ProjectID)

	apps, err := service.ListApps(ctx, "user_1", project.ID)
	require.NoError(t, err)
	require.Len(t, apps, 1)
}

func TestProjectServiceEnforcesOwnership(t *testing.T) {
	service := NewService(NewMemoryRepository())
	ctx := context.Background()

	project, err := service.CreateProject(ctx, "owner", projectvalidators.CreateProjectRequest{
		Name: "Owned",
		Slug: "owned",
	})
	require.NoError(t, err)

	_, err = service.GetProject(ctx, "other_user", project.ID)
	require.ErrorIs(t, err, ErrProjectNotFound)

	_, err = service.CreateApp(ctx, "other_user", project.ID, projectvalidators.CreateProjectAppRequest{
		Name:        "Invalid",
		Platform:    enum.AppPlatformIOS,
		Provider:    enum.AppProviderAppAttest,
		Environment: enum.AppEnvironmentDevelopment,
		BundleID:    "com.acme.ios",
	})
	require.ErrorIs(t, err, ErrProjectNotFound)
}

func TestProjectServiceFindsAppsByIDWithinOwnerScope(t *testing.T) {
	service := NewService(NewMemoryRepository())
	ctx := context.Background()

	project, err := service.CreateProject(ctx, "owner", projectvalidators.CreateProjectRequest{
		Name: "Owned",
		Slug: "owned",
	})
	require.NoError(t, err)
	app, err := service.CreateApp(ctx, "owner", project.ID, projectvalidators.CreateProjectAppRequest{
		Name:        "Owned Android",
		Platform:    enum.AppPlatformAndroid,
		Provider:    enum.AppProviderPlayIntegrity,
		Environment: enum.AppEnvironmentDevelopment,
		PackageName: "com.acme.app",
	})
	require.NoError(t, err)

	found, err := service.GetApp(ctx, "owner", app.ID)
	require.NoError(t, err)
	require.Equal(t, app.ID, found.ID)

	_, err = service.GetApp(ctx, "other_user", app.ID)
	require.ErrorIs(t, err, ErrAppNotFound)
}

func TestProjectServiceValidatesAppIdentifierByPlatform(t *testing.T) {
	service := NewService(NewMemoryRepository())
	ctx := context.Background()

	project, err := service.CreateProject(ctx, "user_1", projectvalidators.CreateProjectRequest{
		Name: "Acme",
		Slug: "acme",
	})
	require.NoError(t, err)

	_, err = service.CreateApp(ctx, "user_1", project.ID, projectvalidators.CreateProjectAppRequest{
		Name:        "Broken Android",
		Platform:    enum.AppPlatformAndroid,
		Provider:    enum.AppProviderPlayIntegrity,
		Environment: enum.AppEnvironmentDevelopment,
		BundleID:    "com.acme.ios",
	})
	require.True(t, errors.Is(err, ErrInvalidApp))
}

func TestProjectServiceReportsAppAttestationStatus(t *testing.T) {
	service := NewService(NewMemoryRepository())
	ctx := context.Background()

	project, err := service.CreateProject(ctx, "user_1", projectvalidators.CreateProjectRequest{
		Name: "Acme",
		Slug: "acme",
	})
	require.NoError(t, err)
	app, err := service.CreateApp(ctx, "user_1", project.ID, projectvalidators.CreateProjectAppRequest{
		Name:        "Acme Android",
		Platform:    enum.AppPlatformAndroid,
		Provider:    enum.AppProviderPlayIntegrity,
		Environment: enum.AppEnvironmentProduction,
		PackageName: "com.acme.app",
	})
	require.NoError(t, err)

	status, err := service.GetAppAttestationStatus(ctx, "user_1", app.ID)
	require.NoError(t, err)
	require.True(t, status.Configured)
	require.True(t, status.RequiresDeveloperCredentials)
	require.Equal(t, "not_configured", status.ExternalConfigurationCheck)
	require.Equal(t, "com.acme.app", status.Identifier)
	require.Contains(t, status.NextSteps, "connect a Google Cloud or Play Console credential before Phylax can inspect Google-side configuration")
}

func TestProjectServiceResolvesAppConfigForAttestation(t *testing.T) {
	service := NewService(NewMemoryRepository())
	ctx := context.Background()

	project, err := service.CreateProject(ctx, "user_1", projectvalidators.CreateProjectRequest{
		Name: "Acme",
		Slug: "acme",
	})
	require.NoError(t, err)
	app, err := service.CreateApp(ctx, "user_1", project.ID, projectvalidators.CreateProjectAppRequest{
		Name:        "Acme Android",
		Platform:    enum.AppPlatformAndroid,
		Provider:    enum.AppProviderPlayIntegrity,
		Environment: enum.AppEnvironmentProduction,
		PackageName: "com.acme.app",
	})
	require.NoError(t, err)

	resolved, err := service.ResolveApp(ctx, app.ID)
	require.NoError(t, err)
	require.Equal(t, app.ID, resolved.ID)
	require.True(t, resolved.Android.Enabled)
	require.Equal(t, "com.acme.app", resolved.Android.PackageName)
	require.Equal(t, "GOOGLE_PLAY_SERVICE_ACCOUNT_JSON", resolved.Android.ServiceAccountJSONEnv)
}

func TestProjectServiceSoftDeletesAppsThroughProjectDelete(t *testing.T) {
	service := NewService(NewMemoryRepository())
	ctx := context.Background()

	project, err := service.CreateProject(ctx, "user_1", projectvalidators.CreateProjectRequest{
		Name: "Acme",
		Slug: "acme",
	})
	require.NoError(t, err)
	app, err := service.CreateApp(ctx, "user_1", project.ID, projectvalidators.CreateProjectAppRequest{
		Name:        "Acme iOS",
		Platform:    enum.AppPlatformIOS,
		Provider:    enum.AppProviderAppAttest,
		Environment: enum.AppEnvironmentDevelopment,
		BundleID:    "com.acme.ios",
	})
	require.NoError(t, err)

	require.NoError(t, service.DeleteProject(ctx, "user_1", project.ID))
	_, err = service.GetProject(ctx, "user_1", project.ID)
	require.ErrorIs(t, err, ErrProjectNotFound)
	_, err = service.GetApp(ctx, "user_1", app.ID)
	require.ErrorIs(t, err, ErrAppNotFound)
}
