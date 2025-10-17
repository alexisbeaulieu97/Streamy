package engine

import (
	"context"
	"time"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

// TestExecutor is a simple, deterministic PluginExecutor implementation used in
// integration tests to validate that infrastructure components can be swapped
// without touching domain or application layers.
type TestExecutor struct {
	registry ports.PluginRegistry
}

// NewTestExecutor constructs a test executor backed by the supplied registry.
func NewTestExecutor(registry ports.PluginRegistry) *TestExecutor {
	return &TestExecutor{registry: registry}
}

// Execute runs the provided plan sequentially. It intentionally avoids worker
// pools and tracing to keep behaviour deterministic for comparison tests.
func (e *TestExecutor) Execute(ctx context.Context, plan *domainpipeline.ExecutionPlan, pipeline *domainpipeline.Pipeline) ([]domainpipeline.StepResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if plan == nil {
		return nil, &domainpipeline.DomainError{Code: domainpipeline.ErrCodeInternal, Message: "execution plan is nil"}
	}
	if pipeline == nil {
		return nil, &domainpipeline.DomainError{Code: domainpipeline.ErrCodeInternal, Message: "pipeline is nil"}
	}

	settings := pipeline.EffectiveSettings()
	results := make([]domainpipeline.StepResult, 0, plan.TotalSteps)
	var firstErr error

	for _, level := range plan.Levels {
		for _, stepID := range level.StepIDs {
			if err := ctx.Err(); err != nil {
				return results, &domainpipeline.DomainError{
					Code:    domainpipeline.ErrCodeCancelled,
					Message: "execution cancelled",
					Cause:   err,
				}
			}

			step, err := pipeline.GetStep(stepID)
			if err != nil {
				return results, err
			}

			stepResult, execErr := e.runStep(ctx, *step, settings.DryRun)
			results = append(results, stepResult)
			if execErr != nil {
				if firstErr == nil {
					firstErr = execErr
				}
				if !settings.ContinueOnError {
					return results, execErr
				}
			}
		}
	}

	return results, firstErr
}

func (e *TestExecutor) runStep(ctx context.Context, step domainpipeline.Step, dryRun bool) (domainpipeline.StepResult, error) {
	pluginType := domainplugin.Type(step.Type)
	handler, err := e.registry.Get(pluginType)
	if err != nil {
		return domainpipeline.StepResult{
			StepID: step.ID,
			Status: domainpipeline.StatusFailure,
			Error:  toDomainError(err),
		}, err
	}

	start := time.Now()

	eval, err := handler.Evaluate(ctx, step)
	if err != nil {
		return domainpipeline.StepResult{
			StepID:   step.ID,
			Status:   domainpipeline.StatusFailure,
			Error:    toDomainError(err),
			Duration: int(time.Since(start).Milliseconds()),
		}, err
	}

	if dryRun {
		return dryRunResult(step, eval, time.Since(start)), nil
	}

	if eval != nil && !eval.RequiresAction {
		return domainpipeline.StepResult{
			StepID:   step.ID,
			Status:   domainpipeline.StatusAlreadySatisfied,
			Message:  "step already satisfied",
			Duration: int(time.Since(start).Milliseconds()),
		}, nil
	}

	result, err := handler.Apply(ctx, eval, step)
	if result == nil {
		result = &domainpipeline.StepResult{StepID: step.ID}
	}
	result.Duration = int(time.Since(start).Milliseconds())

	if err != nil {
		result.Status = domainpipeline.StatusFailure
		result.Error = toDomainError(err)
		return *result, err
	}

	return *result, nil
}

// Verify evaluates each step sequentially and returns the aggregated results.
func (e *TestExecutor) Verify(ctx context.Context, pipeline *domainpipeline.Pipeline) ([]domainpipeline.VerificationResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if pipeline == nil {
		return nil, &domainpipeline.DomainError{Code: domainpipeline.ErrCodeInternal, Message: "pipeline is nil"}
	}

	results := make([]domainpipeline.VerificationResult, 0, len(pipeline.Steps))
	var firstErr error

	for _, step := range pipeline.Steps {
		if err := ctx.Err(); err != nil {
			return results, &domainpipeline.DomainError{
				Code:    domainpipeline.ErrCodeCancelled,
				Message: "verification cancelled",
				Cause:   err,
			}
		}

		result, err := e.verifyStep(ctx, step)
		results = append(results, result)
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return results, firstErr
}

func (e *TestExecutor) verifyStep(ctx context.Context, step domainpipeline.Step) (domainpipeline.VerificationResult, error) {
	pluginType := domainplugin.Type(step.Type)
	handler, err := e.registry.Get(pluginType)
	if err != nil {
		derr := toDomainError(err)
		return domainpipeline.VerificationResult{
			StepID:  step.ID,
			Type:    string(pluginType),
			Status:  domainpipeline.VerificationFailed,
			Message: derr.Error(),
			Details: map[string]interface{}{
				"step_id": step.ID,
			},
		}, err
	}

	eval, err := handler.Evaluate(ctx, step)
	if err != nil {
		derr := toDomainError(err)
		details := map[string]interface{}{"step_id": step.ID}
		if status := categorizeVerificationError(derr); status != "" {
			details["status"] = status
		}
		return domainpipeline.VerificationResult{
			StepID:  step.ID,
			Type:    string(pluginType),
			Status:  domainpipeline.VerificationFailed,
			Message: derr.Error(),
			Details: details,
		}, err
	}

	result := domainpipeline.VerificationResult{
		StepID:  step.ID,
		Type:    string(pluginType),
		Status:  domainpipeline.VerificationSatisfied,
		Message: "step satisfied",
	}

	if eval != nil && eval.RequiresAction {
		result.Status = domainpipeline.VerificationFailed
		result.Message = "step drift detected"
		result.Details = map[string]interface{}{
			"current_state": eval.CurrentState,
			"desired_state": eval.DesiredState,
			"diff":          eval.Diff,
			"status":        "drifted",
		}
	} else if eval != nil {
		result.Details = map[string]interface{}{
			"current_state": eval.CurrentState,
			"desired_state": eval.DesiredState,
		}
	}

	return result, nil
}

var _ ports.PluginExecutor = (*TestExecutor)(nil)
