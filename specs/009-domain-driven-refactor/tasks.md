# Implementation Tasks: Domain-Driven Architecture Refactor

**Feature**: 009-domain-driven-refactor  
**Branch**: `009-domain-driven-refactor`  
**Date**: 2025-10-15  
**Last Updated**: 2025-10-15 (Port Relocation)

---

## Overview

This document breaks down the domain-driven architecture refactor into executable tasks organized by user story priority. The refactoring follows a layer-by-layer approach (Domain → Ports → Application → Infrastructure → Wiring) using the Strangler Pattern for safe incremental migration.

**Total Tasks**: 263  
**Parallelizable Tasks**: 48  
**User Stories**: 6 (P1: 2, P2: 3, P3: 1)

**Key Architectural Decision**: Port interfaces are defined at the **application boundary** (`internal/ports/`) rather than within domain packages. This preserves a truly pure domain core with zero knowledge of infrastructure contracts. See `PORT_RELOCATION_SUMMARY.md` for details.

---

## Task Organization

Tasks are grouped into phases based on user story priorities from the specification:

- **Phase 1**: Setup (project structure, tooling)
- **Phase 2**: Foundational (domain layer + ports - blocks all user stories)
- **Phase 3**: User Story 1 (P1) - Domain Stability
- **Phase 4**: User Story 2 (P1) - Plugin Swappability
- **Phase 5**: User Story 3 (P2) - Unified Observability
- **Phase 6**: User Story 4 (P2) - Isolated Testing
- **Phase 7**: User Story 5 (P2) - Context Propagation
- **Phase 8**: User Story 6 (P3) - Structured Errors
- **Phase 9**: Polish & Cross-Cutting Concerns

---

## Dependencies & Execution Order

### Critical Path
```
Phase 1 (Setup) → Phase 2 (Foundational: Domain + Ports) → 
  ├─→ Phase 3 (US1) ─┐
  ├─→ Phase 4 (US2) ─┤
  ├─→ Phase 5 (US3) ─┼─→ Phase 6 (US4) ─┐
  └─→ Phase 7 (US5) ─┘                   ├─→ Phase 8 (US6) → Phase 9 (Polish)
                                          │
                      Phase 7 (US5) ──────┘
```

### Parallel Opportunities

**After Phase 2 (Domain + Ports Complete)**:
- Phases 3-5 can execute in parallel (US1, US2, US3)
- Phase 7 (US5 - Context) can start in parallel with US1-US3

**After Phases 3-5 Complete**:
- Phase 6 (US4 - Testing) requires mocks from all previous stories
- Phase 8 (US6 - Errors) builds on all layers

**Within Each Phase**:
- Tasks marked `[P]` can execute in parallel (different files, no dependencies)

---

## Implementation Strategy

### MVP Scope (Minimum Viable Product)
**Recommended**: Phase 1 + Phase 2 + Phase 3 (US1 only)

This delivers the core value:
- ✅ Domain layer isolated (zero infrastructure deps, zero port knowledge)
- ✅ Port interfaces defined at application boundary
- ✅ Domain tests run in <100ms
- ✅ Foundation for all other stories

### Incremental Delivery
1. **Iteration 1**: Phases 1-3 (Setup + Domain/Ports + US1) - Core architecture proof
2. **Iteration 2**: Phases 4-5 (US2 + US3) - Infrastructure flexibility + Observability
3. **Iteration 3**: Phases 6-7 (US4 + US5) - Testing + Reliability
4. **Iteration 4**: Phases 8-9 (US6 + Polish) - UX + Production readiness

---

## Phase 1: Setup

**Goal**: Initialize project structure and tooling for domain-driven architecture.

**Deliverables**: Directory structure, CI updates, baseline documentation

### Tasks

- [X] T001 Create `internal/domain/pipeline/` package directory
- [X] T002 Create `internal/domain/plugin/` package directory  
- [X] T003 Create `internal/ports/` package directory (port interfaces at application boundary)
- [X] T004 Create `internal/application/pipeline/` package directory
- [X] T005 Create `internal/application/validation/` package directory
- [X] T006 Create `internal/infrastructure/config/` package directory
- [X] T007 Create `internal/infrastructure/engine/` package directory
- [X] T008 Create `internal/infrastructure/logging/` package directory
- [X] T009 Create `internal/infrastructure/plugin/` package directory
- [X] T010 Create `internal/infrastructure/metrics/` package directory
- [X] T011 Create `internal/infrastructure/tracing/` package directory
- [X] T012 Create `internal/infrastructure/events/` package directory (EventPublisher implementation)
- [X] T013 Update `.gitignore` to exclude coverage reports and temp files
- [X] T014 Create `internal/domain/README.md` documenting domain layer rules (zero infra deps, zero port knowledge - truly pure)
- [X] T015 Create `internal/ports/README.md` documenting port placement rationale (application boundary, not domain)
- [X] T016 Create `internal/application/README.md` documenting application layer rules (uses domain + ports)
- [X] T017 Create `internal/infrastructure/README.md` documenting infrastructure layer rules (implements ports)
- [X] T018 Add `golangci-lint` config with import restrictions (domain cannot import infra/ports, ports cannot import domain)
- [X] T019 Update CI pipeline to run import cycle detection (`go mod graph` analysis)
- [X] T020 Document port relocation decision in `docs/adr/002-port-placement-at-boundary.md`

