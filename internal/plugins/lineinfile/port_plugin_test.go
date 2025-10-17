package lineinfileplugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
)

func TestPortLineInFilePlugin_EvaluateAndApply(t *testing.T) {
	t.Parallel()

	file := filepath.Join(t.TempDir(), "config.txt")
	require.NoError(t, os.WriteFile(file, []byte("key=value\n"), 0o644))

	step := domainpipeline.Step{
		ID:   "ensure_line",
		Type: domainpipeline.StepTypeLineInFile,
		Config: map[string]any{
			"file":  file,
			"line":  "key=value",
			"state": "present",
		},
	}

	plugin := NewPort()

	meta := plugin.Metadata()
	require.Equal(t, domainplugin.TypeLineInFile, meta.Type)

	eval, err := plugin.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.NotNil(t, eval)

	result, err := plugin.Apply(context.Background(), eval, step)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, step.ID, result.StepID)
}

func TestPortLineInFilePlugin_InvalidConfig(t *testing.T) {
	t.Parallel()

	step := domainpipeline.Step{
		ID:     "invalid",
		Type:   domainpipeline.StepTypeLineInFile,
		Config: map[string]any{},
	}

	plugin := NewPort()

	_, err := plugin.Evaluate(context.Background(), step)
	require.Error(t, err)
}
