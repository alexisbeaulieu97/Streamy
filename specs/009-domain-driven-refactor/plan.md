# Implementation Plan: Domain-Driven Architecture Refactor

**Branch**: `009-domain-driven-refactor` | **Date**: 2025-10-15 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/009-domain-driven-refactor/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Refactor Streamy's architecture to achieve strict domain-driven design with clear dependency direction (Infrastructure → Application → Domain). The refactoring will:

1. **Extract pure domain entities** from current mixed packages (`internal/config`, `internal/engine`) into a new `internal/domain/` layer with zero infrastructure dependencies
2. **Define port interfaces** at the application boundary (`internal/ports/`) for all external operations (configuration loading, plugin execution, logging, metrics)
3. **Create application layer** in `internal/application/` to orchestrate use cases through port interfaces
4. **Implement infrastructure adapters** in `internal/infrastructure/` that satisfy domain ports (YAML parser, plugin executor, charmbracelet/log wrapper)
5. **Wire dependencies explicitly** in `cmd/streamy/main.go` using constructor injection with no globals or service locators
6. **Migrate from zerolog to charmbracelet/log** as the unified logging backbone
7. **Enforce context propagation** throughout all layers for cancellation, timeouts, and correlation IDs

The migration follows the strangler pattern: introduce new architecture alongside existing code, validate outputs match in production-like tests until confidence is achieved, migrate incrementally, validate with existing integration tests (no behavior changes), then remove legacy code. Success measured by SC-007 (all existing integration tests pass) and architectural goals (domain tests run in <100ms without infrastructure).

## Technical Context

**Language/Version**: Go 1.25.1  
**Primary Dependencies**: 
- github.com/charmbracelet/log (NEW - replacing zerolog)
- github.com/charmbracelet/bubbletea (existing - TUI)
- github.com/charmbracelet/lipgloss (existing - TUI styling)
- github.com/spf13/cobra (existing - CLI framework)
- github.com/goccy/go-yaml (existing - config parsing)
- No external DI frameworks (manual constructor injection)

**Storage**: File-based YAML configuration, in-memory registry state (no persistent database)  
**Testing**: Go standard testing package with table-driven tests, existing ~70-80% coverage baseline  
**Target Platform**: Linux, macOS, Windows (cross-platform Go binary)  
**Project Type**: Single project - CLI tool with TUI interface  
**Performance Goals**: 
- Domain entity tests: <100ms total without infrastructure setup (SC-001)
- Application service tests: 90%+ coverage with mocks only (SC-002)
- Build time: <10 seconds (SC-008)
- Test suite: <20% increase in execution time (SC-008)
- Graceful shutdown: <5 seconds on context cancellation (SC-005)

**Constraints**: 
- Zero external behavior changes - all existing integration tests must pass (SC-007)
- Backward compatible with existing YAML configs and CLI commands
- No breaking changes to plugin interface (Constitution Principle III - Pre-1.0 exception allows internal refactor)
- Compile-time type safety for dependency wiring (SC-003)
- Must maintain current execution performance characteristics

**Scale/Scope**: 
- Current codebase: ~15-20 internal packages, ~10,000 LOC
- Plugin count: 7 built-in plugins (package, repo, symlink, copy, command, template, lineinfile) - all internal, no external plugin compatibility constraints
- Typical config: 10-50 steps with dependencies
- Maximum supported complexity: Up to 500 steps with 1000 dependencies (complex CI/CD-like workflows)
- Refactoring scope: 8 core packages need restructuring (config, engine, plugin, logger, domain/pipeline, app/pipeline, validation, model)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Principle I: Onboarding First ✅ PASS
- **Status**: PASS - No impact
- **Rationale**: Refactoring is internal architecture only. Binary remains self-contained with zero external dependencies. First-run experience unchanged.

### Principle II: Schema Clarity & Fun ✅ PASS
- **Status**: PASS - No impact
- **Rationale**: YAML schema remains identical. Config parsing moves to infrastructure adapter but format unchanged. Existing configs work without modification (SC-007).

### Principle III: Plugin-Centric Architecture ✅ PASS (Pre-1.0 Exception)
- **Status**: PASS with Pre-1.0 Exception
- **Rationale**: Plugin interface API may change during refactor (Evaluate/Apply pattern to port-based pattern). Constitution allows breaking changes pre-1.0 when no external plugin ecosystem exists. All 7 built-in plugins will migrate simultaneously. External users warned in v0.x that plugin API is unstable until v1.0.
- **Risk Mitigation**: Provide adapter wrappers if needed to preserve backward compatibility for any early external adopters.

