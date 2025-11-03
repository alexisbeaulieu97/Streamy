# Quickstart: Pipeline Dependencies

**Feature**: 010-pipeline-dependencies  
**Date**: October 28, 2025  
**Audience**: Developers implementing this feature

## Overview

This guide walks through implementing pipeline dependency orchestration in Streamy. Follow these steps sequentially to build the feature from foundation to completion.

---

## Prerequisites

- Streamy codebase checked out on branch `010-pipeline-dependencies`
- Go 1.25.1 installed
- Familiarity with Streamy's domain-driven architecture (see `docs/ADR/002-port-placement-at-boundary.md`)
- Read `specs/010-pipeline-dependencies/spec.md`, `research.md`, and `data-model.md`

---

## Implementation Phases

### Phase 1: Registry Schema Evolution (Foundation)

**Goal**: Add dependency field to registry without breaking existing installations

#### Step 1.1: Update Registry Types

**File**: `internal/registry/types.go`

Add `Dependencies` field to `Pipeline` struct:
```go
type Pipeline struct {
    // ... existing fields ...
    Dependencies []string  `json:"dependencies,omitempty"` // NEW
    // ... runtime state fields ...
}
```

Add `StatusBlocked` to status enum:
```go
const (
    // ... existing statuses ...
    StatusBlocked   PipelineStatus = "blocked"  // NEW
)
```

Update status methods to handle blocked:
```go
func (s PipelineStatus) Icon() string {
    // ... existing cases ...
    case StatusBlocked:
        return "🟠"
}

func (s PipelineStatus) Color() lipgloss.Color {
    // ... existing cases ...
    case StatusBlocked:
        return lipgloss.Color("208") // orange
}
```

Add `BlockedBy` field to `ExecutionResult`:
```go
type ExecutionResult struct {
    // ... existing fields ...
    BlockedBy   string         `json:"blocked_by,omitempty"`  // NEW
}
```

Update registry file version:
```go
const (
    RegistryFileVersion = "2.0"  // Changed from "1.0"
)
```

Add orchestration summary type:
```go
type OrchestrationSummary struct {
    RootPipelineID string    `json:"root_pipeline_id"`
    StartTime      time.Time `json:"start_time"`
    EndTime        time.Time `json:"end_time"`
    TotalPipelines int       `json:"total_pipelines"`
    Executed       int       `json:"executed"`
    Blocked        int       `json:"blocked"`
    Failed         int       `json:"failed"`
    PipelineOrder  []string  `json:"pipeline_order"`
}

type StatusCacheFile struct {
    Version       string                  `json:"version"`
    Statuses      map[string]CachedStatus `json:"statuses"`
    Orchestrations []OrchestrationSummary  `json:"orchestrations,omitempty"` // NEW
}
```

**Tests**: `internal/registry/types_test.go`
- Test JSON marshaling with dependencies field
- Test blocked status icon/color methods
- Test backward compatibility (missing dependencies field → empty slice)

---

#### Step 1.2: Create Migration Logic

**File**: `internal/registry/migration.go` (new)

```go
package registry

import (
    "encoding/json"
    "fmt"
    "os"
    "time"
)

// MigrateRegistryFile upgrades older registry formats to current version
func MigrateRegistryFile(path string) error {
    // Read existing file
    data, err := os.ReadFile(path)
    if err != nil {
        return fmt.Errorf("read registry: %w", err)
    }
    
    var file File
    if err := json.Unmarshal(data, &file); err != nil {
        return fmt.Errorf("parse registry: %w", err)
    }
    
    // Check if migration needed
    if file.Version == RegistryFileVersion {
        return nil // Already current version
    }
    
    // Backup original
    backupPath := fmt.Sprintf("%s.v%s.bak", path, file.Version)
    if err := os.WriteFile(backupPath, data, 0644); err != nil {
        return fmt.Errorf("create backup: %w", err)
    }
    
    // Apply migrations
    switch file.Version {
    case "1.0", "": // Empty version treated as 1.0
        // Add dependencies field to all pipelines if missing
        for i := range file.Pipelines {
            if file.Pipelines[i].Dependencies == nil {
                file.Pipelines[i].Dependencies = []string{}
            }
        }
        file.Version = "2.0"
    default:
        return fmt.Errorf("unsupported registry version: %s", file.Version)
    }
    
    // Write updated registry
    updatedData, err := json.MarshalIndent(file, "", "  ")
    if err != nil {
        return fmt.Errorf("serialize registry: %w", err)
    }
    
    if err := os.WriteFile(path, updatedData, 0644); err != nil {
        return fmt.Errorf("write registry: %w", err)
    }
    
    return nil
}
```

**Integration**: Call `MigrateRegistryFile` in `registry.Load()` before parsing

**Tests**: `internal/registry/migration_test.go`
- Test v1.0 → v2.0 migration
- Test backup file creation
- Test idempotency (migrating v2.0 is no-op)

---

#### Step 1.3: Update YAML Schema to Support Dependencies

**File**: `internal/infrastructure/config/schema.go`

Add required `ID` field and optional `Dependencies` field to pipeline config:

```go
type PipelineConfig struct {
    ID           string       `yaml:"id"`                     // NEW: Required pipeline identifier
    Version      string       `yaml:"version"`
    Name         string       `yaml:"name"`
    Description  string       `yaml:"description,omitempty"`
    Dependencies []string     `yaml:"dependencies,omitempty"` // NEW: Upstream dependencies in canonical <id>@<version> format
    Steps        []StepConfig `yaml:"steps"`
    // ... existing fields ...
}
```

Update validation:

```go
func (c *PipelineConfig) Validate() error {
    // ... existing validation ...
    
    // NEW: Validate required ID field
    if c.ID == "" {
        return fmt.Errorf("pipeline config missing required 'id' field")
    }
    
    // NEW: Validate dependencies if provided
    for _, depID := range c.Dependencies {
        if !isValidPipelineID(depID) {
            return fmt.Errorf("invalid dependency ID %q: must contain only alphanumeric, dash, underscore, or @", depID)
        }
    }
    
    return nil
}

var pipelineIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func isValidPipelineID(id string) bool {
    return pipelineIDPattern.MatchString(id)
}
```

