// Package pipelineconv converts domain pipeline data into CLI-friendly structures.
package pipelineconv

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/registry"
	"github.com/alexisbeaulieu97/streamy/internal/tui/components"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

const (
	stepStatusPending      = "pending"
	stepStatusRunning      = "running"
	stepStatusSuccess      = "success"
	stepStatusSkipped      = "skipped"
	stepStatusFailed       = "failed"
	stepStatusWouldCreate  = "would_create"
	stepStatusWouldUpdate  = "would_update"
	defaultVerificationMsg = "verification result unavailable"
)

// VerificationStatus mirrors the legacy verification status strings for CLI output.
type VerificationStatus string

const (
	// VerificationSatisfied indicates the step is up to date.
	VerificationSatisfied VerificationStatus = "satisfied"
	// VerificationMissing indicates the step has not been executed yet.
	VerificationMissing VerificationStatus = "missing"
	// VerificationDrifted indicates the step is out of sync with desired state.
	VerificationDrifted VerificationStatus = "drifted"
	// VerificationBlocked indicates verification cannot proceed due to a dependency.
	VerificationBlocked VerificationStatus = "blocked"
	// VerificationUnknown captures unexpected verification outcomes.
	VerificationUnknown VerificationStatus = "unknown"
)

// VerificationResult represents the outcome of verifying a single step for CLI output.
type VerificationResult struct {
	StepID    string
	Status    VerificationStatus
	Message   string
	Details   string
	Error     error
	Duration  time.Duration
	Timestamp time.Time
}

// VerificationSummary aggregates verification results for CLI and JSON output.
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

// AllSatisfied reports whether every step is satisfied.
func (s *VerificationSummary) AllSatisfied() bool {
	return s.TotalSteps > 0 && s.Satisfied == s.TotalSteps
}

// ExitCode returns an exit code compatible with the legacy CLI.
// 0 = all satisfied, 1 = needs apply.
func (s *VerificationSummary) ExitCode() int {
	if s.AllSatisfied() {
		return 0
	}

	return 1
}

// BuildVerificationSummary aggregates domain verification results into a CLI-friendly summary.
func BuildVerificationSummary(pipeline *domainpipeline.Pipeline, results []domainpipeline.VerificationResult) *VerificationSummary {
	summary := &VerificationSummary{
		TotalSteps: len(results),
		Results:    make([]*VerificationResult, 0, len(results)),
	}

	if pipeline != nil && len(pipeline.Steps) > summary.TotalSteps {
		summary.TotalSteps = len(pipeline.Steps)
	}

	for _, res := range results {
		status := mapVerificationStatus(res)

		message := res.Message
		if strings.TrimSpace(message) == "" {
			message = defaultVerificationMsg
		}

		details := formatVerificationDetails(res.Details)
		vr := &VerificationResult{
			StepID:    res.StepID,
			Status:    status,
			Message:   message,
			Details:   details,
			Timestamp: time.Now().UTC(),
		}

		switch status {
		case VerificationSatisfied:
			summary.Satisfied++
		case VerificationMissing:
			summary.Missing++
		case VerificationDrifted:
			summary.Drifted++
		case VerificationBlocked:
			summary.Blocked++
		case VerificationUnknown:
			summary.Unknown++
		}

		summary.Results = append(summary.Results, vr)
	}

	return summary
}

