# Data Model: Pipeline Dependencies

**Feature**: 010-pipeline-dependencies  
**Date**: October 28, 2025  
**Status**: Complete

## Overview

This document defines the data structures and their relationships for pipeline dependency orchestration. All entities follow Streamy's existing domain-driven architecture with clear separation between domain logic, infrastructure, and presentation layers.

---

## Core Entities

### 1. Pipeline (Modified)

**Location**: `internal/registry/types.go`

**Description**: Represents a registered Streamy pipeline with dependency relationships

**Fields**:
```go
type Pipeline struct {
    // Existing fields
    ID           string    `json:"id"`           // Canonical registry identifier: <id>@<version>
    Name         string    `json:"name"`         // Human-readable name
    Path         string    `json:"path"`         // Absolute path to config file
    Description  string    `json:"description"`  // User-provided description
    RegisteredAt time.Time `json:"registered_at"` // Registration timestamp
    
    // NEW: Dependency relationships
    Dependencies []string  `json:"dependencies,omitempty"` // Registry pipeline IDs in <id>@<version> format
    
    // Runtime state (not persisted in registry file)
    Status     PipelineStatus   `json:"-"`
    LastRun    time.Time        `json:"-"`
    LastResult *ExecutionResult `json:"-"`
}
```

**Validation Rules**:
- `ID`: Must be in `<id>@<version>` format (e.g., `app-deploy@1.0`), unique in registry, non-empty
- `Dependencies`: Each ID must be in `<id>@<version>` format, must exist in registry, no self-references, no cycles
- `Dependencies`: Empty array valid (no dependencies); omitted arrays are normalised to empty slices during load
- **Enforcement layers**:
  - YAML loader fails early when `id` or `version` are missing or blank.
  - Registry validation enforces canonical identifier format, uniqueness, dependency existence, and cycle checks.

**State Transitions**: None (static registry data)

**Relationships**:
- **Depends on**: Other `Pipeline` entities (many-to-many)
- **Has**: `ExecutionResult` (one-to-many via status cache)

---

### 2. DependencyGraph (New)

**Location**: `internal/registry/graph.go`

**Description**: Represents the resolved directed acyclic graph of pipeline dependencies

**Fields**:
```go
type DependencyGraph struct {
    // Adjacency list: pipelineID → list of pipelines it depends on
    edges map[string][]string
    
    // Reverse adjacency: pipelineID → list of pipelines that depend on it
    dependents map[string][]string
    
    // All pipeline IDs in the graph
    nodes map[string]bool
}
```

**Validation Rules**:
- Must be a DAG (no cycles)
- All referenced pipeline IDs must exist in registry
- Root node must be in the graph

**State Transitions**: Immutable after construction

**Relationships**:
- **Contains**: `Pipeline` entities as nodes
- **Represents**: Dependency relationships from `Pipeline.Dependencies`

**Operations**:
```go
// Topological sort with level grouping (Kahn's algorithm)
func (g *DependencyGraph) TopologicalSort() ([][]string, error)

// Direct dependencies of a pipeline
func (g *DependencyGraph) Dependencies(pipelineID string) []string

// Direct dependents of a pipeline
func (g *DependencyGraph) Dependents(pipelineID string) []string

// Find all transitive dependents of a pipeline
func (g *DependencyGraph) GetTransitiveDependents(pipelineID string) []string
```

---

### 3. PipelineStatus (Modified)

**Location**: `internal/registry/types.go`

**Description**: Enum representing the execution state of a pipeline

**Values**:
```go
type PipelineStatus string

const (
    StatusUnknown   PipelineStatus = "unknown"   // Initial state when no status has been recorded
    StatusReady     PipelineStatus = "ready"     // All dependencies registered / last run succeeded
    StatusBlocked   PipelineStatus = "blocked"   // Unregistered dependencies or upstream failure
    StatusSatisfied PipelineStatus = "satisfied" // Verification confirms desired state
    StatusDrifted   PipelineStatus = "drifted"   // Verification detected divergence
    StatusFailed    PipelineStatus = "failed"    // Execution error in last run
    StatusVerifying PipelineStatus = "verifying" // Verification currently in progress
    StatusApplying  PipelineStatus = "applying"  // Apply currently in progress
)
```

**State Transitions**:
```
ready → blocked     (when dependency becomes unregistered)
blocked → ready     (when missing dependency is registered)
ready ↔ satisfied   (based on verification results)
satisfied → drifted (when verification detects drift)
drifted → ready     (when apply reconciles drift)
ready → failed      (when execution fails)
failed → ready      (when a subsequent run succeeds)
unknown → ready     (first successful execution)
unknown → blocked   (first reconciliation finds missing dependencies)
```

