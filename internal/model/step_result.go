package model

import (
	"time"
)

const (
	// StatusPending indicates a step has not started yet.
	StatusPending = "pending"
	// StatusRunning indicates a step is actively executing.
	StatusRunning = "running"
	// StatusSuccess marks a successful step execution.
	StatusSuccess = "success"
	// StatusSkipped indicates the executor skipped the step.
	StatusSkipped = "skipped"
	// StatusFailed marks a failure during step execution.
	StatusFailed = "failed"
	// StatusWouldCreate indicates dry-run would create a resource.
	StatusWouldCreate = "would_create"
	// StatusWouldUpdate indicates dry-run would update a resource.
	StatusWouldUpdate = "would_update"
)

// StepResult captures the outcome of executing a single step.
type StepResult struct {
	StepID    string
	Status    string
	Message   string
	Error     error
	Duration  time.Duration
	Timestamp time.Time
}
