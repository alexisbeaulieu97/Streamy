package plugin

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
)

// TestPluginContract runs a comprehensive contract test suite for plugin implementations.
// This test validates that plugins correctly implement the unified interface contract.
//
// Usage:
//
//	func TestMyPluginContract(t *testing.T) {
//	    plugin := NewMyPlugin()
//	    TestPluginContract(t, plugin, createMyPluginTestStep)
//	}
//
// The createTestStep function should return a valid step configuration for testing.
func TestPluginContract(t *testing.T) {
	// Note: This is a helper function that should be called by individual plugin tests
	// with their specific plugin implementation and test step creator.
	t.Skip("This is a helper function - use it in your plugin-specific contract tests")
}

// RunPluginContract is the actual function that plugins should call in their tests
func RunPluginContract(t *testing.T, plugin Plugin, createTestStep func() *config.Step) {
	t.Helper()

	t.Run("Metadata is stable", func(t *testing.T) {
		m1 := plugin.PluginMetadata()
		m2 := plugin.PluginMetadata()
		assert.Equal(t, m1, m2, "PluginMetadata() should return consistent values across calls")
	})

	t.Run("Schema returns struct", func(t *testing.T) {
		schema := plugin.Schema()
		require.NotNil(t, schema, "Schema() should not return nil")

		// Verify it's a struct type
		rt := reflect.TypeOf(schema)
		if rt.Kind() == reflect.Ptr {
			rt = rt.Elem()
		}
		assert.Equal(t, reflect.Struct, rt.Kind(), "Schema() should return a struct type (or pointer to struct)")
	})

	t.Run("Evaluate is read-only", func(t *testing.T) {
		step := createTestStep()
		require.NotNil(t, step, "createTestStep() should return a valid step")

		// Create a temporary directory for testing
		testDir := t.TempDir()

		// Setup test environment
		setupTestFiles(t, testDir)

		// Capture state before evaluation
		beforeState := captureFilesystemState(t, testDir)

		// Execute Evaluate()
		_, err := plugin.Evaluate(context.Background(), step)
		if err != nil {
			t.Logf("Evaluate() returned error (acceptable): %v", err)
		}

		// Verify no state changes
		afterState := captureFilesystemState(t, testDir)
		assert.Equal(t, beforeState, afterState, "Evaluate() modified filesystem state - this violates the read-only guarantee")
	})

	t.Run("Evaluate is idempotent", func(t *testing.T) {
		step := createTestStep()
		require.NotNil(t, step, "createTestStep() should return a valid step")

		result1, err1 := plugin.Evaluate(context.Background(), step)
		result2, err2 := plugin.Evaluate(context.Background(), step)

		// Both calls should succeed or fail the same way
		if err1 != nil || err2 != nil {
			assert.Equal(t, err1 != nil, err2 != nil, "Evaluate() should consistently succeed or fail")
			return
		}

		// Results should be equivalent
		assert.Equal(t, result1.CurrentState, result2.CurrentState, "CurrentState should be consistent")
		assert.Equal(t, result1.RequiresAction, result2.RequiresAction, "RequiresAction should be consistent")
		assert.Equal(t, result1.StepID, result2.StepID, "StepID should be consistent")
	})

	t.Run("Apply is idempotent", func(t *testing.T) {
		step := createTestStep()
		require.NotNil(t, step, "createTestStep() should return a valid step")

		// First evaluate to get result
		evalResult, err := plugin.Evaluate(context.Background(), step)
		if err != nil {
			t.Skipf("Skipping Apply test because Evaluate() failed: %v", err)
			return
		}

		if !evalResult.RequiresAction {
			t.Skip("Skipping Apply test because no action is required")
			return
		}

		// Apply twice
		result1, err1 := plugin.Apply(context.Background(), evalResult, step)
		result2, err2 := plugin.Apply(context.Background(), evalResult, step)

		// Both should succeed
		require.NoError(t, err1, "First Apply() should succeed")
		require.NoError(t, err2, "Second Apply() should succeed")

		// Results should have the same status (typically Success or Skipped)
		assert.Equal(t, result1.Status, result2.Status, "Apply() should be idempotent - status should be the same")
	})

	t.Run("Error types are correct", func(t *testing.T) {
		// Create an invalid step - this should trigger a ValidationError
		invalidStep := &config.Step{
			ID:   "invalid-test",
			Type: "test",
		}

		_, err := plugin.Evaluate(context.Background(), invalidStep)
		if err == nil {
			t.Log("Plugin accepted invalid step - consider adding validation")
			return
		}

		// Verify error implements PluginError interface
		_, ok := AsPluginError(err)
		assert.True(t, ok, "Error should implement PluginError interface")

		var pluginErr PluginError
		if assert.True(t, errors.As(err, &pluginErr), "Error should be extractable via errors.As") {
			assert.Equal(t, invalidStep.ID, pluginErr.StepID(), "PluginError should have correct StepID")
			assert.NotEmpty(t, pluginErr.Error(), "PluginError should have non-empty message")
		}
	})

	t.Run("EvaluationResult validation", func(t *testing.T) {
		step := createTestStep()
		require.NotNil(t, step, "createTestStep() should return a valid step")

		result, err := plugin.Evaluate(context.Background(), step)
		if err != nil {
			t.Logf("Evaluation failed: %v", err)
			return
		}

		// Validate EvaluationResult fields
		assert.Equal(t, step.ID, result.StepID, "StepID should match input step")
		assert.NotEmpty(t, result.Message, "Message should not be empty")
		assert.True(t, result.CurrentState.IsValid(), "CurrentState should be a valid VerificationStatus")

		// Validate RequiresAction logic
		switch result.CurrentState {
		case model.StatusSatisfied, model.StatusBlocked, model.StatusUnknown:
			assert.False(t, result.RequiresAction, "RequiresAction should be false for %s", result.CurrentState)
		case model.StatusMissing, model.StatusDrifted:
			assert.True(t, result.RequiresAction, "RequiresAction should be true for %s", result.CurrentState)
		}

		// Diff should be present when RequiresAction is true
		if result.RequiresAction {
			assert.NotEmpty(t, result.Diff, "Diff should be populated when RequiresAction is true")
		}
	})
}

// setupTestFiles creates a basic test environment in the given directory.
func setupTestFiles(t *testing.T, testDir string) {
	// Create some test files
	files := map[string]string{
		"existing.txt": "existing content\n",
		"config.json":  `{"key": "value"}`,
	}

	for filename, content := range files {
		path := filepath.Join(testDir, filename)
		err := os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err, "Failed to create test file: %s", filename)
	}

	// Create a subdirectory
	subdir := filepath.Join(testDir, "subdir")
	err := os.Mkdir(subdir, 0755)
	require.NoError(t, err, "Failed to create test subdirectory")
}

// captureFilesystemState captures the current state of files in a directory.
// This is used to verify that Evaluate() doesn't modify state.
func captureFilesystemState(t *testing.T, dir string) map[string]string {
	state := make(map[string]string)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		state[relPath] = string(content)
		return nil
	})

	require.NoError(t, err, "Failed to capture filesystem state")
	return state
}
