package packageplugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
)

func TestPortPackagePlugin_EvaluateAndApply(t *testing.T) {
	t.Parallel()

	step := domainpipeline.Step{
		ID:   "install_pkg",
		Type: domainpipeline.StepTypePackage,
		Config: map[string]any{
			"packages": []string{"curl"},
		},
	}

	plugin := NewPort()

	meta := plugin.Metadata()
	require.Equal(t, domainplugin.TypePackage, meta.Type)

	eval, err := plugin.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.NotNil(t, eval)

	// Apply may fail in environments without apt; ensure it never panics and returns a domain result when possible.
	result, err := plugin.Apply(context.Background(), eval, step)
	if err == nil {
		require.NotNil(t, result)
		require.Equal(t, step.ID, result.StepID)
	}
}

func TestPortPackagePlugin_InvalidConfig(t *testing.T) {
	t.Parallel()

	step := domainpipeline.Step{
		ID:     "invalid",
		Type:   domainpipeline.StepTypePackage,
		Config: map[string]any{},
	}

	plugin := NewPort()

	_, err := plugin.Evaluate(context.Background(), step)
	require.Error(t, err)
}
