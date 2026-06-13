package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadAppliesEnvironmentOverrides(t *testing.T) {
	t.Setenv("PHYLAX_ENV", "production")
	t.Setenv("PORT", "9090")
	t.Setenv("PHYLAX_STORAGE_DRIVER", "memory")
	t.Setenv("PHYLAX_TOKEN_SECRET", "test-secret-with-more-than-32-bytes")

	path := writeConfig(t, `{
	  "token": {"secret": "file-secret-with-more-than-32-bytes"}
	}`)

	cfg, err := Load(path)
	require.NoError(t, err)
	require.Equal(t, "production", cfg.Environment)
	require.Equal(t, ":9090", cfg.Server.Addr)
	require.Equal(t, "test-secret-with-more-than-32-bytes", cfg.Token.Secret)
}

func TestLoadDoesNotRequireStaticApps(t *testing.T) {
	path := writeConfig(t, `{
	  "token": {"secret": "file-secret-with-more-than-32-bytes"}
	}`)

	cfg, err := Load(path)
	require.NoError(t, err)
	require.Equal(t, "file-secret-with-more-than-32-bytes", cfg.Token.Secret)
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "phylax.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}
