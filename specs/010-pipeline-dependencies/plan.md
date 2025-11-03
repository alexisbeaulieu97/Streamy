# Implementation Plan: Pipeline Dependencies

**Branch**: `010-pipeline-dependencies` | **Date**: October 28, 2025 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/010-pipeline-dependencies/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Enable Streamy pipelines to declare dependencies on other registry pipelines, allowing automatic orchestration of multi-pipeline workflows with proper execution ordering, failure propagation, and status visibility. The registry will validate dependency graphs at registration time, rejecting cycles and missing references. The orchestrator will resolve dependencies using topological sorting, execute independent pipelines concurrently, and propagate failures by marking downstream pipelines as "blocked."

## Technical Context

**Language/Version**: Go 1.25.1  
**Primary Dependencies**: Existing Streamy codebase (cobra CLI, bubbletea TUI, domain-driven architecture)  
**Storage**: File-based YAML configuration, in-memory registry state (existing pattern)  
**Testing**: Go test with table-driven tests, integration tests under tests/  
**Target Platform**: Linux, macOS, Windows (existing cross-platform support)  
**Project Type**: Single Go project with cmd/ for CLI and internal/ for domain logic  
**Performance Goals**: 
- Dependency validation: <5 seconds for graphs with 100 pipelines
- DAG resolution: <1 second for 50 pipelines with 10 levels of depth
- Blocked status propagation: <1 second after upstream failure
**Constraints**: 
- No persistent database (registry is file-based YAML + in-memory cache)
- Single compiled binary with no external dependencies
- Idempotent operations with dry-run support
- Backward compatible with existing registry files
**Scale/Scope**: 
- Support up to 100 pipelines in registry
- Dependency graphs up to 10 levels deep
- 50+ pipelines in a single orchestrated execution

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Onboarding First (NON-NEGOTIABLE)
✅ **PASS** - No new external dependencies required. Dependencies declared in pipeline YAML (`dependencies` field), auto-imported during `streamy add`. Backward compatible (field is optional).

### II. Schema Clarity & Fun
✅ **PASS** - Dependencies declared via top-level `dependencies` field in YAML. Auto-imported to registry during `streamy add`. Unregistered dependencies produce clear warnings and Blocked status. `registry list --tree` visualizes relationships with colored status indicators (🟢 Ready, 🟠 Blocked).

### III. Plugin-Centric Architecture
✅ **PASS** - Core responsibility: DAG resolution and orchestration logic added to application/orchestration packages. No domain-specific logic in core execution engine.

### IV. Safety by Default (NON-NEGOTIABLE)
✅ **PASS** - Cycle detection at registration time (fail fast). Unregistered dependencies allowed with Blocked status (warning, not error). Dry-run mode extends to orchestrated execution. Blocked status prevents partial failures from propagating.

### V. Performance & Reliability
✅ **PASS** - Parallel execution of independent pipelines maintained. Clear error messages for cycles, missing dependencies, and blocked states. Dry-run completes in <1s for typical graphs.

### VI. Extensibility & Composability
✅ **PASS** - Feature is additive: existing pipelines continue working without dependencies field. New field enables composition of complex workflows from simple pipelines.

### VII. Ecosystem Consistency
✅ **PASS** - Follows existing naming conventions (`dependencies` field for pipeline relationships). Structured error handling with context, remediation guidance, and clear messages.

**Gate Result**: ✅ ALL GATES PASS - No constitutional violations

## Project Structure

### Documentation (this feature)

```
specs/010-pipeline-dependencies/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── DESIGN_RESOLUTION.md # Phase 1 output (ambiguity resolutions)
├── contracts/           # Phase 1 output (/speckit.plan command)
│   └── registry-api.md  # Registry dependency operations
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```
cmd/streamy/
├── add.go               # [MODIFY] Import dependencies from config YAML at line ~47
├── registry.go          # [MODIFY] Add list --tree subcommand at line ~18
├── apply.go             # [MODIFY] Orchestration entry point at line ~63
└── orchestrate.go       # [NEW] Orchestration command and logic

internal/
├── domain/
│   └── pipeline/
│       └── orchestration.go      # [NEW] OrchestrationPlan, OrchestrationResult, PipelineExecutionSummary
├── application/
│   └── orchestration/
│       └── orchestrate_use_case.go  # [NEW] Orchestration use case with domain/infrastructure conversion
├── registry/
│   ├── types.go                  # [MODIFY] Add Dependencies []string at line ~10, StatusBlocked at line ~27
│   ├── registry.go               # [MODIFY] Add dependency validation methods
│   ├── graph.go                  # [NEW] DependencyGraph, topological sort (Kahn's algorithm)
│   └── migration.go              # [NEW] Registry v2.0 migration (v1.0 → v2.0 with Dependencies field)
└── infrastructure/
    └── engine/
        └── executor.go           # [MODIFY] Add blocked status handling

tests/
├── integration_dependency_test.go   # [NEW] End-to-end dependency tests
└── harness.go                       # [MODIFY] Add dependency test helpers

docs/
├── pipeline-dependencies.md         # [NEW] Feature documentation
└── migration-guide.md               # [NEW] Registry migration instructions

examples/
└── dependencies/                    # [NEW] Example dependency configs
    ├── simple-chain.yaml
    ├── fan-out-fan-in.yaml
    └── multi-tier.yaml
