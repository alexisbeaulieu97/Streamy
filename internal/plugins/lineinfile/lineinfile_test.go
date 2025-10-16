package lineinfileplugin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
	"gopkg.in/yaml.v3"
)

func TestLineinfilePlugin_Metadata(t *testing.T) {
	t.Parallel()

	p := New()
	meta := p.PluginMetadata()

	require.NotEmpty(t, meta.Name)
	require.NotEmpty(t, meta.Version)
	require.Equal(t, "line_in_file", meta.Name)
}

func TestLineinfilePlugin_Schema(t *testing.T) {
	t.Parallel()

	p := New()
	schema := p.Schema()

	require.NotNil(t, schema)
	_, ok := schema.(config.LineInFileStep)
	require.True(t, ok, "schema should be of type LineinfileStep")
}

func makeLineInFileStep(t *testing.T, id string, cfg config.LineInFileStep) *config.Step {
	t.Helper()
	step := &config.Step{ID: id, Type: "lineinfile"}
	require.NoError(t, step.SetConfig(cfg))
	return step
}

func TestLineinfilePlugin_EvaluateMissingFile(t *testing.T) {
	t.Parallel()

	filePath := filepath.Join(t.TempDir(), "test.txt")

	step := makeLineInFileStep(t, "missing_file", config.LineInFileStep{File: filePath, Line: "test line", State: "present"})

	p := New()

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusMissing, evalResult.CurrentState)
	require.True(t, evalResult.RequiresAction)
	require.Contains(t, evalResult.Message, "file does not exist")
}

func TestLineinfilePlugin_EvaluateSatisfied(t *testing.T) {
	t.Parallel()

	filePath := filepath.Join(t.TempDir(), "test.txt")
	line := "test line"

	// Create file with the line
	require.NoError(t, os.WriteFile(filePath, []byte(line+"\n"), 0o644))

	step := makeLineInFileStep(t, "satisfied_line", config.LineInFileStep{File: filePath, Line: line, State: "present"})

	p := New()

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusSatisfied, evalResult.CurrentState)
	require.False(t, evalResult.RequiresAction)
}

func TestLineinfilePlugin_EvaluateDrifted(t *testing.T) {
	t.Parallel()

	filePath := filepath.Join(t.TempDir(), "test.txt")
	line := "test line"

	// Create file without the line
	require.NoError(t, os.WriteFile(filePath, []byte("different line\n"), 0o644))

	step := makeLineInFileStep(t, "drifted_line", config.LineInFileStep{File: filePath, Line: line, State: "present"})

	p := New()

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusDrifted, evalResult.CurrentState)
	require.True(t, evalResult.RequiresAction)
}

func TestLineinfilePlugin_ApplyAddLine(t *testing.T) {
	t.Parallel()

	filePath := filepath.Join(t.TempDir(), "test.txt")
	line := "test line"

	// Create empty file
	require.NoError(t, os.WriteFile(filePath, []byte(""), 0o644))

	step := makeLineInFileStep(t, "add_line", config.LineInFileStep{File: filePath, Line: line, State: "present"})

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

	// Verify line was added
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	require.Contains(t, string(content), line)
}

func TestLineinfilePlugin_ApplyRemoveLine(t *testing.T) {
	t.Parallel()

	filePath := filepath.Join(t.TempDir(), "test.txt")
	line := "test line"

	// Create file with the line
	require.NoError(t, os.WriteFile(filePath, []byte("some line\n"+line+"\nanother line\n"), 0o644))

	step := makeLineInFileStep(t, "remove_line", config.LineInFileStep{File: filePath, Line: line, State: "absent", Match: line})

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

	// Verify line was removed
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	require.NotContains(t, string(content), line)
}

func TestLineinfilePlugin_EvaluateAbsentWhenLineExists(t *testing.T) {
	t.Parallel()

	filePath := filepath.Join(t.TempDir(), "test.txt")
	line := "test line"

	// Create file with the line
	require.NoError(t, os.WriteFile(filePath, []byte(line+"\n"), 0o644))

	step := makeLineInFileStep(t, "line_exists_absent", config.LineInFileStep{File: filePath, Line: line, State: "absent", Match: line})

	p := New()

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusDrifted, evalResult.CurrentState)
	require.True(t, evalResult.RequiresAction)
}

func TestLineinfilePlugin_EvaluateUsesRawConfigWhenStructNil(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "test.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("hello\n"), 0o644))

	yamlStr := fmt.Sprintf(`
id: raw_lineinfile
type: lineinfile
file: %s
line: world
state: present
`, filePath)

	var step config.Step
	require.NoError(t, yaml.Unmarshal([]byte(yamlStr), &step))

	p := New()

	evalResult, err := p.Evaluate(context.Background(), &step)
	require.NoError(t, err)
	require.True(t, evalResult.RequiresAction)
}

