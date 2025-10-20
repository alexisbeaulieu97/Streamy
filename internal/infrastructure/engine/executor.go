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
	if err := validateExecuteInputs(ctx, plan, pipeline); err != nil {
		return nil, err
	}

	settings := pipeline.EffectiveSettings()
	stepTimeout := computeStepTimeout(settings.Timeout)

	results := make([]domainpipeline.StepResult, 0, len(plan.Levels))

	var firstErr error

	for _, level := range plan.Levels {
		parallelism := e.resolveParallelism(settings.Parallel, len(level.StepIDs))

		levelResults, levelErr := e.executeLevel(ctx, pipeline, level, settings.DryRun, stepTimeout, parallelism)
		if len(levelResults) > 0 {
			results = append(results, levelResults...)
		}

		if levelErr != nil && firstErr == nil {
			firstErr = levelErr
		}

		if levelErr != nil && !settings.ContinueOnError {
			return results, levelErr
		}

		if err := ctx.Err(); err != nil {
			return results, e.newPipelineCancelledError(err, pipeline.Name)
		}
	}

	return results, firstErr
}

func validateExecuteInputs(ctx context.Context, plan *domainpipeline.ExecutionPlan, pipeline *domainpipeline.Pipeline) error {
	if plan == nil {
		return domainpipeline.NewInternalError("execution plan is nil", nil, nil)
	}

	if pipeline == nil {
		return domainpipeline.NewInternalError("pipeline is nil", nil, nil)
	}

	if ctx == nil {
		return domainpipeline.NewInternalError("execution context is required", nil, nil)
	}

	return nil
}

func computeStepTimeout(timeoutSeconds int) time.Duration {
	if timeoutSeconds <= 0 {
		return 0
	}

	return time.Duration(timeoutSeconds) * time.Second
}

func (e *Executor) resolveParallelism(requested, levelSize int) int {
	if e.parallelism > 0 {
		requested = e.parallelism
	}

	if requested <= 0 {
		return levelSize
	}

	return requested
}

func (e *Executor) executeLevel(ctx context.Context, pipeline *domainpipeline.Pipeline, level domainpipeline.ExecutionLevel, dryRun bool, stepTimeout time.Duration, parallelism int) ([]domainpipeline.StepResult, error) {
	if len(level.StepIDs) == 0 {
		return nil, nil
	}

	results := make([]domainpipeline.StepResult, len(level.StepIDs))

	var (
		levelErr error
		errOnce  sync.Once
		wg       sync.WaitGroup
	)

	sem := make(chan struct{}, parallelism)
	executed := 0

	for idx, stepID := range level.StepIDs {
		if err := ctx.Err(); err != nil {
			errOnce.Do(func() {
				levelErr = e.newPipelineCancelledError(err, pipeline.Name)
			})

			break
		}

		step, err := pipeline.GetStep(stepID)
		if err != nil {
			return nil, domainpipeline.NewDomainError(
				domainpipeline.ErrCodeNotFound,
				"unable to retrieve step definition",
				err,
				map[string]interface{}{"pipeline": pipeline.Name, "step_id": stepID},
			)
		}

		executed++

		wg.Add(1)

		go func(index int, st domainpipeline.Step) {
			defer wg.Done()

			release, slotErr := acquireSlot(ctx, sem, pipeline.Name, st.ID)
			if slotErr != nil {
				errOnce.Do(func() {
					levelErr = slotErr
				})

				failure := domainpipeline.StepResult{
					StepID: st.ID,
					Status: domainpipeline.StatusFailure,
				}

				if derr, ok := slotErr.(*domainpipeline.DomainError); ok {
					failure.Error = derr
				}

				results[index] = failure

				return
			}
			defer release()

			result, execErr := e.executeStep(ctx, pipeline.Name, st, dryRun, stepTimeout)
			results[index] = result

			if execErr != nil {
				errOnce.Do(func() {
					levelErr = execErr
				})
			}
		}(idx, *step)
	}

	wg.Wait()

	if executed == 0 {
		return nil, levelErr
	}

	return results[:executed], levelErr
}

