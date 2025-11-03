package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	assert "github.com/stretchr/testify/assert"
	require "github.com/stretchr/testify/require"
)

func TestStatusCacheNew(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	cache, err := NewStatusCache(cachePath)
	require.NoError(t, err)
	assert.NotNil(t, cache)

	_, ok := cache.Get("any-id")
	assert.False(t, ok)
	assert.Empty(t, cache.Orchestrations())
}

func TestStatusCacheLoadExisting(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	testData, err := os.ReadFile("../../testdata/cache/populated-cache.json")
	require.NoError(t, err)
	err = os.WriteFile(cachePath, testData, 0o644)
	require.NoError(t, err)

	cache, err := NewStatusCache(cachePath)
	require.NoError(t, err)

	status, ok := cache.Get("dev-env")
	assert.True(t, ok)
	assert.Equal(t, StatusReady, status.Status)
	assert.Equal(t, "", status.BlockedBy)
	assert.Equal(t, "All 5 steps passed", status.Summary)

	orchestrations := cache.Orchestrations()
	require.Len(t, orchestrations, 1)
	assert.Equal(t, "infra@1.0", orchestrations[0].RootPipelineID)
	assert.Equal(t, 3, orchestrations[0].TotalPipelines)
}

func TestStatusCacheSet(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	cache, err := NewStatusCache(cachePath)
	require.NoError(t, err)

	now := time.Now().UTC()
	status := CachedStatus{
		Status:    StatusBlocked,
		LastRun:   now,
		Summary:   "Dependency missing",
		StepCount: 0,
		BlockedBy: "db@1.0",
	}

	err = cache.Set("frontend@1.0", status)
	require.NoError(t, err)

	retrieved, ok := cache.Get("frontend@1.0")
	assert.True(t, ok)
	assert.Equal(t, StatusBlocked, retrieved.Status)
	assert.Equal(t, "db@1.0", retrieved.BlockedBy)
}

func TestStatusCacheInvalidate(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	cache, err := NewStatusCache(cachePath)
	require.NoError(t, err)

	status := CachedStatus{
		Status:    StatusReady,
		LastRun:   time.Now(),
		Summary:   "Execution succeeded",
		StepCount: 4,
	}

	require.NoError(t, cache.Set("pipeline", status))
	require.NoError(t, cache.Invalidate("pipeline"))

	_, ok := cache.Get("pipeline")
	assert.False(t, ok)
}

func TestStatusCacheInvalidateAll(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	cache, err := NewStatusCache(cachePath)
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("pipeline-%d", i)
		require.NoError(t, cache.Set(id, CachedStatus{
			Status:    StatusReady,
			LastRun:   time.Now(),
			Summary:   "OK",
			StepCount: i,
		}))
	}

	require.NoError(t, cache.InvalidateAll())

	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("pipeline-%d", i)
		_, ok := cache.Get(id)
		assert.False(t, ok)
	}
}

func TestStatusCacheSave(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	cache, err := NewStatusCache(cachePath)
	require.NoError(t, err)

	require.NoError(t, cache.Set("pipeline", CachedStatus{
		Status:    StatusReady,
		LastRun:   time.Now().UTC().Round(time.Second),
		Summary:   "Executed successfully",
		StepCount: 6,
	}))

	require.NoError(t, cache.AddOrchestration(OrchestrationSummary{
		RootPipelineID: "pipeline@1.0",
		TotalPipelines: 2,
		Executed:       2,
		Blocked:        0,
		Failed:         0,
		PipelineOrder:  []string{"pipeline@1.0", "secondary@1.0"},
		StartTime:      time.Now().UTC(),
		EndTime:        time.Now().UTC(),
	}))

	require.NoError(t, cache.Save())

	cache2, err := NewStatusCache(cachePath)
	require.NoError(t, err)

	retrieved, ok := cache2.Get("pipeline")
	assert.True(t, ok)
	assert.Equal(t, StatusReady, retrieved.Status)

	orchestrations := cache2.Orchestrations()
	require.Len(t, orchestrations, 1)
	assert.Equal(t, "pipeline@1.0", orchestrations[0].RootPipelineID)
}

// Run this test with `go test -race` to exercise the concurrency path.
func TestStatusCacheConcurrency(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	cache, err := NewStatusCache(cachePath)
	require.NoError(t, err)

	done := make(chan bool)
	errCh := make(chan error, 1)

	go func() {
		for i := 0; i < 100; i++ {
			if err := cache.Set("pipeline-1", CachedStatus{
				Status:    StatusReady,
				LastRun:   time.Now(),
				Summary:   "Stable",
				StepCount: 5,
			}); err != nil {
				select {
				case errCh <- err:
				default:
				}
				break
			}
		}

		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			cache.Get("pipeline-1")
		}

		done <- true
	}()

	<-done
	<-done

	select {
	case err := <-errCh:
		require.NoError(t, err)
	default:
	}

	status, ok := cache.Get("pipeline-1")
	assert.True(t, ok)
	assert.Equal(t, StatusReady, status.Status)
}

func TestStatusCacheClearOrchestrations(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	cache, err := NewStatusCache(cachePath)
	require.NoError(t, err)

	require.NoError(t, cache.AddOrchestration(OrchestrationSummary{RootPipelineID: "demo@1.0"}))
	require.NoError(t, cache.ClearOrchestrations())

	assert.Empty(t, cache.Orchestrations())
}
