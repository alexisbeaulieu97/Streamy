package dashboard

import (
	"time"

	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

// ViewMode determines which screen to render
type ViewMode int

const (
	ViewList ViewMode = iota
	ViewDetail
	ViewHelp
	ViewConfirm
)

// Navigation Messages

// PipelineSelectedMsg indicates a pipeline was selected
type PipelineSelectedMsg struct {
	Pipeline registry.Pipeline
}

// BackToListMsg requests return to list view
type BackToListMsg struct{}

// ScrollMsg indicates scroll action
type ScrollMsg struct {
	Direction int // +1 for down, -1 for up
}

// Operation Messages - Verify

// VerifyStartedMsg indicates verification has started
type VerifyStartedMsg struct {
	PipelineID string
	StartTime  time.Time
}

// VerifyCompleteMsg indicates verification completed successfully
type VerifyCompleteMsg struct {
	PipelineID string
	Result     *registry.ExecutionResult
}

// VerifyErrorMsg indicates verification failed
type VerifyErrorMsg struct {
	PipelineID string
	Error      error
}

// VerifyCancelledMsg indicates verification was cancelled
type VerifyCancelledMsg struct {
	PipelineID string
}

// Operation Messages - Apply

// ApplyStartedMsg indicates apply has started
type ApplyStartedMsg struct {
	PipelineID string
	StartTime  time.Time
}

// ApplyCompleteMsg indicates apply completed successfully
type ApplyCompleteMsg struct {
	PipelineID string
	Result     *registry.ExecutionResult
}

// ApplyErrorMsg indicates apply failed
type ApplyErrorMsg struct {
	PipelineID string
	Error      error
}

// ApplyCancelledMsg indicates apply was cancelled
type ApplyCancelledMsg struct {
	PipelineID string
}

// Operation Messages - Refresh

// RefreshStartedMsg indicates refresh of all pipelines started
type RefreshStartedMsg struct {
	Total int
}

// RefreshPipelineCompleteMsg indicates a single pipeline verification completed during refresh
type RefreshPipelineCompleteMsg struct {
	PipelineID string
	Index      int
	Total      int
	Result     *registry.ExecutionResult
	Error      error
}

// RefreshCompleteMsg indicates refresh completed
type RefreshCompleteMsg struct {
	Results map[string]*registry.ExecutionResult
}

// RefreshCancelledMsg indicates refresh was cancelled
type RefreshCancelledMsg struct{}

// Status Loading Messages

// InitialStatusLoadedMsg indicates cached statuses have been loaded
type InitialStatusLoadedMsg struct {
	Statuses map[string]registry.CachedStatus
}

// StatusCacheSavedMsg indicates status was saved to cache
type StatusCacheSavedMsg struct {
	PipelineID string
}

// Detail View Messages

// DetailLoadedMsg indicates pipeline details were loaded
type DetailLoadedMsg struct {
	PipelineID string
	Details    PipelineDetails
}

// PipelineDetails contains detailed information for display
type PipelineDetails struct {
	ConfigContent string
	StepCount     int
	LastResults   []registry.StepResult
}

// Confirmation Messages

// ConfirmActionMsg requests user confirmation
type ConfirmActionMsg struct {
	Action     string // "apply", "cancel_verify", "cancel_apply"
	PipelineID string
	Message    string
}

// ConfirmResponseMsg contains user's confirmation response
type ConfirmResponseMsg struct {
	Action     string
	PipelineID string
	Confirmed  bool
}

// Cancel Messages

// CancelOperationMsg requests cancellation of operation
type CancelOperationMsg struct {
	PipelineID string
	Operation  string // "verify" or "apply"
}

// OperationCancelledMsg indicates operation was cancelled
type OperationCancelledMsg struct {
	PipelineID string
	Operation  string
}

// Error Messages

// ErrorMsg indicates a general error occurred
type ErrorMsg struct {
	Message string
	Details *registry.ErrorDetail
}

// ClearErrorMsg requests error banner dismissal
type ClearErrorMsg struct{}

// Help Messages

// ToggleHelpMsg requests help overlay toggle
type ToggleHelpMsg struct{}
