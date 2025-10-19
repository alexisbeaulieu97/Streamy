package copyplugin

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
)

func TestEvaluateMissingDestination(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX path expectations")
	}

	tmpDir := t.TempDir()
	source := filepath.Join(tmpDir, "source.txt")
	require.NoError(t, os.WriteFile(source, []byte("streamy"), 0o644))

	step := domainpipeline.Step{
		ID:   "copy_file",
		Type: domainpipeline.StepTypeCopy,
		Config: map[string]interface{}{
			"source":      source,
			"destination": filepath.Join(tmpDir, "dest.txt"),
		},
	}

	eval, err := New().Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, eval.RequiresAction)
	require.Equal(t, string(domainpipeline.VerificationFailed), eval.CurrentState)
}

func TestApplyCopiesFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX path expectations")
	}

	tmpDir := t.TempDir()
	source := filepath.Join(tmpDir, "source.txt")
	dest := filepath.Join(tmpDir, "dest.txt")
	require.NoError(t, os.WriteFile(source, []byte("streamy"), 0o644))

	step := domainpipeline.Step{
		ID:   "copy_file",
		Type: domainpipeline.StepTypeCopy,
		Config: map[string]interface{}{
			"source":      source,
			"destination": dest,
			"overwrite":   true,
		},
	}

	eval := &domainpipeline.EvaluationResult{RequiresAction: true}
	result, err := New().Apply(context.Background(), eval, step)
	require.NoError(t, err)
	require.Equal(t, domainpipeline.StatusSuccess, result.Status)

	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	require.Equal(t, "streamy", string(data))
}

func TestEvaluateRecursiveDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX path expectations")
	}

	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "src")
	destDir := filepath.Join(tmpDir, "dst")
	require.NoError(t, os.Mkdir(sourceDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "file.txt"), []byte("content"), 0o644))

	step := domainpipeline.Step{
		ID:   "copy_dir",
		Type: domainpipeline.StepTypeCopy,
		Config: map[string]interface{}{
			"source":      sourceDir,
			"destination": destDir,
			"recursive":   true,
		},
	}

	eval, err := New().Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, eval.RequiresAction)
	require.Equal(t, string(domainpipeline.VerificationFailed), eval.CurrentState)
	require.Contains(t, eval.Diff, "Would copy:")
}

func TestEvaluateOverwriteDisabledWhenDifferent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX path expectations")
	}

	tmpDir := t.TempDir()
	source := filepath.Join(tmpDir, "source.txt")
	dest := filepath.Join(tmpDir, "dest.txt")
	require.NoError(t, os.WriteFile(source, []byte("new"), 0o644))
	require.NoError(t, os.WriteFile(dest, []byte("old"), 0o644))

	step := domainpipeline.Step{
		ID:   "copy_file",
		Type: domainpipeline.StepTypeCopy,
		Config: map[string]interface{}{
			"source":      source,
			"destination": dest,
			"overwrite":   false,
		},
	}

	eval, err := New().Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.False(t, eval.RequiresAction)
	require.Equal(t, string(domainpipeline.VerificationUnknown), eval.CurrentState)
	require.Contains(t, eval.DesiredState, "overwrite is disabled")
}

func TestApplyDirectoryCopyWhenRecursiveEnabled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX path expectations")
	}

	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "src")
	destDir := filepath.Join(tmpDir, "dst")
	require.NoError(t, os.Mkdir(sourceDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "nested.txt"), []byte("nested"), 0o644))

	step := domainpipeline.Step{
		ID:   "copy_dir",
		Type: domainpipeline.StepTypeCopy,
		Config: map[string]interface{}{
			"source":      sourceDir,
			"destination": destDir,
			"recursive":   true,
		},
	}

	eval, err := New().Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, eval.RequiresAction)

	result, err := New().Apply(context.Background(), eval, step)
	require.NoError(t, err)
	require.Equal(t, domainpipeline.StatusSuccess, result.Status)

	data, err := os.ReadFile(filepath.Join(destDir, "nested.txt"))
	require.NoError(t, err)
	require.Equal(t, "nested", string(data))
}

func TestGatherEvaluationDataHandlesMissingSource(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX path expectations")
	}

	step := domainpipeline.Step{
		ID:   "copy_missing",
		Type: domainpipeline.StepTypeCopy,
		Config: map[string]interface{}{
			"source":      filepath.Join(t.TempDir(), "nonexistent.txt"),
			"destination": filepath.Join(t.TempDir(), "dest.txt"),
		},
	}

	eval, err := New().Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, eval.RequiresAction)
	require.Contains(t, eval.DesiredState, "does not exist")
}

func TestMetadata(t *testing.T) {
	meta := New().Metadata()
	require.Equal(t, "copy", meta.Name)
	require.Equal(t, domainplugin.TypeCopy, meta.Type)
}

func TestMissingConfig(t *testing.T) {
	step := domainpipeline.Step{ID: "missing", Type: domainpipeline.StepTypeCopy}
	_, err := New().Evaluate(context.Background(), step)
	require.Error(t, err)
}
