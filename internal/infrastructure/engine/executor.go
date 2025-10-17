package engine

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/infrastructure/logging"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

// Executor executes pipeline plans using registered plugins.
type Executor struct {
	registry    ports.PluginRegistry
	logger      ports.Logger
	metrics     ports.MetricsCollector
	tracer      ports.Tracer
	events      ports.EventPublisher
	parallelism int
}

// ExecutorOption configures an executor instance.
type ExecutorOption func(*Executor)

// WithExecutorLogger injects a logger into the executor.
func WithExecutorLogger(logger ports.Logger) ExecutorOption {
	return func(e *Executor) {
		e.logger = logger
	}
}

// WithExecutorMetrics injects a metrics collector.
func WithExecutorMetrics(metrics ports.MetricsCollector) ExecutorOption {
	return func(e *Executor) {
		e.metrics = metrics
	}
}

// WithExecutorTracer injects a tracer.
func WithExecutorTracer(tracer ports.Tracer) ExecutorOption {
	return func(e *Executor) {
		e.tracer = tracer
	}
}

// WithExecutorEvents injects an event publisher.
func WithExecutorEvents(events ports.EventPublisher) ExecutorOption {
	return func(e *Executor) {
		e.events = events
	}
}

// WithExecutorParallelism overrides per-level parallelism.
func WithExecutorParallelism(parallelism int) ExecutorOption {
	return func(e *Executor) {
		e.parallelism = parallelism
	}
}

// NewExecutor constructs a PluginExecutor implementation.
func NewExecutor(registry ports.PluginRegistry, opts ...ExecutorOption) *Executor {
	exec := &Executor{
		registry: registry,
		logger:   logging.NewNoOpLogger(),
	}
	for _, opt := range opts {
		opt(exec)
	}
	return exec
}

// Execute runs the supplied plan using registered plugins.
func (e *Executor) Execute(ctx context.Context, plan *domainpipeline.ExecutionPlan, pipeline *domainpipeline.Pipeline) ([]domainpipeline.StepResult, error) {
	if plan == nil {
		return nil, &domainpipeline.DomainError{Code: domainpipeline.ErrCodeInternal, Message: "execution plan is nil"}
	}
	if pipeline == nil {
		return nil, &domainpipeline.DomainError{Code: domainpipeline.ErrCodeInternal, Message: "pipeline is nil"}
	}
	if ctx == nil {
		ctx = context.Background()
	}

	settings := pipeline.EffectiveSettings()
	continueOnError := settings.ContinueOnError
	dryRun := settings.DryRun

	results := make([]domainpipeline.StepResult, 0, len(plan.Levels))
	var firstErr error

	for _, level := range plan.Levels {
		levelResults := make([]domainpipeline.StepResult, len(level.StepIDs))
		var levelErr error
		var levelErrOnce sync.Once
		var wg sync.WaitGroup

		parallelism := settings.Parallel
		if e.parallelism > 0 {
			parallelism = e.parallelism
		}
		if parallelism <= 0 {
			parallelism = len(level.StepIDs)
		}

		sem := make(chan struct{}, parallelism)

		for idx, stepID := range level.StepIDs {
			step, err := pipeline.GetStep(stepID)
			if err != nil {
				return results, err
			}
			wg.Add(1)
			go func(index int, st domainpipeline.Step) {
				defer wg.Done()
				select {
				case sem <- struct{}{}:
					defer func() { <-sem }()
				case <-ctx.Done():
					levelErrOnce.Do(func() {
						levelErr = &domainpipeline.DomainError{
							Code:    domainpipeline.ErrCodeCancelled,
							Message: "execution cancelled",
							Cause:   ctx.Err(),
						}
					})
					return
				}
				result, err := e.executeStep(ctx, pipeline.Name, st, dryRun)
				levelResults[index] = result
				if err != nil {
					levelErrOnce.Do(func() {
						levelErr = err
					})
				}
			}(idx, *step)
		}

		wg.Wait()
		results = append(results, levelResults...)

		if levelErr != nil {
			if firstErr == nil {
				firstErr = levelErr
			}
			if !continueOnError {
				return results, levelErr
			}
		}
	}

	return results, firstErr
}

