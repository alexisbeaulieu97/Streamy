package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

func TestListCommand_TableOutput(t *testing.T) {
	home := setupListHome(t)
	registryPath, statusPath := listPaths(home)

	seedListRegistry(t, registryPath, []registry.Pipeline{
		{ID: "dev-setup", Name: "Development Setup", Path: filepath.Join(home, "configs", "dev.yaml"), Description: "Dev environment", RegisteredAt: time.Now().Add(-4 * time.Hour)},
		{ID: "staging-env", Name: "Staging Environment", Path: filepath.Join(home, "configs", "staging.yaml"), Description: "Staging", RegisteredAt: time.Now().Add(-2 * time.Hour)},
	})
	seedListStatusCache(t, statusPath, map[string]registry.CachedStatus{
		"dev-setup": {
			Status:    registry.StatusSatisfied,
			LastRun:   time.Now().Add(-90 * time.Minute).UTC(),
			Summary:   "All checks passed",
			StepCount: 15,
		},
		"staging-env": {
			Status:      registry.StatusFailed,
			LastRun:     time.Now().Add(-30 * time.Minute).UTC(),
			Summary:     "2 steps failed",
			StepCount:   12,
			FailedSteps: []string{"verify-packages", "apply-config"},
		},
	})

	stdout, err := executeListCommand()
	require.NoError(t, err)
	require.Contains(t, stdout, "ID           NAME")
	require.Contains(t, stdout, "dev-setup")
	require.Contains(t, stdout, "Development Setup")
	// We capture output via buffer (non-TTY), expect ASCII fallback icons
	require.Contains(t, stdout, "[OK] satisfied")
	require.Contains(t, stdout, "[XX] failed")
	require.Contains(t, stdout, filepath.Join(home, "configs", "dev.yaml"))
}

func TestListCommand_TableOutputAsciiFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ASCII fallback differs on Windows buffer capture")
	}

	home := setupListHome(t)
	registryPath, statusPath := listPaths(home)

	seedListRegistry(t, registryPath, []registry.Pipeline{{
		ID:           "dev-setup",
		Name:         "Development Setup",
		Path:         filepath.Join(home, "configs", "dev.yaml"),
		RegisteredAt: time.Now(),
	}})
	seedListStatusCache(t, statusPath, map[string]registry.CachedStatus{
		"dev-setup": {Status: registry.StatusSatisfied},
	})

	stdout, err := executeListCommand()
	require.NoError(t, err)
	require.Contains(t, stdout, "[OK] satisfied")
}

func TestListCommand_JSONOutput(t *testing.T) {
	home := setupListHome(t)
	registryPath, statusPath := listPaths(home)

	seedListRegistry(t, registryPath, []registry.Pipeline{
		{ID: "dev-setup", Name: "Development Setup", Path: filepath.Join(home, "configs", "dev.yaml"), Description: "Dev environment", RegisteredAt: time.Now().Add(-4 * time.Hour)},
	})
	seedListStatusCache(t, statusPath, map[string]registry.CachedStatus{
		"dev-setup": {
			Status:    registry.StatusSatisfied,
			LastRun:   time.Now().Add(-90 * time.Minute).UTC(),
			Summary:   "All checks passed",
			StepCount: 15,
		},
	})

	stdout, err := executeListCommand("--json")
	require.NoError(t, err)

	var payload struct {
		Version   string `json:"version"`
		Count     int    `json:"count"`
		Pipelines []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Status string `json:"status"`
			Path   string `json:"path"`
		} `json:"pipelines"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &payload))
	require.Equal(t, 1, payload.Count)
	require.Equal(t, "1.0", payload.Version)
	require.Len(t, payload.Pipelines, 1)
	require.Equal(t, "dev-setup", payload.Pipelines[0].ID)
	require.Equal(t, "satisfied", payload.Pipelines[0].Status)
	require.Equal(t, filepath.Join(home, "configs", "dev.yaml"), payload.Pipelines[0].Path)
}

func TestListCommand_EmptyRegistry(t *testing.T) {
	home := setupListHome(t)
	registryPath, _ := listPaths(home)
	seedListRegistry(t, registryPath, []registry.Pipeline{})

	stdout, err := executeListCommand()
	require.NoError(t, err)
	require.Contains(t, stdout, "No pipelines registered yet.")
	require.Contains(t, stdout, "Run 'streamy registry add <config-path>'")
}

func TestListCommand_StatusIconsASCII(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ASCII fallback differs on Windows buffer capture")
	}

	home := setupListHome(t)
	registryPath, statusPath := listPaths(home)

	seedListRegistry(t, registryPath, []registry.Pipeline{{
		ID:           "dev-setup",
		Name:         "Development Setup",
		Path:         filepath.Join(home, "configs", "dev.yaml"),
		RegisteredAt: time.Now(),
	}})
	seedListStatusCache(t, statusPath, map[string]registry.CachedStatus{
		"dev-setup": {Status: registry.StatusSatisfied},
	})

	stdout, err := executeListCommand()
	require.NoError(t, err)
	require.Contains(t, stdout, "[OK] satisfied")
}

func executeListCommand(extraArgs ...string) (string, error) {
	root := newRootCmd()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)

	args := append([]string{"registry", "list"}, extraArgs...)
	root.SetArgs(args)

	err := root.Execute()
	return buf.String(), err
}

func setupListHome(t *testing.T) string {
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func listPaths(home string) (registryPath, statusPath string) {
	registryPath = filepath.Join(home, ".streamy", "registry.json")
	statusPath = filepath.Join(home, ".streamy", "status.json")
	return
}

func seedListRegistry(t *testing.T, path string, pipelines []registry.Pipeline) {
	t.Helper()
	file := registry.RegistryFile{Version: "1.0", Pipelines: pipelines}
	data, err := json.MarshalIndent(file, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, data, 0o644))
}

func seedListStatusCache(t *testing.T, path string, statuses map[string]registry.CachedStatus) {
	t.Helper()
	file := registry.StatusCacheFile{Version: "1.0", Statuses: statuses}
	data, err := json.MarshalIndent(file, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, data, 0o644))
}
