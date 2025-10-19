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

func TestMetadata(t *testing.T) {
	meta := New().Metadata()
	require.Equal(t, "symlink", meta.Name)
	require.Equal(t, domainplugin.TypeSymlink, meta.Type)
}

func TestEvaluateMissingSymlink(t *testing.T) {
	step := domainpipeline.Step{
		ID:   "missing_symlink",
		Type: domainpipeline.StepTypeSymlink,
		Config: map[string]interface{}{
			"source": "/tmp/source",
			"target": filepath.Join(t.TempDir(), "link"),
		},
	}

	result, err := New().Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, result.RequiresAction)
	require.Equal(t, string(domainpipeline.VerificationFailed), result.CurrentState)
	require.Contains(t, result.Diff, "would create symlink")
}

func TestApplyCreatesSymlink(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	source := filepath.Join(sourceDir, "file.txt")
	require.NoError(t, os.WriteFile(source, []byte("hello"), 0o644))
	target := filepath.Join(targetDir, "link.txt")

	step := domainpipeline.Step{
		ID:   "create_symlink",
		Type: domainpipeline.StepTypeSymlink,
		Config: map[string]interface{}{
			"source": source,
			"target": target,
		},
	}

	plugin := New()
	eval, err := plugin.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, eval.RequiresAction)

	res, err := plugin.Apply(context.Background(), eval, step)
	require.NoError(t, err)
	require.Equal(t, domainpipeline.StatusSuccess, res.Status)
	require.True(t, res.Changed)

	linkTarget, err := os.Readlink(target)
	require.NoError(t, err)
	require.Equal(t, source, linkTarget)
}

func TestEvaluateSatisfiedSymlink(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	source := filepath.Join(sourceDir, "file.txt")
	target := filepath.Join(targetDir, "link.txt")
	require.NoError(t, os.WriteFile(source, []byte("hello"), 0o644))
	require.NoError(t, os.Symlink(source, target))

	step := domainpipeline.Step{
		ID:   "satisfied_symlink",
		Type: domainpipeline.StepTypeSymlink,
		Config: map[string]interface{}{
			"source": source,
			"target": target,
		},
	}

	result, err := New().Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.False(t, result.RequiresAction)
	require.Equal(t, string(domainpipeline.VerificationSatisfied), result.CurrentState)
}

func TestEvaluateNonSymlinkWithoutForce(t *testing.T) {
	targetDir := t.TempDir()
	target := filepath.Join(targetDir, "link.txt")
	require.NoError(t, os.WriteFile(target, []byte("existing"), 0o644))

	step := domainpipeline.Step{
		ID:   "non_symlink",
		Type: domainpipeline.StepTypeSymlink,
		Config: map[string]interface{}{
			"source": "/tmp/source",
			"target": target,
		},
	}

	result, err := New().Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.False(t, result.RequiresAction)
	require.Equal(t, string(domainpipeline.VerificationFailed), result.CurrentState)
	require.Contains(t, result.DesiredState, "not a symlink")
}

func TestApplyForcedReplacement(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	source := filepath.Join(sourceDir, "file.txt")
	require.NoError(t, os.WriteFile(source, []byte("hello"), 0o644))
	target := filepath.Join(targetDir, "link.txt")

	require.NoError(t, os.WriteFile(target, []byte("existing"), 0o644))

	step := domainpipeline.Step{
		ID:   "force_symlink",
		Type: domainpipeline.StepTypeSymlink,
		Config: map[string]interface{}{
			"source": source,
			"target": target,
			"force":  true,
		},
	}

	plugin := New()
	eval, err := plugin.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, eval.RequiresAction)

	_, err = plugin.Apply(context.Background(), eval, step)
	require.NoError(t, err)

	linkTarget, err := os.Readlink(target)
	require.NoError(t, err)
	require.Equal(t, source, linkTarget)
}

func TestDecodeConfigValidation(t *testing.T) {
	_, err := decodeConfig(domainpipeline.Step{ID: "missing"})
	require.Error(t, err)

	_, err = decodeConfig(domainpipeline.Step{ID: "cfg", Config: map[string]interface{}{}})
	require.Error(t, err)

	cfg, err := decodeConfig(domainpipeline.Step{
		ID: "cfg",
		Config: map[string]interface{}{
			"source": "/tmp/source",
			"target": "/tmp/target",
			"force":  "true",
		},
	})
	require.NoError(t, err)
	require.True(t, cfg.Force)
	require.Equal(t, "/tmp/source", cfg.Source)
	require.Equal(t, "/tmp/target", cfg.Target)
}