func acquireSlot(ctx context.Context, sem chan struct{}, pipelineName, stepID string) (func(), error) {
	select {
	case sem <- struct{}{}:
		return func() { <-sem }, nil
	case <-ctx.Done():
		return nil, domainpipeline.NewDomainError(
			domainpipeline.ErrCodeCancelled,
			"execution cancelled",
			ctx.Err(),
			map[string]interface{}{
				"pipeline": pipelineName,
				"step_id":  stepID,
			},
		)
	}
}

func (e *Executor) newPipelineCancelledError(err error, pipelineName string) error {
	return domainpipeline.NewDomainError(
		domainpipeline.ErrCodeCancelled,
		"execution cancelled",
		err,
		map[string]interface{}{"pipeline": pipelineName},
	)
}

func checkStepContext(ctx context.Context, pipelineName, stepID string) (domainpipeline.StepResult, bool, error) {
	if err := ctx.Err(); err != nil {
		derr := domainpipeline.NewDomainError(
			domainpipeline.ErrCodeCancelled,
			"execution cancelled",
			err,
			map[string]interface{}{"pipeline": pipelineName, "step_id": stepID},
		)

		return domainpipeline.StepResult{
			StepID: stepID,
			Status: domainpipeline.StatusFailure,
			Error:  derr,
		}, true, derr
	}

	return domainpipeline.StepResult{}, false, nil
}

func withStepTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, func() {}
	}

	stepCtx, cancel := context.WithTimeout(ctx, timeout)

	return stepCtx, cancel
}

func (e *Executor) handleRegistryLookupError(pipelineName string, step domainpipeline.Step, pluginType domainplugin.Type, err error) (domainpipeline.StepResult, error) {
	derr := toDomainError(err, step.ID, pluginType).WithContext(map[string]interface{}{
		"pipeline": pipelineName,
		"phase":    "registry_lookup",
	})

	return domainpipeline.StepResult{
		StepID: step.ID,
		Status: domainpipeline.StatusFailure,
		Error:  derr,
	}, derr
}

func (e *Executor) startStepSpan(ctx context.Context, stepID string, pluginType domainplugin.Type) (context.Context, ports.Span, func()) {
	if e.tracer == nil {
		return ctx, nil, func() {}
	}

	spanCtx, span := e.tracer.StartSpan(ctx, "pipeline.step", "step_id", stepID, "step_type", string(pluginType))
	if spanCtx != nil {
		ctx = spanCtx
	}

	return ctx, span, func() {
		if span != nil {
			span.End()
		}
	}
}

func (e *Executor) publishStepStarted(ctx context.Context, pipelineName string, step domainpipeline.Step, dryRun bool, startedAt time.Time) {
	publishEvent(ctx, e.events, e.logger, ports.EventStepStarted, map[string]interface{}{
		"pipeline":  pipelineName,
		"step_id":   step.ID,
		"step_type": step.Type,
		"dry_run":   dryRun,
		"timestamp": startedAt.UTC(),
	})
}

func (e *Executor) handleEvaluateError(ctx context.Context, pipelineName string, step domainpipeline.Step, pluginType domainplugin.Type, startedAt time.Time, span ports.Span, evalErr error) (domainpipeline.StepResult, error) {
	derr := toDomainError(evalErr, step.ID, pluginType).WithContext(map[string]interface{}{
		"pipeline": pipelineName,
		"phase":    "evaluate",
	})

	elapsed := time.Since(startedAt)
	result := domainpipeline.StepResult{
		StepID:   step.ID,
		Status:   domainpipeline.StatusFailure,
		Error:    derr,
		Duration: elapsed,
	}

	e.recordMetrics(ctx, pipelineName, result.StepID, step.Type, result.Status, elapsed)
	setSpanAttribute(span, "step_status", string(result.Status))
	setSpanStatus(span, ports.SpanStatusError, evalErr.Error())
	e.logStepError(ctx, "step evaluation failed", "step_id", step.ID, "error", evalErr)
	e.publishStepFailed(ctx, pipelineName, step, result.Duration, derr)

	return result, derr
}

