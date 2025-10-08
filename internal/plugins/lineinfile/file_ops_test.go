package lineinfileplugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
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

func TestCreateBackupCustomDirectory(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "sample.txt")
	backupDir := filepath.Join(dir, "backups")

	backupPath, err := createBackup(target, backupDir, []byte("custom"), 0o640)
	require.NoError(t, err)
	require.Equal(t, backupDir, filepath.Dir(backupPath))
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
