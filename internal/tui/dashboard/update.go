package dashboard

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

// StepProgressMsg is emitted when a pipeline step updates its status.
type StepProgressMsg struct {
	PipelineID string
	StepID     string
	Status     string
	Message    string
}

// StepProgressTimeoutMsg indicates the dashboard did not receive a progress update before the timeout elapsed.
type StepProgressTimeoutMsg struct {
	PipelineID string
}

// StepProgressChannelClosedMsg is emitted when the progress channel closes.
type StepProgressChannelClosedMsg struct{}

const (
	progressRetryBaseDelay = 250 * time.Millisecond
	maxProgressRetries     = 5
	progressGlobalKey      = "__global__"
	keyEsc                 = "esc"
)

type updateHandler func(Model, tea.Msg) (tea.Model, tea.Cmd)

var updateHandlers = map[reflect.Type]updateHandler{
	reflect.TypeOf(tea.WindowSizeMsg{}): func(m Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m.handleWindowSize(msg.(tea.WindowSizeMsg))
	},
	reflect.TypeOf(spinner.TickMsg{}): func(m Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m.handleSpinnerTick(msg.(spinner.TickMsg))
	},
	reflect.TypeOf(InitialStatusLoadedMsg{}): func(m Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m.handleInitialStatus(msg.(InitialStatusLoadedMsg))
	},
	reflect.TypeOf(VerifyStartedMsg{}): func(m Model, _ tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.spinner.Tick
	},
	reflect.TypeOf(VerifyCompleteMsg{}): func(m Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m.handleVerifyComplete(msg.(VerifyCompleteMsg))
	},
	reflect.TypeOf(VerifyErrorMsg{}): func(m Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m.handleVerifyError(msg.(VerifyErrorMsg))
	},
	reflect.TypeOf(VerifyCancelledMsg{}): func(m Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m.handleVerifyCancelled(msg.(VerifyCancelledMsg))
	},
	reflect.TypeOf(ApplyStartedMsg{}): func(m Model, _ tea.Msg) (tea.Model, tea.Cmd) {
		return m, m.spinner.Tick
	},
	reflect.TypeOf(ApplyCompleteMsg{}): func(m Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m.handleApplyComplete(msg.(ApplyCompleteMsg))
	},
	reflect.TypeOf(ApplyErrorMsg{}): func(m Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m.handleApplyError(msg.(ApplyErrorMsg))
	},
	reflect.TypeOf(ApplyCancelledMsg{}): func(m Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m.handleApplyCancelled(msg.(ApplyCancelledMsg))
	},
	reflect.TypeOf(RefreshStartedMsg{}): func(m Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m.handleRefreshStarted(msg.(RefreshStartedMsg))
	},
	reflect.TypeOf(RefreshPipelineCompleteMsg{}): func(m Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m.handleRefreshPipelineComplete(msg.(RefreshPipelineCompleteMsg))
	},
	reflect.TypeOf(RefreshCompleteMsg{}): func(m Model, _ tea.Msg) (tea.Model, tea.Cmd) {
		return m.handleRefreshComplete()
	},
	reflect.TypeOf(RefreshCancelledMsg{}): func(m Model, _ tea.Msg) (tea.Model, tea.Cmd) {
		return m.handleRefreshCancelled()
	},
	reflect.TypeOf(StepProgressMsg{}): func(m Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m.handleStepProgress(msg.(StepProgressMsg))
	},
	reflect.TypeOf(StepProgressTimeoutMsg{}): func(m Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m.handleStepProgressTimeout(msg.(StepProgressTimeoutMsg))
	},
	reflect.TypeOf(StepProgressChannelClosedMsg{}): func(m Model, _ tea.Msg) (tea.Model, tea.Cmd) {
		return m.handleProgressChannelClosed()
	},
	reflect.TypeOf(PipelineSelectedMsg{}): func(m Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m.handlePipelineSelected(msg.(PipelineSelectedMsg))
	},
	reflect.TypeOf(BackToListMsg{}): func(m Model, _ tea.Msg) (tea.Model, tea.Cmd) {
		return m.handleBackToList()
	},
	reflect.TypeOf(ErrorMsg{}): func(m Model, msg tea.Msg) (tea.Model, tea.Cmd) {
		return m.handleError(msg.(ErrorMsg))
	},
	reflect.TypeOf(ClearErrorMsg{}): func(m Model, _ tea.Msg) (tea.Model, tea.Cmd) {
		return m.handleClearError()
	},
}

