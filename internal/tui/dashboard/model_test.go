package dashboard

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

func TestSortPipelines(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	assert.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	assert.NoError(t, err)

	pipelines := []registry.Pipeline{
		{ID: "p1", Name: "Pipeline 1", Status: registry.StatusSatisfied},
		{ID: "p2", Name: "Pipeline 2", Status: registry.StatusFailed},
		{ID: "p3", Name: "Pipeline 3", Status: registry.StatusDrifted},
		{ID: "p4", Name: "Pipeline 4", Status: registry.StatusUnknown},
	}

	// Pre-populate cache with statuses
	for _, p := range pipelines {
		cache.Set(p.ID, registry.CachedStatus{Status: p.Status})
	}

	m := NewModel(pipelines, reg, cache, nil)

	// After sorting, order should be: failed, drifted, satisfied, unknown
	assert.Equal(t, "p2", m.pipelines[0].ID) // Failed
	assert.Equal(t, "p3", m.pipelines[1].ID) // Drifted
	assert.Equal(t, "p1", m.pipelines[2].ID) // Satisfied
	assert.Equal(t, "p4", m.pipelines[3].ID) // Unknown
}

func TestCountByStatus(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	assert.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	assert.NoError(t, err)

	pipelines := []registry.Pipeline{
		{ID: "p1", Status: registry.StatusSatisfied},
		{ID: "p2", Status: registry.StatusSatisfied},
		{ID: "p3", Status: registry.StatusFailed},
		{ID: "p4", Status: registry.StatusDrifted},
	}

	// Pre-populate cache with statuses
	for _, p := range pipelines {
		cache.Set(p.ID, registry.CachedStatus{Status: p.Status})
	}

	m := NewModel(pipelines, reg, cache, nil)
	counts := m.CountByStatus()

	assert.Equal(t, 2, counts[registry.StatusSatisfied])
	assert.Equal(t, 1, counts[registry.StatusFailed])
	assert.Equal(t, 1, counts[registry.StatusDrifted])
	assert.Equal(t, 0, counts[registry.StatusUnknown])
}

func TestMoveCursor(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	assert.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	assert.NoError(t, err)

	pipelines := []registry.Pipeline{
		{ID: "p1", Name: "Pipeline 1"},
		{ID: "p2", Name: "Pipeline 2"},
		{ID: "p3", Name: "Pipeline 3"},
	}

	m := NewModel(pipelines, reg, cache, nil)

	// Initial cursor should be 0
	assert.Equal(t, 0, m.cursor)

	// Move down
	m.MoveCursorDown()
	assert.Equal(t, 1, m.cursor)

	m.MoveCursorDown()
	assert.Equal(t, 2, m.cursor)

	// Move down should wrap to 0
	m.MoveCursorDown()
	assert.Equal(t, 0, m.cursor)

	// Move up should wrap to last
	m.MoveCursorUp()
	assert.Equal(t, 2, m.cursor)

	m.MoveCursorUp()
	assert.Equal(t, 1, m.cursor)
}

func TestGetSelectedPipeline(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	assert.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	assert.NoError(t, err)

	pipelines := []registry.Pipeline{
		{ID: "p1", Name: "Pipeline 1"},
		{ID: "p2", Name: "Pipeline 2"},
	}

	m := NewModel(pipelines, reg, cache, nil)
	m.cursor = 1

	selected, ok := m.GetSelectedPipeline()
	assert.True(t, ok)
	assert.Equal(t, "p2", selected.ID)
}

func TestUpdatePipelineStatus(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	assert.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	assert.NoError(t, err)

	pipelines := []registry.Pipeline{
		{ID: "p1", Name: "Pipeline 1", Status: registry.StatusUnknown},
	}

	m := NewModel(pipelines, reg, cache, nil)

	now := time.Now()
	m.UpdatePipelineStatus("p1", registry.StatusSatisfied, now)

	assert.Equal(t, registry.StatusSatisfied, m.pipelines[0].Status)
	assert.Equal(t, now, m.pipelines[0].LastRun)
}

