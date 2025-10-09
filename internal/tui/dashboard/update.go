package dashboard

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/alexisbeaulieu97/streamy/internal/engine"
	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

// Update handles incoming messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// System messages
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		ApplyMaxWidth(m.width)

		// Check minimum terminal size
		const minWidth = 80
		const minHeight = 24
		if m.width < minWidth || m.height < minHeight {
			m.showError = true
			m.errorMsg = fmt.Sprintf("Terminal too small (%dx%d). Minimum size: %dx%d",
				m.width, m.height, minWidth, minHeight)
		} else if m.showError && m.errorMsg != "" &&
			strings.HasPrefix(m.errorMsg, "Terminal too small") {
			// Clear size error if terminal is now big enough
			m.showError = false
			m.errorMsg = ""
		}

		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	// Spinner tick for loading animations
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	// Status loading messages
	case InitialStatusLoadedMsg:
		for id, status := range msg.Statuses {
			m.UpdatePipelineStatus(id, status.Status, status.LastRun)
		}
		m.sortPipelines()
		return m, nil

	// Verify messages (US3)
	case VerifyStartedMsg:
		// Message sent inline by verifyCmd, not as separate message
		// Just update spinner
		return m, m.spinner.Tick

	case VerifyCompleteMsg:
		m.UpdatePipelineStatus(msg.PipelineID, msg.Result.Status, time.Now())
		delete(m.loading, msg.PipelineID)
		delete(m.operations, msg.PipelineID)
		delete(m.operationCtxs, msg.PipelineID)
		m.sortPipelines()

		// Save to cache
		return m, saveVerifyStatusToCacheCmd(m.statusCache, msg.PipelineID, msg.Result)

	case VerifyErrorMsg:
		m.UpdatePipelineStatus(msg.PipelineID, registry.StatusFailed, time.Now())
		delete(m.loading, msg.PipelineID)
		delete(m.operations, msg.PipelineID)
		delete(m.operationCtxs, msg.PipelineID)
		m.errors[msg.PipelineID] = msg.Error.Error()
		m.showError = true
		m.errorMsg = fmt.Sprintf("Verification failed: %s", msg.Error.Error())
		return m, nil

	case VerifyCancelledMsg:
		delete(m.loading, msg.PipelineID)
		delete(m.operations, msg.PipelineID)
		delete(m.operationCtxs, msg.PipelineID)
		return m, nil

	// Apply messages (US4 - placeholders)
	case ApplyStartedMsg:
		return m, m.spinner.Tick

	case ApplyCompleteMsg:
		m.UpdatePipelineStatus(msg.PipelineID, msg.Result.Status, time.Now())
		delete(m.loading, msg.PipelineID)
		delete(m.operations, msg.PipelineID)
		delete(m.operationCtxs, msg.PipelineID)
		m.sortPipelines()

		// Save status to cache and auto-verify after successful apply
		cmds := []tea.Cmd{
			saveApplyStatusToCacheCmd(m.statusCache, msg.PipelineID, msg.Result),
		}

		// Auto-verify to check if the system is now in desired state
		if pipeline, _, ok := m.GetPipelineByID(msg.PipelineID); ok {
			// Create new context for verification
			ctx, cancel := context.WithCancel(context.Background())
			m.operationCtxs[msg.PipelineID] = cancel
			m.loading[msg.PipelineID] = true
			m.operations[msg.PipelineID] = Operation{Type: "verifying", PipelineID: msg.PipelineID, StartedAt: time.Now()}
			cmds = append(cmds, verifyCmd(ctx, pipeline.ID, pipeline.Path, m.pluginReg))
		}

		return m, tea.Batch(cmds...)

	case ApplyErrorMsg:
		m.UpdatePipelineStatus(msg.PipelineID, registry.StatusFailed, time.Now())
		delete(m.loading, msg.PipelineID)
		delete(m.operations, msg.PipelineID)
		delete(m.operationCtxs, msg.PipelineID)
		m.errors[msg.PipelineID] = msg.Error.Error()
		m.showError = true
		m.errorMsg = fmt.Sprintf("Apply failed: %s", msg.Error.Error())
		return m, nil

	case ApplyCancelledMsg:
		delete(m.loading, msg.PipelineID)
		delete(m.operations, msg.PipelineID)
		delete(m.operationCtxs, msg.PipelineID)
		return m, nil

	// Refresh messages (US5 - placeholders)
	case RefreshStartedMsg:
		m.refreshing = true
		m.refreshProgress = 0
		m.refreshTotal = msg.Total
		return m, m.spinner.Tick

	case RefreshPipelineCompleteMsg:
		m.refreshProgress = msg.Index + 1
		if msg.Result != nil {
			m.UpdatePipelineStatus(msg.PipelineID, msg.Result.Status, time.Now())
			// Save to cache
			cached := registry.CachedStatus{
				Status:  msg.Result.Status,
				LastRun: time.Now(),
				Summary: "", // Could be populated from result if needed
			}
			if err := m.statusCache.Set(msg.PipelineID, cached); err != nil {
				// Log error but continue
				m.showError = true
				m.errorMsg = fmt.Sprintf("Failed to save cache: %s", err.Error())
			} else {
				// Save cache to disk
				if err := m.statusCache.Save(); err != nil {
					m.showError = true
					m.errorMsg = fmt.Sprintf("Failed to save cache: %s", err.Error())
				}
			}
		}
		// If all pipelines refreshed, trigger completion
		if m.refreshProgress >= m.refreshTotal {
			return m, func() tea.Msg {
				return RefreshCompleteMsg{}
			}
		}
		return m, nil

	case RefreshCompleteMsg:
		m.refreshing = false
		m.refreshProgress = 0
		m.refreshTotal = 0
		m.sortPipelines()
		return m, nil

	case RefreshCancelledMsg:
		m.refreshing = false
		m.refreshProgress = 0
		m.refreshTotal = 0
		return m, nil

	// Navigation messages (will be fully implemented in US2)
	case PipelineSelectedMsg:
		m.selectedID = msg.Pipeline.ID
		m.viewMode = ViewDetail
		return m, nil

	case BackToListMsg:
		m.viewMode = ViewList
		m.selectedID = ""
		return m, nil

	// Error messages
	case ErrorMsg:
		m.showError = true
		m.errorMsg = msg.Message
		return m, nil

	case ClearErrorMsg:
		m.showError = false
		m.errorMsg = ""
		return m, nil
	}

	return m, nil
}

