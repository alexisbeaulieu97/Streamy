# Tasks: Pipeline Dependencies

**Input**: Design documents from `/specs/010-pipeline-dependencies/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Tests included only where they directly support the feature's acceptance criteria.

**Organization**: Tasks are grouped by user story to enable independent implementation and validation of each story.

## Format: `[X?] [ID] [P?] [Story] Description`
- **[X]**: Optional completion marker placed before the task ID when complete
- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Story label (`Setup`, `Foundation`, `US1`, `US2`, `US3`, `Polish`)
- Include exact file paths in descriptions
- Examples: `- [X] T010 [US1] Update docs/...` (completed) and `- [ ] T999 [P] [US2] Implement feature in ...` (in progress)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish shared utilities and documentation needed by every user story.

- [X] T001 [Setup] Create canonical pipeline ID helper utilities in `internal/registry/id.go` with unit tests in `internal/registry/id_test.go`.
- [X] T002 [P] [Setup] Update `docs/schema.md` to document required `id`, `version`, and canonical `<id>@<version>` dependency entries.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before any user story begins. 🔒

- [X] T003 [Foundation] Enforce `id`, `version`, and canonical dependency validation in `internal/infrastructure/config/schema.go` with coverage in `internal/infrastructure/config/schema_test.go`.
- [X] T004 [Foundation] Extend registry status/data structures for dependencies and ready/blocked states in `internal/registry/types.go` and `internal/registry/status_cache.go`, including unit tests.
- [X] T005 [Foundation] Implement registry and status-cache migrations to version 2.0 in `internal/registry/migration.go` with tests in `internal/registry/migration_test.go`.
- [X] T006 [Foundation] Add dependency validation helpers (`FindUnregisteredDependencies`, cycle wrappers) in `internal/registry/registry.go` for reuse by CLI and orchestrator logic.

**Checkpoint**: Foundation ready – user story work can now begin.

---

## Phase 3: User Story 1 – Declare Pipeline Dependencies (Priority: P1) 🎯 MVP

**Goal**: Allow pipeline configs to declare dependencies that are imported into the registry and surfaced via CLI.

**Independent Test**: Register a config containing `dependencies`, confirm registry warns about missing pipelines, and verify `streamy registry list --tree` shows ready/blocked nodes.

### Implementation

- [X] T007 [US1] Update `cmd/streamy/add.go` and `cmd/streamy/add_test.go` to require `id`/`version`, load canonical dependencies, warn on missing entries, and mark new pipelines as blocked when necessary.
- [X] T008 [US1] Expose dependency/status inspection helpers in `internal/registry/registry.go` to support tree rendering and CLI queries.
- [X] T009 [US1] Implement the dependency tree output for `streamy registry list --tree` in `cmd/streamy/registry.go` with coverage in `cmd/streamy/registry_test.go`.
- [X] T010 [P] [US1] Document dependency declaration workflow and tree usage in `docs/pipeline-dependencies.md` and update `docs/ci-cd-pipeline.md` examples.

**Checkpoint**: User Story 1 is independently testable (register config ➜ run tree command).

---

## Phase 4: User Story 2 – Automatic Dependency Resolution and Execution (Priority: P2)

**Goal**: Provide an orchestration command that resolves dependency graphs and executes pipelines in order with parallel fan-out.

**Independent Test**: Run `streamy orchestrate <pipeline-id>` on a sample graph; confirm execution order, dry-run plan, and parallel levels.

### Implementation

- [X] T011 [US2] Implement DAG builder with Kahn’s algorithm in `internal/registry/graph.go` and unit tests in `internal/registry/graph_test.go`.
- [X] T012 [US2] Introduce orchestration DTOs in `internal/domain/pipeline/orchestration.go` with domain tests.
- [X] T013 [US2] Add application orchestration use case in `internal/application/orchestration/usecase.go` (plus tests) that leverages the DAG builder and existing apply pipeline.
- [X] T014 [US2] Add the `streamy orchestrate` CLI command in `cmd/streamy/orchestrate.go`, update root wiring (`cmd/streamy/root.go`) and `cmd/streamy/apply.go`, and cover behavior with CLI tests.
- [X] T015 [US2] Create end-to-end orchestration integration test demonstrating ordering/parallelism in `tests/integration_orchestrate_test.go`.

**Checkpoint**: User Stories 1 & 2 both function independently (dependency declaration + orchestration).

---

## Phase 5: User Story 3 – Failure Handling & Blocked State Propagation (Priority: P3)

**Goal**: Ensure failures block downstream pipelines, persist blocked state, and provide controlled overrides.

**Independent Test**: Force a mid-graph failure during `streamy orchestrate`, verify downstream nodes marked blocked, status cache updated, and `--force` confirmation works.

### Implementation

- [X] T016 [US3] Enhance the orchestration use case in `internal/application/orchestration/usecase.go` and related tests to mark dependents blocked and persist results via `internal/registry/status_cache.go`.
- [X] T017 [US3] Implement `--force` execution with interactive confirmation (and non-interactive `--yes` guard) in `cmd/streamy/orchestrate.go`.
- [X] T018 [US3] Add registry reconciliation to unblock pipelines once dependencies register (`internal/registry/registry.go`, `internal/registry/status_cache.go`) with accompanying tests.
- [X] T019 [P] [US3] Document blocked-state behavior, reconciliation, and `--force` safety in `docs/pipeline-dependencies.md` and `docs/architecture.md`.

**Checkpoint**: All three user stories now operate independently and together.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final documentation, validation, and release readiness.

- [X] T020 [Polish] Refresh `specs/010-pipeline-dependencies/quickstart.md` and `docs/pipeline-dependencies.md` with the final orchestration walkthrough and add a summary to `CHANGELOG.md`.
- [X] T021 [P] [Polish] Run `go fmt ./...`, `golangci-lint run`, and `go test ./...`; resolve issues and commit resulting changes.

---

## Dependencies & Execution Order

- **Phase 1 → Phase 2 → Phase 3 (US1) → Phase 4 (US2) → Phase 5 (US3) → Phase 6**  
- User story dependency chain: **US1 → US2 → US3** (later stories rely on artifacts from earlier ones).
- Within each story, tasks are listed in required execution order unless marked [P].

### Parallel Opportunities

- **Phase 1**: T002 can run after T001 begins (docs only).
- **US1**: T010 (docs) can proceed in parallel once T009 behavior is demonstrable.
- **US2**: After T012 completes, T013 and T014 must follow sequentially; no additional parallelism required. T015 waits for CLI implementation.
- **US3**: T019 (docs) can run in parallel once T016–T018 establish final behavior.
- **Polish**: T021 (quality gates) can run alongside documentation work once code stabilises.

---

## Implementation Strategy

### MVP First (Deliver User Story 1)
1. Complete Setup (Phase 1) and Foundational (Phase 2).
2. Implement Phase 3 (US1) and validate via `streamy add` + `streamy registry list --tree`.
3. Stop here if an MVP dependency registry is sufficient.

### Incremental Delivery
1. Finish Phases 1–3 → deliver MVP dependency declaration.
2. Phase 4 (US2) → deliver orchestration command; validate independently.
3. Phase 5 (US3) → add failure handling and overrides.
4. Phase 6 → final polish prior to release.

### Parallel Team Strategy
1. Team collaborates on Setup & Foundational tasks.
2. Assign stories once foundation solid:
   - Developer A: US1 (dependency declaration)
   - Developer B: US2 (orchestration engine)
   - Developer C: US3 (failure propagation & overrides)
3. Conclude with shared polish tasks.
