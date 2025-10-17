package commandplugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
)

func TestPortCommandPlugin_Evaluate(t *testing.T) {
	t.Parallel()

	step := domainpipeline.Step{
		ID:   "run_command",
		Type: domainpipeline.StepTypeCommand,
		Config: map[string]any{
			"command": "echo hello",
		},
	}

	plugin := NewPort()

	meta := plugin.Metadata()
	require.Equal(t, domainplugin.TypeCommand, meta.Type)

	eval, err := plugin.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.NotNil(t, eval)
}

func TestPortCommandPlugin_ApplyInvalidConfig(t *testing.T) {
	t.Parallel()

	step := domainpipeline.Step{
		ID:     "invalid",
		Type:   domainpipeline.StepTypeCommand,
		Config: map[string]any{},
	}

	plugin := NewPort()

	_, err := plugin.Apply(context.Background(), nil, step)
	require.Error(t, err)
}