---

## Phase 2: Foundational (Domain Layer + Port Interfaces)

**Goal**: Extract pure domain entities with zero infrastructure/port knowledge AND define port interfaces at application boundary. This blocks all user stories.

**Independent Test**: 
- Domain tests run in <100ms without any infrastructure setup (SC-001)
- Domain compiles with zero imports from infrastructure/application/ports
- Ports compile independently with zero domain imports

**Deliverables**: Domain entities (pure Go, no I/O), port interface definitions (application boundary)

### Domain Entities

- [X] T021 [P] [FOUND] Create `internal/domain/pipeline/errors.go` with DomainError type and error codes (11 codes per errors.md)
- [X] T022 [P] [FOUND] Create `internal/domain/pipeline/settings.go` with Settings value object
- [X] T023 [P] [FOUND] Create `internal/domain/pipeline/validation.go` with Validation entity
- [X] T024 [FOUND] Create `internal/domain/pipeline/step.go` with Step entity (extract from `internal/config/types.go`)
- [X] T025 [FOUND] Add Step.Validate() method implementing business rules (ID pattern, type validation, config schema)
- [X] T026 [FOUND] Add Step test in `internal/domain/pipeline/step_test.go` (table-driven, test validation rules)
- [X] T027 [FOUND] Create `internal/domain/pipeline/pipeline.go` with Pipeline aggregate root (extract from `internal/config/types.go`)
- [X] T028 [FOUND] Add Pipeline.Validate() method (unique step IDs, validate dependencies, no cycles)
- [X] T029 [FOUND] Add Pipeline.ValidateDependencies() method checking all DependsOn references exist
- [X] T030 [FOUND] Add Pipeline test in `internal/domain/pipeline/pipeline_test.go` (test invariants, edge cases)
- [X] T031 [P] [FOUND] Create `internal/domain/pipeline/plan.go` with ExecutionPlan value object (extract from `internal/engine/dag.go`)
- [X] T032 [P] [FOUND] Add ExecutionPlan.Validate() ensuring dependencies satisfied per level
- [X] T033 [P] [FOUND] Add ExecutionPlan test in `internal/domain/pipeline/plan_test.go`
- [X] T034 [P] [FOUND] Create `internal/domain/pipeline/result.go` with StepResult and VerificationResult value objects (extract from `internal/model/`)
- [X] T035 [P] [FOUND] Add result helper methods (IsSuccess, IsFailure, FormatOutput)
- [X] T036 [P] [FOUND] Add result tests in `internal/domain/pipeline/result_test.go`

### Port Interfaces (Application Boundary)

- [X] T037 [FOUND] Create `internal/ports/config.go` with ConfigLoader interface (Load, Validate methods with context.Context)
- [X] T038 [FOUND] Add comprehensive godoc to ConfigLoader explaining purpose, implementations, error contracts
- [X] T039 [P] [FOUND] Create `internal/ports/execution.go` with PluginExecutor interface (Execute, Verify methods)
- [X] T040 [P] [FOUND] Add DAGBuilder interface to `execution.go` (Build method taking []Step, returning ExecutionPlan)
- [X] T041 [P] [FOUND] Add ExecutionPlanner interface to `execution.go` (Plan method for execution scheduling)
- [X] T042 [FOUND] Add comprehensive godoc to execution interfaces explaining parallelization, cancellation, error handling
- [X] T043 [P] [FOUND] Create `internal/ports/logging.go` with Logger interface (Debug, Info, Warn, Error, With methods)
- [X] T044 [P] [FOUND] Add correlation ID helpers to `logging.go` (WithCorrelationID, GetCorrelationID, GenerateCorrelationID per observability.md)
- [X] T045 [FOUND] Add godoc explaining structured logging conventions, field naming (snake_case), log levels
- [X] T046 [P] [FOUND] Create `internal/ports/observability.go` with MetricsCollector interface (IncCounter, SetGauge, ObserveHistogram)
- [X] T047 [P] [FOUND] Add Tracer interface to `observability.go` (StartSpan, Extract, Inject methods)
- [X] T048 [P] [FOUND] Add Span interface to `observability.go` (End, SetError, SetAttribute methods)
- [X] T049 [FOUND] Add godoc documenting standard metric names (13 metrics per observability.md)
- [X] T050 [FOUND] Add godoc documenting span naming conventions (11 standard spans per observability.md)
- [X] T051 [P] [FOUND] Create `internal/ports/plugins.go` with Plugin interface (Evaluate, Apply methods)
- [X] T052 [P] [FOUND] Add PluginRegistry interface to `plugins.go` (Get, List, Register methods)
- [X] T053 [FOUND] Add godoc explaining plugin lifecycle and registry management
- [X] T054 [P] [FOUND] Create `internal/ports/events.go` with EventPublisher interface (Publish, Subscribe methods per observability.md)
- [X] T055 [P] [FOUND] Add EventHandler type and DomainEvent interface to `events.go` (Type, Payload, Timestamp, CorrelationID)
- [X] T056 [P] [FOUND] Add Subscription interface to `events.go` (Unsubscribe method)
- [X] T057 [FOUND] Add godoc documenting 11 standard event types (pipeline.*, step.*, validation.*)
- [X] T058 [FOUND] Add godoc explaining synchronous dispatch model, error handling, thread safety
- [X] T059 [P] [FOUND] Create `internal/ports/registry.go` with RegistryStore interface (Store, Get, List, Delete methods)
- [X] T060 [P] [FOUND] Add ValidationService interface to `registry.go` (RunValidations method)
- [X] T061 [FOUND] Add godoc explaining registry persistence and validation orchestration

