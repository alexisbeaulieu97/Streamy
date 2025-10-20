# Feature Specification: Domain-Driven Architecture Refactor

**Feature Branch**: `009-domain-driven-refactor`  
**Created**: 2025-10-15  
**Status**: Draft  
**Input**: User description: "Refactor Streamy into a domain-driven architecture with clear dependency direction (Infrastructure → Application → Domain), explicit wiring, unified error and observability strategy, and charmbracelet/log as the logging backbone."

## Current State Analysis

**Existing Structure** (after code exploration):
- Partial domain/app separation exists (`internal/domain/pipeline`, `internal/app/pipeline`)
- Domain layer heavily coupled to infrastructure:
  - `internal/domain/pipeline/service.go` imports `internal/config`, `internal/engine`, `internal/logger`, `internal/plugin`
  - Domain service depends on concrete `*plugin.PluginRegistry` type
  - No port interfaces - direct coupling to infrastructure implementations
- Logger uses `zerolog` (needs migration to `charmbracelet/log`)
- Context propagation exists in engine layer (`ExecutionContext.Context`) but not consistently used
- Some dependency injection in `cmd/streamy/main.go` but domain services still create their own dependencies
- Mixed concerns in packages:
  - `internal/config`: YAML parsing + domain Step entities (should split)
  - `internal/engine`: Execution logic + ExecutionContext infrastructure (should split)
  - `internal/plugin`: Plugin interface (domain) + PluginRegistry (infrastructure) in same package

**What Needs Refactoring**:
1. Extract pure domain entities from `internal/config` and `internal/engine`
2. Define port interfaces in domain layer for all infrastructure dependencies
3. Move `PluginRegistry`, config parsing, and logging to infrastructure layer
4. Implement adapters in infrastructure that satisfy domain ports
5. Migrate from `zerolog` to `charmbracelet/log`
6. Make context propagation explicit and consistent across all operations
7. Wire everything explicitly in `cmd/streamy/main.go` using constructor injection

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Core Domain Logic Remains Stable Across Interface Changes (Priority: P1)

A maintainer changes the CLI interface or adds a new TUI component without modifying any domain entity logic. The domain layer (pipelines, steps, verification results) continues to work identically because it has no knowledge of infrastructure concerns.

**Why this priority**: This is the foundation of domain-driven design. If domain logic is coupled to infrastructure, the entire refactoring fails to achieve its goal of separation and testability.

**Independent Test**: Can be fully tested by modifying a CLI command structure in `cmd/streamy` and verifying that all domain entity tests pass without modification, and that the system continues to execute pipelines correctly.

**Acceptance Scenarios**:

1. **Given** domain entities exist with pure business logic, **When** a developer modifies the CLI command structure or adds new terminal UI widgets, **Then** no domain entity code requires changes and all domain tests continue passing
2. **Given** a pipeline execution completes successfully, **When** the output format changes from TUI to JSON or plain text, **Then** the pipeline execution results remain identical regardless of presentation format
3. **Given** domain logic validates step dependencies, **When** infrastructure switches from YAML to JSON configuration format, **Then** validation rules continue working without domain layer changes

---

### User Story 2 - Plugin Implementation Swapped Without Domain Changes (Priority: P1)

A developer replaces the current plugin execution engine with a different implementation (e.g., switching from in-process to RPC-based plugins) without touching domain entities or application services. The system continues to execute pipelines identically.

**Why this priority**: Demonstrates that infrastructure is truly replaceable and that domain logic is protected from implementation details. This is critical for long-term maintainability.

**Independent Test**: Can be fully tested by creating an alternative plugin adapter implementation, swapping it in the wiring layer, and verifying that all integration tests pass with identical results.

**Acceptance Scenarios**:

