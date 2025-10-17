package symlinkplugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
)

func TestPortSymlinkPlugin_EvaluateAndApply(t *testing.T) {
	t.Parallel()

	sourceDir := t.TempDir()
	targetDir := filepath.Join(t.TempDir(), "linked")

	sourceFile := filepath.Join(sourceDir, "file.txt")
	require.NoError(t, os.WriteFile(sourceFile, []byte("hello"), 0o644))

	step := domainpipeline.Step{
		ID:     "link_file",
		Type:   domainpipeline.StepTypeSymlink,
		Config: map[string]any{"source": sourceFile, "target": targetDir},
	}

	p := NewPort()

	meta := p.Metadata()
	require.Equal(t, domainplugin.TypeSymlink, meta.Type)

	eval, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.NotNil(t, eval)
	require.True(t, eval.RequiresAction)

	result, err := p.Apply(context.Background(), eval, step)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, domainpipeline.StatusSuccess, result.Status)

	target, err := os.Readlink(targetDir)
	require.NoError(t, err)
	require.Equal(t, sourceFile, target)
}

func TestPortSymlinkPlugin_ApplyInvalidConfig(t *testing.T) {
	t.Parallel()

	step := domainpipeline.Step{
		ID:     "invalid",
		Type:   domainpipeline.StepTypeSymlink,
		Config: map[string]any{},
	}

	p := NewPort()
	_, err := p.Apply(context.Background(), nil, step)
	require.Error(t, err)
}
