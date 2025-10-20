package pipeline

import (
	"context"

	"github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

// PrepareUseCase orchestrates configuration loading, validation, and planning.
type PrepareUseCase struct {
	configLoader ports.ConfigLoader
	dagBuilder   ports.DAGBuilder
	logger       ports.Logger
	tracer       ports.Tracer
	events       ports.EventPublisher
}

// NewPrepareUseCase constructs a prepare use case with the required ports.
func NewPrepareUseCase(loader ports.ConfigLoader, builder ports.DAGBuilder, logger ports.Logger, tracer ports.Tracer, events ports.EventPublisher) *PrepareUseCase {
	return &PrepareUseCase{
		configLoader: loader,
		dagBuilder:   builder,
		logger:       logger,
		tracer:       tracer,
		events:       events,
	}
}

// Prepare loads the pipeline configuration, validates it, builds the DAG, and generates an execution plan.
func (u *PrepareUseCase) Prepare(ctx context.Context, configPath string) (*pipeline.Pipeline, *pipeline.ExecutionPlan, error) {
	ctx, span, endSpan := startPrepareSpan(ctx, u.tracer, configPath)
	defer endSpan()

	logInfo(ctx, u.logger, "preparing pipeline", "config_path", configPath)

	publishEvent(ctx, u.events, u.logger, ports.EventPipelineStarted, map[string]interface{}{
		"config_path": configPath,
		"phase":       "prepare",
	})

	pip, err := u.loadPipeline(ctx, configPath, span)
	if err != nil {
		return nil, nil, err
	}

	setSpanAttribute(span, "pipeline.name", pip.Name)
	setSpanAttribute(span, "step.count", len(pip.Steps))

	plan, err := u.buildExecutionPlan(ctx, configPath, pip, span)
	if err != nil {
		return pip, nil, err
	}

	if err := u.validateExecutionPlan(ctx, configPath, pip, plan, span); err != nil {
		return pip, nil, err
	}

	logInfo(ctx, u.logger, "pipeline prepared", "config_path", configPath, "levels", len(plan.Levels))
	setSpanAttribute(span, "plan.levels", len(plan.Levels))
	setSpanStatus(span, ports.SpanStatusOK, "pipeline prepared")

	publishEvent(ctx, u.events, u.logger, ports.EventPipelineCompleted, map[string]interface{}{
		"config_path": configPath,
		"phase":       "prepare",
		"levels":      len(plan.Levels),
		"step_count":  len(pip.Steps),
	})

	return pip, plan, nil
}

func (u *PrepareUseCase) loadPipeline(ctx context.Context, configPath string, span ports.Span) (*pipeline.Pipeline, error) {
	pip, err := u.configLoader.Load(ctx, configPath)
	if err != nil {
		return nil, u.failPrepare(ctx, configPath, span, err, "failed to load pipeline configuration", "configuration load failed", wrapPrepareError)
	}

	return pip, nil
}

func (u *PrepareUseCase) buildExecutionPlan(ctx context.Context, configPath string, pip *pipeline.Pipeline, span ports.Span) (*pipeline.ExecutionPlan, error) {
	logDebug(ctx, u.logger, "building execution plan", "config_path", configPath, "step_count", len(pip.Steps))

	plan, err := u.dagBuilder.Build(ctx, pip.Steps)
	if err != nil {
		return nil, u.failPrepare(ctx, configPath, span, err, "failed to build execution plan", "execution plan build failed", wrapDagBuildError)
	}

	return plan, nil
}

func (u *PrepareUseCase) validateExecutionPlan(ctx context.Context, configPath string, pip *pipeline.Pipeline, plan *pipeline.ExecutionPlan, span ports.Span) error {
	if err := plan.Validate(*pip); err != nil {
		return u.failPrepare(ctx, configPath, span, err, "execution plan validation failed", "execution plan validation failed", wrapDagValidationError)
	}

	return nil
}

func (u *PrepareUseCase) failPrepare(ctx context.Context, configPath string, span ports.Span, err error, logMessage, spanMessage string, wrap func(error) error) error {
	logError(ctx, u.logger, logMessage, "config_path", configPath, "error", err)
	setSpanStatus(span, ports.SpanStatusError, spanMessage)

	publishEvent(ctx, u.events, u.logger, ports.EventPipelineFailed, map[string]interface{}{
		"config_path": configPath,
		"phase":       "prepare",
		"error":       err,
	})

	return wrap(err)
}

func startPrepareSpan(ctx context.Context, tracer ports.Tracer, configPath string) (context.Context, ports.Span, func()) {
	if tracer == nil {
		return ctx, nil, func() {}
	}

	spanCtx, span := tracer.StartSpan(ctx, "pipeline.prepare", "config_path", configPath)
	if spanCtx != nil {
		ctx = spanCtx
	}

	return ctx, span, func() {
		if span != nil {
			span.End()
		}
	}
}
