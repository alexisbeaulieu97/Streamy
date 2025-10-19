package pipeline

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

const (
	metricPipelineExecutionsTotal          = "streamy_pipeline_executions_total"
	metricPipelineExecutionDurationSeconds = "streamy_pipeline_execution_duration_seconds"
)

// ApplyUseCase coordinates preparing, executing, and validating pipelines.
type ApplyUseCase struct {
	prepareUseCase *PrepareUseCase
	executor       ports.PluginExecutor
	validator      ports.ValidationService
	logger         ports.Logger
	metrics        ports.MetricsCollector
	tracer         ports.Tracer
	events         ports.EventPublisher
}

// NewApplyUseCase constructs an ApplyUseCase with dependencies injected.
func NewApplyUseCase(
	prepare *PrepareUseCase,
	executor ports.PluginExecutor,
	validator ports.ValidationService,
	logger ports.Logger,
	metrics ports.MetricsCollector,
	tracer ports.Tracer,
	events ports.EventPublisher,
) *ApplyUseCase {
	return &ApplyUseCase{
		prepareUseCase: prepare,
		executor:       executor,
		validator:      validator,
		logger:         logger,
		metrics:        metrics,
		tracer:         tracer,
		events:         events,
	}
}

// Apply executes the pipeline and runs validations.
func (u *ApplyUseCase) Apply(ctx context.Context, configPath string, dryRun bool) (*pipeline.Pipeline, []pipeline.StepResult, *pipeline.VerificationSummary, error) {
	start := time.Now()
	dryRunLabel := strconv.FormatBool(dryRun)
	pipelineName := ""
	status := ""

	recordStarted := func() {
		if u.metrics == nil {
			return
		}
		u.metrics.IncCounter(ctx, metricPipelineExecutionsTotal, map[string]string{
			"status":  metricStatusStarted,
			"dry_run": dryRunLabel,
		})
	}

	recordCompletion := func() {
		if status == "" || u.metrics == nil {
			return
		}
		labels := map[string]string{
			"status":  status,
			"dry_run": dryRunLabel,
		}
		if pipelineName != "" {
			labels["pipeline"] = pipelineName
		}
		u.metrics.IncCounter(ctx, metricPipelineExecutionsTotal, labels)
		u.metrics.ObserveHistogram(ctx, metricPipelineExecutionDurationSeconds, time.Since(start).Seconds(), labels)
	}
	defer recordCompletion()

	recordStarted()

	var span ports.Span
	if u.tracer != nil {
		var spanCtx context.Context
		spanCtx, span = u.tracer.StartSpan(ctx, "pipeline.apply", "config_path", configPath, "dry_run", dryRun)
		if spanCtx != nil {
			ctx = spanCtx
		}
		defer span.End()
	}

	if u.logger != nil {
		u.logger.Info(ctx, "applying pipeline", "config_path", configPath, "dry_run", dryRun)
	}

	pip, plan, err := u.prepareUseCase.Prepare(ctx, configPath)
	if err != nil {
		status = pipelineMetricStatus(err)
		if span != nil {
			span.SetStatus(ports.SpanStatusError, "prepare failed")
		}
		if u.logger != nil {
			u.logger.Error(ctx, "failed to prepare pipeline", "config_path", configPath, "error", err)
		}
		publishEvent(ctx, u.events, u.logger, ports.EventPipelineFailed, map[string]interface{}{
			"config_path": configPath,
			"phase":       "prepare",
			"error":       err,
		})
		return nil, nil, nil, wrapPrepareError(err)
	}

	pipelineName = pip.Name
	if span != nil {
		span.SetAttribute("pipeline.name", pip.Name)
		span.SetAttribute("step.count", len(pip.Steps))
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
		status = pipelineMetricStatus(err)
		if span != nil {
			span.SetStatus(ports.SpanStatusError, "execution failed")
		}
		if u.logger != nil {
			u.logger.Error(ctx, "pipeline execution failed", "config_path", configPath, "error", err)
		}
		publishEvent(ctx, u.events, u.logger, ports.EventPipelineFailed, map[string]interface{}{
			"config_path": configPath,
			"pipeline":    pip.Name,
			"error":       err,
		})
		return pip, results, nil, wrapExecutionError(results, err)
	}

	u.emitStepEvents(ctx, pip.Name, results)
	if span != nil {
		span.SetAttribute("step.executed", len(results))
	}

	if dryRun {
		status = metricStatusSuccess
		if span != nil {
			span.SetStatus(ports.SpanStatusOK, "pipeline dry-run completed")
		}
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
			status = pipelineMetricStatus(validationErr)
			if span != nil {
				span.SetStatus(ports.SpanStatusError, "validation failed")
			}
			if u.logger != nil {
				u.logger.Warn(ctx, "post-execution validations failed", "config_path", configPath, "error", validationErr)
			}
			publishEvent(ctx, u.events, u.logger, ports.EventValidationFailed, map[string]interface{}{
				"config_path": configPath,
				"pipeline":    pip.Name,
				"error":       validationErr,
			})
			return pip, results, &summary, wrapValidationError(validationErr)
		} else {
			status = metricStatusSuccess
			if span != nil {
				span.SetStatus(ports.SpanStatusOK, "pipeline applied successfully")
			}
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

	status = metricStatusSuccess
	if span != nil {
		span.SetStatus(ports.SpanStatusOK, "pipeline applied successfully")
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

func pipelineMetricStatus(err error) string {
	if err == nil {
		return metricStatusSuccess
	}
	if errors.Is(err, context.Canceled) {
		return metricStatusCancelled
	}
	var domainErr *pipeline.DomainError
	if errors.As(err, &domainErr) && domainErr.Code == pipeline.ErrCodeCancelled {
		return metricStatusCancelled
	}
	return metricStatusFailure
}
