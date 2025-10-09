package dashboard

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

func TestRenderDetailView(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	pipelines := []registry.Pipeline{
		{
			ID:      "test-1",
			Name:    "Test Pipeline",
			Path:    "/test/path.yaml",
			Status:  registry.StatusSatisfied,
			LastRun: time.Now(),
		},
	}

	m := NewModel(pipelines, reg, cache, nil)
	m.width = 120
	m.height = 40
	m.viewMode = ViewDetail
	m.selectedID = "test-1"

	view := m.renderDetailView()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Test Pipeline")
}

func TestRenderListView(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	pipelines := []registry.Pipeline{
		{ID: "test-1", Name: "Pipeline 1", Status: registry.StatusSatisfied},
		{ID: "test-2", Name: "Pipeline 2", Status: registry.StatusDrifted},
	}

	m := NewModel(pipelines, reg, cache, nil)
	m.width = 120
	m.height = 40
	m.viewMode = ViewList

	view := m.renderListView()
	assert.NotEmpty(t, view)
}

func TestRenderPipelineList(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	now := time.Now()
	pipelines := []registry.Pipeline{
		{ID: "test-1", Name: "Pipeline 1", Status: registry.StatusSatisfied, LastRun: now},
		{ID: "test-2", Name: "Pipeline 2", Status: registry.StatusDrifted, LastRun: now.Add(-1 * time.Hour)},
		{ID: "test-3", Name: "Pipeline 3", Status: registry.StatusFailed, LastRun: now.Add(-24 * time.Hour)},
	}

	m := NewModel(pipelines, reg, cache, nil)
	m.width = 120
	m.cursor = 1

	list := m.renderPipelineList()
	assert.NotEmpty(t, list)
}

func TestRenderPipelineItem(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	pipeline := registry.Pipeline{
		ID:      "test-1",
		Name:    "Test Pipeline",
		Status:  registry.StatusSatisfied,
		LastRun: time.Now(),
	}

	m := NewModel([]registry.Pipeline{pipeline}, reg, cache, nil)
	m.width = 120

	// Test selected item
	item := m.renderPipelineItem(0, true)
	assert.NotEmpty(t, item)

	// Test unselected item
	item = m.renderPipelineItem(0, false)
	assert.NotEmpty(t, item)

	// Test item with loading status
	m.loading["test-1"] = true
	item = m.renderPipelineItem(0, false)
	assert.NotEmpty(t, item)
}

func TestRenderHelpView(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	m := NewModel([]registry.Pipeline{}, reg, cache, nil)
	m.width = 120
	m.height = 40
	m.viewMode = ViewHelp

	view := m.renderHelpView()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Help")
}

func TestRenderConfirmView(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	m := NewModel([]registry.Pipeline{}, reg, cache, nil)
	m.width = 120
	m.height = 40
	m.viewMode = ViewConfirm
	m.confirmAction = "apply"

	view := m.renderConfirmView()
	assert.NotEmpty(t, view)
}

func TestRenderEmptyState(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	m := NewModel([]registry.Pipeline{}, reg, cache, nil)
	m.width = 120
	m.height = 40

	view := m.renderEmptyState()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "No pipelines")
}

func TestRenderHeader(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	pipelines := []registry.Pipeline{
		{ID: "test-1", Name: "Test", Status: registry.StatusSatisfied},
	}

	m := NewModel(pipelines, reg, cache, nil)
	m.width = 120

	header := m.renderHeader()
	assert.NotEmpty(t, header)
	assert.Contains(t, header, "Streamy")
}

func TestRenderFooter(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	m := NewModel([]registry.Pipeline{}, reg, cache, nil)
	m.width = 120
	m.viewMode = ViewList

	footer := m.renderFooter()
	assert.NotEmpty(t, footer)
}

func TestRenderErrorBanner(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	m := NewModel([]registry.Pipeline{}, reg, cache, nil)
	m.width = 120
	m.showError = true
	m.errorMsg = "Test error"

	banner := m.renderErrorBanner()
	assert.NotEmpty(t, banner)
	assert.Contains(t, banner, "Test error")
}
