package engine

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

func TestExecute_SequentialLevels(t *testing.T) {
	fp := &fakePlugin{}
	registry := plugin.NewPluginRegistry(plugin.DefaultConfig(), nil)
	require.NoError(t, registry.Register(fp))

	step1 := config.Step{ID: "step1", Type: "command", Enabled: true}
	require.NoError(t, step1.SetConfig(config.CommandStep{Command: "echo 1"}))
	step2 := config.Step{ID: "step2", Type: "command", Enabled: true, DependsOn: []string{"step1"}}
	require.NoError(t, step2.SetConfig(config.CommandStep{Command: "echo 2"}))
	cfg := &config.Config{
		Version: "1.0",
		Name:    "sequential",
		Steps:   []config.Step{step1, step2},
	}

	graph, err := BuildDAG(cfg.Steps)
	require.NoError(t, err)

	plan, err := GeneratePlan(graph)
	require.NoError(t, err)

	ctx := &ExecutionContext{
		Config:     cfg,
		DryRun:     false,
		WorkerPool: make(chan struct{}, 1),
		Results:    make(map[string]*model.StepResult),
		Context:    context.Background(),
		Registry:   registry,
	}

	results, err := Execute(ctx, plan)
	require.NoError(t, err)
	require.Len(t, results, 2)

	require.Equal(t, "step1", results[0].StepID)
	require.Equal(t, "success", results[0].Status)
	require.Equal(t, "step2", results[1].StepID)
	require.Equal(t, "success", results[1].Status)

	require.ElementsMatch(t, []string{"step1", "step2"}, fp.applyOrder())
}

func TestExecute_ParallelLevels(t *testing.T) {
	fp := &fakePlugin{delay: 50 * time.Millisecond}
	registry := plugin.NewPluginRegistry(plugin.DefaultConfig(), nil)
	require.NoError(t, registry.Register(fp))

	stepA := config.Step{ID: "a", Type: "command", Enabled: true}
	require.NoError(t, stepA.SetConfig(config.CommandStep{Command: "echo a"}))
	stepB := config.Step{ID: "b", Type: "command", Enabled: true}
	require.NoError(t, stepB.SetConfig(config.CommandStep{Command: "echo b"}))
	cfg := &config.Config{
		Version: "1.0",
		Name:    "parallel",
		Steps:   []config.Step{stepA, stepB},
	}

	graph, err := BuildDAG(cfg.Steps)
	require.NoError(t, err)

	plan, err := GeneratePlan(graph)
	require.NoError(t, err)

	ctx := &ExecutionContext{
		Config:     cfg,
		DryRun:     false,
		WorkerPool: make(chan struct{}, 2),
		Results:    make(map[string]*model.StepResult),
		Context:    context.Background(),
		Registry:   registry,
	}

	start := time.Now()
	results, err := Execute(ctx, plan)
	duration := time.Since(start)

	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Less(t, duration, 100*time.Millisecond, "expected parallel execution to complete within 100ms")
}

func TestExecute_FailFastOnError(t *testing.T) {
	fp := &fakePlugin{failStep: "step2"}
	registry := plugin.NewPluginRegistry(plugin.DefaultConfig(), nil)
	require.NoError(t, registry.Register(fp))

	step1 := config.Step{ID: "step1", Type: "command", Enabled: true}
	require.NoError(t, step1.SetConfig(config.CommandStep{Command: "echo 1"}))
	step2 := config.Step{ID: "step2", Type: "command", Enabled: true, DependsOn: []string{"step1"}}
	require.NoError(t, step2.SetConfig(config.CommandStep{Command: "exit 1"}))
	step3 := config.Step{ID: "step3", Type: "command", Enabled: true, DependsOn: []string{"step2"}}
	require.NoError(t, step3.SetConfig(config.CommandStep{Command: "echo 3"}))
	cfg := &config.Config{
		Version: "1.0",
		Name:    "fail-fast",
		Steps:   []config.Step{step1, step2, step3},
	}

	graph, err := BuildDAG(cfg.Steps)
	require.NoError(t, err)
	plan, err := GeneratePlan(graph)
	require.NoError(t, err)

	ctx := &ExecutionContext{
		Config:     cfg,
		DryRun:     false,
		WorkerPool: make(chan struct{}, 1),
		Results:    make(map[string]*model.StepResult),
		Context:    context.Background(),
		Registry:   registry,
	}

	results, err := Execute(ctx, plan)
	require.Error(t, err)

	var execErr *streamyerrors.ExecutionError
	require.ErrorAs(t, err, &execErr)
	require.Contains(t, err.Error(), "step2")
	require.Len(t, results, 2)
	require.Equal(t, "failed", results[1].Status)
}

