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

	// Runtime state (not persisted in registry)
	Status     PipelineStatus   `json:"-"`
	LastRun    time.Time        `json:"-"`
	LastResult *ExecutionResult `json:"-"`
}

// PipelineStatus represents the verification state of a pipeline
type PipelineStatus string

const (
	StatusUnknown   PipelineStatus = "unknown"
	StatusSatisfied PipelineStatus = "satisfied"
	StatusDrifted   PipelineStatus = "drifted"
	StatusFailed    PipelineStatus = "failed"
	StatusVerifying PipelineStatus = "verifying"
	StatusApplying  PipelineStatus = "applying"
)

// Icon returns the Unicode icon for the status
func (s PipelineStatus) Icon() string {
	switch s {
	case StatusSatisfied:
		return "🟢"
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

// RegistryFile is the JSON file format for the pipeline registry
type RegistryFile struct {
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
}

// StatusCacheFile is the JSON file format for the status cache
type StatusCacheFile struct {
	Version  string                  `json:"version"`
	Statuses map[string]CachedStatus `json:"statuses"`
}