**Tests**: `internal/infrastructure/config/schema_test.go`
- Test parsing YAML with `dependencies` field
- Test validation of invalid pipeline IDs
- Test empty dependencies list (valid)

---

#### Step 1.4: Update Add Command to Auto-Import Dependencies

**File**: `cmd/streamy/add.go`

Update registration flow to import dependencies from config:

```go
func (op *addOperation) registerPipeline() error {
    // Load pipeline config
    cfg, err := config.Load(op.absPath)
    if err != nil {
        return op.fail(/* ... */)
    }
    
    // Validate required fields
    if cfg.ID == "" || cfg.Version == "" {
        return op.fail("invalid config", "Pipeline YAML must include 'id' and 'version' fields", nil, "")
    }
    
    // Build canonical registry ID: <id>@<version>
    pipelineID := fmt.Sprintf("%s@%s", cfg.ID, cfg.Version)
    
    // Extract dependencies from config (must already be canonical)
    dependencies := cfg.Dependencies

    if err := ensureCanonicalDependencies(dependencies); err != nil {
        return op.fail(
            "invalid dependency format",
            fmt.Sprintf("validating dependencies for pipeline %q", pipelineID),
            err,
            "Declare each dependency in <id>@<version> format (e.g., backend-api@1.2).",
            "pipeline_id", pipelineID,
            "dependencies", dependencies,
        )
    }
    
    // NEW: Validate dependencies (soft validation - warns on unregistered deps, errors on cycles)
    status := registry.StatusReady
    var blockedBy []string
    
    if len(dependencies) > 0 {
        // Check for cycles
        if err := op.registry.ValidateDependencyCycles(pipelineID, dependencies); err != nil {
            return op.fail(
                "circular dependency detected",
                fmt.Sprintf("validating dependencies for pipeline %q", pipelineID),
                err,
                "Remove circular references from dependency chain.",
                "pipeline_id", pipelineID,
                "dependencies", dependencies,
            )
        }
        
        // Check for unregistered dependencies (warning only)
        unregistered := op.registry.FindUnregisteredDependencies(dependencies)
        if len(unregistered) > 0 {
            blockedBy = unregistered
            status = registry.StatusBlocked
            fmt.Fprintf(os.Stderr, "⚠ Warning: unregistered dependencies: %v\n", unregistered)
            fmt.Fprintf(os.Stderr, "  Pipeline will be marked as Blocked until dependencies are registered.\n")
        }
    }
    
    // Create pipeline with auto-imported metadata
    pipeline := registry.Pipeline{
        ID:           pipelineID,
        Name:         cfg.Name,
        Path:         op.absPath,
        Description:  cfg.Description,
        RegisteredAt: time.Now(),
        Dependencies: dependencies,
        Status:       status,
        BlockedBy:    blockedBy,
    }

    if err := op.registry.Add(pipeline); err != nil {
        return op.fail(/* ... */)
    }

    updated, err := op.registry.Get(pipelineID)
    if err != nil {
        return op.fail(/* ... */)
    }

    // Success message
    fmt.Printf("✓ Pipeline %q registered\n", updated.ID)
    if updated.Status == registry.StatusBlocked {
        fmt.Printf("  Status: 🟠 Blocked (unregistered: %v)\n", updated.BlockedBy)
    } else {
        fmt.Printf("  Status: 🟢 Ready\n")
    }

    if unblocked := op.registry.ReconcileStatuses(op.statusCache); len(unblocked) > 0 {
        fmt.Printf("  ✓ Unblocked pipelines: %v\n", unblocked)
    }
    
    return nil
}

func ensureCanonicalDependencies(deps []string) error {
    for _, dep := range deps {
        parts := strings.Split(dep, "@")
        if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
            return fmt.Errorf("dependency %q must include both id and version", dep)
        }
    }
    return nil
}
```

**Tests**: `cmd/streamy/add_test.go`
- Test adding pipeline with dependencies in YAML (each entry already `<id>@<version>`)
- Test that configs using bare dependency IDs are rejected with a clear error
- Test adding pipeline with unregistered dependency (status: Blocked, warning shown)
- Test adding pipeline that unblocks another pipeline's Blocked status
- Test adding pipeline that creates cycle (should error)
- Test rejecting configs without `id` field
- Test rejecting configs without `version` field

---

#### Step 1.5: Add Registry List --tree Command

**File**: `cmd/streamy/list.go`

- Introduce `--tree` flag on `streamy registry list` to render dependency hierarchies.
- Build roots by scanning pipelines with no dependents (fallback: iterate everything when fully connected).
- Create a small `treeContext` helper that tracks visited nodes to avoid cycles and formats Unicode/ASCII branches.
- Reset the `visitedPath` map before rendering each root so separate trees expand independently.
- Display each node as `<icon> <pipeline-id> (Status: reason)` where the icon maps to `PipelineStatus` (🟢, 🟠, 🔴, etc.).
- Surface missing dependencies inline by marking them as blocked with reason "not registered".

**Tests**: `cmd/streamy/registry_test.go`
- Test setting dependencies on existing pipeline
- Test clearing dependencies (empty list)
- Test validation errors

---

### Phase 2: Dependency Graph Logic (Core Algorithm)

**Goal**: Implement DAG validation, cycle detection, and topological sorting

#### Step 2.1: Create Dependency Graph Package

**File**: `internal/registry/graph.go` (new)

