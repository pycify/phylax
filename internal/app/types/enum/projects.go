package enum

type ProjectMemberStatus string

const (
	ProjectMemberStatusActive  ProjectMemberStatus = "active"
	ProjectMemberStatusInvited ProjectMemberStatus = "invited"
	ProjectMemberStatusRevoked ProjectMemberStatus = "revoked"
)

type ProjectRole string

const (
	ProjectRoleOwner  ProjectRole = "owner"
	ProjectRoleAdmin  ProjectRole = "admin"
	ProjectRoleMember ProjectRole = "member"
	ProjectRoleViewer ProjectRole = "viewer"
)

type AppPlatform string

const (
	AppPlatformAndroid AppPlatform = "android"
	AppPlatformIOS     AppPlatform = "ios"
)

type AppProvider string

const (
	AppProviderPlayIntegrity AppProvider = "play_integrity"
	AppProviderAppAttest     AppProvider = "app_attest"
	AppProviderFirebase      AppProvider = "firebase_app_check"
)

type AppEnvironment string

const (
	AppEnvironmentDevelopment AppEnvironment = "development"
	AppEnvironmentStaging     AppEnvironment = "staging"
	AppEnvironmentProduction  AppEnvironment = "production"
)
