package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

func TestRefreshCommand_SinglePipeline(t *testing.T) {
	home := setupRefreshHome(t)
	registryPath := filepath.Join(home, ".streamy", "registry.json")
	statusPath := filepath.Join(home, ".streamy", "status.json")

	configPath := writeRefreshConfig(t, home, "dev.yaml")

	seedRefreshRegistry(t, registryPath, []registry.Pipeline{{
		ID:           "dev-setup",
		Name:         "Development Setup",
		Path:         configPath,
		RegisteredAt: time.Now().Add(-time.Hour),
	}})

	cleanup := withRefreshVerifyFunc(func(p registry.Pipeline, cfg *config.Config) verifyOutcome {
		require.Equal(t, "dev-setup", p.ID)
		return verifyOutcome{Status: registry.StatusSatisfied, Summary: "All checks passed", StepCount: len(cfg.Steps)}
	})
	defer cleanup()

	stdout, err := executeRefreshCommand("dev-setup")
	require.NoError(t, err)
	require.Contains(t, stdout, "dev-setup")
	require.Contains(t, stdout, "Satisfied")

	cache := mustLoadStatusCache(t, statusPath)
	status, ok := cache.Get("dev-setup")
	require.True(t, ok)
	require.Equal(t, registry.StatusSatisfied, status.Status)
	require.Equal(t, 1, status.StepCount)
}

func TestRefreshCommand_BulkConcurrency(t *testing.T) {
	home := setupRefreshHome(t)
	registryPath := filepath.Join(home, ".streamy", "registry.json")
	statusPath := filepath.Join(home, ".streamy", "status.json")

	pipelines := []registry.Pipeline{
		{ID: "alpha", Path: writeRefreshConfig(t, home, "alpha.yaml"), RegisteredAt: time.Now()},
		{ID: "beta", Path: writeRefreshConfig(t, home, "beta.yaml"), RegisteredAt: time.Now()},
		{ID: "gamma", Path: writeRefreshConfig(t, home, "gamma.yaml"), RegisteredAt: time.Now()},
	}
	seedRefreshRegistry(t, registryPath, pipelines)

	var mu sync.Mutex
	var order []string

	cleanup := withRefreshVerifyFunc(func(p registry.Pipeline, cfg *config.Config) verifyOutcome {
		mu.Lock()
		order = append(order, p.ID)
		mu.Unlock()
		return verifyOutcome{Status: registry.StatusSatisfied, Summary: "OK", StepCount: len(cfg.Steps)}
	})
	defer cleanup()

	_, err := executeRefreshCommand("--concurrency", "2")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"alpha", "beta", "gamma"}, order)

	cache := mustLoadStatusCache(t, statusPath)
	for _, p := range pipelines {
		status, ok := cache.Get(p.ID)
		require.True(t, ok)
		require.Equal(t, registry.StatusSatisfied, status.Status)
	}
}

func TestRefreshCommand_MissingConfigHandled(t *testing.T) {
	home := setupRefreshHome(t)
	registryPath := filepath.Join(home, ".streamy", "registry.json")
	statusPath := filepath.Join(home, ".streamy", "status.json")

	missingPath := filepath.Join(home, "configs", "missing.yaml")
	seedRefreshRegistry(t, registryPath, []registry.Pipeline{{
		ID:           "orphan",
		Name:         "Orphaned Pipeline",
		Path:         missingPath,
		RegisteredAt: time.Now(),
	}})

	stdout, err := executeRefreshCommand()
	require.NoError(t, err)
	require.Contains(t, stdout, "orphan")
	require.Contains(t, stdout, "failed")

	cache := mustLoadStatusCache(t, statusPath)
	status, ok := cache.Get("orphan")
	require.True(t, ok)
	require.Equal(t, registry.StatusFailed, status.Status)
	require.Equal(t, "Configuration load failed", status.Summary)
}

func TestRefreshCommand_ProgressOutput(t *testing.T) {
	home := setupRefreshHome(t)
	registryPath := filepath.Join(home, ".streamy", "registry.json")

	pipelines := []registry.Pipeline{
		{ID: "one", Path: writeRefreshConfig(t, home, "one.yaml"), RegisteredAt: time.Now()},
		{ID: "two", Path: writeRefreshConfig(t, home, "two.yaml"), RegisteredAt: time.Now()},
	}
	seedRefreshRegistry(t, registryPath, pipelines)

	cleanup := withRefreshVerifyFunc(func(p registry.Pipeline, cfg *config.Config) verifyOutcome {
		return verifyOutcome{Status: registry.StatusSatisfied, Summary: "OK", StepCount: len(cfg.Steps)}
	})
	defer cleanup()

	stdout, err := executeRefreshCommand()
	require.NoError(t, err)
	require.Contains(t, stdout, "[1/2] one")
	require.Contains(t, stdout, "[2/2] two")
}

// --- Test helpers ---

func executeRefreshCommand(args ...string) (string, error) {
	root := newRootCmd()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)

	root.SetArgs(append([]string{"registry", "refresh"}, args...))

	err := root.Execute()
	return buf.String(), err
}

func setupRefreshHome(t *testing.T) string {
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func seedRefreshRegistry(t *testing.T, path string, pipelines []registry.Pipeline) {
	t.Helper()
	file := registry.RegistryFile{Version: "1.0", Pipelines: pipelines}
	data, err := json.MarshalIndent(file, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, data, 0o644))
}

func mustLoadStatusCache(t *testing.T, path string) *registry.StatusCache {
	t.Helper()
	cache, err := registry.NewStatusCache(path)
	require.NoError(t, err)
	require.NoError(t, cache.Load())
	return cache
}

func writeRefreshConfig(t *testing.T, home, name string) string {
	t.Helper()
	configDir := filepath.Join(home, "configs")
	require.NoError(t, os.MkdirAll(configDir, 0o755))

	content := `version: "1.0"
name: test
settings:
  parallel: 1
steps:
  - id: step-1
    type: command
    command: echo test
`

	path := filepath.Join(configDir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}
