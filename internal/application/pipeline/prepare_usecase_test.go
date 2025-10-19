package pipeline

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/application/pipeline/testutil"
	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

func TestPrepareUseCase_Success(t *testing.T) {
	t.Parallel()

	pipelineDefinition := &domainpipeline.Pipeline{
		Name: "demo",
		Steps: []domainpipeline.Step{
			{ID: "setup", Type: domainpipeline.StepType("command"), Enabled: true},
		},
	}

	configLoader := &testutil.MockConfigLoader{
		LoadFunc: func(ctx context.Context, path string) (*domainpipeline.Pipeline, error) {
			return pipelineDefinition, nil
		},
	}
	dagBuilder := &testutil.MockDAGBuilder{
		BuildFunc: func(ctx context.Context, steps []domainpipeline.Step) (*domainpipeline.ExecutionPlan, error) {
			return &domainpipeline.ExecutionPlan{
				Levels: []domainpipeline.ExecutionLevel{{Level: 0, StepIDs: []string{"setup"}}},
			}, nil
		},
	}
	logger := testutil.NewMockLogger()
	events := &testutil.MockEventPublisher{}

	tracer := testutil.NewMockTracer()

	useCase := NewPrepareUseCase(configLoader, dagBuilder, logger, tracer, events)

	pip, plan, err := useCase.Prepare(context.Background(), "pipeline.yaml")
	require.NoError(t, err)
	require.Equal(t, pipelineDefinition, pip)
	require.NotNil(t, plan)

	require.Len(t, dagBuilder.Calls, 1)
	require.Len(t, events.Published, 2)
	require.Equal(t, ports.EventPipelineStarted, events.Published[0].Type)
	require.Equal(t, ports.EventPipelineCompleted, events.Published[1].Type)
}

func TestPrepareUseCase_LoadFailure(t *testing.T) {
	t.Parallel()

	loadErr := errors.New("load failed")
	configLoader := &testutil.MockConfigLoader{
		LoadFunc: func(ctx context.Context, path string) (*domainpipeline.Pipeline, error) {
			return nil, loadErr
		},
	}
	dagBuilder := &testutil.MockDAGBuilder{}
	logger := testutil.NewMockLogger()
	events := &testutil.MockEventPublisher{}

	useCase := NewPrepareUseCase(configLoader, dagBuilder, logger, testutil.NewMockTracer(), events)

	pip, plan, err := useCase.Prepare(context.Background(), "pipeline.yaml")
	require.ErrorIs(t, err, loadErr)
	require.Contains(t, err.Error(), prepareHint)
	require.Nil(t, pip)
	require.Nil(t, plan)
	require.Empty(t, dagBuilder.Calls)
	require.Len(t, events.Published, 2)
}

func TestPrepareUseCase_DagBuildFailure(t *testing.T) {
	t.Parallel()

	pipelineDefinition := &domainpipeline.Pipeline{
		Name: "demo",
		Steps: []domainpipeline.Step{
			{ID: "setup", Type: domainpipeline.StepType("command"), Enabled: true},
		},
	}
	buildErr := errors.New("cycle detected")

	configLoader := &testutil.MockConfigLoader{
		LoadFunc: func(ctx context.Context, path string) (*domainpipeline.Pipeline, error) {
			return pipelineDefinition, nil
		},
	}
	dagBuilder := &testutil.MockDAGBuilder{
		BuildFunc: func(ctx context.Context, steps []domainpipeline.Step) (*domainpipeline.ExecutionPlan, error) {
			return nil, buildErr
		},
	}
	logger := testutil.NewMockLogger()
	events := &testutil.MockEventPublisher{}

	useCase := NewPrepareUseCase(configLoader, dagBuilder, logger, testutil.NewMockTracer(), events)

	pip, plan, err := useCase.Prepare(context.Background(), "pipeline.yaml")
	require.ErrorIs(t, err, buildErr)
	require.Contains(t, err.Error(), dagBuildHint)
	require.Equal(t, pipelineDefinition, pip)
	require.Nil(t, plan)
	require.Len(t, events.Published, 1)
}

func TestPrepareUseCase_DagValidationFailure(t *testing.T) {
	t.Parallel()

	pipelineDefinition := &domainpipeline.Pipeline{
		Name: "demo",
		Steps: []domainpipeline.Step{
			{ID: "setup", Type: domainpipeline.StepType("command"), Enabled: true},
		},
	}

	configLoader := &testutil.MockConfigLoader{
		LoadFunc: func(ctx context.Context, path string) (*domainpipeline.Pipeline, error) {
			return pipelineDefinition, nil
		},
	}
	dagBuilder := &testutil.MockDAGBuilder{
		BuildFunc: func(ctx context.Context, steps []domainpipeline.Step) (*domainpipeline.ExecutionPlan, error) {
			return &domainpipeline.ExecutionPlan{
				Levels: []domainpipeline.ExecutionLevel{{Level: 0, StepIDs: []string{}}},
			}, nil
		},
	}
	logger := testutil.NewMockLogger()
	events := &testutil.MockEventPublisher{}

	useCase := NewPrepareUseCase(configLoader, dagBuilder, logger, testutil.NewMockTracer(), events)

	pip, plan, err := useCase.Prepare(context.Background(), "pipeline.yaml")
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), dagValidateHint))
	require.Equal(t, pipelineDefinition, pip)
	require.Nil(t, plan)
	require.Len(t, events.Published, 2)
}
