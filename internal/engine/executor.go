package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/logger"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

// Execute runs the execution plan and returns step results in plan order.
func Execute(execCtx *ExecutionContext, plan *ExecutionPlan) ([]model.StepResult, error) {
	if execCtx == nil {
		return nil, streamyerrors.NewExecutionError("", fmt.Errorf("execution context is nil"))
	}
	if execCtx.Config == nil {
		return nil, streamyerrors.NewExecutionError("", fmt.Errorf("execution context config is nil"))
	}
	if plan == nil {
		return nil, streamyerrors.NewExecutionError("", fmt.Errorf("execution plan is nil"))
	}

	baseCtx := execCtx.Context
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	ctx, cancel := context.WithCancel(baseCtx)
	defer cancel()

	timeoutDuration := time.Duration(execCtx.Config.Settings.Timeout)
	if timeoutDuration > 0 {
		timeoutDuration = timeoutDuration * time.Second
	}

	stepLookup := make(map[string]*config.Step, len(execCtx.Config.Steps))
	for i := range execCtx.Config.Steps {
		step := &execCtx.Config.Steps[i]
		stepLookup[step.ID] = step
	}

	if execCtx.Results == nil {
		execCtx.Results = make(map[string]*model.StepResult)
	}

	var resultsMu sync.Mutex
	var allResults []model.StepResult
	var firstErr error

	for _, level := range plan.Levels {
		levelResults := make([]model.StepResult, len(level.StepIDs))
		var levelErr error
		var once sync.Once
		var wg sync.WaitGroup

		for idx, stepID := range level.StepIDs {
			step, ok := stepLookup[stepID]
			if !ok {
				return allResults, streamyerrors.NewExecutionError(stepID, fmt.Errorf("step not found"))
			}

			wg.Add(1)
			go func(idx int, step *config.Step) {
				defer wg.Done()

				res, err := executeStep(ctx, execCtx, step, timeoutDuration)
				if res != nil {
					levelResults[idx] = *res
					resultsMu.Lock()
					execCtx.Results[step.ID] = res
					resultsMu.Unlock()
				}

				if err != nil {
					once.Do(func() {
						levelErr = err
						if !execCtx.ContinueOnError {
							cancel()
						}
					})
				}
			}(idx, step)
		}

		wg.Wait()

		if levelErr != nil {
			for _, res := range levelResults {
				if res.StepID != "" {
					allResults = append(allResults, res)
				}
			}
			if firstErr == nil {
				firstErr = levelErr
			}
			if !execCtx.ContinueOnError {
				return allResults, levelErr
			}
			levelErr = nil
			continue
		}

		allResults = append(allResults, levelResults...)
	}

	return allResults, firstErr
}

