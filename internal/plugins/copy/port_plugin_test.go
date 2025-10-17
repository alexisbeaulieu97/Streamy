package copyplugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
)

func TestPortCopyPlugin_EvaluateAndApply(t *testing.T) {
	t.Parallel()

	srcDir := t.TempDir()
	dstDir := filepath.Join(t.TempDir(), "out")

	srcFile := filepath.Join(srcDir, "file.txt")
	require.NoError(t, os.WriteFile(srcFile, []byte("hello"), 0o644))

	step := domainpipeline.Step{
		ID:   "copy_file",
		Type: domainpipeline.StepTypeCopy,
		Config: map[string]any{
			"source":      srcFile,
			"destination": dstDir,
			"overwrite":   true,
		},
	}

	plugin := NewPort()

	meta := plugin.Metadata()
	require.Equal(t, domainplugin.TypeCopy, meta.Type)

	eval, err := plugin.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.NotNil(t, eval)
	require.True(t, eval.RequiresAction)

	result, err := plugin.Apply(context.Background(), eval, step)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, domainpipeline.StatusSuccess, result.Status)
}

func TestPortCopyPlugin_InvalidConfig(t *testing.T) {
	t.Parallel()

	step := domainpipeline.Step{
		ID:     "invalid",
		Type:   domainpipeline.StepTypeCopy,
		Config: map[string]any{},
	}

	plugin := NewPort()

	_, err := plugin.Evaluate(context.Background(), step)
	require.Error(t, err)
}