// handleKeyPress handles keyboard input based on current view mode
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.viewMode {
	case ViewList:
		return m.handleListKeys(msg)
	case ViewDetail:
		return m.handleDetailKeys(msg)
	case ViewHelp:
		return m.handleHelpKeys(msg)
	case ViewConfirm:
		return m.handleConfirmKeys(msg)
	default:
		return m, nil
	}
}

// handleListKeys handles keys in list view
func (m Model) handleListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	// Clear error banner
	case "x":
		if m.showError {
			m.showError = false
			m.errorMsg = ""
			return m, nil
		}
		return m, nil

	// Quit
	case "q", "ctrl+c":
		return m, tea.Quit

	// Navigation
	case "up", "k":
		m.MoveCursorUp()
		return m, nil

	case "down", "j":
		m.MoveCursorDown()
		return m, nil

	// Direct selection with number keys
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		index := int(msg.String()[0] - '1')
		if index < len(m.pipelines) {
			m.SetCursor(index)
		}
		return m, nil

	// Select pipeline
	case "enter", " ":
		if selected, ok := m.GetSelectedPipeline(); ok {
			m.selectedID = selected.ID
			m.viewMode = ViewDetail
		}
		return m, nil

	// Refresh all pipelines (US5)
	case "r":
		if m.refreshing {
			// Already refreshing, ignore
			return m, nil
		}

		if len(m.pipelines) == 0 {
			return m, nil
		}

		m.refreshing = true
		m.refreshProgress = 0
		m.refreshTotal = len(m.pipelines)

		// Start parallel refresh
		var cmds []tea.Cmd
		cmds = append(cmds, m.spinner.Tick)

		// Launch verification for each pipeline
		for i, pipeline := range m.pipelines {
			ctx, cancel := context.WithCancel(context.Background())
			pipelineID := pipeline.ID
			m.operationCtxs[pipelineID] = cancel
			m.loading[pipelineID] = true

			cmds = append(cmds, refreshSingleCmd(ctx, pipeline, m.pluginReg, i, len(m.pipelines)))
		}

		return m, tea.Batch(cmds...)

	// Help
	case "?":
		m.viewMode = ViewHelp
		return m, nil

	// Clear error
	case "esc":
		if m.showError {
			m.showError = false
			m.errorMsg = ""
		}
		return m, nil
	}

	return m, nil
}

