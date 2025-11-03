package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// StatusCache persists pipeline status between sessions alongside recent orchestration summaries.
type StatusCache struct {
	path           string
	mu             sync.RWMutex
	version        string
	statuses       map[string]CachedStatus
	orchestrations []OrchestrationSummary
}

const maxOrchestrationHistory = 100

// NewStatusCache creates a new StatusCache instance and loads it from disk.
func NewStatusCache(path string) (*StatusCache, error) {
	c := &StatusCache{
		path:           path,
		version:        statusCacheVersion,
		statuses:       make(map[string]CachedStatus),
		orchestrations: []OrchestrationSummary{},
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	if err := MigrateStatusCacheFile(path); err != nil {
		return nil, err
	}

	if err := c.Load(); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}

	return c, nil
}

// Load reads the cache from disk.
func (c *StatusCache) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(c.path)
	if err != nil {
		return fmt.Errorf("failed to read cache file %q: %w", c.path, err)
	}

	var file StatusCacheFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("failed to parse cache: %w", err)
	}

	if file.Version == "" {
		file.Version = "1.0"
	}

	c.version = file.Version
	if file.Statuses == nil {
		c.statuses = make(map[string]CachedStatus)
	} else {
		c.statuses = file.Statuses
	}

	if file.Orchestrations == nil {
		c.orchestrations = []OrchestrationSummary{}
	} else {
		c.orchestrations = append([]OrchestrationSummary(nil), file.Orchestrations...)
	}

	return nil
}

// Save writes the cache to disk atomically.
func (c *StatusCache) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.saveLocked()
}

// Get retrieves cached status for a pipeline.
func (c *StatusCache) Get(pipelineID string) (CachedStatus, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	status, ok := c.statuses[pipelineID]

	return status, ok
}

// Set updates the cached status for a pipeline.
func (c *StatusCache) Set(pipelineID string, status CachedStatus) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.statuses[pipelineID] = status

	return c.saveLocked()
}

// Invalidate removes cached status for a pipeline.
func (c *StatusCache) Invalidate(pipelineID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.statuses, pipelineID)

	return c.saveLocked()
}

// InvalidateAll removes all cached statuses.
func (c *StatusCache) InvalidateAll() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.statuses = make(map[string]CachedStatus)

	return c.saveLocked()
}

// Orchestrations returns a copy of persisted orchestration summaries.
func (c *StatusCache) Orchestrations() []OrchestrationSummary {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.orchestrations) == 0 {
		return []OrchestrationSummary{}
	}

	out := make([]OrchestrationSummary, len(c.orchestrations))
	copy(out, c.orchestrations)

	return out
}

// AddOrchestration appends a new orchestration summary to the cache.
func (c *StatusCache) AddOrchestration(summary OrchestrationSummary) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if maxOrchestrationHistory > 0 && len(c.orchestrations) >= maxOrchestrationHistory {
		trimStart := len(c.orchestrations) - maxOrchestrationHistory + 1
		c.orchestrations = append([]OrchestrationSummary(nil), c.orchestrations[trimStart:]...)
	}

	c.orchestrations = append(c.orchestrations, summary)

	return c.saveLocked()
}

// ClearOrchestrations removes all persisted orchestration summaries.
func (c *StatusCache) ClearOrchestrations() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.orchestrations = []OrchestrationSummary{}

	return c.saveLocked()
}

func (c *StatusCache) saveLocked() error {
	file := StatusCacheFile{
		Version:        c.version,
		Statuses:       c.statuses,
		Orchestrations: c.orchestrations,
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	return writeAtomic(c.path, data, 0o600)
}
