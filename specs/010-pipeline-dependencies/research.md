# Research: Pipeline Dependencies

**Feature**: 010-pipeline-dependencies  
**Date**: October 28, 2025  
**Status**: Complete

## Overview

This document consolidates research findings for implementing pipeline dependency orchestration in Streamy. All technical unknowns from the planning phase have been resolved through analysis of existing codebase patterns, industry best practices, and architectural constraints.

## Research Tasks

### 1. Dependency Graph Algorithms

**Decision**: Use topological sorting with Kahn's algorithm for DAG resolution

**Rationale**:
- Kahn's algorithm provides both cycle detection and execution ordering in O(V+E) time
- Identifies all nodes with no incoming edges, enabling parallel execution
- Simple to implement with standard Go data structures (maps and slices)
- Produces deterministic ordering when combined with stable sorting of nodes at each level
- Well-understood algorithm with clear error paths for cycle detection

**Alternatives considered**:
- DFS-based topological sort: More complex for parallel execution identification
- Tarjan's algorithm: Overkill for simple cycle detection needs
- External graph libraries: Violates "zero dependencies" principle

**Implementation approach**:
```go
// Pseudo-code for Kahn's algorithm
func TopologicalSort(graph DependencyGraph) ([][]PipelineID, error) {
    inDegree := computeInDegrees(graph)
    queue := findNodesWithZeroInDegree(inDegree)
    levels := [][]PipelineID{}
    totalNodes := len(graph.Nodes())
    processed := 0
    
    for len(queue) > 0 {
        currentLevel := queue
        queue = []PipelineID{}
        levels = append(levels, currentLevel)
        
        for _, node := range currentLevel {
            processed++
            for _, dependent := range graph.Adjacents(node) {
                inDegree[dependent]--
                if inDegree[dependent] == 0 {
                    queue = append(queue, dependent)
                }
            }
        }
    }
    
    if processed != totalNodes {
        return nil, errors.New("cycle detected")
    }
    return levels, nil
}
```

**References**:
- Existing pattern in Streamy: Step execution already handles sequential/parallel logic
- Similar to dependency resolution in package managers (npm, cargo)

---

### 2. Registry Schema Evolution Strategy

**Decision**: Add `Dependencies []string` field to `Pipeline` struct with default empty slice for backward compatibility

**Rationale**:
- Existing registry uses JSON serialization with omitempty tags
- Empty slice is semantically correct for "no dependencies"
- Version bump (1.0 → 2.0) signals schema change but maintains compatibility
- Migration helper provides one-time upgrade with backup
- Follows established pattern in registry/types.go for adding fields

**Alternatives considered**:
- Separate dependency manifest file: Splits related data, complicates queries
- Pointer to slice (*[]string): Adds nil-checking complexity without benefit
- Map structure: Overkill when only pipeline IDs are needed

**Migration strategy**:
1. Detect registry version on load
2. If version < 2.0, add empty Dependencies field to all pipelines
3. Write updated registry with version 2.0
4. Create backup file with .v1.bak extension
5. Log migration completion with backup location

**Schema changes**:
```go
// internal/registry/types.go
type Pipeline struct {
    ID           string    `json:"id"`
    Name         string    `json:"name"`
    Path         string    `json:"path"`
    Description  string    `json:"description"`
    RegisteredAt time.Time `json:"registered_at"`
    Dependencies []string  `json:"dependencies,omitempty"` // NEW: Registry pipeline IDs
    
    // Runtime state (not persisted)
    Status     PipelineStatus   `json:"-"`
    LastRun    time.Time        `json:"-"`
    LastResult *ExecutionResult `json:"-"`
}

type File struct {
    Version   string     `json:"version"` // "2.0"
    Pipelines []Pipeline `json:"pipelines"`
}
```

---

### 3. Parallel Execution Safety

**Decision**: Execute each dependency level sequentially, parallelize within each level using goroutines + sync.WaitGroup

**Rationale**:
- Existing ApplyUseCase already handles single pipeline execution safely
- Level-based parallelism ensures dependencies complete before dependents start
- No shared state between independent pipelines at same level
- Failure in one pipeline doesn't affect unrelated pipelines at same level
- Context cancellation propagates to all goroutines via shared context

**Alternatives considered**:
- Worker pool pattern: Added complexity without clear benefit for small graphs
- Full sequential execution: Misses obvious parallelism opportunities
- Reactive/channel-based: Over-engineered for deterministic DAG traversal

