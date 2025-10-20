package commandplugin

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	require "github.com/stretchr/testify/require"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
)

func TestEvaluateWithCheckCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell assumptions do not hold on Windows")
	}

	binDir := t.TempDir()
	writeScript(t, binDir, "check-script", `#!/bin/sh
if [ "$EXPECT_FAIL" = "1" ]; then
  exit 1
fi
exit 0
`)

	originalPath := os.Getenv("PATH")

	t.Cleanup(func() { _ = os.Setenv("PATH", originalPath) })
	require.NoError(t, os.Setenv("PATH", binDir+":"+originalPath))

	step := domainpipeline.Step{
		ID:   "run_command",
		Type: domainpipeline.StepTypeCommand,
		Config: map[string]interface{}{
			"command": "echo hello",
			"check":   "check-script",
		},
	}

	plugin := New()

	require.NoError(t, os.Setenv("EXPECT_FAIL", "0"))

	result, err := plugin.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.False(t, result.RequiresAction)
	require.Equal(t, string(domainpipeline.VerificationSatisfied), result.CurrentState)

	require.NoError(t, os.Setenv("EXPECT_FAIL", "1"))

	result, err = plugin.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, result.RequiresAction)
	require.Equal(t, string(domainpipeline.VerificationFailed), result.CurrentState)
}

func TestApplyRunsCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell assumptions do not hold on Windows")
	}

	workDir := t.TempDir()
	outputFile := filepath.Join(workDir, "result.txt")

	step := domainpipeline.Step{
		ID:   "run_command",
		Type: domainpipeline.StepTypeCommand,
		Config: map[string]interface{}{
			"command": "echo $CUSTOM_VALUE > result.txt",
			"workdir": workDir,
			"env": map[string]interface{}{
				"CUSTOM_VALUE": "streamy",
			},
		},
	}

	eval := &domainpipeline.EvaluationResult{
		RequiresAction: true,
		CurrentState:   string(domainpipeline.VerificationFailed),
	}

	result, err := New().Apply(context.Background(), eval, step)
	require.NoError(t, err)
	require.Equal(t, domainpipeline.StatusSuccess, result.Status)

	data, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	require.Equal(t, "streamy\n", string(data))
}

func TestMetadata(t *testing.T) {
	meta := New().Metadata()
	require.Equal(t, "command", meta.Name)
	require.Equal(t, domainplugin.TypeCommand, meta.Type)
}

func TestEvaluateMissingConfig(t *testing.T) {
	step := domainpipeline.Step{ID: "missing", Type: domainpipeline.StepTypeCommand}
	_, err := New().Evaluate(context.Background(), step)
	require.Error(t, err)
}

func TestApplyMissingConfig(t *testing.T) {
	step := domainpipeline.Step{ID: "missing", Type: domainpipeline.StepTypeCommand}
	_, err := New().Apply(context.Background(), nil, step)
	require.Error(t, err)
}

func TestDecodeConfigEnvCastsNonStringValues(t *testing.T) {
	step := domainpipeline.Step{
		ID:   "env",
		Type: domainpipeline.StepTypeCommand,
		Config: map[string]interface{}{
			"command": "echo hello",
			"env": map[string]interface{}{
				"STRING": "value",
				"NUMBER": 42,
				"BOOL":   true,
				"NIL":    nil,
			},
		},
	}

	cfg, err := decodeConfig(step)
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		"STRING": "value",
		"NUMBER": "42",
		"BOOL":   "true",
	}, cfg.Env)
	require.NotContains(t, cfg.Env, "NIL")
}

func TestContextCancellation(t *testing.T) {
	step := domainpipeline.Step{
		ID:   "test_step",
		Type: domainpipeline.StepTypeCommand,
		Config: map[string]interface{}{
			"command": "echo hello",
		},
	}

	plugin := New()

	t.Run("Evaluate with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		result, err := plugin.Evaluate(ctx, step)
		require.Error(t, err)
		require.Nil(t, result)

		// Should be a cancellation error
		require.Contains(t, err.Error(), "cancelled")
	})

	t.Run("Apply with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		eval := &domainpipeline.EvaluationResult{
			RequiresAction: true,
			CurrentState:   string(domainpipeline.VerificationFailed),
		}

		result, err := plugin.Apply(ctx, eval, step)
		require.Error(t, err)
		require.Nil(t, result)

		// Should be a cancellation error
		require.Contains(t, err.Error(), "cancelled")
	})
}

