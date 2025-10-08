package commandplugin

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	pluginpkg "github.com/alexisbeaulieu97/streamy/internal/plugin"
)

func TestCommandPlugin_EvaluateUsesCheckCommand(t *testing.T) {
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

	step := &config.Step{
		ID:   "run_command",
		Type: "command",
		Command: &config.CommandStep{
			Command: "echo hello",
			Check:   "check-script",
		},
	}

	p := New()
	require.Implements(t, (*pluginpkg.Plugin)(nil), p)

	require.NoError(t, os.Setenv("EXPECT_FAIL", "0"))
	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusSatisfied, evalResult.CurrentState)
	require.False(t, evalResult.RequiresAction)

	require.NoError(t, os.Setenv("EXPECT_FAIL", "1"))
	evalResult, err = p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusMissing, evalResult.CurrentState)
	require.True(t, evalResult.RequiresAction)
}

func TestCommandPlugin_ApplyRunsCommandWithEnvAndWorkdir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell assumptions do not hold on Windows")
	}
	workDir := t.TempDir()
	outputFile := filepath.Join(workDir, "result.txt")

	step := &config.Step{
		ID:   "run_command",
		Type: "command",
		Command: &config.CommandStep{
			Command: "echo $CUSTOM_VALUE > result.txt",
			WorkDir: workDir,
			Env: map[string]string{
				"CUSTOM_VALUE": "streamy",
			},
		},
	}

	p := New()

	// First evaluate to get result
	evalResult := &model.EvaluationResult{
		StepID:         step.ID,
		CurrentState:   model.StatusMissing,
		RequiresAction: true,
		Message:        "Command needs to be executed",
	}

	res, err := p.Apply(context.Background(), evalResult, step)
	require.NoError(t, err)
	require.Equal(t, "success", strings.ToLower(res.Status))

	data, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	require.Equal(t, "streamy\n", string(data))
}

func TestCommandPlugin_EvaluateForDryRun(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell assumptions do not hold on Windows")
	}
	workDir := t.TempDir()

	step := &config.Step{
		ID:   "run_command",
		Type: "command",
		Command: &config.CommandStep{
			Command: "touch should_not_exist",
			WorkDir: workDir,
		},
	}

	p := New()

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusUnknown, evalResult.CurrentState)
	require.True(t, evalResult.RequiresAction)
}

func TestCommandPlugin_Metadata(t *testing.T) {
	t.Parallel()

	p := New()
	meta := p.PluginMetadata()

	require.NotEmpty(t, meta.Name)
	require.NotEmpty(t, meta.Version)
	require.Equal(t, "command", meta.Name)
}

func TestCommandPlugin_Schema(t *testing.T) {
	t.Parallel()

	p := New()
	schema := p.Schema()

	require.NotNil(t, schema)
	_, ok := schema.(config.CommandStep)
	require.True(t, ok, "schema should be of type CommandStep")
}

func TestCommandPlugin_EvaluateMissingConfig(t *testing.T) {
	_, err := New().Evaluate(context.Background(), &config.Step{ID: "missing", Type: "command"})
	require.Error(t, err)
}

func TestCommandPlugin_ApplyMissingConfig(t *testing.T) {
	evalResult := &model.EvaluationResult{
		StepID:         "missing",
		CurrentState:   model.StatusMissing,
		RequiresAction: true,
	}
	res, err := New().Apply(context.Background(), evalResult, &config.Step{ID: "missing", Type: "command"})
	require.Error(t, err)
	require.Nil(t, res)
}

func writeScript(t *testing.T, dir, name, content string) {
	t.Helper()
	scriptPath := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(scriptPath, []byte(content), 0755))
}

func TestCommandPlugin_EvaluateWithCheckCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell assumptions do not hold on Windows")
	}
	binDir := t.TempDir()
	writeScript(t, binDir, "check-script", `#!/bin/sh
if [ -f "/tmp/testfile" ]; then
  echo "file exists"
  exit 0
else
  echo "file missing"
  exit 1
fi
`)

	originalPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", originalPath) })
	require.NoError(t, os.Setenv("PATH", binDir+":"+originalPath))

	step := &config.Step{
		ID:   "check_file",
		Type: "command",
		Command: &config.CommandStep{
			Command: "echo hello",
			Check:   "check-script",
		},
	}

	p := New()

	// Test when check command fails (file doesn't exist)
	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusMissing, evalResult.CurrentState)
	require.True(t, evalResult.RequiresAction)
	require.Contains(t, evalResult.Message, "file missing")

	// Create the file
	require.NoError(t, os.WriteFile("/tmp/testfile", []byte("test"), 0644))
	t.Cleanup(func() { os.Remove("/tmp/testfile") })

	// Test when check command succeeds (file exists)
	evalResult, err = p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusSatisfied, evalResult.CurrentState)
	require.False(t, evalResult.RequiresAction)
	require.Contains(t, evalResult.Message, "check command succeeded")
}

func TestCommandPlugin_EvaluateWithoutCheckCommand(t *testing.T) {
	step := &config.Step{
		ID:   "no_check",
		Type: "command",
		Command: &config.CommandStep{
			Command: "echo hello",
		},
	}

	p := New()

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusUnknown, evalResult.CurrentState)
	require.True(t, evalResult.RequiresAction)
	require.Contains(t, evalResult.Message, "no verification command specified")
}

func TestCommandPlugin_EvaluateWithErrorInCheckCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell assumptions do not hold on Windows")
	}
	step := &config.Step{
		ID:   "error_check",
		Type: "command",
		Command: &config.CommandStep{
			Command: "echo hello",
			Check:   "nonexistent-command",
		},
	}

	p := New()

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusMissing, evalResult.CurrentState)
	require.True(t, evalResult.RequiresAction)
	require.Contains(t, evalResult.Message, "check command failed")
}