// handleDetailKeys handles keys in detail view
func (m Model) handleDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	// Clear error banner
	case "x":
		if m.showError {
			m.showError = false
			m.errorMsg = ""
			return m, nil
		}
		return m, nil

	// Quit application
	case "q", "ctrl+c":
		return m, tea.Quit

	// Back to list (or cancel operation with confirmation)
	case "esc", "backspace":
		// If operation in progress, ask for confirmation
		if m.loading[m.selectedID] {
			op, ok := m.operations[m.selectedID]
			if ok {
				m.confirmAction = fmt.Sprintf("cancel_%s", op.Type)
				m.confirmPipeline = m.selectedID
				m.confirmMessage = fmt.Sprintf("Cancel %s operation?", op.Type)
				m.viewMode = ViewConfirm
				return m, nil
			}
		}
		// Otherwise go back to list
		m.viewMode = ViewList
		m.selectedID = ""
		return m, nil

	// Verify pipeline (US3)
	case "v":
		// Get the selected pipeline
		var selected *registry.Pipeline
		for i := range m.pipelines {
			if m.pipelines[i].ID == m.selectedID {
				selected = &m.pipelines[i]
				break
			}
		}

		if selected == nil {
			return m, nil
		}

		// Create context for this operation
		ctx, cancel := context.WithCancel(context.Background())
		m.operationCtxs[selected.ID] = cancel
		m.loading[selected.ID] = true
		m.operations[selected.ID] = Operation{
			Type:       "verify",
			PipelineID: selected.ID,
			StartedAt:  time.Now(),
		}

		return m, verifyCmd(ctx, selected.ID, selected.Path, m.pluginReg)

	// Apply pipeline (US4 - with confirmation)
	case "a":
		// Get the selected pipeline
		var selected *registry.Pipeline
		for i := range m.pipelines {
			if m.pipelines[i].ID == m.selectedID {
				selected = &m.pipelines[i]
				break
			}
		}

		if selected == nil {
			return m, nil
		}

		// Show confirmation dialog
		m.confirmAction = "apply"
		m.confirmPipeline = selected.ID
		m.confirmMessage = fmt.Sprintf("Apply configuration for '%s'?", selected.Name)
		m.viewMode = ViewConfirm
		return m, nil

	// Refresh status (US5 - single pipeline)
	case "r":
		// Get the selected pipeline
		var selected *registry.Pipeline
		for i := range m.pipelines {
			if m.pipelines[i].ID == m.selectedID {
				selected = &m.pipelines[i]
				break
			}
		}

		if selected == nil {
			return m, nil
		}

		// Create context for this operation
		ctx, cancel := context.WithCancel(context.Background())
		m.operationCtxs[selected.ID] = cancel
		m.loading[selected.ID] = true
		m.operations[selected.ID] = Operation{
			Type:       "verify",
			PipelineID: selected.ID,
			StartedAt:  time.Now(),
		}

		return m, verifyCmd(ctx, selected.ID, selected.Path, m.pluginReg)

	// Help
	case "?":
		m.viewMode = ViewHelp
		return m, nil
	}
	return m, nil
}