1. **Given** pipeline execution depends on plugins, **When** the plugin engine implementation is replaced with a new adapter, **Then** pipeline execution produces identical results without domain or application layer changes
2. **Given** plugins report execution results, **When** the plugin result format changes at the infrastructure level, **Then** the domain's StepResult entity continues to capture all necessary information through the port interface
3. **Given** dependency resolution occurs in the application layer, **When** plugin metadata retrieval changes implementation, **Then** resolution logic remains unchanged and continues to produce correct execution plans

---

### User Story 3 - Unified Observability Across All Layers (Priority: P2)

A developer investigates a pipeline failure by examining structured logs from charmbracelet/log. All layers (domain, application, infrastructure) emit consistent, contextual log entries with proper correlation IDs, making the failure root cause immediately clear.

**Why this priority**: Essential for production debugging and operational visibility. Unified logging enables rapid troubleshooting without requiring different tools or formats per layer.

**Independent Test**: Can be fully tested by introducing a deliberate error in a plugin, executing a pipeline, and verifying that logs contain correlated entries showing the error propagation through all layers with full context.

**Acceptance Scenarios**:

1. **Given** a pipeline executes with multiple steps across different plugins, **When** an error occurs in any layer, **Then** structured logs show the complete error context including correlation ID, layer source, and full error chain
2. **Given** charmbracelet/log is configured at application startup, **When** any component emits a log entry, **Then** the entry includes consistent metadata (timestamp, level, correlation ID, layer name)
3. **Given** logs are collected during execution, **When** filtering by correlation ID, **Then** all related log entries from domain, application, and infrastructure appear in chronological order
4. **Given** verbose logging is enabled, **When** domain events occur (pipeline started, step validated, dependency resolved), **Then** appropriate log entries are emitted without coupling domain code to logging infrastructure

---

### User Story 4 - Explicit Dependency Injection Enables Isolated Testing (Priority: P2)

A developer writes a unit test for an application service (e.g., pipeline execution orchestrator) by injecting mock implementations of all port interfaces. The test runs in isolation without touching any infrastructure (no file I/O, no plugin loading, no external dependencies).

**Why this priority**: Testability is a primary driver of this refactoring. Without explicit dependency injection and port interfaces, the codebase cannot achieve true unit test isolation.

**Independent Test**: Can be fully tested by writing a new application service, creating mock implementations of its dependencies, and running comprehensive unit tests that achieve 100% coverage without any infrastructure setup.

**Acceptance Scenarios**:

1. **Given** an application service requires a configuration parser, **When** tests inject a mock parser that returns predefined configs, **Then** the service logic can be tested in complete isolation without YAML files
2. **Given** a domain entity emits events during state transitions, **When** tests inject a mock event handler, **Then** the entity's business logic can be verified without any infrastructure dependencies
3. **Given** the plugin execution port is defined, **When** tests inject a fake plugin that returns predetermined results, **Then** pipeline execution logic can be tested without loading actual plugins
4. **Given** dependency wiring occurs in main.go, **When** tests construct services manually with mocks, **Then** all application services can be instantiated and tested independently

---

### User Story 5 - Context Propagation Throughout Execution (Priority: P2)

During pipeline execution, a cancellation signal is received (e.g., user presses Ctrl+C). The context cancellation propagates through all layers—application orchestrator, domain entities, and infrastructure adapters—causing graceful shutdown without resource leaks.

**Why this priority**: Context propagation is critical for production reliability, resource management, and user experience. It must be designed into the architecture from the start.

**Independent Test**: Can be fully tested by starting a long-running pipeline, canceling it mid-execution, and verifying that all resources are released, all goroutines terminate, and the system reaches a clean shutdown state within a defined timeout.

**Acceptance Scenarios**:

1. **Given** a pipeline is executing multiple steps in parallel, **When** the execution context is canceled, **Then** all in-flight step executions terminate gracefully within 5 seconds
2. **Given** plugins are executing long-running operations, **When** context cancellation occurs, **Then** plugins receive the cancellation signal and stop work immediately
3. **Given** validation checks are running post-execution, **When** context is canceled, **Then** validation stops and returns partial results rather than blocking indefinitely
4. **Given** context carries correlation IDs for tracing, **When** execution flows through multiple layers, **Then** the correlation ID is preserved and appears in all log entries and error messages

