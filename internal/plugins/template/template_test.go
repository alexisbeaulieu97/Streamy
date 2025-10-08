package templateplugin

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

func TestTemplatePlugin_Metadata(t *testing.T) {
	t.Parallel()

	p := New()
	meta := p.PluginMetadata()

	require.NotEmpty(t, meta.Name)
	require.NotEmpty(t, meta.Version)
	require.Equal(t, "template", meta.Name)
}

func TestTemplatePlugin_Schema(t *testing.T) {
	t.Parallel()

	p := New()
	schema := p.Schema()

	require.NotNil(t, schema)
	_, ok := schema.(config.TemplateStep)
	require.True(t, ok, "schema should be of type TemplateStep")
}

func TestTemplatePlugin_EvaluateMissingTemplate(t *testing.T) {
	t.Parallel()

	src := filepath.Join(t.TempDir(), "template.txt.tmpl")
	dst := filepath.Join(t.TempDir(), "output.txt")

	// Don't create the source file
	step := &config.Step{
		ID:   "missing_template",
		Type: "template",
		Template: &config.TemplateStep{
			Source:      src,
			Destination: dst,
		},
	}

	p := New()

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusMissing, evalResult.CurrentState)
	require.True(t, evalResult.RequiresAction)
	require.Contains(t, evalResult.Message, "template source not found")
}

func TestTemplatePlugin_EvaluateSatisfied(t *testing.T) {
	t.Parallel()

	src := filepath.Join(t.TempDir(), "template.txt.tmpl")
	dst := filepath.Join(t.TempDir(), "output.txt")

	// Create template
	content := "Hello {{.Name}}!"
	require.NoError(t, os.WriteFile(src, []byte(content), 0o644))

	// Create output with same content (assuming no variables)
	require.NoError(t, os.WriteFile(dst, []byte(content), 0o644))

	step := &config.Step{
		ID:   "satisfied_template",
		Type: "template",
		Template: &config.TemplateStep{
			Source:      src,
			Destination: dst,
		},
	}

	p := New()

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusSatisfied, evalResult.CurrentState)
	require.False(t, evalResult.RequiresAction)
}

func TestTemplatePlugin_EvaluateDrifted(t *testing.T) {
	t.Parallel()

	src := filepath.Join(t.TempDir(), "template.txt.tmpl")
	dst := filepath.Join(t.TempDir(), "output.txt")

	// Create template
	require.NoError(t, os.WriteFile(src, []byte("Hello World!"), 0o644))

	// Create output with different content
	require.NoError(t, os.WriteFile(dst, []byte("Different content"), 0o644))

	step := &config.Step{
		ID:   "drifted_template",
		Type: "template",
		Template: &config.TemplateStep{
			Source:      src,
			Destination: dst,
		},
	}

	p := New()

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusDrifted, evalResult.CurrentState)
	require.True(t, evalResult.RequiresAction)
}

func TestTemplatePlugin_ApplyCreatesFile(t *testing.T) {
	t.Parallel()

	src := filepath.Join(t.TempDir(), "template.txt.tmpl")
	dst := filepath.Join(t.TempDir(), "output.txt")

	// Create template
	require.NoError(t, os.WriteFile(src, []byte("Hello World!"), 0o644))

	step := &config.Step{
		ID:   "create_template",
		Type: "template",
		Template: &config.TemplateStep{
			Source:      src,
			Destination: dst,
		},
	}

	p := New()

	// First evaluate
	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, evalResult.RequiresAction)

	// Then apply
	result, err := p.Apply(context.Background(), evalResult, step)
	require.NoError(t, err)
	require.Equal(t, step.ID, result.StepID)
	require.Equal(t, model.StatusSuccess, result.Status)

	// Verify file was created
	content, err := os.ReadFile(dst)
	require.NoError(t, err)
	require.Equal(t, "Hello World!", string(content))
}

func TestTemplatePlugin_ApplyWithVariables(t *testing.T) {
	t.Parallel()

	src := filepath.Join(t.TempDir(), "template.txt.tmpl")
	dst := filepath.Join(t.TempDir(), "output.txt")

	// Create template with variables
	require.NoError(t, os.WriteFile(src, []byte("Hello {{.Name}}!"), 0o644))

	step := &config.Step{
		ID:   "template_with_vars",
		Type: "template",
		Template: &config.TemplateStep{
			Source:      src,
			Destination: dst,
			Vars: map[string]string{
				"Name": "Streamy",
			},
		},
	}

	p := New()

	// First evaluate
	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, evalResult.RequiresAction)

	// Then apply
	result, err := p.Apply(context.Background(), evalResult, step)
	require.NoError(t, err)
	require.Equal(t, step.ID, result.StepID)
	require.Equal(t, model.StatusSuccess, result.Status)

	// Verify file was created with variables substituted
	content, err := os.ReadFile(dst)
	require.NoError(t, err)
	require.Equal(t, "Hello Streamy!", string(content))
}

func TestTemplatePlugin_EvaluateWithVariables(t *testing.T) {
	t.Parallel()

	src := filepath.Join(t.TempDir(), "template.txt.tmpl")
	dst := filepath.Join(t.TempDir(), "output.txt")

	// Create template with variables
	require.NoError(t, os.WriteFile(src, []byte("Hello {{.Name}}!"), 0o644))

	// Create output with different variable value
	require.NoError(t, os.WriteFile(dst, []byte("Hello World!"), 0o644))

	step := &config.Step{
		ID:   "template_vars_eval",
		Type: "template",
		Template: &config.TemplateStep{
			Source:      src,
			Destination: dst,
			Vars: map[string]string{
				"Name": "Streamy",
			},
		},
	}

	p := New()

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusDrifted, evalResult.CurrentState)
	require.True(t, evalResult.RequiresAction)
}

