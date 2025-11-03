package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

func TestRenderListTreeShowsDependenciesAndStatus(t *testing.T) {

	home := t.TempDir()
	t.Setenv("HOME", home)

	registryPath, err := defaultRegistryPath()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(registryPath), 0o755))

	reg, err := registry.NewRegistry(registryPath)
	require.NoError(t, err)

	require.NoError(t, reg.Add(registry.Pipeline{
		ID:           "frontend@1.0",
		Name:         "Frontend",
		Path:         filepath.Join(home, "frontend.yaml"),
		Dependencies: []string{"backend@1.0", "auth@1.0"},
	}))

	require.NoError(t, reg.Add(registry.Pipeline{
		ID:   "backend@1.0",
		Name: "Backend",
		Path: filepath.Join(home, "backend.yaml"),
	}))

	require.NoError(t, reg.Save())

	// Ensure status cache directory exists
	statusCachePath, err := defaultStatusCachePath()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(statusCachePath), 0o755))

	cmd := &cobra.Command{}

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	opts := &listOptions{tree: true}

	require.NoError(t, runList(context.Background(), nil, cmd, opts))

	expected := "└─ [BL] frontend@1.0 (Blocked: missing dependencies: auth@1.0)\n" +
		"   ├─ [BL] auth@1.0 (Blocked: not registered)\n" +
		"   └─ [RD] backend@1.0 (Ready)\n"

	require.Equal(t, expected, output.String())
}