### Principle IV: Safety by Default ✅ PASS
- **Status**: PASS - Enhanced
- **Rationale**: Dry-run, idempotency, and safety features remain unchanged. Context propagation and explicit error handling improve reliability and cancellation safety (SC-005).

### Principle V: Performance & Reliability ✅ PASS
- **Status**: PASS - Maintained with improvements
- **Rationale**: Success criteria enforce performance maintenance (SC-008: build <10s, tests <20% slower). Context propagation and structured errors improve reliability. Domain layer isolation enables faster unit tests (SC-001: <100ms).

### Principle VI: Extensibility & Composability ✅ PASS
- **Status**: PASS - Enhanced
- **Rationale**: Port/adapter pattern makes infrastructure swappable. New plugin types added without domain changes (SC-006). Clean boundaries enable future features (imports, groups, conditionals) without destabilizing core.

### Principle VII: Ecosystem Consistency ✅ PASS
- **Status**: PASS - Enhanced
- **Rationale**: Plugin interface becomes more consistent through ports. All plugins implement same contracts. Structured error handling standardized across all layers (FR-027).

**GATE RESULT**: ✅ PASS - All principles satisfied. Refactoring aligns with constitution goals of maintainability and extensibility while preserving user-facing behavior.

## Project Structure

### Documentation (this feature)

```
specs/009-domain-driven-refactor/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (architecture patterns, DI strategies, logging migration)
├── data-model.md        # Phase 1 output (domain entities, ports, adapters)
├── quickstart.md        # Phase 1 output (migration guide, testing strategy)
├── contracts/           # Phase 1 output (port interface definitions)
│   ├── ports.go         # Application-boundary execution/config/logging/observability ports
│   ├── registry-ports.go # Registry/validation/event ports
│   └── wiring.go        # Dependency injection patterns
└── checklists/
    └── requirements.md  # Already created - quality validation checklist
```

### Source Code (repository root)

**Current Structure** (before refactor):
```
internal/
├── config/              # MIXED: YAML parsing + domain Step entities
├── engine/              # MIXED: DAG logic + ExecutionContext infrastructure
├── logger/              # INFRASTRUCTURE: zerolog wrapper
├── plugin/              # MIXED: Plugin interface (domain) + PluginRegistry (infra)
├── plugins/             # INFRASTRUCTURE: concrete plugin implementations
├── model/               # DOMAIN: StepResult, EvaluationResult entities
├── validation/          # APPLICATION: validation orchestration
├── registry/            # INFRASTRUCTURE: pipeline registry persistence
├── tui/                 # INFRASTRUCTURE: Bubbletea TUI
├── ui/                  # INFRASTRUCTURE: UI components
├── domain/
│   └── pipeline/        # DOMAIN but tightly coupled to infrastructure
└── app/
    └── pipeline/        # APPLICATION but mixed concerns
```

