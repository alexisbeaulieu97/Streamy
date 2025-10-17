package pipeline

import (
	"context"

	"github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

// ApplyUseCase coordinates preparing, executing, and validating pipelines.
type ApplyUseCase struct {
	prepareUseCase *PrepareUseCase
	executor       ports.PluginExecutor
	validator      ports.ValidationService
	logger         ports.Logger
	events         ports.EventPublisher
}

// NewApplyUseCase constructs an ApplyUseCase with dependencies injected.
func NewApplyUseCase(prepare *PrepareUseCase, executor ports.PluginExecutor, validator ports.ValidationService, logger ports.Logger, events ports.EventPublisher) *ApplyUseCase {
	return &ApplyUseCase{
		prepareUseCase: prepare,
		executor:       executor,
		validator:      validator,
		logger:         logger,
		events:         events,
	}
}

// Apply executes the pipeline and runs validations.
func (u *ApplyUseCase) Apply(ctx context.Context, configPath string, dryRun bool) (*pipeline.Pipeline, []pipeline.StepResult, *pipeline.VerificationSummary, error) {
	if u.logger != nil {
		u.logger.Info(ctx, "applying pipeline", "config_path", configPath, "dry_run", dryRun)
	}

	pip, plan, err := u.prepareUseCase.Prepare(ctx, configPath)
	if err != nil {
		if u.logger != nil {
			u.logger.Error(ctx, "failed to prepare pipeline", "config_path", configPath, "error", err)
		}
		publishEvent(ctx, u.events, u.logger, ports.EventPipelineFailed, map[string]interface{}{
			"config_path": configPath,
			"phase":       "prepare",
			"error":       err,
		})
		return nil, nil, nil, err
	}

	publishEvent(ctx, u.events, u.logger, ports.EventPipelineStarted, map[string]interface{}{
		"config_path": configPath,
		"pipeline":    pip.Name,
		"step_count":  len(pip.Steps),
		"dry_run":     dryRun,
	})

	if u.logger != nil {
		u.logger.Info(ctx, "executing pipeline", "config_path", configPath, "levels", len(plan.Levels))
	}
	results, err := u.executor.Execute(ctx, plan, pip)
	if err != nil {
		if u.logger != nil {
			u.logger.Error(ctx, "pipeline execution failed", "config_path", configPath, "error", err)
		}
		publishEvent(ctx, u.events, u.logger, ports.EventPipelineFailed, map[string]interface{}{
			"config_path": configPath,
			"pipeline":    pip.Name,
			"error":       err,
		})
		return pip, results, nil, err
	}

	u.emitStepEvents(ctx, pip.Name, results)

	if dryRun {
		if u.logger != nil {
			u.logger.Info(ctx, "dry-run execution complete", "config_path", configPath)
		}
		publishEvent(ctx, u.events, u.logger, ports.EventPipelineCompleted, map[string]interface{}{
			"config_path": configPath,
			"pipeline":    pip.Name,
			"dry_run":     true,
		})
		return pip, results, nil, nil
	}

	if u.validator != nil {
		if summary, validationErr := u.validator.RunValidations(ctx, pip.Validations); validationErr != nil {
			if u.logger != nil {
				u.logger.Warn(ctx, "post-execution validations failed", "config_path", configPath, "error", validationErr)
			}
			publishEvent(ctx, u.events, u.logger, ports.EventValidationFailed, map[string]interface{}{
				"config_path": configPath,
				"pipeline":    pip.Name,
				"error":       validationErr,
			})
			return pip, results, &summary, validationErr
		} else {
			if u.logger != nil {
				u.logger.Info(ctx, "post-execution validations complete", "config_path", configPath, "passed", summary.FailedChecks == 0)
			}
			eventType := ports.EventValidationCompleted
			if summary.FailedChecks > 0 {
				eventType = ports.EventValidationFailed
			}
			publishEvent(ctx, u.events, u.logger, eventType, map[string]interface{}{
				"config_path": configPath,
				"pipeline":    pip.Name,
				"total":       summary.TotalChecks,
				"passed":      summary.PassedChecks,
				"failed":      summary.FailedChecks,
			})
			return pip, results, &summary, nil
		}
	} else if u.logger != nil {
		u.logger.Debug(ctx, "no validation service configured", "config_path", configPath)
	}

	if u.logger != nil {
		u.logger.Info(ctx, "pipeline applied successfully", "config_path", configPath)
	}
	publishEvent(ctx, u.events, u.logger, ports.EventPipelineCompleted, map[string]interface{}{
		"config_path": configPath,
		"pipeline":    pip.Name,
		"dry_run":     false,
	})
	return pip, results, nil, nil
}

func (u *ApplyUseCase) emitStepEvents(ctx context.Context, pipelineName string, results []pipeline.StepResult) {
	for _, result := range results {
		payload := map[string]interface{}{
			"pipeline": pipelineName,
			"step_id":  result.StepID,
			"status":   result.Status,
			"changed":  result.Changed,
			"duration": result.Duration,
		}
		eventType := ports.EventStepCompleted
		switch result.Status {
		case pipeline.StatusFailure:
			eventType = ports.EventStepFailed
			if result.Error != nil {
				payload["error"] = result.Error.Error()
			}
		case pipeline.StatusSkipped:
			eventType = ports.EventStepSkipped
		}
		publishEvent(ctx, u.events, u.logger, eventType, payload)
	}
}