```go
package registry

import (
    "errors"
    "fmt"
    "sort"
)

var (
    ErrCircularDependency = errors.New("circular dependency")
    ErrMissingDependency  = errors.New("missing dependency")
    ErrSelfDependency     = errors.New("self dependency")
)

// DependencyGraph represents a directed acyclic graph of pipeline dependencies
type DependencyGraph struct {
    edges      map[string][]string // pipelineID → dependencies
    dependents map[string][]string // pipelineID → dependents (reverse edges)
    nodes      map[string]bool     // all nodes in graph
}

// NewDependencyGraph builds a graph starting from a root pipeline
func NewDependencyGraph(registry *Registry, rootPipelineID string) (*DependencyGraph, error) {
    root, err := registry.GetPipeline(rootPipelineID)
    if err != nil {
        return nil, fmt.Errorf("root pipeline: %w", err)
    }
    
    g := &DependencyGraph{
        edges:      make(map[string][]string),
        dependents: make(map[string][]string),
        nodes:      make(map[string]bool),
    }
    
    // Build graph via DFS
    if err := g.buildFromRoot(registry, rootPipelineID, make(map[string]bool)); err != nil {
        return nil, err
    }
    
    return g, nil
}

func (g *DependencyGraph) buildFromRoot(registry *Registry, pipelineID string, visiting map[string]bool) error {
    if g.nodes[pipelineID] {
        return nil // Already processed
    }
    
    if visiting[pipelineID] {
        return fmt.Errorf("%w: cycle detected involving %s", ErrCircularDependency, pipelineID)
    }
    
    visiting[pipelineID] = true
    defer func() { delete(visiting, pipelineID) }()
    
    pipeline, err := registry.GetPipeline(pipelineID)
    if err != nil {
        return fmt.Errorf("%w: %s", ErrMissingDependency, pipelineID)
    }
    
    g.nodes[pipelineID] = true
    g.edges[pipelineID] = append([]string{}, pipeline.Dependencies...)
    
    for _, depID := range pipeline.Dependencies {
        // Add reverse edge
        g.dependents[depID] = append(g.dependents[depID], pipelineID)
        
        // Recursively build
        if err := g.buildFromRoot(registry, depID, visiting); err != nil {
            return err
        }
    }
    
    return nil
}

// TopologicalSort returns pipelines grouped by dependency level using Kahn's algorithm
func (g *DependencyGraph) TopologicalSort() ([][]string, error) {
    inDegree := make(map[string]int)
    for node := range g.nodes {
        inDegree[node] = len(g.edges[node])
    }
    
    var levels [][]string
    processed := 0
    
    for len(inDegree) > 0 {
        // Find nodes with zero in-degree
        var level []string
        for node, degree := range inDegree {
            if degree == 0 {
                level = append(level, node)
            }
        }
        
        if len(level) == 0 {
            return nil, fmt.Errorf("%w: unable to find next level", ErrCircularDependency)
        }
        
        // Sort for deterministic ordering
        sort.Strings(level)
        levels = append(levels, level)
        
        // Remove processed nodes and update in-degrees
        for _, node := range level {
            delete(inDegree, node)
            processed++
            
            for _, dependent := range g.dependents[node] {
                if _, exists := inDegree[dependent]; exists {
                    inDegree[dependent]--
                }
            }
        }
    }
    
    if processed != len(g.nodes) {
        return nil, fmt.Errorf("%w: graph has cycle", ErrCircularDependency)
    }
    
    return levels, nil
}

// GetTransitiveDependents returns all pipelines that transitively depend on the given pipeline
func (g *DependencyGraph) GetTransitiveDependents(pipelineID string) []string {
    visited := make(map[string]bool)
    g.dfsReverse(pipelineID, visited)
    delete(visited, pipelineID) // Remove self
    
    result := make([]string, 0, len(visited))
    for id := range visited {
        result = append(result, id)
    }
    sort.Strings(result)
    return result
}

func (g *DependencyGraph) dfsReverse(pipelineID string, visited map[string]bool) {
    if visited[pipelineID] {
        return
    }
    visited[pipelineID] = true
    
    for _, dependent := range g.dependents[pipelineID] {
        g.dfsReverse(dependent, visited)
    }
}

// DetectCycle attempts to find a cycle and returns the cycle path
func (g *DependencyGraph) DetectCycle() ([]string, error) {
    visiting := make(map[string]bool)
    visited := make(map[string]bool)
    parent := make(map[string]string)
    
    var cycle []string
    for node := range g.nodes {
        if !visited[node] {
            if path := g.dfsCycle(node, visiting, visited, parent); path != nil {
                cycle = path
                break
            }
        }
    }
    
    if cycle != nil {
        return cycle, ErrCircularDependency
    }
    return nil, nil
}

func (g *DependencyGraph) dfsCycle(node string, visiting, visited map[string]bool, parent map[string]string) []string {
    visiting[node] = true
    
    for _, dep := range g.edges[node] {
        if !visited[dep] {
            parent[dep] = node
            if visiting[dep] {
                // Cycle found, reconstruct path
                var cycle []string
                current := dep
                for current != "" {
                    cycle = append([]string{current}, cycle...)
                    if current == dep && len(cycle) > 1 {
                        break // Completed cycle
                    }
                    current = parent[current]
                }
                cycle = append(cycle, dep) // Close the cycle
                return cycle
            }
            if path := g.dfsCycle(dep, visiting, visited, parent); path != nil {
                return path
            }
        }
    }
    
    visiting[node] = false
    visited[node] = true
    return nil
}
```

**Tests**: `internal/registry/graph_test.go`
- Test simple chain (A → B → C)
- Test fan-out/fan-in (A → B/C, B/C → D)
- Test cycle detection (A → B → C → A)
- Test transitive dependents
- Test topological sort with multiple levels

---

### Phase 3: Registry Validation (Safety Layer)

**Goal**: Validate dependencies at registration time

#### Step 3.1: Add Validation Methods to Registry

**File**: `internal/registry/registry.go`

