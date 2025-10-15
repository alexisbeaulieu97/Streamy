package commandplugin

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	pluginpkg "github.com/alexisbeaulieu97/streamy/internal/plugin"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
	"gopkg.in/yaml.v3"
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

	step := &config.Step{ID: "run_command", Type: "command"}
	require.NoError(t, step.SetConfig(config.CommandStep{
		Command: "echo hello",
		Check:   "check-script",
	}))

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

	step := &config.Step{ID: "run_command", Type: "command"}
	require.NoError(t, step.SetConfig(config.CommandStep{
		Command: "echo $CUSTOM_VALUE > result.txt",
		WorkDir: workDir,
		Env: map[string]string{
			"CUSTOM_VALUE": "streamy",
		},
	}))

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

	step := &config.Step{ID: "run_command", Type: "command"}
	require.NoError(t, step.SetConfig(config.CommandStep{
		Command: "touch should_not_exist",
		WorkDir: workDir,
	}))

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

	step := &config.Step{ID: "check_file", Type: "command"}
	require.NoError(t, step.SetConfig(config.CommandStep{
		Command: "echo hello",
		Check:   "check-script",
	}))

	p := New()

	// Test when check command fails (file doesn't exist)
	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusMissing, evalResult.CurrentState)
	require.True(t, evalResult.RequiresAction)
	require.Contains(t, evalResult.Message, "file missing")

	// Create the file
	require.NoError(t, os.WriteFile("/tmp/testfile", []byte("test"), 0644))
	t.Cleanup(func() { _ = os.Remove("/tmp/testfile") })

	// Test when check command succeeds (file exists)
	evalResult, err = p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusSatisfied, evalResult.CurrentState)
	require.False(t, evalResult.RequiresAction)
	require.Contains(t, evalResult.Message, "check command succeeded")
}

func TestCommandPlugin_EvaluateWithoutCheckCommand(t *testing.T) {
	step := &config.Step{ID: "no_check", Type: "command"}
	require.NoError(t, step.SetConfig(config.CommandStep{Command: "echo hello"}))

	p := New()

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusUnknown, evalResult.CurrentState)
	require.True(t, evalResult.RequiresAction)
	require.Contains(t, evalResult.Message, "no verification command specified")
}

func TestCommandPlugin_EvaluateUsesRawConfigWhenStructNil(t *testing.T) {
	yamlStr := `
id: raw_command
type: command
command: echo raw
`
	var step config.Step
	require.NoError(t, yaml.Unmarshal([]byte(yamlStr), &step))

	p := New()

	evalResult, err := p.Evaluate(context.Background(), &step)
	require.NoError(t, err)
	require.True(t, evalResult.RequiresAction)
}

func TestCommandPlugin_EvaluateWithErrorInCheckCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell assumptions do not hold on Windows")
	}
	step := &config.Step{ID: "error_check", Type: "command"}
	require.NoError(t, step.SetConfig(config.CommandStep{
		Command: "echo hello",
		Check:   "nonexistent-command",
	}))

	p := New()

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusMissing, evalResult.CurrentState)
	require.True(t, evalResult.RequiresAction)
	require.Contains(t, evalResult.Message, "check command failed")
}

func TestCommandPlugin_ApplySkipsWhenNoAction(t *testing.T) {
	step := &config.Step{ID: "skip", Type: "command"}
	require.NoError(t, step.SetConfig(config.CommandStep{Command: "echo noop"}))

	eval := &model.EvaluationResult{
		StepID:         step.ID,
		RequiresAction: false,
		CurrentState:   model.StatusSatisfied,
		Message:        "already satisfied",
		InternalData: &commandEvaluationData{
			ShellDetermined: true,
		},
	}

	result, err := New().Apply(context.Background(), eval, step)
	require.NoError(t, err)
	require.Equal(t, model.StatusSkipped, result.Status)
	require.Equal(t, step.ID, result.StepID)
}

func TestConvertError(t *testing.T) {
	t.Run("wraps validation errors", func(t *testing.T) {
		err := streamyerrors.NewValidationError("field", "invalid", nil)
		converted := convertError("cmd", err)

		var pluginErr *pluginpkg.ValidationError
		require.ErrorAs(t, converted, &pluginErr)
		require.Equal(t, "cmd", pluginErr.StepID())
	})

	t.Run("wraps execution errors", func(t *testing.T) {
		err := streamyerrors.NewExecutionError("legacy", errors.New("boom"))
		converted := convertError("cmd2", err)

		var pluginErr *pluginpkg.ExecutionError
		require.ErrorAs(t, converted, &pluginErr)
		require.Equal(t, "cmd2", pluginErr.StepID())
	})

	t.Run("wraps unknown errors as execution", func(t *testing.T) {
		err := errors.New("other failure")
		converted := convertError("cmd3", err)

		var pluginErr *pluginpkg.ExecutionError
		require.ErrorAs(t, converted, &pluginErr)
		require.Equal(t, "cmd3", pluginErr.StepID())
	})
}

func TestDetermineShellExplicit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell selection differs on Windows")
	}
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("/bin/sh not available")
	}

	shell, args, err := determineShell("/bin/sh")
	require.NoError(t, err)
	require.Equal(t, "/bin/sh", shell)
	require.Equal(t, []string{"-c"}, args)
}

func TestDetermineShellNoFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell lookup differs on Windows")
	}
	originalPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", originalPath) })
	require.NoError(t, os.Setenv("PATH", ""))

	_, _, err := determineShell("")
	require.Error(t, err)
}
