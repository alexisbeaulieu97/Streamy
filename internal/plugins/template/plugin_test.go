package templateplugin

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