// handleHelpKeys handles keys in help view
func (m Model) handleHelpKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "?", "esc", "q":
		// Return to previous view
		if m.selectedID != "" {
			m.viewMode = ViewDetail
		} else {
			m.viewMode = ViewList
		}
		return m, nil
	}
	return m, nil
}

// handleConfirmKeys handles keys in confirmation dialog
func (m Model) handleConfirmKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		// User confirmed action
		action := m.confirmAction
		pipelineID := m.confirmPipeline

		// Clear confirmation state
		m.confirmAction = ""
		m.confirmPipeline = ""
		m.confirmMessage = ""

		// Handle the confirmed action
		switch action {
		case "apply":
			// Find pipeline
			var selected *registry.Pipeline
			for i := range m.pipelines {
				if m.pipelines[i].ID == pipelineID {
					selected = &m.pipelines[i]
					break
				}
			}

			if selected == nil {
				m.viewMode = ViewList
				return m, nil
			}

			// Create context and start apply
			ctx, cancel := context.WithCancel(context.Background())
			m.operationCtxs[selected.ID] = cancel
			m.loading[selected.ID] = true
			m.operations[selected.ID] = Operation{
				Type:       "apply",
				PipelineID: selected.ID,
				StartedAt:  time.Now(),
			}

			// Return to detail view and start apply
			m.viewMode = ViewDetail
			return m, applyCmd(ctx, selected.ID, selected.Path, m.pluginReg)

		case "cancel_verify", "cancel_apply":
			// Cancel the operation
			if cancel, ok := m.operationCtxs[pipelineID]; ok {
				cancel()
				delete(m.operationCtxs, pipelineID)
			}
			delete(m.loading, pipelineID)
			delete(m.operations, pipelineID)

			m.viewMode = ViewDetail
			return m, nil

		default:
			m.viewMode = ViewDetail
			return m, nil
		}

	case "n", "N", "esc":
		// User cancelled, go back to detail view
		m.confirmAction = ""
		m.confirmPipeline = ""
		m.confirmMessage = ""

		if m.selectedID != "" {
			m.viewMode = ViewDetail
		} else {
			m.viewMode = ViewList
		}
		return m, nil
	}
	return m, nil
}

// saveVerifyStatusToCacheCmd saves verification result to cache
func saveVerifyStatusToCacheCmd(cache *registry.StatusCache, pipelineID string, result *engine.VerifyPipelineResult) tea.Cmd {
	return func() tea.Msg {
		cached := registry.CachedStatus{
			Status:      result.Status,
			LastRun:     time.Now(),
			Summary:     result.Summary,
			StepCount:   result.StepCount,
			FailedSteps: result.FailedSteps,
		}
		if err := cache.Set(pipelineID, cached); err != nil {
			return ErrorMsg{Message: fmt.Sprintf("Failed to update status cache: %v", err)}
		}
		if err := cache.Save(); err != nil {
			return ErrorMsg{Message: fmt.Sprintf("Failed to persist status cache: %v", err)}
		}
		return StatusCacheSavedMsg{PipelineID: pipelineID}
	}
}

// saveApplyStatusToCacheCmd saves apply result to cache
func saveApplyStatusToCacheCmd(cache *registry.StatusCache, pipelineID string, result *engine.ApplyPipelineResult) tea.Cmd {
	return func() tea.Msg {
		cached := registry.CachedStatus{
			Status:      result.Status,
			LastRun:     time.Now(),
			Summary:     result.Summary,
			StepCount:   result.StepCount,
			FailedSteps: result.FailedSteps,
		}
		if err := cache.Set(pipelineID, cached); err != nil {
			return ErrorMsg{Message: fmt.Sprintf("Failed to update status cache: %v", err)}
		}
		if err := cache.Save(); err != nil {
			return ErrorMsg{Message: fmt.Sprintf("Failed to persist status cache: %v", err)}
		}
		return StatusCacheSavedMsg{PipelineID: pipelineID}
	}
}
