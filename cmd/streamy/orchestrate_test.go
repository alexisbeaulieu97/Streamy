package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	require "github.com/stretchr/testify/require"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

func TestRunOrchestrateDryRunPrintsPlan(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	registryPath, err := defaultRegistryPath()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(registryPath), 0o755))

	reg, err := registry.NewRegistry(registryPath)
	require.NoError(t, err)

	require.NoError(t, reg.Add(registry.Pipeline{ID: "base@1.0", Path: "base.yaml"}))
	require.NoError(t, reg.Add(registry.Pipeline{ID: "app@1.0", Path: "app.yaml", Dependencies: []string{"base@1.0"}}))
	require.NoError(t, reg.Save())

	out := &bytes.Buffer{}
	app := &AppContext{}

	opts := orchestrateOptions{PipelineID: "app@1.0", DryRun: true}

	err = runOrchestrate(context.Background(), app, opts, nil, out)
	require.NoError(t, err)

	output := out.String()
	require.Contains(t, output, "Orchestration Plan for app@1.0")
	require.Contains(t, output, "Level 0: base@1.0")
}

func TestRunOrchestrateRequiresPipelineID(t *testing.T) {
	err := validateOrchestrateOptions(orchestrateOptions{})
	require.Error(t, err)
}

func TestValidateOrchestrateOptionsForceRequiresYes(t *testing.T) {
	err := validateOrchestrateOptions(orchestrateOptions{
		PipelineID:     "app@1.0",
		Force:          true,
		NonInteractive: true,
	})
	require.Error(t, err)
}

func TestCollectForceTargetsReturnsBlockedPipelines(t *testing.T) {
	cache := newTestStatusCache(t)
	require.NoError(t, cache.Set("blocked@1.0", registry.CachedStatus{
		Status:    registry.StatusBlocked,
		BlockedBy: "db@1.0",
		Summary:   "Dependency failed",
	}))

	plan := &domainpipeline.OrchestrationPlan{
		Levels: [][]string{{"blocked@1.0", "ready@1.0"}},
	}

	targets := collectForceTargets(plan, cache)
	require.Len(t, targets, 1)
	require.Equal(t, "blocked@1.0", targets[0].ID)
	require.Equal(t, "db@1.0", targets[0].BlockedBy)
}

func newTestStatusCache(t *testing.T) *registry.StatusCache {
	t.Helper()

	path := filepath.Join(t.TempDir(), "status-cache.json")
	cache, err := registry.NewStatusCache(path)
	require.NoError(t, err)

	return cache
}
