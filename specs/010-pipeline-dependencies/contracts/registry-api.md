# Registry API Contract: Pipeline Dependencies

**Feature**: 010-pipeline-dependencies  
**Date**: October 30, 2025  
**Audience**: Maintainers & contributors working with `internal/registry`

This document mirrors the Go API that powers pipeline dependency management. It reflects the implementation that landed with Feature 010 (no future intent or pseudo-code).

---

## Registry (`internal/registry/registry.go`)

### Construction & Persistence

```go
func NewRegistry(path string) (*Registry, error)
func (r *Registry) Load() error
func (r *Registry) Save() error
```

- `NewRegistry` initialises the registry at the supplied path, creating parent directories and running migrations as needed.
- `Load` / `Save` read and persist `File` (versioned JSON) using atomic writes.

### Queries

```go
func (r *Registry) List() []Pipeline
func (r *Registry) Get(id string) (Pipeline, error)
func (r *Registry) PipelinesByID() map[string]Pipeline
func (r *Registry) DependentsIndex() map[string][]string
```

- `List` returns a copy of all pipelines sorted by registration time.
- `Get` fetches a canonical pipeline (`id@version`) or returns `ErrPipelineNotFound`.
- `PipelinesByID` materialises a map for fast lookups (used by status reconciliation and graph builders).
- `DependentsIndex` provides the reverse adjacency map: dependency ID → sorted list of dependents.

### Mutations

```go
func (r *Registry) Add(p Pipeline) error
func (r *Registry) Update(p Pipeline) error
func (r *Registry) Remove(id string) error
```

- `Add` validates canonical IDs, dependency existence, and cycle safety before appending the pipeline and saving.
- `Update` replaces an existing pipeline entry by ID (used by migrations / reconciliations) and persists immediately.
- `Remove` deletes a pipeline by canonical ID and saves; callers must ensure dependents are handled beforehand.

### Dependency Utilities

```go
func (r *Registry) FindUnregisteredDependencies(dependencies []string) []string
func (r *Registry) ValidateDependencyCycles(pipelineID string, dependencies []string) error
func (r *Registry) ResolveCanonicalID(reference string) (string, error)
```

- `FindUnregisteredDependencies` returns a sorted list of dependency IDs missing from the registry.
- `ValidateDependencyCycles` runs Tarjan/Kahn style cycle detection against the current registry plus the provided dependency set. Errors:
  - `ErrCircularDependency` (cycle detected)
  - `ErrSelfDependency` (pipeline depends on itself)
  - `ErrMissingDependency` (dependency absent; surfaced when callers opt-in to strict mode)
- `ResolveCanonicalID` validates identifiers and expands the CLI-only `@latest` alias to the highest registered version. It returns a concrete `<id>@<version>` or an error when no versions exist.

### Status Reconciliation

```go
func (r *Registry) ReconcileStatuses(cache *StatusCache) []string
```

- Recomputes each pipeline’s `Status` / `BlockedBy` set using the current registry graph.
- Persists updates via the supplied `StatusCache`.
- Returns a sorted slice of pipeline IDs that transitioned to `ready` during reconciliation.

---

## Dependency Graph (`internal/registry/graph.go`)

```go
func NewDependencyGraph(reg *Registry, rootID string) (*DependencyGraph, error)
func (g *DependencyGraph) Dependencies(pipelineID string) []string
func (g *DependencyGraph) Dependents(pipelineID string) []string
func (g *DependencyGraph) GetTransitiveDependents(pipelineID string) []string
func (g *DependencyGraph) TopologicalSort() ([][]string, error)
```

- `NewDependencyGraph` walks the registry starting at `rootID`, returning an error if references are missing or cycles are detected.
- `Dependencies` / `Dependents` expose direct adjacency lists (defensive copies).
- `GetTransitiveDependents` returns all downstream pipelines that rely on the given ID.
- `TopologicalSort` produces execution levels suitable for orchestration (parallel pipelines share a level).

---

## Status Cache (`internal/registry/status_cache.go`)

```go
func NewStatusCache(path string) (*StatusCache, error)
func (c *StatusCache) Get(id string) (CachedStatus, bool)
func (c *StatusCache) Set(id string, status CachedStatus) error
func (c *StatusCache) Invalidate(id string) error
func (c *StatusCache) InvalidateAll() error
func (c *StatusCache) AddOrchestration(summary OrchestrationSummary) error
func (c *StatusCache) ClearOrchestrations() error
```

- All mutators keep the write lock held while persisting (see `saveLocked`) to avoid lost updates.
- `AddOrchestration` / `ClearOrchestrations` manage the rolling history surfaced by `streamy run --registry`.

---

## Canonical ID Helpers (`internal/registry/id.go`)

```go
func BuildCanonicalPipelineID(id, version string) (string, error)
func ParseCanonicalPipelineID(value string) (string, string, error)
func ValidateCanonicalPipelineID(value string) error
func IsCanonicalPipelineID(value string) bool
```

- All helpers enforce `<id>@<version>` without whitespace padding. Parsing rejects inputs such as `"app @ 1.0"`.
- `BuildCanonicalPipelineID` trims input fields before validation to accept `streamy add --id " app "` style CLI entries while still producing canonical identifiers.

---

## Error Summary

- `ErrPipelineNotFound`
- `ErrPipelineExists`
- `ErrMissingDependency`
- `ErrCircularDependency`
- `ErrSelfDependency`

Errors follow the standard Go wrapping conventions so callers can use `errors.Is`.

---

## Usage Notes

- Registry mutations persist immediately; callers should handle errors and avoid holding locks across long-running operations.
- Dependency-aware operations (graphs, reconciliation) operate on the in-memory snapshot returned by `NewRegistry`/`Load`. Rebuild the graph after each mutation when orchestration is involved.
- CLI commands use `ResolveCanonicalID` to support `@latest`, but pipeline YAML and dependency declarations must remain explicit (`<id>@<version>`).

