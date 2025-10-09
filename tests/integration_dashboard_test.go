package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/logger"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/registry"
	"github.com/alexisbeaulieu97/streamy/internal/tui/dashboard"
)

// setupTestPluginRegistry creates a minimal plugin registry for testing
func setupTestPluginRegistry(t *testing.T) *plugin.PluginRegistry {
	t.Helper()

	log, err := logger.New(logger.Options{Level: "error", HumanReadable: false})
	require.NoError(t, err)

	cfg := &plugin.RegistryConfig{}
	return plugin.NewPluginRegistry(cfg, log)
}

// setupTestDashboard creates a temporary registry and cache with pipelines
func setupTestDashboard(t *testing.T, pipelines []registry.Pipeline, statuses map[string]registry.PipelineStatus) (string, *registry.Registry, *registry.StatusCache, *plugin.PluginRegistry) {
	t.Helper()

	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")
	cachePath := filepath.Join(tmpDir, "status-cache.json")

	// Create registry
	reg, err := registry.NewRegistry(registryPath)
	require.NoError(t, err)
	for _, p := range pipelines {
		err := reg.Add(p)
		require.NoError(t, err)
	}
	err = reg.Save()
	require.NoError(t, err)

	// Create cache with statuses
	cache, err := registry.NewStatusCache(cachePath)
	require.NoError(t, err)
	for id, status := range statuses {
		cache.Set(id, registry.CachedStatus{
			Status:  status,
			LastRun: time.Now(),
			Summary: "",
		})
	}
	err = cache.Save()
	require.NoError(t, err)

	// Create plugin registry
	pluginReg := setupTestPluginRegistry(t)

	return tmpDir, reg, cache, pluginReg
}

// runModelUpdate runs the model's Update function with a message
func runModelUpdate(t *testing.T, model dashboard.Model, msg tea.Msg) dashboard.Model {
	t.Helper()
	newModel, cmd := model.Update(msg)

	// Type assert back to dashboard.Model
	dashModel, ok := newModel.(dashboard.Model)
	require.True(t, ok, "Update should return dashboard.Model")

	// If there's a command, execute it and apply the result
	if cmd != nil {
		result := cmd()
		if result != nil {
			nextModel, _ := dashModel.Update(result)
			dashModel, ok = nextModel.(dashboard.Model)
			require.True(t, ok, "Update should return dashboard.Model")
		}
	}

	return dashModel
}

func TestDashboardDisplaysPipelines(t *testing.T) {
	// Setup: Create 3 pipelines with different statuses
	pipelines := []registry.Pipeline{
		{
			ID:           "dev-env",
			Name:         "Development Environment",
			Path:         "/tmp/dev-env.yaml",
			Description:  "Development environment setup",
			RegisteredAt: time.Now().Add(-24 * time.Hour),
		},
		{
			ID:           "web-server",
			Name:         "Web Server",
			Path:         "/tmp/web-server.yaml",
			Description:  "Web server configuration",
			RegisteredAt: time.Now().Add(-12 * time.Hour),
		},
		{
			ID:           "database",
			Name:         "Database",
			Path:         "/tmp/database.yaml",
			Description:  "Database setup",
			RegisteredAt: time.Now().Add(-6 * time.Hour),
		},
	}

	statuses := map[string]registry.PipelineStatus{
		"dev-env":    registry.StatusSatisfied,
		"web-server": registry.StatusDrifted,
		"database":   registry.StatusFailed,
	}

	_, reg, cache, pluginReg := setupTestDashboard(t, pipelines, statuses)

	// Load pipelines from registry
	loadedPipelines := reg.List()

	// Create dashboard model
	model := dashboard.NewModel(loadedPipelines, reg, cache, pluginReg)
	model = runModelUpdate(t, model, tea.WindowSizeMsg{Width: 80, Height: 40})

	// Get the rendered view
	view := model.View()

	// Assert: All 3 pipelines should appear in the view
	assert.Contains(t, view, "Development Environment", "Should display dev-env pipeline")
	assert.Contains(t, view, "Web Server", "Should display web-server pipeline")
	assert.Contains(t, view, "Database", "Should display database pipeline")

	// Assert: Status indicators should be present
	assert.Contains(t, view, "🟢", "Should display satisfied icon")
	assert.Contains(t, view, "🟡", "Should display drifted icon")
	assert.Contains(t, view, "🔴", "Should display failed icon")

	// Assert: Descriptions should be visible
	assert.Contains(t, view, "Development environment setup")
	assert.Contains(t, view, "Web server configuration")
	assert.Contains(t, view, "Database setup")

	// Assert: Status summary in header should show counts with icons
	assert.Contains(t, view, "🟢 1", "Should show 1 satisfied pipeline")
	assert.Contains(t, view, "🟡 1", "Should show 1 drifted pipeline")
	assert.Contains(t, view, "🔴 1", "Should show 1 failed pipeline")
}

