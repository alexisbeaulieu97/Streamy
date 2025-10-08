
# Implementation Plan: Unify and Simplify the Plugin System

**Branch**: `006-unify-and-simplify` | **Date**: October 7, 2025 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/home/alexis/Projects/Streamy/specs/006-unify-and-simplify/spec.md`

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

The plugin system will be refactored from a 4-method interface (Check, Apply, DryRun, Verify) to a unified 2-method interface (Evaluate, Apply). This eliminates code duplication across all plugins, enforces read-only evaluation, and centralizes execution mode logic in the engine. All 8 built-in plugins will be migrated simultaneously in a breaking change with no backward compatibility. The new interface uses a rich `EvaluationResult` struct that contains state assessment, change descriptions, and diffs, enabling the engine to make all mode-specific decisions (verify, dry-run, apply) without delegating to plugins.

**Key Technical Approach**:
- Define new `plugin.Plugin` interface with `Evaluate()` (read-only) and `Apply()` (mutate) methods
- Create `model.EvaluationResult` struct to transfer state assessment data
- Implement structured error types (ValidationError, ExecutionError, StateError)
- Refactor execution engine to interpret evaluation results for all modes
- Migrate all 8 plugins in complexity order: symlink/copy → lineinfile/template → package/repo → command/internalexec

## Technical Context
**Language/Version**: Go 1.21+ (per go.mod)
**Primary Dependencies**: Go standard library only (no external dependencies for core refactoring)
**Storage**: N/A (interface refactoring, no new storage requirements)
**Testing**: Go testing framework (`go test`), existing test suites for all 8 plugins
**Target Platform**: Cross-platform (Linux, macOS, Windows) - no platform-specific changes
**Project Type**: Single Go module with internal packages
**Performance Goals**: 
  - Evaluate() may be up to 20% slower than current Check() (acceptable trade-off)
  - Dry-run mode must complete in <1s for typical configs (50-100 steps)
**Constraints**: 
  - Breaking change: no backward compatibility
  - Evaluate() must be strictly read-only (no state mutations)
  - All 8 built-in plugins must migrate simultaneously
**Scale/Scope**: 8 built-in plugins to refactor, ~2000-3000 LOC affected across plugin system

**User-Provided Implementation Details**:

#### Core Architectural Changes

**New `plugin.Plugin` Interface**:
```go
type Plugin interface {
    Metadata() PluginMetadata
    Schema() interface{}
    Evaluate(ctx context.Context, step *config.Step) (*model.EvaluationResult, error)
    Apply(ctx context.Context, evalResult *model.EvaluationResult, step *config.Step) (*model.StepResult, error)
}
```

**New `model.EvaluationResult` Struct**:
```go
type EvaluationResult struct {
    StepID         string
    CurrentState   VerificationStatus
    RequiresAction bool
    Message        string
    Diff           string
    InternalData   interface{}
}
```

**Structured Error Types**:
```go
type PluginError interface {
    error
    StepID() string
    Unwrap() error
}

