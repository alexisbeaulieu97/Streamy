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

// VerificationStatus represents the state match level for a single step verification.
type VerificationStatus string

const (
	// StatusSatisfied indicates current state exactly matches expected configuration
	StatusSatisfied VerificationStatus = "satisfied"
	// StatusMissing indicates required resource/file/configuration not found
	StatusMissing VerificationStatus = "missing"
	// StatusDrifted indicates partial match or unexpected difference detected
	StatusDrifted VerificationStatus = "drifted"
	// StatusBlocked indicates cannot verify due to dependency failure or error
	StatusBlocked VerificationStatus = "blocked"
	// StatusUnknown indicates verification status cannot be determined
	StatusUnknown VerificationStatus = "unknown"
)

// IsValid checks if the verification status is one of the defined values.
func (s VerificationStatus) IsValid() bool {
	switch s {
	case StatusSatisfied, StatusMissing, StatusDrifted, StatusBlocked, StatusUnknown:
		return true
	default:
		return false
	}
}

// VerificationResult contains the outcome of verifying a single step.
type VerificationResult struct {
	StepID    string
	Status    VerificationStatus
	Message   string
	Details   string        // Unified diff for drifted status
	Error     error         // Populated for blocked status
	Duration  time.Duration
	Timestamp time.Time
}

// VerificationSummary aggregates verification results across all steps.
type VerificationSummary struct {
	TotalSteps int
	Satisfied  int
	Missing    int
	Drifted    int
	Blocked    int
	Unknown    int
	Results    []*VerificationResult
	Duration   time.Duration
}

// AllSatisfied returns true if all steps are satisfied.
func (s *VerificationSummary) AllSatisfied() bool {
	return s.Satisfied == s.TotalSteps
}

// NeedsApply returns true if any steps need to be applied (not satisfied).
func (s *VerificationSummary) NeedsApply() bool {
	return s.Missing > 0 || s.Drifted > 0 || s.Blocked > 0 || s.Unknown > 0
}

// ExitCode returns the appropriate exit code based on verification results.
// 0 = all satisfied, 1 = needs apply, 2 = configuration error, 3 = execution error
func (s *VerificationSummary) ExitCode() int {
	if s.AllSatisfied() {
		return 0
	}
	return 1
}
