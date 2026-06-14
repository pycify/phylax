package projects

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	projectmodels "github.com/pycify/phylax/internal/app/models/projects"
	"github.com/pycify/phylax/internal/app/types/enum"
	projectvalidators "github.com/pycify/phylax/internal/app/validators/projects"
	"github.com/pycify/phylax/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	ErrProjectNotFound       = errors.New("project not found")
	ErrAppNotFound           = errors.New("app not found")
	ErrAppCredentialNotFound = errors.New("app credential not found")
	ErrAPIKeyNotFound        = errors.New("project api key not found")
	ErrInvalidAPIKey         = errors.New("invalid project api key")
	ErrInvalidApp            = errors.New("invalid app")
	ErrInvalidCredential     = errors.New("invalid app credential")
)

type Repository interface {
	CreateProject(ctx context.Context, project projectmodels.Project) error
	CreateProjectWithAPIKey(ctx context.Context, project projectmodels.Project, key projectmodels.ProjectAPIKey) error
	ListProjects(ctx context.Context, ownerUserID string) ([]projectmodels.Project, error)
	FindProject(ctx context.Context, ownerUserID string, projectID string) (*projectmodels.Project, error)
	UpdateProject(ctx context.Context, project projectmodels.Project) error
	DeleteProject(ctx context.Context, ownerUserID string, projectID string) error
	FindActiveProjectAPIKey(ctx context.Context, ownerUserID string, projectID string) (*projectmodels.ProjectAPIKey, error)
	RotateProjectAPIKey(ctx context.Context, ownerUserID string, projectID string, key projectmodels.ProjectAPIKey) error
	RevokeProjectAPIKey(ctx context.Context, ownerUserID string, projectID string) error
	FindActiveProjectAPIKeyByHash(ctx context.Context, keyHash string) (*projectmodels.ProjectAPIKey, error)
	MarkProjectAPIKeyUsed(ctx context.Context, keyID string, usedAt time.Time) error

	CreateApp(ctx context.Context, app projectmodels.ProjectApp) error
	ListApps(ctx context.Context, projectID string) ([]projectmodels.ProjectApp, error)
	FindApp(ctx context.Context, projectID string, appID string) (*projectmodels.ProjectApp, error)
	FindAppByID(ctx context.Context, appID string) (*projectmodels.ProjectApp, error)
	FindAppForOwner(ctx context.Context, ownerUserID string, appID string) (*projectmodels.ProjectApp, error)
	UpdateApp(ctx context.Context, app projectmodels.ProjectApp) error
	DeleteApp(ctx context.Context, projectID string, appID string) error
	DeleteAppForOwner(ctx context.Context, ownerUserID string, appID string) error
	UpsertAppCredential(ctx context.Context, credential projectmodels.AppCredential) error
	FindActiveAppCredential(ctx context.Context, appID string, provider enum.AppProvider, credentialType string) (*projectmodels.AppCredential, error)
}

func (s *Service) ResolveApp(ctx context.Context, appID string) (config.AppConfig, error) {
	app, err := s.repository.FindAppByID(ctx, appID)
	if err != nil {
		return config.AppConfig{}, err
	}
	cfg := appConfigFromProjectApp(*app)
	credentialType := credentialTypeForProvider(app.Provider)
	if credentialType != "" {
		credential, err := s.repository.FindActiveAppCredential(ctx, app.ID, app.Provider, credentialType)
		if err == nil {
			secret, err := decryptCredentialSecret(*credential)
			if err != nil {
				return config.AppConfig{}, err
			}
			switch app.Provider {
			case enum.AppProviderPlayIntegrity:
				cfg.Android.ServiceAccountJSON = secret
			case enum.AppProviderAppAttest:
				cfg.Apple.TeamID = secret
			}
		} else if !errors.Is(err, ErrAppCredentialNotFound) {
			return config.AppConfig{}, err
		}
	}
	return cfg, nil
}

