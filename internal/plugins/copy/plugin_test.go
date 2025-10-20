package copyplugin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	require "github.com/stretchr/testify/require"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
)

type testStringer struct{}

func (testStringer) String() string { return "stringer" }

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

func TestEvaluateDestinationIsDir(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source")
	require.NoError(t, os.WriteFile(source, []byte("hello"), 0o644))
	dest := t.TempDir()

	step := domainpipeline.Step{
		ID:   "copy_file",
		Type: domainpipeline.StepTypeCopy,
		Config: map[string]interface{}{
			"source":      source,
			"destination": dest,
		},
	}

	result, err := New().Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, result.RequiresAction)
}

func TestEvaluateFilesAreIdentical(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source")
	require.NoError(t, os.WriteFile(source, []byte("hello"), 0o644))
	dest := filepath.Join(t.TempDir(), "dest")
	require.NoError(t, os.WriteFile(dest, []byte("hello"), 0o644))

	step := domainpipeline.Step{
		ID:   "copy_file",
		Type: domainpipeline.StepTypeCopy,
		Config: map[string]interface{}{
			"source":      source,
			"destination": dest,
		},
	}

	result, err := New().Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.False(t, result.RequiresAction)
}

func TestApplyDirectoryCopy(t *testing.T) {
	source := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(source, "file1.txt"), []byte("file1"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(source, "file2.txt"), []byte("file2"), 0o644))
	dest := filepath.Join(t.TempDir(), "dest")

	step := domainpipeline.Step{
		ID:   "copy_dir",
		Type: domainpipeline.StepTypeCopy,
		Config: map[string]interface{}{
			"source":      source,
			"destination": dest,
			"recursive":   true,
		},
	}

	plugin := New()
	_, err := plugin.Apply(context.Background(), nil, step)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dest, "file1.txt"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(dest, "file2.txt"))
	require.NoError(t, err)
}

func TestDecodeConfigMissingSource(t *testing.T) {
	_, err := decodeConfig(domainpipeline.Step{
		ID: "copy_file",
		Config: map[string]interface{}{
			"destination": "/tmp/dest",
		},
	})
	require.Error(t, err)
}

func TestGetBool(t *testing.T) {
	trueValues := []string{"true", "1", "yes", "on"}
	for _, val := range trueValues {
		result, ok := getBool(map[string]interface{}{"key": val}, "key")
		require.True(t, ok)
		require.True(t, result)
	}

	falseValues := []string{"false", "0", "no", "off"}
	for _, val := range falseValues {
		result, ok := getBool(map[string]interface{}{"key": val}, "key")
		require.True(t, ok)
		require.False(t, result)
	}
}

func TestMissingConfig(t *testing.T) {
	step := domainpipeline.Step{ID: "missing", Type: domainpipeline.StepTypeCopy}
	_, err := New().Evaluate(context.Background(), step)
	require.Error(t, err)
}

func TestEnsureDestinationWritable(t *testing.T) {
	tmpDir := t.TempDir()
	existing := filepath.Join(tmpDir, "exists.txt")
	require.NoError(t, os.WriteFile(existing, []byte("data"), 0o644))

	require.NoError(t, ensureDestinationWritable(existing, true))
	require.NoError(t, ensureDestinationWritable(filepath.Join(tmpDir, "new.txt"), false))

	err := ensureDestinationWritable(existing, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "destination exists")
}

func TestEnsureDestinationDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nestedFile := filepath.Join(tmpDir, "nested", "file.txt")

	require.NoError(t, ensureDestinationDirectory(nestedFile))

	info, err := os.Stat(filepath.Join(tmpDir, "nested"))
	require.NoError(t, err)
	require.True(t, info.IsDir())
}

func TestOpenDestinationFile(t *testing.T) {
	tmpDir := t.TempDir()
	dest := filepath.Join(tmpDir, "dest.txt")

	file, err := openDestinationFile(dest)
	require.NoError(t, err)
	require.NoError(t, file.Close())

	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	require.Empty(t, data)
}

func TestStreamFileCopiesData(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src.txt")
	dest := filepath.Join(tmpDir, "dest.txt")

	require.NoError(t, os.WriteFile(src, []byte("streamy"), 0o644))

	srcFile, err := os.Open(src)
	require.NoError(t, err)
	t.Cleanup(func() { _ = srcFile.Close() })

	dstFile, err := os.OpenFile(dest, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o600)
	require.NoError(t, err)
	t.Cleanup(func() { _ = dstFile.Close() })

	require.NoError(t, streamFile(context.Background(), src, dest, srcFile, dstFile))

	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	require.Equal(t, "streamy", string(data))
}

func TestHashFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "hash.txt")

	require.NoError(t, os.WriteFile(path, []byte("hash"), 0o644))

	hash, err := hashFile(context.Background(), path)
	require.NoError(t, err)
	require.Len(t, hash, 64)

	// Ensure hashing respects context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = hashFile(ctx, path)
	require.Error(t, err)
}

func TestGenerateFileDiff(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src.txt")
	dst := filepath.Join(tmpDir, "dst.txt")

	require.NoError(t, os.WriteFile(src, []byte("one"), 0o644))
	require.NoError(t, os.WriteFile(dst, []byte("two"), 0o644))

	diff := generateFileDiff(src, dst)
	require.Contains(t, diff, "-one")
	require.Contains(t, diff, "+two")
}

func TestGetStringCoversStringer(t *testing.T) {
	val, ok := getString(map[string]interface{}{"str": testStringer{}}, "str")
	require.True(t, ok)
	require.Equal(t, "stringer", val)
}

func TestGetBoolInvalidInput(t *testing.T) {
	value, ok := getBool(map[string]interface{}{"key": 123}, "key")
	require.False(t, ok)
	require.False(t, value)
}

func TestCancelError(t *testing.T) {
	err := cancelError("step-id", errors.New("ctx cancelled"))
	require.Equal(t, domainpipeline.ErrCodeCancelled, err.Code)
	require.Equal(t, "step-id", err.Context["step_id"])
}

func TestFailureResult(t *testing.T) {
	result, err := failureResult("step-1", "boom")
	require.Error(t, err)
	require.Equal(t, domainpipeline.StatusFailure, result.Status)
	require.NotNil(t, result.Error)

	domainErr, ok := err.(*domainpipeline.DomainError)
	require.True(t, ok)
	require.Equal(t, domainpipeline.ErrCodeExecution, domainErr.Code)
}

func TestFailureWithError(t *testing.T) {
	cause := fmt.Errorf("%w", errors.New("root"))
	result, err := failureWithError("step-1", "copy file", cause)
	require.Error(t, err)
	require.Equal(t, domainpipeline.StatusFailure, result.Status)
	require.Contains(t, result.Message, "copy file failed")

	domainErr, ok := err.(*domainpipeline.DomainError)
	require.True(t, ok)
	require.Equal(t, cause, domainErr.Cause)
}