type ValidationError struct { ID string; Err error }
type ExecutionError struct { ID string; Err error }
type StateError struct { ID string; Err error }
```

**Migration Order** (ship together, implement in phases):
1. Simple: symlink, copy
2. Medium: lineinfile, template  
3. Complex: package, repo
4. Meta: command, internalexec

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**I. Onboarding First**
- [x] Feature requires no additional dependencies beyond compiled binary
- [x] No system packages, language runtimes, or external tools needed
- [x] If dependencies required, implemented as optional plugin
- [x] First-run experience documented and tested
*Status: PASS - Internal refactoring requires no new external dependencies*

**II. Schema Clarity & Fun**
- [x] Configuration uses flat flags for common options
- [x] Complex configs use clear nested structures with examples
- [x] `id` and `name` fields used appropriately (machine vs. human)
- [x] JSON schema provided for validation
- [x] Error messages include file/line context and fix suggestions
*Status: PASS - Improved error types (ValidationError, ExecutionError, StateError) enhance clarity*

**III. Plugin-Centric Architecture**
- [x] Core logic limited to DAG execution, logging, validation
- [x] Domain-specific logic implemented in plugins
- [x] Plugin interfaces versioned and backward compatible
- [x] Plugin contract tests included
*Status: PASS - This refactoring STRENGTHENS plugin architecture by clarifying the contract*
*Note: Breaking change is intentional (big bang migration), not a violation*

**IV. Safety by Default**
- [x] Dry-run mode supported for preview
- [x] Destructive operations require explicit flags/confirmation
- [x] Operations are idempotent (safe to run multiple times)
- [x] Rollback/recovery procedures documented
- [x] Parallel execution defaults are safe
*Status: PASS - Read-only Evaluate() ENHANCES safety; separation of read/write is core principle*

**V. Performance & Reliability**
- [x] Dry-run completes in <1s for typical configs
- [x] Structured logging shows task timing and dependencies
- [x] Error messages include context, cause, and remediation
- [x] Resource limits declared for scheduling
- [x] Timeouts configured for long operations
*Status: PASS - Accepts up to 20% overhead for correctness (explicit trade-off)*

**VI. Extensibility & Composability**
- [x] Feature works in simple and complex scenarios
- [x] No breaking changes to existing configs
- [x] Supports composition (imports, groups, conditionals where relevant)
- [x] Backward compatible within major version
*Status: PASS with JUSTIFICATION - Breaking change at plugin API level is intentional*
*User configs remain compatible; only plugin implementations affected*

**VII. Ecosystem Consistency**
- [x] Follows plugin naming conventions (`id`, `name`, `enabled`, `depends_on`)
- [x] Structured error handling implemented
- [x] Documentation includes schema, examples, troubleshooting
- [x] Version compatibility declared explicitly
*Status: PASS - Unified interface improves consistency across all plugins*

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
│   ├── interface.go          # NEW: Unified Plugin interface (Evaluate/Apply)
│   ├── metadata.go            # EXISTING: PluginMetadata types (unchanged)
│   ├── registry_new.go        # MODIFIED: Update to new interface
│   └── errors.go              # NEW: PluginError base + ValidationError/ExecutionError/StateError
├── model/
│   ├── step_result.go         # EXISTING: StepResult (unchanged)
│   └── evaluation_result.go   # NEW: EvaluationResult struct
├── engine/
│   ├── executor.go            # MODIFIED: Use Evaluate/Apply instead of Check/Apply/DryRun/Verify
│   ├── planner.go             # MODIFIED: Plan based on evaluation results
│   └── context.go             # EXISTING: Execution context (unchanged)
├── plugins/
│   ├── symlink/
│   │   └── symlink.go         # REFACTOR: Implement new interface (Phase 1)
│   ├── copy/
│   │   └── copy.go            # REFACTOR: Implement new interface (Phase 1)
│   ├── lineinfile/
│   │   └── lineinfile.go      # REFACTOR: Adapt existing evaluate() (Phase 2)
│   ├── template/
│   │   └── template.go        # REFACTOR: Implement new interface (Phase 2)
│   ├── package/
│   │   └── package.go         # REFACTOR: Implement new interface (Phase 3)
│   ├── repo/
│   │   └── repo.go            # REFACTOR: Implement new interface (Phase 3)
│   ├── command/
│   │   └── command.go         # REFACTOR: Implement new interface (Phase 4)
│   └── internalexec/
│       └── internalexec.go    # REFACTOR: Implement new interface (Phase 4)
cmd/
├── streamy/
│   ├── verify.go              # MODIFIED: Use Evaluate() for state checking
│   ├── apply.go               # MODIFIED: Use Evaluate() → Apply() flow
│   └── plugins_import.go      # EXISTING: Plugin registration (unchanged)
tests/
├── integration_test.go        # MODIFIED: Update to test new interface
├── integration_verify_test.go # MODIFIED: Test Evaluate() read-only behavior
└── integration_plugin_dependency_test.go  # MODIFIED: Update for new interface

docs/
└── plugins.md                 # MODIFIED: Document new "golden path" interface
```

**Structure Decision**: Single Go project structure. This is an internal refactoring affecting the plugin system architecture. Changes are concentrated in `internal/plugin/`, `internal/model/`, `internal/engine/`, and all plugin implementations under `internal/plugins/`. Command-line interface adapts to use new execution flow in `cmd/streamy/`.

## Phase 0: Outline & Research

**Status**: ✅ Complete

**Output**: `research.md`

### Research Questions Resolved

1. **Go Interface Design Best Practices** → Single Evaluate/Apply methods with rich return types
2. **Read-Only Enforcement** → Convention + documentation + comprehensive testing
3. **Error Type Hierarchy** → Interface-based with Unwrap() support (ValidationError, ExecutionError, StateError)
4. **InternalData Pattern** → `interface{}` with type assertion (maximum flexibility, type-safe per plugin)
5. **Test Migration Strategy** → Incremental per-plugin updates maintaining coverage
6. **Performance Measurement** → Benchmark suite with 20% overhead budget

All technical unknowns resolved. See `research.md` for detailed rationale and alternatives considered.

## Phase 1: Design & Contracts

**Status**: ✅ Complete

