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

func TestPortTemplatePlugin_EvaluateAndApply(t *testing.T) {
	t.Parallel()

	srcDir := t.TempDir()
	srcFile := filepath.Join(srcDir, "template.tmpl")
	require.NoError(t, os.WriteFile(srcFile, []byte("value: {{.Value}}\n"), 0o644))

	dstFile := filepath.Join(t.TempDir(), "output.txt")

	step := domainpipeline.Step{
		ID:   "render_template",
		Type: domainpipeline.StepTypeTemplate,
		Config: map[string]any{
			"source":      srcFile,
			"destination": dstFile,
			"vars": map[string]any{
				"Value": "hello",
			},
		},
	}

	plugin := NewPort()

	meta := plugin.Metadata()
	require.Equal(t, domainplugin.TypeTemplate, meta.Type)

	eval, err := plugin.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.NotNil(t, eval)

	result, err := plugin.Apply(context.Background(), eval, step)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, step.ID, result.StepID)
}

func TestPortTemplatePlugin_InvalidConfig(t *testing.T) {
	t.Parallel()

	step := domainpipeline.Step{
		ID:     "invalid",
		Type:   domainpipeline.StepTypeTemplate,
		Config: map[string]any{},
	}

	plugin := NewPort()

	_, err := plugin.Evaluate(context.Background(), step)
	require.Error(t, err)
}
