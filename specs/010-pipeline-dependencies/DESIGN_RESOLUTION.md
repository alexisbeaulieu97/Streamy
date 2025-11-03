# Design Resolution: Pipeline Dependencies

**Date**: October 28, 2025  
**Status**: Final design decisions - **REVISED**

This document resolves critical design ambiguities and documents the simplified final design.

---

## Final Design Overview

### Dependencies in YAML
Dependencies declared via top-level `dependencies` field in pipeline YAML:

```yaml
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
```

This reconciliation step rebuilds the dependency-related `BlockedBy` reasons on every pass. Other blocking causes (for example, upstream execution failures recorded by the orchestration cache) are managed separately and are not cleared unless their specific conditions are resolved.

### Two-State Status Model
Pipelines have only two states:
- **Ready**: All dependencies registered and resolvable (🟢)
- **Blocked**: Unregistered dependencies OR upstream failure (🟠 with reason listed)

### Auto-Import on Registration
`streamy add <config>` automatically imports `dependencies` field into registry.

---

## Issue 1: Config vs Registry Boundary

### Problem
Original design oscillated between embedded YAML fields, sidecar files, and CLI flags. Front-line implementers need a single clear pattern.

### Resolution: **Embedded YAML `dependencies` Field**

**Decision**: Dependencies declared as top-level `dependencies` field in pipeline YAML, auto-imported during `streamy add`.

**Rationale**:
- ✅ Single source of truth (no sidecar file confusion)
- ✅ Dependencies version-controlled with pipeline definition
- ✅ Simple schema extension (add one optional field)
- ✅ Clear to implement and understand

**Pipeline config**:
```yaml
id: database-setup
version: "1.0"
name: "Database Setup"
description: "Initialize database schema"
dependencies:
  - storage-tier@1.0
  - network-config@1.0
steps:
  - id: create_tables
    type: command
    command: ./init-db.sh
```

**Registry storage** (auto-populated during `streamy add`):
```go
type Pipeline struct {
    ID           string    `json:"id"`           // Canonical: <id>@<version>
    Name         string    `json:"name"`
    Path         string    `json:"path"`
    Dependencies []string  `json:"dependencies,omitempty"`  // Imported from YAML in canonical form
    Status       string    `json:"status"`       // "ready" or "blocked"
    BlockedBy    []string  `json:"blocked_by,omitempty"`    // Unresolved dep IDs or failure reason
}

> **Persistence note**: `Pipeline.BlockedBy` only persists unresolved dependency identifiers detected at registration time. Runtime failures recorded by orchestration runs are stored in the status cache (`registry.StatusCache`) via `CachedStatus.BlockedBy`; reconciliation clears the registry slice whenever dependencies are revalidated.
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
# ✓ Resolved blocked pipelines: [frontend-deploy@1.0]

# Visualize dependency tree
streamy registry list --tree
└─ 🟢 frontend-deploy@1.0 (Ready)
   ├─ 🟢 backend-api@1.0 (Ready)
   └─ 🟠 auth-service@1.0 (Blocked: not registered)  ← Computed during render
```

**Schema update** (`docs/schema.md`):
```yaml
id: "app-deploy"            # Required (no auto-generation)
version: "1.0"              # Required (enables <id>@<version> registry format)
name: "Pipeline Name"       # Required
dependencies: []            # Optional: upstream IDs in <id>@<version> format
steps: []                   # Required
```

**Registry ID format**: `<id>@<version>` (e.g., `app-deploy@1.0`)
- Enables multiple versions of same logical pipeline
- Prevents ID collisions and drift issues
- Leverages existing `version` field from pipeline configs
- Predictable, debuggable naming scheme

**Config validation**: Pipelines without explicit `id` or `version` fields are **rejected** with validation error. No auto-generation to avoid collision/drift complexity.

---

## Issue 2: Simplified Status Model

### Problem
Original design used multiple states (Healthy, Pending, Running, Failed, Blocked), creating complexity in status transitions and UI rendering.

### Resolution: **Registry focuses on Ready vs Blocked**

**Decision**: The registry still makes orchestrator decisions using a binary Ready/Blocked signal, but we preserved the richer status enum emitted by verification/apply flows so historical data and dashboards stay accurate.

**Rationale**:
- ✅ Operational logic (can we run this pipeline?) depends only on Ready vs Blocked.
- ✅ Verification/apply pipelines continue to surface nuanced states such as `verifying`, `drifted`, or `failed` without widening the registry’s decision surface.
- ✅ CLI/TUI outputs can show the extra states while orchestration keeps a simple gating rule.