**Target Structure** (after refactor):
```
internal/
├── ports/                       # NEW: Port interfaces at application boundary (preserves pure domain)
│   ├── config.go                # ConfigLoader port
│   ├── execution.go             # PluginExecutor, DAGBuilder, ExecutionPlanner ports
│   ├── logging.go               # Logger port
│   ├── observability.go         # MetricsCollector, Tracer ports
│   ├── plugins.go               # Plugin, PluginRegistry ports
│   ├── events.go                # EventPublisher, EventHandler, DomainEvent ports
│   └── registry.go              # RegistryStore, ValidationService ports
│
├── domain/                      # NEW: Pure domain layer (zero infra dependencies, no ports)
│   ├── pipeline/
│   │   ├── pipeline.go          # Pipeline aggregate root with validation
│   │   ├── pipeline_test.go
│   │   ├── step.go              # Step entity (extracted from config package)
│   │   ├── step_test.go
│   │   ├── plan.go              # ExecutionPlan entity (extracted from engine)
│   │   ├── plan_test.go
│   │   ├── result.go            # StepResult, VerificationResult entities
│   │   ├── result_test.go
│   │   └── errors.go            # Domain-specific error types
│   └── plugin/
│       ├── plugin.go            # Plugin domain interface
│       └── metadata.go          # PluginMetadata value objects
│
├── application/                 # NEW: Application layer (orchestrates via ports from ../ports/)
│   ├── pipeline/
│   │   ├── apply_usecase.go     # ApplyPipeline use case
│   │   ├── apply_usecase_test.go
│   │   ├── verify_usecase.go    # VerifyPipeline use case
│   │   ├── verify_usecase_test.go
│   │   └── prepare_usecase.go   # Prepare pipeline (parse + validate + plan)
│   └── validation/
│       ├── service.go           # Validation orchestration service
│       └── service_test.go
│
├── infrastructure/              # NEW: Infrastructure adapters
│   ├── config/
│   │   ├── yaml_loader.go       # YAML parser adapter (implements ConfigLoader port)
│   │   ├── yaml_loader_test.go
│   │   └── parser.go            # Low-level YAML parsing (extracted from old config/)
│   ├── engine/
│   │   ├── dag_builder.go       # DAG construction (extracted from old engine/)
│   │   ├── dag_builder_test.go
│   │   ├── executor.go          # Plugin executor adapter
│   │   ├── executor_test.go
│   │   └── context.go           # ExecutionContext infrastructure
│   ├── logging/
│   │   ├── logger.go            # charmbracelet/log wrapper (implements Logger port)
│   │   ├── logger_test.go
│   │   └── noop_logger.go       # No-op logger for testing
│   ├── metrics/
│   │   ├── collector.go         # Metrics adapter (implements MetricsCollector port)
│   │   └── noop_collector.go    # No-op for dev/test
│   ├── tracing/
│   │   ├── tracer.go            # Tracer adapter (implements Tracer port)
│   │   └── noop_tracer.go       # No-op for dev/test
│   ├── plugin/
│   │   ├── registry.go          # PluginRegistry adapter (moved from old plugin/)
│   │   ├── registry_test.go
│   │   └── loader.go            # Plugin loading logic
│   └── persistence/
│       ├── registry_store.go    # Pipeline registry persistence (existing)
│       └── registry_store_test.go
│
├── plugins/                     # UPDATED: Adapt to new port interfaces
│   ├── package/
│   ├── repo/
│   ├── symlink/
│   ├── copy/
│   ├── command/
│   ├── template/
│   └── lineinfile/
│
└── [legacy packages remain during migration, deprecated after]
    ├── config/          # DEPRECATED: logic moved to domain/ and infrastructure/
    ├── engine/          # DEPRECATED: logic moved to domain/ and infrastructure/
    ├── logger/          # DEPRECATED: replaced by infrastructure/logging/
    ├── plugin/          # DEPRECATED: split into domain/plugin/ and infrastructure/plugin/
    ├── model/           # DEPRECATED: moved to domain/pipeline/
    ├── validation/      # DEPRECATED: moved to application/validation/
    ├── domain/pipeline/ # DEPRECATED: replaced by new domain/pipeline/
    └── app/pipeline/    # DEPRECATED: replaced by application/pipeline/

cmd/
└── streamy/
    ├── main.go          # UPDATED: Explicit DI wiring (creates all adapters, injects to app)
    ├── app_context.go   # UPDATED: Holds wired dependencies
    ├── plugins_import.go # UPDATED: Register plugins with new adapter
    ├── apply.go         # UPDATED: Use new application layer use cases
    ├── verify.go        # UPDATED: Use new application layer use cases
    ├── dashboard.go     # UPDATED: Use new application layer use cases
    └── [other commands] # UPDATED: Use new application layer

pkg/
└── errors/              # EXISTING: Error utilities (may extend for domain errors)

tests/
├── integration_test.go          # UPDATED: Test new architecture end-to-end
├── integration_dashboard_test.go
├── integration_verify_test.go
└── [other integration tests]    # All must pass unchanged (SC-007)

docs/
├── architecture.md              # UPDATED: Document new DDD architecture
├── architecture-overview.md     # NEW: North-star architecture doc (Phase 0)
├── adr/
│   └── 001-domain-driven-refactor.md  # NEW: ADR documenting this refactor
└── [other docs]
```

**Structure Decision**: Single project structure maintained. New top-level directories under `internal/` establish clear layer boundaries (domain/, application/, infrastructure/). Legacy packages deprecated but retained during strangler pattern migration, removed in final cleanup phase.

**Legacy Package Transition Strategy**:
During the migration (Phases 3-8), both old and new package structures will coexist. To avoid ambiguity:

