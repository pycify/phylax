package enum

type UserType string

const (
	UserTypeCustomer UserType = "customer"
	UserTypeAdmin    UserType = "admin"
)

type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusSuspended UserStatus = "suspended"
)

type AdminRoleName string

const (
	AdminRoleSuperAdmin      AdminRoleName = "super_admin"
	AdminRoleAdmin           AdminRoleName = "admin"
	AdminRoleCustomerService AdminRoleName = "customer_service"
)

type AuthProvider string

const (
	AuthProviderStandard AuthProvider = "standard"
	AuthProviderGoogle   AuthProvider = "google"
	AuthProviderGithub   AuthProvider = "github"
	AuthProviderApple    AuthProvider = "apple"
)
