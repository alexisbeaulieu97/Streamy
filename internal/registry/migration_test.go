package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	assert "github.com/stretchr/testify/assert"
	require "github.com/stretchr/testify/require"
)

func TestMigrateRegistryFileUpgrade(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")

	original := `{
  "version": "1.0",
  "pipelines": [
    {
      "id": "app",
      "name": "App",
      "path": "/tmp/app.yaml",
      "description": "",
      "registered_at": "2025-10-08T00:00:00Z"
    }
  ]
}`

	require.NoError(t, os.WriteFile(path, []byte(original), 0o600))

	require.NoError(t, MigrateRegistryFile(path))

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var file File
	require.NoError(t, json.Unmarshal(data, &file))

	assert.Equal(t, registryFileVersion, file.Version)
	require.Len(t, file.Pipelines, 1)

	expectedTime, err := time.Parse(time.RFC3339, "2025-10-08T00:00:00Z")
	require.NoError(t, err)

	expected := Pipeline{
		ID:           "app",
		Name:         "App",
		Path:         "/tmp/app.yaml",
		Description:  "",
		RegisteredAt: expectedTime,
		Dependencies: []string{},
	}

	assert.Equal(t, expected, file.Pipelines[0])

	backupPath := fmt.Sprintf("%s.v1.0.bak", path)
	_, err = os.Stat(backupPath)
	require.NoError(t, err, "expected registry backup file")
}

func TestMigrateRegistryFileMissing(t *testing.T) {
	t.Parallel()

	require.NoError(t, MigrateRegistryFile(filepath.Join(t.TempDir(), "registry.json")))
}

func TestMigrateStatusCacheFileUpgrade(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "status-cache.json")

	original := `{
  "version": "1.0",
  "statuses": {
    "app@1.0": {
      "status": "satisfied",
      "last_run": "2025-10-08T00:00:00Z",
      "summary": "ok",
      "step_count": 3
    }
  }
}`

	require.NoError(t, os.WriteFile(path, []byte(original), 0o600))

	require.NoError(t, MigrateStatusCacheFile(path))

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var file StatusCacheFile
	require.NoError(t, json.Unmarshal(data, &file))

	assert.Equal(t, statusCacheVersion, file.Version)
	require.Len(t, file.Statuses, 1)

	status, ok := file.Statuses["app@1.0"]
	require.True(t, ok, "expected status entry for app@1.0")

	expectedLastRun, err := time.Parse(time.RFC3339, "2025-10-08T00:00:00Z")
	require.NoError(t, err)

	assert.Equal(t, CachedStatus{
		Status:    PipelineStatus("satisfied"),
		LastRun:   expectedLastRun,
		Summary:   "ok",
		StepCount: 3,
	}, status)

	backupPath := fmt.Sprintf("%s.v1.0.bak", path)
	_, err = os.Stat(backupPath)
	require.NoError(t, err, "expected status cache backup file")
}

func TestMigrateStatusCacheFileMissing(t *testing.T) {
	t.Parallel()

	require.NoError(t, MigrateStatusCacheFile(filepath.Join(t.TempDir(), "status-cache.json")))
}