```go
// ValidateDependencies checks if dependencies exist and no cycles would form
func (r *Registry) ValidateDependencies(pipelineID string, dependencies []string) error {
    // Check self-reference
    for _, depID := range dependencies {
        if depID == pipelineID {
            return fmt.Errorf("%w: pipeline %s cannot depend on itself", ErrSelfDependency, pipelineID)
        }
    }
    
    // Check dependencies exist
    for _, depID := range dependencies {
        if _, err := r.GetPipeline(depID); err != nil {
            return fmt.Errorf("%w: %s (required by %s)", ErrMissingDependency, depID, pipelineID)
        }
    }
    
    // Temporarily add/update pipeline to test for cycles
    original := r.copyPipeline(pipelineID) // Backup if exists
    r.setPipelineDependencies(pipelineID, dependencies)
    
    graph, err := NewDependencyGraph(r, pipelineID)
    if err != nil {
        r.restorePipeline(pipelineID, original) // Rollback
        return err
    }
    
    if cycle, err := graph.DetectCycle(); err != nil {
        r.restorePipeline(pipelineID, original) // Rollback
        return fmt.Errorf("%w: %v", ErrCircularDependency, cycle)
    }
    
    r.restorePipeline(pipelineID, original) // Rollback temporary change
    return nil
}

// GetDependencies returns direct dependencies
func (r *Registry) GetDependencies(pipelineID string) ([]string, error) {
    pipeline, err := r.GetPipeline(pipelineID)
    if err != nil {
        return nil, err
    }
    return append([]string{}, pipeline.Dependencies...), nil
}

// GetTransitiveDependencies returns all dependencies (direct + indirect)
func (r *Registry) GetTransitiveDependencies(pipelineID string) ([]string, error) {
    graph, err := NewDependencyGraph(r, pipelineID)
    if err != nil {
        return nil, err
    }
    
    // All nodes except root
    var deps []string
    for node := range graph.nodes {
        if node != pipelineID {
            deps = append(deps, node)
        }
    }
    sort.Strings(deps)
    return deps, nil
}

// GetDependents returns pipelines that depend on this one
func (r *Registry) GetDependents(pipelineID string) ([]string, error) {
    if _, err := r.GetPipeline(pipelineID); err != nil {
        return nil, err
    }
    
    var dependents []string
    for _, pipeline := range r.ListPipelines() {
        for _, depID := range pipeline.Dependencies {
            if depID == pipelineID {
                dependents = append(dependents, pipeline.ID)
                break
            }
        }
    }
    sort.Strings(dependents)
    return dependents, nil
}
```

**Tests**: `internal/registry/registry_test.go`
- Test validating missing dependencies
- Test validating self-reference
- Test validating circular dependencies
- Test getting dependencies (direct, transitive, reverse)

---

#### Step 3.2: Integrate Validation into Add Command

**File**: `cmd/streamy/add.go`

In `addOperation.registerPipeline` method (around line 120), add validation before registration:

```go
func (op *addOperation) registerPipeline() error {
    // ... existing config loading and ID building (pipelineID = fmt.Sprintf("%s@%s", cfg.ID, cfg.Version)) ...
    
    // NEW: Validate dependencies if present
    if len(dependencies) > 0 {
        if err := op.registry.ValidateDependencies(pipelineID, dependencies); err != nil {
            return op.fail(
                "invalid dependencies",
                fmt.Sprintf("validating dependencies for pipeline %q", pipelineID),
                err,
                "Ensure all dependency pipeline IDs exist in the registry and no cycles are formed.",
                "pipeline_id", pipelineID,
                "dependencies", dependencies,
            )
        }
    }
    
    // ... existing registration logic ...
}
```

**Tests**: `cmd/streamy/add_test.go`
- Test adding pipeline with valid dependencies
- Test adding pipeline with missing dependency (should fail)
- Test adding pipeline that creates cycle (should fail)

---

### Phase 4: Orchestration Use Case (Business Logic)

**Goal**: Implement multi-pipeline execution with dependency resolution

#### Step 4.1: Create Orchestration Domain Types

**File**: `internal/domain/pipeline/orchestration.go` (new)

```go
package pipeline

import (
    "fmt"
    "time"
)

// OrchestrationPlan represents the execution plan
type OrchestrationPlan struct {
    RootPipelineID string
    TotalPipelines int
    ExecutionLevels [][]string
    StartTime      time.Time
}

func (p *OrchestrationPlan) Format() string {
    out := fmt.Sprintf("Orchestration Plan for %s\n", p.RootPipelineID)
    for i, level := range p.ExecutionLevels {
        out += fmt.Sprintf("Level %d: %v\n", i, level)
    }
    out += fmt.Sprintf("Total: %d pipelines\n", p.TotalPipelines)
    return out
}

// PipelineExecutionSummary is domain-level execution result (not infrastructure type)
type PipelineExecutionSummary struct {
    PipelineID  string
    Status      string  // "ready" or "blocked"
    Success     bool
    BlockedBy   string
    Summary     string
    Duration    time.Duration
}

// OrchestrationResult captures execution outcome (domain layer)
type OrchestrationResult struct {
    Plan            *OrchestrationPlan
    StartTime       time.Time
    EndTime         time.Time
    PipelineResults map[string]*PipelineExecutionSummary  // Domain summaries, not registry types
    ExecutedCount   int
    ReadyCount      int
    BlockedCount    int
    OverallSuccess  bool
}

func (r *OrchestrationResult) Summary() string {
    return fmt.Sprintf(
        "Orchestration complete: %d executed, %d ready, %d blocked (duration: %v)",
        r.ExecutedCount, r.ReadyCount, r.BlockedCount,
        r.EndTime.Sub(r.StartTime),
    )
}

func (r *OrchestrationResult) HasFailures() bool {
    return r.BlockedCount > 0
}

func (r *OrchestrationResult) GetPipelinesByStatus(status string) []string {
    var pipelines []string
    for id, result := range r.PipelineResults {
        if result.Status == status {
            pipelines = append(pipelines, id)
        }
    }
    return pipelines
}
```

**Architecture Note**: Domain types don't import infrastructure packages. The application layer will convert `registry.ExecutionResult` (infrastructure) to `PipelineExecutionSummary` (domain) to maintain clean architecture boundaries per `docs/architecture.md`.

**Tests**: `internal/domain/pipeline/orchestration_test.go`
- Test plan formatting
- Test result summary
- Test filtering by status

---

#### Step 4.2: Create Orchestration Application Use Case

**File**: `internal/application/orchestration/orchestrate_use_case.go` (new)

**Note**: This belongs in the **application layer**, not domain. The use case coordinates infrastructure (registry, apply) with domain types.