1. **Deprecation Markers**: Add `// Package Deprecated: Replaced by internal/domain/pipeline` comments to legacy package docs
2. **README Warnings**: Create `DEPRECATED.md` files in legacy packages pointing to new locations
3. **Naming Convention**: New packages use clear layer prefixes:
   - Domain: `internal/domain/pipeline/`, `internal/domain/plugin/`
   - Application: `internal/application/pipeline/`, `internal/application/validation/`
   - Infrastructure: `internal/infrastructure/config/`, `internal/infrastructure/engine/`, `internal/infrastructure/logging/`
4. **Import Aliases**: When both versions must coexist temporarily, use import aliases:
   ```go
   import (
       legacyconfig "github.com/.../internal/config"        // Old
       "github.com/.../internal/infrastructure/config"      // New
   )
   ```
5. **Removal Timeline**: Legacy packages removed in Phase 9 after strangler validation passes

**Legacy Packages** (to be removed in Phase 9):
- `internal/config/` → Split into `internal/domain/pipeline/` (entities) + `internal/infrastructure/config/` (parsing)
- `internal/engine/` → Split into `internal/domain/pipeline/` (plan) + `internal/infrastructure/engine/` (executor, DAG builder)
- `internal/logger/` → Replaced by `internal/infrastructure/logging/` (charmbracelet/log wrapper)
- `internal/plugin/` (mixed) → Split into `internal/domain/plugin/` (interface) + `internal/infrastructure/plugin/` (registry)
- `internal/model/` → Moved to `internal/domain/pipeline/` (result entities)
- `internal/domain/pipeline/service.go` → Replaced by `internal/application/pipeline/*_usecase.go`
- `internal/app/pipeline/` → Replaced by `internal/application/pipeline/`

This approach minimizes disruption while enabling incremental validation - each layer can be tested in isolation before integrating with others. The three-layer structure (domain/application/infrastructure) is industry-standard DDD pattern proven in Go ecosystems (e.g., Ben Johnson's "Standard Package Layout", Kat Zien's "How I Structure Go Apps").

## Complexity Tracking

*Fill ONLY if Constitution Check has violations that must be justified*

**No violations** - All constitution principles passed. This section left empty per template instructions.


## Phase 0: Outline & Research

**Objective**: Establish baseline, research architecture patterns, and resolve all NEEDS CLARIFICATION items.

### Actions

1. **Inventory & Baseline Assessment**
   - Map current package structure and dependencies (✅ COMPLETE - see spec.md "Current State Analysis")
   - Document all public APIs and entry points
   - Run `go test ./... -cover` and record coverage baseline
   - Identify all imports that cross layer boundaries (domain→infra violations)
   - List all plugins and their current interface implementations

2. **Architecture Research**
   - Research: Domain-Driven Design patterns in Go (hexagonal architecture, ports & adapters)
   - Research: Dependency Injection strategies in Go (manual constructor injection vs frameworks)
   - Research: charmbracelet/log migration from zerolog (API differences, performance implications)
   - Research: Context propagation best practices (correlation IDs, deadline handling, cancellation)
   - Research: Error wrapping strategies that preserve context through layers
   - Research: Testing strategies for port-based architecture (mocking, fakes, test doubles)

3. **North-Star Documentation**
   - Create `docs/architecture-overview.md` describing:
     - Layer responsibilities (Domain: business logic, Application: use cases, Infrastructure: adapters)
     - Dependency flow diagram (Infrastructure → Application → Domain)
     - Port interface definitions and naming conventions
     - Wiring strategy in `cmd/streamy/main.go`
     - Migration strategy (strangler pattern phases)
   - Create dependency diagram (ASCII or Mermaid) showing:
     - Current architecture with circular dependencies
     - Target architecture with unidirectional dependencies
     - Port interfaces and their adapter implementations

4. **Consolidate Findings**
   - Document all research findings in `research.md`
   - Resolve NEEDS CLARIFICATION items (none identified in Technical Context)
   - Define specific DI patterns to use (constructor injection with explicit parameter lists)
   - Define logging migration approach (zerolog→charmbracelet/log adapter wrapper initially, full migration later)
   - Define context key strategy for correlation IDs (unexported context key type for type safety)
   - Define event buffering strategy for initialization (in-memory buffer, max 1000 events, flush when logger available)
   - Define error aggregation pattern for multi-step failures (continue-on-error, collect all failures, report at end)

### Deliverables

