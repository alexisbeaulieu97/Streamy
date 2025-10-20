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

func TestDecodeConfigValidation(t *testing.T) {
	_, err := newConfigFromDomainStep(domainpipeline.Step{ID: "missing"})
	require.Error(t, err)

	_, err = newConfigFromDomainStep(domainpipeline.Step{
		ID:     "invalid",
		Config: map[string]interface{}{},
	})
	require.Error(t, err)

	cfg, err := newConfigFromDomainStep(domainpipeline.Step{
		ID: "valid",
		Config: map[string]interface{}{
			"file":  "/tmp/test.txt",
			"line":  "hello",
			"state": "present",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "/tmp/test.txt", cfg.File)
	require.Equal(t, "present", cfg.State)
}