```go
package orchestration

import (
    "context"
    "fmt"
    "sync"
    "time"
    
    domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
    "github.com/alexisbeaulieu97/streamy/internal/registry"
    "github.com/alexisbeaulieu97/streamy/internal/application/pipeline"
)

// OrchestrateUseCase coordinates multi-pipeline execution (application layer)
type OrchestrateUseCase struct {
    registry  *registry.Registry
    applyUC   *pipeline.ApplyUseCase
    prepareUC *pipeline.PrepareUseCase
}

func NewOrchestrateUseCase(
    reg *registry.Registry,
    applyUC *pipeline.ApplyUseCase,
    prepareUC *pipeline.PrepareUseCase,
) *OrchestrateUseCase {
    return &OrchestrateUseCase{
        registry:  reg,
        applyUC:   applyUC,
        prepareUC: prepareUC,
    }
}

// CreatePlan generates execution plan without running
func (uc *OrchestrateUseCase) CreatePlan(ctx context.Context, rootPipelineID string) (*domainpipeline.OrchestrationPlan, error) {
    graph, err := registry.NewDependencyGraph(uc.registry, rootPipelineID)
    if err != nil {
        return nil, fmt.Errorf("build dependency graph: %w", err)
    }
    
    levels, err := graph.TopologicalSort()
    if err != nil {
        return nil, fmt.Errorf("topological sort: %w", err)
    }
    
    total := 0
    for _, level := range levels {
        total += len(level)
    }
    
    return &domainpipeline.OrchestrationPlan{
        RootPipelineID:  rootPipelineID,
        TotalPipelines:  total,
        ExecutionLevels: levels,
        StartTime:       time.Now(),
    }, nil
}

// Orchestrate executes pipeline with dependencies
func (uc *OrchestrateUseCase) Orchestrate(
    ctx context.Context,
    rootPipelineID string,
    dryRun bool,
) (*domainpipeline.OrchestrationResult, error) {
    startTime := time.Now()
    
    plan, err := uc.CreatePlan(ctx, rootPipelineID)
    if err != nil {
        return nil, err
    }
    
    result := &domainpipeline.OrchestrationResult{
        Plan:            plan,
        StartTime:       startTime,
        PipelineResults: make(map[string]*domainpipeline.PipelineExecutionSummary),
    }
    
    blocked := make(map[string]string) // pipelineID → blocking pipelineID
    graph, _ := registry.NewDependencyGraph(uc.registry, rootPipelineID)
    
    for _, level := range plan.ExecutionLevels {
        // Skip blocked pipelines in this level
        var toExecute []string
        for _, pipelineID := range level {
            if blockingID, isBlocked := blocked[pipelineID]; isBlocked {
                // Convert to domain summary
                result.PipelineResults[pipelineID] = &domainpipeline.PipelineExecutionSummary{
                    PipelineID: pipelineID,
                    Status:     "blocked",
                    Success:    false,
                    BlockedBy:  blockingID,
                    Summary:    fmt.Sprintf("Blocked due to upstream failure: %s", blockingID),
                }
                result.BlockedCount++
                continue
            }
            toExecute = append(toExecute, pipelineID)
        }
        
        if len(toExecute) == 0 {
            continue // All blocked, move to next level
        }
        
        // Execute pipelines in parallel, convert results to domain types
        levelResults := uc.executeLevel(ctx, toExecute, dryRun)
        
        for pipelineID, summary := range levelResults {
            result.PipelineResults[pipelineID] = summary
            result.ExecutedCount++
            
            if summary.Success {
                result.ReadyCount++
            } else {
                result.BlockedCount++
                
                // Mark transitive dependents as blocked
                dependents := graph.GetTransitiveDependents(pipelineID)
                for _, depID := range dependents {
                    blocked[depID] = pipelineID
                }
            }
        }
    }
    
    result.EndTime = time.Now()
    result.OverallSuccess = result.BlockedCount == 0
    
    return result, nil
}

// executeLevel runs pipelines in parallel and converts infrastructure results to domain summaries
func (uc *OrchestrateUseCase) executeLevel(
    ctx context.Context,
    pipelineIDs []string,
    dryRun bool,
) map[string]*domainpipeline.PipelineExecutionSummary {
    results := make(map[string]*domainpipeline.PipelineExecutionSummary)
    resultsMux := sync.Mutex{}
    
    var wg sync.WaitGroup
    for _, pipelineID := range pipelineIDs {
        wg.Add(1)
        go func(id string) {
            defer wg.Done()
            
            pipeline, _ := uc.registry.GetPipeline(id)
            startTime := time.Now()
            
            // Call existing apply use case (infrastructure layer)
            _, domainResults, _, err := uc.applyUC.Apply(ctx, pipeline.Path, dryRun)
            
            // Convert infrastructure ExecutionResult to domain PipelineExecutionSummary
            summary := &domainpipeline.PipelineExecutionSummary{
                PipelineID: id,
                Duration:   time.Since(startTime),
            }
            
            if err != nil {
                summary.Success = false
                summary.Status = "blocked"
                summary.Summary = fmt.Sprintf("Pipeline failed: %v", err)
                summary.BlockedBy = id
            } else {
                summary.Success = true
                summary.Status = "ready"
                summary.Summary = "Pipeline completed successfully"
            }
            
            resultsMux.Lock()
            results[id] = summary
            resultsMux.Unlock()
        }(pipelineID)
    }
    
    wg.Wait()
    return results
}
```

**Key Architecture Points**:
- Use case lives in `internal/application/orchestration/` (application layer)
- Imports domain types from `internal/domain/pipeline`
- Imports infrastructure from `internal/registry` and existing use cases
- Converts infrastructure results to domain summaries (maintains layer boundaries)
- No domain-to-infrastructure imports

```go
func (uc *UseCase) persistSummary(result *domainpipeline.OrchestrationResult) error {
    if uc.statusCache == nil || result == nil || result.Plan == nil {
        return nil
    }

    for pipelineID, execSummary := range result.PipelineResults {
        if execSummary == nil {
            continue
        }

        lastRun := execSummary.FinishedAt
        if lastRun.IsZero() {
            lastRun = result.FinishedAt
        }

        cached := registry.CachedStatus{
            Status:    registry.PipelineStatus(execSummary.Status),
            LastRun:   lastRun,
            Summary:   execSummary.Summary,
            BlockedBy: execSummary.BlockedBy,
        }

        if err := uc.statusCache.Set(pipelineID, cached); err != nil {
            return err
        }
    }

    order := flattenLevels(result.Plan.Levels)
    summary := registry.OrchestrationSummary{
        RootPipelineID: result.Plan.RootPipelineID,
        StartTime:      result.StartedAt,
        EndTime:        result.FinishedAt,
        TotalPipelines: len(order),
        Executed:       result.ExecutedCount,
        Blocked:        result.BlockedCount,
        Failed:         result.FailedCount,
        PipelineOrder:  order,
    }

    return uc.statusCache.AddOrchestration(summary)
}
```

