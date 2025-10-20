// Package pipeline implements application use cases orchestrating pipeline execution.
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

// Apply orchestrates pipeline execution and optional post-run validations.
func (u *ApplyUseCase) Apply(ctx context.Context, configPath string, dryRun bool) (*pipeline.Pipeline, []pipeline.StepResult, *pipeline.VerificationSummary, error) {
	recorder := newApplyMetricsRecorder(u.metrics, dryRun)
	recorder.Started(ctx)

	var (
		status       string
		pipelineName string
	)

	defer func() {
		recorder.Completed(ctx, status, pipelineName)
	}()

	ctx, span, endSpan := startApplySpan(ctx, u.tracer, configPath, dryRun)
	defer endSpan()

	logInfo(ctx, u.logger, "applying pipeline", "config_path", configPath, "dry_run", dryRun)

	pip, plan, err := u.prepareUseCase.Prepare(ctx, configPath)
	if err != nil {
		status = pipelineMetricStatus(err)

		setSpanStatus(span, ports.SpanStatusError, "prepare failed")
		logError(ctx, u.logger, "failed to prepare pipeline", "config_path", configPath, "error", err)

		publishEvent(ctx, u.events, u.logger, ports.EventPipelineFailed, map[string]interface{}{
			"config_path": configPath,
			"phase":       "prepare",
			"error":       err,
		})

		return nil, nil, nil, wrapPrepareError(err)
	}

	pipelineName = pip.Name
	setSpanAttribute(span, "pipeline.name", pip.Name)
	setSpanAttribute(span, "step.count", len(pip.Steps))

	publishEvent(ctx, u.events, u.logger, ports.EventPipelineStarted, map[string]interface{}{
		"config_path": configPath,
		"pipeline":    pip.Name,
		"step_count":  len(pip.Steps),
		"dry_run":     dryRun,
	})

	logInfo(ctx, u.logger, "executing pipeline", "config_path", configPath, "levels", len(plan.Levels))

	results, err := u.executor.Execute(ctx, plan, pip)
	if err != nil {
		status = pipelineMetricStatus(err)

		setSpanStatus(span, ports.SpanStatusError, "execution failed")
		logError(ctx, u.logger, "pipeline execution failed", "config_path", configPath, "error", err)

		publishEvent(ctx, u.events, u.logger, ports.EventPipelineFailed, map[string]interface{}{
			"config_path": configPath,
			"pipeline":    pip.Name,
			"error":       err,
		})

		return pip, results, nil, wrapExecutionError(results, err)
	}

	u.emitStepEvents(ctx, pip.Name, results)

	setSpanAttribute(span, "step.executed", len(results))

	if dryRun {
		status = metricStatusSuccess

		setSpanStatus(span, ports.SpanStatusOK, "pipeline dry-run completed")
		logInfo(ctx, u.logger, "dry-run execution complete", "config_path", configPath)

		publishEvent(ctx, u.events, u.logger, ports.EventPipelineCompleted, map[string]interface{}{
			"config_path": configPath,
			"pipeline":    pip.Name,
			"dry_run":     true,
		})

		return pip, results, nil, nil
	}

	status = metricStatusSuccess

	summary, validationErr := u.runValidations(ctx, configPath, pip, span)
	if validationErr != nil {
		status = pipelineMetricStatus(validationErr)
		return pip, results, summary, validationErr
	}

	setSpanStatus(span, ports.SpanStatusOK, "pipeline applied successfully")
	logInfo(ctx, u.logger, "pipeline applied successfully", "config_path", configPath)

	publishEvent(ctx, u.events, u.logger, ports.EventPipelineCompleted, map[string]interface{}{
		"config_path": configPath,
		"pipeline":    pip.Name,
		"dry_run":     false,
	})

	return pip, results, summary, nil
}