func TestShellDetection(t *testing.T) {
	testCases := []struct {
		name        string
		explicit    string
		expectError bool
		expected    string
	}{
		{"explicit bash", "/bin/bash", false, "/bin/bash"},
		{"explicit sh", "/bin/sh", false, "/bin/sh"},
		{"empty", "", false, ""},                                           // Will try to find bash/sh
		{"nonexistent", "/nonexistent/shell", false, "/nonexistent/shell"}, // Explicit shell is returned as-is
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			shell, args, err := determineShell(tc.explicit)

			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tc.explicit != "" {
				require.Equal(t, tc.expected, shell)
				require.Equal(t, []string{"-c"}, args)
			}
		})
	}
}

func TestWorkingDirectoryHandling(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX-specific tests")
	}

	step := domainpipeline.Step{
		ID:   "test_step",
		Type: domainpipeline.StepTypeCommand,
		Config: map[string]interface{}{
			"command": "echo hello",
		},
	}

	t.Run("valid working directory", func(t *testing.T) {
		workDir := t.TempDir()
		step.Config["workdir"] = workDir

		result, err := New().Evaluate(context.Background(), step)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("nonexistent working directory", func(t *testing.T) {
		step.Config["workdir"] = "/nonexistent/directory/path"

		result, err := New().Evaluate(context.Background(), step)
		// This should not fail during evaluation, only during execution
		require.NoError(t, err)
		require.NotNil(t, result)
	})
}

func TestEnvironmentVariableHandling(t *testing.T) {
	step := domainpipeline.Step{
		ID:   "test_step",
		Type: domainpipeline.StepTypeCommand,
		Config: map[string]interface{}{
			"command": "echo hello",
		},
	}

	t.Run("map[string]interface{} env", func(t *testing.T) {
		step.Config["env"] = map[string]interface{}{
			"VAR1": "value1",
			"VAR2": 42,
			"VAR3": true,
			"VAR4": nil, // Should be ignored
		}

		cfg, err := decodeConfig(step)
		require.NoError(t, err)

		require.Equal(t, "value1", cfg.Env["VAR1"])
		require.Equal(t, "42", cfg.Env["VAR2"])
		require.Equal(t, "true", cfg.Env["VAR3"])
		_, exists := cfg.Env["VAR4"]
		require.False(t, exists)
	})

	t.Run("map[string]string env", func(t *testing.T) {
		step.Config["env"] = map[string]string{
			"VAR1": "value1",
			"VAR2": "value2",
		}

		cfg, err := decodeConfig(step)
		require.NoError(t, err)

		require.Equal(t, "value1", cfg.Env["VAR1"])
		require.Equal(t, "value2", cfg.Env["VAR2"])
	})

	t.Run("invalid env type", func(t *testing.T) {
		step.Config["env"] = "not a map"

		cfg, err := decodeConfig(step)
		require.NoError(t, err) // Should not error, just ignore invalid env
		require.Empty(t, cfg.Env)
	})
}

func TestConfigurationValidation(t *testing.T) {
	testCases := []struct {
		name        string
		config      map[string]interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name: "missing command",
			config: map[string]interface{}{
				"check": "echo check",
			},
			expectError: true,
			errorMsg:    "command is required",
		},
		{
			name: "empty command",
			config: map[string]interface{}{
				"command": "",
			},
			expectError: true,
			errorMsg:    "command is required",
		},
		{
			name: "whitespace only command",
			config: map[string]interface{}{
				"command": "   ",
			},
			expectError: true,
			errorMsg:    "command is required",
		},
		{
			name: "valid command",
			config: map[string]interface{}{
				"command": "echo hello",
			},
			expectError: false,
		},
		{
			name: "valid with all fields",
			config: map[string]interface{}{
				"command": "echo hello",
				"check":   "echo check",
				"shell":   "/bin/bash",
				"workdir": "/tmp",
				"env": map[string]string{
					"VAR": "value",
				},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			step := domainpipeline.Step{
				ID:     "test_step",
				Type:   domainpipeline.StepTypeCommand,
				Config: tc.config,
			}

			cfg, err := decodeConfig(step)
			if tc.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errorMsg)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, cfg.Command)
			}
		})
	}
}

