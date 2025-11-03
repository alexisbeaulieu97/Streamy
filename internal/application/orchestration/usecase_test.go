package orchestration

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

func TestPlanBuildsLevels(t *testing.T) {
	t.Parallel()

	reg := newTestRegistry(t,
		registry.Pipeline{ID: "base@1.0", Path: "/configs/base.yaml"},
		registry.Pipeline{ID: "network@1.0", Path: "/configs/network.yaml"},
		registry.Pipeline{ID: "db@1.0", Path: "/configs/db.yaml", Dependencies: []string{"base@1.0"}},
		registry.Pipeline{ID: "app@1.0", Path: "/configs/app.yaml", Dependencies: []string{"db@1.0", "network@1.0"}},
	)

	cache := newTestStatusCache(t)
	uc, err := NewUseCase(reg, cache, &stubExecutor{})
	require.NoError(t, err)

	plan, err := uc.Plan(context.Background(), "app@1.0")
	require.NoError(t, err)
	require.NotNil(t, plan)

	assert.Equal(t, "app@1.0", plan.RootPipelineID)
	require.Len(t, plan.Levels, 3)
	assert.ElementsMatch(t, []string{"base@1.0", "network@1.0"}, plan.Levels[0])
	assert.Equal(t, []string{"db@1.0"}, plan.Levels[1])
	assert.Equal(t, []string{"app@1.0"}, plan.Levels[2])
}

func TestExecuteRunsLevelsAndUpdatesCache(t *testing.T) {
	t.Parallel()

	reg := newTestRegistry(t,
		registry.Pipeline{ID: "base@1.0", Path: "base.yaml"},
		registry.Pipeline{ID: "db@1.0", Path: "db.yaml", Dependencies: []string{"base@1.0"}},
		registry.Pipeline{ID: "app@1.0", Path: "app.yaml", Dependencies: []string{"db@1.0"}},
	)

	cache := newTestStatusCache(t)
	executor := &stubExecutor{}

	uc, err := NewUseCase(reg, cache, executor)
	require.NoError(t, err)

	result, err := uc.Execute(context.Background(), "app@1.0", false, false)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 3, result.ExecutedCount)
	assert.True(t, result.OverallSuccess)
	assert.Equal(t, 3, len(executor.calls()))

	baseStatus, ok := cache.Get("base@1.0")
	require.True(t, ok)
	assert.Equal(t, registry.StatusReady, baseStatus.Status)
	assert.Empty(t, baseStatus.BlockedBy)
	assert.False(t, baseStatus.LastRun.IsZero())

	dbStatus, ok := cache.Get("db@1.0")
	require.True(t, ok)
	assert.Equal(t, registry.StatusReady, dbStatus.Status)
	assert.Empty(t, dbStatus.BlockedBy)

	appStatus, ok := cache.Get("app@1.0")
	require.True(t, ok)
	assert.Equal(t, registry.StatusReady, appStatus.Status)
	assert.Empty(t, appStatus.BlockedBy)

	orchestrations := cache.Orchestrations()
	require.Len(t, orchestrations, 1)
	assert.Equal(t, "app@1.0", orchestrations[0].RootPipelineID)
	assert.Equal(t, 3, orchestrations[0].Executed)
}