func TestLineinfilePlugin_EvaluateAbsentWhenLineMissing(t *testing.T) {
	t.Parallel()

	filePath := filepath.Join(t.TempDir(), "test.txt")
	line := "test line"

	// Create file without the line
	require.NoError(t, os.WriteFile(filePath, []byte("different line\n"), 0o644))

	step := makeLineInFileStep(t, "line_missing_absent", config.LineInFileStep{File: filePath, Line: line, State: "absent", Match: line})

	p := New()

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusSatisfied, evalResult.CurrentState)
	require.False(t, evalResult.RequiresAction)
}

func TestLineinfilePlugin_EvaluateWithRegex(t *testing.T) {
	t.Parallel()

	filePath := filepath.Join(t.TempDir(), "test.txt")

	// Create file with a line matching regex
	require.NoError(t, os.WriteFile(filePath, []byte("version: 1.2.3\n"), 0o644))

	step := makeLineInFileStep(t, "regex_match", config.LineInFileStep{File: filePath, Line: "version: .*", State: "present", Match: "version: 2.0.0"})

	p := New()

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusDrifted, evalResult.CurrentState)
	require.True(t, evalResult.RequiresAction)
}

func TestLineinfilePlugin_EvaluateErrors(t *testing.T) {
	t.Run("returns error when lineinfile config is nil", func(t *testing.T) {
		p := New()

		step := &config.Step{ID: "test_lineinfile", Type: "lineinfile"}

		_, err := p.Evaluate(context.Background(), step)
		require.Error(t, err)
		require.Contains(t, err.Error(), "lineinfile configuration missing")
	})

	t.Run("returns error when line is empty and no regex", func(t *testing.T) {
		p := New()

		step := makeLineInFileStep(t, "empty_line", config.LineInFileStep{File: "/tmp/test.txt", Line: "", State: "present"})

		_, err := p.Evaluate(context.Background(), step)
		require.Error(t, err)
	})
}

func TestLineinfilePlugin_ApplyErrors(t *testing.T) {
	t.Run("returns error when lineinfile config is nil", func(t *testing.T) {
		p := New()

		step := &config.Step{ID: "test_lineinfile", Type: "lineinfile"}

		evalResult := &model.EvaluationResult{
			StepID:         step.ID,
			CurrentState:   model.StatusMissing,
			RequiresAction: true,
			Message:        "Test",
		}

		_, err := p.Apply(context.Background(), evalResult, step)
		require.Error(t, err)
		require.Contains(t, err.Error(), "lineinfile configuration missing")
	})

	t.Run("returns error when file cannot be created", func(t *testing.T) {
		p := New()

		step := makeLineInFileStep(t, "invalid_path", config.LineInFileStep{File: "/invalid/path/test.txt", Line: "test line", State: "present"})

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

func TestLineinfileConvertError(t *testing.T) {
	t.Run("wraps validation errors", func(t *testing.T) {
		err := streamyerrors.NewValidationError("field", "invalid", nil)
		converted := convertError("line", err)

		var pluginErr *plugin.ValidationError
		require.ErrorAs(t, converted, &pluginErr)
		require.Equal(t, "line", pluginErr.StepID())
	})

	t.Run("wraps execution errors", func(t *testing.T) {
		err := streamyerrors.NewExecutionError("legacy", errors.New("boom"))
		converted := convertError("line2", err)

		var pluginErr *plugin.ExecutionError
		require.ErrorAs(t, converted, &pluginErr)
		require.Equal(t, "line2", pluginErr.StepID())
	})

	t.Run("wraps unknown errors as execution", func(t *testing.T) {
		err := errors.New("other failure")
		converted := convertError("line3", err)

		var pluginErr *plugin.ExecutionError
		require.ErrorAs(t, converted, &pluginErr)
		require.Equal(t, "line3", pluginErr.StepID())
	})
}

// Contract tests for the new plugin interface
func TestLineinfilePlugin_Contract(t *testing.T) {
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
			_, ok := schema.(config.LineInFileStep)
			require.True(t, ok, "Schema() should return a LineinfileStep struct")
		})

		t.Run("Evaluate is idempotent", func(t *testing.T) {
			lineStep := makeLineInFileStep(t, "idempotent-test", config.LineInFileStep{File: "/nonexistent/test.txt", Line: "test line", State: "present"})
			ctx := context.Background()

			// Call Evaluate twice
			result1, err1 := plugin.Evaluate(ctx, lineStep)
			result2, err2 := plugin.Evaluate(ctx, lineStep)

			require.NoError(t, err1, "First Evaluate() should not return an error")
			require.NoError(t, err2, "Second Evaluate() should not return an error")

			// Results should be equivalent for non-existent file
			require.Equal(t, result1.CurrentState, result2.CurrentState, "CurrentState should be consistent across calls")
			require.Equal(t, result1.RequiresAction, result2.RequiresAction, "RequiresAction should be consistent across calls")
		})
	})
}