func TestEnsureEvaluationData(t *testing.T) {
	step := domainpipeline.Step{
		ID:   "test_step",
		Type: domainpipeline.StepTypeCommand,
		Config: map[string]interface{}{
			"command": "echo hello",
		},
	}

	t.Run("evaluation data exists and valid", func(t *testing.T) {
		plugin := New()
		data := &evaluationData{
			Shell:     "/bin/bash",
			ShellArgs: []string{"-c"},
			Command:   "echo hello",
		}

		eval := &domainpipeline.EvaluationResult{
			InternalData: data,
		}

		// Test indirectly by calling Apply with already satisfied evaluation
		result, err := plugin.Apply(context.Background(), eval, step)
		require.NoError(t, err)
		require.Equal(t, domainpipeline.StatusAlreadySatisfied, result.Status)
	})

	t.Run("evaluation data missing", func(t *testing.T) {
		plugin := New()
		eval := &domainpipeline.EvaluationResult{
			InternalData:   nil,
			RequiresAction: true,
		}

		// Test indirectly by calling Apply - it should handle missing evaluation data
		result, err := plugin.Apply(context.Background(), eval, step)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, domainpipeline.StatusSuccess, result.Status)
	})

	t.Run("evaluation data invalid type", func(t *testing.T) {
		plugin := New()
		eval := &domainpipeline.EvaluationResult{
			InternalData:   "not evaluation data",
			RequiresAction: true,
		}

		// Test indirectly by calling Apply - it should handle invalid evaluation data
		result, err := plugin.Apply(context.Background(), eval, step)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, domainpipeline.StatusSuccess, result.Status)
	})
}

func TestApplyAlreadySatisfied(t *testing.T) {
	step := domainpipeline.Step{
		ID:   "test_step",
		Type: domainpipeline.StepTypeCommand,
		Config: map[string]interface{}{
			"command": "echo hello",
		},
	}

	eval := &domainpipeline.EvaluationResult{
		RequiresAction: false,
		CurrentState:   string(domainpipeline.VerificationSatisfied),
	}

	result, err := New().Apply(context.Background(), eval, step)
	require.NoError(t, err)
	require.Equal(t, domainpipeline.StatusAlreadySatisfied, result.Status)
	require.Equal(t, "no changes needed", result.Message)
}

func TestGetStringHelper(t *testing.T) {
	testCases := []struct {
		name     string
		values   map[string]interface{}
		key      string
		expected string
		found    bool
	}{
		{
			name:     "string value",
			values:   map[string]interface{}{"key": "value"},
			key:      "key",
			expected: "value",
			found:    true,
		},
		{
			name:     "stringer value",
			values:   map[string]interface{}{"key": mockStringer{"test"}},
			key:      "key",
			expected: "test",
			found:    true,
		},
		{
			name:     "number value",
			values:   map[string]interface{}{"key": 42},
			key:      "key",
			expected: "42",
			found:    true,
		},
		{
			name:     "bool value",
			values:   map[string]interface{}{"key": true},
			key:      "key",
			expected: "true",
			found:    true,
		},
		{
			name:     "missing key",
			values:   map[string]interface{}{"other": "value"},
			key:      "key",
			expected: "",
			found:    false,
		},
		{
			name:     "nil map",
			values:   nil,
			key:      "key",
			expected: "",
			found:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, found := getString(tc.values, tc.key)
			require.Equal(t, tc.expected, result)
			require.Equal(t, tc.found, found)
		})
	}
}

func TestBuildEnvHelper(t *testing.T) {
	// Store original environment
	originalEnv := os.Environ()

	t.Cleanup(func() {
		// Restore original environment
		os.Clearenv()

		for _, env := range originalEnv {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				require.NoError(t, os.Setenv(parts[0], parts[1]))
			}
		}
	})

	// Set a test environment variable
	require.NoError(t, os.Setenv("EXISTING_VAR", "existing_value"))

	custom := map[string]string{
		"NEW_VAR": "new_value",
	}

	env := buildEnv(custom)

	// Should contain both existing and new variables
	envMap := make(map[string]string)

	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	require.Equal(t, "existing_value", envMap["EXISTING_VAR"])
	require.Equal(t, "new_value", envMap["NEW_VAR"])
}

func TestStepValidation(t *testing.T) {
	testCases := []struct {
		name        string
		step        domainpipeline.Step
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid step",
			step: domainpipeline.Step{
				ID:   "valid_step",
				Type: domainpipeline.StepTypeCommand,
				Config: map[string]interface{}{
					"command": "echo hello",
				},
			},
			expectError: false,
		},
		{
			name: "empty step ID",
			step: domainpipeline.Step{
				ID:   "",
				Type: domainpipeline.StepTypeCommand,
				Config: map[string]interface{}{
					"command": "echo hello",
				},
			},
			expectError: true,
			errorMsg:    "command step missing id",
		},
		{
			name: "whitespace only step ID",
			step: domainpipeline.Step{
				ID:   "   ",
				Type: domainpipeline.StepTypeCommand,
				Config: map[string]interface{}{
					"command": "echo hello",
				},
			},
			expectError: true,
			errorMsg:    "command step missing id",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := decodeConfig(tc.step)
			if tc.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Helper types
type mockStringer struct {
	msg string
}

func (m mockStringer) String() string {
	return m.msg
}

func writeScript(t *testing.T, dir, name, content string) {
	t.Helper()

	scriptPath := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(scriptPath, []byte(content), 0o755))
}
