package commandplugin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	pluginpkg "github.com/alexisbeaulieu97/streamy/internal/plugin"
)

func TestCommandPlugin_CheckUsesCheckCommand(t *testing.T) {
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
	ok, err := p.Check(context.Background(), step)
	require.NoError(t, err)
	require.True(t, ok)

	require.NoError(t, os.Setenv("EXPECT_FAIL", "1"))
	ok, err = p.Check(context.Background(), step)
	require.NoError(t, err)
	require.False(t, ok)
}

func TestCommandPlugin_ApplyRunsCommandWithEnvAndWorkdir(t *testing.T) {
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

	res, err := p.Apply(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, "success", strings.ToLower(res.Status))

	data, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	require.Equal(t, "streamy\n", string(data))
}

func TestCommandPlugin_DryRunSkipsExecution(t *testing.T) {
	workDir := t.TempDir()
	target := filepath.Join(workDir, "should_not_exist")

	step := &config.Step{
		ID:   "run_command",
		Type: "command",
		Command: &config.CommandStep{
			Command: "touch should_not_exist",
			WorkDir: workDir,
		},
	}

	p := New()

	res, err := p.DryRun(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, "skipped", strings.ToLower(res.Status))

	_, err = os.Stat(target)
	require.Error(t, err)
}

func writeScript(t *testing.T, dir, name, contents string) {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o755))
}
