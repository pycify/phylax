package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadReadsEnvironment(t *testing.T) {
	t.Setenv("PHYLAX_ENV", "production")
	t.Setenv("PORT", "9090")
	t.Setenv("DATABASE_URL", "postgresql://example")
	t.Setenv("PHYLAX_TOKEN_SECRET", "test-secret-with-more-than-32-bytes")
	t.Setenv("PHYLAX_TOKEN_ISSUER", "phylax-test")
	t.Setenv("PHYLAX_TOKEN_TTL_SECONDS", "1200")
	t.Setenv("PHYLAX_CHALLENGE_TTL_SECONDS", "600")

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "production", cfg.Environment)
	require.Equal(t, ":9090", cfg.Server.Addr)
	require.Equal(t, "postgres", cfg.Storage.Driver)
	require.Equal(t, "postgresql://example", cfg.Storage.DatabaseURL)
	require.Equal(t, "test-secret-with-more-than-32-bytes", cfg.Token.Secret)
	require.Equal(t, "phylax-test", cfg.Token.Issuer)
	require.Equal(t, 1200, cfg.Token.TTLSeconds)
	require.Equal(t, 600, cfg.Challenges.TTLSeconds)
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("PHYLAX_ENV", "development")
	t.Setenv("PHYLAX_TOKEN_SECRET", "test-secret-with-more-than-32-bytes")

	_, err := Load()
	require.Error(t, err)
	require.Contains(t, err.Error(), "DATABASE_URL")
}