### Plugin Domain

- [X] T062 [P] [FOUND] Create `internal/domain/plugin/plugin.go` with Plugin domain types (PluginType, PluginStatus enums)
- [X] T063 [P] [FOUND] Create `internal/domain/plugin/metadata.go` with PluginMetadata value object (Name, Version, Dependencies)
- [X] T064 [P] [FOUND] Add plugin tests in `internal/domain/plugin/plugin_test.go`

### Domain Validation

- [X] T065 [FOUND] Run `go test ./internal/domain/... -v` and verify all tests pass
- [X] T066 [FOUND] Run `go test ./internal/domain/... -cover` and verify >90% coverage
- [X] T067 [FOUND] Measure domain test execution time - must be <100ms (SC-001)
- [X] T068 [FOUND] Run `go list -f '{{.Imports}}' ./internal/domain/...` and verify zero imports from application/infrastructure/ports (truly pure)
- [X] T069 [FOUND] Run `go list -f '{{.Imports}}' ./internal/ports/...` and verify it only imports domain packages and stdlib
- [X] T070 [FOUND] Run `go list -f '{{.Deps}}' ./internal/ports/...` and verify ports depend only on domain packages and stdlib
- [X] T071 [FOUND] Update `docs/architecture-overview.md` with domain layer status: COMPLETE
- [X] T072 [FOUND] Update `docs/architecture-overview.md` with ports layer status: COMPLETE

---

## Phase 3: User Story 1 (P1) - Domain Stability

**User Story**: A maintainer changes the CLI interface or adds a new TUI component without modifying any domain entity logic. The domain layer continues to work identically because it has no knowledge of infrastructure concerns.

**Independent Test**: Modify CLI command structure in `cmd/streamy` and verify all domain entity tests pass without modification, and system executes pipelines correctly.

**Success Criteria**: 
- Domain layer has zero dependencies on CLI/TUI
- CLI changes don't require domain changes
- All domain tests pass unchanged

### Application Layer (Use Cases)

- [X] T073 [US1] Create `internal/application/pipeline/prepare_usecase.go` with PrepareUseCase struct
- [X] T074 [US1] Add PrepareUseCase constructor accepting ports from `internal/ports` (ConfigLoader, DAGBuilder, Logger)
- [X] T075 [US1] Implement PrepareUseCase.Prepare(ctx, configPath) method: load → validate → build plan
- [X] T076 [US1] Create `internal/application/pipeline/apply_usecase.go` with ApplyUseCase struct
- [X] T077 [US1] Add ApplyUseCase constructor accepting all required ports from `internal/ports` (ConfigLoader, PluginExecutor, Logger, EventPublisher, MetricsCollector)
- [X] T078 [US1] Implement ApplyUseCase.Apply(ctx, configPath, dryRun) method orchestrating: prepare → execute → validate
- [X] T079 [US1] Add error aggregation logic to ApplyUseCase (continue-on-error, collect failures, report at end per FR-009 and errors.md)
- [X] T080 [US1] Create `internal/application/pipeline/verify_usecase.go` with VerifyUseCase struct  
- [X] T081 [US1] Implement VerifyUseCase.Verify(ctx, configPath) method: load → build plan → verify each step
- [X] T082 [US1] Create `internal/application/validation/service.go` with ValidationService (pending)
- [X] T083 [US1] Implement ValidationService.RunValidations(ctx, pipeline, results) method (pending)

### Infrastructure Adapters