```

**Structure Decision**: Single Go project following existing Streamy architecture. Domain logic in `internal/domain/pipeline/`, infrastructure concerns in `internal/registry/` and `internal/infrastructure/config/`, CLI wiring in `cmd/streamy/`. This maintains the clean architecture boundaries already established (see ADR 002).

## Complexity Tracking

*No constitutional violations - this section is empty.*

---

## Phase 0: Research ✅ COMPLETE

**Output**: [research.md](./research.md)

All technical unknowns resolved:
- Dependency graph algorithm: Kahn's topological sort
- Registry schema evolution: Add Dependencies field with migration
- Parallel execution safety: Level-based concurrency with goroutines
- Failure propagation: StatusBlocked with transitive marking
- YAML schema: `dependencies` array field
- CLI design: `orchestrate` command + `registry deps` queries
- Status cache: Extended with orchestration summaries
- Error messaging: Structured errors with context and remediation

---

## Phase 1: Design & Contracts ✅ COMPLETE

**Outputs**:
- [data-model.md](./data-model.md) - Complete entity definitions and relationships
- [contracts/registry-api.md](./contracts/registry-api.md) - Internal Go API contracts
- [quickstart.md](./quickstart.md) - Implementation guide for developers

**Key Design Decisions**:
1. **No versioning constraints**: Dependencies point to concrete registry IDs in `<id>@<version>` format
2. **Mandatory ID and version**: Pipeline configs MUST include explicit `id` and `version` fields (no auto-generation to prevent collisions/drift)
3. **Canonical registry IDs**: Stored as `<id>@<version>` (e.g., `app-deploy@1.0`) enabling multi-version coexistence
4. **Snapshot semantics**: Orchestration uses registry state at start
5. **Fail-fast validation**: All checks at registration time, not runtime
6. **Level-based parallelism**: Independent pipelines run concurrently
7. **Blocked status propagation**: Clear distinction from failed status

**Constitution Re-check**: ✅ ALL GATES STILL PASS
- Backward compatible schema migration maintains onboarding simplicity
- Clear, minimal YAML schema with actionable error messages
- Clean separation between domain logic and infrastructure
- Safety enforced through validation and blocked status
- Performance targets achievable with efficient algorithms
- Composable design allows gradual adoption

---

## Architectural Corrections ✅ COMPLETE

After initial planning, iterative design refinement produced the final simplified approach:

### Iteration 1: Initial Layering Fix
**Problem**: `data-model.md:191` and `:238` placed `OrchestrationPlan` and `OrchestrationResult` in domain package but wired them to `internal/registry` types, forcing domain layer to import infrastructure code.

**Solution**: Moved orchestration to application layer with pure domain DTOs.

### Iteration 2: Config vs Registry Boundary
**Problem**: Original design mixed embedded YAML fields, sidecar files, and CLI flags, creating confusion for implementers.

**Initial Solution**: Sidecar `.deps.yaml` files to avoid schema changes.

**Final Solution**: Top-level `dependencies` field in YAML (simpler, single source of truth).

### Iteration 3: Status Model Simplification
**Problem**: Multi-state model (Healthy, Pending, Running, Failed, Blocked) created complex state transitions.

**Solution**: Two-state model (Ready/Blocked) with reasons. "Running" is transient execution state, not persistent registry state.

### Final Design Summary

| Aspect | Decision |
|--------|----------|
| **Dependency Declaration** | Top-level `dependencies` field in YAML |
| **Status Model** | Two states: Ready (🟢), Blocked (🟠) |
| **Auto-Import** | `streamy add` imports dependencies from YAML |
| **Unregistered Dependencies** | Allowed, marked as Blocked with warnings |
| **Cycle Detection** | Rejected at registration time with clear error |
| **Orchestration Location** | Application layer (`internal/application/orchestration/`) |
| **Status Reconciliation** | Automatic on every `add` operation |

**Files Updated**:
- `specs/010-pipeline-dependencies/spec.md` - User Story 1, FR-001-010, simplified Key Entities
- `specs/010-pipeline-dependencies/DESIGN_RESOLUTION.md` - Complete design rationale
- `specs/010-pipeline-dependencies/research.md` - Section 5 (YAML schema with dependencies field)
- `specs/010-pipeline-dependencies/plan.md` - Constitution gates, this summary

**Removed from Design**:
- ~~Sidecar `.deps.yaml` files~~
- ~~CLI `--depends-on` flag~~
- ~~`registry set-deps` command~~
- ~~StatusPending, StatusHealthy, StatusRunning, StatusFailed enums~~
- ~~Separate `id` field (use generated ID from path)~~

**Validation**: All corrections maintain clean architecture boundaries, support source control workflows, and provide clear operational visibility with minimal complexity.

---

## Agent Context Update ✅ COMPLETE

Updated `.github/copilot-instructions.md` with:
- Go 1.25.1 + Existing Streamy codebase
- File-based YAML configuration, in-memory registry state
- Active technologies list refreshed

---

## Planning Phase Complete

**Status**: ✅ All planning tasks complete + design ambiguities resolved

**Deliverables**:
- ✅ Implementation plan (this file)
- ✅ Research document with all decisions
- ✅ Complete data model
- ✅ API contracts (Go interfaces)
- ✅ Developer quickstart guide
- ✅ Agent context updated
- ✅ **DESIGN_RESOLUTION.md** - Resolves 4 critical ambiguities:
  - Config vs registry boundary (sidecar `.deps.yaml` files)
  - Layering leak (orchestration in application layer, DTOs for cross-layer)
  - Status reconciliation (dynamic computation, `validate` command for stale deps)
  - Virtual nodes (render-time artifacts only, not persisted)

**Next Command**: `/speckit.tasks` to break down implementation into atomic tasks

**Branch**: `010-pipeline-dependencies`  
**Spec**: [spec.md](./spec.md)  
**READ THIS FIRST:** [DESIGN_RESOLUTION.md](./DESIGN_RESOLUTION.md)  
**Estimated Complexity**: Medium (extending existing patterns, no new external dependencies)
