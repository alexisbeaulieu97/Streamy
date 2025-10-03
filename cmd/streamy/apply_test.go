package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
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

func executeCommand(cmd *cobra.Command, args ...string) error {
	cmd.SetArgs(args)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	return cmd.Execute()
}