// Verify evaluates each step without applying changes.
func (e *Executor) Verify(ctx context.Context, pipeline *domainpipeline.Pipeline) ([]domainpipeline.VerificationResult, error) {
	if pipeline == nil {
		return nil, &domainpipeline.DomainError{Code: domainpipeline.ErrCodeInternal, Message: "pipeline is nil"}
	}
	if ctx == nil {
		ctx = context.Background()
	}

	results := make([]domainpipeline.VerificationResult, 0, len(pipeline.Steps))
	var firstErr error

	for _, step := range pipeline.Steps {
		if err := ctx.Err(); err != nil {
			dErr := &domainpipeline.DomainError{Code: domainpipeline.ErrCodeCancelled, Message: "verification cancelled", Cause: err}
			if firstErr == nil {
				firstErr = dErr
			}
			break
		}

		result, err := e.verifyStep(ctx, step)
		results = append(results, result)

		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return results, firstErr
}

// executeStep evaluates and applies a single step.
func (e *Executor) executeStep(ctx context.Context, pipelineName string, step domainpipeline.Step, dryRun bool) (domainpipeline.StepResult, error) {
	logger := e.logger
	if ctx.Err() != nil {
		return domainpipeline.StepResult{
			StepID: step.ID,
			Status: domainpipeline.StatusFailure,
			Error: &domainpipeline.DomainError{
				Code:    domainpipeline.ErrCodeCancelled,
				Message: "execution cancelled",
				Cause:   ctx.Err(),
			},
		}, ctx.Err()
	}

	pluginType := domainplugin.Type(step.Type)
	handler, err := e.registry.Get(pluginType)
	if err != nil {
		return domainpipeline.StepResult{
			StepID: step.ID,
			Status: domainpipeline.StatusFailure,
			Error:  toDomainError(err),
		}, err
	}

	if logger != nil {
		logger.Debug(ctx, "executing step", "step_id", step.ID, "step_type", pluginType)
	}

	start := time.Now()

	var span ports.Span
	if e.tracer != nil {
		var spanCtx context.Context
		spanCtx, span = e.tracer.StartSpan(ctx, "pipeline.step", "step_id", step.ID, "step_type", string(pluginType))
		if spanCtx != nil {
			ctx = spanCtx
		}
	}

	publishEvent(ctx, e.events, logger, ports.EventStepStarted, map[string]interface{}{
		"pipeline":  pipelineName,
		"step_id":   step.ID,
		"step_type": step.Type,
		"dry_run":   dryRun,
		"timestamp": start.UTC(),
	})

	eval, err := handler.Evaluate(ctx, step)
	if err != nil {
		result := domainpipeline.StepResult{
			StepID:   step.ID,
			Status:   domainpipeline.StatusFailure,
			Error:    toDomainError(err),
			Duration: int(time.Since(start).Milliseconds()),
		}
		e.recordMetrics(ctx, result.StepID, result.Status, time.Since(start))
		if span != nil {
			span.SetStatus(ports.SpanStatusError, err.Error())
		}
		if logger != nil {
			logger.Error(ctx, "step evaluation failed", "step_id", step.ID, "error", err)
		}
		publishEvent(ctx, e.events, logger, ports.EventStepFailed, map[string]interface{}{
			"pipeline":  pipelineName,
			"step_id":   step.ID,
			"step_type": step.Type,
			"error":     err,
		})
		return result, err
	}

	if dryRun {
		result := dryRunResult(step, eval, time.Since(start))
		e.recordMetrics(ctx, result.StepID, result.Status, time.Since(start))
		if logger != nil {
			logger.Info(ctx, "step dry-run evaluation complete", "step_id", step.ID, "requires_action", eval != nil && eval.RequiresAction)
		}
		if span != nil {
			span.SetAttribute("step_status", string(result.Status))
			span.SetStatus(ports.SpanStatusOK, "dry-run")
		}
		publishEvent(ctx, e.events, logger, ports.EventStepCompleted, map[string]interface{}{
			"pipeline":  pipelineName,
			"step_id":   step.ID,
			"step_type": step.Type,
			"dry_run":   true,
			"changed":   result.Changed,
			"duration":  result.Duration,
		})
		return result, nil
	}

	if eval != nil && !eval.RequiresAction {
		result := domainpipeline.StepResult{
			StepID:   step.ID,
			Status:   domainpipeline.StatusAlreadySatisfied,
			Message:  "step already satisfied",
			Duration: int(time.Since(start).Milliseconds()),
		}
		e.recordMetrics(ctx, result.StepID, result.Status, time.Since(start))
		if logger != nil {
			logger.Info(ctx, "step already satisfied", "step_id", step.ID)
		}
		if span != nil {
			span.SetAttribute("step_status", string(result.Status))
			span.SetStatus(ports.SpanStatusOK, "already_satisfied")
		}
		publishEvent(ctx, e.events, logger, ports.EventStepSkipped, map[string]interface{}{
			"pipeline":  pipelineName,
			"step_id":   step.ID,
			"step_type": step.Type,
			"duration":  result.Duration,
		})
		return result, nil
	}

	result, err := handler.Apply(ctx, eval, step)
	if result == nil {
		result = &domainpipeline.StepResult{StepID: step.ID}
	}
	result.Duration = int(time.Since(start).Milliseconds())

	if err != nil {
		result.Status = domainpipeline.StatusFailure
		result.Error = toDomainError(err)
		e.recordMetrics(ctx, result.StepID, result.Status, time.Since(start))
		if span != nil {
			span.SetStatus(ports.SpanStatusError, err.Error())
		}
		if logger != nil {
			logger.Error(ctx, "step execution failed", "step_id", step.ID, "error", err)
		}
		publishEvent(ctx, e.events, logger, ports.EventStepFailed, map[string]interface{}{
			"pipeline":  pipelineName,
			"step_id":   step.ID,
			"step_type": step.Type,
			"duration":  result.Duration,
			"error":     err,
		})
		return *result, err
	}

	if logger != nil {
		logger.Info(ctx, "step executed", "step_id", step.ID, "status", result.Status)
	}
	e.recordMetrics(ctx, result.StepID, result.Status, time.Since(start))
	if span != nil {
		span.SetAttribute("step_status", string(result.Status))
		span.SetStatus(ports.SpanStatusOK, "success")
	}
	publishEvent(ctx, e.events, logger, ports.EventStepCompleted, map[string]interface{}{
		"pipeline":  pipelineName,
		"step_id":   step.ID,
		"step_type": step.Type,
		"duration":  result.Duration,
		"changed":   result.Changed,
	})

	return *result, nil
}

func (e *Executor) verifyStep(ctx context.Context, step domainpipeline.Step) (domainpipeline.VerificationResult, error) {
	pluginType := domainplugin.Type(step.Type)
	handler, err := e.registry.Get(pluginType)
	if err != nil {
		return domainpipeline.VerificationResult{
			StepID:  step.ID,
			Type:    string(pluginType),
			Status:  domainpipeline.VerificationUnknown,
			Message: err.Error(),
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

func (e *Executor) recordMetrics(ctx context.Context, stepID string, status domainpipeline.ResultStatus, duration time.Duration) {
	if e.metrics == nil {
		return
	}
	labels := map[string]string{
		"step_id": stepID,
		"status":  string(status),
	}
	e.metrics.IncCounter(ctx, "streamy_step_executions_total", labels)
	e.metrics.ObserveHistogram(ctx, "streamy_step_execution_duration_seconds", duration.Seconds(), labels)
}

func dryRunResult(step domainpipeline.Step, eval *domainpipeline.EvaluationResult, elapsed time.Duration) domainpipeline.StepResult {
	status := domainpipeline.StatusSkipped
	message := "no changes required"
	changed := false
	if eval != nil && eval.RequiresAction {
		status = domainpipeline.StatusSuccess
		message = "dry-run: changes would be applied"
		changed = true
	}
	return domainpipeline.StepResult{
		StepID:   step.ID,
		Status:   status,
		Message:  message,
		Changed:  changed,
		Duration: int(elapsed.Milliseconds()),
	}
}

func toDomainError(err error) *domainpipeline.DomainError {
	if err == nil {
		return nil
	}
	var derr *domainpipeline.DomainError
	if errors.As(err, &derr) {
		return derr
	}
	return &domainpipeline.DomainError{
		Code:    domainpipeline.ErrCodeExecution,
		Message: err.Error(),
		Cause:   err,
	}
}

func categorizeVerificationError(err *domainpipeline.DomainError) string {
	if err == nil {
		return ""
	}
	switch err.Code {
	case domainpipeline.ErrCodeNotFound, domainpipeline.ErrCodeMissing:
		return "missing"
	case domainpipeline.ErrCodeDependency:
		return "blocked"
	case domainpipeline.ErrCodeValidation:
		return "unknown"
	}
	msg := strings.ToLower(err.Message)
	if strings.Contains(msg, "no such file") || strings.Contains(msg, "not found") {
		return "missing"
	}
	return ""
}

type executorEvent struct {
	eventType string
	payload   interface{}
}

func (e executorEvent) EventType() string    { return e.eventType }
func (e executorEvent) Payload() interface{} { return e.payload }

func publishEvent(ctx context.Context, publisher ports.EventPublisher, logger ports.Logger, eventType string, payload map[string]interface{}) {
	if publisher == nil {
		return
	}
	event := executorEvent{eventType: eventType, payload: payload}
	if err := publisher.Publish(ctx, event); err != nil && logger != nil {
		logger.Warn(ctx, "failed to publish executor event", "event_type", eventType, "error", err)
	}
}

var _ ports.PluginExecutor = (*Executor)(nil)