func TestTemplatePlugin_EvaluateErrors(t *testing.T) {
	t.Run("returns error when template config is nil", func(t *testing.T) {
		p := New()

		step := &config.Step{
			ID:       "test_template",
			Type:     "template",
			Template: nil,
		}

		_, err := p.Evaluate(context.Background(), step)
		require.Error(t, err)
		require.Contains(t, err.Error(), "template configuration missing")
	})

	t.Run("returns error when template syntax is invalid", func(t *testing.T) {
		src := filepath.Join(t.TempDir(), "invalid.txt.tmpl")
		dst := filepath.Join(t.TempDir(), "output.txt")

		// Create template with invalid syntax
		require.NoError(t, os.WriteFile(src, []byte("Hello {{.invalid"), 0o644))

		step := &config.Step{
			ID:   "invalid_template",
			Type: "template",
			Template: &config.TemplateStep{
				Source:      src,
				Destination: dst,
			},
		}

		p := New()

		_, err := p.Evaluate(context.Background(), step)
		// The result depends on how the plugin handles template parsing errors
		// It might return an error or a blocked status
		require.Error(t, err)
	})
}

func TestTemplatePlugin_ApplyErrors(t *testing.T) {
	t.Run("returns error when template config is nil", func(t *testing.T) {
		p := New()

		step := &config.Step{
			ID:       "test_template",
			Type:     "template",
			Template: nil,
		}

		evalResult := &model.EvaluationResult{
			StepID:         step.ID,
			CurrentState:   model.StatusMissing,
			RequiresAction: true,
			Message:        "Test",
		}

		_, err := p.Apply(context.Background(), evalResult, step)
		require.Error(t, err)
		require.Contains(t, err.Error(), "template configuration missing")
	})

	t.Run("returns error when template is missing", func(t *testing.T) {
		p := New()

		step := &config.Step{
			ID:   "missing_template_apply",
			Type: "template",
			Template: &config.TemplateStep{
				Source:      "/nonexistent/template.txt.tmpl",
				Destination: "/tmp/output.txt",
			},
		}

		evalResult, err := p.Evaluate(context.Background(), step)
		require.NoError(t, err)
		require.True(t, evalResult.RequiresAction)

		result, err := p.Apply(context.Background(), evalResult, step)
		require.Error(t, err)
		if result != nil {
			require.Equal(t, model.StatusFailed, result.Status)
		}
	})
}

// Contract tests for the new plugin interface
func TestTemplatePlugin_Contract(t *testing.T) {
	t.Run("Plugin Contract", func(t *testing.T) {
		plugin := New()

		t.Run("Metadata is stable", func(t *testing.T) {
			m1 := plugin.PluginMetadata()
			m2 := plugin.PluginMetadata()
			require.Equal(t, m1, m2, "PluginMetadata() should return consistent values across calls")
		})

		t.Run("Schema returns struct", func(t *testing.T) {
			schema := plugin.Schema()
			require.NotNil(t, schema, "Schema() should not return nil")
			_, ok := schema.(config.TemplateStep)
			require.True(t, ok, "Schema() should return a TemplateStep struct")
		})

		t.Run("Evaluate is idempotent", func(t *testing.T) {
			src := filepath.Join(t.TempDir(), "idempotent.txt.tmpl")
			dst := filepath.Join(t.TempDir(), "output.txt")

			// Don't create files to keep it simple
			step := &config.Step{
				ID:   "idempotent-test",
				Type: "template",
				Template: &config.TemplateStep{
					Source:      src,
					Destination: dst,
				},
			}
			ctx := context.Background()

			// Call Evaluate twice
			result1, err1 := plugin.Evaluate(ctx, step)
			result2, err2 := plugin.Evaluate(ctx, step)

			require.NoError(t, err1, "First Evaluate() should not return an error")
			require.NoError(t, err2, "Second Evaluate() should not return an error")

			// Results should be equivalent for missing template
			require.Equal(t, result1.CurrentState, result2.CurrentState, "CurrentState should be consistent across calls")
			require.Equal(t, result1.RequiresAction, result2.RequiresAction, "RequiresAction should be consistent across calls")
		})
	})
}

func TestTemplateConvertError(t *testing.T) {
	t.Run("wraps validation errors", func(t *testing.T) {
		err := streamyerrors.NewValidationError("field", "invalid", nil)
		converted := convertError("tmpl", err)

		var pluginErr *plugin.ValidationError
		require.ErrorAs(t, converted, &pluginErr)
		require.Equal(t, "tmpl", pluginErr.StepID())
	})

	t.Run("wraps execution errors", func(t *testing.T) {
		err := streamyerrors.NewExecutionError("legacy", errors.New("boom"))
		converted := convertError("tmpl2", err)

		var pluginErr *plugin.ExecutionError
		require.ErrorAs(t, converted, &pluginErr)
		require.Equal(t, "tmpl2", pluginErr.StepID())
	})

	t.Run("wraps unknown errors as execution errors", func(t *testing.T) {
		err := errors.New("other failure")
		converted := convertError("tmpl3", err)

		var pluginErr *plugin.ExecutionError
		require.ErrorAs(t, converted, &pluginErr)
		require.Equal(t, "tmpl3", pluginErr.StepID())
	})
}
