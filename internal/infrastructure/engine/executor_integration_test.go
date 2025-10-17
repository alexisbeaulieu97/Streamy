package engine

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	configinfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/config"
	logginginfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/logging"
	plugininfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/plugin"
	commandplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/command"
)

func TestExecutorSwapProducesIdenticalResults(t *testing.T) {
	ctx := context.Background()
	pipeline := loadTestPipeline(t, filepath.Join("testdata", "configs", "simple.yaml"))
	plan := buildTestPlan(t, pipeline)

	defaultRegistry := plugininfra.NewRegistry()
	require.NoError(t, defaultRegistry.Register(commandplugin.NewPort()))

	altRegistry := plugininfra.NewRegistry()
	require.NoError(t, altRegistry.Register(commandplugin.NewPort()))

	defaultExec := NewExecutor(defaultRegistry, WithExecutorLogger(logginginfra.NewNoOpLogger()))
	altExec := NewTestExecutor(altRegistry)

	pipelineA := pipeline.Clone()
	resultsDefault, err := defaultExec.Execute(ctx, plan, &pipelineA)
	require.NoError(t, err)

	pipelineB := pipeline.Clone()
	resultsAlt, err := altExec.Execute(ctx, plan, &pipelineB)
	require.NoError(t, err)

	require.Equal(t, normalizeStepResults(resultsDefault), normalizeStepResults(resultsAlt))

	verifyDefault, err := defaultExec.Verify(ctx, &pipelineA)
	require.NoError(t, err)

	verifyAlt, err := altExec.Verify(ctx, &pipelineB)
	require.NoError(t, err)

	require.Equal(t, verifyDefault, verifyAlt)
}

func loadTestPipeline(t *testing.T, configPath string) *domainpipeline.Pipeline {
	t.Helper()
	loader := configinfra.NewYAMLLoader(logginginfra.NewNoOpLogger())
	absPath := filepath.Join("..", "..", "..", configPath)
	pipeline, err := loader.Load(context.Background(), absPath)
	require.NoError(t, err)
	return pipeline
}

func buildTestPlan(t *testing.T, pipeline *domainpipeline.Pipeline) *domainpipeline.ExecutionPlan {
	t.Helper()
	builder := NewDAGBuilder()
	plan, err := builder.Build(context.Background(), pipeline.Steps)
	require.NoError(t, err)
	require.NoError(t, plan.Validate(*pipeline))
	return plan
}

func normalizeStepResults(results []domainpipeline.StepResult) []domainpipeline.StepResult {
	normalized := make([]domainpipeline.StepResult, len(results))
	for i, res := range results {
		res.Duration = 0
		normalized[i] = res
	}
	return normalized
}