- [X] T084 [P] [US1] Create `internal/infrastructure/config/yaml_loader.go` implementing ports.ConfigLoader
- [X] T085 [US1] Add YAMLLoader.Load(ctx, path) method: read file → parse YAML → validate → return domain.Pipeline
- [X] T086 [US1] Add YAMLLoader.Validate(ctx, path) method checking YAML syntax and schema
- [X] T087 [US1] Add context cancellation checks in YAMLLoader before expensive operations
- [X] T088 [US1] Implement error wrapping per errors.md (wrap parse errors with ErrCodeValidation, file errors with ErrCodeNotFound)
- [X] T089 [US1] Add YAMLLoader tests in `internal/infrastructure/config/yaml_loader_test.go` using `t.TempDir()`
- [X] T090 [P] [US1] Create `internal/infrastructure/engine/dag_builder.go` implementing ports.DAGBuilder
- [X] T091 [US1] Extract DAG construction logic from `internal/engine/dag.go` to dag_builder.go
- [X] T092 [US1] Add DAGBuilder.Build(ctx, steps) method returning domain.ExecutionPlan
- [X] T093 [US1] Add cycle detection in DAGBuilder returning DomainError with ErrCodeCycle if cycle found
- [X] T094 [US1] Add DAGBuilder tests in `internal/infrastructure/engine/dag_builder_test.go`

### Wiring & Integration

- [X] T095 [US1] Update `cmd/streamy/main.go` to create YAMLLoader adapter (imports internal/infrastructure/config)
- [X] T096 [US1] Update `cmd/streamy/main.go` to create DAGBuilder adapter (imports internal/infrastructure/engine)
- [X] T097 [US1] Update `cmd/streamy/main.go` to wire PrepareUseCase with adapters (imports internal/application/pipeline)
- [X] T098 [US1] Update `cmd/streamy/main.go` to wire ApplyUseCase with adapters
- [X] T099 [US1] Update `cmd/streamy/main.go` to wire VerifyUseCase with adapters
- [X] T100 [US1] Update `cmd/streamy/apply.go` to use ApplyUseCase instead of direct domain service
- [X] T101 [US1] Update `cmd/streamy/verify.go` to use VerifyUseCase instead of direct domain service

### Story Validation

- [X] T102 [US1] Run all domain tests - verify zero changes required
- [X] T103 [US1] Run `tests/integration_test.go` - verify pipeline execution works
- [X] T104 [US1] Modify CLI command structure (add/remove flag) - verify domain unchanged
- [X] T105 [US1] Run full test suite - verify all integration tests pass (SC-007 validation)
- [X] T106 [US1] Document US1 completion in `specs/009-domain-driven-refactor/PROGRESS.md`

---

## Phase 4: User Story 2 (P1) - Plugin Swappability

**User Story**: A developer replaces the current plugin execution engine with a different implementation without touching domain entities or application services. The system continues to execute pipelines identically.

**Independent Test**: Create an alternative plugin adapter implementation, swap it in the wiring layer, verify all integration tests pass with identical results.

**Success Criteria**:
- Plugin executor is swappable via port interface
- Domain and application layers unchanged
- Integration tests pass with new adapter

### Infrastructure Adapters

- [X] T107 [P] [US2] Create `internal/infrastructure/engine/executor.go` implementing ports.PluginExecutor
- [X] T108 [US2] Extract plugin execution logic from `internal/engine/executor.go` to new executor.go
- [X] T109 [US2] Implement Executor.Execute(ctx, plan, pipeline) method: iterate levels → run steps in parallel → collect results
- [X] T110 [US2] Add context cancellation handling in Executor (check ctx.Err() between levels per FR-017)
- [X] T111 [US2] Implement Executor.Verify(ctx, pipeline) method calling plugin Evaluate for each step
- [X] T112 [US2] Implement error wrapping per errors.md (ErrCodeExecution, ErrCodeTimeout, ErrCodeCancelled)
- [X] T113 [US2] Add Executor tests in `internal/infrastructure/engine/executor_test.go`
- [X] T114 [P] [US2] Create `internal/infrastructure/plugin/registry.go` implementing ports.PluginRegistry
- [X] T115 [US2] Extract plugin registry logic from `internal/plugin/registry_new.go` to new registry.go
- [X] T116 [US2] Implement Registry.Get(pluginType) and Registry.List() methods
- [X] T117 [US2] Add Registry.Register(pluginType, factory) method for plugin registration
- [X] T118 [US2] Add Registry tests in `internal/infrastructure/plugin/registry_test.go`

### Plugin Migration

- [X] T119 [P] [US2] Update `internal/plugins/package/` to implement ports.Plugin interface (Evaluate, Apply methods)
- [X] T120 [P] [US2] Update `internal/plugins/repo/` to implement ports.Plugin interface
- [X] T121 [P] [US2] Update `internal/plugins/symlink/` to implement ports.Plugin interface
- [X] T122 [P] [US2] Update `internal/plugins/copy/` to implement ports.Plugin interface
- [X] T123 [P] [US2] Update `internal/plugins/command/` to implement ports.Plugin interface
- [X] T124 [P] [US2] Update `internal/plugins/template/` to implement ports.Plugin interface
- [X] T125 [P] [US2] Update `internal/plugins/lineinfile/` to implement ports.Plugin interface

