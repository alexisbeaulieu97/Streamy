package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

type stubCommandPlugin struct {
	requireAction bool
}

func (p *stubCommandPlugin) PluginMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name:    "command",
		Type:    "command",
		Version: "1.0.0",
	}
}

func (p *stubCommandPlugin) Schema() any {
	return config.CommandStep{}
}

func (p *stubCommandPlugin) Evaluate(_ context.Context, step *config.Step) (*model.EvaluationResult, error) {
	return &model.EvaluationResult{
		StepID:         step.ID,
		CurrentState:   model.StatusSatisfied,
		RequiresAction: p.requireAction,
		Message:        "stub evaluation",
	}, nil
}

func (p *stubCommandPlugin) Apply(_ context.Context, _ *model.EvaluationResult, step *config.Step) (*model.StepResult, error) {
	return &model.StepResult{
		StepID:    step.ID,
		Status:    model.StatusSuccess,
		Message:   "stub apply",
		Timestamp: time.Now(),
	}, nil
}

func writeTempConfig(t *testing.T, contents string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "pipeline.yaml")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0644))
	return path
}

const minimalCommandConfig = `version: "1.0.0"
name: "test pipeline"
steps:
  - id: step_one
    type: command
    command: "echo hello"
`

func TestVerifyPipeline_Success(t *testing.T) {
	t.Parallel()

	configPath := writeTempConfig(t, minimalCommandConfig)

	pluginRegistry := plugin.NewPluginRegistry(nil, nil)
	require.NoError(t, pluginRegistry.Register(&stubCommandPlugin{requireAction: false}))

	result, err := VerifyPipeline(context.Background(), configPath, pluginRegistry)
	require.NoError(t, err)
	require.NotNil(t, result)

	require.Equal(t, registry.StatusSatisfied, result.Status)
	require.Equal(t, 1, result.StepCount)
	require.Equal(t, "All 1 steps passed", result.Summary)
}

func TestVerifyPipeline_MissingPlugin(t *testing.T) {
	t.Parallel()

	configPath := writeTempConfig(t, minimalCommandConfig)
	pluginRegistry := plugin.NewPluginRegistry(nil, nil)

	result, err := VerifyPipeline(context.Background(), configPath, pluginRegistry)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, registry.StatusFailed, result.Status)
	require.Equal(t, "1 steps failed or unknown", result.Summary)
}

func TestApplyPipeline_Success(t *testing.T) {
	t.Parallel()

	configPath := writeTempConfig(t, minimalCommandConfig)

	pluginRegistry := plugin.NewPluginRegistry(nil, nil)
	require.NoError(t, pluginRegistry.Register(&stubCommandPlugin{requireAction: true}))

	result, err := ApplyPipeline(context.Background(), configPath, pluginRegistry)
	require.NoError(t, err)
	require.NotNil(t, result)

	require.Equal(t, registry.StatusSatisfied, result.Status)
	require.Equal(t, 1, result.StepCount)
	require.Equal(t, "All 1 steps applied successfully", result.Summary)
}

func TestApplyPipeline_ParseError(t *testing.T) {
	t.Parallel()

	configPath := writeTempConfig(t, "version: invalid\n")
	pluginRegistry := plugin.NewPluginRegistry(nil, nil)

	result, err := ApplyPipeline(context.Background(), configPath, pluginRegistry)
	require.Error(t, err)
	require.Nil(t, result)
}
