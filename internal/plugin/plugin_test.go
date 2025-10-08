package plugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
)

var _ Plugin = (*testPlugin)(nil)

type testPlugin struct{}

func (p *testPlugin) PluginMetadata() PluginMetadata {
	return PluginMetadata{
		Name:    "test",
		Version: "1.0.0",
		Type:    "command",
	}
}

func (p *testPlugin) Schema() any {
	return struct {
		Command string `yaml:"command"`
	}{}
}

func (p *testPlugin) Evaluate(ctx context.Context, step *config.Step) (*model.EvaluationResult, error) {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &model.EvaluationResult{
		StepID:         step.ID,
		CurrentState:   model.StatusUnknown,
		RequiresAction: true,
		Message:        "test evaluation",
		InternalData:   &struct{ value string }{value: "test-data"},
	}, nil
}

func (p *testPlugin) Apply(ctx context.Context, evalResult *model.EvaluationResult, step *config.Step) (*model.StepResult, error) {
	// Use the evaluation data if available
	if evalResult.InternalData != nil {
		if data, ok := evalResult.InternalData.(*struct{ value string }); ok {
			return &model.StepResult{
				StepID:  step.ID,
				Status:  "success",
				Message: "applied with data: " + data.value,
			}, nil
		}
	}

	return &model.StepResult{
		StepID:  step.ID,
		Status:  "success",
		Message: "applied",
	}, nil
}

func TestPluginMetadata(t *testing.T) {
	p := &testPlugin{}
	meta := p.PluginMetadata()

	require.Equal(t, "test", meta.Name)
	require.Equal(t, "1.0.0", meta.Version)
}

func TestPluginSchemaProvidesTypeInformation(t *testing.T) {
	p := &testPlugin{}
	schema := p.Schema()
	require.NotNil(t, schema)
}

func TestPluginEvaluateAndApply(t *testing.T) {
	p := &testPlugin{}
	step := &config.Step{
		ID:   "run_command",
		Type: "command",
		Command: &config.CommandStep{
			Command: "echo hello",
		},
	}

	// Test evaluation
	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, evalResult.RequiresAction)
	require.Equal(t, model.StatusUnknown, evalResult.CurrentState)
	require.NotNil(t, evalResult.InternalData)

	// Test apply using evaluation result
	applied, err := p.Apply(context.Background(), evalResult, step)
	require.NoError(t, err)
	require.Equal(t, "success", applied.Status)
	require.Equal(t, step.ID, applied.StepID)
	require.Contains(t, applied.Message, "test-data")
}

func TestPluginEvaluateSupportsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	p := &testPlugin{}
	result, err := p.Evaluate(ctx, &config.Step{ID: "noop"})
	require.Error(t, err)
	require.Nil(t, result)
}
