package repoplugin

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
)

func TestPortRepoPlugin_Evaluate(t *testing.T) {
	t.Parallel()

	step := domainpipeline.Step{
		ID:   "clone_repo",
		Type: domainpipeline.StepTypeRepo,
		Config: map[string]any{
			"url":         "https://example.com/repo.git",
			"destination": filepath.Join(t.TempDir(), "repo"),
		},
	}

	plugin := NewPort()

	meta := plugin.Metadata()
	require.Equal(t, domainplugin.TypeRepo, meta.Type)

	eval, err := plugin.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.NotNil(t, eval)
}

func TestPortRepoPlugin_ApplyInvalidConfig(t *testing.T) {
	t.Parallel()

	step := domainpipeline.Step{
		ID:     "invalid_repo",
		Type:   domainpipeline.StepTypeRepo,
		Config: map[string]any{},
	}

	plugin := NewPort()

	_, err := plugin.Apply(context.Background(), nil, step)
	require.Error(t, err)
}