---

### User Story 6 - Structured Error Handling with Full Context (Priority: P3)

When a plugin fails during pipeline execution, the error bubbles up through application services to the CLI with full context: which step failed, why it failed, the attempted operation, and suggested remediation actions. Users receive actionable error messages without needing to dig through logs.

**Why this priority**: Improves user experience and reduces support burden. While important, it's not critical to the architectural foundation and can be refined after core structure is established.

**Independent Test**: Can be fully tested by simulating various failure scenarios in different layers, capturing the errors at the CLI level, and verifying that each error message contains all contextual information and follows a consistent format.

**Acceptance Scenarios**:

1. **Given** a plugin fails during step execution, **When** the error reaches the CLI, **Then** the error message includes step ID, plugin type, operation attempted, root cause, and suggested fix
2. **Given** configuration parsing fails, **When** the error is reported, **Then** the message includes file path, line number, field name, and validation rule that failed
3. **Given** a dependency cycle is detected, **When** validation runs, **Then** the error message shows the complete cycle path and the steps involved
4. **Given** multiple errors occur during execution, **When** pipeline completes, **Then** all errors are collected and presented together with proper categorization (parse errors, validation errors, execution errors)

---

### Edge Cases

- What happens when domain entities need to emit events but logging infrastructure hasn't been initialized yet? **Resolution**: Events are buffered in memory during initialization, then flushed to logger once available. A lightweight in-memory buffer (max 1000 events) handles the initialization window.
- How does the system handle circular dependencies between domain entities themselves, not just between steps? **Resolution**: Domain model should be designed to prevent circular references through composition rather than bidirectional associations. Code review and static analysis enforce this pattern.
- What happens when context deadline is reached during a critical non-cancellable operation (e.g., atomic state transition)? **Resolution**: Critical sections should check context before starting and handle cancellation at safe points, with clear documentation of non-cancellable sections. Operations log warning if cancellation requested during critical section.
- How are database transactions or other infrastructure-level transactions handled across domain operations? **Resolution**: Application layer orchestrates transactions using repository interfaces; domain remains transaction-agnostic. Application services define transaction boundaries.
- What happens when dependency injection container fails to wire dependencies due to missing or incompatible implementations? **Resolution**: Application fails fast at startup with clear error message identifying missing dependency and expected interface. No runtime service location attempted.
- How should the application layer handle multi-step pipeline operations when some steps fail midway? **Resolution**: Continue execution where possible (respecting dependencies); collect all errors; report aggregated failures at end. This matches typical infrastructure automation tools and provides complete failure visibility.

## Requirements *(mandatory)*

### Functional Requirements

#### Domain Layer

- **FR-001**: System MUST define domain entities for Pipeline, Step, StepResult, VerificationResult, and ExecutionPlan as pure data structures with business logic methods but no infrastructure dependencies
- **FR-002**: Domain entities MUST validate their own invariants (e.g., Pipeline validates that step IDs are unique, Step validates that it has required fields)
- **FR-003**: Domain MUST define repository interfaces (ports) for persistence operations without coupling to specific storage implementations
- **FR-004**: Domain MUST define service interfaces (ports) for external operations (plugin execution, validation checks) without coupling to specific implementations
- **FR-005**: Domain MUST represent errors as typed error values that carry business context (step ID, operation type, validation rule) without referencing infrastructure details
- **FR-006**: Domain entities MUST be serializable for testing and state management without requiring infrastructure dependencies to be present

#### Application Layer

