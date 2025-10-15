package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/logger"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/registry"
	"github.com/alexisbeaulieu97/streamy/internal/validation"
)

// Re-export domain types to preserve the public API for callers within the app layer.
type PreparedPipeline = pipeline.PreparedPipeline

// Service coordinates high-level pipeline operations and adapts domain results for registry/TUI consumers.
type Service struct {
	domain *pipeline.Service
}

// NewService constructs an application pipeline service.
func NewService(reg *plugin.PluginRegistry) *Service {
	return &Service{
		domain: pipeline.NewService(reg),
	}
}

// Prepare loads configuration and execution artefacts.
func (s *Service) Prepare(configPath string) (*PreparedPipeline, error) {
	return s.domain.Prepare(configPath)
}

// VerifyRequest configures a verification run (app-level).
type VerifyRequest struct {
	Prepared       *PreparedPipeline
	ConfigPath     string
	LoggerOptions  logger.Options
	Verbose        bool
	PerStepTimeout time.Duration
	DefaultTimeout time.Duration
}

// VerifyOutcome returns verification details along with registry execution metadata.
type VerifyOutcome struct {
	Prepared        *PreparedPipeline
	Summary         *model.VerificationSummary
	ExecutionResult *registry.ExecutionResult
}

// Verify executes verification for a pipeline, returning the summary and registry execution result.
func (s *Service) Verify(ctx context.Context, req VerifyRequest) (*VerifyOutcome, error) {
	log, err := logger.New(req.LoggerOptions)
	if err != nil {
		return nil, fmt.Errorf("create logger: %w", err)
	}

	domainOutcome, verifyErr := s.domain.Verify(ctx, pipeline.VerifyRequest{
		Prepared:       req.Prepared,
		ConfigPath:     req.ConfigPath,
		Logger:         log,
		Verbose:        req.Verbose,
		PerStepTimeout: req.PerStepTimeout,
		DefaultTimeout: req.DefaultTimeout,
	})

	if domainOutcome == nil {
		return nil, verifyErr
	}

	outcome := &VerifyOutcome{
		Prepared: domainOutcome.Prepared,
		Summary:  domainOutcome.Summary,
	}

	if domainOutcome.Summary != nil {
		outcome.ExecutionResult = convertVerificationSummary(domainOutcome.Summary, domainOutcome.Prepared.Path)
	}

	if verifyErr != nil {
		if outcome.ExecutionResult == nil {
			outcome.ExecutionResult = failedExecutionResult("verify", domainOutcome.Prepared.Path, verifyErr)
		} else {
			outcome.ExecutionResult.Status = registry.StatusFailed
			outcome.ExecutionResult.Success = false
			outcome.ExecutionResult.Error = &registry.ErrorDetail{
				Code:       "VERIFY_FAILED",
				Message:    verifyErr.Error(),
				Context:    fmt.Sprintf("Config: %s", domainOutcome.Prepared.Path),
				Suggestion: "Check config file syntax and step definitions",
			}
		}
		return outcome, verifyErr
	}

	return outcome, nil
}

// ApplyRequest configures an apply run (app-level).
type ApplyRequest struct {
	Prepared        *PreparedPipeline
	ConfigPath      string
	LoggerOptions   logger.Options
	DryRunOverride  bool
	VerboseOverride bool
	ContinueOnError bool
	OnStepResult    func(model.StepResult)
	OnValidation    func(validation.ValidationResult)
}

// ApplyOutcome captures app-level apply execution details.
type ApplyOutcome struct {
	Prepared          *PreparedPipeline
	Results           []model.StepResult
	ValidationResults []validation.ValidationResult
	ExecutionResult   *registry.ExecutionResult
}

// Apply executes a pipeline apply operation, returning step results, validation data, and a registry execution summary.
func (s *Service) Apply(ctx context.Context, req ApplyRequest) (*ApplyOutcome, error) {
	log, err := logger.New(req.LoggerOptions)
	if err != nil {
		return nil, fmt.Errorf("create logger: %w", err)
	}

	domainOutcome, applyErr := s.domain.Apply(ctx, pipeline.ApplyRequest{
		Prepared:        req.Prepared,
		ConfigPath:      req.ConfigPath,
		Logger:          log,
		DryRunOverride:  req.DryRunOverride,
		VerboseOverride: req.VerboseOverride,
		ContinueOnError: req.ContinueOnError,
		OnStepResult:    req.OnStepResult,
		OnValidation:    req.OnValidation,
	})
	if domainOutcome == nil {
		return nil, applyErr
	}

	outcome := &ApplyOutcome{
		Prepared:          domainOutcome.Prepared,
		Results:           domainOutcome.Results,
		ValidationResults: domainOutcome.ValidationResults,
	}

	outcome.ExecutionResult = convertApplyResults(domainOutcome.Results, domainOutcome.Prepared.Path, domainOutcome.ExecutionErr, domainOutcome.ValidationErr)
	outcome.ExecutionResult.StepCount = len(domainOutcome.Prepared.Config.Steps)

	if applyErr != nil {
		return outcome, applyErr
	}

	return outcome, nil
}

func convertVerificationSummary(summary *model.VerificationSummary, configPath string) *registry.ExecutionResult {
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

	result.StepCount = len(summary.Results)
	result.FailedSteps = dedupeStrings(failed)

	switch {
	case summary == nil:
		result.Summary = "verification unavailable"
	case summary.Missing > 0 || summary.Drifted > 0:
		result.Summary = fmt.Sprintf("%d steps need changes", summary.Missing+summary.Drifted)
	case summary.Blocked > 0 || summary.Unknown > 0:
		result.Summary = fmt.Sprintf("%d steps failed or unknown", summary.Blocked+summary.Unknown)
	default:
		result.Summary = fmt.Sprintf("All %d steps passed", summary.Satisfied)
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

func convertApplyResults(results []model.StepResult, configPath string, execErr, validationErr error) *registry.ExecutionResult {
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

func pipelineStatusFromSummary(summary *model.VerificationSummary) registry.PipelineStatus {
	switch {
	case summary == nil:
		return registry.StatusFailed
	case summary.AllSatisfied():
		return registry.StatusSatisfied
	case summary.Missing > 0 || summary.Drifted > 0:
		return registry.StatusDrifted
	default:
		return registry.StatusFailed
	}
}

func failedExecutionResult(operation, configPath string, err error) *registry.ExecutionResult {
	return &registry.ExecutionResult{
		Operation: operation,
		Status:    registry.StatusFailed,
		Success:   false,
		Error: &registry.ErrorDetail{
			Code:       "PIPELINE_ERROR",
			Message:    err.Error(),
			Context:    fmt.Sprintf("Config: %s", configPath),
			Suggestion: "Inspect error details and fix configuration",
		},
		CompletedAt: time.Now().UTC(),
	}
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
