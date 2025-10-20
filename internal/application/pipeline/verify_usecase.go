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
	recorder := newVerifyMetricsRecorder(u.metrics)
	recorder.Started(ctx)

	var (
		status       string
		pipelineName string
	)

	defer func() {
		recorder.Completed(ctx, status, pipelineName)
	}()

	ctx, span, endSpan := startVerifySpan(ctx, u.tracer, configPath)
	defer endSpan()

	logInfo(ctx, u.logger, "verifying pipeline", "config_path", configPath)

	publishEvent(ctx, u.events, u.logger, ports.EventValidationStarted, map[string]interface{}{
		"config_path": configPath,
	})

	pip, _, err := u.prepareUseCase.Prepare(ctx, configPath)
	if err != nil {
		status = pipelineMetricStatus(err)

		setSpanStatus(span, ports.SpanStatusError, "prepare failed")
		logError(ctx, u.logger, "failed to prepare pipeline for verification", "config_path", configPath, "error", err)

		publishEvent(ctx, u.events, u.logger, ports.EventValidationFailed, map[string]interface{}{
			"config_path": configPath,
			"error":       err,
		})

		return nil, nil, wrapPrepareError(err)
	}

	pipelineName = pip.Name
	setSpanAttribute(span, "pipeline.name", pip.Name)
	setSpanAttribute(span, "step.count", len(pip.Steps))

	results, err := u.executor.Verify(ctx, pip)
	if err != nil {
		status = pipelineMetricStatus(err)

		setSpanStatus(span, ports.SpanStatusError, "verification failed")
		logError(ctx, u.logger, "pipeline verification failed", "config_path", configPath, "error", err)

		publishEvent(ctx, u.events, u.logger, ports.EventValidationFailed, map[string]interface{}{
			"config_path": configPath,
			"pipeline":    pip.Name,
			"error":       err,
		})

		return pip, results, wrapVerificationError(err)
	}

	status = metricStatusSuccess

	setSpanAttribute(span, "result.count", len(results))
	setSpanStatus(span, ports.SpanStatusOK, "pipeline verified")
	logInfo(ctx, u.logger, "pipeline verification complete", "config_path", configPath)

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

type verifyMetricsRecorder struct {
	metrics ports.MetricsCollector
	start   time.Time
}

func newVerifyMetricsRecorder(metrics ports.MetricsCollector) verifyMetricsRecorder {
	return verifyMetricsRecorder{
		metrics: metrics,
		start:   time.Now(),
	}
}

func (r verifyMetricsRecorder) Started(ctx context.Context) {
	if r.metrics == nil {
		return
	}

	r.metrics.IncCounter(ctx, metricPipelineVerificationsTotal, map[string]string{
		"status": metricStatusStarted,
	})
}

func (r verifyMetricsRecorder) Completed(ctx context.Context, status, pipelineName string) {
	if r.metrics == nil || status == "" {
		return
	}

	labels := map[string]string{
		"status": status,
	}
	if pipelineName != "" {
		labels["pipeline"] = pipelineName
	}

	r.metrics.IncCounter(ctx, metricPipelineVerificationsTotal, labels)
	r.metrics.ObserveHistogram(ctx, metricPipelineVerificationDurationSeconds, time.Since(r.start).Seconds(), labels)
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

func startVerifySpan(ctx context.Context, tracer ports.Tracer, configPath string) (context.Context, ports.Span, func()) {
	if tracer == nil {
		return ctx, nil, func() {}
	}

	spanCtx, span := tracer.StartSpan(ctx, "pipeline.verify", "config_path", configPath)
	if spanCtx != nil {
		ctx = spanCtx
	}

	return ctx, span, func() {
		if span != nil {
			span.End()
		}
	}
}
