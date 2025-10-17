package pipelineconv

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	configpkg "github.com/alexisbeaulieu97/streamy/internal/config"
	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	enginepkg "github.com/alexisbeaulieu97/streamy/internal/engine"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/registry"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

// ConvertPipelineToConfig maps a domain pipeline to the legacy config representation.
func ConvertPipelineToConfig(p *domainpipeline.Pipeline) *configpkg.Config {
	if p == nil {
		return &configpkg.Config{}
	}
	cfg := &configpkg.Config{
		Name:        p.Name,
		Description: p.Description,
		Settings: configpkg.Settings{
			Parallel:        p.Settings.Parallel,
			Timeout:         p.Settings.Timeout,
			ContinueOnError: p.Settings.ContinueOnError,
			DryRun:          p.Settings.DryRun,
			Verbose:         p.Settings.Verbose,
		},
		Steps: make([]configpkg.Step, len(p.Steps)),
	}
	for i, step := range p.Steps {
		cfg.Steps[i] = configpkg.Step{
			ID:            step.ID,
			Name:          step.Name,
			Type:          string(step.Type),
			DependsOn:     append([]string(nil), step.DependsOn...),
			Enabled:       step.Enabled,
			VerifyTimeout: step.VerifyTimeout,
		}
	}
	return cfg
}

// ConvertPlanToEngine translates the domain execution plan into the legacy engine representation.
func ConvertPlanToEngine(plan *domainpipeline.ExecutionPlan) *enginepkg.ExecutionPlan {
	if plan == nil {
		return &enginepkg.ExecutionPlan{}
	}
	levels := make([]enginepkg.ExecutionLevel, len(plan.Levels))
	baseDuration := time.Second
	if plan.EstimatedDuration > 0 && len(plan.Levels) > 0 {
		calculated := time.Duration(plan.EstimatedDuration/len(plan.Levels)) * time.Millisecond
		if calculated > 0 {
			baseDuration = calculated
		}
	}
	for i, level := range plan.Levels {
		levels[i] = enginepkg.ExecutionLevel{
			StepIDs:           append([]string(nil), level.StepIDs...),
			EstimatedDuration: baseDuration,
		}
	}
	return &enginepkg.ExecutionPlan{Levels: levels}
}

// ConvertStepResult maps a domain step result into the legacy model result for UI consumption.
func ConvertStepResult(res domainpipeline.StepResult, dryRun bool) model.StepResult {
	status := mapStepStatus(res.Status)
	if dryRun && res.Changed {
		status = model.StatusWouldUpdate
	}
	result := model.StepResult{
		StepID:    res.StepID,
		Status:    status,
		Message:   res.FormatOutput(),
		Error:     res.Error,
		Duration:  time.Duration(res.Duration) * time.Millisecond,
		Timestamp: time.Now().UTC(),
	}
	if res.Status == domainpipeline.StatusSkipped && dryRun && res.Changed {
		result.Status = model.StatusWouldCreate
	}
	return result
}

func mapStepStatus(status domainpipeline.ResultStatus) string {
	switch status {
	case domainpipeline.StatusSuccess:
		return model.StatusSuccess
	case domainpipeline.StatusFailure:
		return model.StatusFailed
	case domainpipeline.StatusSkipped:
		return model.StatusSkipped
	case domainpipeline.StatusAlreadySatisfied:
		return model.StatusSuccess
	default:
		return model.StatusFailed
	}
}

// BuildVerificationSummary aggregates verification results into the legacy summary model.
func BuildVerificationSummary(pipeline *domainpipeline.Pipeline, results []domainpipeline.VerificationResult) *model.VerificationSummary {
	summary := &model.VerificationSummary{
		TotalSteps: len(results),
		Results:    make([]*model.VerificationResult, 0, len(results)),
	}

	if pipeline != nil && len(pipeline.Steps) > 0 {
		summary.TotalSteps = len(pipeline.Steps)
	}

	for _, res := range results {
		status := deriveVerificationStatus(res)
		switch status {
		case model.StatusSatisfied:
			summary.Satisfied++
		case model.StatusMissing:
			summary.Missing++
		case model.StatusDrifted:
			summary.Drifted++
		case model.StatusBlocked:
			summary.Blocked++
		case model.StatusUnknown:
			summary.Unknown++
		}

		vr := &model.VerificationResult{
			StepID:    res.StepID,
			Status:    status,
			Message:   res.Message,
			Details:   formatVerificationDetails(res.Details),
			Timestamp: time.Now().UTC(),
		}
		summary.Results = append(summary.Results, vr)
	}

	return summary
}

