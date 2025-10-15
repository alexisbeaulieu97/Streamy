package dashboard

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

func TestUpdate_WindowSizeMsg(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	m := NewModel([]registry.Pipeline{}, reg, cache, nil)

	// Test window resize
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	dashModel, ok := newModel.(Model)
	require.True(t, ok)

	assert.Equal(t, 100, dashModel.width)
	assert.Equal(t, 40, dashModel.height)
}

func TestUpdate_WindowSizeMsg_TooSmall(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	m := NewModel([]registry.Pipeline{}, reg, cache, nil)

	// Test small terminal size
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	dashModel, ok := newModel.(Model)
	require.True(t, ok)

	assert.True(t, dashModel.showError, "Should show error for small terminal")
	assert.Contains(t, dashModel.errorMsg, "Terminal too small")
}

func TestUpdate_SpinnerTickMsg(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	m := NewModel([]registry.Pipeline{}, reg, cache, nil)

	// Test spinner tick
	newModel, cmd := m.Update(spinner.TickMsg{})
	_, ok := newModel.(Model)
	require.True(t, ok)
	assert.NotNil(t, cmd)
}

func TestUpdate_InitialStatusLoadedMsg(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	pipelines := []registry.Pipeline{
		{ID: "test-1", Name: "Test 1", Status: registry.StatusUnknown},
	}
	m := NewModel(pipelines, reg, cache, nil)

	// Test initial status loaded
	msg := InitialStatusLoadedMsg{
		Statuses: map[string]registry.CachedStatus{
			"test-1": {Status: registry.StatusSatisfied, LastRun: time.Now()},
		},
	}

	newModel, _ := m.Update(msg)
	dashModel, ok := newModel.(Model)
	require.True(t, ok)

	assert.Equal(t, registry.StatusSatisfied, dashModel.pipelines[0].Status)
}

func TestUpdate_VerifyCompleteMsg(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	pipelines := []registry.Pipeline{
		{ID: "test-1", Name: "Test 1", Status: registry.StatusUnknown},
	}
	m := NewModel(pipelines, reg, cache, nil)
	m.loading["test-1"] = true

	// Test verify complete
	msg := VerifyCompleteMsg{
		PipelineID: "test-1",
		Result:     &registry.ExecutionResult{Status: registry.StatusSatisfied},
	}

	newModel, _ := m.Update(msg)
	dashModel, ok := newModel.(Model)
	require.True(t, ok)

	assert.Equal(t, registry.StatusSatisfied, dashModel.pipelines[0].Status)
	assert.False(t, dashModel.loading["test-1"])
}

func TestUpdate_VerifyErrorMsg(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	pipelines := []registry.Pipeline{
		{ID: "test-1", Name: "Test 1", Status: registry.StatusUnknown},
	}
	m := NewModel(pipelines, reg, cache, nil)
	m.loading["test-1"] = true

	// Test verify error
	msg := VerifyErrorMsg{
		PipelineID: "test-1",
		Error:      assert.AnError,
	}

	newModel, _ := m.Update(msg)
	dashModel, ok := newModel.(Model)
	require.True(t, ok)

	assert.Equal(t, registry.StatusFailed, dashModel.pipelines[0].Status)
	assert.False(t, dashModel.loading["test-1"])
	assert.True(t, dashModel.showError)
}

func TestUpdate_ApplyCompleteMsg(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	pipelines := []registry.Pipeline{
		{ID: "test-1", Name: "Test 1", Path: "/tmp/test.yaml", Status: registry.StatusDrifted},
	}
	m := NewModel(pipelines, reg, cache, nil)
	m.loading["test-1"] = true

	// Test apply complete
	msg := ApplyCompleteMsg{
		PipelineID: "test-1",
		Result:     &registry.ExecutionResult{Status: registry.StatusSatisfied},
	}

	newModel, _ := m.Update(msg)
	dashModel, ok := newModel.(Model)
	require.True(t, ok)

	assert.Equal(t, registry.StatusSatisfied, dashModel.pipelines[0].Status)
	// Should trigger auto-verify, so loading should be true again
	assert.True(t, dashModel.loading["test-1"])
}

func TestUpdate_ApplyErrorMsg(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	pipelines := []registry.Pipeline{
		{ID: "test-1", Name: "Test 1", Status: registry.StatusDrifted},
	}
	m := NewModel(pipelines, reg, cache, nil)
	m.loading["test-1"] = true

	// Test apply error
	msg := ApplyErrorMsg{
		PipelineID: "test-1",
		Error:      assert.AnError,
	}

	newModel, _ := m.Update(msg)
	dashModel, ok := newModel.(Model)
	require.True(t, ok)

	assert.Equal(t, registry.StatusFailed, dashModel.pipelines[0].Status)
	assert.False(t, dashModel.loading["test-1"])
	assert.True(t, dashModel.showError)
}

