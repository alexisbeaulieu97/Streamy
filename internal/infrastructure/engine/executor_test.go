package engine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/infrastructure/logging"
	infraPlugin "github.com/alexisbeaulieu97/streamy/internal/infrastructure/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

type executorStubPlugin struct {
	meta          domainplugin.Metadata
	requireAction bool
	applyError    error
	evaluateError error
	onEvaluate    func(context.Context, domainpipeline.Step) error
	onApply       func(context.Context, domainpipeline.Step) error
}

func (s *executorStubPlugin) Metadata() domainplugin.Metadata { return s.meta }

func (s *executorStubPlugin) Evaluate(ctx context.Context, step domainpipeline.Step) (*domainpipeline.EvaluationResult, error) {
	if s.onEvaluate != nil {
		if err := s.onEvaluate(ctx, step); err != nil {
			return nil, err
		}
	}

	if s.evaluateError != nil {
		return nil, s.evaluateError
	}

	return &domainpipeline.EvaluationResult{RequiresAction: s.requireAction, Diff: "diff"}, nil
}

func (s *executorStubPlugin) Apply(ctx context.Context, eval *domainpipeline.EvaluationResult, step domainpipeline.Step) (*domainpipeline.StepResult, error) {
	if s.onApply != nil {
		if err := s.onApply(ctx, step); err != nil {
			return nil, err
		}
	}

	if s.applyError != nil {
		return nil, s.applyError
	}

	return &domainpipeline.StepResult{StepID: step.ID, Status: domainpipeline.StatusSuccess, Changed: eval.RequiresAction}, nil
}

func TestExecutorExecuteSuccess(t *testing.T) {
	registry := infraPlugin.NewRegistry()

	plug := &executorStubPlugin{meta: domainplugin.Metadata{ID: "cmd", Name: "Command", Type: domainplugin.Type("command"), Version: "1.0.0"}, requireAction: true}
	if err := registry.Register(plug); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	eventRecorder := &stubEventPublisher{}
	executor := NewExecutor(registry,
		WithExecutorLogger(logging.NewNoOpLogger()),
		WithExecutorEvents(eventRecorder),
	)
	pipeline := &domainpipeline.Pipeline{
		Name: "test",
		Settings: domainpipeline.Settings{
			ContinueOnError: false,
			DryRun:          false,
			Parallel:        1,
		},
		Steps: []domainpipeline.Step{{ID: "cmd", Type: domainpipeline.StepType("command"), Enabled: true}},
	}
	plan := &domainpipeline.ExecutionPlan{Levels: []domainpipeline.ExecutionLevel{{Level: 0, StepIDs: []string{"cmd"}}}}

	results, err := executor.Execute(context.Background(), plan, pipeline)
	if err != nil {
		t.Fatalf("unexpected execute err: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected one result, got %d", len(results))
	}

	if results[0].Status != domainpipeline.StatusSuccess {
		t.Fatalf("expected success status, got %s", results[0].Status)
	}

	if !eventRecorder.contains(ports.EventStepStarted) || !eventRecorder.contains(ports.EventStepCompleted) {
		t.Fatalf("expected step started and completed events, got %#v", eventRecorder.events)
	}
}

func TestExecutorExecuteDryRun(t *testing.T) {
	registry := infraPlugin.NewRegistry()

	plug := &executorStubPlugin{meta: domainplugin.Metadata{ID: "cmd", Name: "Command", Type: domainplugin.Type("command"), Version: "1.0.0"}, requireAction: true}
	if err := registry.Register(plug); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	eventRecorder := &stubEventPublisher{}
	executor := NewExecutor(registry, WithExecutorEvents(eventRecorder))
	pipeline := &domainpipeline.Pipeline{
		Settings: domainpipeline.Settings{DryRun: true},
		Steps:    []domainpipeline.Step{{ID: "cmd", Type: domainpipeline.StepType("command"), Enabled: true}},
	}
	plan := &domainpipeline.ExecutionPlan{Levels: []domainpipeline.ExecutionLevel{{Level: 0, StepIDs: []string{"cmd"}}}}

	results, err := executor.Execute(context.Background(), plan, pipeline)
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected result count 1, got %d", len(results))
	}

	if !results[0].Changed {
		t.Fatalf("expected dry-run to mark changed")
	}

	if !eventRecorder.contains(ports.EventStepCompleted) {
		t.Fatalf("expected step completed event during dry run")
	}
}

