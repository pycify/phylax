package models

import (
	"testing"

	"github.com/pycify/phylax/internal/app/types/enum"
	"github.com/stretchr/testify/require"
)

func TestIdentityTableNames(t *testing.T) {
	require.Equal(t, "users", User{}.TableName())
	require.Equal(t, "roles", Role{}.TableName())
	require.Equal(t, "user_roles", UserRole{}.TableName())
}

func TestIdentityConstants(t *testing.T) {
	require.Equal(t, enum.UserType("customer"), enum.UserTypeCustomer)
	require.Equal(t, enum.UserType("admin"), enum.UserTypeAdmin)
	require.Equal(t, enum.AdminRoleName("super_admin"), enum.AdminRoleSuperAdmin)
	require.Equal(t, enum.AdminRoleName("admin"), enum.AdminRoleAdmin)
	require.Equal(t, enum.AdminRoleName("customer_service"), enum.AdminRoleCustomerService)
}
