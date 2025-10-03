package validation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckCommandExists(t *testing.T) {
	t.Parallel()

	require.NoError(t, CheckCommandExists("echo"))

	err := CheckCommandExists("command-that-should-not-exist-12345")
	require.Error(t, err)
}

func TestCheckFileExists(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "exists.txt")
	require.NoError(t, os.WriteFile(file, []byte("hello"), 0o644))

	require.NoError(t, CheckFileExists(file))
	require.Error(t, CheckFileExists(filepath.Join(dir, "missing.txt")))
}

func TestCheckPathContains(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "content.txt")
	require.NoError(t, os.WriteFile(file, []byte("the quick brown fox"), 0o644))

	require.NoError(t, CheckPathContains(file, "quick"))
	require.Error(t, CheckPathContains(file, "lazy"))
}