**Tests**: `internal/domain/pipeline/orchestrator_test.go`
- Test orchestration with simple chain
- Test orchestration with fan-out/fan-in
- Test failure propagation (blocked status)
- Test parallel execution at same level

---

### Phase 5: CLI Commands (User Interface)

**Goal**: Expose orchestration and dependency queries via CLI

#### Step 5.1: Create the Unified Run Command

**File**: `cmd/streamy/run.go`

```go
type runOptions struct {
    ConfigPath          string
    PipelineID          string
    Force               bool
    AssumeYes           bool
    NonInteractive      bool
    Timeout             time.Duration
}

func newRunCmd(root *rootFlags, app *AppContext) *cobra.Command {
    opts := runOptions{}

    cmd := &cobra.Command{
        Use:   "run",
        Short: "Run a pipeline from a file or the registry",
        Long: strings.TrimSpace(` ... `),
        RunE: func(cmd *cobra.Command, _ []string) error {
            return executeRun(cmd, root, app, opts)
        },
    }

    cmd.Flags().StringVar(&opts.ConfigPath, "file", "", "Path to pipeline configuration file")
    cmd.Flags().StringVar(&opts.PipelineID, "registry", "", "Canonical pipeline ID (<id>@<version> or <id>@latest) to run from the registry")
    cmd.Flags().BoolVar(&opts.NonInteractive, "non-interactive", false, "Disable the interactive TUI and print results to stdout")
    cmd.Flags().BoolVar(&opts.Force, "force", false, "Execute downstream pipelines despite upstream failures (dangerous)")
    cmd.Flags().BoolVar(&opts.AssumeYes, "yes", false, "Automatically confirm prompts (required for --force)")
    cmd.Flags().DurationVar(&opts.Timeout, "timeout", 0, "Maximum duration for the run command (defaults to root timeout)")

    cmd.MarkFlagsMutuallyExclusive("file", "registry")

    return cmd
}

func executeRun(cmd *cobra.Command, root *rootFlags, app *AppContext, opts runOptions) error {
    if err := validateRunOptions(opts); err != nil {
        return err
    }

    timeout := opts.Timeout
    if timeout <= 0 {
        timeout = root.timeout
    }

    nonInteractive := opts.NonInteractive || !term.IsTerminal(int(os.Stdout.Fd()))
    ctx, _ := app.CommandContext(cmd, "command.run")

    if opts.ConfigPath != "" {
        applyOpts := applyOptions{ ... }
        // validate + run single config
        return runApply(ctx, app, applyOpts, app.LoggerFor("command.apply"))
    }

    orchestrateOpts := orchestrateOptions{ ... }
    // validate + run registry graph
    return runOrchestrate(ctx, app, orchestrateOpts, app.LoggerFor("command.orchestrate"), cmd.OutOrStdout())
}
```

Key behaviours:

- Exactly one of `--file` or `--registry` must be provided. `validateRunOptions` rejects missing/duplicate values and enforces `--yes` to accompany `--force` in registry mode.
- File runs reuse the existing apply flow (TUI + status output) without touching the registry.
- Registry runs delegate to the orchestration use case, updating the status cache, printing the dependency plan when `--dry-run` is active, and enforcing confirmation prompts for forced execution.

**Tests**: `cmd/streamy/run_test.go`
- Validates flag combinations (missing operands, conflicting flags, misuse of `--force`/`--yes`).

##### Supporting Functions

- `runApply(ctx context.Context, app *AppContext, opts applyOptions, logger ports.Logger) error`
  - Reuses the existing single-pipeline apply flow.
  - Selects TUI vs plain output based on terminal capabilities.
  - Surfaces validation or execution errors directly to the caller.
- `runOrchestrate(ctx context.Context, app *AppContext, opts orchestrateOptions, logger ports.Logger, out io.Writer) error`
  - Invokes the orchestration use case, wiring in registry + status cache.
  - Supports `--dry-run` to print the execution plan without running pipelines.
  - Updates the status cache with execution summaries and honours `--force` (with confirmation prompts and `--yes` for automation).

---

#### Step 5.2: Add Registry Dependency Commands

**File**: `cmd/streamy/registry.go`

Add new subcommands to registry command group:

```go
func newRegistryCmd(root *rootFlags, app *AppContext) *cobra.Command {
    // ... existing registry command setup ...
    
    cmd.AddCommand(newRegistryDepsCmd(root, app))      // NEW
    cmd.AddCommand(newRegistryGraphCmd(root, app))     // NEW (future enhancement)
    
    // Modify existing list command to support --dependencies flag
    
    return cmd
}

func newRegistryDepsCmd(root *rootFlags, app *AppContext) *cobra.Command {
    var transitive, reverse bool
    var format string
    
    cmd := &cobra.Command{
        Use:   "deps <pipeline-id>",
        Short: "Show dependencies of a pipeline",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            pipelineID := args[0]
            cmdCtx, logger := app.CommandContext(cmd, "command.registry.deps")
            
            var deps []string
            var err error
            
            if reverse {
                deps, err = app.Registry.GetDependents(pipelineID)
            } else if transitive {
                deps, err = app.Registry.GetTransitiveDependencies(pipelineID)
            } else {
                deps, err = app.Registry.GetDependencies(pipelineID)
            }
            
            if err != nil {
                return newCommandError("registry deps", "querying dependencies", err,
                    "Ensure the pipeline ID exists in the registry.")
            }
            
            // Format output
            if format == "json" {
                // JSON output for scripting
                // ... JSON marshal and print ...
            } else {
                // Human-readable text
                prefix := "Dependencies"
                if reverse {
                    prefix = "Dependents"
                }
                if transitive {
                    prefix += " (transitive)"
                }
                fmt.Printf("%s of %s:\n", prefix, pipelineID)
                for _, dep := range deps {
                    fmt.Printf("  - %s\n", dep)
                }
            }
            
            return nil
        },
    }
    
    cmd.Flags().BoolVar(&transitive, "transitive", false, "Include transitive dependencies")
    cmd.Flags().BoolVar(&reverse, "reverse", false, "Show dependents instead of dependencies")
    cmd.Flags().StringVar(&format, "format", "text", "Output format (text|json)")
    
    return cmd
}
```

**Tests**: `cmd/streamy/registry_test.go`
- Test deps command with various flags
- Test JSON output format
- Test error handling for missing pipelines