- ✅ `research.md` - Architecture patterns, DI strategy, logging migration plan (COMPLETE)
- ✅ `docs/architecture-overview.md` - North-star architecture description (COMPLETE)
- ✅ Dependency diagram (in architecture-overview.md) (COMPLETE)
- ✅ Test coverage baseline report (COMPLETE)
- ✅ `docs/adr/001-domain-driven-refactor.md` - ADR documenting this refactoring decision (COMPLETE)

### Research Questions to Answer

| Question | Focus Area |
|----------|------------|
| What DDD patterns fit Go idioms? | Domain entity design, value objects, aggregate roots |
| How do Go projects structure ports/adapters? | Package naming, interface placement, adapter registration |
| What's the migration path from zerolog to charmbracelet/log? | API mapping, performance comparison, breaking changes |
| How to implement correlation IDs? | Context keys, log field injection, propagation across goroutines |
| What's the best DI approach for ~20 dependencies? | Constructor injection patterns, wire/dig vs manual, factory functions |
| How to test port-based architecture effectively? | Mock generation, test double patterns, integration test strategies |

---

## Phase 1: Design & Contracts

**Objective**: Define domain model, port interfaces, and API contracts.

**Prerequisites**: ✅ Phase 0 complete (research.md, architecture-overview.md)

### Actions

1. **Extract Domain Entities**
   - Identify all entities from current codebase:
     - `Pipeline` (from config.Config + domain logic)
     - `Step` (from config.Step with type-specific fields removed)
     - `ExecutionPlan` (from engine.ExecutionPlan + level-based execution)
     - `StepResult` (from model.StepResult)
     - `VerificationResult` (from model.VerificationSummary)
     - `Plugin` (from plugin.Plugin interface)
     - `PluginMetadata` (from plugin.PluginMetadata)
   - Document each entity in `data-model.md`:
     - Fields (no infrastructure types like loggers or contexts as fields)
     - Business rules and invariants (validation logic, state transitions)
     - Relationships (Pipeline has Steps, ExecutionPlan references Steps)
     - Methods (validation, state transitions, queries)

2. **Define Port Interfaces**
   - Create `contracts/ports.go` with application-boundary execution/config/logging/observability ports:
     - `ConfigLoader` - Load and parse pipeline configurations
     - `PluginExecutor` - Execute plugins against steps
     - `MetricsCollector` - Record execution metrics (step duration, success/failure counts)
     - `Tracer` - Distributed tracing spans
     - `Logger` - Structured logging with context
     - `DAGBuilder` / `ExecutionPlanner` - Execution planning ports
   - Create `contracts/registry-ports.go` with application auxiliary ports:
     - `RegistryStore` - Pipeline registry persistence
     - `ValidationService` - Post-execution validation checks
     - `EventPublisher` - Domain/event distribution
   - Document each port interface:
     - Purpose and responsibilities
     - Method signatures with context.Context as first parameter
     - Return types (use domain entities, not infrastructure types)
     - Error contracts (what errors each method can return)

3. **Define Wiring Patterns**
   - Create `contracts/wiring.go` documenting:
     - Constructor injection pattern for application services
     - Factory functions for creating fully-wired systems
     - Example: `func NewApplyUseCase(loader ConfigLoader, executor PluginExecutor, logger Logger) *ApplyUseCase`
     - Validation logic: compile-time type checking, no reflection

4. **Generate Quickstart Guide**
   - Create `quickstart.md` with:
     - Migration guide: how to add new features using new architecture
     - Example: Adding a new use case with port dependencies
     - Example: Implementing a new adapter for an existing port
     - Testing strategy: unit tests with mocks, integration tests with real adapters
     - Common patterns: error wrapping, context propagation, logging conventions

5. **Update Agent Context**
   - Run `.specify/scripts/bash/update-agent-context.sh copilot`
   - Add new packages to agent understanding: domain/, application/, infrastructure/
   - Document new patterns: port/adapter, DI wiring

### Deliverables

- ✅ `data-model.md` - All domain entities with fields, rules, relationships (COMPLETE)
- ✅ `contracts/ports.go` - Application-boundary execution/config/logging/observability ports (COMPLETE)
- ✅ `contracts/registry-ports.go` - Registry/validation/event ports (COMPLETE)
- ✅ `contracts/wiring.go` - DI wiring patterns and examples (COMPLETE)
- ✅ `quickstart.md` - Migration guide and testing strategies (COMPLETE)
- ✅ Updated `.github/copilot-instructions.md` - Agent context updated (COMPLETE)