func TestUpdate_RefreshStartedMsg(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	m := NewModel([]registry.Pipeline{}, reg, cache, nil)

	// Test refresh started
	msg := RefreshStartedMsg{Total: 5}

	newModel, _ := m.Update(msg)
	dashModel, ok := newModel.(Model)
	require.True(t, ok)

	assert.True(t, dashModel.refreshing)
	assert.Equal(t, 0, dashModel.refreshProgress)
	assert.Equal(t, 5, dashModel.refreshTotal)
}

func TestUpdate_RefreshPipelineCompleteMsg(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	pipelines := []registry.Pipeline{
		{ID: "test-1", Name: "Test 1", Status: registry.StatusUnknown},
	}
	m := NewModel(pipelines, reg, cache, nil)
	m.refreshing = true
	m.refreshTotal = 1
	m.refreshProgress = 0

	// Test refresh pipeline complete
	msg := RefreshPipelineCompleteMsg{
		PipelineID: "test-1",
		Index:      0,
		Result:     &registry.ExecutionResult{Status: registry.StatusSatisfied},
	}

	newModel, _ := m.Update(msg)
	dashModel, ok := newModel.(Model)
	require.True(t, ok)

	assert.Equal(t, 1, dashModel.refreshProgress)
	assert.Equal(t, registry.StatusSatisfied, dashModel.pipelines[0].Status)
}

func TestUpdate_RefreshCompleteMsg(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	m := NewModel([]registry.Pipeline{}, reg, cache, nil)
	m.refreshing = true
	m.refreshProgress = 3
	m.refreshTotal = 3

	// Test refresh complete
	msg := RefreshCompleteMsg{}

	newModel, _ := m.Update(msg)
	dashModel, ok := newModel.(Model)
	require.True(t, ok)

	assert.False(t, dashModel.refreshing)
	assert.Equal(t, 0, dashModel.refreshProgress)
	assert.Equal(t, 0, dashModel.refreshTotal)
}

func TestUpdate_PipelineSelectedMsg(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	pipelines := []registry.Pipeline{
		{ID: "test-1", Name: "Test 1"},
	}
	m := NewModel(pipelines, reg, cache, nil)

	// Test pipeline selected
	msg := PipelineSelectedMsg{Pipeline: pipelines[0]}

	newModel, _ := m.Update(msg)
	dashModel, ok := newModel.(Model)
	require.True(t, ok)

	assert.Equal(t, ViewDetail, dashModel.viewMode)
	assert.Equal(t, "test-1", dashModel.selectedID)
}

func TestUpdate_BackToListMsg(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	m := NewModel([]registry.Pipeline{}, reg, cache, nil)
	m.viewMode = ViewDetail
	m.selectedID = "test-1"

	// Test back to list
	msg := BackToListMsg{}

	newModel, _ := m.Update(msg)
	dashModel, ok := newModel.(Model)
	require.True(t, ok)

	assert.Equal(t, ViewList, dashModel.viewMode)
	assert.Equal(t, "", dashModel.selectedID)
}

func TestUpdate_KeyMsg_ListNavigation(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	pipelines := []registry.Pipeline{
		{ID: "test-1", Name: "Test 1"},
		{ID: "test-2", Name: "Test 2"},
		{ID: "test-3", Name: "Test 3"},
	}
	m := NewModel(pipelines, reg, cache, nil)
	m.cursor = 0
	m.viewMode = ViewList

	// Test down key
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	dashModel, ok := newModel.(Model)
	require.True(t, ok)
	assert.Equal(t, 1, dashModel.cursor)

	// Test up key
	newModel, _ = dashModel.Update(tea.KeyMsg{Type: tea.KeyUp})
	dashModel, ok = newModel.(Model)
	require.True(t, ok)
	assert.Equal(t, 0, dashModel.cursor)

	// Test 'j' key (down)
	newModel, _ = dashModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	dashModel, ok = newModel.(Model)
	require.True(t, ok)
	assert.Equal(t, 1, dashModel.cursor)

	// Test 'k' key (up)
	newModel, _ = dashModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	dashModel, ok = newModel.(Model)
	require.True(t, ok)
	assert.Equal(t, 0, dashModel.cursor)
}