---

### Phase 6: Integration and Testing

**Goal**: End-to-end testing and documentation

#### Step 6.1: Integration Tests

**File**: `tests/integration_dependency_test.go` (new)

**Note**: All test helpers should operate on canonical IDs (e.g., `"pipeline-a@1.0"`). Provide constants in tests to avoid typos and keep assertions consistent.

```go
package tests

import (
    "context"
    "testing"
    "time"
    
    "github.com/alexisbeaulieu97/streamy/internal/registry"
    "github.com/stretchr/testify/require"
)

func TestOrchestrationSimpleChain(t *testing.T) {
    h := newHarness(t)
    defer h.Cleanup()
    
    // Register pipelines: A → B → C
    h.RegisterPipeline("pipeline-a", withDependencies())
    h.RegisterPipeline("pipeline-b", withDependencies("pipeline-a"))
    h.RegisterPipeline("pipeline-c", withDependencies("pipeline-b"))
    
    // Orchestrate from C
    result := h.Orchestrate("pipeline-c", false)
    require.True(t, result.OverallSuccess)
    require.Equal(t, 3, result.ExecutedCount)
    
    // Verify execution order
    require.Contains(t, result.Plan.ExecutionLevels[0], "pipeline-a")
    require.Contains(t, result.Plan.ExecutionLevels[1], "pipeline-b")
    require.Contains(t, result.Plan.ExecutionLevels[2], "pipeline-c")
}

func TestOrchestrationFanOutFanIn(t *testing.T) {
    h := newHarness(t)
    defer h.Cleanup()
    
    // Register pipelines: A → B/C, B/C → D
    h.RegisterPipeline("pipeline-a@1.0", withDependencies())
    h.RegisterPipeline("pipeline-b@1.0", withDependencies("pipeline-a@1.0"))
    h.RegisterPipeline("pipeline-c@1.0", withDependencies("pipeline-a@1.0"))
    h.RegisterPipeline("pipeline-d@1.0", withDependencies("pipeline-b@1.0", "pipeline-c@1.0"))
    
    result := h.Orchestrate("pipeline-d", false)
    require.True(t, result.OverallSuccess)
    
    // B and C should be at same level (parallel execution)
    level1 := result.Plan.ExecutionLevels[1]
    require.Contains(t, level1, "pipeline-b@1.0")
    require.Contains(t, level1, "pipeline-c@1.0")
}

func TestOrchestrationFailurePropagation(t *testing.T) {
    h := newHarness(t)
    defer h.Cleanup()
    
    // Register pipelines: A → B → C, force B to fail
    h.RegisterPipeline("pipeline-a@1.0", withDependencies())
    h.RegisterPipeline("pipeline-b@1.0", withDependencies("pipeline-a@1.0"), withForceFailure())
    h.RegisterPipeline("pipeline-c@1.0", withDependencies("pipeline-b@1.0"))
    
    result := h.Orchestrate("pipeline-c", false)
    require.False(t, result.OverallSuccess)
    
    // A should succeed, B should fail (blocked), C should be blocked
    require.Equal(t, registry.StatusReady, result.PipelineResults["pipeline-a@1.0"].Status)
    require.Equal(t, registry.StatusBlocked, result.PipelineResults["pipeline-b@1.0"].Status)
    require.Equal(t, registry.StatusBlocked, result.PipelineResults["pipeline-c@1.0"].Status)
    require.Equal(t, "pipeline-b@1.0", result.PipelineResults["pipeline-c@1.0"].BlockedBy)
}

func TestCircularDependencyRejection(t *testing.T) {
    h := newHarness(t)
    defer h.Cleanup()
    
    h.RegisterPipeline("pipeline-a@1.0", withDependencies())
    h.RegisterPipeline("pipeline-b@1.0", withDependencies("pipeline-a@1.0"))
    
    // Try to create cycle: A → B, B → C, C → A
    err := h.RegisterPipelineExpectError("pipeline-c@1.0", withDependencies("pipeline-b@1.0"))
    require.NoError(t, err) // C → B is fine
    
    // Now try to make A depend on C (creates cycle)
    err = h.UpdatePipelineDependencies("pipeline-a@1.0", []string{"pipeline-c@1.0"})
    require.ErrorIs(t, err, registry.ErrCircularDependency)
}
```

**Harness helpers** in `tests/harness.go`:

```go
type pipelineConfig struct {
    pipeline     registry.Pipeline
    forceFailure bool
}

type pipelineOption func(*pipelineConfig)

func withDependencies(deps ...string) pipelineOption {
    return func(cfg *pipelineConfig) {
        cfg.pipeline.Dependencies = append([]string(nil), deps...)
    }
}

func withForceFailure() pipelineOption {
    return func(cfg *pipelineConfig) {
        cfg.forceFailure = true
    }
}

type testHarness struct {
    t        *testing.T
    registry *registry.Registry
    status   *registry.StatusCache
    executor *mockExecutor
    useCase  *orchestration.UseCase
    tempDir  string
}

func (h *testHarness) RegisterPipeline(id string, opts ...pipelineOption) {
    cfg := pipelineConfig{pipeline: registry.Pipeline{ID: id, RegisteredAt: time.Now().UTC()}}
    for _, opt := range opts {
        opt(&cfg)
    }

    require.NoError(h.t, h.registry.Add(cfg.pipeline))

    if cfg.forceFailure {
        h.executor.failures[cfg.pipeline.Path] = errors.New("forced failure")
    }
}

func (h *testHarness) Orchestrate(pipelineID string, dryRun bool) *domainpipeline.OrchestrationResult {
    result, err := h.useCase.Execute(context.Background(), pipelineID, dryRun, false)
    require.NoError(h.t, err)
    return result
}
```

The harness owns a temporary registry + status cache, injects a mock executor that honours the `force_failure` hint, and exposes helpers to register pipelines and run orchestration scenarios concisely.

---

#### Step 6.2: Documentation

**File**: `docs/pipeline-dependencies.md` (new)

Create comprehensive user documentation covering:
- Overview and motivation
- How to declare dependencies in YAML
- Using the `run` command with `--registry`
- Querying dependencies with `registry deps`
- Understanding blocked status
- Troubleshooting cycles and missing dependencies
- Examples (simple chain, fan-out/fan-in, multi-tier app)