// SummaryToExecutionResult converts a verification summary into the registry execution result format.
func SummaryToExecutionResult(summary *VerificationSummary, configPath string) *registry.ExecutionResult {
	if summary == nil {
		return &registry.ExecutionResult{
			Operation:   "verify",
			Status:      registry.StatusFailed,
			Success:     false,
			Summary:     "verification unavailable",
			StepResults: []registry.StepResult{},
			Error: &registry.ErrorDetail{
				Message:    "Verification did not produce a summary",
				Context:    fmt.Sprintf("Config: %s", configPath),
				Suggestion: "Retry 'streamy verify' and check logs",
			},
			CompletedAt: time.Now().UTC(),
		}
	}

	execResult := &registry.ExecutionResult{
		Operation:   "verify",
		Status:      pipelineStatusFromSummary(summary),
		Success:     summary.AllSatisfied(),
		StepCount:   summary.TotalSteps,
		StepResults: make([]registry.StepResult, 0, len(summary.Results)),
		CompletedAt: time.Now().UTC(),
	}

	var failed []string
	for _, res := range summary.Results {
		stepRes := registry.StepResult{
			StepID:   res.StepID,
			Status:   string(res.Status),
			Message:  res.Message,
			Duration: res.Duration,
		}
		if res.Error != nil {
			stepRes.Error = &registry.ErrorDetail{
				Message: res.Error.Error(),
				Context: fmt.Sprintf("Config: %s, Step: %s", configPath, res.StepID),
			}
			failed = append(failed, res.StepID)
		}

		if res.Status == VerificationMissing || res.Status == VerificationDrifted || res.Status == VerificationBlocked || res.Status == VerificationUnknown {
			failed = append(failed, res.StepID)
		}

		execResult.StepResults = append(execResult.StepResults, stepRes)
	}

	execResult.FailedSteps = dedupeStrings(failed)
	if len(execResult.FailedSteps) > 0 {
		execResult.Success = false
		execResult.Status = registry.StatusDrifted

		execResult.Summary = fmt.Sprintf("%d steps need changes", len(execResult.FailedSteps))
		if execResult.Error == nil {
			execResult.Error = &registry.ErrorDetail{
				Message:    "Verification detected drift",
				Context:    fmt.Sprintf("Config: %s", configPath),
				Suggestion: "Run 'streamy apply' to reconcile changes",
			}
		}
	} else {
		execResult.Summary = fmt.Sprintf("All %d steps passed", summary.TotalSteps)
	}

	return execResult
}

func pipelineStatusFromSummary(summary *VerificationSummary) registry.PipelineStatus {
	if summary == nil {
		return registry.StatusFailed
	}

	if summary.AllSatisfied() {
		return registry.StatusSatisfied
	}

	if summary.Missing > 0 || summary.Drifted > 0 {
		return registry.StatusDrifted
	}

	return registry.StatusFailed
}

func mapVerificationStatus(res domainpipeline.VerificationResult) VerificationStatus {
	if status := legacyStateFromDetails(res.Details); status != "" {
		switch strings.ToLower(strings.TrimSpace(status)) {
		case "satisfied":
			return VerificationSatisfied
		case "missing", "not_satisfied", "not satisfied":
			return VerificationMissing
		case "drifted", "drift", "not_matching":
			return VerificationDrifted
		case "blocked":
			return VerificationBlocked
		case "unknown":
			return VerificationUnknown
		}
	}

	switch res.Status {
	case domainpipeline.VerificationSatisfied:
		return VerificationSatisfied
	case domainpipeline.VerificationFailed:
		return VerificationDrifted
	case domainpipeline.VerificationUnknown:
		return VerificationUnknown
	default:
		return VerificationUnknown
	}
}

func legacyStateFromDetails(details map[string]interface{}) string {
	if details == nil {
		return ""
	}

	if state, ok := details["current_state"].(string); ok {
		return state
	}

	if state, ok := details["status"].(string); ok {
		return state
	}

	return ""
}

func formatVerificationDetails(details map[string]interface{}) string {
	if len(details) == 0 {
		return ""
	}

	data, err := json.MarshalIndent(details, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", details)
	}

	return string(data)
}

// ConvertStepResult maps a domain step result into a registry step result.
func ConvertStepResult(res domainpipeline.StepResult, dryRun bool) registry.StepResult {
	status := mapStepStatus(res.Status)
	if dryRun && res.Changed {
		status = stepStatusWouldUpdate
		if res.Status == domainpipeline.StatusSkipped {
			status = stepStatusWouldCreate
		}
	}

	stepResult := registry.StepResult{
		StepID:   res.StepID,
		Status:   status,
		Message:  res.FormatOutput(),
		Duration: res.Duration,
	}

	if res.Error != nil {
		stepResult.Error = &registry.ErrorDetail{
			Code:       string(res.Error.Code),
			Message:    res.Error.Error(),
			Context:    fmt.Sprintf("%v", res.Error.Context),
			Suggestion: "",
		}
	}

	return stepResult
}

func mapStepStatus(status domainpipeline.ResultStatus) string {
	switch status {
	case domainpipeline.StatusSuccess, domainpipeline.StatusAlreadySatisfied:
		return stepStatusSuccess
	case domainpipeline.StatusFailure:
		return stepStatusFailed
	case domainpipeline.StatusSkipped:
		return stepStatusSkipped
	default:
		return stepStatusFailed
	}
}

