package tests

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	streamconfig "github.com/alexisbeaulieu97/streamy/internal/config"
	streamengine "github.com/alexisbeaulieu97/streamy/internal/engine"
	streamlogger "github.com/alexisbeaulieu97/streamy/internal/logger"
	streammodel "github.com/alexisbeaulieu97/streamy/internal/model"
	streamplugin "github.com/alexisbeaulieu97/streamy/internal/plugin"
	streamvalidation "github.com/alexisbeaulieu97/streamy/internal/validation"

	commandplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/command"
	copyplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/copy"
	packageplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/package"
	repoplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/repo"
	symlinkplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/symlink"
)

// testRegistry creates a registry with all built-in plugins registered
func testRegistry(t *testing.T, logger *streamlogger.Logger) *streamplugin.PluginRegistry {
	t.Helper()
	registry := streamplugin.NewPluginRegistry(streamplugin.DefaultConfig(), logger)
	require.NoError(t, registry.Register(commandplugin.New()))
	require.NoError(t, registry.Register(copyplugin.New()))
	require.NoError(t, registry.Register(packageplugin.New()))
	require.NoError(t, registry.Register(repoplugin.New()))
	require.NoError(t, registry.Register(symlinkplugin.New()))
	require.NoError(t, registry.ValidateDependencies())
	require.NoError(t, registry.InitializePlugins())
	return registry
}

func TestIntegrationSimpleExecution(t *testing.T) {
	cfg := loadConfig(t, "simple.yaml")
	graph := buildDAG(t, cfg)
	plan := generatePlan(t, graph)

	logger := testLogger(t)
	registry := testRegistry(t, logger)
	ctx := &streamengine.ExecutionContext{
		Config:     cfg,
		DryRun:     false,
		WorkerPool: make(chan struct{}, 2),
		Results:    make(map[string]*streammodel.StepResult),
		Logger:     logger,
		Context:    context.Background(),
		Registry:   registry,
	}

	results, err := streamengine.Execute(ctx, plan)
	require.NoError(t, err)
	require.Len(t, results, len(cfg.Steps))

	for _, res := range results {
		require.Equal(t, streammodel.StatusSuccess, res.Status)
	}

	validations, err := streamvalidation.RunValidations(context.Background(), cfg.Validations)
	require.NoError(t, err)
	require.NotEmpty(t, validations)
}

func TestIntegrationComplexPlan(t *testing.T) {
	cfg := loadConfig(t, "complex.yaml")
	graph := buildDAG(t, cfg)
	plan := generatePlan(t, graph)

	require.GreaterOrEqual(t, len(plan.Levels), 3)
	require.Contains(t, plan.Levels[0].StepIDs, "install_tools")
	require.Contains(t, plan.Levels[1].StepIDs, "fetch_repo")
}

func TestIntegrationDryRunSkipsExecution(t *testing.T) {
	cfg := loadConfig(t, "simple.yaml")
	graph := buildDAG(t, cfg)
	plan := generatePlan(t, graph)

	logger := testLogger(t)
	registry := testRegistry(t, logger)
	ctx := &streamengine.ExecutionContext{
		Config:     cfg,
		DryRun:     true,
		WorkerPool: make(chan struct{}, 2),
		Results:    make(map[string]*streammodel.StepResult),
		Logger:     logger,
		Context:    context.Background(),
		Registry:   registry,
	}

	results, err := streamengine.Execute(ctx, plan)
	require.NoError(t, err)
	require.Len(t, results, len(cfg.Steps))
	for _, res := range results {
		// With new Evaluate/Apply interface, dry run shows what would be updated
		require.Equal(t, streammodel.StatusWouldUpdate, res.Status)
	}
}

