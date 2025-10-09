package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Registry manages the pipeline registry persistence
type Registry struct {
	path      string
	mu        sync.RWMutex
	version   string
	pipelines []Pipeline
}

// NewRegistry creates a new Registry instance and loads it from disk
func NewRegistry(path string) (*Registry, error) {
	r := &Registry{
		path:    path,
		version: "1.0",
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create registry directory: %w", err)
	}

	// Load existing registry or create empty one
	if err := r.Load(); err != nil {
		// If file doesn't exist, start with empty registry
		if !os.IsNotExist(err) {
			return nil, err
		}
		r.pipelines = []Pipeline{}
	}

	return r, nil
}

// Load reads the registry from disk
func (r *Registry) Load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.path)
	if err != nil {
		return err
	}

	var file RegistryFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("failed to parse registry: %w", err)
	}

	r.version = file.Version
	r.pipelines = file.Pipelines

	return nil
}

// Save writes the registry to disk atomically
func (r *Registry) Save() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	file := RegistryFile{
		Version:   r.version,
		Pipelines: r.pipelines,
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	// Write to temporary file first
	tmpPath := r.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, r.path); err != nil {
		os.Remove(tmpPath) // Clean up temp file on failure
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}

// List returns all registered pipelines
func (r *Registry) List() []Pipeline {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]Pipeline, len(r.pipelines))
	copy(result, r.pipelines)
	return result
}

// Get retrieves a pipeline by ID
func (r *Registry) Get(id string) (Pipeline, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.pipelines {
		if p.ID == id {
			return p, nil
		}
	}

	return Pipeline{}, fmt.Errorf("pipeline not found: %s", id)
}

// Add adds a new pipeline to the registry
func (r *Registry) Add(p Pipeline) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate ID
	for _, existing := range r.pipelines {
		if existing.ID == p.ID {
			return fmt.Errorf("pipeline with ID %s already exists", p.ID)
		}
	}

	r.pipelines = append(r.pipelines, p)
	return nil
}

// Update updates an existing pipeline
func (r *Registry) Update(p Pipeline) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, existing := range r.pipelines {
		if existing.ID == p.ID {
			r.pipelines[i] = p
			return nil
		}
	}

	return fmt.Errorf("pipeline not found: %s", p.ID)
}

// Remove removes a pipeline from the registry
func (r *Registry) Remove(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, p := range r.pipelines {
		if p.ID == id {
			r.pipelines = append(r.pipelines[:i], r.pipelines[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("pipeline not found: %s", id)
}
