package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

func TestShowCommand_DetailedOutput(t *testing.T) {
	home := setupShowHome(t)
	registryPath := filepath.Join(home, ".streamy", "registry.json")
	statusPath := filepath.Join(home, ".streamy", "status.json")

	seedShowRegistry(t, registryPath, []registry.Pipeline{{
		ID:           "dev-setup",
		Name:         "Development Setup",
		Path:         filepath.Join(home, "configs", "dev.yaml"),
		Description:  "Dev environment",
		RegisteredAt: time.Now().Add(-2 * time.Hour),
	}})

	seedShowStatusCache(t, statusPath, map[string]registry.CachedStatus{
		"dev-setup": {
			Status:    registry.StatusSatisfied,
			LastRun:   time.Now().Add(-time.Minute * 45).UTC(),
			Summary:   "All checks passed",
			StepCount: 15,
		},
	})

	stdout, err := executeShowCommand("dev-setup")
	require.NoError(t, err)
	require.Contains(t, stdout, "Pipeline: dev-setup")
	require.Contains(t, stdout, "Status:")
	require.Contains(t, stdout, "All checks passed")
}

func TestShowCommand_NeverVerified(t *testing.T) {
	home := setupShowHome(t)
	registryPath := filepath.Join(home, ".streamy", "registry.json")

	seedShowRegistry(t, registryPath, []registry.Pipeline{{
		ID:           "new",
		Name:         "New Pipeline",
		Path:         filepath.Join(home, "configs", "new.yaml"),
		RegisteredAt: time.Now(),
	}})

	stdout, err := executeShowCommand("new")
	require.NoError(t, err)
	require.Contains(t, stdout, "Last Run: never")
}

func TestShowCommand_NotFound(t *testing.T) {
	setupShowHome(t)

	_, err := executeShowCommand("missing")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestShowCommand_JSONOutput(t *testing.T) {
	home := setupShowHome(t)
	registryPath := filepath.Join(home, ".streamy", "registry.json")
	statusPath := filepath.Join(home, ".streamy", "status.json")

	seedShowRegistry(t, registryPath, []registry.Pipeline{{
		ID:           "dev-setup",
		Name:         "Development Setup",
		Path:         filepath.Join(home, "configs", "dev.yaml"),
		Description:  "Dev environment",
		RegisteredAt: time.Now().Add(-2 * time.Hour),
	}})
	seedShowStatusCache(t, statusPath, map[string]registry.CachedStatus{
		"dev-setup": {
			Status:      registry.StatusDrifted,
			LastRun:     time.Now().Add(-time.Minute * 30).UTC(),
			Summary:     "2 steps failed",
			StepCount:   12,
			FailedSteps: []string{"install"},
		},
	})

	stdout, err := executeShowCommand("dev-setup", "--json")
	require.NoError(t, err)

	var payload struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Status      string   `json:"status"`
		Summary     string   `json:"summary"`
		StepCount   int      `json:"step_count"`
		FailedSteps []string `json:"failed_steps"`
	}

	require.NoError(t, json.Unmarshal([]byte(stdout), &payload))
	require.Equal(t, "dev-setup", payload.ID)
	require.Equal(t, "2 steps failed", payload.Summary)
	require.Equal(t, "drifted", payload.Status)
	require.Equal(t, []string{"install"}, payload.FailedSteps)
}

// --- helpers ---

func executeShowCommand(args ...string) (string, error) {
	root := newRootCmd()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)

	root.SetArgs(append([]string{"registry", "show"}, args...))

	err := root.Execute()
	return buf.String(), err
}

func setupShowHome(t *testing.T) string {
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func seedShowRegistry(t *testing.T, path string, pipelines []registry.Pipeline) {
	t.Helper()
	payload := registry.RegistryFile{Version: "1.0", Pipelines: pipelines}
	data, err := json.MarshalIndent(payload, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, data, 0o644))
}

func seedShowStatusCache(t *testing.T, path string, statuses map[string]registry.CachedStatus) {
	t.Helper()
	payload := registry.StatusCacheFile{Version: "1.0", Statuses: statuses}
	data, err := json.MarshalIndent(payload, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, data, 0o644))
}