// Update handles incoming messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg == nil {
		return m, nil
	}

	if handler, ok := updateHandlers[reflect.TypeOf(msg)]; ok {
		updated, cmd := handler(m, msg)
		return updated, cmd
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		return m.handleKeyPress(keyMsg)
	}

	return m, nil
}

func (m Model) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	ApplyMaxWidth(m.width)

	const (
		minWidth  = 80
		minHeight = 24
	)

	if m.width < minWidth || m.height < minHeight {
		m.showError = true
		m.errorMsg = fmt.Sprintf("Terminal too small (%dx%d). Minimum size: %dx%d",
			m.width, m.height, minWidth, minHeight)
	} else if m.showError && m.errorMsg != "" && strings.HasPrefix(m.errorMsg, "Terminal too small") {
		m.showError = false
		m.errorMsg = ""
	}

	return m, nil
}

func (m Model) handleSpinnerTick(msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	m.spinner, cmd = m.spinner.Update(msg)

	return m, cmd
}

func (m Model) handleInitialStatus(msg InitialStatusLoadedMsg) (tea.Model, tea.Cmd) {
	for id, status := range msg.Statuses {
		m.UpdatePipelineStatus(id, status.Status, status.LastRun)
	}

	m.sortPipelines()

	return m, nil
}

func (m Model) handleVerifyComplete(msg VerifyCompleteMsg) (tea.Model, tea.Cmd) {
	m.UpdatePipelineStatus(msg.PipelineID, msg.Result.Status, time.Now())
	delete(m.loading, msg.PipelineID)
	delete(m.operations, msg.PipelineID)
	delete(m.operationCtxs, msg.PipelineID)
	m.sortPipelines()

	return m, saveVerifyStatusToCacheCmd(m.statusCache, msg.PipelineID, msg.Result)
}

func (m Model) handleVerifyError(msg VerifyErrorMsg) (tea.Model, tea.Cmd) {
	m.UpdatePipelineStatus(msg.PipelineID, registry.StatusFailed, time.Now())
	delete(m.loading, msg.PipelineID)
	delete(m.operations, msg.PipelineID)
	delete(m.operationCtxs, msg.PipelineID)
	m.errors[msg.PipelineID] = msg.Error.Error()
	m.showError = true
	m.errorMsg = fmt.Sprintf("Verification failed: %s", msg.Error.Error())

	return m, nil
}

func (m Model) handleVerifyCancelled(msg VerifyCancelledMsg) (tea.Model, tea.Cmd) {
	delete(m.loading, msg.PipelineID)
	delete(m.operations, msg.PipelineID)
	delete(m.operationCtxs, msg.PipelineID)

	return m, nil
}