func (e *Executor) handleDryRun(ctx context.Context, pipelineName string, step domainpipeline.Step, eval *domainpipeline.EvaluationResult, startedAt time.Time, span ports.Span) (domainpipeline.StepResult, error) {
	elapsed := time.Since(startedAt)
	result := dryRunResult(step, eval, elapsed)

	e.recordMetrics(ctx, pipelineName, result.StepID, step.Type, result.Status, elapsed)
	e.logStepInfo(ctx, "step dry-run evaluation complete", "step_id", step.ID, "requires_action", eval != nil && eval.RequiresAction)
	setSpanAttribute(span, "step_status", string(result.Status))
	setSpanStatus(span, ports.SpanStatusOK, "dry-run")
	e.publishStepCompleted(ctx, pipelineName, step, result.Duration, result.Changed, true)

	return result, nil
}

func (e *Executor) handleAlreadySatisfied(ctx context.Context, pipelineName string, step domainpipeline.Step, startedAt time.Time, span ports.Span) (domainpipeline.StepResult, error) {
	elapsed := time.Since(startedAt)
	result := domainpipeline.StepResult{
		StepID:   step.ID,
		Status:   domainpipeline.StatusAlreadySatisfied,
		Message:  "step already satisfied",
		Duration: elapsed,
	}

	e.recordMetrics(ctx, pipelineName, result.StepID, step.Type, result.Status, elapsed)
	e.logStepInfo(ctx, "step already satisfied", "step_id", step.ID)
	setSpanAttribute(span, "step_status", string(result.Status))
	setSpanStatus(span, ports.SpanStatusOK, "already_satisfied")
	e.publishStepSkipped(ctx, pipelineName, step, result.Duration)

	return result, nil
}

func (e *Executor) handleApply(ctx context.Context, pipelineName string, step domainpipeline.Step, pluginType domainplugin.Type, handler ports.Plugin, eval *domainpipeline.EvaluationResult, startedAt time.Time, span ports.Span) (domainpipeline.StepResult, error) {
	result, err := handler.Apply(ctx, eval, step)
	if result == nil {
		result = &domainpipeline.StepResult{StepID: step.ID}
	}

	elapsed := time.Since(startedAt)
	result.Duration = elapsed

	if err != nil {
		result.Status = domainpipeline.StatusFailure
		result.Error = toDomainError(err, step.ID, pluginType).WithContext(map[string]interface{}{
			"pipeline": pipelineName,
			"phase":    "apply",
		})

		e.recordMetrics(ctx, pipelineName, result.StepID, step.Type, result.Status, elapsed)
		setSpanAttribute(span, "step_status", string(result.Status))
		setSpanStatus(span, ports.SpanStatusError, err.Error())
		e.logStepError(ctx, "step execution failed", "step_id", step.ID, "error", err)
		e.publishStepFailed(ctx, pipelineName, step, result.Duration, result.Error)

		return *result, result.Error
	}

	e.logStepInfo(ctx, "step executed", "step_id", step.ID, "status", result.Status)
	e.recordMetrics(ctx, pipelineName, result.StepID, step.Type, result.Status, elapsed)
	setSpanAttribute(span, "step_status", string(result.Status))
	setSpanStatus(span, ports.SpanStatusOK, "success")
	e.publishStepCompleted(ctx, pipelineName, step, result.Duration, result.Changed, false)

	return *result, nil
}

func (e *Executor) publishStepFailed(ctx context.Context, pipelineName string, step domainpipeline.Step, duration time.Duration, err error) {
	publishEvent(ctx, e.events, e.logger, ports.EventStepFailed, map[string]interface{}{
		"pipeline":  pipelineName,
		"step_id":   step.ID,
		"step_type": step.Type,
		"duration":  duration.Milliseconds(),
		"error":     err,
	})
}

func (e *Executor) publishStepCompleted(ctx context.Context, pipelineName string, step domainpipeline.Step, duration time.Duration, changed, dryRun bool) {
	payload := map[string]interface{}{
		"pipeline":  pipelineName,
		"step_id":   step.ID,
		"step_type": step.Type,
		"duration":  duration.Milliseconds(),
		"changed":   changed,
	}

	if dryRun {
		payload["dry_run"] = true
	}

	publishEvent(ctx, e.events, e.logger, ports.EventStepCompleted, payload)
}

