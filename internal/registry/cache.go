package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// StatusCache persists pipeline status between sessions
type StatusCache struct {
	path     string
	mu       sync.RWMutex
	version  string
	statuses map[string]CachedStatus
}

// NewStatusCache creates a new StatusCache instance and loads it from disk
func NewStatusCache(path string) (*StatusCache, error) {
	c := &StatusCache{
		path:     path,
		version:  "1.0",
		statuses: make(map[string]CachedStatus),
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Load existing cache or start with empty one
	if err := c.Load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		// Start with empty cache if file doesn't exist
	}

	return c, nil
}

// Load reads the cache from disk
func (c *StatusCache) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(c.path)
	if err != nil {
		return err
	}

	var file StatusCacheFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("failed to parse cache: %w", err)
	}

	c.version = file.Version
	c.statuses = file.Statuses
	if c.statuses == nil {
		c.statuses = make(map[string]CachedStatus)
	}

	return nil
}

// Save writes the cache to disk atomically
func (c *StatusCache) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	file := StatusCacheFile{
		Version:  c.version,
		Statuses: c.statuses,
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	// Write to temporary file first
	tmpPath := c.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, c.path); err != nil {
		_ = os.Remove(tmpPath) // Clean up temp file on failure
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}

// Get retrieves cached status for a pipeline
func (c *StatusCache) Get(pipelineID string) (CachedStatus, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	status, ok := c.statuses[pipelineID]
	return status, ok
}

// Set updates the cached status for a pipeline
func (c *StatusCache) Set(pipelineID string, status CachedStatus) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.statuses[pipelineID] = status
	return nil
}

// Invalidate removes cached status for a pipeline
func (c *StatusCache) Invalidate(pipelineID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.statuses, pipelineID)
	return nil
}

// InvalidateAll removes all cached statuses
func (c *StatusCache) InvalidateAll() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.statuses = make(map[string]CachedStatus)
	return nil
}
