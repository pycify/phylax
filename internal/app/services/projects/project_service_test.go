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
	t.Setenv("PHYLAX_CREDENTIAL_ENCRYPTION_KEY", "test-encryption-key-with-32-characters")
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
	credential, err := service.StoreAppCredential(ctx, "user_1", app.ID, projectvalidators.StoreAppCredentialRequest{
		Provider:       enum.AppProviderPlayIntegrity,
		CredentialType: "google_service_account",
		Secret:         `{"type":"service_account","client_email":"phylax@example.com"}`,
	})
	require.NoError(t, err)
	require.Equal(t, app.ID, credential.AppID)
	require.NotEqual(t, `{"type":"service_account","client_email":"phylax@example.com"}`, credential.EncryptedSecret)

	status, err := service.GetAppAttestationStatus(ctx, "user_1", app.ID)
	require.NoError(t, err)
	require.True(t, status.Configured)
	require.True(t, status.RequiresDeveloperCredentials)
	require.Equal(t, "manual_verification_required", status.ExternalConfigurationCheck)
	require.Equal(t, "com.acme.app", status.Identifier)
	require.Contains(t, status.NextSteps, "In Play Console, open the app, go to Protected with Play, then link the Cloud project under Play Integrity API.")
	require.Contains(t, status.SetupSteps, AppSetupStep{
		Key:                "service_account_credentials",
		Title:              "Configure backend service account credentials",
		Status:             "completed",
		VerificationMethod: "database",
		Details:            "An active encrypted Google service account credential is stored for this app.",
		Action:             "Upload the service account JSON for the linked Cloud project to this app's credentials.",
	})
}

func TestProjectServiceResolvesAppConfigForAttestation(t *testing.T) {
	t.Setenv("PHYLAX_CREDENTIAL_ENCRYPTION_KEY", "test-encryption-key-with-32-characters")
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
	_, err = service.StoreAppCredential(ctx, "user_1", app.ID, projectvalidators.StoreAppCredentialRequest{
		Provider:       enum.AppProviderPlayIntegrity,
		CredentialType: "google_service_account",
		Secret:         `{"type":"service_account","client_email":"phylax@example.com"}`,
	})
	require.NoError(t, err)

	resolved, err := service.ResolveApp(ctx, app.ID)
	require.NoError(t, err)
	require.Equal(t, app.ID, resolved.ID)
	require.True(t, resolved.Android.Enabled)
	require.Equal(t, "com.acme.app", resolved.Android.PackageName)
	require.Empty(t, resolved.Android.ServiceAccountJSONEnv)
	require.Equal(t, `{"type":"service_account","client_email":"phylax@example.com"}`, resolved.Android.ServiceAccountJSON)
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
