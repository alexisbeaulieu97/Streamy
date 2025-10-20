package lineinfileplugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	require "github.com/stretchr/testify/require"
)

func TestCreateBackupDefaultDirectory(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "sample.txt")
	require.NoError(t, os.WriteFile(target, []byte("original"), 0o644))

	backupPath, err := createBackup(target, "", []byte("backup data"), 0o600)
	require.NoError(t, err)

	info, err := os.Stat(backupPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	require.Equal(t, filepath.Dir(target), filepath.Dir(backupPath))
	require.True(t, strings.HasPrefix(filepath.Base(backupPath), "sample.txt."))
	require.True(t, strings.HasSuffix(filepath.Base(backupPath), ".bak"))

	data, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	require.Equal(t, "backup data", string(data))
}

func TestReadFileStateNonExistent(t *testing.T) {
	cfg := &LineInFileConfig{File: "/tmp/non-existent-file.txt"}
	state, err := readFileState(cfg)
	require.NoError(t, err)
	require.False(t, state.Exists)
}

func TestSplitJoinLines(t *testing.T) {
	testCases := []struct {
		name             string
		content          string
		expectedLines    []string
		expectedTrailing bool
	}{
		{"empty", "", []string{}, false},
		{"single line", "hello", []string{"hello"}, false},
		{"single line with newline", "hello\n", []string{"hello"}, true},
		{"multiple lines", "hello\nworld", []string{"hello", "world"}, false},
		{"multiple lines with newline", "hello\nworld\n", []string{"hello", "world"}, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			lines, trailing := splitLines(tc.content)
			require.Equal(t, tc.expectedLines, lines)
			require.Equal(t, tc.expectedTrailing, trailing)

			joined := joinLines(lines, trailing)
			require.Equal(t, tc.content, joined)
		})
	}
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	path, err := expandPath("~/test")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(home, "test"), path)
}

func TestEncodeDecodeContent(t *testing.T) {
	content := "Olá Mundo"

	encoded, err := encodeContent(content, "latin-1")
	require.NoError(t, err)

	decoded, err := decodeContent(encoded, "latin-1")
	require.NoError(t, err)
	require.Equal(t, content, decoded)
}

func TestEncodingSupportHelpers(t *testing.T) {
	require.True(t, isSupportedEncoding("utf-8"))
	require.True(t, isSupportedEncoding("LATIN1"))
	require.False(t, isSupportedEncoding("utf-32"))

	require.NotNil(t, encodingByName("latin-1"))
	require.Nil(t, encodingByName("unknown-encoding"))
}