**Validation Rules**:
- `StatusBlocked` typically includes `BlockedBy` to describe the cause (unregistered dependency or upstream failure); legacy entries may omit it
- Transitions occur automatically during reconciliation (on `streamy add`)

**UI Representation**:
```go
func (s PipelineStatus) Icon() string {
    switch s {
    case StatusUnknown:
        return "⚪"
    case StatusReady:
        return "🟢"
    case StatusBlocked:
        return "🟠"
    case StatusSatisfied:
        return "✅"
    case StatusDrifted:
        return "🟡"
    case StatusFailed:
        return "🔴"
    default:
        return "⚪"
    }
}

func (s PipelineStatus) Color() lipgloss.Color {
    switch s {
    case StatusUnknown:
        return lipgloss.Color("250")
    case StatusReady:
        return lipgloss.Color("34")  // green
    case StatusBlocked:
        return lipgloss.Color("208") // orange
    case StatusSatisfied:
        return lipgloss.Color("42")  // green (satisfied)
    case StatusDrifted:
        return lipgloss.Color("226") // yellow
    case StatusFailed:
        return lipgloss.Color("196") // red
    default:
        return lipgloss.Color("250") // light gray fallback
    }
}
```

---

### 4. ExecutionResult (Modified)

**Location**: `internal/registry/types.go`

**Description**: Captures the outcome of a pipeline execution (verify or apply)

**Fields**:
```go
type ExecutionResult struct {
    PipelineID  string         `json:"pipeline_id"`
    Operation   string         `json:"operation"`    // "verify" or "apply"
    Status      PipelineStatus `json:"status"`
    Success     bool           `json:"success"`
    Summary     string         `json:"summary"`
    StepCount   int            `json:"step_count"`
    FailedSteps []string       `json:"failed_steps,omitempty"`
    StepResults []StepResult   `json:"step_results"`
    Duration    time.Duration  `json:"duration"`
    CompletedAt time.Time      `json:"completed_at"`
    Error       *ErrorDetail   `json:"error,omitempty"`
    
    // NEW: Orchestration-related fields
    BlockedBy   string         `json:"blocked_by,omitempty"`  // Upstream pipeline ID that caused block
}
```

**Validation Rules**:
- Fresh orchestration results MUST populate `BlockedBy` when `Status == StatusBlocked`; legacy entries migrated from older caches may omit it.
- `Success == false` implies `Error != nil` or `BlockedBy != ""`
- `Operation` must be "verify" or "apply"

**Relationships**:
- **Belongs to**: One `Pipeline`
- **References**: Blocking `Pipeline` via `BlockedBy` (optional)

---

### 5. OrchestrationPlan (New)

**Location**: `internal/domain/pipeline/orchestration.go`

**Description**: Represents the execution plan for a dependency-aware pipeline orchestration

**Fields**:
```go
type OrchestrationPlan struct {
    RootPipelineID string      // The pipeline the user invoked
    Levels         [][]string  // Topologically sorted pipeline IDs by level
    GeneratedAt    time.Time   // Timestamp when the plan was created
}
```

**Validation Rules**:
- `RootPipelineID` must exist in the registry snapshot used to build the plan
- `Levels` must provide a valid topological ordering (all dependencies appear in earlier levels)
- The union of the IDs in `Levels` must contain `RootPipelineID`

**State Transitions**: Immutable after creation (planning phase)