func (m Model) handleApplyComplete(msg ApplyCompleteMsg) (tea.Model, tea.Cmd) {
	m.UpdatePipelineStatus(msg.PipelineID, msg.Result.Status, time.Now())
	delete(m.loading, msg.PipelineID)
	delete(m.operations, msg.PipelineID)
	delete(m.operationCtxs, msg.PipelineID)
	m.sortPipelines()

	cmds := []tea.Cmd{
		saveApplyStatusToCacheCmd(m.statusCache, msg.PipelineID, msg.Result),
	}

	if pipeline, _, ok := m.GetPipelineByID(msg.PipelineID); ok {
		ctx, cancel := context.WithCancel(context.Background())
		m.operationCtxs[msg.PipelineID] = cancel
		m.loading[msg.PipelineID] = true
		m.operations[msg.PipelineID] = Operation{
			Type:       "verifying",
			PipelineID: msg.PipelineID,
			StartedAt:  time.Now(),
		}
		cmds = append(cmds, verifyCmd(ctx, pipeline.ID, pipeline.Path, m.service))
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleApplyError(msg ApplyErrorMsg) (tea.Model, tea.Cmd) {
	m.UpdatePipelineStatus(msg.PipelineID, registry.StatusFailed, time.Now())
	delete(m.loading, msg.PipelineID)
	delete(m.operations, msg.PipelineID)
	delete(m.operationCtxs, msg.PipelineID)
	m.errors[msg.PipelineID] = msg.Error.Error()
	m.showError = true
	m.errorMsg = fmt.Sprintf("Apply failed: %s", msg.Error.Error())

	return m, nil
}

func (m Model) handleApplyCancelled(msg ApplyCancelledMsg) (tea.Model, tea.Cmd) {
	delete(m.loading, msg.PipelineID)
	delete(m.operations, msg.PipelineID)
	delete(m.operationCtxs, msg.PipelineID)

	return m, nil
}

func (m Model) handleRefreshStarted(msg RefreshStartedMsg) (tea.Model, tea.Cmd) {
	m.refreshing = true
	m.refreshProgress = 0
	m.refreshTotal = msg.Total

	return m, m.spinner.Tick
}

func (m Model) handleRefreshPipelineComplete(msg RefreshPipelineCompleteMsg) (tea.Model, tea.Cmd) {
	m.refreshProgress = msg.Index + 1
	if msg.Result != nil {
		m.UpdatePipelineStatus(msg.PipelineID, msg.Result.Status, time.Now())

		cached := registry.CachedStatus{
			Status:  msg.Result.Status,
			LastRun: time.Now(),
			Summary: "",
		}
		if err := m.statusCache.Set(msg.PipelineID, cached); err != nil {
			m.showError = true
			m.errorMsg = fmt.Sprintf("Failed to save cache: %s", err.Error())
		} else if err := m.statusCache.Save(); err != nil {
			m.showError = true
			m.errorMsg = fmt.Sprintf("Failed to save cache: %s", err.Error())
		}
	}

	if m.refreshProgress >= m.refreshTotal {
		return m, func() tea.Msg { return RefreshCompleteMsg{} }
	}

	return m, nil
}

func (m Model) handleRefreshComplete() (tea.Model, tea.Cmd) {
	m.refreshing = false
	m.refreshProgress = 0
	m.refreshTotal = 0
	m.sortPipelines()

	return m, nil
}

func (m Model) handleRefreshCancelled() (tea.Model, tea.Cmd) {
	m.refreshing = false
	m.refreshProgress = 0
	m.refreshTotal = 0

	return m, nil
}

func (m Model) handleStepProgress(msg StepProgressMsg) (tea.Model, tea.Cmd) {
	m.stepProgress[msg.PipelineID] = StepProgress{
		StepID:   msg.StepID,
		Status:   msg.Status,
		Message:  msg.Message,
		Recorded: time.Now(),
	}
	if m.progressRetries != nil {
		delete(m.progressRetries, msg.PipelineID)
		delete(m.progressRetries, progressGlobalKey)
	}

	if next := m.nextProgressCmd(msg.PipelineID); next != nil {
		return m, next
	}

	return m, nil
}

func (m Model) handleStepProgressTimeout(msg StepProgressTimeoutMsg) (tea.Model, tea.Cmd) {
	retryKey := m.resolveProgressKey(msg.PipelineID)

	targetPipelineID := msg.PipelineID
	if targetPipelineID == "" && retryKey != progressGlobalKey {
		targetPipelineID = retryKey
	}

	if m.progressRetries == nil {
		m.progressRetries = make(map[string]int)
	}

	m.progressRetries[retryKey]++

	retries := m.progressRetries[retryKey]
	if retries >= maxProgressRetries {
		m.showError = true
		if targetPipelineID != "" && targetPipelineID != progressGlobalKey {
			m.errors[targetPipelineID] = "No progress updates received; stopping dashboard listener."
			m.errorMsg = fmt.Sprintf("Pipeline %s stopped emitting progress updates.", targetPipelineID)
		} else if _, exists := m.errors[progressGlobalKey]; !exists {
			m.errors[progressGlobalKey] = "No progress updates received; stopping dashboard listener."
			m.errorMsg = "Progress updates timed out."
		}

		delete(m.progressRetries, retryKey)

		return m, nil
	}

	delay := progressRetryBaseDelay << (retries - 1)

	maxDelay := progressRetryBaseDelay << (maxProgressRetries - 1)
	if delay > maxDelay {
		delay = maxDelay
	}

	return m, tea.Tick(delay, func(time.Time) tea.Msg {
		if next := m.nextProgressCmd(targetPipelineID); next != nil {
			return next()
		}

		return nil
	})
}

func (m Model) handleProgressChannelClosed() (tea.Model, tea.Cmd) {
	for key := range m.progressRetries {
		delete(m.progressRetries, key)
	}

	return m, nil
}

func (m Model) handlePipelineSelected(msg PipelineSelectedMsg) (tea.Model, tea.Cmd) {
	m.selectedID = msg.Pipeline.ID
	m.viewMode = ViewDetail

	return m, nil
}

func (m Model) handleBackToList() (tea.Model, tea.Cmd) {
	m.viewMode = ViewList
	m.selectedID = ""

	return m, nil
}

func (m Model) handleError(msg ErrorMsg) (tea.Model, tea.Cmd) {
	m.showError = true
	m.errorMsg = msg.Message

	return m, nil
}

func (m Model) handleClearError() (tea.Model, tea.Cmd) {
	m.showError = false
	m.errorMsg = ""

	return m, nil
}

func (m Model) resolveProgressKey(pipelineID string) string {
	if strings.TrimSpace(pipelineID) != "" {
		return pipelineID
	}

	if len(m.operations) == 1 {
		for id := range m.operations {
			if strings.TrimSpace(id) != "" {
				return id
			}
		}
	}

	if len(m.loading) == 1 {
		for id := range m.loading {
			if strings.TrimSpace(id) != "" {
				return id
			}
		}
	}

	return progressGlobalKey
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

type listKeyHandler func(Model) (Model, tea.Cmd)

var listKeyHandlers = map[string]listKeyHandler{
	"x":      handleListClearError,
	"q":      handleListQuit,
	"ctrl+c": handleListQuit,
	"up":     handleListMoveUp,
	"k":      handleListMoveUp,
	"down":   handleListMoveDown,
	"j":      handleListMoveDown,
	"enter":  handleListSelectPipeline,
	" ":      handleListSelectPipeline,
	"r":      handleListRefresh,
	"?":      handleListShowHelp,
	keyEsc:   handleListEscape,
}

// handleListKeys handles keys in list view
func (m Model) handleListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if handler, ok := listKeyHandlers[key]; ok {
		return handler(m)
	}

	if isDigitKey(key) {
		return handleListDigitSelection(m, key)
	}

	return m, nil
}

func handleListClearError(m Model) (Model, tea.Cmd) {
	if m.showError {
		m.showError = false
		m.errorMsg = ""
	}

	return m, nil
}

func handleListQuit(m Model) (Model, tea.Cmd) {
	return m, tea.Quit
}

func handleListMoveUp(m Model) (Model, tea.Cmd) {
	m.MoveCursorUp()
	return m, nil
}

func handleListMoveDown(m Model) (Model, tea.Cmd) {
	m.MoveCursorDown()
	return m, nil
}

func handleListDigitSelection(m Model, key string) (Model, tea.Cmd) {
	index := int(key[0] - '1')
	if index < len(m.pipelines) {
		m.SetCursor(index)
	}

	return m, nil
}

func handleListSelectPipeline(m Model) (Model, tea.Cmd) {
	if selected, ok := m.GetSelectedPipeline(); ok {
		m.selectedID = selected.ID
		m.viewMode = ViewDetail
	}

	return m, nil
}

func handleListRefresh(m Model) (Model, tea.Cmd) {
	if m.refreshing || len(m.pipelines) == 0 {
		return m, nil
	}

	m.refreshing = true
	m.refreshProgress = 0
	m.refreshTotal = len(m.pipelines)

	cmds := []tea.Cmd{m.spinner.Tick}

	for i, pipeline := range m.pipelines {
		ctx, cancel := context.WithCancel(context.Background())
		pipelineID := pipeline.ID
		m.operationCtxs[pipelineID] = cancel
		m.loading[pipelineID] = true

		cmds = append(cmds, refreshSingleCmd(ctx, pipeline, m.service, i, len(m.pipelines)))
	}

	return m, tea.Batch(cmds...)
}

func handleListShowHelp(m Model) (Model, tea.Cmd) {
	m.viewMode = ViewHelp
	return m, nil
}

func handleListEscape(m Model) (Model, tea.Cmd) {
	if m.showError {
		m.showError = false
		m.errorMsg = ""
	}

	return m, nil
}

func isDigitKey(key string) bool {
	return len(key) == 1 && key[0] >= '1' && key[0] <= '9'
}

// handleDetailKeys handles keys in detail view
func (m Model) handleDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if handler, ok := detailKeyHandlers[key]; ok {
		return handler(m)
	}

	return m, nil
}

type detailKeyHandler func(Model) (Model, tea.Cmd)

var detailKeyHandlers = map[string]detailKeyHandler{
	"x":         handleDetailClearError,
	"q":         handleDetailQuit,
	"ctrl+c":    handleDetailQuit,
	keyEsc:      handleDetailEscape,
	"backspace": handleDetailEscape,
	"v":         handleDetailVerify,
	"a":         handleDetailApply,
	"r":         handleDetailRefresh,
	"?":         handleDetailShowHelp,
}

func handleDetailClearError(m Model) (Model, tea.Cmd) {
	if m.showError {
		m.showError = false
		m.errorMsg = ""
	}

	return m, nil
}

func handleDetailQuit(m Model) (Model, tea.Cmd) {
	return m, tea.Quit
}

func handleDetailEscape(m Model) (Model, tea.Cmd) {
	if m.loading[m.selectedID] {
		if op, ok := m.operations[m.selectedID]; ok {
			m.confirmAction = fmt.Sprintf("cancel_%s", op.Type)
			m.confirmPipeline = m.selectedID
			m.confirmMessage = fmt.Sprintf("Cancel %s operation?", op.Type)
			m.viewMode = ViewConfirm

			return m, nil
		}
	}

	m.viewMode = ViewList
	m.selectedID = ""

	return m, nil
}

func handleDetailVerify(m Model) (Model, tea.Cmd) {
	selected, ok := m.GetSelectedPipeline()
	if !ok {
		return m, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.operationCtxs[selected.ID] = cancel
	m.loading[selected.ID] = true
	m.operations[selected.ID] = Operation{
		Type:       operationVerify,
		PipelineID: selected.ID,
		StartedAt:  time.Now(),
	}

	return m, verifyCmd(ctx, selected.ID, selected.Path, m.service)
}

func handleDetailApply(m Model) (Model, tea.Cmd) {
	selected, ok := m.GetSelectedPipeline()
	if !ok {
		return m, nil
	}

	m.confirmAction = actionApply
	m.confirmPipeline = selected.ID
	m.confirmMessage = fmt.Sprintf("Apply configuration for '%s'?", selected.Name)
	m.viewMode = ViewConfirm

	return m, nil
}

func handleDetailRefresh(m Model) (Model, tea.Cmd) {
	selected, ok := m.GetSelectedPipeline()
	if !ok {
		return m, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.operationCtxs[selected.ID] = cancel
	m.loading[selected.ID] = true
	m.operations[selected.ID] = Operation{
		Type:       operationVerify,
		PipelineID: selected.ID,
		StartedAt:  time.Now(),
	}

	return m, verifyCmd(ctx, selected.ID, selected.Path, m.service)
}

func handleDetailShowHelp(m Model) (Model, tea.Cmd) {
	m.viewMode = ViewHelp
	return m, nil
}

// handleHelpKeys handles keys in help view
func (m Model) handleHelpKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "?", keyEsc, "q":
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
		case actionApply:
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
				Type:       operationApply,
				PipelineID: selected.ID,
				StartedAt:  time.Now(),
			}

			// Return to detail view and start apply
			m.viewMode = ViewDetail

			return m, applyCmd(ctx, selected.ID, selected.Path, m.service)

		case actionCancelVerify, actionCancelApply:
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

	case "n", "N", keyEsc:
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
func saveVerifyStatusToCacheCmd(cache *registry.StatusCache, pipelineID string, result *registry.ExecutionResult) tea.Cmd {
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
func saveApplyStatusToCacheCmd(cache *registry.StatusCache, pipelineID string, result *registry.ExecutionResult) tea.Cmd {
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
