package errors

import (
	stdErrors "errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseErrorWrapsUnderlying(t *testing.T) {
	t.Parallel()

	underlying := fmt.Errorf("unexpected token")
	err := NewParseError("config.yaml", 12, underlying)

	var parseErr *ParseError
	require.ErrorAs(t, err, &parseErr)
	require.Equal(t, "config.yaml", parseErr.Path)
	require.Equal(t, 12, parseErr.Line)
	require.True(t, stdErrors.Is(err, underlying))
	require.Contains(t, err.Error(), "config.yaml")
}

func TestValidationErrorAggregatesFields(t *testing.T) {
	t.Parallel()

	err := NewValidationError("steps[1].depends_on", "references unknown step", nil)

	var validationErr *ValidationError
	require.ErrorAs(t, err, &validationErr)
	require.Equal(t, "steps[1].depends_on", validationErr.Field)
	require.Contains(t, validationErr.Message, "references unknown step")
}

func TestExecutionErrorIncludesStepContext(t *testing.T) {
	t.Parallel()

	underlying := stdErrors.New("command failed")
	err := NewExecutionError("install_git", underlying)

	var executionErr *ExecutionError
	require.ErrorAs(t, err, &executionErr)
	require.Equal(t, "install_git", executionErr.StepID)
	require.True(t, stdErrors.Is(err, underlying))
}

func TestPluginErrorIncludesPluginName(t *testing.T) {
	t.Parallel()

	underlying := stdErrors.New("not supported")
	err := NewPluginError("command", underlying)

	var pluginErr *PluginError
	require.ErrorAs(t, err, &pluginErr)
	require.Equal(t, "command", pluginErr.Plugin)
	require.True(t, stdErrors.Is(err, underlying))
}

func TestParseErrorNilHandling(t *testing.T) {
	t.Parallel()

	t.Run("nil error returns empty string", func(t *testing.T) {
		t.Parallel()
		var err *ParseError
		require.Equal(t, "", err.Error())
	})

	t.Run("nil error unwrap returns nil", func(t *testing.T) {
		t.Parallel()
		var err *ParseError
		require.Nil(t, err.Unwrap())
	})

	t.Run("error without line number", func(t *testing.T) {
		t.Parallel()
		err := NewParseError("config.yaml", 0, fmt.Errorf("syntax error"))
		require.Contains(t, err.Error(), "config.yaml")
		require.NotContains(t, err.Error(), ":0:")
	})

	t.Run("error with line number", func(t *testing.T) {
		t.Parallel()
		err := NewParseError("config.yaml", 42, fmt.Errorf("syntax error"))
		require.Contains(t, err.Error(), "config.yaml:42")
	})
}

func TestValidationErrorNilHandling(t *testing.T) {
	t.Parallel()

	t.Run("nil error returns empty string", func(t *testing.T) {
		t.Parallel()
		var err *ValidationError
		require.Equal(t, "", err.Error())
	})

	t.Run("nil error unwrap returns nil", func(t *testing.T) {
		t.Parallel()
		var err *ValidationError
		require.Nil(t, err.Unwrap())
	})

	t.Run("error without field", func(t *testing.T) {
		t.Parallel()
		err := NewValidationError("", "something went wrong", nil)
		msg := err.Error()
		require.Contains(t, msg, "validation error")
		require.Contains(t, msg, "something went wrong")
		require.NotContains(t, msg, ": :")
	})

	t.Run("error with field", func(t *testing.T) {
		t.Parallel()
		err := NewValidationError("steps[0].id", "required field", nil)
		require.Contains(t, err.Error(), "steps[0].id")
		require.Contains(t, err.Error(), "required field")
	})

	t.Run("error with underlying error", func(t *testing.T) {
		t.Parallel()
		underlying := fmt.Errorf("underlying issue")
		err := NewValidationError("field", "message", underlying)

		var validationErr *ValidationError
		require.ErrorAs(t, err, &validationErr)
		require.True(t, stdErrors.Is(err, underlying))
	})
}

func TestExecutionErrorNilHandling(t *testing.T) {
	t.Parallel()

	t.Run("nil error returns empty string", func(t *testing.T) {
		t.Parallel()
		var err *ExecutionError
		require.Equal(t, "", err.Error())
	})

	t.Run("nil error unwrap returns nil", func(t *testing.T) {
		t.Parallel()
		var err *ExecutionError
		require.Nil(t, err.Unwrap())
	})

	t.Run("error without step id", func(t *testing.T) {
		t.Parallel()
		underlying := fmt.Errorf("command failed")
		err := NewExecutionError("", underlying)
		msg := err.Error()
		require.Contains(t, msg, "execution error")
		require.Contains(t, msg, "command failed")
	})

	t.Run("error with step id", func(t *testing.T) {
		t.Parallel()
		underlying := fmt.Errorf("timeout")
		err := NewExecutionError("install_package", underlying)
		require.Contains(t, err.Error(), "install_package")
		require.Contains(t, err.Error(), "timeout")
	})
}

func TestPluginErrorNilHandling(t *testing.T) {
	t.Parallel()

	t.Run("nil error returns empty string", func(t *testing.T) {
		t.Parallel()
		var err *PluginError
		require.Equal(t, "", err.Error())
	})

	t.Run("nil error unwrap returns nil", func(t *testing.T) {
		t.Parallel()
		var err *PluginError
		require.Nil(t, err.Unwrap())
	})

	t.Run("error without plugin name", func(t *testing.T) {
		t.Parallel()
		underlying := fmt.Errorf("failed")
		err := NewPluginError("", underlying)
		msg := err.Error()
		require.Contains(t, msg, "plugin error")
		require.Contains(t, msg, "failed")
	})

	t.Run("error with plugin name", func(t *testing.T) {
		t.Parallel()
		underlying := fmt.Errorf("not supported")
		err := NewPluginError("repo", underlying)
		require.Contains(t, err.Error(), "[repo]")
		require.Contains(t, err.Error(), "not supported")
	})

	t.Run("error with nil underlying", func(t *testing.T) {
		t.Parallel()
		err := NewPluginError("test", nil)

		var pluginErr *PluginError
		require.ErrorAs(t, err, &pluginErr)
		require.Equal(t, "test", pluginErr.Plugin)
		require.Empty(t, pluginErr.Message)
	})
}
