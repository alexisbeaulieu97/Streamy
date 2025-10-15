package registry

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusCacheNew(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	cache, err := NewStatusCache(cachePath)
	require.NoError(t, err)
	assert.NotNil(t, cache)

	// Should start empty
	_, ok := cache.Get("any-id")
	assert.False(t, ok)
}

func TestStatusCacheLoadExisting(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	// Copy test fixture
	testData, err := os.ReadFile("../../testdata/cache/populated-cache.json")
	require.NoError(t, err)
	err = os.WriteFile(cachePath, testData, 0644)
	require.NoError(t, err)

	cache, err := NewStatusCache(cachePath)
	require.NoError(t, err)

	// Check loaded data
	status, ok := cache.Get("dev-env")
	assert.True(t, ok)
	assert.Equal(t, StatusSatisfied, status.Status)
	assert.Equal(t, "All 5 steps passed", status.Summary)
	assert.Equal(t, 5, status.StepCount)
}

func TestStatusCacheSet(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	cache, err := NewStatusCache(cachePath)
	require.NoError(t, err)

	status := CachedStatus{
		Status:    StatusSatisfied,
		LastRun:   time.Now(),
		Summary:   "All steps passed",
		StepCount: 5,
	}

	err = cache.Set("test-pipeline", status)
	require.NoError(t, err)

	retrieved, ok := cache.Get("test-pipeline")
	assert.True(t, ok)
	assert.Equal(t, StatusSatisfied, retrieved.Status)
	assert.Equal(t, "All steps passed", retrieved.Summary)
}

func TestStatusCacheInvalidate(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	cache, err := NewStatusCache(cachePath)
	require.NoError(t, err)

	status := CachedStatus{
		Status:    StatusSatisfied,
		LastRun:   time.Now(),
		Summary:   "All steps passed",
		StepCount: 5,
	}

	err = cache.Set("test-pipeline", status)
	require.NoError(t, err)

	err = cache.Invalidate("test-pipeline")
	require.NoError(t, err)

	_, ok := cache.Get("test-pipeline")
	assert.False(t, ok)
}

func TestStatusCacheInvalidateAll(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	cache, err := NewStatusCache(cachePath)
	require.NoError(t, err)

	// Add multiple statuses
	for i := 0; i < 5; i++ {
		status := CachedStatus{
			Status:    StatusSatisfied,
			LastRun:   time.Now(),
			Summary:   "All steps passed",
			StepCount: 5,
		}
		err = cache.Set("pipeline-"+string(rune('0'+i)), status)
		require.NoError(t, err)
	}

	err = cache.InvalidateAll()
	require.NoError(t, err)

	// Check all are gone
	for i := 0; i < 5; i++ {
		_, ok := cache.Get("pipeline-" + string(rune('0'+i)))
		assert.False(t, ok)
	}
}

func TestStatusCacheSave(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	cache, err := NewStatusCache(cachePath)
	require.NoError(t, err)

	status := CachedStatus{
		Status:    StatusDrifted,
		LastRun:   time.Now(),
		Summary:   "2 steps need changes",
		StepCount: 8,
	}

	err = cache.Set("test-pipeline", status)
	require.NoError(t, err)

	err = cache.Save()
	require.NoError(t, err)

	// Load in a new cache instance
	cache2, err := NewStatusCache(cachePath)
	require.NoError(t, err)

	retrieved, ok := cache2.Get("test-pipeline")
	assert.True(t, ok)
	assert.Equal(t, StatusDrifted, retrieved.Status)
	assert.Equal(t, "2 steps need changes", retrieved.Summary)
}

func TestStatusCacheConcurrency(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache.json")

	cache, err := NewStatusCache(cachePath)
	require.NoError(t, err)

	// Simulate concurrent reads and writes
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			status := CachedStatus{
				Status:    StatusSatisfied,
				LastRun:   time.Now(),
				Summary:   "All steps passed",
				StepCount: 5,
			}
			_ = cache.Set("pipeline-1", status)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			cache.Get("pipeline-1")
		}
		done <- true
	}()

	<-done
	<-done
	// If we get here without deadlock, the test passes
}