### Wiring & Integration

- [X] T126 [US2] Update `cmd/streamy/main.go` to create Executor adapter
- [X] T127 [US2] Update `cmd/streamy/main.go` to create Registry adapter
- [X] T128 [US2] Update `cmd/streamy/plugins_import.go` to register plugins with new Registry
- [X] T129 [US2] Wire Executor into ApplyUseCase and VerifyUseCase in main.go
- [X] T130 [US2] Remove old plugin imports from domain layer (if any remain)

### Story Validation

- [X] T131 [US2] Create alternative test plugin executor adapter in `internal/infrastructure/engine/executor_test_impl.go`
- [X] T132 [US2] Swap test executor in integration test, verify identical results
- [X] T133 [US2] Run `tests/integration_plugin_dependency_test.go` - verify plugin system works
- [X] T134 [US2] Run full test suite - verify all tests pass (SC-007)
- [X] T135 [US2] Document US2 completion in `PROGRESS.md`

---

## Phase 5: User Story 3 (P2) - Unified Observability

**User Story**: A developer investigates a pipeline failure by examining structured logs from charmbracelet/log. All layers emit consistent, contextual log entries with proper correlation IDs.

**Independent Test**: Introduce deliberate error in plugin, execute pipeline, verify logs contain correlated entries showing error propagation through all layers.

**Success Criteria**:
- All layers use charmbracelet/log
- Correlation IDs propagate through context
- 95%+ log statements structured (SC-004)

### Infrastructure Adapters

- [X] T111 [P] [US3] Create `internal/infrastructure/logging/logger.go` implementing Logger port
- [X] T112 [US3] Add Logger struct wrapping `*charm/log.Logger`
- [X] T113 [US3] Implement Logger.Debug/Info/Warn/Error methods with structured fields
- [X] T114 [US3] Implement Logger.With(fields...) method returning child logger
- [X] T115 [US3] Add correlation ID extraction from context in all log methods
- [X] T116 [US3] Add layer name field injection ("domain", "application", "infrastructure")
- [X] T117 [US3] Create `internal/infrastructure/logging/noop_logger.go` for testing
- [X] T118 [US3] Add Logger tests in `internal/infrastructure/logging/logger_test.go`
- [X] T119 [P] [US3] Create `internal/infrastructure/logging/context.go` with correlation ID helpers
- [X] T120 [US3] Add WithCorrelationID(ctx, id) function returning context with ID
- [X] T121 [US3] Add GetCorrelationID(ctx) function extracting ID from context
- [X] T122 [US3] Add GenerateCorrelationID() function creating UUID
- [X] T123 [US3] Add event buffer implementation in `logging/event_buffer.go` (max 1000 events, flush on logger init per FR-012)

### Wiring & Integration

- [X] T124 [US3] Update `cmd/streamy/main.go` to create Logger adapter from charmbracelet/log
- [X] T125 [US3] Generate correlation ID at CLI entry point, add to context
- [X] T126 [US3] Wire Logger into all use cases (PrepareUseCase, ApplyUseCase, VerifyUseCase)
- [X] T127 [US3] Wire Logger into all infrastructure adapters (YAMLLoader, Executor, Registry)
- [X] T128 [US3] Add log statements in domain layer (via EventEmitter port, not direct logger)
- [X] T129 [US3] Add log statements in application layer (use injected Logger)
- [X] T130 [US3] Add log statements in infrastructure layer (use injected Logger)
- [X] T131 [US3] Remove all `zerolog` imports from codebase
- [ ] T132 [US3] Delete old `internal/logger/` package

### Story Validation

- [X] T133 [US3] Introduce deliberate plugin error in test
- [X] T134 [US3] Execute pipeline, capture logs
- [X] T135 [US3] Verify correlation ID appears in all log entries
- [X] T136 [US3] Verify layer names correct in all log entries
- [ ] T137 [US3] Verify error context chain visible in logs
- [X] T138 [US3] Run `grep -r "zerolog" internal/` - verify zero matches
- [ ] T139 [US3] Audit log statements - verify 95%+ are structured (SC-004)
- [ ] T140 [US3] Document US3 completion in `PROGRESS.md`

---

## Phase 6: User Story 4 (P2) - Isolated Testing

**User Story**: A developer writes a unit test for an application service by injecting mock implementations of all port interfaces. The test runs in isolation without touching any infrastructure.

**Independent Test**: Write new application service, create mock implementations of its dependencies, run comprehensive unit tests achieving 100% coverage without infrastructure setup.