func TestExecutorExecuteStepTimeout(t *testing.T) {
	registry := infraPlugin.NewRegistry()

	plug := &executorStubPlugin{
		meta:          domainplugin.Metadata{ID: "command", Name: "Command", Type: domainplugin.TypeCommand, Version: "1.0.0"},
		requireAction: true,
		onApply: func(ctx context.Context, _ domainpipeline.Step) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}
	if err := registry.Register(plug); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	pipeline := &domainpipeline.Pipeline{
		Name: "timeout",
		Settings: domainpipeline.Settings{
			Timeout:  1,
			Parallel: 1,
		},
		Steps: []domainpipeline.Step{{ID: "slow", Type: domainpipeline.StepTypeCommand, Enabled: true}},
	}

	plan := &domainpipeline.ExecutionPlan{Levels: []domainpipeline.ExecutionLevel{{Level: 0, StepIDs: []string{"slow"}}}}

	results, err := NewExecutor(registry, WithExecutorLogger(logging.NewNoOpLogger())).Execute(context.Background(), plan, pipeline)
	if err == nil {
		t.Fatal("expected timeout error")
	}

	var derr *domainpipeline.DomainError
	if !errors.As(err, &derr) || derr.Code != domainpipeline.ErrCodeTimeout {
		t.Fatalf("expected timeout domain error, got %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected single result, got %d", len(results))
	}

	if results[0].Error == nil {
		t.Fatal("expected step result to include timeout error")
	}

	var resErr *domainpipeline.DomainError
	if !errors.As(results[0].Error, &resErr) || resErr.Code != domainpipeline.ErrCodeTimeout {
		t.Fatalf("expected timeout domain error, got %+v", results[0].Error)
	}
}

func TestExecutorVerify(t *testing.T) {
	registry := infraPlugin.NewRegistry()

	plug := &executorStubPlugin{meta: domainplugin.Metadata{ID: "cmd", Name: "Command", Type: domainplugin.Type("command"), Version: "1.0.0"}, requireAction: false}
	if err := registry.Register(plug); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	executor := NewExecutor(registry)

	pipeline := &domainpipeline.Pipeline{
		Steps: []domainpipeline.Step{{ID: "cmd", Type: domainpipeline.StepType("command"), Enabled: true}},
	}

	results, err := executor.Verify(context.Background(), pipeline)
	if err != nil {
		t.Fatalf("verify err: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected one verification result, got %d", len(results))
	}

	if results[0].Status != domainpipeline.VerificationSatisfied {
		t.Fatalf("expected satisfied status, got %s", results[0].Status)
	}
}

func TestExecutorVerifyStepTimeout(t *testing.T) {
	registry := infraPlugin.NewRegistry()

	plug := &executorStubPlugin{
		meta: domainplugin.Metadata{ID: "command", Name: "Command", Type: domainplugin.TypeCommand, Version: "1.0.0"},
		onEvaluate: func(ctx context.Context, _ domainpipeline.Step) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}
	if err := registry.Register(plug); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	pipeline := &domainpipeline.Pipeline{
		Steps: []domainpipeline.Step{{ID: "check", Type: domainpipeline.StepTypeCommand, Enabled: true, VerifyTimeout: 1}},
	}

	_, err := NewExecutor(registry).Verify(context.Background(), pipeline)
	if err == nil {
		t.Fatal("expected timeout error")
	}

	var derr *domainpipeline.DomainError
	if !errors.As(err, &derr) || derr.Code != domainpipeline.ErrCodeTimeout {
		t.Fatalf("expected timeout domain error, got %v", err)
	}
}

func TestExecutorStepFailure(t *testing.T) {
	registry := infraPlugin.NewRegistry()

	plug := &executorStubPlugin{meta: domainplugin.Metadata{ID: "cmd", Name: "Command", Type: domainplugin.Type("command"), Version: "1.0.0"}, requireAction: true, applyError: errors.New("boom")}
	if err := registry.Register(plug); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	eventRecorder := &stubEventPublisher{}
	executor := NewExecutor(registry, WithExecutorEvents(eventRecorder))

	pipeline := &domainpipeline.Pipeline{
		Steps: []domainpipeline.Step{{ID: "cmd", Type: domainpipeline.StepType("command"), Enabled: true}},
	}
	plan := &domainpipeline.ExecutionPlan{Levels: []domainpipeline.ExecutionLevel{{Level: 0, StepIDs: []string{"cmd"}}}}

	results, err := executor.Execute(context.Background(), plan, pipeline)
	if err == nil {
		t.Fatal("expected execution error")
	}

	if len(results) != 1 {
		t.Fatalf("expected single result, got %d", len(results))
	}

	if results[0].Status != domainpipeline.StatusFailure {
		t.Fatalf("expected failure status, got %s", results[0].Status)
	}

	if !eventRecorder.contains(ports.EventStepFailed) {
		t.Fatalf("expected step failed event")
	}
}