func (e *Executor) publishStepSkipped(ctx context.Context, pipelineName string, step domainpipeline.Step, duration time.Duration) {
	publishEvent(ctx, e.events, e.logger, ports.EventStepSkipped, map[string]interface{}{
		"pipeline":  pipelineName,
		"step_id":   step.ID,
		"step_type": step.Type,
		"duration":  duration.Milliseconds(),
	})
}

func (e *Executor) logStepDebug(ctx context.Context, msg string, keyvals ...interface{}) {
	if e.logger == nil {
		return
	}

	e.logger.Debug(ctx, msg, keyvals...)
}

func (e *Executor) logStepInfo(ctx context.Context, msg string, keyvals ...interface{}) {
	if e.logger == nil {
		return
	}

	e.logger.Info(ctx, msg, keyvals...)
}

func (e *Executor) logStepError(ctx context.Context, msg string, keyvals ...interface{}) {
	if e.logger == nil {
		return
	}

	e.logger.Error(ctx, msg, keyvals...)
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

// Verify evaluates each step without applying changes.
func (e *Executor) Verify(ctx context.Context, pipeline *domainpipeline.Pipeline) ([]domainpipeline.VerificationResult, error) {
	if pipeline == nil {
		return nil, domainpipeline.NewInternalError("pipeline is nil", nil, nil)
	}

	if ctx == nil {
		return nil, domainpipeline.NewInternalError("verification context is required", nil, nil)
	}

	settings := pipeline.EffectiveSettings()

	defaultTimeout := time.Duration(settings.Timeout) * time.Second
	if defaultTimeout <= 0 {
		defaultTimeout = 0
	}

	results := make([]domainpipeline.VerificationResult, 0, len(pipeline.Steps))

	var firstErr error

	for _, step := range pipeline.Steps {
		if err := ctx.Err(); err != nil {
			dErr := domainpipeline.NewDomainError(
				domainpipeline.ErrCodeCancelled,
				"verification cancelled",
				err,
				map[string]interface{}{"pipeline": pipeline.Name, "step_id": step.ID},
			)
			if firstErr == nil {
				firstErr = dErr
			}

			break
		}

		stepTimeout := defaultTimeout
		if step.VerifyTimeout > 0 {
			stepTimeout = time.Duration(step.VerifyTimeout) * time.Second
		}

		result, err := e.verifyStep(ctx, step, stepTimeout)
		results = append(results, result)

		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return results, firstErr
}

// executeStep evaluates and applies a single step.
func (e *Executor) executeStep(ctx context.Context, pipelineName string, step domainpipeline.Step, dryRun bool, timeout time.Duration) (domainpipeline.StepResult, error) {
	result, cancelled, err := checkStepContext(ctx, pipelineName, step.ID)
	if cancelled {
		return result, err
	}

	stepCtx, cancel := withStepTimeout(ctx, timeout)
	defer cancel()

	result, cancelled, err = checkStepContext(stepCtx, pipelineName, step.ID)
	if cancelled {
		return result, err
	}

	pluginType := domainplugin.Type(step.Type)

	handler, err := e.registry.Get(pluginType)
	if err != nil {
		return e.handleRegistryLookupError(pipelineName, step, pluginType, err)
	}

	e.logStepDebug(stepCtx, "executing step", "step_id", step.ID, "step_type", pluginType)

	start := time.Now()

	stepCtx, span, finishSpan := e.startStepSpan(stepCtx, step.ID, pluginType)
	defer finishSpan()

	e.publishStepStarted(stepCtx, pipelineName, step, dryRun, start)

	eval, err := handler.Evaluate(stepCtx, step)
	if err != nil {
		return e.handleEvaluateError(stepCtx, pipelineName, step, pluginType, start, span, err)
	}

	if dryRun {
		return e.handleDryRun(stepCtx, pipelineName, step, eval, start, span)
	}

	if eval != nil && !eval.RequiresAction {
		return e.handleAlreadySatisfied(stepCtx, pipelineName, step, start, span)
	}

	return e.handleApply(stepCtx, pipelineName, step, pluginType, handler, eval, start, span)
}

func (e *Executor) verifyStep(ctx context.Context, step domainpipeline.Step, timeout time.Duration) (domainpipeline.VerificationResult, error) {
	pluginType := domainplugin.Type(step.Type)

	handler, err := e.registry.Get(pluginType)
	if err != nil {
		derr := toDomainError(err, step.ID, pluginType)

		return domainpipeline.VerificationResult{
			StepID:  step.ID,
			Type:    string(pluginType),
			Status:  domainpipeline.VerificationUnknown,
			Message: derr.Error(),
		}, derr
	}

	stepCtx := ctx

	var cancel context.CancelFunc
	if timeout > 0 {
		stepCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	eval, err := handler.Evaluate(stepCtx, step)
	if err != nil {
		derr := toDomainError(err, step.ID, pluginType).WithContext(map[string]interface{}{
			"phase": "verify_evaluate",
		})

		details := map[string]interface{}{"step_id": step.ID, "plugin_type": string(pluginType)}
		if status := categorizeVerificationError(derr); status != "" {
			details["status"] = status
		}

		return domainpipeline.VerificationResult{
			StepID:  step.ID,
			Type:    string(pluginType),
			Status:  domainpipeline.VerificationFailed,
			Message: derr.Error(),
			Details: details,
		}, derr
	}

	if err := stepCtx.Err(); err != nil {
		derr := toDomainError(err, step.ID, pluginType).WithContext(map[string]interface{}{
			"phase": "verify_evaluate",
		})

		details := map[string]interface{}{"step_id": step.ID, "plugin_type": string(pluginType)}
		if status := categorizeVerificationError(derr); status != "" {
			details["status"] = status
		}

		return domainpipeline.VerificationResult{
			StepID:  step.ID,
			Type:    string(pluginType),
			Status:  domainpipeline.VerificationFailed,
			Message: derr.Error(),
			Details: details,
		}, derr
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

func (e *Executor) recordMetrics(ctx context.Context, pipelineName, stepID string, stepType domainpipeline.StepType, status domainpipeline.ResultStatus, duration time.Duration) {
	if e.metrics == nil {
		return
	}

	labels := map[string]string{
		"pipeline":  pipelineName,
		"step_id":   stepID,
		"step_type": string(stepType),
		"status":    string(status),
	}
	e.metrics.IncCounter(ctx, "streamy_step_executions_total", labels)
	e.metrics.ObserveHistogram(ctx, "streamy_step_execution_duration_seconds", duration.Seconds(), labels)

	if status == domainpipeline.StatusFailure {
		failureLabels := map[string]string{
			"pipeline":  pipelineName,
			"step_id":   stepID,
			"step_type": string(stepType),
		}
		e.metrics.IncCounter(ctx, "streamy_step_failures_total", failureLabels)
	}
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
		Duration: elapsed,
	}
}

func toDomainError(err error, stepID string, pluginType domainplugin.Type) *domainpipeline.DomainError {
	if err == nil {
		return nil
	}

	contextFields := map[string]interface{}{}
	if stepID != "" {
		contextFields["step_id"] = stepID
	}

	if pluginType != "" {
		contextFields["plugin_type"] = string(pluginType)
	}

	var derr *domainpipeline.DomainError
	if errors.As(err, &derr) {
		if len(contextFields) == 0 {
			return derr
		}

		return derr.WithContext(contextFields)
	}

	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return domainpipeline.NewTimeoutError("plugin operation timed out", err, contextFields)
	case errors.Is(err, context.Canceled):
		return domainpipeline.NewDomainError(domainpipeline.ErrCodeCancelled, "plugin operation cancelled", err, contextFields)
	default:
		return domainpipeline.NewExecutionError("plugin execution failed", err, contextFields)
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
	case domainpipeline.ErrCodeTimeout:
		return "timeout"
	case domainpipeline.ErrCodeCancelled:
		return "cancelled"
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