**Success Criteria**:
- Application tests use mocks only
- 90%+ application layer coverage (SC-002)
- Tests run without file I/O, plugins, or network

### Test Infrastructure

- [ ] T141 [P] [US4] Create `internal/application/pipeline/testutil/` package for test doubles
- [ ] T142 [P] [US4] Create `testutil/mock_config_loader.go` implementing ConfigLoader port
- [ ] T143 [P] [US4] Create `testutil/mock_plugin_executor.go` implementing PluginExecutor port
- [ ] T144 [P] [US4] Create `testutil/mock_logger.go` implementing Logger port with call tracking
- [ ] T145 [P] [US4] Create `testutil/mock_dag_builder.go` implementing DAGBuilder port
- [ ] T146 [P] [US4] Create `testutil/mock_planner.go` implementing ExecutionPlanner port
- [ ] T147 [P] [US4] Create `testutil/mock_metrics.go` implementing MetricsCollector port
- [ ] T148 [P] [US4] Create `testutil/mock_tracer.go` implementing Tracer port
- [ ] T149 [P] [US4] Create `testutil/mock_event_emitter.go` implementing EventEmitter port
- [ ] T150 [P] [US4] Create `testutil/mock_registry.go` implementing PluginRegistry port

### Application Layer Tests

- [ ] T151 [US4] Create `internal/application/pipeline/prepare_usecase_test.go`
- [ ] T152 [US4] Add test: PrepareUseCase with successful preparation
- [ ] T153 [US4] Add test: PrepareUseCase with load failure (verify error handling)
- [ ] T154 [US4] Add test: PrepareUseCase with validation failure
- [ ] T155 [US4] Add test: PrepareUseCase with DAG cycle error
- [ ] T156 [US4] Create `internal/application/pipeline/apply_usecase_test.go`
- [ ] T157 [US4] Add test: ApplyUseCase with successful apply
- [ ] T158 [US4] Add test: ApplyUseCase with mid-execution failures (verify error aggregation per FR-009)
- [ ] T159 [US4] Add test: ApplyUseCase with context cancellation
- [ ] T160 [US4] Add test: ApplyUseCase with dry-run mode
- [ ] T161 [US4] Create `internal/application/pipeline/verify_usecase_test.go`
- [ ] T162 [US4] Add test: VerifyUseCase with all steps satisfied
- [ ] T163 [US4] Add test: VerifyUseCase with drifted steps
- [ ] T164 [US4] Add test: VerifyUseCase with verification errors
- [ ] T165 [US4] Create `internal/application/validation/service_test.go`
- [ ] T166 [US4] Add test: ValidationService with all validations passing
- [ ] T167 [US4] Add test: ValidationService with validation failures

### Story Validation

- [ ] T168 [US4] Run `go test ./internal/application/... -v` - verify all tests pass
- [ ] T169 [US4] Run `go test ./internal/application/... -cover` - verify >90% coverage (SC-002)
- [ ] T170 [US4] Verify application tests complete in <500ms (fast unit tests)
- [ ] T171 [US4] Run `go list -test -f '{{.TestImports}}' ./internal/application/...` - verify no infrastructure imports in tests
- [ ] T172 [US4] Document US4 completion in `PROGRESS.md`

---

## Phase 7: User Story 5 (P2) - Context Propagation

**User Story**: During pipeline execution, a cancellation signal is received (Ctrl+C). Context cancellation propagates through all layers causing graceful shutdown without resource leaks.

**Independent Test**: Start long-running pipeline, cancel mid-execution, verify all resources released, goroutines terminate, clean shutdown within 5 seconds (SC-005).

**Success Criteria**:
- Context passed as first parameter in all methods
- Cancellation handled at all layers
- Graceful shutdown <5 seconds (SC-005)

### Context Implementation

- [ ] T173 [US5] Audit all domain methods - ensure `ctx context.Context` is first parameter
- [ ] T174 [US5] Audit all application methods - ensure context passed through
- [ ] T175 [US5] Audit all infrastructure methods - ensure context propagated
- [ ] T176 [US5] Add context cancellation checks in Executor.Execute between execution levels
- [ ] T177 [US5] Add context cancellation checks in DAGBuilder before expensive operations
- [ ] T178 [US5] Add context cancellation checks in YAMLLoader before file I/O
- [ ] T179 [US5] Add context deadline enforcement in plugin execution (respect step timeout)
- [ ] T180 [US5] Update all plugin implementations to check context.Done() channel
- [ ] T181 [US5] Add goroutine cleanup in Executor on context cancellation
- [ ] T182 [US5] Add resource cleanup (file handles, temp files) in infrastructure adapters on cancellation

### Wiring & Integration