func TestExecutorCancellationBetweenLevels(t *testing.T) {
	registry := infraPlugin.NewRegistry()

	ctx, cancel := context.WithCancel(context.Background())

	pluginImpl := &executorStubPlugin{
		meta:          domainplugin.Metadata{ID: "command", Name: "Command", Type: domainplugin.TypeCommand, Version: "1.0.0"},
		requireAction: true,
		onApply: func(_ context.Context, step domainpipeline.Step) error {
			if step.ID == "first-step" {
				cancel()
			}

			return nil
		},
	}

	var secondEvaluations int32

	pluginImpl.onEvaluate = func(_ context.Context, step domainpipeline.Step) error {
		if step.ID == "second-step" {
			atomic.AddInt32(&secondEvaluations, 1)
		}

		return nil
	}

	if err := registry.Register(pluginImpl); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	pipeline := &domainpipeline.Pipeline{
		Name: "cancel-between-levels",
		Settings: domainpipeline.Settings{
			Parallel: 1,
		},
		Steps: []domainpipeline.Step{
			{ID: "first-step", Type: domainpipeline.StepTypeCommand, Enabled: true},
			{ID: "second-step", Type: domainpipeline.StepTypeCommand, Enabled: true},
		},
	}

	plan := &domainpipeline.ExecutionPlan{
		Levels: []domainpipeline.ExecutionLevel{
			{Level: 0, StepIDs: []string{"first-step"}},
			{Level: 1, StepIDs: []string{"second-step"}},
		},
	}

	executor := NewExecutor(registry, WithExecutorLogger(logging.NewNoOpLogger()))

	results, err := executor.Execute(ctx, plan, pipeline)
	if err == nil {
		t.Fatal("expected cancellation error")
	}

	var domainErr *domainpipeline.DomainError
	if !errors.As(err, &domainErr) || domainErr.Code != domainpipeline.ErrCodeCancelled {
		t.Fatalf("expected cancellation domain error, got %v", err)
	}

	if len(results) == 0 {
		t.Fatalf("expected at least one step result")
	}

	if atomic.LoadInt32(&secondEvaluations) != 0 {
		t.Fatalf("expected second level plugin not to evaluate")
	}
}

func TestExecutorExecuteCancellationCleanup(t *testing.T) {
	registry := infraPlugin.NewRegistry()

	plug := &executorStubPlugin{
		meta:          domainplugin.Metadata{ID: "cmd", Name: "Command", Type: domainplugin.TypeCommand, Version: "1.0.0"},
		requireAction: true,
		// Delay Apply so that cancellation happens while goroutine is active.
		onApply: func(ctx context.Context, _ domainpipeline.Step) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}
	if err := registry.Register(plug); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	executor := NewExecutor(registry, WithExecutorLogger(logging.NewNoOpLogger()))

	plan := &domainpipeline.ExecutionPlan{
		Levels: []domainpipeline.ExecutionLevel{{Level: 0, StepIDs: []string{"cmd"}}},
	}
	pipeline := &domainpipeline.Pipeline{
		Steps: []domainpipeline.Step{{ID: "cmd", Type: domainpipeline.StepTypeCommand, Enabled: true}},
	}

	ctx, cancel := context.WithCancel(context.Background())
	startG := runtime.NumGoroutine()
	resultCh := make(chan struct{})

	go func() {
		_, _ = executor.Execute(ctx, plan, pipeline)

		close(resultCh)
	}()

	cancel()

	select {
	case <-resultCh:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("executor did not return promptly after cancellation")
	}

	waitForGoroutinesToSettle(startG, t)
}

func TestExecutorCancellationLongRunningPipeline(t *testing.T) {
	registry := infraPlugin.NewRegistry()
	startedCh := make(chan string, 4)
	tempDir := t.TempDir()

	var (
		tempFilesMu sync.Mutex
		tempFiles   []string
	)

	plug := createLongRunningPlugin(tempDir, startedCh, &tempFiles, &tempFilesMu)
	if err := registry.Register(plug); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	pipeline := &domainpipeline.Pipeline{
		Name: "long-running",
		Settings: domainpipeline.Settings{
			Parallel: 3,
		},
		Steps: []domainpipeline.Step{
			{ID: "slow-1", Type: domainpipeline.StepTypeCommand, Enabled: true},
			{ID: "slow-2", Type: domainpipeline.StepTypeCommand, Enabled: true},
			{ID: "slow-3", Type: domainpipeline.StepTypeCommand, Enabled: true},
		},
	}
	plan := &domainpipeline.ExecutionPlan{
		Levels: []domainpipeline.ExecutionLevel{
			{Level: 0, StepIDs: []string{"slow-1", "slow-2", "slow-3"}},
		},
	}

	executor := NewExecutor(registry, WithExecutorLogger(logging.NewNoOpLogger()))

	startG := runtime.NumGoroutine()

	ctx, cancel := context.WithCancel(context.Background())
	resultCh := make(chan error, 1)

	go func() {
		_, err := executor.Execute(ctx, plan, pipeline)
		resultCh <- err
	}()

	waitForStepsToStart(startedCh, len(plan.Levels[0].StepIDs), time.Second, t)

	cancel()

	execErr := awaitCancellationResult(resultCh, 5*time.Second, t)

	assertCancellationError(execErr, t)

	waitForGoroutinesToSettleWithin(startG, 5*time.Second, t)

	assertTempFilesCleaned(tempDir, &tempFilesMu, tempFiles, t)
}

