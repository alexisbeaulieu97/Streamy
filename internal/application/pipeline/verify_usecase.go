package pipeline

import (
	"context"
	"time"

	"github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

const (
	metricPipelineVerificationsTotal          = "streamy_pipeline_verifications_total"
	metricPipelineVerificationDurationSeconds = "streamy_pipeline_verification_duration_seconds"
)

// VerifyUseCase coordinates preparing and verifying pipelines without applying changes.
type VerifyUseCase struct {
	prepareUseCase *PrepareUseCase
	executor       ports.PluginExecutor
	logger         ports.Logger
	metrics        ports.MetricsCollector
	tracer         ports.Tracer
	events         ports.EventPublisher
}

// NewVerifyUseCase constructs a VerifyUseCase with dependencies injected.
func NewVerifyUseCase(prepare *PrepareUseCase, executor ports.PluginExecutor, logger ports.Logger, metrics ports.MetricsCollector, tracer ports.Tracer, events ports.EventPublisher) *VerifyUseCase {
	return &VerifyUseCase{
		prepareUseCase: prepare,
		executor:       executor,
		logger:         logger,
		metrics:        metrics,
		tracer:         tracer,
		events:         events,
	}
}

// Verify loads the pipeline and runs verification via the plugin executor.
func (u *VerifyUseCase) Verify(ctx context.Context, configPath string) (*pipeline.Pipeline, []pipeline.VerificationResult, error) {
	start := time.Now()
	status := ""
	pipelineName := ""

	recordStarted := func() {
		if u.metrics == nil {
			return
		}
		u.metrics.IncCounter(ctx, metricPipelineVerificationsTotal, map[string]string{
			"status": metricStatusStarted,
		})
	}

	recordCompletion := func() {
		if status == "" || u.metrics == nil {
			return
		}
		labels := map[string]string{
			"status": status,
		}
		if pipelineName != "" {
			labels["pipeline"] = pipelineName
		}
		u.metrics.IncCounter(ctx, metricPipelineVerificationsTotal, labels)
		u.metrics.ObserveHistogram(ctx, metricPipelineVerificationDurationSeconds, time.Since(start).Seconds(), labels)
	}
	defer recordCompletion()

	recordStarted()

	var span ports.Span
	if u.tracer != nil {
		var spanCtx context.Context
		spanCtx, span = u.tracer.StartSpan(ctx, "pipeline.verify", "config_path", configPath)
		if spanCtx != nil {
			ctx = spanCtx
		}
		defer span.End()
	}

	if u.logger != nil {
		u.logger.Info(ctx, "verifying pipeline", "config_path", configPath)
	}
	publishEvent(ctx, u.events, u.logger, ports.EventValidationStarted, map[string]interface{}{
		"config_path": configPath,
	})

	pip, _, err := u.prepareUseCase.Prepare(ctx, configPath)
	if err != nil {
		status = pipelineMetricStatus(err)
		if span != nil {
			span.SetStatus(ports.SpanStatusError, "prepare failed")
		}
		if u.logger != nil {
			u.logger.Error(ctx, "failed to prepare pipeline for verification", "config_path", configPath, "error", err)
		}
		publishEvent(ctx, u.events, u.logger, ports.EventValidationFailed, map[string]interface{}{
			"config_path": configPath,
			"error":       err,
		})
		return nil, nil, wrapPrepareError(err)
	}
	pipelineName = pip.Name
	if span != nil {
		span.SetAttribute("pipeline.name", pip.Name)
		span.SetAttribute("step.count", len(pip.Steps))
	}

	results, err := u.executor.Verify(ctx, pip)
	if err != nil {
		status = pipelineMetricStatus(err)
		if span != nil {
			span.SetStatus(ports.SpanStatusError, "verification failed")
		}
		if u.logger != nil {
			u.logger.Error(ctx, "pipeline verification failed", "config_path", configPath, "error", err)
		}
		publishEvent(ctx, u.events, u.logger, ports.EventValidationFailed, map[string]interface{}{
			"config_path": configPath,
			"pipeline":    pip.Name,
			"error":       err,
		})
		return pip, results, wrapVerificationError(err)
	}

	status = metricStatusSuccess
	if span != nil {
		span.SetAttribute("result.count", len(results))
		span.SetStatus(ports.SpanStatusOK, "pipeline verified")
	}

	if u.logger != nil {
		u.logger.Info(ctx, "pipeline verification complete", "config_path", configPath)
	}
	passed, failed, unknown := summarizeVerificationResults(results)
	publishEvent(ctx, u.events, u.logger, ports.EventValidationCompleted, map[string]interface{}{
		"config_path": configPath,
		"pipeline":    pip.Name,
		"passed":      passed,
		"failed":      failed,
		"unknown":     unknown,
	})
	return pip, results, nil
}

func summarizeVerificationResults(results []pipeline.VerificationResult) (int, int, int) {
	var passed, failed, unknown int
	for _, result := range results {
		switch result.Status {
		case pipeline.VerificationSatisfied:
			passed++
		case pipeline.VerificationFailed:
			failed++
		default:
			unknown++
		}
	}
	return passed, failed, unknown
}