### Re-evaluate Constitution Check

After design phase, re-check all principles:

- ✅ **Principle I (Onboarding First)**: No changes to binary or dependencies
- ✅ **Principle II (Schema Clarity)**: YAML schema unchanged, configs backward compatible
- ✅ **Principle III (Plugin-Centric)**: Plugin architecture preserved, interface improved with ports
- ✅ **Principle IV (Safety by Default)**: Dry-run, idempotency preserved, context cancellation added
- ✅ **Principle V (Performance)**: Performance goals maintained (SC-008), testing improved
- ✅ **Principle VI (Extensibility)**: Port/adapter pattern enhances extensibility
- ✅ **Principle VII (Ecosystem Consistency)**: Structured errors, consistent interfaces across plugins

**Constitution Check Result**: ✅ ALL PASS - Design phase maintains all constitution principles.

---

## Phase 2: Task Planning Approach

*This section describes what the /tasks command will do - DO NOT execute during /plan*

**Task Generation Strategy**:

1. **Load `.specify/templates/tasks-template.md`** as base structure

2. **Generate tasks from Phase 1 artifacts**:
   - `data-model.md` → entity implementation tasks
   - `contracts/*.go` → port interface implementation tasks
   - `quickstart.md` → migration and integration tasks

3. **Follow incremental migration approach**:
   - Tasks organized by layer (domain → application → infrastructure → wiring)
   - Each layer fully tested before moving to next
   - Strangler pattern: new code alongside old, migrate consumers incrementally

4. **Task grouping into phases**:
   - **Phase 3**: Domain Layer Implementation (entities, ports, tests)
   - **Phase 4**: Application Layer Implementation (use cases, orchestration, tests)
   - **Phase 5**: Infrastructure Adapters (config loader, executor, logging, tests)
   - **Phase 6**: Wiring & Integration (DI in main.go, CLI updates, migration)
   - **Phase 7**: Migration & Cleanup (strangler pattern completion, legacy removal, docs)

5. **Task ordering within each phase**:
   - Tests first (TDD): Write failing tests for interfaces
   - Interfaces: Define port interfaces
   - Implementation: Implement adapters satisfying ports
   - Integration: Wire together and test end-to-end
   - Documentation: Update docs and ADRs

**Task Categorization**:
- **Critical Path**: Domain entities → Application use cases → Infrastructure adapters → Wiring
- **Parallel Work**: Multiple adapters can be implemented concurrently once ports are defined
- **Validation Points**: After each phase, run full test suite and validate SC-007 (all integration tests pass)

**Dependencies**:
- Phase 3 (Domain) has no dependencies - pure business logic
- Phase 4 (Application) depends on Phase 3 (uses domain entities and ports)
- Phase 5 (Infrastructure) depends on Phase 3 (implements domain ports)
- Phase 6 (Wiring) depends on Phases 4 and 5 (wire application and infrastructure)
- Phase 7 (Cleanup) depends on Phase 6 (strangler pattern complete)

**Success Criteria Mapping**:
Each task must contribute to at least one success criterion:
- SC-001: Domain tests <100ms → Phase 3 tasks ensure zero infrastructure imports
- SC-002: 90%+ app coverage with mocks → Phase 4 tasks include mock-based tests
- SC-003: Compile-time DI → Phase 6 tasks use explicit constructors
- SC-004: 95%+ structured logging → Phase 5 logging adapter + Phase 6 integration
- SC-005: <5s graceful shutdown → Phase 4/5 context handling tasks
- SC-006: Add plugins without domain changes → Phase 5 adapter pattern
- SC-007: All integration tests pass → Validation after every phase
- SC-008: Performance maintained → Benchmarking tasks in Phase 7
- SC-009: Full error context → Phase 4/5 error wrapping tasks
- SC-010: Understandable domain → Phase 3 documentation tasks

---

## Risk Mitigation

