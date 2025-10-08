package plugin

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidationError(t *testing.T) {
	stepID := "test-step"
	underlyingErr := errors.New("required field missing")
	err := NewValidationError(stepID, underlyingErr)

	t.Run("Error returns formatted message", func(t *testing.T) {
		expected := "validation error in step test-step: required field missing"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("StepID returns correct ID", func(t *testing.T) {
		assert.Equal(t, stepID, err.StepID())
	})

	t.Run("Unwrap returns underlying error", func(t *testing.T) {
		assert.Equal(t, underlyingErr, err.Unwrap())
	})

	t.Run("Is matches ValidationError type", func(t *testing.T) {
		var targetErr *ValidationError
		assert.True(t, errors.As(err, &targetErr))
		assert.Equal(t, err, targetErr)
	})

	t.Run("errors.Is works correctly", func(t *testing.T) {
		assert.True(t, errors.Is(err, &ValidationError{}))
		assert.False(t, errors.Is(err, &ExecutionError{}))
	})
}

func TestExecutionError(t *testing.T) {
	stepID := "test-step"
	underlyingErr := errors.New("command failed with exit code 1")
	err := NewExecutionError(stepID, underlyingErr)

	t.Run("Error returns formatted message", func(t *testing.T) {
		expected := "execution error in step test-step: command failed with exit code 1"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("StepID returns correct ID", func(t *testing.T) {
		assert.Equal(t, stepID, err.StepID())
	})

	t.Run("Unwrap returns underlying error", func(t *testing.T) {
		assert.Equal(t, underlyingErr, err.Unwrap())
	})

	t.Run("Is matches ExecutionError type", func(t *testing.T) {
		var targetErr *ExecutionError
		assert.True(t, errors.As(err, &targetErr))
		assert.Equal(t, err, targetErr)
	})

	t.Run("errors.Is works correctly", func(t *testing.T) {
		assert.True(t, errors.Is(err, &ExecutionError{}))
		assert.False(t, errors.Is(err, &ValidationError{}))
	})
}

func TestStateError(t *testing.T) {
	stepID := "test-step"
	underlyingErr := errors.New("cannot read file permissions")
	err := NewStateError(stepID, underlyingErr)

	t.Run("Error returns formatted message", func(t *testing.T) {
		expected := "state error in step test-step: cannot read file permissions"
		assert.Equal(t, expected, err.Error())
	})

	t.Run("StepID returns correct ID", func(t *testing.T) {
		assert.Equal(t, stepID, err.StepID())
	})

	t.Run("Unwrap returns underlying error", func(t *testing.T) {
		assert.Equal(t, underlyingErr, err.Unwrap())
	})

	t.Run("Is matches StateError type", func(t *testing.T) {
		var targetErr *StateError
		assert.True(t, errors.As(err, &targetErr))
		assert.Equal(t, err, targetErr)
	})

	t.Run("errors.Is works correctly", func(t *testing.T) {
		assert.True(t, errors.Is(err, &StateError{}))
		assert.False(t, errors.Is(err, &ValidationError{}))
	})
}

func TestAsPluginError(t *testing.T) {
	t.Run("ValidationError is recognized", func(t *testing.T) {
		originalErr := NewValidationError("step1", errors.New("bad config"))
		pluginErr, ok := AsPluginError(originalErr)
		require.True(t, ok)
		assert.Equal(t, originalErr, pluginErr)
	})

	t.Run("ExecutionError is recognized", func(t *testing.T) {
		originalErr := NewExecutionError("step2", errors.New("command failed"))
		pluginErr, ok := AsPluginError(originalErr)
		require.True(t, ok)
		assert.Equal(t, originalErr, pluginErr)
	})

	t.Run("StateError is recognized", func(t *testing.T) {
		originalErr := NewStateError("step3", errors.New("cannot read state"))
		pluginErr, ok := AsPluginError(originalErr)
		require.True(t, ok)
		assert.Equal(t, originalErr, pluginErr)
	})

	t.Run("Regular error is not recognized", func(t *testing.T) {
		regularErr := errors.New("regular error")
		pluginErr, ok := AsPluginError(regularErr)
		assert.False(t, ok)
		assert.Nil(t, pluginErr)
	})

	t.Run("Wrapped PluginError is recognized", func(t *testing.T) {
		originalErr := NewValidationError("step4", errors.New("bad config"))
		wrappedErr := fmt.Errorf("additional context: %w", originalErr)
		pluginErr, ok := AsPluginError(wrappedErr)
		require.True(t, ok)
		assert.Equal(t, originalErr, pluginErr)
	})
}

func TestPluginErrorInterface(t *testing.T) {
	t.Run("All error types implement PluginError", func(t *testing.T) {
		var _ PluginError = &ValidationError{}
		var _ PluginError = &ExecutionError{}
		var _ PluginError = &StateError{}
	})
}

func TestErrorChaining(t *testing.T) {
	t.Run("Error chaining works", func(t *testing.T) {
		rootErr := errors.New("root cause")
		valErr := NewValidationError("test-step", rootErr)
		wrappedErr := fmt.Errorf("wrapper: %w", valErr)

		// Test that we can unwrap to the root
		assert.True(t, errors.Is(wrappedErr, rootErr))

		// Test that we can extract the ValidationError
		var pluginErr PluginError
		assert.True(t, errors.As(wrappedErr, &pluginErr))
		assert.Equal(t, "test-step", pluginErr.StepID())
	})
}