// SummaryToExecutionResult maps a verification summary to the registry execution result.
func SummaryToExecutionResult(summary *model.VerificationSummary, configPath string) *registry.ExecutionResult {
	if summary == nil {
		return &registry.ExecutionResult{
			Operation:   "verify",
			Status:      registry.StatusFailed,
			Success:     false,
			Duration:    0,
			CompletedAt: time.Now().UTC(),
			StepResults: make([]registry.StepResult, 0),
			StepCount:   0,
			Summary:     "verification unavailable",
			Error: &registry.ErrorDetail{
				Message:    "Verification did not produce a summary",
				Context:    fmt.Sprintf("Config: %s", configPath),
				Suggestion: "Retry 'streamy verify' and check logs",
			},
		}
	}

	result := &registry.ExecutionResult{
		Operation:   "verify",
		Status:      pipelineStatusFromSummary(summary),
		Success:     summary.AllSatisfied(),
		Duration:    summary.Duration,
		CompletedAt: time.Now().UTC(),
		StepResults: make([]registry.StepResult, 0, len(summary.Results)),
	}

	var failed []string
	for _, r := range summary.Results {
		stepResult := registry.StepResult{
			StepID:   r.StepID,
			Status:   string(r.Status),
			Message:  r.Message,
			Duration: r.Duration,
		}
		if r.Error != nil {
			stepResult.Error = &registry.ErrorDetail{
				Message: r.Error.Error(),
				Context: fmt.Sprintf("Config: %s", configPath),
			}
			failed = append(failed, r.StepID)
		}
		switch r.Status {
		case model.StatusMissing, model.StatusDrifted, model.StatusBlocked, model.StatusUnknown:
			failed = append(failed, r.StepID)
		}

		result.StepResults = append(result.StepResults, stepResult)
	}

	result.StepCount = summary.TotalSteps
	result.FailedSteps = dedupeStrings(failed)

	switch {
	case summary.Missing > 0 || summary.Drifted > 0:
		result.Summary = fmt.Sprintf("%d steps need changes", summary.Missing+summary.Drifted)
	case summary.Blocked > 0 || summary.Unknown > 0:
		result.Summary = fmt.Sprintf("%d steps failed or unknown", summary.Blocked+summary.Unknown)
	default:
		result.Summary = fmt.Sprintf("All %d steps passed", summary.TotalSteps)
	}

	if len(result.FailedSteps) > 0 && result.Error == nil {
		result.Error = &registry.ErrorDetail{
			Message:    "Verification detected drift",
			Context:    fmt.Sprintf("Config: %s", configPath),
			Suggestion: "Run 'streamy apply' to reconcile changes",
		}
	}

	return result
}

func pipelineStatusFromSummary(summary *model.VerificationSummary) registry.PipelineStatus {
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

func deriveVerificationStatus(res domainpipeline.VerificationResult) model.VerificationStatus {
	if status := legacyStateFromDetails(res.Details); status != "" {
		switch strings.ToLower(strings.TrimSpace(status)) {
		case "satisfied":
			return model.StatusSatisfied
		case "missing", "not_satisfied", "not satisfied":
			return model.StatusMissing
		case "drifted", "drift", "not_matching":
			return model.StatusDrifted
		case "blocked":
			return model.StatusBlocked
		case "unknown":
			return model.StatusUnknown
		}
	}

	switch res.Status {
	case domainpipeline.VerificationSatisfied:
		return model.StatusSatisfied
	case domainpipeline.VerificationFailed:
		return model.StatusDrifted
	case domainpipeline.VerificationUnknown:
		return model.StatusUnknown
	default:
		return model.StatusUnknown
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

func ConvertApplyResults(results []model.StepResult, configPath string, execErr, validationErr error) *registry.ExecutionResult {
	execResult := &registry.ExecutionResult{
		Operation:   "apply",
		Status:      registry.StatusSatisfied,
		Success:     true,
		StepResults: make([]registry.StepResult, 0, len(results)),
		CompletedAt: time.Now().UTC(),
	}

	var totalDuration time.Duration
	var failed []string
	for _, res := range results {
		stepResult := registry.StepResult{
			StepID:   res.StepID,
			Status:   res.Status,
			Message:  res.Message,
			Duration: res.Duration,
		}

		totalDuration += res.Duration

		if res.Error != nil {
			stepResult.Error = &registry.ErrorDetail{
				Message: res.Error.Error(),
				Context: fmt.Sprintf("Config: %s, Step: %s", configPath, res.StepID),
			}
			failed = append(failed, res.StepID)
		}
		if res.Status == model.StatusFailed {
			failed = append(failed, res.StepID)
		}
		execResult.StepResults = append(execResult.StepResults, stepResult)
	}
	execResult.Duration = totalDuration

	if execErr != nil || validationErr != nil || len(failed) > 0 {
		execResult.Success = false
		execResult.Status = registry.StatusFailed
		execResult.FailedSteps = dedupeStrings(failed)
		if execErr != nil {
			execResult.Error = &registry.ErrorDetail{
				Code:       "APPLY_FAILED",
				Message:    execErr.Error(),
				Context:    fmt.Sprintf("Config: %s", configPath),
				Suggestion: "Review step output for details",
			}
			execResult.Summary = execErr.Error()
		} else if validationErr != nil {
			execResult.Error = &registry.ErrorDetail{
				Code:       "VALIDATION_FAILED",
				Message:    validationErr.Error(),
				Context:    fmt.Sprintf("Config: %s", configPath),
				Suggestion: "Review validation results and retry",
			}
			execResult.Summary = validationErr.Error()
		} else if len(execResult.FailedSteps) > 0 {
			execResult.Summary = fmt.Sprintf("%d steps failed", len(execResult.FailedSteps))
		}
	} else {
		execResult.Summary = fmt.Sprintf("All %d steps applied successfully", len(results))
	}

	return execResult
}

func IsParseError(err error) bool {
	var parseErr *streamyerrors.ParseError
	return errors.As(err, &parseErr)
}

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
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
