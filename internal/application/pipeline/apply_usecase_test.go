package pipeline

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	logginginfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/logging"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

func TestApplyUseCasePublishesEventsWithCorrelationID(t *testing.T) {
	t.Parallel()

	correlationID := "corr-123"
	ctx := logginginfra.WithCorrelationID(context.Background(), correlationID)

	pipelineName := "demo"

	loader := &stubConfigLoader{
		pipeline: &domainpipeline.Pipeline{
			Name: pipelineName,
			Steps: []domainpipeline.Step{{
				ID:      "setup",
				Type:    domainpipeline.StepType("command"),
				Enabled: true,
			}},
		},
	}

	builder := &stubDAGBuilder{}
	executor := &stubExecutor{
		results: []domainpipeline.StepResult{{
			StepID: "setup",
			Status: domainpipeline.StatusSuccess,
		}},
	}
	validator := &stubValidationService{}
	logger := logginginfra.NewNoOpLogger()
	events := &recordingPublisher{}

	prepareUC := NewPrepareUseCase(loader, builder, logger, events)
	applyUC := NewApplyUseCase(prepareUC, executor, validator, logger, events)

	pip, results, summary, err := applyUC.Apply(ctx, "pipeline.yaml", false)
	require.NoError(t, err)
	require.NotNil(t, pip)
	require.Len(t, results, 1)
	require.NotNil(t, summary)

	require.True(t, events.contains(ports.EventPipelineStarted))
	require.True(t, events.contains(ports.EventPipelineCompleted))
	require.True(t, events.contains(ports.EventValidationCompleted))

	for _, evt := range events.events {
		require.Equal(t, correlationID, evt.correlationID)
	}
}

type recordingPublisher struct {
	mu     sync.Mutex
	events []eventRecord
}

type eventRecord struct {
	eventType     string
	payload       map[string]interface{}
	correlationID string
}

func (r *recordingPublisher) Publish(ctx context.Context, event ports.DomainEvent) error {
	if event == nil {
		return nil
	}
	payload := map[string]interface{}{}
	if raw, ok := event.Payload().(map[string]interface{}); ok {
		payload = raw
	}
	record := eventRecord{
		eventType:     event.EventType(),
		payload:       payload,
		correlationID: ports.GetCorrelationID(ctx),
	}
	r.mu.Lock()
	r.events = append(r.events, record)
	r.mu.Unlock()
	return nil
}

func (r *recordingPublisher) Subscribe(string, ports.EventHandler) (ports.Subscription, error) {
	return noopSubscription{}, nil
}

func (r *recordingPublisher) contains(eventType string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, evt := range r.events {
		if evt.eventType == eventType {
			return true
		}
	}
	return false
}

type noopSubscription struct{}

func (noopSubscription) Unsubscribe() {}

type stubConfigLoader struct {
	pipeline *domainpipeline.Pipeline
	err      error
}

func (s *stubConfigLoader) Load(context.Context, string) (*domainpipeline.Pipeline, error) {
	return s.pipeline, s.err
}

func (s *stubConfigLoader) Validate(context.Context, string) error { return nil }

type stubDAGBuilder struct{}

func (s *stubDAGBuilder) Build(context.Context, []domainpipeline.Step) (*domainpipeline.ExecutionPlan, error) {
	return &domainpipeline.ExecutionPlan{Levels: []domainpipeline.ExecutionLevel{{Level: 0, StepIDs: []string{"setup"}}}}, nil
}

type stubExecutor struct {
	results []domainpipeline.StepResult
	err     error
}

func (s *stubExecutor) Execute(context.Context, *domainpipeline.ExecutionPlan, *domainpipeline.Pipeline) ([]domainpipeline.StepResult, error) {
	return s.results, s.err
}

func (s *stubExecutor) Verify(context.Context, *domainpipeline.Pipeline) ([]domainpipeline.VerificationResult, error) {
	return nil, nil
}

type stubValidationService struct{}

func (s *stubValidationService) RunValidations(context.Context, []domainpipeline.Validation) (domainpipeline.VerificationSummary, error) {
	return domainpipeline.VerificationSummary{}, nil
}
