package engine

import (
	"context"
	"errors"
	"sync"
	"testing"

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
}

func (s *executorStubPlugin) Metadata() domainplugin.Metadata { return s.meta }

func (s *executorStubPlugin) Evaluate(ctx context.Context, step domainpipeline.Step) (*domainpipeline.EvaluationResult, error) {
	return &domainpipeline.EvaluationResult{RequiresAction: s.requireAction, Diff: "diff"}, nil
}

func (s *executorStubPlugin) Apply(ctx context.Context, eval *domainpipeline.EvaluationResult, step domainpipeline.Step) (*domainpipeline.StepResult, error) {
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
