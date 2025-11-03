package registry

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	assert "github.com/stretchr/testify/assert"
	require "github.com/stretchr/testify/require"
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
	err = os.WriteFile(registryPath, testData, 0o644)
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

func TestPipelinesByID(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	reg, err := NewRegistry(registryPath)
	require.NoError(t, err)

	now := time.Now().UTC()

	require.NoError(t, reg.Add(Pipeline{
		ID:           "backend@1.0",
		Name:         "Backend",
		Path:         "/pipelines/backend.yaml",
		RegisteredAt: now,
	}))

	require.NoError(t, reg.Add(Pipeline{
		ID:           "frontend@1.0",
		Name:         "Frontend",
		Path:         "/pipelines/frontend.yaml",
		Dependencies: []string{"backend@1.0"},
		RegisteredAt: now,
	}))

	byID := reg.PipelinesByID()

	require.Len(t, byID, 2)
	assert.Equal(t, "Backend", byID["backend@1.0"].Name)
	assert.Equal(t, []string{"backend@1.0"}, byID["frontend@1.0"].Dependencies)
}

func TestDependentsIndex(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	reg, err := NewRegistry(registryPath)
	require.NoError(t, err)

	now := time.Now().UTC()

	require.NoError(t, reg.Add(Pipeline{
		ID:           "db@1.0",
		Name:         "Database",
		RegisteredAt: now,
	}))

	require.NoError(t, reg.Add(Pipeline{
		ID:           "backend@1.0",
		Name:         "Backend",
		Dependencies: []string{"db@1.0"},
		RegisteredAt: now,
	}))

	require.NoError(t, reg.Add(Pipeline{
		ID:           "frontend@1.0",
		Name:         "Frontend",
		Dependencies: []string{"backend@1.0", "db@1.0"},
		RegisteredAt: now,
	}))

	index := reg.DependentsIndex()

	assert.ElementsMatch(t, []string{"backend@1.0", "frontend@1.0"}, index["db@1.0"])
	assert.ElementsMatch(t, []string{"frontend@1.0"}, index["backend@1.0"])
}

func TestFindUnregisteredDependencies(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	reg, err := NewRegistry(registryPath)
	require.NoError(t, err)

	require.NoError(t, reg.Add(Pipeline{ID: "db@1.0"}))
	require.NoError(t, reg.Add(Pipeline{ID: "cache@1.0"}))

	missing := reg.FindUnregisteredDependencies([]string{"db@1.0", "frontend@1.0", "cache@1.0", "queue@1.0"})
	assert.Equal(t, []string{"frontend@1.0", "queue@1.0"}, missing)
}

func TestValidateDependencyCycles_AllowsAcyclicGraph(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	reg, err := NewRegistry(registryPath)
	require.NoError(t, err)

	require.NoError(t, reg.Add(Pipeline{
		ID:           "backend@1.0",
		Dependencies: []string{"db@1.0"},
	}))

	require.NoError(t, reg.Add(Pipeline{
		ID:           "db@1.0",
		Dependencies: nil,
	}))

	err = reg.ValidateDependencyCycles("frontend@1.0", []string{"backend@1.0"})
	require.NoError(t, err)
}

func TestValidateDependencyCyclesDetectsCycle(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	reg, err := NewRegistry(registryPath)
	require.NoError(t, err)

	require.NoError(t, reg.Add(Pipeline{
		ID:           "backend@1.0",
		Dependencies: []string{"frontend@1.0"},
	}))

	err = reg.ValidateDependencyCycles("frontend@1.0", []string{"backend@1.0"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dependency cycle detected")
}

func TestValidateDependencyCyclesRejectsSelfDependency(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	reg, err := NewRegistry(registryPath)
	require.NoError(t, err)

	err = reg.ValidateDependencyCycles("pipeline@1.0", []string{"pipeline@1.0"})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSelfDependency)
}

func TestRegistryReconcileStatuses(t *testing.T) {
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")
	statusCachePath := filepath.Join(tmpDir, "status-cache.json")

	reg, err := NewRegistry(registryPath)
	require.NoError(t, err)

	cache, err := NewStatusCache(statusCachePath)
	require.NoError(t, err)

	now := time.Now().UTC()

	require.NoError(t, reg.Add(Pipeline{ID: "db@1.0", Name: "Database", RegisteredAt: now}))
	require.NoError(t, reg.Add(Pipeline{ID: "api@1.0", Name: "API", Dependencies: []string{"db@1.0"}, RegisteredAt: now}))
	require.NoError(t, reg.Add(Pipeline{
		ID:           "frontend@1.0",
		Name:         "Frontend",
		Dependencies: []string{"api@1.0", "cdn@1.0"},
		Status:       StatusBlocked,
		BlockedBy:    []string{"cdn@1.0"},
		RegisteredAt: now,
	}))

	unblocked := reg.ReconcileStatuses(cache)
	assert.Empty(t, unblocked)

	frontend, err := reg.Get("frontend@1.0")
	require.NoError(t, err)
	assert.Equal(t, StatusBlocked, frontend.Status)
	assert.Equal(t, []string{"cdn@1.0"}, frontend.BlockedBy)

	cachedStatus, ok := cache.Get("frontend@1.0")
	assert.True(t, ok)
	assert.Equal(t, StatusBlocked, cachedStatus.Status)
	assert.Contains(t, cachedStatus.Summary, "Missing dependencies")

	require.NoError(t, reg.Add(Pipeline{ID: "cdn@1.0", Name: "CDN", RegisteredAt: now}))

	unblocked = reg.ReconcileStatuses(cache)
	require.Equal(t, []string{"frontend@1.0"}, unblocked)

	frontend, err = reg.Get("frontend@1.0")
	require.NoError(t, err)
	assert.Equal(t, StatusReady, frontend.Status)
	assert.Empty(t, frontend.BlockedBy)

	cachedStatus, ok = cache.Get("frontend@1.0")
	assert.True(t, ok)
	assert.Equal(t, StatusReady, cachedStatus.Status)
	assert.Contains(t, cachedStatus.Summary, "Dependencies satisfied")
}

func TestResolveCanonicalID(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")

	reg, err := NewRegistry(registryPath)
	require.NoError(t, err)

	now := time.Now().UTC()

	require.NoError(t, reg.Add(Pipeline{ID: "svc@1.0", RegisteredAt: now}))
	require.NoError(t, reg.Add(Pipeline{ID: "svc@2.0", RegisteredAt: now}))
	require.NoError(t, reg.Add(Pipeline{ID: "svc@1.5", RegisteredAt: now}))

	resolved, err := reg.ResolveCanonicalID("svc@latest")
	require.NoError(t, err)
	assert.Equal(t, "svc@2.0", resolved)

	resolvedSame, err := reg.ResolveCanonicalID("svc@1.5")
	require.NoError(t, err)
	assert.Equal(t, "svc@1.5", resolvedSame)

	_, err = reg.ResolveCanonicalID("unknown@latest")
	require.Error(t, err)

	_, err = reg.ResolveCanonicalID("svc")
	require.Error(t, err)
}
