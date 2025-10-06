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
	pluginpkg "github.com/alexisbeaulieu97/streamy/internal/plugin"
)

func TestCommandPlugin_CheckUsesCheckCommand(t *testing.T) {
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
	ok, err := p.Check(context.Background(), step)
	require.NoError(t, err)
	require.True(t, ok)

	require.NoError(t, os.Setenv("EXPECT_FAIL", "1"))
	ok, err = p.Check(context.Background(), step)
	require.NoError(t, err)
	require.False(t, ok)
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

	res, err := p.Apply(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, "success", strings.ToLower(res.Status))

	data, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	require.Equal(t, "streamy\n", string(data))
}

func TestCommandPlugin_DryRunSkipsExecution(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell assumptions do not hold on Windows")
	}
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

func TestCommandPlugin_Metadata(t *testing.T) {
	t.Parallel()

	p := New()
	meta := p.Metadata()

	require.NotEmpty(t, meta.Name)
	require.NotEmpty(t, meta.Version)
	require.Equal(t, "command", meta.Type)
}

func TestCommandPlugin_Schema(t *testing.T) {
	t.Parallel()

	p := New()
	schema := p.Schema()

	require.NotNil(t, schema)
	_, ok := schema.(config.CommandStep)
	require.True(t, ok, "schema should be of type CommandStep")
}

func TestCommandPlugin_CheckMissingConfig(t *testing.T) {
	_, err := New().Check(context.Background(), &config.Step{ID: "missing", Type: "command"})
	require.Error(t, err)
}

func TestCommandPlugin_ApplyMissingConfig(t *testing.T) {
	res, err := New().Apply(context.Background(), &config.Step{ID: "missing", Type: "command"})
	require.Error(t, err)
	require.Nil(t, res)
}

func TestDetermineShell(t *testing.T) {
	t.Parallel()

	t.Run("uses explicit shell when provided", func(t *testing.T) {
		t.Parallel()
		shell, args, err := determineShell("/usr/bin/zsh")
		require.NoError(t, err)
		require.Equal(t, "/usr/bin/zsh", shell)
		require.Equal(t, []string{"-c"}, args)
	})

	t.Run("uses default shell when not provided", func(t *testing.T) {
		t.Parallel()
		shell, args, err := determineShell("")
		require.NoError(t, err)
		require.NotEmpty(t, shell)
		require.NotEmpty(t, args)
	})
}

func TestCommandPlugin_CheckWithNoCheckCommand(t *testing.T) {
	t.Parallel()

	step := &config.Step{
		ID:   "run_command",
		Type: "command",
		Command: &config.CommandStep{
			Command: "echo hello",
			Check:   "",
		},
	}

	p := New()

	ok, err := p.Check(context.Background(), step)
	require.NoError(t, err)
	require.False(t, ok, "expected Check to return false when check command is empty")
}

func TestCommandPlugin_ApplyWithWorkDir(t *testing.T) {
	workDir := t.TempDir()
	outputFile := "output.txt"

	step := &config.Step{
		ID:   "run_command",
		Type: "command",
		Command: &config.CommandStep{
			Command: "pwd > output.txt",
			WorkDir: workDir,
		},
	}

	p := New()

	res, err := p.Apply(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, "success", res.Status)

	data, err := os.ReadFile(filepath.Join(workDir, outputFile))
	require.NoError(t, err)
	require.Contains(t, string(data), workDir)
}

func writeScript(t *testing.T, dir, name, contents string) {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o755))
}

func TestCommandPlugin_Verify(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell assumptions do not hold on Windows")
	}

	t.Run("returns satisfied when check command succeeds", func(t *testing.T) {
		binDir := t.TempDir()
		writeScript(t, binDir, "check-script", `#!/bin/sh
exit 0
`)

		originalPath := os.Getenv("PATH")
		t.Cleanup(func() { _ = os.Setenv("PATH", originalPath) })
		require.NoError(t, os.Setenv("PATH", binDir+":"+originalPath))

		p := New()

		step := &config.Step{
			ID:   "run_command",
			Type: "command",
			Command: &config.CommandStep{
				Command: "echo hello",
				Check:   "check-script",
			},
		}

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "satisfied", string(result.Status))
		require.Contains(t, result.Message, "succeeded")
	})

	t.Run("returns missing when check command fails", func(t *testing.T) {
		binDir := t.TempDir()
		writeScript(t, binDir, "check-script", `#!/bin/sh
exit 1
`)

		originalPath := os.Getenv("PATH")
		t.Cleanup(func() { _ = os.Setenv("PATH", originalPath) })
		require.NoError(t, os.Setenv("PATH", binDir+":"+originalPath))

		p := New()

		step := &config.Step{
			ID:   "run_command",
			Type: "command",
			Command: &config.CommandStep{
				Command: "echo hello",
				Check:   "check-script",
			},
		}

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "missing", string(result.Status))
		require.Contains(t, result.Message, "failed")
	})

	t.Run("returns unknown when no check command specified", func(t *testing.T) {
		p := New()

		step := &config.Step{
			ID:   "run_command",
			Type: "command",
			Command: &config.CommandStep{
				Command: "echo hello",
				Check:   "",
			},
		}

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "unknown", string(result.Status))
		require.Contains(t, result.Message, "no verification command")
	})

	t.Run("returns blocked when context is cancelled", func(t *testing.T) {
		p := New()

		step := &config.Step{
			ID:   "run_command",
			Type: "command",
			Command: &config.CommandStep{
				Command: "echo hello",
				Check:   "test -f /tmp/file",
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		result, err := p.Verify(ctx, step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "blocked", string(result.Status))
		require.Contains(t, result.Message, "cancelled")
		require.NotNil(t, result.Error)
	})

	t.Run("returns error when command config is nil", func(t *testing.T) {
		p := New()

		step := &config.Step{
			ID:      "run_command",
			Type:    "command",
			Command: nil,
		}

		_, err := p.Verify(context.Background(), step)
		require.Error(t, err)
		require.Contains(t, err.Error(), "command configuration missing")
	})

	t.Run("uses custom env and workdir for check", func(t *testing.T) {
		workDir := t.TempDir()
		testFile := filepath.Join(workDir, "test.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("content"), 0o644))

		p := New()

		step := &config.Step{
			ID:   "run_command",
			Type: "command",
			Command: &config.CommandStep{
				Command: "echo hello",
				Check:   "test -f test.txt",
				WorkDir: workDir,
			},
		}

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, "satisfied", string(result.Status))
	})
}

func TestCommandPlugin_Apply_ErrorHandling(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell assumptions do not hold on Windows")
	}

	t.Run("returns error when command fails", func(t *testing.T) {
		p := New()

		step := &config.Step{
			ID:   "run_command",
			Type: "command",
			Command: &config.CommandStep{
				Command: "exit 1",
			},
		}

		result, err := p.Apply(context.Background(), step)
		require.Error(t, err)
		require.NotNil(t, result)
		require.Equal(t, "failed", result.Status)
	})
}
