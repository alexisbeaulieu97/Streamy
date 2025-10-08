package copyplugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	pluginpkg "github.com/alexisbeaulieu97/streamy/internal/plugin"
)

func TestCopyPlugin_Metadata(t *testing.T) {
	t.Parallel()

	p := New()
	meta := p.PluginMetadata()

	require.NotEmpty(t, meta.Name)
	require.NotEmpty(t, meta.Version)
	require.Equal(t, "copy", meta.Name)
}

func TestCopyPlugin_Schema(t *testing.T) {
	t.Parallel()

	p := New()
	schema := p.Schema()

	require.NotNil(t, schema)
	_, ok := schema.(config.CopyStep)
	require.True(t, ok, "schema should be of type CopyStep")
}

func TestCopyPlugin_EvaluateUsesHashForIdempotency(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	srcFile := filepath.Join(srcDir, "file.txt")
	dstFile := filepath.Join(dstDir, "file.txt")

	require.NoError(t, os.WriteFile(srcFile, []byte("same"), 0o644))
	require.NoError(t, os.WriteFile(dstFile, []byte("same"), 0o644))

	step := &config.Step{
		ID:   "copy_file",
		Type: "copy",
		Copy: &config.CopyStep{
			Source:      srcFile,
			Destination: dstFile,
			Overwrite:   true,
		},
	}

	p := New()
	require.Implements(t, (*pluginpkg.Plugin)(nil), p)

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusSatisfied, evalResult.CurrentState)
	require.False(t, evalResult.RequiresAction)

	require.NoError(t, os.WriteFile(dstFile, []byte("different"), 0o644))
	evalResult, err = p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusDrifted, evalResult.CurrentState)
	require.True(t, evalResult.RequiresAction)
}

func TestCopyPlugin_ApplyCopiesFileAndPermissions(t *testing.T) {
	srcDir := t.TempDir()
	dstFile := filepath.Join(t.TempDir(), "copied.txt")
	srcFile := filepath.Join(srcDir, "original.txt")

	require.NoError(t, os.WriteFile(srcFile, []byte("content"), 0o750))

	step := &config.Step{
		ID:   "copy_file",
		Type: "copy",
		Copy: &config.CopyStep{
			Source:      srcFile,
			Destination: dstFile,
			Overwrite:   true,
		},
	}

	p := New()

	evalResult := &model.EvaluationResult{
		StepID:         step.ID,
		CurrentState:   model.StatusMissing,
		RequiresAction: true,
		Message:        "File needs to be copied",
	}

	result, err := p.Apply(context.Background(), evalResult, step)
	require.NoError(t, err)
	require.Equal(t, "success", result.Status)

	data, err := os.ReadFile(dstFile)
	require.NoError(t, err)
	require.Equal(t, "content", string(data))

	info, err := os.Stat(dstFile)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o750), info.Mode().Perm())
}

func TestCopyPlugin_ApplyRecursiveCopy(t *testing.T) {
	srcDir := t.TempDir()
	nested := filepath.Join(srcDir, "nested")
	require.NoError(t, os.MkdirAll(nested, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(nested, "file.txt"), []byte("recursive"), 0o644))

	dstDir := t.TempDir()

	step := &config.Step{
		ID:   "copy_recursive",
		Type: "copy",
		Copy: &config.CopyStep{
			Source:      srcDir,
			Destination: dstDir,
			Recursive:   true,
			Overwrite:   true,
		},
	}

	p := New()

	evalResult := &model.EvaluationResult{
		StepID:         step.ID,
		CurrentState:   model.StatusMissing,
		RequiresAction: true,
		Message:        "Directory needs to be copied",
	}

	result, err := p.Apply(context.Background(), evalResult, step)
	require.NoError(t, err)
	require.Equal(t, "success", result.Status)

	copiedFile := filepath.Join(dstDir, "nested", "file.txt")
	data, err := os.ReadFile(copiedFile)
	require.NoError(t, err)
	require.Equal(t, "recursive", string(data))
}