**File**: `docs/migration-guide.md` (new)

Create migration guide for existing users:
- What changed (registry v2.0 format)
- Automatic migration process
- Backup file location
- How to rollback if needed
- Compatibility notes

---

#### Step 6.3: Examples

**Directory**: `examples/dependencies/`

Create example pipeline configurations. **Note**: Dependencies are declared directly in pipeline YAML files via the `dependencies` field and auto-imported during `streamy add`.

**frontend-deployment.yaml**:
```yaml
id: frontend-deploy
version: "1.0"
name: "Frontend Deployment"
description: "Deploy frontend application"
dependencies:
  - backend-api@1.0

steps:
  - id: deploy_frontend
    type: command
    command: echo "Deploying frontend"
```

Register with dependencies:
```bash
streamy add examples/dependencies/backend-api.yaml
streamy add examples/dependencies/frontend-deployment.yaml
# Dependencies auto-imported from YAML
```

**integration-tests.yaml**:
```yaml
id: integration-tests
version: "1.0"
name: "Integration Tests"
description: "Run end-to-end integration tests"
dependencies:
  - api-server@1.0
  - database@2.1

steps:
  - id: run_tests
    type: command
    command: ./run-integration-tests.sh
```

Register with multiple dependencies:
```bash
streamy add examples/dependencies/api-server.yaml
streamy add examples/dependencies/database.yaml
streamy add examples/dependencies/integration-tests.yaml
# Registry stores as: integration-tests@1.0 depends on [api-server@1.0, database@2.1]
```

**production-deployment.yaml**:
```yaml
id: prod-deploy
version: "1.0"
name: "Production Deployment"
description: "Deploy complete production stack"
dependencies:
  - integration-tests@1.0

steps:
  - id: finalize_deployment
    type: command
    command: ./finalize-prod.sh
```

**storage.yaml**, **database.yaml**, **app.yaml**, **web.yaml** (similar structure with dependencies field):
```yaml
id: database-tier
version: "1.0"
name: "Database Tier"
dependencies:
  - storage-tier@1.0

steps:
  - id: setup_db
    type: command
    command: ./setup-database.sh
```

Register with dependency chain:
```bash
# Register base tiers (dependencies auto-imported from YAML)
streamy add storage.yaml
streamy add database.yaml
streamy add app.yaml
streamy add web.yaml
streamy add production-deployment.yaml
```

**Why this approach**: Dependencies are source-controlled alongside pipeline definitions, creating a single source of truth. The registry mirrors these relationships at runtime.

## Orchestrated Execution Walkthrough

1. Ensure each pipeline has been added to the registry using `streamy add <config.yaml>`.
2. Preview the orchestration plan without executing any steps:

   ```bash
   streamy run --registry production-deployment@1.0 --dry-run
   ```

   This prints the dependency levels in execution order so teams can review the rollout before running it.

3. Execute the full graph:

   ```bash
   streamy run --registry production-deployment@1.0
   ```

   Streamy resolves dependencies with Kahn’s algorithm, executes each level in parallel, and records a summary in the status cache. For CI or scripting environments, add `--non-interactive` to print a textual summary instead of launching the TUI.

4. Inspect historical results:

   ```bash
   streamy registry list --tree
   streamy registry list --json | jq '.pipelines[] | {id, status}'
   ```

   Or read the orchestration history from `~/.streamy/status-cache.json` (field `orchestrations`).

---

## Testing Strategy

### Unit Tests
- All new types (graph, orchestrator, errors)
- Validation methods (registry)
- Status handling (blocked state)

### Integration Tests
- End-to-end orchestration flows
- Failure propagation
- Cycle detection
- Migration logic

### Manual Testing Checklist
- [ ] Register pipelines with dependencies
- [ ] Orchestrate simple chain
- [ ] Orchestrate fan-out/fan-in
- [ ] Force failure and verify blocked status
- [ ] Query dependencies (direct, transitive, reverse)
- [ ] Dry-run orchestration shows plan
- [ ] Migration from v1.0 registry works
- [ ] TUI displays orchestration progress
- [ ] Status cache persists results

---

## Rollout Plan

### Phase 1: Internal Testing
- Implement core features (schema, graph, orchestrator)
- Run integration tests
- Test migration with sample registries

### Phase 2: Documentation and Examples
- Write user documentation
- Create example configurations
- Update CHANGELOG

### Phase 3: Release
- Merge to main branch
- Tag release (v0.x.y)
- Announce feature in release notes

---

## Architecture Refresher

For contributors unfamiliar with Streamy's layering:

- **Config Parsing**: `internal/infrastructure/config/schema.go` (line ~40) defines YAML structure
- **Registry Storage**: `internal/registry/` handles file I/O and in-memory state
- **Domain Logic**: `internal/domain/pipeline/` contains use cases (business rules)
- **CLI Orchestration**: `cmd/streamy/` wires commands to use cases
- **Ports/Interfaces**: `internal/ports/` defines abstractions for cross-layer communication

**Key principle**: Dependencies flow inward. Domain logic never imports from `cmd/` or `infrastructure/`.

---

## Troubleshooting

### Common Issues

**"Missing dependency" error**:
- Ensure all referenced pipelines are registered first
- Check for typos in pipeline IDs
- Use `streamy registry list` to see available pipelines

**"Circular dependency" error**:
- Draw dependency graph on paper to visualize cycle
- Use `streamy registry deps --transitive` to see full graph
- Remove one dependency edge to break the cycle

**Blocked pipelines not showing correct upstream**:
- Check `BlockedBy` field in execution result
- Verify transitive dependent calculation in graph

**Migration fails**:
- Check backup file was created
- Verify registry file has write permissions
- Check logs for detailed error messages

---

## Next Steps (Future Enhancements)

- **Dependency Visualization**: `streamy registry graph --output dot` for GraphViz
- **Conditional Dependencies**: `depends_on_if: <condition>` for optional deps
- **Orchestration Resume**: Continue from failure point after fixing upstream
- **Dependency Constraints**: `required: false` for optional dependencies
- **TUI Improvements**: Real-time progress bars for each pipeline in orchestration

---

**Status**: Quickstart complete. All implementation guidance provided. Ready for Phase 2 (tasks breakdown with `/speckit.tasks`).