- **FR-007**: Application layer MUST orchestrate domain entities through use cases (ApplyPipeline, VerifyPipeline, ValidateConfiguration, BuildExecutionPlan)
- **FR-008**: Application services MUST depend only on domain entities and port interfaces, never on concrete infrastructure implementations
- **FR-009**: Application layer MUST manage transactions and coordinate multiple domain operations while maintaining consistency. When multi-step pipeline operations encounter failures midway, execution continues where possible (respecting dependencies), collects all errors, and reports aggregated failures at end.
- **FR-010**: Application layer MUST propagate context.Context through all operations to enable cancellation, timeouts, and distributed tracing
- **FR-011**: Application layer MUST translate domain errors into application-specific errors with additional context (user-facing messages, remediation suggestions)
- **FR-012**: Application layer MUST emit structured events at key points (pipeline started, step executed, validation completed) through event port interfaces. Events are buffered in memory during initialization (max 1000 events) and flushed to logger once available.

#### Infrastructure Layer

- **FR-013**: Infrastructure layer MUST implement all port interfaces defined in domain and application layers
- **FR-014**: Configuration parsing implementation MUST support YAML format and implement the configuration repository interface
- **FR-015**: Plugin engine implementation MUST implement the plugin execution port and handle plugin lifecycle (loading, initialization, execution, cleanup)
- **FR-016**: Logging infrastructure MUST use charmbracelet/log as the foundation and implement structured logging with consistent metadata
- **FR-017**: Infrastructure adapters MUST handle context cancellation gracefully and clean up resources (file handles, network connections, goroutines) on shutdown
- **FR-018**: TUI and CLI implementations MUST depend only on application layer use cases, never directly accessing domain entities or infrastructure

#### Wiring and Dependency Injection

- **FR-019**: System MUST wire all dependencies explicitly in cmd/streamy/main.go using constructor injection (no globals, no service locators)
- **FR-020**: Each infrastructure adapter MUST have a constructor that accepts its dependencies as interface parameters
- **FR-021**: Application services MUST have constructors that accept port interfaces, not concrete implementations
- **FR-022**: System MUST validate that all required dependencies are present at application startup and fail fast with clear error messages if any are missing
- **FR-023**: Dependency wiring MUST follow the direction: Infrastructure → Application → Domain (infrastructure depends on application; application depends on domain; domain depends on nothing)

#### Observability and Error Handling

- **FR-024**: System MUST generate correlation IDs at the entry point (CLI command invocation) and propagate them through all layers via context
- **FR-025**: All log entries MUST include correlation ID, layer name (domain/application/infrastructure), component name, and timestamp using charmbracelet/log format
- **FR-026**: Errors MUST preserve context as they bubble up through layers, adding layer-specific information without losing original error details
- **FR-027**: System MUST support structured error types that can be pattern-matched for specific handling (validation errors, execution errors, configuration errors)
- **FR-028**: Infrastructure layer MUST handle panic recovery at boundaries (plugin execution, goroutine workers) and convert panics to structured errors

### Key Entities

