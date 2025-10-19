package commandplugin

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

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

func writeScript(t *testing.T, dir, name, content string) {
	t.Helper()
	scriptPath := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(scriptPath, []byte(content), 0o755))
}
