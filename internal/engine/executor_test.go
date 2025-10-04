package engine

import (
	"context"
	"errors"
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
	plugin.ResetRegistry()
	fp := &fakePlugin{}
	require.NoError(t, plugin.RegisterPlugin("command", fp))

	cfg := &config.Config{
		Version: "1.0",
		Name:    "sequential",
		Steps: []config.Step{
			{ID: "step1", Type: "command", Enabled: true, Command: &config.CommandStep{Command: "echo 1"}},
			{ID: "step2", Type: "command", Enabled: true, DependsOn: []string{"step1"}, Command: &config.CommandStep{Command: "echo 2"}},
		},
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
	plugin.ResetRegistry()
	fp := &fakePlugin{delay: 50 * time.Millisecond}
	require.NoError(t, plugin.RegisterPlugin("command", fp))

	cfg := &config.Config{
		Version: "1.0",
		Name:    "parallel",
		Steps: []config.Step{
			{ID: "a", Type: "command", Enabled: true, Command: &config.CommandStep{Command: "echo a"}},
			{ID: "b", Type: "command", Enabled: true, Command: &config.CommandStep{Command: "echo b"}},
		},
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
	}

	start := time.Now()
	results, err := Execute(ctx, plan)
	duration := time.Since(start)

	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Less(t, duration, 100*time.Millisecond, "expected parallel execution to complete within 100ms")
}

func TestExecute_FailFastOnError(t *testing.T) {
	plugin.ResetRegistry()
	fp := &fakePlugin{failStep: "step2"}
	require.NoError(t, plugin.RegisterPlugin("command", fp))

	cfg := &config.Config{
		Version: "1.0",
		Name:    "fail-fast",
		Steps: []config.Step{
			{ID: "step1", Type: "command", Enabled: true, Command: &config.CommandStep{Command: "echo 1"}},
			{ID: "step2", Type: "command", Enabled: true, DependsOn: []string{"step1"}, Command: &config.CommandStep{Command: "exit 1"}},
			{ID: "step3", Type: "command", Enabled: true, DependsOn: []string{"step2"}, Command: &config.CommandStep{Command: "echo 3"}},
		},
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
	plugin.ResetRegistry()
	fp := &fakePlugin{delay: 2 * time.Second}
	require.NoError(t, plugin.RegisterPlugin("command", fp))

	cfg := &config.Config{
		Version: "1.0",
		Name:    "timeout",
		Settings: config.Settings{
			Timeout: 1,
		},
		Steps: []config.Step{
			{ID: "slow", Type: "command", Enabled: true, Command: &config.CommandStep{Command: "sleep"}},
		},
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
	}

	results, err := Execute(ctx, plan)
	require.Error(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "failed", results[0].Status)
}

func TestExecute_HandlesCancellation(t *testing.T) {
	plugin.ResetRegistry()
	fp := &fakePlugin{delay: 200 * time.Millisecond}
	require.NoError(t, plugin.RegisterPlugin("command", fp))

	cfg := &config.Config{
		Version: "1.0",
		Name:    "cancel",
		Steps: []config.Step{
			{ID: "long", Type: "command", Enabled: true, Command: &config.CommandStep{Command: "sleep"}},
		},
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
	}

	results, err := Execute(ctx, plan)
	require.Error(t, err)
	require.Len(t, results, 0)
}

type fakePlugin struct {
	mu       sync.Mutex
	calls    []string
	failStep string
	delay    time.Duration
}

func (p *fakePlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		Name: "fake",
		Type: "command",
	}
}

func (p *fakePlugin) Schema() interface{} {
	return nil
}

func (p *fakePlugin) Check(ctx context.Context, step *config.Step) (bool, error) {
	return false, nil
}

func (p *fakePlugin) Apply(ctx context.Context, step *config.Step) (*model.StepResult, error) {
	if p.delay > 0 {
		select {
		case <-ctx.Done():
			return &model.StepResult{StepID: step.ID, Status: "failed", Error: ctx.Err()}, ctx.Err()
		case <-time.After(p.delay):
		}
	}

	p.mu.Lock()
	p.calls = append(p.calls, step.ID)
	p.mu.Unlock()

	if p.failStep == step.ID {
		return &model.StepResult{StepID: step.ID, Status: "failed", Error: errors.New("boom")}, errors.New("boom")
	}

	return &model.StepResult{StepID: step.ID, Status: "success", Message: "ok"}, nil
}

func (p *fakePlugin) DryRun(ctx context.Context, step *config.Step) (*model.StepResult, error) {
	return &model.StepResult{StepID: step.ID, Status: "skipped", Message: "dry-run"}, nil
}

func (p *fakePlugin) applyOrder() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.calls))
	copy(out, p.calls)
	return out
}

func TestExecute_ContinueOnError(t *testing.T) {
	plugin.ResetRegistry()
	fp := &fakePlugin{failStep: "step1"}
	require.NoError(t, plugin.RegisterPlugin("command", fp))

	cfg := &config.Config{
		Version: "1.0",
		Name:    "continue",
		Settings: config.Settings{
			ContinueOnError: true,
		},
		Steps: []config.Step{
			{ID: "step1", Type: "command", Enabled: true, Command: &config.CommandStep{Command: "fail"}},
			{ID: "step2", Type: "command", Enabled: true, Command: &config.CommandStep{Command: "echo"}},
		},
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