- [ ] T183 [US5] Update `cmd/streamy/main.go` to create root context with signal handling (SIGINT, SIGTERM)
- [ ] T184 [US5] Add context with timeout to CLI commands (default 30 minutes, configurable)
- [ ] T185 [US5] Wire context through all use case calls in CLI
- [ ] T186 [US5] Add defer cleanup handlers in main.go for graceful shutdown

### Story Validation

- [ ] T187 [US5] Create long-running test pipeline (sleeps, large files, etc.)
- [ ] T188 [US5] Start pipeline execution in test
- [ ] T189 [US5] Cancel context mid-execution (simulate Ctrl+C)
- [ ] T190 [US5] Verify all goroutines terminate within 5 seconds (use pprof or runtime.NumGoroutine)
- [ ] T191 [US5] Verify no resource leaks (check file descriptors, temp files cleaned)
- [ ] T192 [US5] Run `tests/integration_test.go` with context cancellation scenarios
- [ ] T193 [US5] Document US5 completion in `PROGRESS.md`

---

## Phase 8: User Story 6 (P3) - Structured Errors

**User Story**: When a plugin fails during pipeline execution, the error bubbles up through application services to the CLI with full context: which step failed, why it failed, the attempted operation, and suggested remediation actions.

**Independent Test**: Simulate various failure scenarios in different layers, capture errors at CLI level, verify each error message contains all contextual information and follows consistent format.

**Success Criteria**:
- Error messages include full context chain (SC-009)
- Domain errors have typed error codes
- User-friendly messages at CLI level

### Error Implementation

- [ ] T194 [P] [US6] Enhance `internal/domain/pipeline/errors.go` with additional error codes (Config, Plugin, Timeout, etc.)
- [ ] T195 [P] [US6] Add error helper functions: NewValidationError, NewNotFoundError, NewDependencyError, etc.
- [ ] T196 [US6] Add error wrapping in application layer - add user guidance to domain errors
- [ ] T197 [US6] Add error wrapping in infrastructure layer - add technical context to errors
- [ ] T198 [US6] Update YAMLLoader to return DomainError with file path, line number on parse failure
- [ ] T199 [US6] Update Executor to return DomainError with step ID, plugin type on execution failure
- [ ] T200 [US6] Update DAGBuilder to return DomainError with cycle path on dependency cycle
- [ ] T201 [US6] Update ValidationService to collect all validation errors and return aggregated error

### CLI Error Formatting

- [ ] T202 [US6] Create `cmd/streamy/errors.go` with error formatting functions
- [ ] T203 [US6] Add FormatError(err) function extracting DomainError details for display
- [ ] T204 [US6] Add error categorization (parse, validation, execution, system)
- [ ] T205 [US6] Add suggested remediation messages per error code
- [ ] T206 [US6] Update all CLI commands to use FormatError for error display
- [ ] T207 [US6] Add error examples to user documentation

### Story Validation

- [ ] T208 [US6] Simulate config parse error - verify error message includes file path, line number, field name
- [ ] T209 [US6] Simulate plugin execution error - verify error includes step ID, plugin type, operation, root cause
- [ ] T210 [US6] Simulate dependency cycle - verify error shows complete cycle path
- [ ] T211 [US6] Simulate multiple errors - verify all collected and displayed with categorization
- [ ] T212 [US6] Verify all error messages follow consistent format (SC-009)
- [ ] T213 [US6] Document US6 completion in `PROGRESS.md`

---

## Phase 9: Polish & Cross-Cutting Concerns

**Goal**: Finalize migration, remove legacy code, optimize performance, complete documentation.

**Success Criteria**: All success criteria met (SC-001 through SC-010), legacy code removed, documentation complete.

### Metrics & Tracing

- [ ] T214 [P] Create `internal/infrastructure/metrics/collector.go` implementing MetricsCollector port
- [ ] T215 [P] Add basic metrics: pipeline_executions_total, step_duration_seconds, step_failures_total
- [ ] T216 [P] Create `internal/infrastructure/metrics/noop_collector.go` for dev/test
- [ ] T217 [P] Create `internal/infrastructure/tracing/tracer.go` implementing Tracer port
- [ ] T218 [P] Add span creation for pipeline execution, step execution
- [ ] T219 [P] Create `internal/infrastructure/tracing/noop_tracer.go` for dev/test
- [ ] T220 Wire MetricsCollector into use cases in main.go
- [ ] T221 Wire Tracer into use cases in main.go

### Legacy Code Removal

- [ ] T222 Mark `internal/config/` package as DEPRECATED in README
- [ ] T223 Mark `internal/engine/` package as DEPRECATED in README
- [ ] T224 Mark `internal/logger/` package as DEPRECATED in README (already migrated)
- [ ] T225 Mark `internal/plugin/` package as DEPRECATED in README (split into domain/infra)
- [ ] T226 Mark `internal/model/` package as DEPRECATED in README (moved to domain)
- [ ] T227 Mark old `internal/domain/pipeline/service.go` as DEPRECATED
- [ ] T228 Mark old `internal/app/pipeline/service.go` as DEPRECATED
- [ ] T229 Remove all DEPRECATED packages after strangler validation passes
- [ ] T230 Update all imports to use new packages
- [ ] T231 Remove unused dependencies from `go.mod`