func TestExecute_RespectsTimeout(t *testing.T) {
	fp := &fakePlugin{delay: 2 * time.Second}
	registry := plugin.NewPluginRegistry(plugin.DefaultConfig(), nil)
	require.NoError(t, registry.Register(fp))

	step := config.Step{ID: "slow", Type: "command", Enabled: true}
	require.NoError(t, step.SetConfig(config.CommandStep{Command: "sleep"}))
	cfg := &config.Config{
		Version:  "1.0",
		Name:     "timeout",
		Settings: config.Settings{Timeout: 1},
		Steps:    []config.Step{step},
	}

	graph, err := BuildDAG(cfg.Steps)
	require.NoError(t, err)
	plan, err := GeneratePlan(graph)
	require.NoError(t, err)

	ctx := &ExecutionContext{
		Config:     cfg,
		DryRun:     false,
		WorkerPool: make(chan struct{}, 1),
		Results:    make(map[string]*model.StepResult),
		Context:    context.Background(),
		Registry:   registry,
	}

	results, err := Execute(ctx, plan)
	require.Error(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "failed", results[0].Status)
}

func TestExecute_HandlesCancellation(t *testing.T) {
	fp := &fakePlugin{delay: 200 * time.Millisecond}
	registry := plugin.NewPluginRegistry(plugin.DefaultConfig(), nil)
	require.NoError(t, registry.Register(fp))

	step := config.Step{ID: "long", Type: "command", Enabled: true}
	require.NoError(t, step.SetConfig(config.CommandStep{Command: "sleep"}))
	cfg := &config.Config{
		Version: "1.0",
		Name:    "cancel",
		Steps:   []config.Step{step},
	}

	graph, err := BuildDAG(cfg.Steps)
	require.NoError(t, err)
	plan, err := GeneratePlan(graph)
	require.NoError(t, err)

	ctxWithCancel, cancel := context.WithCancel(context.Background())
	cancel()

	ctx := &ExecutionContext{
		Config:     cfg,
		DryRun:     false,
		WorkerPool: make(chan struct{}, 1),
		Results:    make(map[string]*model.StepResult),
		Context:    ctxWithCancel,
		Registry:   registry,
	}

	results, err := Execute(ctx, plan)
	require.Error(t, err)
	require.Len(t, results, 0)
}

type fakePlugin struct {
	mu             sync.Mutex
	calls          []string
	failStep       string
	delay          time.Duration
	verifyStatuses map[string]model.VerificationStatus
	verifyMessages map[string]string
	verifyErrors   map[string]error
	verifyCalls    []string
}

func (p *fakePlugin) PluginMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name:    "command",
		Version: "1.0.0",
		Type:    "command",
	}
}

func (p *fakePlugin) Schema() any {
	return nil
}

func (p *fakePlugin) Evaluate(ctx context.Context, step *config.Step) (*model.EvaluationResult, error) {
	p.mu.Lock()
	p.verifyCalls = append(p.verifyCalls, step.ID)
	p.mu.Unlock()

	if p.delay > 0 {
		select {
		case <-ctx.Done():
			return &model.EvaluationResult{StepID: step.ID, CurrentState: model.StatusUnknown, RequiresAction: false}, ctx.Err()
		case <-time.After(p.delay):
		}
	}

	if err, ok := p.verifyErrors[step.ID]; ok {
		return nil, err
	}

	if status, ok := p.verifyStatuses[step.ID]; ok {
		message := p.verifyMessages[step.ID]
		if message == "" {
			message = fmt.Sprintf("fake evaluation %s", status)
		}
		return &model.EvaluationResult{
			StepID:         step.ID,
			CurrentState:   status,
			RequiresAction: status != model.StatusSatisfied,
			Message:        message,
		}, nil
	}

	return &model.EvaluationResult{
		StepID:         step.ID,
		CurrentState:   model.StatusMissing,
		RequiresAction: true,
		Message:        "fake evaluation requires action",
	}, nil
}

