package registry

import (
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Pipeline represents a registered Streamy pipeline
type Pipeline struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	Description  string    `json:"description"`
	RegisteredAt time.Time `json:"registered_at"`

	Dependencies []string `json:"dependencies,omitempty"`

	// Runtime state (not persisted in registry)
	Status     PipelineStatus   `json:"-"`
	BlockedBy  []string         `json:"-"`
	LastRun    time.Time        `json:"-"`
	LastResult *ExecutionResult `json:"-"`
}

// PipelineStatus represents the verification state of a pipeline
type PipelineStatus string

// Pipeline status constants used to record registry execution outcomes.
const (
	StatusUnknown   PipelineStatus = "unknown"
	StatusReady     PipelineStatus = "ready"
	StatusBlocked   PipelineStatus = "blocked"
	StatusSatisfied PipelineStatus = "satisfied"
	StatusDrifted   PipelineStatus = "drifted"
	StatusFailed    PipelineStatus = "failed"
	StatusVerifying PipelineStatus = "verifying"
	StatusApplying  PipelineStatus = "applying"
)

// Icon returns the Unicode icon for the status
func (s PipelineStatus) Icon() string {
	switch s {
	case StatusReady:
		return "🟢"
	case StatusBlocked:
		return "🟠"
	case StatusSatisfied:
		return "✅"
	case StatusDrifted:
		return "🟡"
	case StatusFailed:
		return "🔴"
	default:
		return "⚪"
	}
}

// IconFallback returns ASCII fallback when Unicode is not supported
func (s PipelineStatus) IconFallback() string {
	switch s {
	case StatusReady:
		return "[RD]"
	case StatusBlocked:
		return "[BL]"
	case StatusSatisfied:
		return "[OK]"
	case StatusDrifted:
		return "[!!]"
	case StatusFailed:
		return "[XX]"
	default:
		return "[??]"
	}
}

// Color returns the Lipgloss color for the status
func (s PipelineStatus) Color() lipgloss.Color {
	switch s {
	case StatusReady:
		return lipgloss.Color("34") // green
	case StatusBlocked:
		return lipgloss.Color("208") // orange
	case StatusSatisfied:
		return lipgloss.Color("42") // green
	case StatusDrifted:
		return lipgloss.Color("226") // yellow
	case StatusFailed:
		return lipgloss.Color("196") // red
	default:
		return lipgloss.Color("250") // light gray
	}
}

// String returns the string representation of the status
func (s PipelineStatus) String() string {
	return string(s)
}

// ExecutionResult captures the outcome of a verify or apply operation
type ExecutionResult struct {
	PipelineID  string         `json:"pipeline_id"`
	Operation   string         `json:"operation"` // "verify" or "apply"
	Status      PipelineStatus `json:"status"`
	Success     bool           `json:"success"`
	BlockedBy   string         `json:"blocked_by,omitempty"`
	Summary     string         `json:"summary"`
	StepCount   int            `json:"step_count"`
	FailedSteps []string       `json:"failed_steps,omitempty"`
	StepResults []StepResult   `json:"step_results"`
	Duration    time.Duration  `json:"duration"`
	CompletedAt time.Time      `json:"completed_at"`
	Error       *ErrorDetail   `json:"error,omitempty"`
}

// StepResult represents the outcome of a single step
type StepResult struct {
	StepID   string        `json:"step_id"`
	Status   string        `json:"status"` // "pending", "running", "success", "failed", "skipped"
	Message  string        `json:"message,omitempty"`
	Duration time.Duration `json:"duration"`
	Error    *ErrorDetail  `json:"error,omitempty"`
}

// ErrorDetail provides structured error information
type ErrorDetail struct {
	Code       string   `json:"code"`
	Message    string   `json:"message"`
	Context    string   `json:"context"`
	Suggestion string   `json:"suggestion"`
	Stacktrace []string `json:"stacktrace,omitempty"`
}

// File is the JSON file format for the pipeline registry.
type File struct {
	Version   string     `json:"version"`
	Pipelines []Pipeline `json:"pipelines"`
}

// CachedStatus stores status metadata for a pipeline
type CachedStatus struct {
	Status      PipelineStatus `json:"status"`
	LastRun     time.Time      `json:"last_run"`
	Summary     string         `json:"summary"`
	StepCount   int            `json:"step_count"`
	FailedSteps []string       `json:"failed_steps,omitempty"`
	BlockedBy   string         `json:"blocked_by,omitempty"`
}

// StatusCacheFile is the JSON file format for the status cache
type StatusCacheFile struct {
	Version        string                  `json:"version"`
	Statuses       map[string]CachedStatus `json:"statuses"`
	Orchestrations []OrchestrationSummary  `json:"orchestrations,omitempty"`
}

// OrchestrationSummary captures a condensed history of orchestration runs for status cache persistence.
type OrchestrationSummary struct {
	RootPipelineID string    `json:"root_pipeline_id"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	TotalPipelines int       `json:"total_pipelines"`
	Executed       int       `json:"executed"`
	Blocked        int       `json:"blocked"`
	Failed         int       `json:"failed"`
	PipelineOrder  []string  `json:"pipeline_order"`
}