**Implementation pattern**:
```go
for _, level := range sortedLevels {
    var wg sync.WaitGroup
    results := make(chan ExecutionResult, len(level))
    
    for _, pipelineID := range level {
        wg.Add(1)
        go func(id string) {
            defer wg.Done()
            result := executePipeline(ctx, id)
            results <- result
        }(pipelineID)
    }
    
    wg.Wait()
    close(results)
    
    // Check for failures and mark dependents as blocked
    for result := range results {
        if !result.Success {
            markDependentsBlocked(graph, result.PipelineID)
        }
    }
}
```

**Safety guarantees**:
- Context timeout applies to entire orchestration
- Pipeline timeouts are per-pipeline (existing behavior)
- Early termination via context cancellation on critical failures
- No pipeline starts until all its dependencies complete

---

### 4. Failure Propagation Strategy

**Decision**: Add `StatusBlocked` to registry types, propagate immediately on upstream failure, store blocking reason

**Rationale**:
- Clear distinction between "failed" (this pipeline errored) and "blocked" (upstream failed)
- Enables users to identify root cause quickly
- Matches industry patterns (CI/CD systems: Jenkins, GitHub Actions)
- Status cache persists blocked state for post-execution queries
- TUI can render blocked status distinctly (orange/gray vs red)

**Alternatives considered**:
- Single "failed" status: Loses valuable diagnostic information
- "skipped" status: Ambiguous (intentional skip vs forced skip)
- Storing full failure chain: Overly complex, users need immediate cause

**Status additions**:
```go
const (
    StatusReady   PipelineStatus = "ready"    // All dependencies registered
    StatusBlocked PipelineStatus = "blocked"  // Unregistered dependencies or upstream failure
)

type ExecutionResult struct {
    PipelineID  string         `json:"pipeline_id"`
    Operation   string         `json:"operation"`
    Status      PipelineStatus `json:"status"`
    Success     bool           `json:"success"`
    Summary     string         `json:"summary"`
    BlockedBy   string         `json:"blocked_by,omitempty"` // NEW: Immediate failure cause (single pipeline ID)
    // ... existing fields
}
```

**Propagation logic**:
1. On pipeline failure, mark status as `StatusBlocked`
2. Query graph for all transitive dependents
3. Set their status to `StatusBlocked` with `BlockedBy = failedPipelineID`
4. Skip execution of blocked pipelines
5. Continue executing unrelated branches

---

### 5. YAML Configuration Schema

