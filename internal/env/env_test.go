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
