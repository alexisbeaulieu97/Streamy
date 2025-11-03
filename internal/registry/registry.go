package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
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
		version: registryFileVersion,
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create registry directory: %w", err)
	}

	if err := MigrateRegistryFile(path); err != nil {
		return nil, err
	}

	// Load existing registry or create empty one
	if err := r.Load(); err != nil {
		// If file doesn't exist, start with empty registry
		if !errors.Is(err, os.ErrNotExist) {
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
		return fmt.Errorf("failed to read registry file %q: %w", r.path, err)
	}

	var file File
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

	file := File{
		Version:   r.version,
		Pipelines: r.pipelines,
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	// Write to temporary file first
	tmpPath := r.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write temporary registry file %q: %w", tmpPath, err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, r.path); err != nil {
		_ = os.Remove(tmpPath) // Clean up temp file on failure
		return fmt.Errorf("failed to rename temporary registry file %q to %q: %w", tmpPath, r.path, err)
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

// FindUnregisteredDependencies returns dependency IDs that are not yet present in the registry.
func (r *Registry) FindUnregisteredDependencies(dependencies []string) []string {
	if len(dependencies) == 0 {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	existing := make(map[string]struct{}, len(r.pipelines))

	for _, pipeline := range r.pipelines {
		existing[pipeline.ID] = struct{}{}
	}

	missing := make(map[string]struct{})

	for _, dep := range dependencies {
		if _, ok := existing[dep]; !ok {
			missing[dep] = struct{}{}
		}
	}

	if len(missing) == 0 {
		return nil
	}

	result := make([]string, 0, len(missing))
	for dep := range missing {
		result = append(result, dep)
	}

	slices.Sort(result)

	return result
}

// ValidateDependencyCycles ensures that introducing the provided dependency edges does not create cycles.
func (r *Registry) ValidateDependencyCycles(pipelineID string, dependencies []string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, dep := range dependencies {
		if dep == pipelineID {
			return fmt.Errorf("%w: pipeline %s cannot depend on itself", ErrSelfDependency, pipelineID)
		}
	}

	adjacency := make(map[string][]string, len(r.pipelines)+1)

	for _, pipeline := range r.pipelines {
		adjacency[pipeline.ID] = append([]string(nil), pipeline.Dependencies...)
		for _, dep := range pipeline.Dependencies {
			if _, ok := adjacency[dep]; !ok {
				adjacency[dep] = nil
			}
		}
	}

	adjacency[pipelineID] = append([]string(nil), dependencies...)
	for _, dep := range dependencies {
		if _, ok := adjacency[dep]; !ok {
			adjacency[dep] = nil
		}
	}

	visited := make(map[string]bool, len(adjacency))
	stack := make(map[string]bool, len(adjacency))

	var path []string

	var cycle []string

	var dfs func(string) bool

	dfs = func(node string) bool {
		visited[node] = true
		stack[node] = true
		path = append(path, node)

		for _, dep := range adjacency[node] {
			if !visited[dep] {
				if dfs(dep) {
					return true
				}
			} else if stack[dep] {
				for i, candidate := range path {
					if candidate == dep {
						cycle = append(append([]string(nil), path[i:]...), dep)

						return true
					}
				}

				cycle = append([]string{dep}, dep)

				return true
			}
		}

		stack[node] = false
		path = path[:len(path)-1]

		return false
	}

	for node := range adjacency {
		if !visited[node] {
			if dfs(node) {
				return fmt.Errorf("dependency cycle detected: %s", strings.Join(cycle, " -> "))
			}
		}
	}

	return nil
}

// PipelinesByID returns a copy of the registered pipelines indexed by their canonical IDs.
func (r *Registry) PipelinesByID() map[string]Pipeline {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]Pipeline, len(r.pipelines))
	for _, pipeline := range r.pipelines {
		result[pipeline.ID] = pipeline
	}

	return result
}

// DependentsIndex builds a reverse dependency lookup: dependency ID → dependent pipeline IDs.
func (r *Registry) DependentsIndex() map[string][]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	index := make(map[string][]string)

	for _, pipeline := range r.pipelines {
		for _, depID := range pipeline.Dependencies {
			index[depID] = append(index[depID], pipeline.ID)
		}
	}

	for id := range index {
		slices.Sort(index[id])
	}

	return index
}

// ResolveCanonicalID expands pipeline references, supporting the `@latest`
// alias to target the highest registered version. It returns the concrete
// canonical identifier (<id>@<version>) or an error if no versions exist.
func (r *Registry) ResolveCanonicalID(reference string) (string, error) {
	ref := strings.TrimSpace(reference)
	if ref == "" {
		return "", fmt.Errorf("pipeline reference cannot be empty")
	}

	id, version, splitErr := ParseCanonicalPipelineID(ref)
	if splitErr == nil {
		if strings.EqualFold(version, "latest") {
			return r.resolveLatestVersion(id)
		}

		return ref, nil
	}

	// Allow <id>@latest even if ParseCanonicalPipelineID rejects "latest".
	if idx := strings.LastIndex(ref, canonicalIDSeparator); idx > 0 {
		baseID := ref[:idx]
		suffix := ref[idx+1:]

		if strings.EqualFold(suffix, "latest") {
			if err := ValidatePipelineID(baseID); err != nil {
				return "", fmt.Errorf("invalid pipeline reference %q: %w", reference, err)
			}

			return r.resolveLatestVersion(baseID)
		}
	}

	// Otherwise require canonical <id>@<version> input.
	return "", fmt.Errorf("invalid pipeline reference %q: %w", reference, splitErr)
}