**⚠️ DESIGN RESOLUTION**: See [DESIGN_RESOLUTION.md](./DESIGN_RESOLUTION.md#issue-1-config-vs-registry-boundary) for complete analysis.

**Decision**: Dependencies declared via top-level `dependencies` field in pipeline YAML.

**Rationale**:
- Single source of truth (no sidecar confusion)
- Dependencies version-controlled with pipeline definition
- Simple schema extension (add one optional field)
- Auto-imported during `streamy add`

**Pipeline config with dependencies**:
```yaml
id: frontend-deploy
version: "1.0"
name: "Frontend Deployment"
description: "Deploy frontend application"
dependencies:
  - backend-api@1.0
  - auth-service@2.1
steps:
  - id: deploy
    type: command
    command: ./deploy-frontend.sh
```

**Pipeline config without dependencies**:
```yaml
id: standalone-job
version: "1.0"
name: "Storage Tier"
description: "Initialize storage layer"
steps:
  - id: setup_storage
    type: command
    command: ./setup-storage.sh
```

**Registry storage** (auto-populated during `streamy add`):
```go
type Pipeline struct {
    ID           string    `json:"id"`
    Name         string    `json:"name"`
    Path         string    `json:"path"`
    Dependencies []string  `json:"dependencies,omitempty"`  // Imported from YAML
    Status       string    `json:"status"`  // "ready" or "blocked"
    BlockedBy    []string  `json:"blocked_by,omitempty"`    // Unresolved dependency IDs
}

// Note: Registry state (`Pipeline.BlockedBy`) tracks all unresolved dependencies, whereas execution
// results (`ExecutionResult.BlockedBy`) capture only the immediate pipeline that caused a block.
```

**User workflow**:
```bash
# Create pipeline config with dependencies
cat > frontend-deploy.yaml <<EOF
id: frontend-deploy
version: "1.0"
name: "Frontend Deployment"
dependencies:
  - backend-api@1.0
  - auth-service@1.0
steps:
  - id: deploy
    type: command
    command: ./deploy-frontend.sh
EOF

# Register pipeline (auto-imports dependencies field)
streamy add frontend-deploy.yaml
# → Loads dependencies from YAML
# → Validates cycle detection
# → Checks for unregistered dependencies
# ⚠ Warning: unregistered dependencies: [backend-api@1.0, auth-service@1.0]
# ✓ Pipeline "frontend-deploy@1.0" registered (status: Blocked)

# Later, register missing dependency (triggers reconciliation)
streamy add backend-api.yaml
# ✓ Pipeline "backend-api@1.0" registered
# ✓ Unblocked pipelines: [frontend-deploy@1.0]

# Visualize tree (missing deps shown as render-time artifacts)
streamy registry list --tree
└─ 🟢 frontend-deploy@1.0 (Ready)
   ├─ 🟢 backend-api@1.0 (Ready)
   └─ 🟠 auth-service@1.0 (Blocked: not registered)  ← Computed during render
```

**Status reconciliation**:
- Registry recomputes BlockedBy on every `add` operation
- Status automatically transitions Ready ↔ Blocked as dependencies are added/removed
- Two-state model: Ready (can execute), Blocked (cannot execute, reason listed)

#### Version resolution policy

**Decision**: Keep canonical `<id>@<version>` identifiers as the only persisted representation while allowing the CLI to resolve the convenience alias `<id>@latest` at runtime.

**Rationale**:
- Prevents ambiguous imports—every stored dependency remains an explicit version.
- Gives operators an escape hatch for "always run the newest" flows without changing YAML.
- Keeps resolution deterministic by reusing the registry’s view of which versions exist.

**Behaviour**:
- Pipeline YAML and registry storage continue to require explicit versions (e.g. `app@1.2.3`). Inputs missing a version are rejected during validation.
- `streamy run --registry <id>@latest` (alias case-insensitive) asks the registry for the highest available version. If no versions exist, the command fails with `no registered versions found`.
- Highest-version selection prefers numeric comparison for dotted segments (`1.10` > `1.2`), falling back to lexical comparison when segments contain prerelease/build metadata; ties break on the raw string comparison.
- Alias resolution happens before orchestration begins and the resolved canonical ID is logged so users can audit which version ran. Downstream dependency checks always work against the resolved `<id>@<version>` value.
- The alias is not imported into dependencies or stored in the registry; only CLI entry points accept it so that source-controlled configuration stays explicit.

---

### 6. CLI Command Design

**Decision**: Replace the split `apply`/`orchestrate` workflow with a unified `streamy run` command that accepts mutually exclusive inputs (`--file` for single configs, `--registry` for orchestrated runs) while keeping `streamy registry` for read-only inspection.

**Rationale**:
- Single entry point lowers the learning curve—users reach for one verb regardless of source.
- Shared validation and context handling (timeouts, logging, verbosity) eliminate duplicated logic between commands.
- Makes future enhancements (e.g. dry-run, force, non-interactive) consistent across file-based and registry-based executions.
- Preserves the existing mental model for `streamy registry` as the place to inspect state without executing anything.

**Alternatives considered**:
- Keep `apply` and add a new `orchestrate` verb (rejected: increases surface area and forces users to learn two nearly identical commands).
- Add an `--with-dependencies` flag to `apply` (rejected: hard to discover and complicates help output).
- Auto-detect file vs registry arguments without flags (rejected: ambiguous when paths resemble canonical IDs and vice versa).

**Resulting UX**:
```bash
# Run a single config file (existing behaviour)
streamy run --file ./pipelines/web.yaml

# Orchestrate a registry pipeline (dependency-aware)
streamy run --registry web-stack@1.0 [--dry-run] [--force --yes] [--non-interactive]

# Inspect registry state
streamy registry list --tree
streamy registry show web-stack@1.0
```

**Command behaviour**:
- Flags `--file` and `--registry` are mutually exclusive; at least one must be provided.
- `--registry` mode invokes the orchestration use case (plan preview, dependency resolution, forced execution confirmation, status cache updates).
- `--file` mode reuses the existing apply path with no registry side effects.
- `--non-interactive` switches both modes to plain-text summaries for CI/CD.
- `--force` requires `--yes` (or interactive confirmation) and is logged alongside any forced pipeline IDs.

---

### 7. Status Cache Design

**Decision**: Extend existing `StatusCacheFile` with orchestration metadata, persist per-pipeline blocked status

**Rationale**:
- Existing status cache infrastructure in registry package
- Preserves per-pipeline granularity for dashboard/list commands
- Orchestration summary provides high-level overview
- File-based persistence matches existing pattern (no database)

**Alternatives considered**:
- Separate orchestration cache file: Duplicates pipeline status data
- In-memory only: Loses visibility after command exits
- Database: Violates "zero dependencies" principle

**Cache schema additions**:
```go
type CachedStatus struct {
    Status      PipelineStatus `json:"status"`
    LastRun     time.Time      `json:"last_run"`
    Summary     string         `json:"summary"`
    StepCount   int            `json:"step_count"`
    FailedSteps []string       `json:"failed_steps,omitempty"`
    BlockedBy   string         `json:"blocked_by,omitempty"`  // NEW
}

type OrchestrationRun struct {
    RootPipelineID string                  `json:"root_pipeline_id"`
    StartTime      time.Time               `json:"start_time"`
    EndTime        time.Time               `json:"end_time"`
    TotalPipelines int                     `json:"total_pipelines"`
    Executed       int                     `json:"executed"`
    Blocked        int                     `json:"blocked"`
    Failed         int                     `json:"failed"`
    PipelineOrder  []string                `json:"pipeline_order"`
}

type StatusCacheFile struct {
    Version       string                  `json:"version"`
    Statuses      map[string]CachedStatus `json:"statuses"`
    Orchestrations []OrchestrationRun      `json:"orchestrations,omitempty"` // NEW
}
```

---

### 8. Error Message Design

**Decision**: Structured errors with context, affected pipelines, and remediation steps

**Rationale**:
- Existing `ErrorDetail` struct provides foundation
- User-focused messages ("pipeline X depends on missing Y") not technical jargon
- Remediation suggestions guide users to fix
- Consistent with Streamy's existing error patterns

**Error categories and messages**:

**Missing dependency**:
```
Error: Missing dependency
Pipeline "app-deployment@1.0" depends on "database-setup@1.0", but it is not registered.

Suggestion: Register the missing pipeline first:
  streamy add database-setup.yaml
```

**Circular dependency**:
```
Error: Circular dependency detected
Cycle found: app → database → storage → app

Suggestion: Remove one of these dependencies to break the cycle.
Review your pipeline configurations for:
  - app-deployment.yaml
  - database-setup.yaml
  - storage-init.yaml
```

**Blocked pipeline**:
```
Pipeline "app-deployment" blocked
Upstream dependency "database-setup" failed with error:
  Connection timeout to database host

Suggestion: Fix the upstream failure and retry orchestration.
Use --force to execute despite failures (not recommended).
```

---

## Best Practices Applied

### From Graph Theory
- Kahn's algorithm for topological sorting and cycle detection
- Adjacency list representation for efficient traversal
- Level-based grouping for parallelization opportunities

### From CI/CD Systems
- Blocked status (Jenkins, GitHub Actions)
- Dependency visualization (GitLab CI, Azure DevOps)
- Dry-run mode for change preview (Terraform, Kubernetes)

### From Package Managers
- Transitive dependency resolution (npm, cargo)
- Version-agnostic dependency references (aligned with Streamy philosophy)
- Lock file pattern for reproducible builds (adapted as status cache)

### From Existing Streamy Patterns
- Domain-driven architecture (use cases, ports, domain entities)
- File-based configuration with in-memory state
- Structured logging with context
- TUI for interactive feedback, CLI for automation
- Idempotent operations with dry-run support

---

## Open Questions (Resolved)

1. **How to handle concurrent writes to status cache during parallel execution?**
   - **Answer**: Use mutex-protected cache updates or collect results and write once per level

2. **Should orchestration be resumable after failure?**
   - **Answer**: No for MVP. Status cache shows what completed; users re-run orchestration (idempotent)

3. **How to handle dynamic dependencies (resolved at runtime)?**
   - **Answer**: Out of scope. Dependencies must be declared statically in YAML

4. **Should transitive dependencies be explicit in config?**
   - **Answer**: No. Users declare direct dependencies; system resolves transitive automatically

---

## References

- Existing Streamy codebase: `internal/registry/`, `cmd/streamy/`, `internal/domain/pipeline/`
- Go concurrency patterns: https://go.dev/blog/pipelines
- Kahn's algorithm: https://en.wikipedia.org/wiki/Topological_sorting
- CI/CD dependency patterns: GitHub Actions workflow dependencies

---

**Status**: All research tasks complete. Ready for Phase 1 (Design & Contracts).
