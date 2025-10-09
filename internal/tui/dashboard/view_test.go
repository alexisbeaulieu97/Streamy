package dashboard

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

func TestFormatLastRun(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			name:     "zero time",
			time:     time.Time{},
			expected: "Never",
		},
		{
			name:     "just now",
			time:     now.Add(-30 * time.Second),
			expected: "Just now",
		},
		{
			name:     "1 minute ago",
			time:     now.Add(-1 * time.Minute),
			expected: "1 minute ago",
		},
		{
			name:     "5 minutes ago",
			time:     now.Add(-5 * time.Minute),
			expected: "5 minutes ago",
		},
		{
			name:     "1 hour ago",
			time:     now.Add(-1 * time.Hour),
			expected: "1 hour ago",
		},
		{
			name:     "3 hours ago",
			time:     now.Add(-3 * time.Hour),
			expected: "3 hours ago",
		},
		{
			name:     "1 day ago",
			time:     now.Add(-24 * time.Hour),
			expected: "1 day ago",
		},
		{
			name:     "3 days ago",
			time:     now.Add(-72 * time.Hour),
			expected: "3 days ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatLastRun(tt.time)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatLastRunOldDates(t *testing.T) {
	t.Helper()

	// For dates older than 7 days, FormatLastRun should fall back to a formatted calendar date.
	oldDate := time.Now().Add(-8 * 24 * time.Hour)

	result := FormatLastRun(oldDate)
	assert.Equal(t, oldDate.Format("Jan 2, 2006"), result)
}

func TestGetStatusStyle(t *testing.T) {
	tests := []struct {
		status string
		desc   string
	}{
		{"satisfied", "should return style for satisfied"},
		{"drifted", "should return style for drifted"},
		{"failed", "should return style for failed"},
		{"unknown", "should return style for unknown"},
		{"verifying", "should return style for verifying"},
		{"applying", "should return style for applying"},
		{"invalid", "should return style for unknown status"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			style := GetStatusStyle(tt.status)
			assert.NotNil(t, style, tt.desc)
		})
	}
}

func TestView(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	pipelines := []registry.Pipeline{
		{ID: "test-1", Name: "Test 1", Status: registry.StatusSatisfied},
		{ID: "test-2", Name: "Test 2", Status: registry.StatusDrifted},
	}

	// Test list view
	m := NewModel(pipelines, reg, cache, nil)
	m.width = 120
	m.height = 40
	m.viewMode = ViewList

	view := m.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Test 1")
	assert.Contains(t, view, "Test 2")

	// Test detail view
	m.viewMode = ViewDetail
	m.selectedID = "test-1"
	view = m.View()
	assert.NotEmpty(t, view)

	// Test help view
	m.viewMode = ViewHelp
	view = m.View()
	assert.NotEmpty(t, view)

	// Test confirm view
	m.viewMode = ViewConfirm
	m.confirmAction = "apply"
	view = m.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Apply Changes")
}

func TestView_WithError(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	m := NewModel([]registry.Pipeline{}, reg, cache, nil)
	m.width = 120
	m.height = 40
	m.viewMode = ViewList
	m.showError = true
	m.errorMsg = "Test error message"

	view := m.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Test error message")
}

func TestView_EmptyPipelines(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	m := NewModel([]registry.Pipeline{}, reg, cache, nil)
	m.width = 120
	m.height = 40
	m.viewMode = ViewList

	view := m.View()
	assert.NotEmpty(t, view)
	// Should show empty state
}

func TestView_WithRefreshing(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	pipelines := []registry.Pipeline{
		{ID: "test-1", Name: "Test 1", Status: registry.StatusSatisfied},
	}

	m := NewModel(pipelines, reg, cache, nil)
	m.width = 120
	m.height = 40
	m.viewMode = ViewList
	m.refreshing = true
	m.refreshProgress = 5
	m.refreshTotal = 10

	view := m.View()
	assert.NotEmpty(t, view)
	// Should show refresh progress
}

func TestView_WithLoading(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	pipelines := []registry.Pipeline{
		{ID: "test-1", Name: "Test 1", Status: registry.StatusSatisfied},
	}

	m := NewModel(pipelines, reg, cache, nil)
	m.width = 120
	m.height = 40
	m.viewMode = ViewDetail
	m.selectedID = "test-1"
	m.loading["test-1"] = true

	view := m.View()
	assert.NotEmpty(t, view)
	// Should show loading indicator
}

func TestView_AllStatuses(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	pipelines := []registry.Pipeline{
		{ID: "test-1", Name: "Satisfied", Status: registry.StatusSatisfied},
		{ID: "test-2", Name: "Drifted", Status: registry.StatusDrifted},
		{ID: "test-3", Name: "Failed", Status: registry.StatusFailed},
		{ID: "test-4", Name: "Unknown", Status: registry.StatusUnknown},
	}

	m := NewModel(pipelines, reg, cache, nil)
	m.width = 120
	m.height = 40
	m.viewMode = ViewList

	view := m.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Satisfied")
	assert.Contains(t, view, "Drifted")
	assert.Contains(t, view, "Failed")
	assert.Contains(t, view, "Unknown")
}