func (r *Registry) resolveLatestVersion(baseID string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var versions []string
	for _, pipeline := range r.pipelines {
		pid, pver, err := ParseCanonicalPipelineID(pipeline.ID)
		if err != nil {
			continue
		}

		if pid == baseID {
			versions = append(versions, pver)
		}
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("no registered versions found for pipeline %q", baseID)
	}

	latest := versions[0]
	for _, candidate := range versions[1:] {
		if comparePipelineVersions(candidate, latest) > 0 {
			latest = candidate
		}
	}

	return fmt.Sprintf("%s%s%s", baseID, canonicalIDSeparator, latest), nil
}

// ReconcileStatuses recomputes pipeline readiness based on currently registered dependencies.
// It updates the registry entries in place and synchronises the optional status cache.
// The returned slice contains pipeline IDs that transitioned from blocked to ready during reconciliation.
func (r *Registry) ReconcileStatuses(cache *StatusCache) []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.pipelines) == 0 {
		return nil
	}

	existing := make(map[string]struct{}, len(r.pipelines))
	for _, pipeline := range r.pipelines {
		existing[pipeline.ID] = struct{}{}
	}

	var unblocked []string

	for i := range r.pipelines {
		pipeline := &r.pipelines[i]
		previousStatus := pipeline.Status

		missing := collectMissingDependencies(pipeline.Dependencies, existing)

		if len(missing) == 0 {
			pipeline.Status = StatusReady
			if len(pipeline.BlockedBy) > 0 {
				pipeline.BlockedBy = nil
			}

			if previousStatus == StatusBlocked {
				unblocked = append(unblocked, pipeline.ID)
				if cache != nil {
					reconcileCacheReady(cache, pipeline.ID)
				}
			}

			continue
		}

		pipeline.Status = StatusBlocked

		pipeline.BlockedBy = append([]string(nil), missing...)

		if cache != nil {
			reconcileCacheBlocked(cache, pipeline.ID, missing)
		}
	}

	if len(unblocked) > 1 {
		slices.Sort(unblocked)
	}

	return unblocked
}

func collectMissingDependencies(dependencies []string, existing map[string]struct{}) []string {
	if len(dependencies) == 0 {
		return nil
	}

	missingSet := make(map[string]struct{})

	for _, dep := range dependencies {
		if _, ok := existing[dep]; !ok {
			missingSet[dep] = struct{}{}
		}
	}

	if len(missingSet) == 0 {
		return nil
	}

	missing := make([]string, 0, len(missingSet))
	for dep := range missingSet {
		missing = append(missing, dep)
	}

	slices.Sort(missing)

	return missing
}

func comparePipelineVersions(a, b string) int {
	av, okA := parseVersionParts(a)
	bv, okB := parseVersionParts(b)

	switch {
	case okA && okB:
		maxLen := len(av)
		if len(bv) > maxLen {
			maxLen = len(bv)
		}

		for i := 0; i < maxLen; i++ {
			var ai, bi int
			if i < len(av) {
				ai = av[i]
			}
			if i < len(bv) {
				bi = bv[i]
			}

			if ai > bi {
				return 1
			}
			if ai < bi {
				return -1
			}
		}

		return strings.Compare(a, b)
	case okA:
		return 1
	case okB:
		return -1
	default:
		return strings.Compare(a, b)
	}
}

func parseVersionParts(version string) ([]int, bool) {
	base := version
	if idx := strings.IndexAny(base, "-+"); idx >= 0 {
		base = base[:idx]
	}

	if base == "" {
		return nil, false
	}

	segments := strings.Split(base, ".")
	parts := make([]int, len(segments))

	for i, segment := range segments {
		if segment == "" {
			return nil, false
		}

		value, err := strconv.Atoi(segment)
		if err != nil {
			return nil, false
		}

		parts[i] = value
	}

	return parts, true
}

func reconcileCacheReady(cache *StatusCache, pipelineID string) {
	_ = cache.Set(pipelineID, CachedStatus{
		Status:  StatusReady,
		Summary: "Dependencies satisfied",
	})
}

func reconcileCacheBlocked(cache *StatusCache, pipelineID string, missing []string) {
	reason := strings.Join(missing, ", ")
	_ = cache.Set(pipelineID, CachedStatus{
		Status:    StatusBlocked,
		Summary:   fmt.Sprintf("Missing dependencies: %s", reason),
		BlockedBy: reason,
	})
}