### Documentation

- [ ] T232 [P] Update `README.md` with new architecture section
- [ ] T233 [P] Update `docs/architecture.md` with implementation details
- [ ] T234 [P] Create `docs/testing-guide.md` with examples from quickstart.md
- [ ] T235 [P] Create `docs/adding-plugins.md` guide for plugin development
- [ ] T236 [P] Update `docs/plugins.md` with new port-based plugin interface
- [ ] T237 Add architecture diagrams to docs/ (generated from research.md diagrams)
- [ ] T238 Update CHANGELOG.md with refactoring summary

### Performance & Optimization

- [ ] T239 Run `go test ./... -bench=. -benchmem` - capture baseline benchmarks
- [ ] T240 Verify domain tests <100ms (SC-001)
- [ ] T241 Verify build time <10 seconds (SC-008)
- [ ] T242 Verify test suite <20% slower than baseline (SC-008)
- [ ] T243 Profile memory usage during large pipeline execution (500 steps)
- [ ] T244 Profile CPU usage during parallel step execution
- [ ] T245 Optimize hot paths if performance regressions found

### Final Validation

- [ ] T246 Run full test suite: `go test ./... -v -cover`
- [ ] T247 Verify all integration tests pass unchanged (SC-007 - CRITICAL)
- [ ] T248 Verify domain tests <100ms (SC-001)
- [ ] T249 Verify application coverage >90% (SC-002)
- [ ] T250 Verify compile-time DI (SC-003 - no interface{} in constructors)
- [ ] T251 Verify 95%+ structured logging (SC-004)
- [ ] T252 Verify graceful shutdown <5s (SC-005)
- [ ] T253 Verify plugin extensibility (SC-006 - add test plugin without domain changes)
- [ ] T254 Verify build time <10s, tests <20% slower (SC-008)
- [ ] T255 Verify full error context in all scenarios (SC-009)
- [ ] T256 Verify architecture understandable by reading domain first (SC-010 - doc review)
- [ ] T257 Run strangler validation: compare outputs from old vs new implementation in production-like tests
- [ ] T258 Update `specs/009-domain-driven-refactor/PROGRESS.md` - mark feature COMPLETE

### CI/CD Updates

- [ ] T259 Update CI pipeline to run new test structure
- [ ] T260 Add import cycle detection to CI
- [ ] T261 Add coverage reporting to CI (domain >90%, app >85%, infra >75%)
- [ ] T262 Add performance regression detection to CI
- [ ] T263 Update deployment scripts if needed

---

## Progress Tracking

Create `specs/009-domain-driven-refactor/PROGRESS.md` to track:

```markdown
# Implementation Progress

## Phase 1: Setup
- [ ] Complete (0/16 tasks)

## Phase 2: Foundational
- [ ] Complete (0/33 tasks)

## Phase 3: User Story 1 (P1)
- [ ] Complete (0/33 tasks)

## Phase 4: User Story 2 (P1)
- [ ] Complete (0/28 tasks)

## Phase 5: User Story 3 (P2)
- [ ] Complete (0/30 tasks)

## Phase 6: User Story 4 (P2)
- [ ] Complete (0/32 tasks)

## Phase 7: User Story 5 (P2)
- [ ] Complete (0/21 tasks)

## Phase 8: User Story 6 (P3)
- [ ] Complete (0/20 tasks)

## Phase 9: Polish
- [ ] Complete (0/47 tasks)

**Total**: 0/263 tasks complete
```

---

## Validation Checklist

After completing all phases, verify:

- [ ] ✅ **SC-001**: Domain tests <100ms without infrastructure
- [ ] ✅ **SC-002**: Application tests >90% coverage with mocks
- [ ] ✅ **SC-003**: DI is compile-time type-checked
- [ ] ✅ **SC-004**: 95%+ logs structured with charmbracelet/log
- [ ] ✅ **SC-005**: Graceful shutdown <5 seconds
- [ ] ✅ **SC-006**: Add plugins without domain/app changes
- [ ] ✅ **SC-007**: All integration tests pass unchanged (CRITICAL)
- [ ] ✅ **SC-008**: Build <10s, tests <20% slower
- [ ] ✅ **SC-009**: Error messages have full context chain
- [ ] ✅ **SC-010**: Architecture understandable from domain

---

## Notes

- Tasks marked `[P]` can be executed in parallel
- Tasks marked `[US#]` map to user stories from spec.md
- Tasks marked `[FOUND]` are foundational (block all user stories)
- File paths are absolute from repository root
- Each task is independently verifiable
- Strangler Pattern: validate outputs match before removing legacy code