- **Pipeline**: Represents a complete configuration with metadata (name, version), steps, validations, and settings. Contains methods for dependency graph construction and validation.
- **Step**: Represents a single unit of work with unique ID, type, configuration payload, and dependency list. Validates its own invariants.
- **StepResult**: Captures the outcome of step execution including status (success/failure/skipped), duration, output messages, and error details.
- **ExecutionPlan**: Represents the DAG-based execution plan with level-based grouping for parallel execution. Contains methods for traversal and execution order calculation.
- **VerificationResult**: Captures the outcome of post-execution validation checks (command_exists, file_exists, path_contains) with status and diagnostic information.
- **ConfigurationRepository (port)**: Interface for loading and saving pipeline configurations. Implementations provide YAML parsing, file I/O, etc.
- **PluginExecutor (port)**: Interface for executing plugins against steps. Implementations provide plugin loading, execution, and lifecycle management.
- **ValidationService (port)**: Interface for running post-execution validation checks. Implementations provide specific check types.
- **EventPublisher (port)**: Interface for emitting domain and application events. Implementations provide logging, metrics, notifications, etc.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All domain entity tests run in under 100ms total without requiring any infrastructure setup (no files, no plugins, no network)
- **SC-002**: Application service tests achieve 90%+ code coverage using only mock implementations of port interfaces
- **SC-003**: Dependency wiring in main.go is explicit and type-checked at compile time (zero use of interface{} or reflection for DI)
- **SC-004**: All layers emit logs to charmbracelet/log with consistent structured metadata in at least 95% of log statements
- **SC-005**: Context cancellation causes graceful shutdown within 5 seconds across all integration test scenarios
- **SC-006**: Developers can add new step types by implementing infrastructure adapters without modifying domain or application layers (verified by adding a test plugin)
- **SC-007**: All existing integration tests pass with identical results after refactoring (no behavior changes, only structural changes)
- **SC-008**: Build time remains under 10 seconds and test suite execution time does not increase by more than 20%
- **SC-009**: Error messages include full context chain from domain through application to infrastructure in 100% of error scenarios tested
- **SC-010**: New developers can understand the architecture by reading the domain layer first without needing to understand infrastructure details (validated through documentation review)

## Clarifications

### Session 2025-10-15

- Q: How should the refactoring validate that behavior remains unchanged during incremental migration? → A: Implement Strangler Pattern with parallel implementations; compare outputs in production-like tests until confidence achieved
- Q: What is the maximum pipeline complexity (steps/dependencies) the architecture must support? → A: Up to 500 steps with 1000 dependencies - support complex CI/CD-like workflows
- Q: What happens when domain entities need to emit events but logging infrastructure hasn't been initialized yet? → A: Buffer events in memory during initialization; flush to logger once available
- Q: Are there external third-party plugins that must maintain compatibility, or are all plugins internal to the repository? → A: All plugins are internal - no external plugin compatibility required
- Q: How should the application layer handle multi-step pipeline operations when some steps fail midway? → A: Continue execution where possible; collect all errors; report aggregated failures at end

## Assumptions

1. **Backward Compatibility**: External behavior remains identical - all existing YAML configs, CLI commands, and integration tests work without modification (verified by SC-007)
2. **Migration Strategy**: Refactoring happens incrementally using Strangler Pattern - new architecture runs alongside existing code with output comparison in production-like tests until confidence is achieved, then old implementation is removed. Domain entities extracted first, then ports defined, then infrastructure adapters implemented, minimizing disruption
3. **Logging Migration**: Switching from `zerolog` to `charmbracelet/log` is acceptable and won't cause performance regressions for typical workloads (10-50 steps) or complex workflows (up to 500 steps with 1000 dependencies)
4. **No External Dependencies**: Refactoring stays within `internal/` - no changes to public `pkg/` APIs. **All plugins are internal to the repository** - no external plugin ecosystem exists as of 2025-10-15. There are zero known third-party plugins consuming Streamy's plugin API. The 7 built-in plugins (package, repo, symlink, copy, command, template, lineinfile) are all maintained in-tree and will migrate simultaneously. This eliminates external compatibility constraints and allows clean plugin interface redesign.
5. **Test Coverage**: Existing test coverage (~70-80% based on typical Go projects) is sufficient to validate refactoring doesn't break behavior
6. **Context Overhead**: Adding context propagation to all methods has negligible performance impact (<5% based on Go benchmarks)
7. **DI Approach**: Manual constructor injection (no frameworks like wire or dig) is acceptable given Streamy's moderate complexity
8. **Package Structure**: New structure creates `internal/domain/`, `internal/application/`, and `internal/infrastructure/` top-level packages while preserving existing `cmd/` and `pkg/`
9. **Port Naming**: Port interfaces named descriptively (ConfigurationLoader, PluginExecutor) rather than generic names (Repository, Service)
10. **Event Handling**: Domain events can be handled through simple observer pattern - no need for event bus infrastructure in initial refactor
