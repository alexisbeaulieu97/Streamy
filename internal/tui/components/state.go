package components

import "time"

// StepStatus represents the lifecycle state of a pipeline step for rendering.
type StepStatus string

const (
	StepStatusPending     StepStatus = "pending"
	StepStatusRunning     StepStatus = "running"
	StepStatusSuccess     StepStatus = "success"
	StepStatusSkipped     StepStatus = "skipped"
	StepStatusFailed      StepStatus = "failed"
	StepStatusWouldCreate StepStatus = "would_create"
	StepStatusWouldUpdate StepStatus = "would_update"
)

// StepState captures the information required to display a step entry.
type StepState struct {
	Status   StepStatus
	Message  string
	Error    error
	Duration time.Duration
	Changed  bool
}