func TestGetPipelineByID(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	assert.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	assert.NoError(t, err)

	pipelines := []registry.Pipeline{
		{ID: "test-1", Name: "Test 1"},
		{ID: "test-2", Name: "Test 2"},
	}

	m := NewModel(pipelines, reg, cache, nil)

	// Found
	pipeline, index, ok := m.GetPipelineByID("test-2")
	assert.True(t, ok)
	assert.Equal(t, "test-2", pipeline.ID)
	assert.Equal(t, 1, index)

	// Not found
	_, _, ok = m.GetPipelineByID("nonexistent")
	assert.False(t, ok)
}

func TestSetCursor(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	assert.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	assert.NoError(t, err)

	pipelines := []registry.Pipeline{
		{ID: "1"},
		{ID: "2"},
		{ID: "3"},
	}

	m := NewModel(pipelines, reg, cache, nil)

	// Valid index
	m.SetCursor(2)
	assert.Equal(t, 2, m.cursor)

	// Invalid negative
	m.SetCursor(-1)
	assert.Equal(t, 2, m.cursor) // Should not change

	// Invalid out of bounds
	m.SetCursor(10)
	assert.Equal(t, 2, m.cursor) // Should not change
}

func TestIsLoading(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	assert.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	assert.NoError(t, err)

	m := NewModel([]registry.Pipeline{}, reg, cache, nil)

	// Not loading
	assert.False(t, m.IsLoading("test-1"))

	// Loading
	m.loading["test-1"] = true
	assert.True(t, m.IsLoading("test-1"))
}

func TestHasError(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	assert.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	assert.NoError(t, err)

	m := NewModel([]registry.Pipeline{}, reg, cache, nil)

	// No error
	assert.False(t, m.HasError("test-1"))

	// Has error
	m.errors["test-1"] = "test error"
	assert.True(t, m.HasError("test-1"))
}

func TestGetError(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	assert.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	assert.NoError(t, err)

	m := NewModel([]registry.Pipeline{}, reg, cache, nil)

	m.errors["test-1"] = "test error message"
	assert.Equal(t, "test error message", m.GetError("test-1"))
	assert.Equal(t, "", m.GetError("nonexistent"))
}

func TestClearError(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	assert.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	assert.NoError(t, err)

	m := NewModel([]registry.Pipeline{}, reg, cache, nil)

	m.errors["test-1"] = "test error"
	m.ClearError("test-1")
	assert.False(t, m.HasError("test-1"))
}

func TestGetViewMode(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	assert.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	assert.NoError(t, err)

	m := NewModel([]registry.Pipeline{}, reg, cache, nil)

	assert.Equal(t, ViewList, m.GetViewMode())

	m.viewMode = ViewDetail
	assert.Equal(t, ViewDetail, m.GetViewMode())
}

func TestIsRefreshing(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	assert.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	assert.NoError(t, err)

	m := NewModel([]registry.Pipeline{}, reg, cache, nil)

	assert.False(t, m.IsRefreshing())

	m.refreshing = true
	assert.True(t, m.IsRefreshing())
}

func TestGetRefreshTotal(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	assert.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	assert.NoError(t, err)

	m := NewModel([]registry.Pipeline{}, reg, cache, nil)

	assert.Equal(t, 0, m.GetRefreshTotal())

	m.refreshTotal = 10
	assert.Equal(t, 10, m.GetRefreshTotal())
}

func TestGetStatusPriority(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	assert.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	assert.NoError(t, err)

	m := NewModel([]registry.Pipeline{}, reg, cache, nil)

	assert.Equal(t, 0, m.getStatusPriority(registry.StatusFailed))
	assert.Equal(t, 1, m.getStatusPriority(registry.StatusDrifted))
	assert.Equal(t, 2, m.getStatusPriority(registry.StatusSatisfied))
	assert.Equal(t, 3, m.getStatusPriority(registry.StatusVerifying))
	assert.Equal(t, 3, m.getStatusPriority(registry.StatusApplying))
	assert.Equal(t, 4, m.getStatusPriority(registry.StatusUnknown))
}

func TestInit(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	assert.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	assert.NoError(t, err)

	pipelines := []registry.Pipeline{
		{ID: "test-1", Name: "Test 1", Status: registry.StatusSatisfied},
	}

	m := NewModel(pipelines, reg, cache, nil)
	cmd := m.Init()

	assert.NotNil(t, cmd, "Init should return a command")
}