func (p *fakePlugin) Apply(ctx context.Context, evalResult *model.EvaluationResult, step *config.Step) (*model.StepResult, error) {
	p.mu.Lock()
	p.calls = append(p.calls, step.ID)
	p.mu.Unlock()

	if p.failStep == step.ID {
		return &model.StepResult{StepID: step.ID, Status: "failed", Error: errors.New("boom")}, errors.New("boom")
	}

	return &model.StepResult{StepID: step.ID, Status: "success", Message: "ok"}, nil
}

func (p *fakePlugin) applyOrder() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.calls))
	copy(out, p.calls)
	return out
}

func (p *fakePlugin) verifyOrder() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.verifyCalls))
	copy(out, p.verifyCalls)
	return out
}

func TestExecute_ContinueOnError(t *testing.T) {
	fp := &fakePlugin{failStep: "step1"}
	registry := plugin.NewPluginRegistry(plugin.DefaultConfig(), nil)
	require.NoError(t, registry.Register(fp))

	step1 := config.Step{ID: "step1", Type: "command", Enabled: true}
	require.NoError(t, step1.SetConfig(config.CommandStep{Command: "fail"}))
	step2 := config.Step{ID: "step2", Type: "command", Enabled: true}
	require.NoError(t, step2.SetConfig(config.CommandStep{Command: "echo"}))
	cfg := &config.Config{
		Version:  "1.0",
		Name:     "continue",
		Settings: config.Settings{ContinueOnError: true},
		Steps:    []config.Step{step1, step2},
	}

	graph, err := BuildDAG(cfg.Steps)
	require.NoError(t, err)
	plan, err := GeneratePlan(graph)
	require.NoError(t, err)

	ctx := &ExecutionContext{
		Config:          cfg,
		DryRun:          false,
		ContinueOnError: true,
		WorkerPool:      make(chan struct{}, 1),
		Results:         make(map[string]*model.StepResult),
		Context:         context.Background(),
		Registry:        registry,
	}

	results, err := Execute(ctx, plan)
	require.Error(t, err)
	require.Len(t, results, 2)
	statuses := []string{results[0].Status, results[1].Status}
	require.Contains(t, statuses, model.StatusFailed)
	require.Contains(t, statuses, model.StatusSuccess)
}

func TestTimeoutResult(t *testing.T) {
	t.Run("creates timeout result with nil error", func(t *testing.T) {
		stepID := "test-step"

		result, err := timeoutResult(stepID, nil)

		require.Error(t, err)
		require.IsType(t, &streamyerrors.ExecutionError{}, err)

		require.NotNil(t, result)
		require.Equal(t, stepID, result.StepID)
		require.Equal(t, model.StatusFailed, result.Status)
		require.Equal(t, "timeout exceeded", result.Message)
		require.ErrorIs(t, result.Error, context.DeadlineExceeded)
	})

	t.Run("creates timeout result with provided error", func(t *testing.T) {
		stepID := "test-step"
		customErr := errors.New("custom timeout error")

		result, err := timeoutResult(stepID, customErr)

		require.Error(t, err)
		require.IsType(t, &streamyerrors.ExecutionError{}, err)

		require.NotNil(t, result)
		require.Equal(t, stepID, result.StepID)
		require.Equal(t, model.StatusFailed, result.Status)
		require.Equal(t, "timeout exceeded", result.Message)
		require.ErrorIs(t, result.Error, customErr)
	})
}