**Status enum excerpt**:
```go
// internal/registry/types.go
type PipelineStatus string

const (
    StatusUnknown   PipelineStatus = "unknown"
    StatusReady     PipelineStatus = "ready"
    StatusBlocked   PipelineStatus = "blocked"
    StatusSatisfied PipelineStatus = "satisfied"
    StatusDrifted   PipelineStatus = "drifted"
    StatusFailed    PipelineStatus = "failed"
    StatusVerifying PipelineStatus = "verifying"
    StatusApplying  PipelineStatus = "applying"
)
```

**Registry schema (runtime fields)**:
```go
type Pipeline struct {
    ID           string    `json:"id"`
    Name         string    `json:"name"`
    Path         string    `json:"path"`
    Description  string    `json:"description"`
    RegisteredAt time.Time `json:"registered_at"`

    Dependencies []string `json:"dependencies,omitempty"`

    // Runtime-only execution state (not persisted in registry.json)
    Status     PipelineStatus   `json:"-"`
    BlockedBy  []string         `json:"-"`
    LastRun    time.Time        `json:"-"`
    LastResult *ExecutionResult `json:"-"`
}
```

**Status transitions**:

1. **On registration** (with missing dependencies):
```go
func (r *Registry) AddPipeline(p *Pipeline) error {
    // Check for unregistered dependencies
    unresolved := r.FindUnregisteredDependencies(p.Dependencies)
    if len(unresolved) > 0 {
        p.Status = StatusBlocked
        p.BlockedBy = unresolved
        fmt.Fprintf(os.Stderr, "⚠ Warning: unregistered dependencies: %v\n", unresolved)
    } else {
        p.Status = StatusReady
        p.BlockedBy = nil
    }
    
    r.pipelines[p.ID] = p
    
    // Reconcile: Check if this new pipeline unblocks others
    for id, existing := range r.pipelines {

        if existing.Status != StatusBlocked {
            continue
        }

        missing := r.FindUnregisteredDependencies(existing.Dependencies)
        if len(missing) == 0 {
            existing.Status = StatusReady
            existing.BlockedBy = nil
            fmt.Printf("✓ Unblocked pipeline: %s\n", existing.ID)
            r.pipelines[id] = existing
            continue
        }

        existing.BlockedBy = missing // Only unresolved dependency reasons remain
        fmt.Fprintf(os.Stderr, "⚠ %s still blocked by unregistered dependencies: %v\n", existing.ID, missing)
        r.pipelines[id] = existing
    }
    
    return r.save()
}
```

2. **During orchestrated execution** (upstream failure):
```go
func (uc *OrchestrateUseCase) Execute(rootID string) error {
    for _, level := range plan.Levels {
        for _, pipelineID := range level {
            result, err := uc.executor.Execute(pipelineID)
            if err != nil || !result.Success {
                // Mark all downstream dependents as blocked
                uc.markDownstreamAsBlocked(pipelineID, fmt.Sprintf("upstream failure: %s", pipelineID))
                return fmt.Errorf("pipeline %s failed", pipelineID)
            }
        }
    }
}
```

**Tree view rendering**:
```bash
streamy registry list --tree

└─ 🟢 frontend-deploy@1.0 (Ready)
   ├─ 🟢 backend-api@1.0 (Ready)
   └─ 🟠 auth-service@1.0 (Blocked: not registered)

└─ 🟠 broken-pipeline@1.0 (Blocked: upstream failure: database-setup@1.0)
   └─ 🟢 database-setup@1.0 (Ready)
```

**Domain types** (simplified):
```go
// internal/domain/pipeline/orchestration.go
type PipelineExecutionSummary struct {
    PipelineID  string
    Status      string  // "ready" or "blocked"
    Success     bool
    BlockedBy   string  // Reason if blocked
    Forced      bool
    ForcedBy    string
    Summary     string
    StartedAt   time.Time
    FinishedAt  time.Time
    Duration    time.Duration
}
```

------

## Issue 3: Application Layer Orchestration

### Problem
Original design placed orchestration in domain layer, creating risk of infrastructure imports violating clean architecture.

### Resolution: **Orchestration in Application Layer**

**Decision**: Orchestrator lives in `internal/application/orchestration/`, converts between infrastructure and domain types.

**Correct layering**:
```
CLI (cmd/streamy/orchestrate.go)
  ↓
Application (internal/application/orchestration/)  ← Orchestrator HERE
  ↓ uses domain types      ↓ uses infrastructure types
Domain (internal/domain/pipeline/) ← Registry (internal/registry/)
```

