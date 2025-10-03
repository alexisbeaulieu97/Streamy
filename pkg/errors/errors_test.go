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
