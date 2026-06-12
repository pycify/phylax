package models

import (
	"testing"

	"github.com/pycify/phylax/internal/app/types/enum"
	"github.com/stretchr/testify/require"
)

func TestProjectTableNames(t *testing.T) {
	require.Equal(t, "projects", Project{}.TableName())
	require.Equal(t, "project_members", ProjectMember{}.TableName())
	require.Equal(t, "project_apps", ProjectApp{}.TableName())
}

func TestProjectConstants(t *testing.T) {
	require.Equal(t, enum.AppPlatform("android"), enum.AppPlatformAndroid)
	require.Equal(t, enum.AppPlatform("ios"), enum.AppPlatformIOS)
	require.Equal(t, enum.AppProvider("play_integrity"), enum.AppProviderPlayIntegrity)
	require.Equal(t, enum.AppProvider("app_attest"), enum.AppProviderAppAttest)
	require.Equal(t, enum.AppEnvironment("production"), enum.AppEnvironmentProduction)
	require.Equal(t, enum.ProjectRole("owner"), enum.ProjectRoleOwner)
}
