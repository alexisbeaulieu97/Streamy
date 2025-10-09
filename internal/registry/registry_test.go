package registry

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryNew(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	reg, err := NewRegistry(registryPath)
	require.NoError(t, err)
	assert.NotNil(t, reg)
	assert.Empty(t, reg.List())
}

func TestRegistryLoadExisting(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	// Copy test fixture
	testData, err := os.ReadFile("../../testdata/registry/single-pipeline.json")
	require.NoError(t, err)
	err = os.WriteFile(registryPath, testData, 0644)
	require.NoError(t, err)

	reg, err := NewRegistry(registryPath)
	require.NoError(t, err)
	pipelines := reg.List()
	assert.Len(t, pipelines, 1)
	assert.Equal(t, "dev-env", pipelines[0].ID)
	assert.Equal(t, "Development Environment", pipelines[0].Name)
}

func TestRegistryAdd(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	reg, err := NewRegistry(registryPath)
	require.NoError(t, err)

	pipeline := Pipeline{
		ID:           "test-pipeline",
		Name:         "Test Pipeline",
		Path:         "/path/to/config.yaml",
		Description:  "Test description",
		RegisteredAt: time.Now(),
	}

	err = reg.Add(pipeline)
	require.NoError(t, err)

	pipelines := reg.List()
	assert.Len(t, pipelines, 1)
	assert.Equal(t, "test-pipeline", pipelines[0].ID)
}

func TestRegistryAddDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	reg, err := NewRegistry(registryPath)
	require.NoError(t, err)

	pipeline := Pipeline{
		ID:           "test-pipeline",
		Name:         "Test Pipeline",
		Path:         "/path/to/config.yaml",
		Description:  "Test description",
		RegisteredAt: time.Now(),
	}

	err = reg.Add(pipeline)
	require.NoError(t, err)

	// Try to add again with same ID
	err = reg.Add(pipeline)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestRegistryGet(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	reg, err := NewRegistry(registryPath)
	require.NoError(t, err)

	pipeline := Pipeline{
		ID:           "test-pipeline",
		Name:         "Test Pipeline",
		Path:         "/path/to/config.yaml",
		Description:  "Test description",
		RegisteredAt: time.Now(),
	}

	err = reg.Add(pipeline)
	require.NoError(t, err)

	retrieved, err := reg.Get("test-pipeline")
	require.NoError(t, err)
	assert.Equal(t, "test-pipeline", retrieved.ID)
	assert.Equal(t, "Test Pipeline", retrieved.Name)
}

func TestRegistryGetNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	reg, err := NewRegistry(registryPath)
	require.NoError(t, err)

	_, err = reg.Get("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRegistryUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	reg, err := NewRegistry(registryPath)
	require.NoError(t, err)

	pipeline := Pipeline{
		ID:           "test-pipeline",
		Name:         "Test Pipeline",
		Path:         "/path/to/config.yaml",
		Description:  "Original description",
		RegisteredAt: time.Now(),
	}

	err = reg.Add(pipeline)
	require.NoError(t, err)

	// Update the pipeline
	pipeline.Description = "Updated description"
	err = reg.Update(pipeline)
	require.NoError(t, err)

	retrieved, err := reg.Get("test-pipeline")
	require.NoError(t, err)
	assert.Equal(t, "Updated description", retrieved.Description)
}

func TestRegistryRemove(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	reg, err := NewRegistry(registryPath)
	require.NoError(t, err)

	pipeline := Pipeline{
		ID:           "test-pipeline",
		Name:         "Test Pipeline",
		Path:         "/path/to/config.yaml",
		Description:  "Test description",
		RegisteredAt: time.Now(),
	}

	err = reg.Add(pipeline)
	require.NoError(t, err)

	err = reg.Remove("test-pipeline")
	require.NoError(t, err)

	assert.Empty(t, reg.List())
}

func TestRegistrySave(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	reg, err := NewRegistry(registryPath)
	require.NoError(t, err)

	pipeline := Pipeline{
		ID:           "test-pipeline",
		Name:         "Test Pipeline",
		Path:         "/path/to/config.yaml",
		Description:  "Test description",
		RegisteredAt: time.Now(),
	}

	err = reg.Add(pipeline)
	require.NoError(t, err)

	err = reg.Save()
	require.NoError(t, err)

	// Load in a new registry instance
	reg2, err := NewRegistry(registryPath)
	require.NoError(t, err)

	pipelines := reg2.List()
	assert.Len(t, pipelines, 1)
	assert.Equal(t, "test-pipeline", pipelines[0].ID)
}