**Domain types remain pure** (`internal/domain/pipeline/orchestration.go`):
```go
package pipeline

type OrchestrationPlan struct {
    RootPipelineID string
    Levels         [][]string  // Execution levels (for parallelization)
    AllPipelines   []string
}

type PipelineExecutionSummary struct {
    PipelineID  string
    Status      string  // "ready" or "blocked"
    Success     bool
    BlockedBy   string
    Forced      bool
    ForcedBy    string
    Summary     string
    StartedAt   time.Time
    FinishedAt  time.Time
    Duration    time.Duration
}

type OrchestrationResult struct {
    Plan            *OrchestrationPlan
    StartedAt       time.Time
    FinishedAt      time.Time
    PipelineResults map[string]*PipelineExecutionSummary
    ExecutedCount   int
    BlockedCount    int
    ReadyCount      int
    FailedCount     int
    OverallSuccess  bool
}
```

**Application layer** (`internal/application/orchestration/usecase.go`):
```go
func (uc *UseCase) Execute(ctx context.Context, rootID string, dryRun, force bool) (*pipeline.OrchestrationResult, error) {
    start := time.Now()

    plan, graph, err := uc.buildPlan(ctx, rootID)
    if err != nil {
        return nil, err
    }

    result := &pipeline.OrchestrationResult{
        Plan:            plan,
        PipelineResults: map[string]*pipeline.PipelineExecutionSummary{},
        StartedAt:       start,
    }

    blocked := map[string]string{}
    forced := map[string]string{}

    for _, level := range plan.Levels {
        if err := ctx.Err(); err != nil {
            return nil, fmt.Errorf("context cancelled: %w", err)
        }

        runnable := collectRunnable(level, blocked, force, forced)
        if len(runnable) == 0 {
            continue
        }

        summaries := uc.executeLevel(ctx, runnable, dryRun)

        for id, summary := range summaries {
            result.PipelineResults[id] = summary
            result.ExecutedCount++

            switch {
            case summary.Success:
                result.ReadyCount++
            case summary.BlockedBy == id:
                result.FailedCount++
                for _, dep := range graph.GetTransitiveDependents(id) {
                    blocked[dep] = id
                }
            default:
                result.BlockedCount++
            }

            if forcedBy, ok := forced[id]; ok {
                summary.Forced = true
                summary.ForcedBy = forcedBy
                if summary.Success {
                    summary.BlockedBy = ""
                }
                summary.Summary = fmt.Sprintf("%s (forced despite %s failure)", summary.Summary, forcedBy)
            }
        }
    }

    result.FinishedAt = time.Now()
    result.OverallSuccess = result.FailedCount == 0

    if err := uc.persistSummary(result); err != nil {
        return nil, fmt.Errorf("persist orchestration summary: %w", err)
    }

    return result, nil
}
```

**Key principle**: The application layer owns orchestration, converts infrastructure results into domain summaries, and persists outcomes via the registry/status-cache adapters.

---

## Summary of Final Design

| Aspect | Decision |
|--------|----------|
| **Dependency Declaration** | Top-level `dependencies` field in YAML |
| **Status Model** | Two states: Ready, Blocked |
| **Auto-Import** | `streamy add` imports dependencies from YAML |
| **Unregistered Dependencies** | Allowed, marked as Blocked with warnings |
| **Cycle Detection** | Rejected at registration time with clear error |
| **Orchestration Location** | Application layer (`internal/application/orchestration/`) |
| **Tree Visualization** | `registry list --tree` shows 🟢 Ready / 🟠 Blocked |
| **Status Reconciliation** | Automatic on every `add` operation |

## Implementation Checklist

- [ ] Update `docs/schema.md` - Add `dependencies` field
- [ ] Update `internal/infrastructure/config/schema.go` - Parse `dependencies` from YAML
- [ ] Update `cmd/streamy/add.go` - Auto-import dependencies, validate cycles
- [ ] Update `internal/registry/types.go` - Two-state status enum (Ready/Blocked)
- [ ] Create `internal/application/orchestration/orchestrate_use_case.go`
- [ ] Add reconciliation logic to `registry.AddPipeline()`
- [ ] Update tree renderer to show Ready/Blocked with icons
- [ ] Define blocked-execution behaviour: CLI rejects blocked pipelines unless `--force`, update docs/tests and add confirmation UX
- [ ] Remove references to:
  - ~~Sidecar `.deps.yaml` files~~
  - ~~CLI `--depends-on` flag~~
  - ~~`registry set-deps` command~~
  - ~~StatusPending, StatusHealthy, StatusRunning, StatusFailed~~

**All design ambiguities resolved. Specification locked and ready for `/speckit.tasks`.**