func (u *ApplyUseCase) runValidations(ctx context.Context, configPath string, pip *pipeline.Pipeline, span ports.Span) (*pipeline.VerificationSummary, error) {
	if u.validator == nil {
		logDebug(ctx, u.logger, "no validation service configured", "config_path", configPath)
		return nil, nil
	}

	summary, validationErr := u.validator.RunValidations(ctx, pip.Validations)
	if validationErr != nil {
		setSpanStatus(span, ports.SpanStatusError, "validation failed")
		logWarn(ctx, u.logger, "post-execution validations failed", "config_path", configPath, "error", validationErr)

		publishEvent(ctx, u.events, u.logger, ports.EventValidationFailed, map[string]interface{}{
			"config_path": configPath,
			"pipeline":    pip.Name,
			"error":       validationErr,
		})

		return &summary, wrapValidationError(validationErr)
	}

	logInfo(ctx, u.logger, "post-execution validations complete", "config_path", configPath, "passed", summary.FailedChecks == 0)

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

	setSpanStatus(span, ports.SpanStatusOK, "pipeline applied successfully")

	return &summary, nil
}

func (u *ApplyUseCase) emitStepEvents(ctx context.Context, pipelineName string, results []pipeline.StepResult) {
	for _, result := range results {
		payload := map[string]interface{}{
			"pipeline": pipelineName,
			"step_id":  result.StepID,
			"status":   result.Status,
			"changed":  result.Changed,
			"duration": result.Duration.Milliseconds(),
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

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return metricStatusCancelled
	}

	var domainErr *pipeline.DomainError
	if errors.As(err, &domainErr) {
		switch domainErr.Code {
		case pipeline.ErrCodeCancelled, pipeline.ErrCodeTimeout:
			return metricStatusCancelled
		}
	}

	return metricStatusFailure
}

type applyMetricsRecorder struct {
	metrics     ports.MetricsCollector
	dryRunLabel string
	start       time.Time
}

func newApplyMetricsRecorder(metrics ports.MetricsCollector, dryRun bool) applyMetricsRecorder {
	return applyMetricsRecorder{
		metrics:     metrics,
		dryRunLabel: strconv.FormatBool(dryRun),
		start:       time.Now(),
	}
}

func (r applyMetricsRecorder) Started(ctx context.Context) {
	if r.metrics == nil {
		return
	}

	r.metrics.IncCounter(ctx, metricPipelineExecutionsTotal, map[string]string{
		"status":  metricStatusStarted,
		"dry_run": r.dryRunLabel,
	})
}

func (r applyMetricsRecorder) Completed(ctx context.Context, status, pipelineName string) {
	if r.metrics == nil || status == "" {
		return
	}

	labels := map[string]string{
		"status":  status,
		"dry_run": r.dryRunLabel,
	}
	if pipelineName != "" {
		labels["pipeline"] = pipelineName
	}

	r.metrics.IncCounter(ctx, metricPipelineExecutionsTotal, labels)
	r.metrics.ObserveHistogram(ctx, metricPipelineExecutionDurationSeconds, time.Since(r.start).Seconds(), labels)
}

func startApplySpan(ctx context.Context, tracer ports.Tracer, configPath string, dryRun bool) (context.Context, ports.Span, func()) {
	if tracer == nil {
		return ctx, nil, func() {}
	}

	spanCtx, span := tracer.StartSpan(ctx, "pipeline.apply", "config_path", configPath, "dry_run", dryRun)
	if spanCtx != nil {
		ctx = spanCtx
	}

	return ctx, span, func() {
		if span != nil {
			span.End()
		}
	}
}

func setSpanStatus(span ports.Span, status ports.SpanStatus, message string) {
	if span == nil {
		return
	}

	span.SetStatus(status, message)
}

func setSpanAttribute(span ports.Span, key string, value interface{}) {
	if span == nil {
		return
	}

	span.SetAttribute(key, value)
}

func logInfo(ctx context.Context, logger ports.Logger, msg string, keyvals ...interface{}) {
	if logger == nil {
		return
	}

	logger.Info(ctx, msg, keyvals...)
}

func logWarn(ctx context.Context, logger ports.Logger, msg string, keyvals ...interface{}) {
	if logger == nil {
		return
	}

	logger.Warn(ctx, msg, keyvals...)
}

func logError(ctx context.Context, logger ports.Logger, msg string, keyvals ...interface{}) {
	if logger == nil {
		return
	}

	logger.Error(ctx, msg, keyvals...)
}

func logDebug(ctx context.Context, logger ports.Logger, msg string, keyvals ...interface{}) {
	if logger == nil {
		return
	}

	logger.Debug(ctx, msg, keyvals...)
}
