package templateplugin

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	require "github.com/stretchr/testify/require"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
)

func TestMetadata(t *testing.T) {
	meta := New().Metadata()
	require.Equal(t, "template", meta.Name)
	require.Equal(t, domainplugin.TypeTemplate, meta.Type)
}

func TestEvaluateMissingSource(t *testing.T) {
	step := domainpipeline.Step{
		ID:   "missing_src",
		Type: domainpipeline.StepTypeTemplate,
		Config: map[string]interface{}{
			"source":      filepath.Join(t.TempDir(), "missing.tmpl"),
			"destination": filepath.Join(t.TempDir(), "out.txt"),
		},
	}

	result, err := New().Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, result.RequiresAction)
	require.Equal(t, string(domainpipeline.VerificationFailed), result.CurrentState)
	require.Contains(t, result.DesiredState, "template source not found")
}

func TestApplyWritesRenderedTemplate(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "input.tmpl")
	dest := filepath.Join(dir, "output.txt")

	require.NoError(t, os.WriteFile(source, []byte("Hello {{.Name}}!"), 0o644))

	step := domainpipeline.Step{
		ID:   "render",
		Type: domainpipeline.StepTypeTemplate,
		Config: map[string]interface{}{
			"source":      source,
			"destination": dest,
			"vars": map[string]interface{}{
				"Name": "Streamy",
			},
		},
	}

	plugin := New()

	eval, err := plugin.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, eval.RequiresAction)

	res, err := plugin.Apply(context.Background(), eval, step)
	require.NoError(t, err)
	require.Equal(t, domainpipeline.StatusSuccess, res.Status)

	content, err := os.ReadFile(dest)
	require.NoError(t, err)
	require.Equal(t, "Hello Streamy!", string(content))
}

func TestEvaluateSatisfiedWhenUnchanged(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "input.tmpl")
	dest := filepath.Join(dir, "output.txt")

	require.NoError(t, os.WriteFile(source, []byte("static content"), 0o644))
	require.NoError(t, os.WriteFile(dest, []byte("static content"), 0o644))

	step := domainpipeline.Step{
		ID:   "satisfied",
		Type: domainpipeline.StepTypeTemplate,
		Config: map[string]interface{}{
			"source":      source,
			"destination": dest,
		},
	}

	result, err := New().Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.False(t, result.RequiresAction)
	require.Equal(t, string(domainpipeline.VerificationSatisfied), result.CurrentState)
}

func TestEvaluateDriftedContentProducesDiff(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "input.tmpl")
	dest := filepath.Join(dir, "output.txt")

	require.NoError(t, os.WriteFile(source, []byte("desired\nvalue\n"), 0o644))
	require.NoError(t, os.WriteFile(dest, []byte("old\nvalue\n"), 0o644))

	step := domainpipeline.Step{
		ID:   "drifted",
		Type: domainpipeline.StepTypeTemplate,
		Config: map[string]interface{}{
			"source":      source,
			"destination": dest,
		},
	}

	result, err := New().Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, result.RequiresAction)
	require.Equal(t, string(domainpipeline.VerificationFailed), result.CurrentState)
	require.NotEmpty(t, result.Diff)
}

func TestDecodeConfigValidation(t *testing.T) {
	_, err := decodeConfig(domainpipeline.Step{ID: "missing"})
	require.Error(t, err)

	_, err = decodeConfig(domainpipeline.Step{
		ID:     "template",
		Config: map[string]interface{}{},
	})
	require.Error(t, err)

	cfg, err := decodeConfig(domainpipeline.Step{
		ID: "template",
		Config: map[string]interface{}{
			"source":      "/tmp/src",
			"destination": "/tmp/dst",
			"mode":        "0640",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, cfg.Mode)
	require.Equal(t, uint32(0o640), *cfg.Mode)
}

func TestEnsureEvaluationData(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "src.tmpl")
	dest := filepath.Join(dir, "dst.txt")

	require.NoError(t, os.WriteFile(source, []byte("content"), 0o644))

	step := domainpipeline.Step{
		ID:   "ensure",
		Type: domainpipeline.StepTypeTemplate,
		Config: map[string]interface{}{
			"source":      source,
			"destination": dest,
		},
	}

	eval := &domainpipeline.EvaluationResult{InternalData: &evaluationData{RenderedContent: "content"}}
	data, err := ensureEvaluationData(context.Background(), eval, step)
	require.NoError(t, err)
	require.NotNil(t, data)

	data, err = ensureEvaluationData(context.Background(), &domainpipeline.EvaluationResult{}, step)
	require.NoError(t, err)
	require.NotNil(t, data)

	_, err = ensureEvaluationData(context.Background(), nil, domainpipeline.Step{ID: "bad"})
	require.Error(t, err)
}

func TestParseBool(t *testing.T) {
	cases := map[interface{}]bool{
		true:   true,
		"true": true,
		"1":    true,
		false:  false,
		"no":   false,
		"":     false,
	}

	for raw, expected := range cases {
		val, err := parseBool(raw)
		require.NoError(t, err)
		require.Equal(t, expected, val)
	}

	if _, err := parseBool(123); err == nil {
		t.Fatal("expected error for invalid boolean type")
	}
}

func TestParseMode(t *testing.T) {
	modes := []interface{}{0o640, int32(0o644), int64(0o600), "0750"}
	for _, raw := range modes {
		val, err := parseMode(raw)
		require.NoError(t, err)
		require.NotZero(t, val)
	}

	if _, err := parseMode(-1); err == nil {
		t.Fatal("expected error for negative mode")
	}
}

func TestParseVarsInvalid(t *testing.T) {
	if _, err := parseVars(42); err == nil {
		t.Fatal("expected error for invalid vars type")
	}

	_, err := parseVars(map[string]interface{}{"k": 123})
	require.Error(t, err)
}

func TestWrapTemplatePathError(t *testing.T) {
	err := wrapTemplatePathError("read", "/tmp/file", errors.New("boom"))
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "read"))
}