func TestCopyPlugin_EvaluateMissingSource(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	srcFile := filepath.Join(srcDir, "nonexistent.txt")
	dstFile := filepath.Join(dstDir, "file.txt")

	step := &config.Step{
		ID:   "copy_missing_source",
		Type: "copy",
		Copy: &config.CopyStep{
			Source:      srcFile,
			Destination: dstFile,
		},
	}

	p := New()

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusMissing, evalResult.CurrentState)
	require.True(t, evalResult.RequiresAction)
	require.Contains(t, evalResult.Message, "does not exist")
}

func TestCopyPlugin_EvaluateDestinationExists(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	srcFile := filepath.Join(srcDir, "file.txt")
	dstFile := filepath.Join(dstDir, "file.txt")

	require.NoError(t, os.WriteFile(srcFile, []byte("content"), 0o644))
	require.NoError(t, os.WriteFile(dstFile, []byte("content"), 0o644))

	step := &config.Step{
		ID:   "copy_existing_dest",
		Type: "copy",
		Copy: &config.CopyStep{
			Source:      srcFile,
			Destination: dstFile,
		},
	}

	p := New()

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusSatisfied, evalResult.CurrentState)
	require.False(t, evalResult.RequiresAction)
	require.Contains(t, evalResult.Message, "files are identical")
}

func TestCopyPlugin_EvaluateDestinationDifferent(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	srcFile := filepath.Join(srcDir, "file.txt")
	dstFile := filepath.Join(dstDir, "file.txt")

	require.NoError(t, os.WriteFile(srcFile, []byte("source"), 0o644))
	require.NoError(t, os.WriteFile(dstFile, []byte("destination"), 0o644))

	step := &config.Step{
		ID:   "copy_different_dest",
		Type: "copy",
		Copy: &config.CopyStep{
			Source:      srcFile,
			Destination: dstFile,
			Overwrite:   true,
		},
	}

	p := New()

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusDrifted, evalResult.CurrentState)
	require.True(t, evalResult.RequiresAction)
	require.Contains(t, evalResult.Message, "files differ")
}

func TestCopyPlugin_EvaluateNoOverwrite(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	srcFile := filepath.Join(srcDir, "file.txt")
	dstFile := filepath.Join(dstDir, "file.txt")

	require.NoError(t, os.WriteFile(srcFile, []byte("source"), 0o644))
	require.NoError(t, os.WriteFile(dstFile, []byte("destination"), 0o644))

	step := &config.Step{
		ID:   "copy_no_overwrite",
		Type: "copy",
		Copy: &config.CopyStep{
			Source:      srcFile,
			Destination: dstFile,
			Overwrite:   false,
		},
	}

	p := New()

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusBlocked, evalResult.CurrentState)
	require.False(t, evalResult.RequiresAction)
	require.Contains(t, evalResult.Message, "destination exists and overwrite is disabled")
}

func TestCopyPlugin_EvaluateForDryRun(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	srcFile := filepath.Join(srcDir, "file.txt")
	dstFile := filepath.Join(dstDir, "file.txt")

	require.NoError(t, os.WriteFile(srcFile, []byte("content"), 0o644))

	step := &config.Step{
		ID:   "copy_dry_run",
		Type: "copy",
		Copy: &config.CopyStep{
			Source:      srcFile,
			Destination: dstFile,
		},
	}

	p := New()

	// Test dry run when destination doesn't exist
	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusMissing, evalResult.CurrentState)
	require.True(t, evalResult.RequiresAction)

	// Test dry run when destination exists and is identical
	require.NoError(t, os.WriteFile(dstFile, []byte("content"), 0o644))
	evalResult, err = p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusSatisfied, evalResult.CurrentState)
	require.False(t, evalResult.RequiresAction)
}