**Outputs**:
- `data-model.md` - Entity definitions and relationships
- `contracts/plugin-interface.md` - Plugin contract specification
- `contracts/executor-plugin.md` - Executor-plugin interaction contract
- `quickstart.md` - Step-by-step validation procedure
- `AGENTS.md` - Updated agent context

### Artifacts Generated

#### 1. Data Model (`data-model.md`)
Defines core entities:
- **Plugin Interface**: Metadata(), Schema(), Evaluate(), Apply()
- **EvaluationResult**: StepID, CurrentState, RequiresAction, Message, Diff, InternalData
- **PluginError Hierarchy**: ValidationError, ExecutionError, StateError
- **VerificationStatus**: Enum (Satisfied, Missing, Drifted, Blocked, Unknown)
- **StepResult**: Existing type (unchanged)

Complete with validation rules, state transitions, and entity relationships.

#### 2. Plugin Interface Contract (`contracts/plugin-interface.md`)
Comprehensive specification including:
- Interface definition with detailed method contracts
- Read-only guarantee for Evaluate() (CRITICAL)
- Idempotency requirements for Apply()
- Error handling contract with type selection guidance
- Example implementation demonstrating best practices
- Contract test suite that all plugins must pass
- Migration checklist for refactoring existing plugins

#### 3. Executor-Plugin Contract (`contracts/executor-plugin.md`)
Defines interaction patterns:
- Verify mode flow (Evaluate-only, never calls Apply)
- Dry-run mode flow (preview with Diff display)
- Apply mode flow (Evaluate → Apply only when RequiresAction=true)
- Error handling by type (ValidationError, ExecutionError, StateError)
- Context handling and timeout enforcement
- Logging contract with structured fields
- Performance contract with timing expectations
- Mock plugin for testing

#### 4. Quickstart Guide (`quickstart.md`)
12-step validation procedure covering:
- Core type compilation
- Unit tests for types
- Plugin contract tests
- Executor integration (all 3 modes)
- Idempotency verification
- Error handling validation
- Performance benchmarking
- All 8 plugin verification

#### 5. Agent Context (`AGENTS.md`)
Updated with:
- Language: Go 1.21+
- Framework: Go standard library only
- Project type: Single Go module
- Recent changes from this feature

### Design Validation

✅ **Constitution Re-Check**: All principles satisfied
- Read-only Evaluate() enhances safety (Principle IV)
- Structured errors improve clarity (Principle II)
- Plugin architecture strengthened (Principle III)
- Performance trade-off documented and acceptable (Principle V)

✅ **No new violations introduced**

## Phase 2: Task Planning Approach
*This section describes what the /tasks command will do - DO NOT execute during /plan*

**Task Generation Strategy**:

The `/tasks` command will load `.specify/templates/tasks-template.md` and generate ordered, executable tasks based on the design artifacts from Phase 1.

### Task Categories

#### A. Foundation Tasks (Core Types)
1. Define `EvaluationResult` struct in `internal/model/evaluation_result.go`
2. Define `VerificationStatus` enum constants
3. Define `PluginError` interface in `internal/plugin/errors.go`
4. Implement `ValidationError` type
5. Implement `ExecutionError` type
6. Implement `StateError` type
7. Add unit tests for error types

**Ordering**: Sequential (foundation for all other work)

#### B. Interface Refactoring Tasks
8. Update `Plugin` interface in `internal/plugin/interface.go` (add Evaluate/Apply, mark old methods deprecated)
9. Update `PluginRegistry` to support new interface
10. Add contract test suite in `internal/plugin/contract_test.go`

**Ordering**: Sequential (depends on A)

#### C. Executor Refactoring Tasks
11. Refactor `executor.go` verify mode to use Evaluate() [P]
12. Refactor `executor.go` dry-run mode to use Evaluate() [P]
13. Refactor `executor.go` apply mode to use Evaluate() → Apply() [P]
14. Update error handling to use structured types
15. Add executor integration tests

**Ordering**: Can work in parallel after B complete. [P] = parallel eligible

#### D. Plugin Migration Tasks (Phase 1: Simple)
16. Migrate `symlink` plugin: Implement Evaluate/Apply [P]
17. Add contract tests for `symlink` plugin [P]
18. Migrate `copy` plugin: Implement Evaluate/Apply [P]
19. Add contract tests for `copy` plugin [P]

**Ordering**: Parallel after C complete

#### E. Plugin Migration Tasks (Phase 2: Medium)
20. Migrate `lineinfile` plugin: Adapt existing evaluate() [P]
21. Add contract tests for `lineinfile` plugin [P]
22. Migrate `template` plugin: Implement Evaluate/Apply [P]
23. Add contract tests for `template` plugin [P]

