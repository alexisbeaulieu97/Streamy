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

	t.Run("returns error when file doesn't exist", func(t *testing.T) {
		t.Parallel()
		err := CheckPathContains(filepath.Join(dir, "nonexistent.txt"), "text")
		require.Error(t, err)
	})

	t.Run("finds text at beginning of file", func(t *testing.T) {
		t.Parallel()
		file := filepath.Join(t.TempDir(), "file.txt")
		require.NoError(t, os.WriteFile(file, []byte("start middle end"), 0o644))
		require.NoError(t, CheckPathContains(file, "start"))
	})

	t.Run("finds text at end of file", func(t *testing.T) {
		t.Parallel()
		file := filepath.Join(t.TempDir(), "file.txt")
		require.NoError(t, os.WriteFile(file, []byte("start middle end"), 0o644))
		require.NoError(t, CheckPathContains(file, "end"))
	})

	t.Run("returns error for empty file when searching for text", func(t *testing.T) {
		t.Parallel()
		file := filepath.Join(t.TempDir(), "empty.txt")
		require.NoError(t, os.WriteFile(file, []byte(""), 0o644))
		err := CheckPathContains(file, "text")
		require.Error(t, err)
	})
}

func TestCheckFileExistsWithDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, CheckFileExists(dir), "directories should pass CheckFileExists")
}

func TestCheckCommandExistsWithCommonCommands(t *testing.T) {
	t.Parallel()

	// Test some common commands that should exist
	require.NoError(t, CheckCommandExists("ls"))
	require.NoError(t, CheckCommandExists("sh"))
}
