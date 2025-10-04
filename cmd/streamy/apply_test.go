package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/engine"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/tui"
)

func TestApplyCommandParsesFlags(t *testing.T) {
	t.Cleanup(func() { applyCmdRunner = runApply })

	var captured applyOptions
	applyCmdRunner = func(opts applyOptions) error {
		captured = opts
		return nil
	}

	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "config.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte("version: \"1.0\"\nname: test\nsteps: []\n"), 0o644))

	root := newRootCmd()
	require.NoError(t, executeCommand(root, "apply", "--config", cfgPath, "--dry-run", "--verbose"))

	require.Equal(t, cfgPath, captured.ConfigPath)
	require.True(t, captured.DryRun)
	require.True(t, captured.Verbose)
}

func TestApplyCommandValidatesConfigFile(t *testing.T) {
	t.Cleanup(func() { applyCmdRunner = runApply })
	applyCmdRunner = func(opts applyOptions) error { return nil }

	root := newRootCmd()
	err := executeCommand(root, "apply", "--config", "/path/does/not/exist")
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not exist")
}

func TestValidateApplyOptions(t *testing.T) {
	t.Parallel()

	t.Run("returns error when config path is empty", func(t *testing.T) {
		t.Parallel()
		opts := applyOptions{ConfigPath: ""}
		err := validateApplyOptions(opts)
		require.Error(t, err)
		require.Contains(t, err.Error(), "required")
	})

	t.Run("returns error when config path is whitespace", func(t *testing.T) {
		t.Parallel()
		opts := applyOptions{ConfigPath: "   "}
		err := validateApplyOptions(opts)
		require.Error(t, err)
		require.Contains(t, err.Error(), "required")
	})

	t.Run("returns error when config file does not exist", func(t *testing.T) {
		t.Parallel()
		opts := applyOptions{ConfigPath: "/nonexistent/path/config.yaml"}
		err := validateApplyOptions(opts)
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not exist")
	})

	t.Run("returns error when config path is a directory", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		opts := applyOptions{ConfigPath: dir}
		err := validateApplyOptions(opts)
		require.Error(t, err)
		require.Contains(t, err.Error(), "directory")
	})

	t.Run("succeeds for valid config file", func(t *testing.T) {
		t.Parallel()
		tmpFile := filepath.Join(t.TempDir(), "config.yaml")
		require.NoError(t, os.WriteFile(tmpFile, []byte("test"), 0o644))

		opts := applyOptions{ConfigPath: tmpFile}
		err := validateApplyOptions(opts)
		require.NoError(t, err)
	})
}

func executeCommand(cmd *cobra.Command, args ...string) error {
	cmd.SetArgs(args)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	return cmd.Execute()
}

func TestRunApply(t *testing.T) {
	t.Run("handles invalid config file", func(t *testing.T) {
		cfgDir := t.TempDir()
		cfgPath := filepath.Join(cfgDir, "invalid.yaml")
		require.NoError(t, os.WriteFile(cfgPath, []byte("invalid: yaml: content: ["), 0o644))

		opts := applyOptions{
			ConfigPath: cfgPath,
		}

		err := runApply(opts)
		require.Error(t, err)
		require.Contains(t, err.Error(), "parse error")
	})
}

func TestDispatchTuiMessage(t *testing.T) {
	t.Run("non-interactive mode calls update without panic", func(t *testing.T) {
		result := &tui.StepCompleteMsg{
			Result: model.StepResult{
				StepID:  "test-step",
				Status:  "success",
				Message: "Test completed",
			},
		}

		modelState := tui.NewModel(&config.Config{}, &engine.ExecutionPlan{}, true)

		// This should not panic and should call Update
		dispatchTuiMessage(false, nil, &modelState, result)

		// The function should have called Update without error
		require.NotNil(t, modelState)
	})

	t.Run("interactive mode with nil program does nothing", func(t *testing.T) {
		result := &tui.StepCompleteMsg{
			Result: model.StepResult{
				StepID:  "test-step",
				Status:  "success",
				Message: "Test completed",
			},
		}

		modelState := tui.NewModel(&config.Config{}, &engine.ExecutionPlan{}, false)

		// Should not panic when program is nil
		dispatchTuiMessage(true, nil, &modelState, result)

		// State should still be valid
		require.NotNil(t, modelState)
	})
}
