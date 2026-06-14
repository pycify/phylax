package env

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadFileSetsUnsetValues(t *testing.T) {
	t.Setenv("PHYLAX_EXISTING", "original")
	path := filepath.Join(t.TempDir(), ".env")
	content := "PHYLAX_ENV=staging\nPHYLAX_EXISTING=ignored\nQUOTED=\"value\"\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	require.NoError(t, LoadFile(path))

	require.Equal(t, "staging", os.Getenv("PHYLAX_ENV"))
	require.Equal(t, "original", os.Getenv("PHYLAX_EXISTING"))
	require.Equal(t, "value", os.Getenv("QUOTED"))
}

func TestLoadFileReturnsMissingFileError(t *testing.T) {
	require.ErrorIs(t, LoadFile(filepath.Join(t.TempDir(), ".env")), os.ErrNotExist)
}

func TestValidateRequiredReportsMissingSecrets(t *testing.T) {
	t.Setenv("PHYLAX_ENV", "")
	t.Setenv("PHYLAX_TOKEN_SECRET", "")
	t.Setenv("PHYLAX_CREDENTIAL_ENCRYPTION_KEY", "")
	t.Setenv("DATABASE_URL", "")

	err := ValidateRequired()
	require.Error(t, err)
	require.Contains(t, err.Error(), "PHYLAX_ENV")
	require.Contains(t, err.Error(), "PHYLAX_TOKEN_SECRET")
	require.Contains(t, err.Error(), "PHYLAX_CREDENTIAL_ENCRYPTION_KEY")
	require.Contains(t, err.Error(), "DATABASE_URL")
}

func TestValidateRequiredAllowsOptionalSMTPPassword(t *testing.T) {
	t.Setenv("PHYLAX_ENV", "production")
	t.Setenv("PHYLAX_TOKEN_SECRET", "test-secret-with-more-than-32-bytes")
	t.Setenv("PHYLAX_CREDENTIAL_ENCRYPTION_KEY", "test-encryption-key-with-32-characters")
	t.Setenv("DATABASE_URL", "postgresql://example")
	t.Setenv("MAIL_DRIVER", "smtp")
	t.Setenv("MAIL_FROM_EMAIL", "no-reply@example.com")
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_PORT", "587")
	t.Setenv("SMTP_PASS", "")

	require.NoError(t, ValidateRequired())
}

func TestValidateRequiredRejectsInvalidSMTPPort(t *testing.T) {
	t.Setenv("PHYLAX_ENV", "production")
	t.Setenv("PHYLAX_TOKEN_SECRET", "test-secret-with-more-than-32-bytes")
	t.Setenv("PHYLAX_CREDENTIAL_ENCRYPTION_KEY", "test-encryption-key-with-32-characters")
	t.Setenv("DATABASE_URL", "postgresql://example")
	t.Setenv("MAIL_DRIVER", "smtp")
	t.Setenv("MAIL_FROM_EMAIL", "no-reply@example.com")
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_PORT", "invalid")

	err := ValidateRequired()
	require.Error(t, err)
	require.Contains(t, err.Error(), "SMTP_PORT")
}