func TestDashboardEmptyState(t *testing.T) {
	// Setup: Create empty registry and cache
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")
	cachePath := filepath.Join(tmpDir, "status-cache.json")

	reg, err := registry.NewRegistry(registryPath)
	require.NoError(t, err)
	err = reg.Save()
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(cachePath)
	require.NoError(t, err)
	err = cache.Save()
	require.NoError(t, err)

	// Plugin registry
	pluginReg := setupTestPluginRegistry(t)

	// Load pipelines (should be empty)
	loadedPipelines := reg.List()

	// Create dashboard model
	model := dashboard.NewModel(loadedPipelines, reg, cache, pluginReg)
	model = runModelUpdate(t, model, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Get the rendered view
	view := model.View()

	// Assert: Empty state message should appear
	assert.Contains(t, view, "No pipelines registered", "Should display empty state message")
	assert.Contains(t, view, "streamy register", "Should suggest register command")

	// Assert: Status summary should show zeros with icons
	assert.Contains(t, view, "🟢 0")
	assert.Contains(t, view, "🟡 0")
	assert.Contains(t, view, "🔴 0")
}

func TestDashboardSortsByPriority(t *testing.T) {
	// Setup: Create pipelines with mixed statuses
	pipelines := []registry.Pipeline{
		{
			ID:           "pipeline-a",
			Name:         "Pipeline A",
			Path:         "/tmp/a.yaml",
			Description:  "Pipeline A (satisfied)",
			RegisteredAt: time.Now().Add(-24 * time.Hour),
		},
		{
			ID:           "pipeline-b",
			Name:         "Pipeline B",
			Path:         "/tmp/b.yaml",
			Description:  "Pipeline B (failed)",
			RegisteredAt: time.Now().Add(-12 * time.Hour),
		},
		{
			ID:           "pipeline-c",
			Name:         "Pipeline C",
			Path:         "/tmp/c.yaml",
			Description:  "Pipeline C (drifted)",
			RegisteredAt: time.Now().Add(-6 * time.Hour),
		},
		{
			ID:           "pipeline-d",
			Name:         "Pipeline D",
			Path:         "/tmp/d.yaml",
			Description:  "Pipeline D (unknown)",
			RegisteredAt: time.Now().Add(-3 * time.Hour),
		},
	}

	statuses := map[string]registry.PipelineStatus{
		"pipeline-a": registry.StatusSatisfied,
		"pipeline-b": registry.StatusFailed,
		"pipeline-c": registry.StatusDrifted,
		"pipeline-d": registry.StatusUnknown,
	}

	_, reg, cache, pluginReg := setupTestDashboard(t, pipelines, statuses)

	// Load pipelines from registry
	loadedPipelines := reg.List()

	// Create dashboard model
	model := dashboard.NewModel(loadedPipelines, reg, cache, pluginReg)
	model = runModelUpdate(t, model, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Get the rendered view
	view := model.View()

	// Find positions of pipeline names in the view
	lines := strings.Split(view, "\n")
	positions := make(map[string]int)
	for i, line := range lines {
		if strings.Contains(line, "Pipeline A") {
			positions["pipeline-a"] = i
		}
		if strings.Contains(line, "Pipeline B") {
			positions["pipeline-b"] = i
		}
		if strings.Contains(line, "Pipeline C") {
			positions["pipeline-c"] = i
		}
		if strings.Contains(line, "Pipeline D") {
			positions["pipeline-d"] = i
		}
	}

	// Assert: All pipelines found
	require.Equal(t, 4, len(positions), "All 4 pipelines should be in the view")

	// Assert: Sorting priority (failed > drifted > satisfied > unknown)
	assert.Less(t, positions["pipeline-b"], positions["pipeline-c"],
		"Failed (B) should appear before Drifted (C)")
	assert.Less(t, positions["pipeline-c"], positions["pipeline-a"],
		"Drifted (C) should appear before Satisfied (A)")
	assert.Less(t, positions["pipeline-a"], positions["pipeline-d"],
		"Satisfied (A) should appear before Unknown (D)")
}

func TestDashboardLoadsCachedStatuses(t *testing.T) {
	// Setup: Create pipeline with cached status
	pipeline := registry.Pipeline{
		ID:           "test-pipeline",
		Name:         "Test Pipeline",
		Path:         "/tmp/test.yaml",
		Description:  "Test pipeline",
		RegisteredAt: time.Now(),
	}

	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")
	cachePath := filepath.Join(tmpDir, "status-cache.json")

	// Create registry
	reg, err := registry.NewRegistry(registryPath)
	require.NoError(t, err)
	err = reg.Add(pipeline)
	require.NoError(t, err)
	err = reg.Save()
	require.NoError(t, err)

	// Create cache with status
	cache, err := registry.NewStatusCache(cachePath)
	require.NoError(t, err)
	lastRun := time.Now().Add(-30 * time.Minute)
	cache.Set("test-pipeline", registry.CachedStatus{
		Status:  registry.StatusSatisfied,
		LastRun: lastRun,
		Summary: "",
	})
	err = cache.Save()
	require.NoError(t, err)

	// Plugin registry
	pluginReg := setupTestPluginRegistry(t)

	// Load pipelines from registry
	loadedPipelines := reg.List()

	// Create dashboard model
	model := dashboard.NewModel(loadedPipelines, reg, cache, pluginReg)

	// Trigger Init() to load cached statuses
	cmd := model.Init()
	require.NotNil(t, cmd, "Init() should return a command to load statuses")

	// Execute the command to get the status message
	msg := cmd()
	model = runModelUpdate(t, model, msg)
	counts := model.CountByStatus()
	assert.Equal(t, 1, counts[registry.StatusSatisfied], "Model should report satisfied pipeline from cache")

	// Get updated pipelines from model
	updatedPipelines := reg.List()
	require.Len(t, updatedPipelines, 1)

	// The model should have loaded the cached status
	// Since we can't access model internals directly, we check the cache was loaded
	cached, ok := cache.Get("test-pipeline")
	require.True(t, ok, "Cache should have the pipeline status")
	assert.Equal(t, registry.StatusSatisfied, cached.Status,
		"Cached status should be satisfied")
}

func TestDashboardHandlesWindowResize(t *testing.T) {
	// Setup: Create simple registry
	pipeline := registry.Pipeline{
		ID:           "test-pipeline",
		Name:         "Test Pipeline",
		Path:         "/tmp/test.yaml",
		Description:  "Test pipeline with a longer description",
		RegisteredAt: time.Now(),
	}

	_, reg, cache, pluginReg := setupTestDashboard(t, []registry.Pipeline{pipeline}, map[string]registry.PipelineStatus{
		"test-pipeline": registry.StatusSatisfied,
	})

	// Load pipelines from registry
	loadedPipelines := reg.List()

	// Create dashboard model
	model := dashboard.NewModel(loadedPipelines, reg, cache, pluginReg)

	// Test with narrow width
	model = runModelUpdate(t, model, tea.WindowSizeMsg{Width: 40, Height: 10})
	viewNarrow := model.View()
	assert.NotEmpty(t, viewNarrow, "Should render with narrow width")

	// Test with wide width
	model = runModelUpdate(t, model, tea.WindowSizeMsg{Width: 120, Height: 30})
	viewWide := model.View()
	assert.NotEmpty(t, viewWide, "Should render with wide width")

	// Wide view should contain the pipeline name (narrow might truncate)
	assert.Contains(t, viewWide, "Test Pipeline")
}

func TestDashboardNavigationWithKeyboard(t *testing.T) {
	// Setup: Create 3 pipelines
	pipelines := []registry.Pipeline{
		{
			ID:           "pipeline-1",
			Name:         "Pipeline 1",
			Path:         "/tmp/1.yaml",
			Description:  "First pipeline",
			RegisteredAt: time.Now(),
		},
		{
			ID:           "pipeline-2",
			Name:         "Pipeline 2",
			Path:         "/tmp/2.yaml",
			Description:  "Second pipeline",
			RegisteredAt: time.Now(),
		},
		{
			ID:           "pipeline-3",
			Name:         "Pipeline 3",
			Path:         "/tmp/3.yaml",
			Description:  "Third pipeline",
			RegisteredAt: time.Now(),
		},
	}

	_, reg, cache, pluginReg := setupTestDashboard(t, pipelines, map[string]registry.PipelineStatus{
		"pipeline-1": registry.StatusSatisfied,
		"pipeline-2": registry.StatusSatisfied,
		"pipeline-3": registry.StatusSatisfied,
	})

	// Load pipelines from registry
	loadedPipelines := reg.List()

	// Create dashboard model
	model := dashboard.NewModel(loadedPipelines, reg, cache, pluginReg)
	model = runModelUpdate(t, model, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Get initial view (cursor at 0)
	view0 := model.View()
	assert.Contains(t, view0, "Pipeline 1", "Should start at first pipeline")

	// Move down with 'j'
	model = runModelUpdate(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	view1 := model.View()

	// Move down again with arrow key
	model = runModelUpdate(t, model, tea.KeyMsg{Type: tea.KeyDown})
	view2 := model.View()

	// Move back up with 'k'
	model = runModelUpdate(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	view3 := model.View()

	// All views should render successfully
	assert.NotEmpty(t, view1)
	assert.NotEmpty(t, view2)
	assert.NotEmpty(t, view3)
}

func TestDashboardJSONFiles(t *testing.T) {
	// Test that registry and cache files maintain valid JSON format
	pipeline := registry.Pipeline{
		ID:           "test-json",
		Name:         "Test JSON",
		Path:         "/tmp/test.yaml",
		Description:  "Testing JSON serialization",
		RegisteredAt: time.Now(),
	}

	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "registry.json")
	cachePath := filepath.Join(tmpDir, "status-cache.json")

	// Create and save registry
	reg, err := registry.NewRegistry(registryPath)
	require.NoError(t, err)
	err = reg.Add(pipeline)
	require.NoError(t, err)
	err = reg.Save()
	require.NoError(t, err)

	// Verify registry JSON is valid
	registryData, err := os.ReadFile(registryPath)
	require.NoError(t, err)
	var registryFile registry.RegistryFile
	err = json.Unmarshal(registryData, &registryFile)
	require.NoError(t, err, "Registry JSON should be valid")
	assert.Equal(t, "1.0", registryFile.Version)
	assert.Len(t, registryFile.Pipelines, 1)

	// Create and save cache
	cache, err := registry.NewStatusCache(cachePath)
	require.NoError(t, err)
	cache.Set("test-json", registry.CachedStatus{
		Status:  registry.StatusSatisfied,
		LastRun: time.Now(),
		Summary: "",
	})
	err = cache.Save()
	require.NoError(t, err)

	// Verify cache JSON is valid
	cacheData, err := os.ReadFile(cachePath)
	require.NoError(t, err)
	var cacheFile registry.StatusCacheFile
	err = json.Unmarshal(cacheData, &cacheFile)
	require.NoError(t, err, "Cache JSON should be valid")
	assert.Equal(t, "1.0", cacheFile.Version)
	assert.Len(t, cacheFile.Statuses, 1)
}

func TestDashboardFormatLastRun(t *testing.T) {
	// This test verifies the FormatLastRun function through the dashboard view
	lastRunTime := time.Now().Add(-2 * time.Hour)
	pipeline := registry.Pipeline{
		ID:           "test-time",
		Name:         "Test Time Formatting",
		Path:         "/tmp/test.yaml",
		Description:  "Test time display",
		RegisteredAt: time.Now(),
		LastRun:      lastRunTime,
	}

	_, reg, cache, pluginReg := setupTestDashboard(t, []registry.Pipeline{pipeline}, map[string]registry.PipelineStatus{
		"test-time": registry.StatusSatisfied,
	})

	// Load pipelines from registry
	loadedPipelines := reg.List()

	// Create dashboard model
	model := dashboard.NewModel(loadedPipelines, reg, cache, pluginReg)
	model = runModelUpdate(t, model, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Get the rendered view
	view := model.View()

	// Assert: Should contain "Last checked:" and relative time
	// Note: The pipeline's LastRun is set 2 hours ago, but the cache uses time.Now()
	// So we'll just check that "Last checked:" appears
	assert.Contains(t, view, "Last checked:",
		fmt.Sprintf("Should show last checked time. View:\n%s", view))
}

// User Story 2 Tests - Navigation and Selection

func TestDashboardNavigateToDetail(t *testing.T) {
	// Setup: Create a pipeline
	pipeline := registry.Pipeline{
		ID:           "test-pipeline",
		Name:         "Test Pipeline",
		Path:         "/tmp/test.yaml",
		Description:  "Test pipeline for detail view",
		RegisteredAt: time.Now(),
	}

	_, reg, cache, pluginReg := setupTestDashboard(t, []registry.Pipeline{pipeline}, map[string]registry.PipelineStatus{
		"test-pipeline": registry.StatusSatisfied,
	})

	// Load pipelines from registry
	loadedPipelines := reg.List()

	// Create dashboard model
	model := dashboard.NewModel(loadedPipelines, reg, cache, pluginReg)
	model = runModelUpdate(t, model, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Verify we start in list view
	viewList := model.View()
	assert.Contains(t, viewList, "Test Pipeline", "Should show pipeline in list")
	assert.Contains(t, viewList, "↑/↓: navigate", "Should show list view footer")

	// Press Enter to select the pipeline
	model = runModelUpdate(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	// Get detail view
	viewDetail := model.View()

	// Assert: Detail view should show pipeline details
	assert.Contains(t, viewDetail, "📋 Test Pipeline", "Should show pipeline name in header")
	assert.Contains(t, viewDetail, "test-pipeline", "Should show pipeline ID")
	assert.Contains(t, viewDetail, "/tmp/test.yaml", "Should show pipeline path")
	assert.Contains(t, viewDetail, "Test pipeline for detail view", "Should show description")
	assert.Contains(t, viewDetail, "v: verify", "Should show detail view actions")
	assert.Contains(t, viewDetail, "esc: back", "Should show back action")
}

func TestDashboardBackToList(t *testing.T) {
	// Setup: Create a pipeline
	pipeline := registry.Pipeline{
		ID:           "test-pipeline",
		Name:         "Test Pipeline",
		Path:         "/tmp/test.yaml",
		Description:  "Test pipeline",
		RegisteredAt: time.Now(),
	}

	_, reg, cache, pluginReg := setupTestDashboard(t, []registry.Pipeline{pipeline}, map[string]registry.PipelineStatus{
		"test-pipeline": registry.StatusSatisfied,
	})

	loadedPipelines := reg.List()
	model := dashboard.NewModel(loadedPipelines, reg, cache, pluginReg)
	model = runModelUpdate(t, model, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Navigate to detail view
	model = runModelUpdate(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	viewDetail := model.View()
	assert.Contains(t, viewDetail, "📋 Test Pipeline", "Should be in detail view")

	// Press Esc to go back
	model = runModelUpdate(t, model, tea.KeyMsg{Type: tea.KeyEsc})
	viewList := model.View()

	// Assert: Back in list view
	assert.Contains(t, viewList, "🚀 Streamy Dashboard", "Should be back in list view")
	assert.Contains(t, viewList, "Test Pipeline", "Should show pipeline in list")
	assert.Contains(t, viewList, "↑/↓: navigate", "Should show list view footer")
}

func TestDashboardDirectSelection(t *testing.T) {
	// Setup: Create 5 pipelines
	pipelines := []registry.Pipeline{
		{
			ID:           "pipeline-1",
			Name:         "Pipeline 1",
			Path:         "/tmp/1.yaml",
			Description:  "First pipeline",
			RegisteredAt: time.Now(),
		},
		{
			ID:           "pipeline-2",
			Name:         "Pipeline 2",
			Path:         "/tmp/2.yaml",
			Description:  "Second pipeline",
			RegisteredAt: time.Now(),
		},
		{
			ID:           "pipeline-3",
			Name:         "Pipeline 3",
			Path:         "/tmp/3.yaml",
			Description:  "Third pipeline",
			RegisteredAt: time.Now(),
		},
	}

	_, reg, cache, pluginReg := setupTestDashboard(t, pipelines, map[string]registry.PipelineStatus{
		"pipeline-1": registry.StatusSatisfied,
		"pipeline-2": registry.StatusSatisfied,
		"pipeline-3": registry.StatusSatisfied,
	})

	loadedPipelines := reg.List()
	model := dashboard.NewModel(loadedPipelines, reg, cache, pluginReg)
	model = runModelUpdate(t, model, tea.WindowSizeMsg{Width: 80, Height: 24})

	// Press '3' to jump to the third pipeline
	model = runModelUpdate(t, model, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})

	// Press Enter to select
	model = runModelUpdate(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	// Get detail view
	viewDetail := model.View()

	// Assert: Should show the third pipeline's details
	assert.Contains(t, viewDetail, "📋 Pipeline 3", "Should show third pipeline")
	assert.Contains(t, viewDetail, "pipeline-3", "Should show correct pipeline ID")
}

func TestDashboardDetailViewWithResult(t *testing.T) {
	// Setup: Create a pipeline with execution result
	lastRun := time.Now().Add(-1 * time.Hour)
	pipeline := registry.Pipeline{
		ID:           "test-pipeline",
		Name:         "Test Pipeline",
		Path:         "/tmp/test.yaml",
		Description:  "Test pipeline",
		RegisteredAt: time.Now(),
		LastRun:      lastRun,
		Status:       registry.StatusFailed,
		LastResult: &registry.ExecutionResult{
			PipelineID:  "test-pipeline",
			Operation:   "verify",
			Status:      registry.StatusFailed,
			Success:     false,
			CompletedAt: lastRun,
			Duration:    2 * time.Second,
			StepResults: []registry.StepResult{
				{
					StepID:   "step-1",
					Status:   "success",
					Duration: 500 * time.Millisecond,
				},
				{
					StepID:   "step-2",
					Status:   "failed",
					Message:  "Step failed",
					Duration: 1500 * time.Millisecond,
				},
			},
			Error: &registry.ErrorDetail{
				Message:    "Pipeline verification failed",
				Suggestion: "Check step-2 configuration",
			},
		},
	}

	_, reg, cache, pluginReg := setupTestDashboard(t, []registry.Pipeline{pipeline}, map[string]registry.PipelineStatus{
		"test-pipeline": registry.StatusFailed,
	})

	loadedPipelines := reg.List()
	model := dashboard.NewModel(loadedPipelines, reg, cache, pluginReg)
	model = runModelUpdate(t, model, tea.WindowSizeMsg{Width: 80, Height: 40}) // Taller to avoid truncation

	// Navigate to detail view
	model = runModelUpdate(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	viewDetail := model.View()

	// Assert: Should show execution results
	assert.Contains(t, viewDetail, "Last Execution", "Should have Last Execution section")
	assert.Regexp(t, "Operation:\\s+verify", viewDetail, "Should show operation type")
	assert.Regexp(t, "Steps:\\s+2 total", viewDetail, "Should show total steps")
	assert.Regexp(t, "Summary:\\s+1 success, 1 failed", viewDetail, "Should show step counts")
	assert.Contains(t, viewDetail, "Error", "Should show error section")
	assert.Contains(t, viewDetail, "Pipeline verification failed", "Should show error message")
	assert.Regexp(t, "Suggestion:\\s+Check step-2 configuration", viewDetail, "Should show suggestion")
}

// ============================================================================
// US3: Verification Tests
// ============================================================================

func TestDashboardVerifyTriggersOperation(t *testing.T) {
	// Setup: Create a pipeline with unknown status
	pipeline := registry.Pipeline{
		ID:           "test-pipeline",
		Name:         "Test Pipeline",
		Path:         "/tmp/test.yaml",
		Description:  "Test pipeline for verification",
		RegisteredAt: time.Now(),
		Status:       registry.StatusUnknown,
	}

	_, reg, cache, pluginReg := setupTestDashboard(t, []registry.Pipeline{pipeline}, map[string]registry.PipelineStatus{
		"test-pipeline": registry.StatusUnknown,
	})

	loadedPipelines := reg.List()
	model := dashboard.NewModel(loadedPipelines, reg, cache, pluginReg)
	model = runModelUpdate(t, model, tea.WindowSizeMsg{Width: 100, Height: 40})

	// Navigate to detail view
	model = runModelUpdate(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	// Verify model is in detail view
	assert.True(t, model.GetViewMode() == dashboard.ViewDetail, "Should be in detail view")

	// Press 'v' to trigger verify
	// Note: In real scenario, this would start async verification
	// For integration test, we just verify the key is recognized
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	dashModel, ok := newModel.(dashboard.Model)
	require.True(t, ok, "Update should return dashboard.Model")

	// Assert: Model should show loading state
	assert.True(t, dashModel.IsLoading("test-pipeline"), "Pipeline should be in loading state")
	assert.NotNil(t, cmd, "Should return a command for async verification")
}

func TestDashboardVerifyShowsLoadingIndicator(t *testing.T) {
	// Setup
	pipeline := registry.Pipeline{
		ID:           "test-pipeline",
		Name:         "Test Pipeline",
		Path:         "/tmp/test.yaml",
		Description:  "Test pipeline",
		RegisteredAt: time.Now(),
		Status:       registry.StatusUnknown,
	}

	_, reg, cache, pluginReg := setupTestDashboard(t, []registry.Pipeline{pipeline}, map[string]registry.PipelineStatus{
		"test-pipeline": registry.StatusUnknown,
	})

	loadedPipelines := reg.List()
	model := dashboard.NewModel(loadedPipelines, reg, cache, pluginReg)
	model = runModelUpdate(t, model, tea.WindowSizeMsg{Width: 100, Height: 40})

	// Navigate to detail view and trigger verify
	model = runModelUpdate(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	dashModel, ok := newModel.(dashboard.Model)
	require.True(t, ok)

	// Get the view - should show loading indicator
	view := dashModel.View()
	assert.Contains(t, view, "verify in progress", "Should show verify in progress message")
}

// ============================================================================
// US4: Apply Tests
// ============================================================================

func TestDashboardApplyRequiresConfirmation(t *testing.T) {
	// Setup: Create a drifted pipeline
	pipeline := registry.Pipeline{
		ID:           "test-pipeline",
		Name:         "Test Pipeline",
		Path:         "/tmp/test.yaml",
		Description:  "Drifted pipeline",
		RegisteredAt: time.Now(),
		Status:       registry.StatusDrifted,
	}

	_, reg, cache, pluginReg := setupTestDashboard(t, []registry.Pipeline{pipeline}, map[string]registry.PipelineStatus{
		"test-pipeline": registry.StatusDrifted,
	})

	loadedPipelines := reg.List()
	model := dashboard.NewModel(loadedPipelines, reg, cache, pluginReg)
	model = runModelUpdate(t, model, tea.WindowSizeMsg{Width: 100, Height: 40})

	// Navigate to detail view
	model = runModelUpdate(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	// Press 'a' to trigger apply - should show confirmation
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	dashModel, ok := newModel.(dashboard.Model)
	require.True(t, ok)

	// Assert: Should be in confirm view
	assert.True(t, dashModel.GetViewMode() == dashboard.ViewConfirm, "Should show confirmation dialog")
	view := dashModel.View()
	assert.Contains(t, view, "Apply Changes", "Should show apply confirmation title")
	assert.Contains(t, view, "This will modify your system configuration.", "Should show warning")
}

func TestDashboardApplyConfirmationAccept(t *testing.T) {
	// Setup
	pipeline := registry.Pipeline{
		ID:           "test-pipeline",
		Name:         "Test Pipeline",
		Path:         "/tmp/test.yaml",
		Description:  "Drifted pipeline",
		RegisteredAt: time.Now(),
		Status:       registry.StatusDrifted,
	}

	_, reg, cache, pluginReg := setupTestDashboard(t, []registry.Pipeline{pipeline}, map[string]registry.PipelineStatus{
		"test-pipeline": registry.StatusDrifted,
	})

	loadedPipelines := reg.List()
	model := dashboard.NewModel(loadedPipelines, reg, cache, pluginReg)
	model = runModelUpdate(t, model, tea.WindowSizeMsg{Width: 100, Height: 40})

	// Navigate to detail view and trigger apply
	model = runModelUpdate(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	dashModel, ok := newModel.(dashboard.Model)
	require.True(t, ok)

	// Accept confirmation with 'y'
	newModel2, cmd := dashModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	dashModel2, ok := newModel2.(dashboard.Model)
	require.True(t, ok)

	// Assert: Should return to detail view and start apply
	assert.True(t, dashModel2.GetViewMode() == dashboard.ViewDetail, "Should return to detail view")
	assert.True(t, dashModel2.IsLoading("test-pipeline"), "Pipeline should be in loading state")
	assert.NotNil(t, cmd, "Should return command for apply operation")
}

func TestDashboardApplyConfirmationReject(t *testing.T) {
	// Setup
	pipeline := registry.Pipeline{
		ID:           "test-pipeline",
		Name:         "Test Pipeline",
		Path:         "/tmp/test.yaml",
		Description:  "Drifted pipeline",
		RegisteredAt: time.Now(),
		Status:       registry.StatusDrifted,
	}

	_, reg, cache, pluginReg := setupTestDashboard(t, []registry.Pipeline{pipeline}, map[string]registry.PipelineStatus{
		"test-pipeline": registry.StatusDrifted,
	})

	loadedPipelines := reg.List()
	model := dashboard.NewModel(loadedPipelines, reg, cache, pluginReg)
	model = runModelUpdate(t, model, tea.WindowSizeMsg{Width: 100, Height: 40})

	// Navigate to detail view and trigger apply
	model = runModelUpdate(t, model, tea.KeyMsg{Type: tea.KeyEnter})
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	dashModel, ok := newModel.(dashboard.Model)
	require.True(t, ok)

	// Reject confirmation with 'n'
	newModel2, cmd := dashModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	dashModel2, ok := newModel2.(dashboard.Model)
	require.True(t, ok)

	// Assert: Should return to detail view without starting apply
	assert.True(t, dashModel2.GetViewMode() == dashboard.ViewDetail, "Should return to detail view")
	assert.False(t, dashModel2.IsLoading("test-pipeline"), "Pipeline should not be loading")
	assert.Nil(t, cmd, "Should not return any command")
}

// ============================================================================
// US5: Refresh Tests
// ============================================================================

func TestDashboardRefreshAllPipelines(t *testing.T) {
	// Setup: Create multiple pipelines with different statuses
	pipelines := []registry.Pipeline{
		{
			ID:           "pipeline-1",
			Name:         "Pipeline 1",
			Path:         "/tmp/p1.yaml",
			Description:  "First pipeline",
			RegisteredAt: time.Now(),
			Status:       registry.StatusSatisfied,
		},
		{
			ID:           "pipeline-2",
			Name:         "Pipeline 2",
			Path:         "/tmp/p2.yaml",
			Description:  "Second pipeline",
			RegisteredAt: time.Now(),
			Status:       registry.StatusDrifted,
		},
		{
			ID:           "pipeline-3",
			Name:         "Pipeline 3",
			Path:         "/tmp/p3.yaml",
			Description:  "Third pipeline",
			RegisteredAt: time.Now(),
			Status:       registry.StatusUnknown,
		},
	}

	statuses := map[string]registry.PipelineStatus{
		"pipeline-1": registry.StatusSatisfied,
		"pipeline-2": registry.StatusDrifted,
		"pipeline-3": registry.StatusUnknown,
	}

	_, reg, cache, pluginReg := setupTestDashboard(t, pipelines, statuses)

	loadedPipelines := reg.List()
	model := dashboard.NewModel(loadedPipelines, reg, cache, pluginReg)
	model = runModelUpdate(t, model, tea.WindowSizeMsg{Width: 100, Height: 40})

	// Press 'r' to refresh all
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	dashModel, ok := newModel.(dashboard.Model)
	require.True(t, ok)

	// Assert: Should show refreshing state
	assert.True(t, dashModel.IsRefreshing(), "Should be in refreshing state")
	assert.NotNil(t, cmd, "Should return command for refresh")
	assert.Equal(t, 3, dashModel.GetRefreshTotal(), "Should be refreshing 3 pipelines")
}

func TestDashboardRefreshShowsProgress(t *testing.T) {
	// Setup
	pipelines := []registry.Pipeline{
		{
			ID:           "pipeline-1",
			Name:         "Pipeline 1",
			Path:         "/tmp/p1.yaml",
			RegisteredAt: time.Now(),
			Status:       registry.StatusUnknown,
		},
		{
			ID:           "pipeline-2",
			Name:         "Pipeline 2",
			Path:         "/tmp/p2.yaml",
			RegisteredAt: time.Now(),
			Status:       registry.StatusUnknown,
		},
	}

	statuses := map[string]registry.PipelineStatus{
		"pipeline-1": registry.StatusUnknown,
		"pipeline-2": registry.StatusUnknown,
	}

	_, reg, cache, pluginReg := setupTestDashboard(t, pipelines, statuses)

	loadedPipelines := reg.List()
	model := dashboard.NewModel(loadedPipelines, reg, cache, pluginReg)
	model = runModelUpdate(t, model, tea.WindowSizeMsg{Width: 100, Height: 40})

	// Start refresh
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	dashModel, ok := newModel.(dashboard.Model)
	require.True(t, ok)

	// Execute RefreshStartedMsg
	if cmd != nil {
		msg := cmd()
		newModel, _ := dashModel.Update(msg)
		dashModel = newModel.(dashboard.Model)
	}

	// View should show progress
	view := dashModel.View()
	assert.Contains(t, view, "Refreshing", "Should show refreshing indicator")
	assert.Contains(t, view, "0/2", "Should show progress counter")
}

func TestDashboardRefreshSinglePipeline(t *testing.T) {
	// Setup
	pipeline := registry.Pipeline{
		ID:           "test-pipeline",
		Name:         "Test Pipeline",
		Path:         "/tmp/test.yaml",
		Description:  "Test pipeline",
		RegisteredAt: time.Now(),
		Status:       registry.StatusUnknown,
	}

	_, reg, cache, pluginReg := setupTestDashboard(t, []registry.Pipeline{pipeline}, map[string]registry.PipelineStatus{
		"test-pipeline": registry.StatusUnknown,
	})

	loadedPipelines := reg.List()
	model := dashboard.NewModel(loadedPipelines, reg, cache, pluginReg)
	model = runModelUpdate(t, model, tea.WindowSizeMsg{Width: 100, Height: 40})

	// Navigate to detail view
	model = runModelUpdate(t, model, tea.KeyMsg{Type: tea.KeyEnter})

	// Press 'r' to refresh single pipeline
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	dashModel, ok := newModel.(dashboard.Model)
	require.True(t, ok)

	// Assert: Should start verification for single pipeline
	assert.True(t, dashModel.IsLoading("test-pipeline"), "Pipeline should be loading")
	assert.NotNil(t, cmd, "Should return command for verification")
}