func createLongRunningPlugin(tempDir string, startedCh chan<- string, files *[]string, mu *sync.Mutex) *executorStubPlugin {
	return &executorStubPlugin{
		meta:          domainplugin.Metadata{ID: "command", Name: "Command", Type: domainplugin.TypeCommand, Version: "1.0.0"},
		requireAction: true,
		onApply: func(ctx context.Context, step domainpipeline.Step) error {
			tmpFile, err := os.CreateTemp(tempDir, fmt.Sprintf("%s-*", step.ID))
			if err != nil {
				return fmt.Errorf("create temp file for step %s: %w", step.ID, err)
			}

			if err := tmpFile.Close(); err != nil {
				return fmt.Errorf("close temp file for step %s: %w", step.ID, err)
			}

			defer func() { _ = os.Remove(tmpFile.Name()) }()

			mu.Lock()

			*files = append(*files, tmpFile.Name())

			mu.Unlock()

			select {
			case startedCh <- step.ID:
			default:
			}

			select {
			case <-ctx.Done():
				return fmt.Errorf("step %s cancelled: %w", step.ID, ctx.Err())
			case <-time.After(5 * time.Second):
				return nil
			}
		},
	}
}

func waitForStepsToStart(started <-chan string, count int, timeout time.Duration, t *testing.T) {
	t.Helper()

	for i := 0; i < count; i++ {
		select {
		case <-started:
		case <-time.After(timeout):
			t.Fatalf("timed out waiting for step %d to start", i)
		}
	}
}

func awaitCancellationResult(results <-chan error, timeout time.Duration, t *testing.T) error {
	t.Helper()

	select {
	case err := <-results:
		return err
	case <-time.After(timeout):
		t.Fatal("executor did not return within cancellation timeout")
	}

	return nil
}

func assertCancellationError(err error, t *testing.T) {
	t.Helper()

	if err == nil {
		t.Fatal("expected cancellation error")
	}

	var domainErr *domainpipeline.DomainError
	if !errors.As(err, &domainErr) || domainErr.Code != domainpipeline.ErrCodeCancelled {
		t.Fatalf("expected cancellation domain error, got %v", err)
	}
}

func assertTempFilesCleaned(tempDir string, mu *sync.Mutex, files []string, t *testing.T) {
	t.Helper()

	mu.Lock()
	defer mu.Unlock()

	if len(files) == 0 {
		t.Fatal("expected temp files to be created during execution")
	}

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("read temp dir: %v", err)
	}

	if len(entries) != 0 {
		t.Fatalf("expected temp files to be cleaned up, found %d leftovers", len(entries))
	}
}

func waitForGoroutinesToSettle(initial int, t *testing.T) {
	t.Helper()
	waitForGoroutinesToSettleWithin(initial, 200*time.Millisecond, t)
}

func waitForGoroutinesToSettleWithin(initial int, timeout time.Duration, t *testing.T) {
	t.Helper()

	deadline := time.Now().Add(timeout)

	for {
		current := runtime.NumGoroutine()
		if current <= initial {
			return
		}

		if time.Now().After(deadline) {
			t.Fatalf("goroutines did not settle: initial=%d current=%d", initial, current)
		}

		time.Sleep(5 * time.Millisecond)
	}
}

var _ ports.Plugin = (*executorStubPlugin)(nil)

type stubEventPublisher struct {
	mu     sync.Mutex
	events []ports.DomainEvent
}

func (s *stubEventPublisher) Publish(_ context.Context, event ports.DomainEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events = append(s.events, event)

	return nil
}

func (s *stubEventPublisher) Subscribe(string, ports.EventHandler) (ports.Subscription, error) {
	return noopSubscription{}, nil
}

func (s *stubEventPublisher) contains(eventType string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, evt := range s.events {
		if evt.EventType() == eventType {
			return true
		}
	}

	return false
}

type noopSubscription struct{}

func (noopSubscription) Unsubscribe() {}
