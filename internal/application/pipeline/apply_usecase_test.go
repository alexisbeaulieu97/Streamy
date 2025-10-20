package pipeline

import (
	"context"
	"errors"
	"testing"

	require "github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/application/pipeline/testutil"
	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

func TestApplyUseCase_Success(t *testing.T) {
	t.Parallel()

	pip := &domainpipeline.Pipeline{
		Name: "demo",
		Steps: []domainpipeline.Step{
			{ID: "setup", Type: domainpipeline.StepType("command"), Enabled: true},
		},
		Validations: []domainpipeline.Validation{
			{Type: "file_exists"},
		},
	}
	plan := &domainpipeline.ExecutionPlan{
		Levels: []domainpipeline.ExecutionLevel{{Level: 0, StepIDs: []string{"setup"}}},
	}

	loader := &testutil.MockConfigLoader{
		LoadFunc: func(_ context.Context, _ string) (*domainpipeline.Pipeline, error) {
			return pip, nil
		},
	}
	builder := &testutil.MockDAGBuilder{
		BuildFunc: func(_ context.Context, _ []domainpipeline.Step) (*domainpipeline.ExecutionPlan, error) {
			return plan, nil
		},
	}
	logger := testutil.NewMockLogger()
	eventPublisher := &testutil.MockEventPublisher{}
	prepareUC := NewPrepareUseCase(loader, builder, logger, testutil.NewMockTracer(), eventPublisher)

	executor := &testutil.MockPluginExecutor{
		ExecuteFunc: func(_ context.Context, _ *domainpipeline.ExecutionPlan, _ *domainpipeline.Pipeline) ([]domainpipeline.StepResult, error) {
			return []domainpipeline.StepResult{
				{StepID: "setup", Status: domainpipeline.StatusSuccess},
			}, nil
		},
	}
	validator := &testutil.MockValidationService{
		RunFunc: func(_ context.Context, _ []domainpipeline.Validation) (domainpipeline.VerificationSummary, error) {
			return domainpipeline.VerificationSummary{
				TotalChecks:  1,
				PassedChecks: 1,
			}, nil
		},
	}
	metrics := testutil.NewMockMetricsCollector()
	tracer := testutil.NewMockTracer()

	applyUC := NewApplyUseCase(prepareUC, executor, validator, logger, metrics, tracer, eventPublisher)

	ctx := context.Background()
	returnedPipeline, results, summary, err := applyUC.Apply(ctx, "pipeline.yaml", false)
	require.NoError(t, err)
	require.Equal(t, pip, returnedPipeline)
	require.Len(t, results, 1)
	require.NotNil(t, summary)

	require.NotEmpty(t, eventPublisher.Published)
	require.True(t, containsEvent(eventPublisher.Published, ports.EventPipelineStarted))
	require.True(t, containsEvent(eventPublisher.Published, ports.EventPipelineCompleted))
	require.True(t, containsEvent(eventPublisher.Published, ports.EventValidationCompleted))
	require.Len(t, validator.Calls, 1)
	require.Len(t, executor.ExecuteCalls, 1)
}

func TestApplyUseCase_ExecutorFailure(t *testing.T) {
	t.Parallel()

	pip := &domainpipeline.Pipeline{
		Name: "demo",
		Steps: []domainpipeline.Step{
			{ID: "setup", Type: domainpipeline.StepType("command"), Enabled: true},
		},
	}
	plan := &domainpipeline.ExecutionPlan{
		Levels: []domainpipeline.ExecutionLevel{{Level: 0, StepIDs: []string{"setup"}}},
	}
	execErr := errors.New("executor failed")

	loader := &testutil.MockConfigLoader{
		LoadFunc: func(_ context.Context, _ string) (*domainpipeline.Pipeline, error) {
			return pip, nil
		},
	}
	builder := &testutil.MockDAGBuilder{
		BuildFunc: func(_ context.Context, _ []domainpipeline.Step) (*domainpipeline.ExecutionPlan, error) {
			return plan, nil
		},
	}
	logger := testutil.NewMockLogger()
	events := &testutil.MockEventPublisher{}
	prepareUC := NewPrepareUseCase(loader, builder, logger, testutil.NewMockTracer(), events)

	executor := &testutil.MockPluginExecutor{
		ExecuteFunc: func(_ context.Context, _ *domainpipeline.ExecutionPlan, _ *domainpipeline.Pipeline) ([]domainpipeline.StepResult, error) {
			results := []domainpipeline.StepResult{
				{StepID: "setup", Status: domainpipeline.StatusFailure},
			}

			return results, execErr
		},
	}
	validator := &testutil.MockValidationService{
		RunFunc: func(_ context.Context, _ []domainpipeline.Validation) (domainpipeline.VerificationSummary, error) {
			t.Fatal("validator should not run on execution failure")
			return domainpipeline.VerificationSummary{}, nil
		},
	}
	applyUC := NewApplyUseCase(prepareUC, executor, validator, logger, testutil.NewMockMetricsCollector(), testutil.NewMockTracer(), events)

	returnedPipeline, results, summary, err := applyUC.Apply(context.Background(), "pipeline.yaml", false)
	require.ErrorIs(t, err, execErr)
	require.Contains(t, err.Error(), "execute pipeline")
	require.Contains(t, err.Error(), "Hint:")
	require.Equal(t, pip, returnedPipeline)
	require.Len(t, results, 1)
	require.Nil(t, summary)

	require.True(t, containsEvent(events.Published, ports.EventPipelineFailed))
	require.Empty(t, validator.Calls)
}

func TestApplyUseCase_ContextCancellation(t *testing.T) {
	t.Parallel()

	pip := &domainpipeline.Pipeline{
		Name: "demo",
		Steps: []domainpipeline.Step{
			{ID: "setup", Type: domainpipeline.StepType("command"), Enabled: true},
		},
	}
	plan := &domainpipeline.ExecutionPlan{
		Levels: []domainpipeline.ExecutionLevel{{Level: 0, StepIDs: []string{"setup"}}},
	}

	loader := &testutil.MockConfigLoader{
		LoadFunc: func(_ context.Context, _ string) (*domainpipeline.Pipeline, error) {
			return pip, nil
		},
	}
	builder := &testutil.MockDAGBuilder{
		BuildFunc: func(_ context.Context, _ []domainpipeline.Step) (*domainpipeline.ExecutionPlan, error) {
			return plan, nil
		},
	}
	logger := testutil.NewMockLogger()
	events := &testutil.MockEventPublisher{}
	prepareUC := NewPrepareUseCase(loader, builder, logger, testutil.NewMockTracer(), events)

	executor := &testutil.MockPluginExecutor{
		ExecuteFunc: func(_ context.Context, _ *domainpipeline.ExecutionPlan, _ *domainpipeline.Pipeline) ([]domainpipeline.StepResult, error) {
			return nil, context.Canceled
		},
	}
	validator := &testutil.MockValidationService{}
	applyUC := NewApplyUseCase(prepareUC, executor, validator, logger, testutil.NewMockMetricsCollector(), testutil.NewMockTracer(), events)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	returnedPipeline, results, summary, err := applyUC.Apply(ctx, "pipeline.yaml", false)
	require.ErrorIs(t, err, context.Canceled)
	require.Contains(t, err.Error(), "execute pipeline")
	require.Equal(t, pip, returnedPipeline)
	require.Nil(t, results)
	require.Nil(t, summary)
	require.True(t, containsEvent(events.Published, ports.EventPipelineFailed))
}

func TestApplyUseCase_DryRunSkipsValidation(t *testing.T) {
	t.Parallel()

	pip := &domainpipeline.Pipeline{
		Name: "demo",
		Steps: []domainpipeline.Step{
			{ID: "setup", Type: domainpipeline.StepType("command"), Enabled: true},
		},
	}
	plan := &domainpipeline.ExecutionPlan{
		Levels: []domainpipeline.ExecutionLevel{{Level: 0, StepIDs: []string{"setup"}}},
	}

	loader := &testutil.MockConfigLoader{
		LoadFunc: func(_ context.Context, _ string) (*domainpipeline.Pipeline, error) {
			return pip, nil
		},
	}
	builder := &testutil.MockDAGBuilder{
		BuildFunc: func(_ context.Context, _ []domainpipeline.Step) (*domainpipeline.ExecutionPlan, error) {
			return plan, nil
		},
	}
	logger := testutil.NewMockLogger()
	events := &testutil.MockEventPublisher{}
	prepareUC := NewPrepareUseCase(loader, builder, logger, testutil.NewMockTracer(), events)

	executor := &testutil.MockPluginExecutor{
		ExecuteFunc: func(_ context.Context, _ *domainpipeline.ExecutionPlan, _ *domainpipeline.Pipeline) ([]domainpipeline.StepResult, error) {
			return []domainpipeline.StepResult{
				{StepID: "setup", Status: domainpipeline.StatusSuccess},
			}, nil
		},
	}
	validator := &testutil.MockValidationService{}

	applyUC := NewApplyUseCase(prepareUC, executor, validator, logger, testutil.NewMockMetricsCollector(), testutil.NewMockTracer(), events)

	returnedPipeline, results, summary, err := applyUC.Apply(context.Background(), "pipeline.yaml", true)
	require.NoError(t, err)
	require.Equal(t, pip, returnedPipeline)
	require.Len(t, results, 1)
	require.Nil(t, summary)
	require.Empty(t, validator.Calls, "dry run should skip validation")

	require.True(t, containsEvent(events.Published, ports.EventPipelineCompleted))
}

func TestApplyUseCase_ValidationFailure(t *testing.T) {
	t.Parallel()

	pipelineDefinition := &domainpipeline.Pipeline{
		Name: "demo",
		Steps: []domainpipeline.Step{
			{ID: "setup", Type: domainpipeline.StepType("command"), Enabled: true},
		},
	}
	plan := &domainpipeline.ExecutionPlan{
		Levels: []domainpipeline.ExecutionLevel{{Level: 0, StepIDs: []string{"setup"}}},
	}

	loader := &testutil.MockConfigLoader{
		LoadFunc: func(_ context.Context, _ string) (*domainpipeline.Pipeline, error) {
			return pipelineDefinition, nil
		},
	}
	builder := &testutil.MockDAGBuilder{
		BuildFunc: func(_ context.Context, _ []domainpipeline.Step) (*domainpipeline.ExecutionPlan, error) {
			return plan, nil
		},
	}
	logger := testutil.NewMockLogger()
	events := &testutil.MockEventPublisher{}

	publishErr := errors.New("publish failed")
	events.PublishFunc = func(ctx context.Context, event ports.DomainEvent) error {
		events.Published = append(events.Published, testutil.EventRecord{
			Ctx:     ctx,
			Event:   event,
			Type:    event.EventType(),
			Payload: event.Payload(),
		})
		if event.EventType() == ports.EventValidationFailed {
			return publishErr
		}

		return nil
	}

	prepareUC := NewPrepareUseCase(loader, builder, logger, testutil.NewMockTracer(), events)

	executor := &testutil.MockPluginExecutor{
		ExecuteFunc: func(_ context.Context, _ *domainpipeline.ExecutionPlan, _ *domainpipeline.Pipeline) ([]domainpipeline.StepResult, error) {
			return []domainpipeline.StepResult{
				{StepID: "setup", Status: domainpipeline.StatusFailure, Error: &domainpipeline.DomainError{Message: "failed"}},
				{StepID: "noop", Status: domainpipeline.StatusSkipped},
			}, nil
		},
	}

	validationErr := errors.New("validation failed")
	validator := &testutil.MockValidationService{
		RunFunc: func(_ context.Context, _ []domainpipeline.Validation) (domainpipeline.VerificationSummary, error) {
			return domainpipeline.VerificationSummary{
				TotalChecks:  1,
				FailedChecks: 1,
				Results: []domainpipeline.VerificationResult{
					{StepID: "check", Status: domainpipeline.VerificationFailed},
				},
			}, validationErr
		},
	}
	applyUC := NewApplyUseCase(prepareUC, executor, validator, logger, testutil.NewMockMetricsCollector(), testutil.NewMockTracer(), events)

	returnedPipeline, results, summary, err := applyUC.Apply(context.Background(), "pipeline.yaml", false)
	require.ErrorIs(t, err, validationErr)
	require.Equal(t, pipelineDefinition, returnedPipeline)
	require.Len(t, results, 2)
	require.NotNil(t, summary)
	require.Equal(t, 1, summary.FailedChecks)

	require.True(t, containsEvent(events.Published, ports.EventValidationFailed))
	require.True(t, containsEvent(events.Published, ports.EventStepFailed))
	require.True(t, containsEvent(events.Published, ports.EventStepSkipped))

	warnSeen := false

	for _, entry := range logger.Entries() {
		if entry.Level == "warn" && entry.Msg == "failed to publish domain event" {
			warnSeen = true
			break
		}
	}

	require.True(t, warnSeen, "expected warning when publish fails")
}

func TestApplyUseCase_PrepareFailure(t *testing.T) {
	t.Parallel()

	prepareErr := errors.New("prepare failed")
	loader := &testutil.MockConfigLoader{
		LoadFunc: func(_ context.Context, _ string) (*domainpipeline.Pipeline, error) {
			return nil, prepareErr
		},
	}
	builder := &testutil.MockDAGBuilder{}
	logger := testutil.NewMockLogger()
	events := &testutil.MockEventPublisher{}

	prepareUC := NewPrepareUseCase(loader, builder, logger, testutil.NewMockTracer(), events)
	executor := &testutil.MockPluginExecutor{}
	validator := &testutil.MockValidationService{}

	applyUC := NewApplyUseCase(prepareUC, executor, validator, logger, testutil.NewMockMetricsCollector(), testutil.NewMockTracer(), events)

	pip, results, summary, err := applyUC.Apply(context.Background(), "pipeline.yaml", false)
	require.ErrorIs(t, err, prepareErr)
	require.Nil(t, pip)
	require.Nil(t, results)
	require.Nil(t, summary)
	require.True(t, containsEvent(events.Published, ports.EventPipelineFailed))
}

func TestApplyUseCase_NoValidatorConfigured(t *testing.T) {
	t.Parallel()

	pipelineDefinition := &domainpipeline.Pipeline{
		Name:  "demo",
		Steps: []domainpipeline.Step{{ID: "setup", Type: domainpipeline.StepType("command"), Enabled: true}},
	}
	plan := &domainpipeline.ExecutionPlan{Levels: []domainpipeline.ExecutionLevel{{Level: 0, StepIDs: []string{"setup"}}}}

	loader := &testutil.MockConfigLoader{
		LoadFunc: func(_ context.Context, _ string) (*domainpipeline.Pipeline, error) {
			return pipelineDefinition, nil
		},
	}
	builder := &testutil.MockDAGBuilder{
		BuildFunc: func(_ context.Context, _ []domainpipeline.Step) (*domainpipeline.ExecutionPlan, error) {
			return plan, nil
		},
	}
	logger := testutil.NewMockLogger()
	events := &testutil.MockEventPublisher{}

	prepareUC := NewPrepareUseCase(loader, builder, logger, testutil.NewMockTracer(), events)
	executor := &testutil.MockPluginExecutor{
		ExecuteFunc: func(_ context.Context, _ *domainpipeline.ExecutionPlan, _ *domainpipeline.Pipeline) ([]domainpipeline.StepResult, error) {
			return []domainpipeline.StepResult{{StepID: "setup", Status: domainpipeline.StatusSuccess}}, nil
		},
	}

	applyUC := NewApplyUseCase(prepareUC, executor, nil, logger, testutil.NewMockMetricsCollector(), testutil.NewMockTracer(), events)

	pip, results, summary, err := applyUC.Apply(context.Background(), "pipeline.yaml", false)
	require.NoError(t, err)
	require.Equal(t, pipelineDefinition, pip)
	require.Len(t, results, 1)
	require.Nil(t, summary)

	require.True(t, containsEvent(events.Published, ports.EventPipelineCompleted))

	debugSeen := false

	for _, entry := range logger.Entries() {
		if entry.Level == "debug" && entry.Msg == "no validation service configured" {
			debugSeen = true
			break
		}
	}

	require.True(t, debugSeen, "expected debug log when validator is nil")
}

func containsEvent(events []testutil.EventRecord, eventType string) bool {
	for _, evt := range events {
		if evt.Type == eventType {
			return true
		}
	}

	return false
}