func TestIntegrationIdempotentRuns(t *testing.T) {
	cfg := loadConfig(t, "simple.yaml")
	graph := buildDAG(t, cfg)
	plan := generatePlan(t, graph)

	logger := testLogger(t)
	registry := testRegistry(t, logger)
	ctx := &streamengine.ExecutionContext{
		Config:     cfg,
		DryRun:     false,
		WorkerPool: make(chan struct{}, 2),
		Results:    make(map[string]*streammodel.StepResult),
		Logger:     logger,
		Context:    context.Background(),
		Registry:   registry,
	}

	_, err := streamengine.Execute(ctx, plan)
	require.NoError(t, err)

	ctx.Results = make(map[string]*streammodel.StepResult)
	_, err = streamengine.Execute(ctx, plan)
	require.NoError(t, err)
}

func TestIntegrationErrorHandling(t *testing.T) {
	cfg := &streamconfig.Config{
		Version: "1.0",
		Name:    "Fails",
		Steps: []streamconfig.Step{
			{
				ID:      "fail",
				Type:    "command",
				Enabled: true,
				Command: &streamconfig.CommandStep{
					Command: "__streamy_fail__",
				},
			},
		},
	}

	graph := buildDAG(t, cfg)
	plan := generatePlan(t, graph)

	logger := testLogger(t)
	registry := testRegistry(t, logger)
	ctx := &streamengine.ExecutionContext{
		Config:     cfg,
		DryRun:     false,
		WorkerPool: make(chan struct{}, 1),
		Results:    make(map[string]*streammodel.StepResult),
		Logger:     logger,
		Context:    context.Background(),
		Registry:   registry,
	}

	results, err := streamengine.Execute(ctx, plan)
	require.Error(t, err)

	if len(results) == 0 {
		res, ok := ctx.Results["fail"]
		require.True(t, ok, "expected result for failing step")
		require.Equal(t, streammodel.StatusFailed, res.Status)
	} else {
		require.Equal(t, streammodel.StatusFailed, results[0].Status)
	}
}

func TestIntegrationValidationFailure(t *testing.T) {
	cfg := loadConfig(t, "simple.yaml")
	validation := streamconfig.Validation{
		Type: "file_exists",
		FileExists: &streamconfig.FileExistsValidation{
			Path: filepath.Join(t.TempDir(), "missing.txt"),
		},
	}

	results, err := streamvalidation.RunValidations(context.Background(), append(cfg.Validations, validation))
	require.Error(t, err)
	require.Len(t, results, len(cfg.Validations)+1)
}

func TestIntegrationParseError(t *testing.T) {
	_, err := streamconfig.ParseConfig(fixturePath("invalid.yaml"))
	require.Error(t, err)
}

func TestIntegrationCycleDetection(t *testing.T) {
	_, err := streamconfig.ParseConfig(fixturePath("cycle.yaml"))
	require.Error(t, err)
}

func TestIntegrationMissingReference(t *testing.T) {
	_, err := streamconfig.ParseConfig(fixturePath("missing_ref.yaml"))
	require.Error(t, err)
}

func loadConfig(t *testing.T, name string) *streamconfig.Config {
	t.Helper()
	cfg, err := streamconfig.ParseConfig(fixturePath(name))
	require.NoError(t, err)
	return cfg
}

func buildDAG(t *testing.T, cfg *streamconfig.Config) *streamengine.Graph {
	t.Helper()
	graph, err := streamengine.BuildDAG(cfg.Steps)
	require.NoError(t, err)
	return graph
}

func generatePlan(t *testing.T, graph *streamengine.Graph) *streamengine.ExecutionPlan {
	t.Helper()
	plan, err := streamengine.GeneratePlan(graph)
	require.NoError(t, err)
	return plan
}

func testLogger(t *testing.T) *streamlogger.Logger {
	t.Helper()
	log, err := streamlogger.New(streamlogger.Options{Level: "info", HumanReadable: false})
	require.NoError(t, err)
	return log
}

func fixturePath(name string) string {
	return filepath.Join("..", "testdata", "configs", name)
}
