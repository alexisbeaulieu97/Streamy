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