func appConfigFromProjectApp(app projectmodels.ProjectApp) config.AppConfig {
	cfg := config.AppConfig{
		ID:   app.ID,
		Name: app.Name,
	}
	switch app.Provider {
	case enum.AppProviderPlayIntegrity:
		cfg.Android = config.AndroidConfig{
			Enabled:               app.Enabled,
			Provider:              string(app.Provider),
			PackageName:           strings.TrimSpace(app.PackageName),
			CloudProjectNumber:    strings.TrimSpace(app.CloudProjectNumber),
			RequirePlayRecognized: app.Environment == enum.AppEnvironmentProduction,
			RequiredDeviceIntegrity: []string{
				"MEETS_DEVICE_INTEGRITY",
			},
		}
	case enum.AppProviderAppAttest:
		cfg.Apple = config.AppleConfig{
			Enabled:     app.Enabled,
			Provider:    string(app.Provider),
			BundleID:    strings.TrimSpace(app.BundleID),
			Environment: string(app.Environment),
		}
	case enum.AppProviderFirebase:
		cfg.Firebase = config.FirebaseConfig{
			Enabled:        app.Enabled,
			CredentialsEnv: "FIREBASE_CREDENTIALS_JSON",
		}
		cfg.Android = config.AndroidConfig{
			Enabled:     app.Enabled && app.Platform == enum.AppPlatformAndroid,
			Provider:    string(app.Provider),
			PackageName: strings.TrimSpace(app.PackageName),
		}
		cfg.Apple = config.AppleConfig{
			Enabled:     app.Enabled && app.Platform == enum.AppPlatformIOS,
			Provider:    string(app.Provider),
			BundleID:    strings.TrimSpace(app.BundleID),
			Environment: string(app.Environment),
		}
	}
	return cfg
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

type ProjectWithAPIKey struct {
	projectmodels.Project
	APIKey ProjectAPIKey
}

type ProjectAPIKey struct {
	ID         string
	ProjectID  string
	Name       string
	Prefix     string
	Plaintext  string
	Active     bool
	LastUsedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
	RevokedAt  *time.Time
}

func (s *Service) StoreAppCredential(ctx context.Context, ownerUserID string, appID string, payload projectvalidators.StoreAppCredentialRequest) (*projectmodels.AppCredential, error) {
	app, err := s.repository.FindAppForOwner(ctx, ownerUserID, appID)
	if err != nil {
		return nil, err
	}
	if payload.Provider != app.Provider {
		return nil, fmt.Errorf("%w: credential provider must match app provider", ErrInvalidCredential)
	}
	if err := validateCredentialType(app.Provider, payload.CredentialType); err != nil {
		return nil, err
	}

	encryptedSecret, nonce, fingerprint, err := encryptCredentialSecret(payload.Secret)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	credential := projectmodels.AppCredential{
		ID:              uuid.NewString(),
		AppID:           app.ID,
		Provider:        payload.Provider,
		CredentialType:  payload.CredentialType,
		EncryptedSecret: encryptedSecret,
		SecretNonce:     nonce,
		SecretSHA256:    fingerprint,
		Active:          true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.repository.UpsertAppCredential(ctx, credential); err != nil {
		return nil, err
	}
	return &credential, nil
}

func (s *Service) CreateProject(ctx context.Context, ownerUserID string, payload projectvalidators.CreateProjectRequest) (ProjectWithAPIKey, error) {
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
	apiKey, err := newProjectAPIKey(project.ID, "Default")
	if err != nil {
		return ProjectWithAPIKey{}, err
	}
	if err := s.repository.CreateProjectWithAPIKey(ctx, project, apiKey.model); err != nil {
		return ProjectWithAPIKey{}, err
	}
	return ProjectWithAPIKey{Project: project, APIKey: apiKey.public}, nil
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

func (s *Service) GetProjectAPIKey(ctx context.Context, ownerUserID string, projectID string) (*ProjectAPIKey, error) {
	key, err := s.repository.FindActiveProjectAPIKey(ctx, ownerUserID, projectID)
	if err != nil {
		return nil, err
	}
	response := projectAPIKeyFromModel(*key, "")
	return &response, nil
}

func (s *Service) RotateProjectAPIKey(ctx context.Context, ownerUserID string, projectID string) (*ProjectAPIKey, error) {
	apiKey, err := newProjectAPIKey(projectID, "Default")
	if err != nil {
		return nil, err
	}
	if err := s.repository.RotateProjectAPIKey(ctx, ownerUserID, projectID, apiKey.model); err != nil {
		return nil, err
	}
	return &apiKey.public, nil
}

func (s *Service) RevokeProjectAPIKey(ctx context.Context, ownerUserID string, projectID string) error {
	return s.repository.RevokeProjectAPIKey(ctx, ownerUserID, projectID)
}

func (s *Service) AuthenticateProjectAPIKey(ctx context.Context, plaintext string) (*projectmodels.ProjectAPIKey, error) {
	plaintext = strings.TrimSpace(plaintext)
	if plaintext == "" {
		return nil, ErrInvalidAPIKey
	}
	key, err := s.repository.FindActiveProjectAPIKeyByHash(ctx, apiKeyHash(plaintext))
	if err != nil {
		return nil, ErrInvalidAPIKey
	}
	_ = s.repository.MarkProjectAPIKeyUsed(ctx, key.ID, time.Now().UTC())
	return key, nil
}

func (s *Service) ProjectOwnsApp(ctx context.Context, projectID string, appID string) error {
	_, err := s.repository.FindApp(ctx, projectID, appID)
	return err
}

func (s *Service) CreateApp(ctx context.Context, ownerUserID string, projectID string, payload projectvalidators.CreateProjectAppRequest) (projectmodels.ProjectApp, error) {
	if _, err := s.repository.FindProject(ctx, ownerUserID, projectID); err != nil {
		return projectmodels.ProjectApp{}, err
	}
	if err := validateProviderForPlatform(payload.Platform, payload.Provider); err != nil {
		return projectmodels.ProjectApp{}, err
	}
	if err := validateAppIdentifiers(payload.Platform, payload.PackageName, payload.BundleID); err != nil {
		return projectmodels.ProjectApp{}, err
	}
	if err := validateProviderConfiguration(payload.Provider, payload.CloudProjectNumber); err != nil {
		return projectmodels.ProjectApp{}, err
	}

	enabled := true
	if payload.Enabled != nil {
		enabled = *payload.Enabled
	}
	now := time.Now().UTC()
	app := projectmodels.ProjectApp{
		ID:                 uuid.NewString(),
		ProjectID:          projectID,
		Name:               strings.TrimSpace(payload.Name),
		Platform:           payload.Platform,
		Provider:           payload.Provider,
		Environment:        payload.Environment,
		PackageName:        strings.TrimSpace(payload.PackageName),
		BundleID:           strings.TrimSpace(payload.BundleID),
		CloudProjectNumber: strings.TrimSpace(payload.CloudProjectNumber),
		Enabled:            enabled,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	return app, s.repository.CreateApp(ctx, app)
}

func (s *Service) ListApps(ctx context.Context, ownerUserID string, projectID string) ([]projectmodels.ProjectApp, error) {
	if _, err := s.repository.FindProject(ctx, ownerUserID, projectID); err != nil {
		return nil, err
	}
	return s.repository.ListApps(ctx, projectID)
}

func (s *Service) GetApp(ctx context.Context, ownerUserID string, appID string) (*projectmodels.ProjectApp, error) {
	return s.repository.FindAppForOwner(ctx, ownerUserID, appID)
}

func (s *Service) UpdateApp(ctx context.Context, ownerUserID string, appID string, payload projectvalidators.UpdateProjectAppRequest) (*projectmodels.ProjectApp, error) {
	app, err := s.repository.FindAppForOwner(ctx, ownerUserID, appID)
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
	if strings.TrimSpace(payload.CloudProjectNumber) != "" {
		app.CloudProjectNumber = strings.TrimSpace(payload.CloudProjectNumber)
	}
	if payload.Enabled != nil {
		app.Enabled = *payload.Enabled
	}
	if err := validateProviderForPlatform(app.Platform, app.Provider); err != nil {
		return nil, err
	}
	if err := validateAppIdentifiers(app.Platform, app.PackageName, app.BundleID); err != nil {
		return nil, err
	}
	if err := validateProviderConfiguration(app.Provider, app.CloudProjectNumber); err != nil {
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

func (s *Service) DeleteAppByID(ctx context.Context, ownerUserID string, appID string) error {
	return s.repository.DeleteAppForOwner(ctx, ownerUserID, appID)
}

type AppAttestationStatus struct {
	AppID                        string
	Platform                     enum.AppPlatform
	Provider                     enum.AppProvider
	Enabled                      bool
	Configured                   bool
	Identifier                   string
	ExternalConfigurationCheck   string
	RequiresDeveloperCredentials bool
	Issues                       []string
	NextSteps                    []string
	SetupSteps                   []AppSetupStep
}

type AppSetupStep struct {
	Key                string
	Title              string
	Status             string
	VerificationMethod string
	Details            string
	Action             string
	DocsURL            string
}

func (s *Service) GetAppAttestationStatus(ctx context.Context, ownerUserID string, appID string) (*AppAttestationStatus, error) {
	app, err := s.repository.FindAppForOwner(ctx, ownerUserID, appID)
	if err != nil {
		return nil, err
	}

	credentialConfigured := false
	credentialType := credentialTypeForProvider(app.Provider)
	if credentialType != "" {
		_, err := s.repository.FindActiveAppCredential(ctx, app.ID, app.Provider, credentialType)
		if err == nil {
			credentialConfigured = true
		} else if !errors.Is(err, ErrAppCredentialNotFound) {
			return nil, err
		}
	}

	status := appAttestationStatus(*app, credentialConfigured)
	return &status, nil
}

func appAttestationStatus(app projectmodels.ProjectApp, credentialConfigured bool) AppAttestationStatus {
	status := AppAttestationStatus{
		AppID:                        app.ID,
		Platform:                     app.Platform,
		Provider:                     app.Provider,
		Enabled:                      app.Enabled,
		RequiresDeveloperCredentials: true,
		ExternalConfigurationCheck:   "not_configured",
	}

	if !app.Enabled {
		status.Issues = append(status.Issues, "app is disabled in Phylax")
	}

	switch app.Provider {
	case enum.AppProviderPlayIntegrity:
		status.Identifier = strings.TrimSpace(app.PackageName)
		status.SetupSteps = androidPlayIntegritySetupSteps(status.Identifier, app.CloudProjectNumber, credentialConfigured)
		if app.Platform != enum.AppPlatformAndroid {
			status.Issues = append(status.Issues, "Play Integrity requires an Android app")
		}
		if status.Identifier == "" {
			status.Issues = append(status.Issues, "package_name is required for Play Integrity")
		}
		if strings.TrimSpace(app.CloudProjectNumber) == "" {
			status.Issues = append(status.Issues, "cloud_project_number is required for Play Integrity")
		}
	case enum.AppProviderAppAttest:
		status.Identifier = strings.TrimSpace(app.BundleID)
		status.SetupSteps = appleAppAttestSetupSteps(status.Identifier, credentialConfigured)
		if app.Platform != enum.AppPlatformIOS {
			status.Issues = append(status.Issues, "App Attest requires an iOS app")
		}
		if status.Identifier == "" {
			status.Issues = append(status.Issues, "bundle_id is required for App Attest")
		}
	case enum.AppProviderFirebase:
		status.Identifier = strings.TrimSpace(app.PackageName)
		if status.Identifier == "" {
			status.Identifier = strings.TrimSpace(app.BundleID)
		}
		status.SetupSteps = firebaseAppCheckSetupSteps(status.Identifier, credentialConfigured)
		if status.Identifier == "" {
			status.Issues = append(status.Issues, "package_name or bundle_id is required for Firebase App Check")
		}
	default:
		status.Issues = append(status.Issues, "unsupported attestation provider")
	}

	status.ExternalConfigurationCheck = externalConfigurationCheck(status.SetupSteps)
	status.NextSteps = nextSetupSteps(status.SetupSteps)
	status.Configured = app.Enabled && status.Identifier != "" && len(status.Issues) == 0
	return status
}

func androidPlayIntegritySetupSteps(packageName string, cloudProjectNumber string, credentialConfigured bool) []AppSetupStep {
	credentialStatus := "pending"
	credentialDetails := "No active encrypted Google service account credential is stored for this app."
	if credentialConfigured {
		credentialStatus = "completed"
		credentialDetails = "An active encrypted Google service account credential is stored for this app."
	}

	identifierStatus := "completed"
	identifierDetails := "Android package name is present on the Phylax app record."
	if strings.TrimSpace(packageName) == "" {
		identifierStatus = "pending"
		identifierDetails = "Add the Android package name to this Phylax app."
	}

	cloudProjectStatus := "completed"
	cloudProjectDetails := "Google Cloud project number is present on the Phylax app record."
	if strings.TrimSpace(cloudProjectNumber) == "" {
		cloudProjectStatus = "pending"
		cloudProjectDetails = "Add the Google Cloud project number used for Play Integrity token requests."
	}

	return []AppSetupStep{
		{
			Key:                "phylax_app_record",
			Title:              "Create the Android app in Phylax",
			Status:             "completed",
			VerificationMethod: "database",
			Details:            "This app record exists in Phylax.",
		},
		{
			Key:                "android_package_name",
			Title:              "Configure Android package name",
			Status:             identifierStatus,
			VerificationMethod: "database",
			Details:            identifierDetails,
			Action:             "Set package_name to the app id used by Google Play.",
		},
		{
			Key:                "google_cloud_project",
			Title:              "Configure Google Cloud project number",
			Status:             cloudProjectStatus,
			VerificationMethod: "database",
			Details:            cloudProjectDetails,
			Action:             "Set cloud_project_number to the numeric Google Cloud project number linked to Play Integrity.",
			DocsURL:            "https://developer.android.com/google/play/integrity/setup",
		},
		{
			Key:                "play_integrity_api_enabled",
			Title:              "Enable Play Integrity API",
			Status:             "manual",
			VerificationMethod: "manual",
			Details:            "Phylax can use this during token verification, but this setup endpoint does not call Google without an integrity token.",
			Action:             "Enable Play Integrity API in Google Cloud, or link the Cloud project from Play Console.",
			DocsURL:            "https://developer.android.com/google/play/integrity/setup",
		},
		{
			Key:                "play_console_linked",
			Title:              "Link the Cloud project in Google Play Console",
			Status:             "manual",
			VerificationMethod: "manual",
			Details:            "Google Play Console linking is not exposed to Phylax through the package name. The developer must confirm this step.",
			Action:             "In Play Console, open the app, go to Protected with Play, then link the Cloud project under Play Integrity API.",
			DocsURL:            "https://developer.android.com/google/play/integrity/setup",
		},
		{
			Key:                "service_account_credentials",
			Title:              "Configure backend service account credentials",
			Status:             credentialStatus,
			VerificationMethod: "database",
			Details:            credentialDetails,
			Action:             "Upload the service account JSON for the linked Cloud project to this app's credentials.",
		},
		{
			Key:                "android_client_token_flow",
			Title:              "Integrate Play Integrity token requests in the Android app",
			Status:             "manual",
			VerificationMethod: "manual",
			Details:            "The client must request a Play Integrity token and send it to Phylax during attestation verification.",
			Action:             "Use the challenge nonce from Phylax as the Play Integrity nonce or request hash.",
			DocsURL:            "https://developer.android.com/google/play/integrity/standard",
		},
	}
}

func appleAppAttestSetupSteps(bundleID string, credentialConfigured bool) []AppSetupStep {
	identifierStatus := "completed"
	identifierDetails := "iOS bundle id is present on the Phylax app record."
	if strings.TrimSpace(bundleID) == "" {
		identifierStatus = "pending"
		identifierDetails = "Add the iOS bundle id to this Phylax app."
	}

	return []AppSetupStep{
		{
			Key:                "phylax_app_record",
			Title:              "Create the iOS app in Phylax",
			Status:             "completed",
			VerificationMethod: "database",
			Details:            "This app record exists in Phylax.",
		},
		{
			Key:                "ios_bundle_id",
			Title:              "Configure iOS bundle id",
			Status:             identifierStatus,
			VerificationMethod: "database",
			Details:            identifierDetails,
			Action:             "Set bundle_id to the app bundle identifier used in Apple Developer.",
		},
		{
			Key:                "apple_app_attest_capability",
			Title:              "Enable App Attest capability in Apple Developer",
			Status:             "manual",
			VerificationMethod: "manual",
			Details:            "Apple does not provide Phylax a simple public status check from bundle id alone.",
			Action:             "Enable App Attest on the app identifier in Apple Developer and add the entitlement to the iOS app.",
			DocsURL:            "https://developer.apple.com/documentation/devicecheck/establishing-your-app-s-integrity",
		},
		{
			Key:                "apple_team_id",
			Title:              "Configure Apple team id",
			Status:             credentialSetupStatus(credentialConfigured),
			VerificationMethod: "database",
			Details:            credentialSetupDetails(credentialConfigured, "Apple team id is stored as an encrypted app credential.", "No Apple team id credential is stored for this app."),
			Action:             "Store the developer's Apple team id as this app's apple_team credential.",
		},
		{
			Key:                "ios_client_key_flow",
			Title:              "Integrate App Attest key and assertion flow in the iOS app",
			Status:             "manual",
			VerificationMethod: "manual",
			Details:            "The client must create an App Attest key, attest it, then produce assertions for protected requests.",
			Action:             "Send key attestations and assertions to Phylax once the App Attest verifier is implemented.",
			DocsURL:            "https://developer.apple.com/documentation/devicecheck/establishing-your-app-s-integrity",
		},
	}
}

func firebaseAppCheckSetupSteps(identifier string, credentialConfigured bool) []AppSetupStep {
	identifierStatus := "completed"
	identifierDetails := "App identifier is present on the Phylax app record."
	if strings.TrimSpace(identifier) == "" {
		identifierStatus = "pending"
		identifierDetails = "Add package_name for Android or bundle_id for iOS."
	}

	return []AppSetupStep{
		{
			Key:                "phylax_app_record",
			Title:              "Create the app in Phylax",
			Status:             "completed",
			VerificationMethod: "database",
			Details:            "This app record exists in Phylax.",
		},
		{
			Key:                "app_identifier",
			Title:              "Configure app identifier",
			Status:             identifierStatus,
			VerificationMethod: "database",
			Details:            identifierDetails,
		},
		{
			Key:                "firebase_app_check_provider",
			Title:              "Enable Firebase App Check provider",
			Status:             "manual",
			VerificationMethod: "manual",
			Details:            "Phylax needs Firebase project/app credentials before it can inspect provider setup.",
			Action:             "Enable the matching App Check provider in Firebase Console.",
			DocsURL:            "https://firebase.google.com/docs/app-check",
		},
		{
			Key:                "firebase_service_account_credentials",
			Title:              "Configure Firebase service account credentials",
			Status:             credentialSetupStatus(credentialConfigured),
			VerificationMethod: "database",
			Details:            credentialSetupDetails(credentialConfigured, "An active encrypted Firebase service account credential is stored for this app.", "No active Firebase service account credential is stored for this app."),
			Action:             "Upload the Firebase service account JSON for this app.",
		},
	}
}

func credentialTypeForProvider(provider enum.AppProvider) string {
	switch provider {
	case enum.AppProviderPlayIntegrity:
		return "google_service_account"
	case enum.AppProviderFirebase:
		return "firebase_service_account"
	case enum.AppProviderAppAttest:
		return "apple_team"
	default:
		return ""
	}
}

func credentialSetupStatus(configured bool) string {
	if configured {
		return "completed"
	}
	return "pending"
}

func credentialSetupDetails(configured bool, completed string, pending string) string {
	if configured {
		return completed
	}
	return pending
}

func externalConfigurationCheck(steps []AppSetupStep) string {
	hasManual := false
	for _, step := range steps {
		switch step.Status {
		case "pending", "blocked":
			return "pending"
		case "manual":
			hasManual = true
		}
	}
	if hasManual {
		return "manual_verification_required"
	}
	return "verified"
}

func nextSetupSteps(steps []AppSetupStep) []string {
	next := []string{}
	for _, step := range steps {
		if step.Status == "completed" {
			continue
		}
		if step.Action != "" {
			next = append(next, step.Action)
			continue
		}
		next = append(next, step.Title)
	}
	return next
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

func validateProviderConfiguration(provider enum.AppProvider, cloudProjectNumber string) error {
	if provider == enum.AppProviderPlayIntegrity && strings.TrimSpace(cloudProjectNumber) == "" {
		return fmt.Errorf("%w: play_integrity apps require cloud_project_number", ErrInvalidApp)
	}
	return nil
}

func validateProviderForPlatform(platform enum.AppPlatform, provider enum.AppProvider) error {
	switch platform {
	case enum.AppPlatformAndroid:
		if provider != enum.AppProviderPlayIntegrity && provider != enum.AppProviderFirebase {
			return fmt.Errorf("%w: android apps support providers play_integrity or firebase_app_check", ErrInvalidApp)
		}
	case enum.AppPlatformIOS:
		if provider != enum.AppProviderAppAttest && provider != enum.AppProviderFirebase {
			return fmt.Errorf("%w: ios apps support providers app_attest or firebase_app_check", ErrInvalidApp)
		}
	default:
		return fmt.Errorf("%w: unsupported platform", ErrInvalidApp)
	}
	return nil
}

func validateCredentialType(provider enum.AppProvider, credentialType string) error {
	switch provider {
	case enum.AppProviderPlayIntegrity:
		if credentialType != "google_service_account" {
			return fmt.Errorf("%w: Play Integrity requires google_service_account credentials", ErrInvalidCredential)
		}
	case enum.AppProviderFirebase:
		if credentialType != "firebase_service_account" {
			return fmt.Errorf("%w: Firebase App Check requires firebase_service_account credentials", ErrInvalidCredential)
		}
	case enum.AppProviderAppAttest:
		if credentialType != "apple_team" {
			return fmt.Errorf("%w: App Attest requires apple_team credentials", ErrInvalidCredential)
		}
	default:
		return fmt.Errorf("%w: unsupported provider", ErrInvalidCredential)
	}
	return nil
}

type generatedProjectAPIKey struct {
	model  projectmodels.ProjectAPIKey
	public ProjectAPIKey
}

func newProjectAPIKey(projectID string, name string) (generatedProjectAPIKey, error) {
	secret := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, secret); err != nil {
		return generatedProjectAPIKey{}, err
	}
	plaintext := "phx_" + base64.RawURLEncoding.EncodeToString(secret)
	now := time.Now().UTC()
	model := projectmodels.ProjectAPIKey{
		ID:        uuid.NewString(),
		ProjectID: projectID,
		Name:      strings.TrimSpace(name),
		Prefix:    apiKeyPrefix(plaintext),
		KeyHash:   apiKeyHash(plaintext),
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if model.Name == "" {
		model.Name = "Default"
	}
	return generatedProjectAPIKey{
		model:  model,
		public: projectAPIKeyFromModel(model, plaintext),
	}, nil
}

func projectAPIKeyFromModel(key projectmodels.ProjectAPIKey, plaintext string) ProjectAPIKey {
	return ProjectAPIKey{
		ID:         key.ID,
		ProjectID:  key.ProjectID,
		Name:       key.Name,
		Prefix:     key.Prefix,
		Plaintext:  plaintext,
		Active:     key.Active,
		LastUsedAt: key.LastUsedAt,
		CreatedAt:  key.CreatedAt,
		UpdatedAt:  key.UpdatedAt,
		RevokedAt:  key.RevokedAt,
	}
}

func apiKeyPrefix(plaintext string) string {
	if len(plaintext) <= 12 {
		return plaintext
	}
	return plaintext[:12]
}

func apiKeyHash(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

func encryptCredentialSecret(secret string) (string, string, string, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return "", "", "", fmt.Errorf("%w: secret is required", ErrInvalidCredential)
	}
	gcm, err := credentialCipher()
	if err != nil {
		return "", "", "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", "", "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(secret), nil)
	sum := sha256.Sum256([]byte(secret))
	return base64.StdEncoding.EncodeToString(ciphertext), base64.StdEncoding.EncodeToString(nonce), hex.EncodeToString(sum[:]), nil
}

func decryptCredentialSecret(credential projectmodels.AppCredential) (string, error) {
	gcm, err := credentialCipher()
	if err != nil {
		return "", err
	}
	nonce, err := base64.StdEncoding.DecodeString(credential.SecretNonce)
	if err != nil {
		return "", err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(credential.EncryptedSecret)
	if err != nil {
		return "", err
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func credentialCipher() (cipher.AEAD, error) {
	secret := strings.TrimSpace(os.Getenv("PHYLAX_CREDENTIAL_ENCRYPTION_KEY"))
	if len(secret) < 32 {
		return nil, fmt.Errorf("%w: PHYLAX_CREDENTIAL_ENCRYPTION_KEY must be at least 32 characters", ErrInvalidCredential)
	}
	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
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

func (r *GormRepository) CreateProjectWithAPIKey(ctx context.Context, project projectmodels.Project, key projectmodels.ProjectAPIKey) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&project).Error; err != nil {
			return err
		}
		return tx.Create(&key).Error
	})
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

func (r *GormRepository) FindActiveProjectAPIKey(ctx context.Context, ownerUserID string, projectID string) (*projectmodels.ProjectAPIKey, error) {
	var key projectmodels.ProjectAPIKey
	err := r.db.WithContext(ctx).
		Joins("join projects on projects.id = project_api_keys.project_id").
		Where("project_api_keys.project_id = ? and projects.owner_user_id = ? and projects.deleted_at is null and project_api_keys.active = true and project_api_keys.revoked_at is null", projectID, ownerUserID).
		First(&key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrAPIKeyNotFound
	}
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func (r *GormRepository) RotateProjectAPIKey(ctx context.Context, ownerUserID string, projectID string, key projectmodels.ProjectAPIKey) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var project projectmodels.Project
		err := tx.Where("owner_user_id = ? and id = ?", ownerUserID, projectID).First(&project).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrProjectNotFound
		}
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		if err := tx.Model(&projectmodels.ProjectAPIKey{}).
			Where("project_id = ? and active = true and revoked_at is null", projectID).
			Updates(map[string]any{"active": false, "revoked_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
		return tx.Create(&key).Error
	})
}

func (r *GormRepository) RevokeProjectAPIKey(ctx context.Context, ownerUserID string, projectID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var project projectmodels.Project
		err := tx.Where("owner_user_id = ? and id = ?", ownerUserID, projectID).First(&project).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrProjectNotFound
		}
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		result := tx.Model(&projectmodels.ProjectAPIKey{}).
			Where("project_id = ? and active = true and revoked_at is null", projectID).
			Updates(map[string]any{"active": false, "revoked_at": now, "updated_at": now})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrAPIKeyNotFound
		}
		return nil
	})
}

func (r *GormRepository) FindActiveProjectAPIKeyByHash(ctx context.Context, keyHash string) (*projectmodels.ProjectAPIKey, error) {
	var key projectmodels.ProjectAPIKey
	err := r.db.WithContext(ctx).
		Joins("join projects on projects.id = project_api_keys.project_id").
		Where("project_api_keys.key_hash = ? and project_api_keys.active = true and project_api_keys.revoked_at is null and projects.deleted_at is null", keyHash).
		First(&key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrAPIKeyNotFound
	}
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func (r *GormRepository) MarkProjectAPIKeyUsed(ctx context.Context, keyID string, usedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&projectmodels.ProjectAPIKey{}).
		Where("id = ?", keyID).
		Updates(map[string]any{"last_used_at": usedAt, "updated_at": usedAt}).Error
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

func (r *GormRepository) FindAppByID(ctx context.Context, appID string) (*projectmodels.ProjectApp, error) {
	var app projectmodels.ProjectApp
	err := r.db.WithContext(ctx).
		Joins("join projects on projects.id = project_apps.project_id").
		Where("project_apps.id = ? and projects.deleted_at is null", appID).
		First(&app).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrAppNotFound
	}
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (r *GormRepository) FindAppForOwner(ctx context.Context, ownerUserID string, appID string) (*projectmodels.ProjectApp, error) {
	var app projectmodels.ProjectApp
	err := r.db.WithContext(ctx).
		Joins("join projects on projects.id = project_apps.project_id").
		Where("project_apps.id = ? and projects.owner_user_id = ? and projects.deleted_at is null", appID, ownerUserID).
		First(&app).Error
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

func (r *GormRepository) DeleteAppForOwner(ctx context.Context, ownerUserID string, appID string) error {
	app, err := r.FindAppForOwner(ctx, ownerUserID, appID)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Delete(app).Error
}

func (r *GormRepository) UpsertAppCredential(ctx context.Context, credential projectmodels.AppCredential) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&projectmodels.AppCredential{}).
			Where("app_id = ? and provider = ? and credential_type = ? and active = true", credential.AppID, credential.Provider, credential.CredentialType).
			Updates(map[string]any{"active": false, "updated_at": time.Now().UTC()}).Error; err != nil {
			return err
		}
		return tx.Create(&credential).Error
	})
}

func (r *GormRepository) FindActiveAppCredential(ctx context.Context, appID string, provider enum.AppProvider, credentialType string) (*projectmodels.AppCredential, error) {
	var credential projectmodels.AppCredential
	err := r.db.WithContext(ctx).
		Where("app_id = ? and provider = ? and credential_type = ? and active = true", appID, provider, credentialType).
		First(&credential).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrAppCredentialNotFound
	}
	if err != nil {
		return nil, err
	}
	return &credential, nil
}