func TestUpdate_KeyMsg_DirectSelection(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	pipelines := []registry.Pipeline{
		{ID: "test-1", Name: "Test 1"},
		{ID: "test-2", Name: "Test 2"},
		{ID: "test-3", Name: "Test 3"},
	}
	m := NewModel(pipelines, reg, cache, nil)
	m.viewMode = ViewList

	// Test direct selection with '2'
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	dashModel, ok := newModel.(Model)
	require.True(t, ok)
	assert.Equal(t, 1, dashModel.cursor)

	// Test direct selection with '3'
	newModel, _ = dashModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	dashModel, ok = newModel.(Model)
	require.True(t, ok)
	assert.Equal(t, 2, dashModel.cursor)
}

func TestUpdate_KeyMsg_EnterSelectsPipeline(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	pipelines := []registry.Pipeline{
		{ID: "test-1", Name: "Test 1"},
		{ID: "test-2", Name: "Test 2"},
	}
	m := NewModel(pipelines, reg, cache, nil)
	m.viewMode = ViewList
	m.cursor = 1

	// Test enter key
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	dashModel, ok := newModel.(Model)
	require.True(t, ok)
	assert.Equal(t, ViewDetail, dashModel.viewMode)
	assert.Equal(t, "test-2", dashModel.selectedID)
}

func TestUpdate_KeyMsg_HelpToggle(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	m := NewModel([]registry.Pipeline{}, reg, cache, nil)
	m.viewMode = ViewList

	// Test '?' key to show help
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	dashModel, ok := newModel.(Model)
	require.True(t, ok)
	assert.Equal(t, ViewHelp, dashModel.viewMode)

	// Test '?' again to hide help
	newModel, _ = dashModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	dashModel, ok = newModel.(Model)
	require.True(t, ok)
	assert.Equal(t, ViewList, dashModel.viewMode)
}

func TestUpdate_KeyMsg_ErrorClear(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	m := NewModel([]registry.Pipeline{}, reg, cache, nil)
	m.viewMode = ViewList
	m.showError = true
	m.errorMsg = "Test error"

	// Test 'x' key to clear error
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	dashModel, ok := newModel.(Model)
	require.True(t, ok)
	assert.False(t, dashModel.showError)
	assert.Equal(t, "", dashModel.errorMsg)

	// Test 'esc' key to clear error
	m.showError = true
	m.errorMsg = "Test error"
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	dashModel, ok = newModel.(Model)
	require.True(t, ok)
	assert.False(t, dashModel.showError)
	assert.Equal(t, "", dashModel.errorMsg)
}

func TestUpdate_KeyMsg_DetailView_BackToList(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	m := NewModel([]registry.Pipeline{}, reg, cache, nil)
	m.viewMode = ViewDetail
	m.selectedID = "test-1"

	// Test 'esc' key to go back
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	dashModel, ok := newModel.(Model)
	require.True(t, ok)
	assert.Equal(t, ViewList, dashModel.viewMode)
	assert.Equal(t, "", dashModel.selectedID)

	// Test 'backspace' key to go back
	m.viewMode = ViewDetail
	m.selectedID = "test-1"
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	dashModel, ok = newModel.(Model)
	require.True(t, ok)
	assert.Equal(t, ViewList, dashModel.viewMode)
	assert.Equal(t, "", dashModel.selectedID)
}

func TestUpdate_KeyMsg_ConfirmDialog(t *testing.T) {
	tmpDir := t.TempDir()
	reg, err := registry.NewRegistry(filepath.Join(tmpDir, "registry.json"))
	require.NoError(t, err)

	cache, err := registry.NewStatusCache(filepath.Join(tmpDir, "cache.json"))
	require.NoError(t, err)

	m := NewModel([]registry.Pipeline{}, reg, cache, nil)
	m.viewMode = ViewConfirm
	m.confirmAction = "apply"
	m.confirmPipeline = "test-1"
	m.confirmMessage = "Apply configuration?"
	m.selectedID = "test-1"

	// Test 'n' key to cancel
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	dashModel, ok := newModel.(Model)
	require.True(t, ok)
	assert.Equal(t, ViewDetail, dashModel.viewMode)
	assert.Equal(t, "", dashModel.confirmAction)

	// Test 'esc' key to cancel
	m.viewMode = ViewConfirm
	m.confirmAction = "apply"
	m.selectedID = "test-1"
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	dashModel, ok = newModel.(Model)
	require.True(t, ok)
	assert.Equal(t, ViewDetail, dashModel.viewMode)
	assert.Equal(t, "", dashModel.confirmAction)
}
