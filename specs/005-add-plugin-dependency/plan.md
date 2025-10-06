
# Implementation Plan: Plugin Dependency Registry for Composable Plugins

**Branch**: `005-add-plugin-dependency` | **Date**: October 6, 2025 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/home/alexis/Projects/Streamy/specs/005-add-plugin-dependency/spec.md`

## Execution Flow (/plan command scope)
```
1. Load feature spec from Input path
   → If not found: ERROR "No feature spec at {path}"
2. Fill Technical Context (scan for NEEDS CLARIFICATION)
   → Detect Project Type from file system structure or context (web=frontend+backend, mobile=app+api)
   → Set Structure Decision based on project type
3. Fill the Constitution Check section based on the content of the constitution document.
4. Evaluate Constitution Check section below
   → If violations exist: Document in Complexity Tracking
   → If no justification possible: ERROR "Simplify approach first"
   → Update Progress Tracking: Initial Constitution Check
5. Execute Phase 0 → research.md
   → If NEEDS CLARIFICATION remain: ERROR "Resolve unknowns"
6. Execute Phase 1 → contracts, data-model.md, quickstart.md, agent-specific template file (e.g., `CLAUDE.md` for Claude Code, `.github/copilot-instructions.md` for GitHub Copilot, `GEMINI.md` for Gemini CLI, `QWEN.md` for Qwen Code, or `AGENTS.md` for all other agents).
7. Re-evaluate Constitution Check section
   → If new violations: Refactor design, return to Phase 1
   → Update Progress Tracking: Post-Design Constitution Check
