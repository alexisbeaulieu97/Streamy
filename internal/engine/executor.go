package engine

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/alexisbeaulieu97/streamy/internal/config"
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

	impl, err := plugin.GetPlugin(step.Type)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	var result *model.StepResult
	if execCtx.DryRun {
		result, err = impl.DryRun(stepCtx, step)
	} else {
		result, err = impl.Apply(stepCtx, step)
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
