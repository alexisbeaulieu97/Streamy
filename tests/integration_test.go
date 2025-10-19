package tests

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

func TestIntegrationSimpleExecution(t *testing.T) {
	t.Parallel()

	h := newAppHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pipeline, results, summary, err := h.ApplyUseCase.Apply(ctx, fixturePath("simple.yaml"), false)
	require.NoError(t, err)
	require.NotNil(t, pipeline)
	require.NotNil(t, summary)
	require.Zero(t, summary.FailedChecks, "expected all validations to pass")

	require.Len(t, results, len(pipeline.Steps))
	for _, res := range results {
		require.Truef(t, res.IsSuccess(), "step %s completed with status %s", res.StepID, res.Status)
	}
}

func TestIntegrationComplexPlan(t *testing.T) {
	t.Parallel()

	h := newAppHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pipeline, plan, err := h.PrepareUseCase.Prepare(ctx, fixturePath("complex.yaml"))
	require.NoError(t, err)
	require.NotNil(t, pipeline)
	require.NotNil(t, plan)
	require.GreaterOrEqual(t, len(plan.Levels), 3)

	require.Contains(t, plan.Levels[0].StepIDs, "install_tools")
	require.Contains(t, plan.Levels[1].StepIDs, "fetch_repo")
}

func TestIntegrationDryRunSkipsExecution(t *testing.T) {
	t.Parallel()

	h := newAppHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pipeline, results, summary, err := h.ApplyUseCase.Apply(ctx, fixturePath("simple.yaml"), true)
	require.NoError(t, err)
	require.NotNil(t, pipeline)
	require.Nil(t, summary, "dry-run should not produce validation summary")
	require.Len(t, results, len(pipeline.Steps))

	for _, res := range results {
		require.Contains(t, []domainpipeline.ResultStatus{domainpipeline.StatusSuccess, domainpipeline.StatusSkipped}, res.Status)
	}
}

func TestIntegrationIdempotentRuns(t *testing.T) {
	t.Parallel()

	h := newAppHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, firstResults, _, err := h.ApplyUseCase.Apply(ctx, fixturePath("simple.yaml"), false)
	require.NoError(t, err)
	require.NotEmpty(t, firstResults)

	_, secondResults, _, err := h.ApplyUseCase.Apply(ctx, fixturePath("simple.yaml"), false)
	require.NoError(t, err)
	require.Len(t, secondResults, len(firstResults))
	for _, res := range secondResults {
		require.Falsef(t, res.IsFailure(), "step %s should not fail on subsequent run", res.StepID)
	}
}

func TestIntegrationErrorHandling(t *testing.T) {
	t.Parallel()

	h := newAppHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	configPath := writeConfig(t, `
version: "1.0"
name: "Failure"
steps:
  - id: fail
    type: command
    command: "__streamy_fail__"
`)

	_, results, _, err := h.ApplyUseCase.Apply(ctx, configPath, false)
	require.Error(t, err)
	require.NotEmpty(t, results)

	var domainErr *domainpipeline.DomainError
	require.ErrorAs(t, err, &domainErr)
	require.Equal(t, domainpipeline.ErrCodeExecution, domainErr.Code)
	require.Equal(t, domainpipeline.StatusFailure, results[0].Status)
	require.NotNil(t, results[0].Error)
}

func TestIntegrationValidationFailure(t *testing.T) {
	t.Parallel()

	h := newAppHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	missingPath := filepath.Join(t.TempDir(), "missing.txt")
	configPath := writeConfig(t, fmt.Sprintf(`
version: "1.0"
name: "Validation Failure"
steps:
  - id: say_hello
    type: command
    command: "echo hello"
validations:
  - type: file_exists
    path: %q
`, missingPath))

	_, results, summary, err := h.ApplyUseCase.Apply(ctx, configPath, false)
	require.Error(t, err)
	require.NotNil(t, summary)
	require.Equal(t, 1, summary.FailedChecks)
	require.NotEmpty(t, results)
}

func TestIntegrationParseError(t *testing.T) {
	t.Parallel()

	h := newAppHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	invalidPath := writeConfig(t, "not: [valid")
	_, _, err := h.PrepareUseCase.Prepare(ctx, invalidPath)
	require.Error(t, err)
}

func TestIntegrationCycleDetection(t *testing.T) {
	t.Parallel()

	h := newAppHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, _, err := h.PrepareUseCase.Prepare(ctx, fixturePath("cycle.yaml"))
	require.Error(t, err)

	var domainErr *domainpipeline.DomainError
	require.ErrorAs(t, err, &domainErr)
	require.True(t,
		domainErr.Code == domainpipeline.ErrCodeCycle || domainErr.Code == domainpipeline.ErrCodeValidation,
		"expected cycle-related validation error, got %s", domainErr.Code,
	)
}