| Risk | Impact | Probability | Mitigation Strategy |
|------|--------|-------------|---------------------|
| **Large scope causes delays** | High | Medium | Break into 5 phases with clear milestones. Each phase independently testable. Stop and reassess after each phase. |
| **Plugin compatibility breaks** | High | Low | All plugins are internal - no external compatibility constraints. Migrate all 7 plugins simultaneously with parallel testing. |
| **Performance regression** | Medium | Low | Benchmark after each phase (SC-008). Domain layer has no overhead (pure Go). Context propagation overhead negligible per Go benchmarks. Max 500 steps with 1000 deps supported. |
| **Context misuse** | Medium | Medium | Enforce via: 1) Linter rules (context as first param), 2) Code review checklist, 3) Example code in quickstart.md, 4) CI checks. |
| **Logging migration breaks output** | Low | Low | charmbracelet/log is drop-in replacement for most zerolog patterns. Adapter wrapper provides compatibility layer. |
| **Test coverage drops** | Medium | Low | TDD approach: tests written before implementation. Coverage gates in CI (90% domain, 80% app, 70% infra). |
| **DI wiring becomes complex** | Medium | Medium | Document wiring patterns in contracts/wiring.go. Use factory functions to hide complexity. Max 5-7 deps per constructor (validated in code review). |
| **Integration tests fail** | High | Medium | SC-007 is hard requirement. Run full integration suite after every phase using strangler pattern with output comparison. If failures occur, fix before proceeding. |
| **Team unfamiliarity with DDD** | Medium | Medium | Provide quickstart.md with examples. Document patterns in research.md. Use pair programming for first implementations. |
| **Circular dependencies reappear** | Low | Low | Enforce via: 1) Package import rules, 2) CI checks for dependency cycles, 3) Architecture decision records documenting boundaries. |
| **Event loss during initialization** | Low | Low | In-memory event buffer (max 1000 events) handles initialization window. Events flushed when logger available. Acceptable for startup phase. |
| **Partial pipeline failures unclear** | Low | Low | Continue-on-error pattern collects all failures. User sees complete failure report at end. Matches typical infrastructure automation tools (Ansible, Terraform). |

---

## Expected Outcomes

### Technical Outcomes

1. **Strict Dependency Direction** (Infrastructure → Application → Domain)
   - Domain package has zero imports from application or infrastructure
   - Application imports only domain packages
   - Infrastructure imports domain and application packages
   - Verified by: Import analysis tooling + CI checks

2. **Explicit Dependency Wiring**
   - All dependencies created in `cmd/streamy/main.go`
   - Constructor injection with interface parameters
   - No globals, no service locators, no reflection-based DI
   - Verified by: SC-003 (compile-time type checking)

3. **Unified Observability**
   - charmbracelet/log as logging backbone
   - Correlation IDs in all log entries (via context)
   - Structured errors with full context chain
   - Metrics and tracing ports ready for future implementation
   - Verified by: SC-004 (95%+ structured logging), SC-009 (full error context)

4. **Domain Logic Isolation**
   - Domain entities testable without infrastructure
   - Pure business logic with no framework coupling
   - Fast unit tests (<100ms for entire domain layer)
   - Verified by: SC-001 (domain tests <100ms)

5. **Easier Testing and Extension**
   - Mock-based unit tests for application services
   - New plugins added via infrastructure adapters
   - New use cases added without domain changes
   - Verified by: SC-002 (90%+ app coverage with mocks), SC-006 (plugin extensibility)

### Behavioral Outcomes

1. **Zero Breaking Changes**
   - All existing YAML configs work unchanged
   - All CLI commands maintain same behavior
   - All integration tests pass without modification
   - Verified by: SC-007 (all integration tests pass)

2. **Performance Maintained**
   - Build time <10 seconds
   - Test suite execution <20% slower
   - Context cancellation <5 seconds
   - Verified by: SC-008, SC-005

3. **Improved Developer Experience**
   - New developers understand architecture by reading domain layer first
   - Clear boundaries make it obvious where code belongs
   - Compile-time errors catch wiring mistakes early
   - Verified by: SC-010 (architecture understandability)

### Deliverables Summary

**Documentation**:
- ✅ `docs/architecture-overview.md` - North-star architecture
- ✅ `docs/adr/001-domain-driven-refactor.md` - Architecture Decision Record
- ✅ `specs/009-domain-driven-refactor/research.md` - Research findings
- ✅ `specs/009-domain-driven-refactor/data-model.md` - Domain model
- ✅ `specs/009-domain-driven-refactor/contracts/` - Port interfaces
- ✅ `specs/009-domain-driven-refactor/quickstart.md` - Migration guide

**Code Structure**:
- ✅ `internal/domain/` - Pure domain entities and ports (zero infra deps)
- ✅ `internal/application/` - Use case orchestration (depends on domain only)
- ✅ `internal/infrastructure/` - Adapters for all ports (implements interfaces)
- ✅ `cmd/streamy/main.go` - Explicit DI wiring (composition root)

