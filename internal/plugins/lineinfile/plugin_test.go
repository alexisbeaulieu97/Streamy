package lineinfileplugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	require "github.com/stretchr/testify/require"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
)

func TestMetadata(t *testing.T) {
	meta := New().Metadata()
	require.Equal(t, "line_in_file", meta.Name)
	require.Equal(t, domainplugin.TypeLineInFile, meta.Type)
}

func TestEvaluateMissingFile(t *testing.T) {
	step := domainpipeline.Step{
		ID:   "missing-file",
		Type: domainpipeline.StepTypeLineInFile,
		Config: map[string]interface{}{
			"file": "/tmp/does-not-exist.txt",
			"line": "hello",
		},
	}

	result, err := New().Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, result.RequiresAction)
	require.Equal(t, string(domainpipeline.VerificationFailed), result.CurrentState)
}

func TestApplyAddsLine(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "file.txt")
	require.NoError(t, os.WriteFile(target, []byte("first\n"), 0o644))

	step := domainpipeline.Step{
		ID:   "add-line",
		Type: domainpipeline.StepTypeLineInFile,
		Config: map[string]interface{}{
			"file": target,
			"line": "second",
		},
	}

	plugin := New()
	eval, err := plugin.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, eval.RequiresAction)

	res, err := plugin.Apply(context.Background(), eval, step)
	require.NoError(t, err)
	require.Equal(t, domainpipeline.StatusSuccess, res.Status)

	content, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Contains(t, string(content), "second")
}

func TestApplyRemovesLine(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "file.txt")
	require.NoError(t, os.WriteFile(target, []byte("first\nremove\nthird\n"), 0o644))

	step := domainpipeline.Step{
		ID:   "remove-line",
		Type: domainpipeline.StepTypeLineInFile,
		Config: map[string]interface{}{
			"file":  target,
			"line":  "remove",
			"state": "absent",
			"match": "remove",
		},
	}

	plugin := New()
	eval, err := plugin.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, eval.RequiresAction)

	res, err := plugin.Apply(context.Background(), eval, step)
	require.NoError(t, err)
	require.Equal(t, domainpipeline.StatusSuccess, res.Status)

	content, err := os.ReadFile(target)
	require.NoError(t, err)
	require.NotContains(t, string(content), "remove")
}

func TestEvaluateLineAlreadyPresent(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "file.txt")
	require.NoError(t, os.WriteFile(target, []byte("hello\n"), 0o644))

	step := domainpipeline.Step{
		ID:   "add-line",
		Type: domainpipeline.StepTypeLineInFile,
		Config: map[string]interface{}{
			"file": target,
			"line": "hello",
		},
	}

	result, err := New().Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.False(t, result.RequiresAction)
}

func TestApplyNoChangesNeeded(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "file.txt")
	require.NoError(t, os.WriteFile(target, []byte("hello\n"), 0o644))

	step := domainpipeline.Step{
		ID:   "add-line",
		Type: domainpipeline.StepTypeLineInFile,
		Config: map[string]interface{}{
			"file": target,
			"line": "hello",
		},
	}

	plugin := New()
	_, err := plugin.Apply(context.Background(), nil, step)
	require.NoError(t, err)
}

func TestApplyWithBackup(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "file.txt")
	require.NoError(t, os.WriteFile(target, []byte("first\n"), 0o644))

	step := domainpipeline.Step{
		ID:   "add-line",
		Type: domainpipeline.StepTypeLineInFile,
		Config: map[string]interface{}{
			"file":   target,
			"line":   "second",
			"backup": true,
		},
	}

	plugin := New()
	_, err := plugin.Apply(context.Background(), nil, step)
	require.NoError(t, err)

	files, err := filepath.Glob(filepath.Join(dir, "*.bak"))
	require.NoError(t, err)
	require.Len(t, files, 1)
}