func TestIntegrationCancellationStopsExecutor(t *testing.T) {
	t.Parallel()

	started := make(chan string, 3)
	sleepPlugin := newSleepPlugin(started)
	h := newCustomAppHarness(t, sleepPlugin)

	tempDir := t.TempDir()
	configPath := writeConfig(t, fmt.Sprintf(`
version: "1.0"
name: "Cancellation"
settings:
  parallel: 3
steps:
  - id: sleep_1
    type: command
    command: "echo sleep"
    duration_ms: 5000
    temp_dir: %q
  - id: sleep_2
    type: command
    command: "echo sleep"
    duration_ms: 5000
    temp_dir: %q
  - id: sleep_3
    type: command
    command: "echo sleep"
    duration_ms: 5000
    temp_dir: %q
`, tempDir, tempDir, tempDir))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type outcome struct {
		results []domainpipeline.StepResult
		err     error
	}

	resultCh := make(chan outcome, 1)
	startGoroutines := runtime.NumGoroutine()

	go func() {
		_, results, _, execErr := h.ApplyUseCase.Apply(ctx, configPath, false)
		resultCh <- outcome{results: results, err: execErr}
	}()

	startedCount := 0
	for startedCount < 3 {
		select {
		case <-started:
			startedCount++
		case out := <-resultCh:
			require.NoError(t, out.err, "executor exited before cancellation")
			t.Fatalf("executor completed before all steps started")
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for step %d to start", startedCount+1)
		}
	}

	cancel()

	var out outcome
	select {
	case out = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("executor did not return within timeout after cancellation")
	}

	require.Error(t, out.err)
	require.True(t, errors.Is(out.err, context.Canceled) || errors.Is(out.err, context.DeadlineExceeded))
	require.NotEmpty(t, out.results, "expected partial results when cancelled")

	waitForGoroutinesToDrop(t, startGoroutines, 5*time.Second)

	entries, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	require.Len(t, entries, 0, "expected temporary files to be cleaned up")
}

func fixturePath(name string) string {
	return filepath.Join("..", "testdata", "configs", name)
}

func writeConfig(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o644))
	return path
}

func waitForGoroutinesToDrop(t *testing.T, baseline int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if runtime.NumGoroutine() <= baseline {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("goroutine leak detected; baseline=%d current=%d", baseline, runtime.NumGoroutine())
		}
		time.Sleep(50 * time.Millisecond)
	}
}

type sleepConfig struct {
	Duration time.Duration
	TempDir  string
}

type sleepPlugin struct {
	started chan<- string
}

func newSleepPlugin(started chan<- string) ports.Plugin {
	return &sleepPlugin{started: started}
}

func (p *sleepPlugin) Metadata() domainplugin.Metadata {
	return domainplugin.Metadata{
		ID:          "sleep-command",
		Name:        "sleep-command",
		Version:     "1.0.0",
		Type:        domainplugin.TypeCommand,
		Description: "sleeps for a configurable duration (test only)",
	}
}

func (p *sleepPlugin) Evaluate(ctx context.Context, step domainpipeline.Step) (*domainpipeline.EvaluationResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, domainpipeline.NewDomainError(domainpipeline.ErrCodeCancelled, "evaluation cancelled", err, map[string]interface{}{"step_id": step.ID})
	}

	cfg := parseSleepConfig(step)
	return &domainpipeline.EvaluationResult{
		RequiresAction: true,
		CurrentState:   "pending",
		Diff:           fmt.Sprintf("sleep for %s", cfg.Duration),
		InternalData:   cfg,
	}, nil
}

func (p *sleepPlugin) Apply(ctx context.Context, evaluation *domainpipeline.EvaluationResult, step domainpipeline.Step) (*domainpipeline.StepResult, error) {
	if evaluation == nil {
		return nil, domainpipeline.NewDomainError(domainpipeline.ErrCodeExecution, "missing evaluation data", nil, map[string]interface{}{"step_id": step.ID})
	}

	cfg, ok := evaluation.InternalData.(sleepConfig)
	if !ok {
		return nil, domainpipeline.NewDomainError(domainpipeline.ErrCodeExecution, "invalid evaluation payload", nil, map[string]interface{}{"step_id": step.ID})
	}

	if p.started != nil {
		select {
		case p.started <- step.ID:
		default:
		}
	}

	var tempFile string
	if cfg.TempDir != "" {
		tempFile = filepath.Join(cfg.TempDir, fmt.Sprintf("%s.tmp", step.ID))
		if err := os.WriteFile(tempFile, []byte("sleep"), 0o600); err != nil {
			return nil, err
		}
		defer func() {
			_ = os.Remove(tempFile)
		}()
	}

	timer := time.NewTimer(cfg.Duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
	}

	return &domainpipeline.StepResult{
		StepID:   step.ID,
		Status:   domainpipeline.StatusSuccess,
		Message:  "slept successfully",
		Duration: int(cfg.Duration / time.Millisecond),
	}, nil
}

func parseSleepConfig(step domainpipeline.Step) sleepConfig {
	cfg := sleepConfig{
		Duration: 5 * time.Second,
	}

	if step.Config == nil {
		return cfg
	}

	if raw, ok := step.Config["duration_ms"]; ok {
		switch v := raw.(type) {
		case int:
			cfg.Duration = time.Duration(v) * time.Millisecond
		case int64:
			cfg.Duration = time.Duration(v) * time.Millisecond
		case float64:
			cfg.Duration = time.Duration(int(v)) * time.Millisecond
		case string:
			if parsed, err := strconv.Atoi(v); err == nil {
				cfg.Duration = time.Duration(parsed) * time.Millisecond
			}
		}
	}

	if raw, ok := step.Config["temp_dir"]; ok {
		if dir, ok := raw.(string); ok {
			cfg.TempDir = dir
		}
	}

	if cfg.Duration <= 0 {
		cfg.Duration = 5 * time.Second
	}
	return cfg
}
