//go:build integration
// +build integration

package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	assert "github.com/stretchr/testify/assert"
	require "github.com/stretchr/testify/require"

	orchestrateapp "github.com/alexisbeaulieu97/streamy/internal/application/orchestration"
	apppipeline "github.com/alexisbeaulieu97/streamy/internal/application/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

func TestOrchestrateEndToEndSuccess(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	harness := newAppHarness(t)

	configDir := t.TempDir()
	baseConfig := writePipelineConfig(t, configDir, "base", nil, "echo base")
	appConfig := writePipelineConfig(t, configDir, "frontend", []string{"base@1.0"}, "echo frontend")

	reg := createRegistry(t,
		registry.Pipeline{ID: "base@1.0", Name: "base", Path: baseConfig, RegisteredAt: time.Now()},
		registry.Pipeline{ID: "frontend@1.0", Name: "frontend", Path: appConfig, RegisteredAt: time.Now(), Dependencies: []string{"base@1.0"}},
	)

	cache := createStatusCache(t)

	uc, err := orchestrateapp.NewUseCase(reg, cache, harness.ApplyUseCase)
	require.NoError(t, err)

	plan, err := uc.Plan(ctx, "frontend@1.0")
	require.NoError(t, err)
	require.NotNil(t, plan)
	require.Len(t, plan.Levels, 2)

	result, err := uc.Execute(ctx, "frontend@1.0", false, false)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.OverallSuccess)
	assert.Equal(t, 2, result.ExecutedCount)
	assert.Equal(t, 0, result.BlockedCount)
	assert.Equal(t, []string{"frontend@1.0"}, result.Plan.Levels[len(result.Plan.Levels)-1])

	orchestrations := cache.Orchestrations()
	require.Len(t, orchestrations, 1)
	summary := orchestrations[0]
	assert.Equal(t, "frontend@1.0", summary.RootPipelineID)
	assert.Equal(t, 2, summary.Executed)
	assert.Equal(t, 0, summary.Blocked)
	assert.Equal(t, []string{"base@1.0", "frontend@1.0"}, summary.PipelineOrder)
}

func TestOrchestrateFailureBlocksDependents(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	harness := newAppHarness(t)

	configDir := t.TempDir()
	foundation := writePipelineConfig(t, configDir, "foundation", nil, "echo foundation")
	fragile := writePipelineConfig(t, configDir, "fragile", []string{"foundation@1.0"}, "sh -c 'exit 1'")
	dependent := writePipelineConfig(t, configDir, "consumer", []string{"fragile@1.0"}, "echo consumer")

	reg := createRegistry(t,
		registry.Pipeline{ID: "foundation@1.0", Name: "foundation", Path: foundation, RegisteredAt: time.Now()},
		registry.Pipeline{ID: "fragile@1.0", Name: "fragile", Path: fragile, RegisteredAt: time.Now(), Dependencies: []string{"foundation@1.0"}},
		registry.Pipeline{ID: "consumer@1.0", Name: "consumer", Path: dependent, RegisteredAt: time.Now(), Dependencies: []string{"fragile@1.0"}},
	)

	cache := createStatusCache(t)

	uc, err := orchestrateapp.NewUseCase(reg, cache, harness.ApplyUseCase)
	require.NoError(t, err)

	result, err := uc.Execute(ctx, "consumer@1.0", false, false)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.False(t, result.OverallSuccess)
	assert.Equal(t, 2, result.ExecutedCount)
	assert.Equal(t, 1, result.BlockedCount)
	assert.Equal(t, 1, result.FailedCount)

	fragileSummary := result.PipelineResults["fragile@1.0"]
	require.NotNil(t, fragileSummary)
	assert.False(t, fragileSummary.Success)
	assert.Equal(t, registry.StatusFailed.String(), fragileSummary.Status)

	consumerSummary := result.PipelineResults["consumer@1.0"]
	require.NotNil(t, consumerSummary)
	assert.False(t, consumerSummary.Success)
	assert.Equal(t, "fragile@1.0", consumerSummary.BlockedBy)
	assert.Contains(t, consumerSummary.Summary, "fragile@1.0")
	assert.Equal(t, registry.StatusBlocked.String(), consumerSummary.Status)

	baseStatus, ok := cache.Get("foundation@1.0")
	require.True(t, ok)
	assert.Equal(t, registry.StatusReady, baseStatus.Status)

	fragileStatus, ok := cache.Get("fragile@1.0")
	require.True(t, ok)
	assert.Equal(t, registry.StatusFailed, fragileStatus.Status)
	assert.Equal(t, "fragile@1.0", fragileStatus.BlockedBy)

	consumerStatus, ok := cache.Get("consumer@1.0")
	require.True(t, ok)
	assert.Equal(t, registry.StatusBlocked, consumerStatus.Status)
	assert.Equal(t, "fragile@1.0", consumerStatus.BlockedBy)

	orchestrations := cache.Orchestrations()
	require.Len(t, orchestrations, 1)
	summary := orchestrations[0]
	assert.Equal(t, 2, summary.Executed)
	assert.Equal(t, 1, summary.Blocked)
	assert.Equal(t, 1, summary.Failed)
}

func writePipelineConfig(t *testing.T, dir, id string, deps []string, command string) string {
	t.Helper()

	path := filepath.Join(dir, fmt.Sprintf("%s.yaml", id))

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("id: %s\n", id))
	builder.WriteString("version: \"1.0\"\n")
	builder.WriteString(fmt.Sprintf("name: \"%s pipeline\"\n", id))

	if len(deps) > 0 {
		builder.WriteString("dependencies:\n")

		for _, dep := range deps {
			builder.WriteString(fmt.Sprintf("  - %s\n", dep))
		}
	}

	builder.WriteString("steps:\n")
	builder.WriteString("  - id: run\n")
	builder.WriteString("    type: command\n")
	builder.WriteString(fmt.Sprintf("    command: \"%s\"\n", command))

	require.NoError(t, os.WriteFile(path, []byte(builder.String()), 0o644))

	return path
}

func createRegistry(t *testing.T, pipelines ...registry.Pipeline) *registry.Registry {
	t.Helper()

	registryPath := filepath.Join(t.TempDir(), "registry.json")
	reg, err := registry.NewRegistry(registryPath)
	require.NoError(t, err)

	for _, entry := range pipelines {
		p := entry
		if p.Dependencies == nil {
			p.Dependencies = []string{}
		}

		require.NoError(t, reg.Add(p))
	}

	require.NoError(t, reg.Save())

	return reg
}

func createStatusCache(t *testing.T) *registry.StatusCache {
	t.Helper()

	cachePath := filepath.Join(t.TempDir(), "status-cache.json")
	cache, err := registry.NewStatusCache(cachePath)
	require.NoError(t, err)

	return cache
}

// Ensure the interface check stays in sync with the real apply use case at compile time.
var _ orchestrateapp.PipelineExecutor = (*apppipeline.ApplyUseCase)(nil)