**Validation**:
- ✅ All 10 success criteria met (SC-001 through SC-010)
- ✅ All integration tests pass unchanged
- ✅ Test coverage maintained or improved
- ✅ Performance characteristics maintained
- ✅ Constitution principles all pass

---

## Next Steps

This plan ends after **Phase 1: Design & Contracts**. The next command (`/speckit.tasks`) will:

1. Load Phase 1 artifacts (data-model.md, contracts/, quickstart.md)
2. Generate detailed implementation tasks for Phases 3-7
3. Create `tasks.md` with task breakdown, dependencies, and estimates
4. Organize tasks by layer (domain → application → infrastructure → wiring → cleanup)

**To proceed**:
```bash
# Current command (you are here): /speckit.plan
# Creates: plan.md, research.md, data-model.md, contracts/, quickstart.md

# Next command: /speckit.tasks  
# Creates: tasks.md with implementation tasks for Phases 3-7
```

**Branch Status**: `009-domain-driven-refactor`  
**Plan File**: `/home/alexis/Projects/Streamy/specs/009-domain-driven-refactor/plan.md`  
**Ready For**: `/speckit.tasks` to generate implementation tasks for Phases 3-7

---

## Plan Execution Status

### ✅ Phase 0: Outline & Research - COMPLETE

All deliverables generated:
- ✅ `research.md` - 6 architectural decisions documented (DDD patterns, DI strategy, logging migration, context propagation, error wrapping, testing strategy)
- ✅ `docs/architecture-overview.md` - North-star architecture with layer responsibilities, dependency diagrams, testing strategies
- ✅ `docs/adr/001-domain-driven-refactor.md` - Architecture Decision Record with context, decision rationale, alternatives considered
- ✅ Test coverage baseline - 80% average (config: 89.3%, engine: 80.9%, domain: 79.1%, logger: 85.0%)
- ✅ Dependency graphs - Current (circular) vs Target (unidirectional) in architecture-overview.md

### ✅ Phase 1: Design & Contracts - COMPLETE

All deliverables generated:
- ✅ `data-model.md` - 17 domain entities defined with fields, validation rules, relationships
- ✅ `contracts/ports.go` - Application-boundary execution/config/logging/observability ports (ConfigLoader, PluginExecutor, Logger, MetricsCollector, Tracer, DAGBuilder, ExecutionPlanner)
- ✅ `contracts/registry-ports.go` - Registry, validation, and event port interfaces (RegistryStore, ValidationService, EventPublisher) plus supporting types
- ✅ `contracts/wiring.go` - 6 DI patterns with examples, anti-patterns, guidelines
- ✅ `quickstart.md` - Migration guide with examples for adding use cases, implementing adapters, testing strategies, common patterns, troubleshooting
- ✅ Agent context updated - `.github/copilot-instructions.md` now includes Go 1.25.1 and database info

### 🔄 Phase 2: Task Planning - READY

**Command**: `/speckit.tasks`

**What it will do**:
1. Load Phase 1 artifacts (data-model.md, contracts/, quickstart.md)
2. Generate detailed implementation tasks for Phases 3-7
3. Create `tasks.md` with:
   - Task breakdown by layer (domain → application → infrastructure → wiring → cleanup)
   - Dependencies between tasks
   - Time estimates
   - Success criteria mapping
   - Validation checkpoints

**Phases to be tasked**:
- Phase 3: Domain Layer Implementation (entities, ports, tests)
- Phase 4: Application Layer Implementation (use cases, mocks, tests)
- Phase 5: Infrastructure Adapters (config loader, executor, charmbracelet/log, metrics)
- Phase 6: Wiring & Integration (DI in main.go, CLI updates, strangler pattern)
- Phase 7: Migration & Cleanup (legacy removal, documentation, benchmarking)

### Clarifications Applied

Based on clarification session 2025-10-15:
1. **Migration validation**: Strangler Pattern with parallel implementations and output comparison
2. **Scale targets**: Support up to 500 steps with 1000 dependencies
3. **Event buffering**: In-memory buffer (max 1000 events) during initialization, flush when logger available
4. **Plugin compatibility**: All plugins internal - no external compatibility constraints
5. **Error handling**: Continue-on-error with aggregated failure reporting