func executeStep(ctx context.Context, execCtx *ExecutionContext, step *config.Step, timeout time.Duration) (*model.StepResult, error) {
	if ctx.Err() != nil {
		return nil, streamyerrors.NewExecutionError(step.ID, ctx.Err())
	}

	stepCtx := ctx
	var cancel context.CancelFunc
	if timeout > 0 {
		stepCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	if execCtx.WorkerPool != nil {
		select {
		case execCtx.WorkerPool <- struct{}{}:
			defer func() { <-execCtx.WorkerPool }()
		case <-stepCtx.Done():
			return timeoutResult(step.ID, stepCtx.Err())
		}
	}

	impl, err := execCtx.Registry.Get(step.Type)
	if err != nil {
		return nil, err
	}

	start := time.Now()

	// First evaluate the step
	evalResult, err := impl.Evaluate(stepCtx, step)
	if err != nil {
		// If we got an evaluation result despite the error (e.g., timeout), create a failed result
		if evalResult != nil {
			return &model.StepResult{
				StepID:    evalResult.StepID,
				Status:    model.StatusFailed,
				Message:   fmt.Sprintf("Evaluation failed: %v", err),
				Duration:  time.Since(start),
				Timestamp: time.Now(),
				Error:     err,
			}, fmt.Errorf("evaluation failed for step %s: %w", step.ID, err)
		}
		return nil, fmt.Errorf("evaluation failed for step %s: %w", step.ID, err)
	}

	var result *model.StepResult
	if execCtx.DryRun {
		// For dry-run, create a result based on evaluation
		if evalResult.RequiresAction {
			result = &model.StepResult{
				StepID:    evalResult.StepID,
				Status:    model.StatusWouldUpdate,
				Message:   evalResult.Message,
				Duration:  time.Since(start),
				Timestamp: time.Now(),
			}
		} else {
			result = &model.StepResult{
				StepID:    evalResult.StepID,
				Status:    model.StatusSkipped,
				Message:   evalResult.Message,
				Duration:  time.Since(start),
				Timestamp: time.Now(),
			}
		}
	} else {
		// For apply mode, only call Apply() if action is required
		if evalResult.RequiresAction {
			result, err = impl.Apply(stepCtx, evalResult, step)
		} else {
			// Skip the step since it's already satisfied
			result = &model.StepResult{
				StepID:    evalResult.StepID,
				Status:    model.StatusSkipped,
				Message:   evalResult.Message,
				Duration:  time.Since(start),
				Timestamp: time.Now(),
			}
		}
	}
	duration := time.Since(start)

	if result == nil {
		result = &model.StepResult{StepID: step.ID}
	}
	if result.StepID == "" {
		result.StepID = step.ID
	}
	result.Duration = duration
	if result.Timestamp.IsZero() {
		result.Timestamp = time.Now()
	}

	if err != nil {
		return finalizeFailure(result, stepCtx, step.ID, err)
	}

	if result.Status == "" {
		if execCtx.DryRun {
			result.Status = model.StatusSkipped
			if result.Message == "" {
				result.Message = "dry-run"
			}
		} else {
			result.Status = model.StatusSuccess
			if result.Message == "" {
				result.Message = "completed"
			}
		}
	}

	return result, nil
}

func finalizeFailure(result *model.StepResult, stepCtx context.Context, stepID string, err error) (*model.StepResult, error) {
	if result.Status == "" {
		result.Status = model.StatusFailed
	}
	if result.Error == nil {
		result.Error = err
	}
	if result.Message == "" {
		result.Message = err.Error()
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(stepCtx.Err(), context.DeadlineExceeded) {
		result.Message = "timeout exceeded"
	}

	return result, streamyerrors.NewExecutionError(stepID, err)
}

func timeoutResult(stepID string, err error) (*model.StepResult, error) {
	if err == nil {
		err = context.DeadlineExceeded
	}
	res := &model.StepResult{
		StepID:  stepID,
		Status:  model.StatusFailed,
		Message: "timeout exceeded",
		Error:   err,
	}
	return res, streamyerrors.NewExecutionError(stepID, err)
}

// Executor handles verification and apply operations.
type Executor struct {
	logger *logger.Logger
}

// NewExecutor creates a new executor instance.
func NewExecutor(log *logger.Logger) *Executor {
	return &Executor{
		logger: log,
	}
}

// VerifySteps performs verification on all steps and returns a summary
func (e *Executor) VerifySteps(ctx *ExecutionContext, steps []config.Step, defaultTimeout time.Duration) (*model.VerificationSummary, error) {
	start := time.Now()

	stepIndex := make(map[string]*config.Step, len(steps))
	enabledSteps := 0
	for i := range steps {
		if !steps[i].Enabled {
			continue
		}
		step := &steps[i]
		stepIndex[step.ID] = step
		enabledSteps++
	}

	graph, err := BuildDAG(steps)
	if err != nil {
		return nil, err
	}

	summary := &model.VerificationSummary{
		TotalSteps: enabledSteps,
		Results:    make([]*model.VerificationResult, 0, enabledSteps),
	}

	if defaultTimeout <= 0 {
		defaultTimeout = 30 * time.Second
	}

	resultsByID := make(map[string]*model.VerificationResult, enabledSteps)

	for _, level := range graph.Levels {
		for _, stepID := range level {
			step, ok := stepIndex[stepID]
			if !ok {
				continue
			}

			if ctx.Context.Err() != nil {
				return summary, ctx.Context.Err()
			}

			unsatisfied := make([]string, 0, len(step.DependsOn))
			for _, depID := range step.DependsOn {
				depResult, exists := resultsByID[depID]
				if !exists {
					continue
				}
				if depResult == nil || depResult.Status != model.StatusSatisfied {
					status := model.StatusUnknown
					if depResult != nil {
						status = depResult.Status
					}
					unsatisfied = append(unsatisfied, fmt.Sprintf("%s (%s)", depID, status))
				}
			}

			if len(unsatisfied) > 0 {
				msg := fmt.Sprintf("blocked: dependencies not satisfied: %s", strings.Join(unsatisfied, ", "))
				err := fmt.Errorf("dependencies not satisfied: %s", strings.Join(unsatisfied, ", "))
				result := &model.VerificationResult{
					StepID:    step.ID,
					Status:    model.StatusBlocked,
					Message:   msg,
					Error:     err,
					Duration:  0,
					Timestamp: time.Now(),
				}
				summary.Results = append(summary.Results, result)
				summary.Blocked++
				resultsByID[step.ID] = result
				continue
			}

			p, err := ctx.Registry.Get(step.Type)
			if err != nil {
				result := &model.VerificationResult{
					StepID:    step.ID,
					Status:    model.StatusBlocked,
					Message:   fmt.Sprintf("plugin not found for type %s", step.Type),
					Error:     err,
					Duration:  0,
					Timestamp: time.Now(),
				}
				summary.Results = append(summary.Results, result)
				summary.Blocked++
				resultsByID[step.ID] = result
				continue
			}

			timeout := defaultTimeout
			if step.VerifyTimeout > 0 {
				timeout = time.Duration(step.VerifyTimeout) * time.Second
			}

			stepStart := time.Now()
			stepCtx, cancel := context.WithTimeout(ctx.Context, timeout)

			// Use new Evaluate() method for verification
			evalResult, verifyErr := p.Evaluate(stepCtx, step)
			cancel()

			if verifyErr != nil {
				summary.Duration = time.Since(start)

				// Handle new structured error types
				var pluginErr plugin.PluginError
				if errors.As(verifyErr, &pluginErr) {
					switch pluginErr.(type) {
					case *plugin.ValidationError:
						// Configuration error - always fatal
						return summary, verifyErr
					case *plugin.ExecutionError:
						// Execution error - always fatal in verify mode
						return summary, verifyErr
					case *plugin.StateError:
						// State detection error - treat as Unknown status
						result := &model.VerificationResult{
							StepID:    step.ID,
							Status:    model.StatusUnknown,
							Message:   pluginErr.Error(),
							Error:     pluginErr.Unwrap(),
							Duration:  time.Since(stepStart),
							Timestamp: time.Now(),
						}
						summary.Results = append(summary.Results, result)
						summary.Unknown++
						resultsByID[step.ID] = result
						continue
					}
				}

				// Handle legacy error types
				var validationErr *streamyerrors.ValidationError
				if errors.As(verifyErr, &validationErr) {
					return summary, verifyErr
				}

				var parseErr *streamyerrors.ParseError
				if errors.As(verifyErr, &parseErr) {
					return summary, verifyErr
				}

				var execErr *streamyerrors.ExecutionError
				if errors.As(verifyErr, &execErr) {
					return summary, verifyErr
				}

				var legacyPluginErr *streamyerrors.PluginError
				if errors.As(verifyErr, &legacyPluginErr) {
					return summary, verifyErr
				}

				return summary, streamyerrors.NewExecutionError(step.ID, verifyErr)
			}

			// Convert EvaluationResult to VerificationResult for compatibility
			result := &model.VerificationResult{
				StepID:    evalResult.StepID,
				Status:    evalResult.CurrentState,
				Message:   evalResult.Message,
				Details:   evalResult.Diff, // Use Diff as Details for compatibility
				Duration:  time.Since(stepStart),
				Timestamp: time.Now(),
			}

			summary.Results = append(summary.Results, result)
			resultsByID[step.ID] = result

			switch result.Status {
			case model.StatusSatisfied:
				summary.Satisfied++
			case model.StatusMissing:
				summary.Missing++
			case model.StatusDrifted:
				summary.Drifted++
			case model.StatusBlocked:
				summary.Blocked++
			case model.StatusUnknown:
				summary.Unknown++
			}
		}
	}

	summary.Duration = time.Since(start)
	return summary, nil
}