func TestExecutePropagatesFailures(t *testing.T) {
	t.Parallel()

	reg := newTestRegistry(t,
		registry.Pipeline{ID: "base@1.0", Path: "base.yaml"},
		registry.Pipeline{ID: "db@1.0", Path: "db.yaml", Dependencies: []string{"base@1.0"}},
		registry.Pipeline{ID: "app@1.0", Path: "app.yaml", Dependencies: []string{"db@1.0"}},
	)

	cache := newTestStatusCache(t)
	executor := &stubExecutor{
		failures: map[string]error{
			"db.yaml": errors.New("db failure"),
		},
	}

	uc, err := NewUseCase(reg, cache, executor)
	require.NoError(t, err)

	result, err := uc.Execute(context.Background(), "app@1.0", false, false)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 1, result.FailedCount)
	assert.Equal(t, 1, result.BlockedCount)
	assert.False(t, result.OverallSuccess)

	dbSummary := result.PipelineResults["db@1.0"]
	require.NotNil(t, dbSummary)
	assert.False(t, dbSummary.Success)
	assert.Equal(t, registry.StatusFailed.String(), dbSummary.Status)

	appSummary := result.PipelineResults["app@1.0"]
	require.NotNil(t, appSummary)
	assert.False(t, appSummary.Success)
	assert.Equal(t, "db@1.0", appSummary.BlockedBy)
	assert.Equal(t, registry.StatusBlocked.String(), appSummary.Status)

	baseStatus, ok := cache.Get("base@1.0")
	require.True(t, ok)
	assert.Equal(t, registry.StatusReady, baseStatus.Status)
	assert.Empty(t, baseStatus.BlockedBy)

	dbStatus, ok := cache.Get("db@1.0")
	require.True(t, ok)
	assert.Equal(t, registry.StatusFailed, dbStatus.Status)
	assert.Equal(t, "db@1.0", dbStatus.BlockedBy)

	appStatus, ok := cache.Get("app@1.0")
	require.True(t, ok)
	assert.Equal(t, registry.StatusBlocked, appStatus.Status)
	assert.Equal(t, "db@1.0", appStatus.BlockedBy)

	orchestrations := cache.Orchestrations()
	require.Len(t, orchestrations, 1)
	summary := orchestrations[0]
	assert.Equal(t, 2, summary.Executed)
	assert.Equal(t, 1, summary.Blocked)
	assert.Equal(t, 1, summary.Failed)
}

func TestExecuteForceRunsDependents(t *testing.T) {
	t.Parallel()

	reg := newTestRegistry(t,
		registry.Pipeline{ID: "base@1.0", Path: "base.yaml"},
		registry.Pipeline{ID: "db@1.0", Path: "db.yaml", Dependencies: []string{"base@1.0"}},
		registry.Pipeline{ID: "app@1.0", Path: "app.yaml", Dependencies: []string{"db@1.0"}},
	)

	cache := newTestStatusCache(t)
	executor := &stubExecutor{
		failures: map[string]error{
			"db.yaml": errors.New("db failure"),
		},
	}

	uc, err := NewUseCase(reg, cache, executor)
	require.NoError(t, err)

	result, err := uc.Execute(context.Background(), "app@1.0", false, true)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 3, result.ExecutedCount)
	assert.Equal(t, 1, result.FailedCount)
	assert.Equal(t, 0, result.BlockedCount)

	appSummary := result.PipelineResults["app@1.0"]
	require.NotNil(t, appSummary)
	assert.True(t, appSummary.Success)
	assert.True(t, appSummary.Forced)
	assert.Equal(t, "db@1.0", appSummary.ForcedBy)
	assert.Contains(t, appSummary.Summary, "forced despite db@1.0 failure")

	appStatus, ok := cache.Get("app@1.0")
	require.True(t, ok)
	assert.Equal(t, registry.StatusReady, appStatus.Status)
	assert.Empty(t, appStatus.BlockedBy)
}

// Helpers --------------------------------------------------------------------

type stubExecutor struct {
	mu       sync.Mutex
	executed []string
	failures map[string]error
}

func (s *stubExecutor) Apply(_ context.Context, configPath string, _ bool) (*domainpipeline.Pipeline, []domainpipeline.StepResult, *domainpipeline.VerificationSummary, error) {
	s.mu.Lock()
	s.executed = append(s.executed, configPath)
	err := s.failures[configPath]
	s.mu.Unlock()

	if err != nil {
		return nil, nil, nil, err
	}

	return &domainpipeline.Pipeline{
		Name: "stub",
	}, nil, nil, nil
}

func (s *stubExecutor) calls() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]string(nil), s.executed...)
}

func newTestRegistry(t *testing.T, pipelines ...registry.Pipeline) *registry.Registry {
	t.Helper()

	regPath := filepath.Join(t.TempDir(), "registry.json")
	reg, err := registry.NewRegistry(regPath)
	require.NoError(t, err)

	for _, p := range pipelines {
		require.NoError(t, reg.Add(p))
	}

	require.NoError(t, reg.Save())

	return reg
}

func newTestStatusCache(t *testing.T) *registry.StatusCache {
	t.Helper()

	cachePath := filepath.Join(t.TempDir(), "status-cache.json")
	cache, err := registry.NewStatusCache(cachePath)
	require.NoError(t, err)

	return cache
}