// ConvertApplyResults builds a registry execution result from domain step results.
func ConvertApplyResults(results []domainpipeline.StepResult, configPath string, dryRun bool, execErr, validationErr error) *registry.ExecutionResult {
	execResult := &registry.ExecutionResult{
		Operation:   "apply",
		Status:      registry.StatusSatisfied,
		Success:     true,
		StepResults: make([]registry.StepResult, 0, len(results)),
		CompletedAt: time.Now().UTC(),
	}

	var (
		totalDuration time.Duration
		failed        []string
	)

	for _, res := range results {
		stepResult := ConvertStepResult(res, dryRun)
		totalDuration += stepResult.Duration

		if stepResult.Error != nil || stepResult.Status == stepStatusFailed {
			failed = append(failed, res.StepID)
		}

		execResult.StepResults = append(execResult.StepResults, stepResult)
	}

	execResult.StepCount = len(results)
	execResult.Duration = totalDuration

	if execErr != nil || validationErr != nil || len(failed) > 0 {
		execResult.Success = false
		execResult.Status = registry.StatusFailed
		execResult.FailedSteps = dedupeStrings(failed)

		switch {
		case execErr != nil:
			execResult.Error = &registry.ErrorDetail{
				Code:       "APPLY_FAILED",
				Message:    execErr.Error(),
				Context:    fmt.Sprintf("Config: %s", configPath),
				Suggestion: "Review step output for details",
			}
			execResult.Summary = execErr.Error()
		case validationErr != nil:
			execResult.Error = &registry.ErrorDetail{
				Code:       "VALIDATION_FAILED",
				Message:    validationErr.Error(),
				Context:    fmt.Sprintf("Config: %s", configPath),
				Suggestion: "Review validation results and retry",
			}
			execResult.Summary = validationErr.Error()
		case len(execResult.FailedSteps) > 0:
			execResult.Summary = fmt.Sprintf("%d steps failed", len(execResult.FailedSteps))
		default:
			execResult.Summary = "apply failed"
		}
	} else {
		execResult.Summary = fmt.Sprintf("All %d steps applied successfully", len(results))
	}

	return execResult
}

// ToStepState converts a domain step result into a TUI step state.
func ToStepState(res domainpipeline.StepResult, dryRun bool) components.StepState {
	status := mapStepStateStatus(res.Status)

	changed := res.Changed
	if dryRun && changed {
		status = components.StepStatusWouldUpdate
	}

	if res.Status == domainpipeline.StatusSkipped && dryRun && changed {
		status = components.StepStatusWouldCreate
	}

	return components.StepState{
		Status:   status,
		Message:  res.FormatOutput(),
		Error:    res.Error,
		Duration: res.Duration,
		Changed:  changed,
	}
}

func mapStepStateStatus(status domainpipeline.ResultStatus) components.StepStatus {
	switch status {
	case domainpipeline.StatusSuccess, domainpipeline.StatusAlreadySatisfied:
		return components.StepStatusSuccess
	case domainpipeline.StatusFailure:
		return components.StepStatusFailed
	case domainpipeline.StatusSkipped:
		return components.StepStatusSkipped
	default:
		return components.StepStatusFailed
	}
}

// IsParseError reports whether the error represents a configuration parse failure.
func IsParseError(err error) bool {
	var parseErr *streamyerrors.ParseError
	return errors.As(err, &parseErr)
}

// IsConfigError reports whether the error represents a configuration validation failure.
func IsConfigError(err error) bool {
	if err == nil {
		return false
	}

	if IsParseError(err) {
		return true
	}

	var validationErr *streamyerrors.ValidationError
	if errors.As(err, &validationErr) {
		return true
	}

	var domainErr *domainpipeline.DomainError
	if errors.As(err, &domainErr) {
		switch domainErr.Code {
		case domainpipeline.ErrCodeValidation,
			domainpipeline.ErrCodeDuplicate,
			domainpipeline.ErrCodeDependency,
			domainpipeline.ErrCodeMissing,
			domainpipeline.ErrCodeNotFound,
			domainpipeline.ErrCodeType:
			return true
		}
	}

	return false
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))

	result := make([]string, 0, len(values))
	for _, v := range values {
		if _, ok := seen[v]; ok {
			continue
		}

		seen[v] = struct{}{}
		result = append(result, v)
	}

	sort.Strings(result)

	return result
}