func TestVerifySteps_BlocksDependentsWhenPrerequisitesUnsatisfied(t *testing.T) {
	fp := &fakePlugin{
		verifyStatuses: map[string]model.VerificationStatus{
			"provision_vm": model.StatusMissing,
		},
		verifyMessages: map[string]string{
			"provision_vm": "resource missing",
		},
	}
	registry := plugin.NewPluginRegistry(plugin.DefaultConfig(), nil)
	if err := registry.Register(fp); err != nil {
		t.Fatalf("failed to register fake plugin: %v", err)
	}

	stepProvision := config.Step{ID: "provision_vm", Type: "command", Enabled: true}
	require.NoError(t, stepProvision.SetConfig(config.CommandStep{Command: "echo"}))
	stepDeploy := config.Step{ID: "deploy_app", Type: "command", Enabled: true, DependsOn: []string{"provision_vm"}}
	require.NoError(t, stepDeploy.SetConfig(config.CommandStep{Command: "echo"}))
	steps := []config.Step{stepProvision, stepDeploy}

	executor := NewExecutor(nil)
	execCtx := &ExecutionContext{
		Registry: registry,
		Context:  context.Background(),
	}
	summary, err := executor.VerifySteps(execCtx, steps, time.Second)
	require.NoError(t, err)

	require.Len(t, summary.Results, 2)
	require.Equal(t, "provision_vm", summary.Results[0].StepID)
	require.Equal(t, model.StatusMissing, summary.Results[0].Status)
	require.Equal(t, "deploy_app", summary.Results[1].StepID)
	require.Equal(t, model.StatusBlocked, summary.Results[1].Status)
	require.Contains(t, summary.Results[1].Message, "dependencies not satisfied")
	require.NotNil(t, summary.Results[1].Error)
	require.Equal(t, 1, summary.Missing)
	require.Equal(t, 1, summary.Blocked)

	require.Equal(t, []string{"provision_vm"}, fp.verifyOrder())
}

func TestVerifySteps_PropagatesPluginVerificationErrors(t *testing.T) {
	t.Run("returns validation errors", func(t *testing.T) {
		validationErr := &streamyerrors.ValidationError{Field: "pattern", Message: "invalid regex"}
		fp := &fakePlugin{
			verifyErrors: map[string]error{
				"lint": validationErr,
			},
		}
		registry := plugin.NewPluginRegistry(plugin.DefaultConfig(), nil)
		if err := registry.Register(fp); err != nil {
			t.Fatalf("failed to register fake plugin: %v", err)
		}

		step := config.Step{ID: "lint", Type: "command", Enabled: true}
		require.NoError(t, step.SetConfig(config.CommandStep{Command: "echo"}))
		steps := []config.Step{step}

		executor := NewExecutor(nil)
		execCtx := &ExecutionContext{
			Registry: registry,
			Context:  context.Background(),
		}
		summary, err := executor.VerifySteps(execCtx, steps, time.Second)

		require.NotNil(t, summary)
		require.Error(t, err)
		require.ErrorIs(t, err, validationErr)
		require.Empty(t, summary.Results)
	})

	t.Run("wraps unexpected errors as execution errors", func(t *testing.T) {
		underlying := errors.New("verify failed")
		fp := &fakePlugin{
			verifyErrors: map[string]error{
				"lint": underlying,
			},
		}
		registry := plugin.NewPluginRegistry(plugin.DefaultConfig(), nil)
		require.NoError(t, registry.Register(fp))

		step := config.Step{ID: "lint", Type: "command", Enabled: true}
		require.NoError(t, step.SetConfig(config.CommandStep{Command: "echo"}))
		steps := []config.Step{step}

		executor := NewExecutor(nil)
		execCtx := &ExecutionContext{
			Registry: registry,
			Context:  context.Background(),
		}
		summary, err := executor.VerifySteps(execCtx, steps, time.Second)

		require.NotNil(t, summary)
		require.Error(t, err)

		execErr := &streamyerrors.ExecutionError{}
		require.ErrorAs(t, err, &execErr)
		require.Equal(t, "lint", execErr.StepID)
		require.ErrorIs(t, err, underlying)
		require.Empty(t, summary.Results)
	})
}
