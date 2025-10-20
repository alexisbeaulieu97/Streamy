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

func TestVerifyUseCase_Success(t *testing.T) {
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
		VerifyFunc: func(_ context.Context, _ *domainpipeline.Pipeline) ([]domainpipeline.VerificationResult, error) {
			return []domainpipeline.VerificationResult{{StepID: "setup", Status: domainpipeline.VerificationSatisfied}}, nil
		},
	}
	verifyUC := NewVerifyUseCase(prepareUC, executor, logger, testutil.NewMockMetricsCollector(), testutil.NewMockTracer(), events)

	pip, results, err := verifyUC.Verify(context.Background(), "pipeline.yaml")
	require.NoError(t, err)
	require.Equal(t, pipelineDefinition, pip)
	require.Len(t, results, 1)
	require.True(t, containsEvent(events.Published, ports.EventValidationCompleted))
}

func TestVerifyUseCase_PrepareFailure(t *testing.T) {
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
	verifyUC := NewVerifyUseCase(prepareUC, executor, logger, testutil.NewMockMetricsCollector(), testutil.NewMockTracer(), events)

	pip, results, err := verifyUC.Verify(context.Background(), "pipeline.yaml")
	require.ErrorIs(t, err, prepareErr)
	require.Contains(t, err.Error(), prepareHint)
	require.Nil(t, pip)
	require.Nil(t, results)
	require.True(t, containsEvent(events.Published, ports.EventValidationFailed))
}

func TestVerifyUseCase_ExecutorFailure(t *testing.T) {
	t.Parallel()

	pipelineDefinition := &domainpipeline.Pipeline{
		Name:  "demo",
		Steps: []domainpipeline.Step{{ID: "setup", Type: domainpipeline.StepType("command"), Enabled: true}},
	}

	loader := &testutil.MockConfigLoader{
		LoadFunc: func(_ context.Context, _ string) (*domainpipeline.Pipeline, error) {
			return pipelineDefinition, nil
		},
	}
	builder := &testutil.MockDAGBuilder{
		BuildFunc: func(_ context.Context, _ []domainpipeline.Step) (*domainpipeline.ExecutionPlan, error) {
			return &domainpipeline.ExecutionPlan{Levels: []domainpipeline.ExecutionLevel{{Level: 0, StepIDs: []string{"setup"}}}}, nil
		},
	}
	logger := testutil.NewMockLogger()
	events := &testutil.MockEventPublisher{}
	prepareUC := NewPrepareUseCase(loader, builder, logger, testutil.NewMockTracer(), events)

	execErr := errors.New("verification failed")
	executor := &testutil.MockPluginExecutor{
		VerifyFunc: func(_ context.Context, _ *domainpipeline.Pipeline) ([]domainpipeline.VerificationResult, error) {
			return nil, execErr
		},
	}
	verifyUC := NewVerifyUseCase(prepareUC, executor, logger, testutil.NewMockMetricsCollector(), testutil.NewMockTracer(), events)

	pip, results, err := verifyUC.Verify(context.Background(), "pipeline.yaml")
	require.ErrorIs(t, err, execErr)
	require.Contains(t, err.Error(), verifyHint)
	require.Equal(t, pipelineDefinition, pip)
	require.Nil(t, results)
	require.True(t, containsEvent(events.Published, ports.EventValidationFailed))
}