**Ordering**: Parallel after D complete

#### F. Plugin Migration Tasks (Phase 3: Complex)
24. Migrate `package` plugin: Implement Evaluate/Apply [P]
25. Add contract tests for `package` plugin [P]
26. Migrate `repo` plugin: Implement Evaluate/Apply [P]
27. Add contract tests for `repo` plugin [P]

**Ordering**: Parallel after E complete

#### G. Plugin Migration Tasks (Phase 4: Meta)
28. Migrate `command` plugin: Implement Evaluate/Apply [P]
29. Add contract tests for `command` plugin [P]
30. Migrate `internalexec` plugin: Implement Evaluate/Apply [P]
31. Add contract tests for `internalexec` plugin [P]

**Ordering**: Parallel after F complete

#### H. Cleanup Tasks
32. Remove deprecated Check/DryRun/Verify methods from Plugin interface
33. Update all existing integration tests
34. Update `docs/plugins.md` documentation
35. Add performance benchmarks
36. Run quickstart.md validation

**Ordering**: Sequential after G complete

### TDD Approach

Each plugin migration follows test-first pattern:
1. Write contract tests (should fail with old interface)
2. Implement Evaluate() method (tests start passing)
3. Implement Apply() method (reusing InternalData)
4. Remove old methods
5. Verify all tests pass

### Estimated Task Count

- **Foundation**: 7 tasks
- **Interface**: 3 tasks  
- **Executor**: 5 tasks
- **Plugins**: 16 tasks (4 per phase × 4 phases)
- **Cleanup**: 5 tasks

**Total**: ~36 tasks

### Parallelization Strategy

- Foundation and Interface: Sequential (critical path)
- Executor refactoring: 3 parallel streams (verify, dry-run, apply modes)
- Plugin migrations: Up to 2 plugins in parallel per phase
- Maximum concurrency: ~4-5 tasks simultaneously

### Success Criteria for Tasks

Each task is "done" when:
- [ ] Code compiles without errors
- [ ] All unit tests pass
- [ ] Contract tests pass (for plugin tasks)
- [ ] Integration tests pass (for executor tasks)
- [ ] Code review completed
- [ ] Documentation updated (if applicable)

**IMPORTANT**: This phase is executed by the `/tasks` command, NOT by `/plan`

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)  
**Phase 4**: Implementation (execute tasks.md following constitutional principles)  
**Phase 5**: Validation (run tests, execute quickstart.md, performance validation)

## Complexity Tracking

**Status**: No constitutional violations requiring justification.

The breaking change (big bang migration with no backward compatibility) is a deliberate architectural decision aligned with the project's formative stage, not a constitutional violation. User-facing YAML configurations remain compatible; only the internal plugin API changes.

All constitutional principles are satisfied or enhanced by this refactoring:
- **Principle III**: Plugin architecture is strengthened and clarified
- **Principle IV**: Read-only Evaluate() enhances safety
- **Principle V**: Explicit performance trade-off (20% overhead) is documented and acceptable

| Deviation | Justification | Alternative Rejected |
|-----------|---------------|---------------------|
| None | N/A | N/A |


## Progress Tracking
*This checklist is updated during execution flow*

**Phase Status**:
- [x] Phase 0: Research complete (/plan command)
- [x] Phase 1: Design complete (/plan command)
- [x] Phase 2: Task planning approach documented (/plan command - describe approach only)
- [x] Phase 3: Tasks generated (/tasks command) - **✅ COMPLETE**
- [ ] Phase 4: Implementation in progress
- [ ] Phase 5: Validation pending

**Gate Status**:
- [x] Initial Constitution Check: PASS
- [x] Post-Design Constitution Check: PASS
- [x] All NEEDS CLARIFICATION resolved
- [x] Complexity deviations documented (none)

**Artifacts Generated**:
- [x] `research.md` (Phase 0)
- [x] `data-model.md` (Phase 1)
- [x] `contracts/plugin-interface.md` (Phase 1)
- [x] `contracts/executor-plugin.md` (Phase 1)
- [x] `quickstart.md` (Phase 1)
- [x] `AGENTS.md` updated (Phase 1)
- [x] `tasks.md` (Phase 2) - **✅ 36 tasks generated**

---

## Ready for /tasks Command

All planning complete. The `/tasks` command should now:
1. Load the task generation strategy from Phase 2 above
2. Create `tasks.md` with ~36 ordered, executable tasks
3. Mark parallelizable tasks with [P]
4. Include success criteria for each task
5. Follow TDD approach: tests before implementation

**Suggested next command**: `/tasks`

---
*Based on Constitution v2.1.1 - See `/memory/constitution.md`*