**Relationships**:
- **Plans execution for**: Multiple pipeline IDs (domain layer doesn't know Pipeline struct details)
- **Derived from**: Dependency graph resolution (via application layer port)

**Operations**:
```go
// Pretty-print execution order
func (p *OrchestrationPlan) Format() string
```

---

### 6. OrchestrationResult (New)

**Location**: `internal/domain/pipeline/orchestration.go`

**Description**: Captures the outcome of a complete orchestration run (domain-level summary)

**Fields**:
```go
type ExecutionSummary struct {
    PipelineID string        // Registry pipeline ID that was evaluated
    Status     string        // "ready", "blocked", or "failed" at the end of execution (plain string to avoid importing registry enums into domain layer)
    Success    bool          // True when the pipeline executed successfully
    BlockedBy  string        // Upstream pipeline (or self) that caused block/failure
    Forced     bool          // True when run proceeded despite being blocked
    ForcedBy   string        // The blocking pipeline that was overridden
    Summary    string        // Human-readable summary for CLI/TUI display
    StartedAt  time.Time     // Timestamp when execution began (per pipeline)
    FinishedAt time.Time     // Timestamp when execution finished
    Duration   time.Duration // Convenience duration (FinishedAt - StartedAt)
}

type OrchestrationResult struct {
    Plan            *OrchestrationPlan
    PipelineResults map[string]*ExecutionSummary
    StartedAt       time.Time
    FinishedAt      time.Time
    ExecutedCount   int // Pipelines that attempted execution (success or fail)
    ReadyCount      int // Pipelines that completed successfully
    BlockedCount    int // Pipelines skipped because of blockers
    FailedCount     int // Pipelines that executed but failed
    OverallSuccess  bool
}
```

**Field Note**: Detailed step information (`StepCount`, `FailedSteps`, `StepResults`) remains in the infrastructure-level `ExecutionResult`. The orchestration domain intentionally keeps a condensed summary for CLI/TUI presentation; converters can enrich summaries later if step-level detail becomes necessary. The aggregated counters (Executed/Ready/Blocked/Failed) provide O(1) access for CLI summaries and TUI dashboards—validation rules below guarantee they stay aligned with `PipelineResults`.

**Validation Rules**:
- `ExecutedCount` must equal `ReadyCount + FailedCount`.
- `ExecutedCount + BlockedCount` must equal the total number of unique pipeline IDs in `Plan.Levels`.
- `FinishedAt` must be zero or after `StartedAt`.
- Each entry in `PipelineResults` MUST have either `Success == true` or a non-empty `BlockedBy`/failure summary.

**State Transitions**: Immutable after orchestration completes

**Relationships**:
- **Executes**: `OrchestrationPlan`
- **Contains**: Domain-level execution summaries (not infrastructure `ExecutionResult` objects)

**Layer Boundary Note**: The application layer converts infrastructure `registry.ExecutionResult` instances into domain-level `ExecutionSummary` structs to maintain clean architecture boundaries.

**Operations**:
```go
// Summarize results for logging/display
func (r *OrchestrationResult) Summary() string

// Return successful pipeline IDs (sorted)
func (r *OrchestrationResult) Executed() []string
```

---

### 7. CachedStatus (Modified)

**Location**: `internal/registry/types.go`

**Description**: Persisted status metadata for a pipeline in the status cache file

**Fields**:
```go
type CachedStatus struct {
    Status      PipelineStatus `json:"status"`
    LastRun     time.Time      `json:"last_run"`
    Summary     string         `json:"summary"`
    StepCount   int            `json:"step_count"`
    FailedSteps []string       `json:"failed_steps,omitempty"`
    
    // NEW: Orchestration-related fields
    BlockedBy   string         `json:"blocked_by,omitempty"`  // Upstream pipeline ID
}
```

**Validation Rules**:
- Serialized to `~/.config/streamy/status-cache.json` (or equivalent)
- Version bumped to indicate schema change

---

### 8. StatusCacheFile (Modified)

**Location**: `internal/registry/types.go`

**Description**: Top-level structure for the status cache JSON file

**Fields**:
```go
type StatusCacheFile struct {
    Version       string                  `json:"version"`  // "2.0"
    Statuses      map[string]CachedStatus `json:"statuses"`
    
    // NEW: Historical orchestration runs
    Orchestrations []OrchestrationSummary  `json:"orchestrations,omitempty"`
}

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
```

**Validation Rules**:
- Version must be "2.0" for orchestration support
- Orchestrations history retains the most recent 100 runs (older entries are evicted FIFO during `AddOrchestration`).

---

## Entity Relationships Diagram

```
┌─────────────────┐
│   Registry      │
│   (File)        │
└────────┬────────┘
         │ contains
         ▼
┌─────────────────┐    dependencies    ┌─────────────────┐
│   Pipeline      │◄───────────────────│   Pipeline      │
│                 │                     │                 │
│ - Dependencies[]│                     │ - Dependencies[]│
└────────┬────────┘                     └─────────────────┘
         │
         │ builds
         ▼
┌─────────────────┐      creates      ┌──────────────────────┐
│ DependencyGraph │──────────────────►│ OrchestrationPlan    │
│                 │                    │                      │
│ - edges         │                    │ - ExecutionLevels    │
│ - dependents    │                    │ - RootPipelineID     │
└────────┬────────┘                    └──────────┬───────────┘
         │                                        │
         │ validates                              │ executes
         │                                        ▼
         │                             ┌──────────────────────┐
         │                             │ OrchestrationResult  │
         │                             │                      │
         │                             │ - PipelineResults    │
         │                             └──────────┬───────────┘
         │                                        │
         │                                        │ contains
         │                                        ▼
         │                             ┌──────────────────────┐
         └─────────────────────────────│  ExecutionResult     │
                                       │                      │
                                       │  - BlockedBy         │
                                       │  - Status            │
                                       └──────────┬───────────┘
                                                  │
                                                  │ persists to
                                                  ▼
                                       ┌──────────────────────┐
                                       │  CachedStatus        │
                                       │                      │
                                       │  (StatusCacheFile)   │
                                       └──────────────────────┘
```

---

## File Format Changes

### Registry File (registry.json) - Version 2.0

**Before (v1.0)**:
```json
{
  "version": "1.0",
  "pipelines": [
    {
      "id": "app-deployment@1.0",
      "name": "Application Deployment",
      "path": "/home/user/pipelines/app.yaml",
      "description": "Deploy the application",
      "registered_at": "2025-10-28T10:00:00Z"
    }
  ]
}
```

**After (v2.0)**:
```json
{
  "version": "2.0",
  "pipelines": [
    {
      "id": "app-deployment@1.0",
      "name": "Application Deployment",
      "path": "/home/user/pipelines/app.yaml",
      "description": "Deploy the application",
      "registered_at": "2025-10-28T10:00:00Z",
      "dependencies": ["database-setup@2.1", "network-config@1.0"]
    },
    {
      "id": "database-setup@2.1",
      "name": "Database Setup",
      "path": "/home/user/pipelines/db.yaml",
      "description": "Initialize database",
      "registered_at": "2025-10-28T09:00:00Z",
      "dependencies": []
    }
  ]
}
```

### Status Cache File (status-cache.json) - Version 2.0

**Before (v1.0)**:
```json
{
  "version": "1.0",
  "statuses": {
    "app-deployment@1.0": {
      "status": "ready",
      "last_run": "2025-10-28T11:00:00Z",
      "summary": "All steps completed successfully",
      "step_count": 5
    }
  }
}
```

**After (v2.0)**:
```json
{
  "version": "2.0",
  "statuses": {
    "app-deployment@1.0": {
      "status": "ready",
      "last_run": "2025-10-28T11:00:00Z",
      "summary": "All steps completed successfully",
      "step_count": 5
    },
    "frontend-deploy@1.2": {
      "status": "blocked",
      "last_run": "2025-10-28T11:05:00Z",
      "summary": "Blocked by upstream failure",
      "step_count": 0,
      "blocked_by": "app-deployment@1.0"
    }
  },
  "orchestrations": [
    {
      "root_pipeline_id": "frontend-deploy@1.2",
      "start_time": "2025-10-28T11:00:00Z",
      "end_time": "2025-10-28T11:05:00Z",
      "total_pipelines": 3,
      "executed": 2,
      "blocked": 1,
      "failed": 1,
      "pipeline_order": ["database-setup@2.1", "app-deployment@1.0", "frontend-deploy@1.2"]
    }
  ]
}
```

---

## Migration Strategy

### Registry Migration (1.0 → 2.0)

1. **Detection**: Check `File.Version` on load
2. **Migration**: Add `Dependencies: []` to all pipelines if missing
3. **Backup**: Create `registry.json.v1.bak` before writing
4. **Write**: Save updated registry with `Version: "2.0"`
5. **Logging**: Log migration completion with backup path

**Code location**: `internal/registry/migration.go`

### Status Cache Migration (1.0 → 2.0)

1. **Detection**: Check `StatusCacheFile.Version` on load
2. **Migration**: Add `Orchestrations: []` if missing
3. **Write**: Save with `Version: "2.0"`
4. **No backup needed**: Status cache is regenerated on next run

**Code location**: `internal/registry/migration.go`

---

## Index Structures (In-Memory)

For efficient graph operations, maintain these indexes:

```go
type RegistryIndexes struct {
    // Quick lookup: pipelineID → Pipeline
    byID map[string]*Pipeline
    
    // Dependencies: pipelineID → list of dependency IDs
    dependencies map[string][]string
    
    // Reverse dependencies: pipelineID → list of dependent IDs
    dependents map[string][]string
    
    // Topological order cache (invalidated on registry change)
    topoCache map[string][][]string  // rootID → sorted levels
}
```

**Update strategy**: Rebuild indexes on registry load/modification

---

## Validation Rules Summary

| Entity              | Validation Rule                                    | Error Message                                      |
|---------------------|----------------------------------------------------|----------------------------------------------------|
| Pipeline            | Dependencies must exist in registry                | "pipeline X depends on missing pipeline Y"         |
| Pipeline            | No self-references                                 | "pipeline X cannot depend on itself"               |
| DependencyGraph     | Must be acyclic                                    | "cycle detected: A → B → C → A"                    |
| DependencyGraph     | All nodes must be in registry                      | "dependency graph references unknown pipeline X"   |
| ExecutionResult     | BlockedBy implies Status == StatusBlocked          | "inconsistent blocked state"                       |
| OrchestrationResult | ExecutedCount + BlockedCount == TotalPipelines     | "pipeline count mismatch"                          |
| File (registry)     | Version must be "1.0" or "2.0"                     | "unsupported registry version X"                   |

---

**Status**: Data model complete. Ready for contract generation (Phase 1).