8. Plan Phase 2 → Describe task generation approach (DO NOT create tasks.md)
9. STOP - Ready for /tasks command
```

**IMPORTANT**: The /plan command STOPS at step 7. Phases 2-4 are executed by other commands:
- Phase 2: /tasks command creates tasks.md
- Phase 3-4: Implementation execution (manual or via tools)

## Summary
Implement a centralized PluginRegistry system that enables composable plugin architecture. Plugins can declare dependencies on other plugins through metadata, allowing reuse of functionality (e.g., a `shell_profile` plugin can depend on `line_in_file` plugin). The registry validates dependency graphs, detects circular dependencies, enforces version constraints using major version matching (`1.x`, `2.x`), and manages plugin initialization order. Environment-aware failure policies ensure strict validation in CI/automation contexts and graceful degradation in interactive CLI/TUI usage.

## Technical Context
**Language/Version**: Go 1.21+  
**Primary Dependencies**: None (zero external dependencies per Constitution Principle I)  
**Storage**: In-memory registry, no persistent state  
**Testing**: Go standard testing package, table-driven tests for dependency resolution  
**Target Platform**: Linux, macOS, Windows (cross-platform Go binary)  
**Project Type**: Single project - core Streamy functionality  
**Performance Goals**: 
- Dependency resolution: O(n) where n = number of plugins (topological sort)
- Dry-run validation: <50ms for typical plugin graphs (10-20 plugins)
- Zero runtime overhead after initialization
**Constraints**: 
- Stateless registry (no persistence between runs)
- Thread-safe for concurrent plugin lookups
- Backward compatible with existing plugins (graceful migration)
**Scale/Scope**: 
- Expected plugin count: 10-30 built-in, potentially 50+ with external plugins
- Circular dependency detection required
- Transitive dependencies resolved automatically

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**I. Onboarding First**
- [x] Feature requires no additional dependencies beyond compiled binary
- [x] No system packages, language runtimes, or external tools needed
- [x] If dependencies required, implemented as optional plugin (N/A - pure Go internal feature)
- [x] First-run experience documented and tested (transparent to users, plugins declare dependencies in metadata)

**II. Schema Clarity & Fun**
- [x] Configuration uses flat flags for common options (plugin metadata is simple struct)
- [x] Complex configs use clear nested structures with examples (Dependencies array in metadata)
- [x] `id` and `name` fields used appropriately (plugin Name serves as unique identifier)
- [x] JSON schema provided for validation (plugin metadata validated at registration)
- [x] Error messages include file/line context and fix suggestions (dependency errors identify missing/circular plugins)

**III. Plugin-Centric Architecture**
- [x] Core logic limited to DAG execution, logging, validation (registry is core infrastructure)
- [x] Domain-specific logic implemented in plugins (registry enables plugins to use other plugins)
- [x] Plugin interfaces versioned and backward compatible (Init() method added, old plugins still work)
- [x] Plugin contract tests included (dependency resolution tests validate interface)

**IV. Safety by Default**
- [x] Dry-run mode supported for preview (dependency validation happens at startup before any operations)
- [x] Destructive operations require explicit flags/confirmation (N/A - registry is read-only lookup)
- [x] Operations are idempotent (registry initialization is deterministic)
- [x] Rollback/recovery procedures documented (startup failures prevent execution)
- [x] Parallel execution defaults are safe (plugins initialized in dependency order)

**V. Performance & Reliability**
- [x] Dry-run completes in <1s for typical configs (dependency resolution <50ms)
- [x] Structured logging shows task timing and dependencies (registry logs initialization order)
- [x] Error messages include context, cause, and remediation (missing/circular dependency errors are detailed)
- [x] Resource limits declared for scheduling (O(n) memory, single-pass validation)
- [x] Timeouts configured for long operations (N/A - validation is synchronous and fast)

**VI. Extensibility & Composability**
- [x] Feature works in simple and complex scenarios (plugins can have 0 to many dependencies)
- [x] No breaking changes to existing configs (backward compatible, existing plugins work without metadata)
- [x] Supports composition (imports, groups, conditionals where relevant) (enables plugin composition by design)
- [x] Backward compatible within major version (graceful migration with warnings)

**VII. Ecosystem Consistency**
- [x] Follows plugin naming conventions (`id`, `name`, `enabled`, `depends_on`) (uses Name, Dependencies fields)
- [x] Structured error handling implemented (typed errors: ErrPluginNotFound, ErrCircularDependency, ErrVersionConflict)
- [x] Documentation includes schema, examples, troubleshooting (will include in docs/plugins.md)
- [x] Version compatibility declared explicitly (PluginMetadata.Version and APIVersion fields)

## Project Structure

### Documentation (this feature)
```
specs/[###-feature]/
├── plan.md              # This file (/plan command output)
├── research.md          # Phase 0 output (/plan command)
├── data-model.md        # Phase 1 output (/plan command)
├── quickstart.md        # Phase 1 output (/plan command)
├── contracts/           # Phase 1 output (/plan command)
└── tasks.md             # Phase 2 output (/tasks command - NOT created by /plan)
```

### Source Code (repository root)
```
internal/
├── plugin/
│   ├── interface.go          # Existing plugin interface
│   ├── registry.go            # NEW: PluginRegistry implementation
│   ├── registry_test.go       # NEW: Registry unit tests
│   ├── metadata.go            # NEW: PluginMetadata, VersionConstraint types
│   ├── dependency_graph.go    # NEW: Dependency resolution, cycle detection
│   ├── dependency_graph_test.go # NEW: Graph algorithm tests
│   └── errors.go              # NEW: Typed errors (ErrPluginNotFound, etc.)
├── plugins/
│   ├── command/
│   ├── copy/
│   ├── lineinfile/
│   └── [other plugins]        # UPDATED: Add Init() method, declare Dependencies
cmd/
├── streamy/
│   └── main.go                # UPDATED: Initialize registry, inject into plugins
docs/
├── plugins.md                 # UPDATED: Document dependency system
└── architecture.md            # UPDATED: Registry design documentation
tests/
├── integration_test.go        # UPDATED: Test plugin composition scenarios
└── plugin_dependency_test.go  # NEW: End-to-end dependency tests
```

**Structure Decision**: Single project structure. The PluginRegistry is core infrastructure living in `internal/plugin/` alongside the existing plugin interface. All changes are internal to Streamy's core, with updates to existing plugins to support optional dependency declaration. No new top-level directories needed.

## Phase 0: Outline & Research
1. **Extract unknowns from Technical Context** above:
   - Dependency graph algorithm choice → Research Kahn's vs DFS topological sort
   - Version constraint syntax → Research semver matching patterns
   - Plugin state isolation strategy → Research singleton vs per-dependent instances
   - Failure policy defaults → Research environment detection patterns
   - Thread safety patterns → Research Go sync primitives for registries

2. **Generate and dispatch research agents**:
   - Task: "Research topological sort algorithms for dependency resolution"
   - Task: "Research version constraint patterns for plugin compatibility"
   - Task: "Research state isolation strategies for shared plugin instances"
   - Task: "Research environment-aware configuration defaults in Go"
   - Task: "Research thread-safe registry patterns in Go"

3. **Consolidate findings** in `research.md` using format:
   - Decision: Kahn's algorithm for topological sorting
   - Decision: Major version matching (`X.x` format)
   - Decision: Configurable state isolation via metadata flag
   - Decision: Environment-aware dependency policy
   - Decision: RWMutex for concurrent registry access

**Output**: ✅ research.md complete - All technical decisions documented with rationale

## Phase 1: Design & Contracts
*Prerequisites: research.md complete*

1. **Extract entities from feature spec** → `data-model.md`:
   - ✅ PluginRegistry: Central repository with dependency resolution
   - ✅ PluginMetadata: Plugin information with dependencies
   - ✅ Dependency: Dependency declaration with version constraints
   - ✅ VersionConstraint: Major version matching specification
   - ✅ DependencyGraph: DAG for cycle detection and topological sorting
   - ✅ RegistryConfig: Configuration for policies
   - ✅ Error types: Typed errors for all failure scenarios

2. **Generate API contracts** from functional requirements:
   - ✅ PluginRegistry interface: Register, ValidateDependencies, InitializePlugins, Get, GetForDependent
   - ✅ PluginInitializer interface: Optional Init() for dependency access
   - ✅ Metadata and constraint types documented
   - ✅ Error types with remediation hints
   - ✅ Thread safety model documented

3. **Generate contract tests** from contracts:
   - ✅ Test: Register and Get
   - ✅ Test: Missing Dependency Detection
   - ✅ Test: Circular Dependency Detection  
   - ✅ Test: Initialization Order
   - ✅ Tests defined in contracts/registry-api.md (implementation in Phase 3)

4. **Extract test scenarios** from user stories:
   - ✅ Scenario: Shell profile plugin depends on line_in_file
   - ✅ Scenario: Transitive dependencies (A → B → C)
   - ✅ Scenario: Environment-aware policy modes
   - ✅ Scenario: Backward compatibility with legacy plugins
   - ✅ Quickstart guide with complete working example

5. **Update agent file incrementally**:
   - ✅ Executed update-agent-context.sh codex
   - ✅ Added Go 1.21+ language context
   - ✅ Added project type and dependencies
   - ✅ AGENTS.md updated with feature context

**Output**: ✅ data-model.md, contracts/registry-api.md, quickstart.md, AGENTS.md updated

## Phase 2: Task Planning Approach
*This section describes what the /tasks command will do - DO NOT execute during /plan*

**Task Generation Strategy**:
- Load `.specify/templates/tasks-template.md` as base
- Generate tasks from Phase 1 design docs (contracts, data model, quickstart)
- Follow TDD approach: tests before implementation
- Group tasks into logical phases:
  1. **Core Infrastructure**: Data types, error types, basic registry structure
  2. **Dependency Graph**: Cycle detection, topological sort algorithms
  3. **Registry Operations**: Register, validate, initialize, lookup methods
  4. **Policy Enforcement**: Environment detection, policy-based validation, access control
  5. **Plugin Interface Updates**: Add Init() support, update existing plugins
  6. **Integration**: Wire into Streamy main, end-to-end testing
  7. **Documentation**: Update docs/plugins.md and architecture.md

**Task Ordering Strategy**:
- **Test-First**: Write contract tests before implementation (fail initially)
- **Dependency Order**: 
  1. Error types (no dependencies)
  2. Data structures (PluginMetadata, Dependency, VersionConstraint)
  3. DependencyGraph (uses data structures)
  4. PluginRegistry (uses graph and data structures)
  5. Policy logic (uses registry)
  6. Plugin updates (uses registry)
  7. Integration (uses everything)
- **Parallelizable Tasks** marked [P]: Independent files can be edited simultaneously
  - Error types [P]
  - Data structure tests [P]
  - Documentation updates [P]

**Estimated Task Breakdown**:
1. **Setup & Data Types** (5-7 tasks): Error types, metadata structs, version constraints
2. **Dependency Graph** (4-5 tasks): Graph structure, cycle detection, topological sort
3. **Registry Core** (6-8 tasks): Register, validate, initialize, lookup methods
4. **Policy System** (3-4 tasks): Config detection, policy enforcement, access control
5. **Plugin Integration** (4-5 tasks): Update interface, migrate existing plugins
6. **Testing** (5-6 tasks): Unit tests, integration tests, contract validation
7. **Documentation** (3-4 tasks): API docs, architecture docs, migration guide

**Total Estimated Tasks**: 30-35 numbered, ordered tasks in tasks.md

**IMPORTANT**: This phase is executed by the /tasks command, NOT by /plan

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)  
**Phase 4**: Implementation (execute tasks.md following constitutional principles)  
**Phase 5**: Validation (run tests, execute quickstart.md, performance validation)

## Complexity Tracking
*No constitutional violations - all checks passed*

The Plugin Dependency Registry implementation fully aligns with Streamy's constitutional principles:
- Zero external dependencies (pure Go implementation)
- Simple, clear configuration (plugin metadata)
- Plugin-centric architecture (registry is core infrastructure enabling plugins)
- Safety by default (validation before execution, environment-aware policies)
- Performance-aware (O(n) algorithms, <50ms validation)
- Backward compatible (graceful migration for existing plugins)
- Consistent patterns (follows existing plugin interface conventions)

No complexity deviations to document.


## Progress Tracking
*This checklist is updated during execution flow*

**Phase Status**:
- [x] Phase 0: Research complete (/plan command)
- [x] Phase 1: Design complete (/plan command)
- [x] Phase 2: Task planning complete (/plan command - describe approach only)
- [x] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [x] Initial Constitution Check: PASS
- [x] Post-Design Constitution Check: PASS
- [x] All NEEDS CLARIFICATION resolved (via /clarify)
- [x] Complexity deviations documented (none required)

---
*Based on Constitution v2.1.1 - See `/memory/constitution.md`*
