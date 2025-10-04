
# Implementation Plan: line_in_file Plugin

**Branch**: `003-add-built-in` | **Date**: October 4, 2025 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/home/alexis/Projects/Streamy/specs/003-add-built-in/spec.md`

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
Implement a new built-in plugin `line_in_file` that provides idempotent, declarative management of text file lines. The plugin ensures specific lines exist or are removed in target files through pattern matching, supports backup creation, handles multiple encoding formats, and integrates fully with Streamy's dry-run and verbose modes. Core operations include appending lines, replacing matched patterns, removing lines, and creating atomic file modifications with optional backup functionality.

## Technical Context
**Language/Version**: Go 1.21+ (existing Streamy codebase)
**Primary Dependencies**: 
- Standard library: `os`, `bufio`, `regexp`, `io`, `path/filepath`, `time`
- Encoding support: `golang.org/x/text/encoding` for configurable encoding
- Existing: `internal/plugin` (Plugin interface), `internal/engine` (DAG execution), `internal/logger`
**Storage**: File system operations (read/write text files)
**Testing**: Go testing (`go test`), table-driven tests, integration tests
**Target Platform**: Cross-platform (Linux, macOS, Windows)
**Project Type**: Single Go project (plugin addition to existing codebase)
**Performance Goals**: 
- Handle files of any size without memory limits (streaming where possible)
- Dry-run preview generation <100ms for typical config files (<10MB)
- Regex compilation cached per step execution
**Constraints**: 
- Must preserve file permissions and ownership
- Atomic writes via temp file + rename pattern
- No external dependencies beyond Go stdlib and x/text
- Interactive prompts must work in TTY and non-TTY contexts
**Scale/Scope**: 
- Plugin field count: ~8-10 configuration fields
- Core logic: ~500-800 lines
- Test coverage: >85%
- Support files from 0 bytes to multi-GB

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**I. Onboarding First**
- [x] Feature requires no additional dependencies beyond compiled binary (uses Go stdlib + x/text)
- [x] No system packages, language runtimes, or external tools needed
- [x] If dependencies required, implemented as optional plugin (N/A - built-in plugin)
- [x] First-run experience documented and tested (part of core binary)

**II. Schema Clarity & Fun**
- [x] Configuration uses flat flags for common options (`file`, `line`, `state`, `match`, `backup`)
- [x] Complex configs use clear nested structures with examples (optional fields well-documented)
- [x] `id` and `name` fields used appropriately (machine vs. human) (follows step conventions)
- [x] JSON schema provided for validation (will be generated for plugin config)
- [x] Error messages include file/line context and fix suggestions (regex validation, permission errors)

**III. Plugin-Centric Architecture**
- [x] Core logic limited to DAG execution, logging, validation (this IS a plugin)
- [x] Domain-specific logic implemented in plugins (file editing is domain-specific)
- [x] Plugin interfaces versioned and backward compatible (implements existing Plugin interface)
- [x] Plugin contract tests included (will test Execute, Validate, DryRun methods)

**IV. Safety by Default**
- [x] Dry-run mode supported for preview (FR-012: integrates with --dry-run)
- [x] Destructive operations require explicit flags/confirmation (`state: absent` requires `match` pattern, FR-008)
- [x] Operations are idempotent (safe to run multiple times) (FR-011: core requirement)
- [x] Rollback/recovery procedures documented (backup files with FR-010, atomic writes)
- [x] Parallel execution defaults are safe (DAG handles dependencies, file locks if needed)

**V. Performance & Reliability**
- [x] Dry-run completes in <1s for typical configs (<10MB files, <100ms target)
- [x] Structured logging shows task timing and dependencies (uses internal/logger)
- [x] Error messages include context, cause, and remediation (FR-014, FR-018: permission, regex errors)
- [x] Resource limits declared for scheduling (memory scales with file size, streaming approach)
- [x] Timeouts configured for long operations (will use context.Context for cancellation)

**VI. Extensibility & Composability**
- [x] Feature works in simple and complex scenarios (append, replace, remove, backup, encoding)
- [x] No breaking changes to existing configs (new plugin type, no changes to existing plugins)
- [x] Supports composition (imports, groups, conditionals where relevant) (DAG dependency via depends_on)
- [x] Backward compatible within major version (initial implementation, will version plugin API)

**VII. Ecosystem Consistency**
- [x] Follows plugin naming conventions (`id`, `name`, `enabled`, `depends_on`) (reuses Step fields)
- [x] Structured error handling implemented (will use pkg/errors for wrapped context)
- [x] Documentation includes schema, examples, troubleshooting (quickstart.md will include)
- [x] Version compatibility declared explicitly (Streamy version requirement in plugin metadata)

## Project Structure

### Documentation (this feature)
```
specs/003-add-built-in/
├── plan.md              # This file (/plan command output)
├── research.md          # Phase 0 output (/plan command)
├── data-model.md        # Phase 1 output (/plan command)
├── quickstart.md        # Phase 1 output (/plan command)
├── contracts/           # Phase 1 output (/plan command)
│   └── plugin-interface.md  # LineInFile plugin contract
└── tasks.md             # Phase 2 output (/tasks command - NOT created by /plan)
```

### Source Code (repository root)
```
internal/
└── plugins/
    └── lineinfile/
        ├── lineinfile.go      # Main plugin implementation
        ├── lineinfile_test.go # Unit tests
        ├── config.go          # Configuration struct and validation
        ├── file_ops.go        # File operations (read, write, backup)
        ├── matcher.go         # Regex matching and line manipulation
        └── differ.go          # Dry-run diff generation

cmd/
└── streamy/
    └── plugins_import.go  # Register line_in_file plugin

testdata/
└── lineinfile/
    ├── simple.txt         # Test fixture files
    ├── multiline.txt
    ├── utf8.txt
    └── latin1.txt

tests/
└── integration_test.go    # Add line_in_file integration tests
```

**Structure Decision**: Single Go project structure. This is a new built-in plugin addition to the existing Streamy codebase. Following the established pattern in `internal/plugins/` (command, copy, package, repo, symlink, template), the new `lineinfile` package will be added alongside existing plugins. The plugin will be registered in `cmd/streamy/plugins_import.go` to make it available as a built-in plugin type.

## Phase 0: Outline & Research
1. **Extract unknowns from Technical Context** above:
   - For each NEEDS CLARIFICATION → research task
   - For each dependency → best practices task
   - For each integration → patterns task

2. **Generate and dispatch research agents**:
   ```
   For each unknown in Technical Context:
     Task: "Research {unknown} for {feature context}"
   For each technology choice:
     Task: "Find best practices for {tech} in {domain}"
   ```

3. **Consolidate findings** in `research.md` using format:
   - Decision: [what was chosen]
   - Rationale: [why chosen]
   - Alternatives considered: [what else evaluated]

**Output**: research.md with all NEEDS CLARIFICATION resolved

## Phase 1: Design & Contracts
*Prerequisites: research.md complete*

1. **Extract entities from feature spec** → `data-model.md`:
   - Entity name, fields, relationships
   - Validation rules from requirements
   - State transitions if applicable

2. **Generate API contracts** from functional requirements:
   - For each user action → endpoint
   - Use standard REST/GraphQL patterns
   - Output OpenAPI/GraphQL schema to `/contracts/`

3. **Generate contract tests** from contracts:
   - One test file per endpoint
   - Assert request/response schemas
   - Tests must fail (no implementation yet)

4. **Extract test scenarios** from user stories:
   - Each story → integration test scenario
   - Quickstart test = story validation steps

5. **Update agent file incrementally** (O(1) operation):
   - Run `.specify/scripts/bash/update-agent-context.sh codex`
     **IMPORTANT**: Execute it exactly as specified above. Do not add or remove any arguments.
   - If exists: Add only NEW tech from current plan
   - Preserve manual additions between markers
   - Update recent changes (keep last 3)
   - Keep under 150 lines for token efficiency
   - Output to repository root

**Output**: data-model.md, /contracts/*, failing tests, quickstart.md, agent-specific file

## Phase 2: Task Planning Approach
*This section describes what the /tasks command will do - DO NOT execute during /plan*

**Task Generation Strategy**:

The `/tasks` command will generate a comprehensive, ordered task list following TDD principles and constitutional guidelines. Tasks will be derived from:

1. **Plugin Contract Tests** (from `contracts/plugin-interface.md`):
   - Contract test for `Name()` method
   - Contract tests for `Validate()` with all error cases (8+ scenarios)
   - Contract tests for `Execute()` covering all acceptance scenarios (20+ scenarios)
   - Contract tests for `DryRun()` with diff validation (10+ scenarios)
   - **Parallel**: [P] - Each test file independent

2. **Data Model Implementation** (from `data-model.md`):
   - Create `config.go` with LineInFileConfig struct and validation
   - Create `file_ops.go` with FileState operations (read, write, backup)
   - Create `matcher.go` with MatchResult and regex logic
   - Create `differ.go` with ChangeSet and diff generation
   - **Parallel**: [P] - Each file independent after shared types defined

3. **Plugin Core Implementation** (from `contracts/plugin-interface.md`):
   - Create `lineinfile.go` with Plugin interface implementation
   - Implement `Name()` method (trivial)
   - Implement `Validate()` method with all validation rules
   - Implement `Execute()` method with atomic write pattern
   - Implement `DryRun()` method with diff preview
   - **Sequential**: Must follow data model tasks

4. **Integration with Streamy Core**:
   - Register plugin in `cmd/streamy/plugins_import.go`
   - Add integration tests in `tests/integration_test.go`
   - Create test fixtures in `testdata/lineinfile/`

5. **Integration Test Scenarios** (from `quickstart.md`):
   - Scenario 1: Fresh shell profile setup
   - Scenario 2: Replace debug setting
   - Scenario 3: Remove multiple matches
   - Scenario 4: Multiple matches with prompt
   - Scenario 5: Backup verification
   - Scenario 6: Encoding handling
   - **Parallel**: [P] - Integration tests can run in parallel

**Ordering Strategy**:

1. **Test-First (TDD)**: Contract tests before implementation
2. **Bottom-Up**: Data structures → File operations → Plugin interface
3. **Dependency-Aware**: Config validation → File ops → Execute logic
4. **Parallelizable**: Mark independent tasks with [P] for concurrent execution

**Task Breakdown by Phase**:

| Phase | Task Type | Count | Parallel |
|-------|-----------|-------|----------|
| Setup | Create directory structure, test fixtures | 2 | No |
| Tests | Contract tests (Name, Validate, Execute, DryRun) | 8-10 | Yes [P] |
| Models | Config, FileState, MatchResult, ChangeSet, error types | 5-6 | Yes [P] |
| Operations | File read/write, backup, atomic write, encoding | 4-5 | Partial [P] |
| Matching | Regex compilation, line matching, change detection | 3-4 | Yes [P] |
| Plugin | Interface implementation, integration with core | 4-5 | No |
| Integration | End-to-end tests, fixture creation, registration | 6-8 | Yes [P] |
| Documentation | Update docs/plugins.md, generate JSON schema | 2 | Yes [P] |

**Estimated Total**: 35-45 numbered, ordered tasks

**Example Task Sequence** (first 10 tasks):

1. Create `internal/plugins/lineinfile/` directory structure
2. Create test fixture files in `testdata/lineinfile/`
3. [P] Write contract test for `Name()` method (TDD)
4. [P] Write contract tests for `Validate()` - all error cases (TDD)
5. [P] Create `config.go` with LineInFileConfig struct
6. Implement config validation logic in `config.go`
7. [P] Create `file_ops.go` with FileState struct
8. Implement file read operation in `file_ops.go`
9. [P] Write contract tests for `Execute()` - append scenario (TDD)
10. [P] Write contract tests for `Execute()` - replace scenario (TDD)
... (continue through all 35-45 tasks)

**IMPORTANT**: This phase is executed by the `/tasks` command, NOT by `/plan`

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)  
**Phase 4**: Implementation (execute tasks.md following constitutional principles)  
**Phase 5**: Validation (run tests, execute quickstart.md, performance validation)

## Complexity Tracking
*Fill ONLY if Constitution Check has violations that must be justified*

**No violations identified.** All constitutional principles satisfied:
- Plugin uses only Go stdlib + x/text (no external dependencies)
- Configuration is flat and clear (8-10 fields, well-documented)
- Plugin architecture properly followed
- Safety defaults enforced (dry-run, backup, idempotency)
- Performance targets reasonable for file operations
- Backward compatible (new plugin, no breaking changes)
- Follows ecosystem conventions (plugin naming, error handling)

---

## Progress Tracking
*This checklist is updated during execution flow*

**Phase Status**:
- [x] Phase 0: Research complete (/plan command)
- [x] Phase 1: Design complete (/plan command)
- [x] Phase 2: Task planning complete (/plan command - describe approach only)
- [x] Phase 3: Tasks generated (/tasks command) - 42 actionable tasks created
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [x] Initial Constitution Check: PASS (all 7 principles satisfied)
- [x] Post-Design Constitution Check: PASS (no violations in design)
- [x] All NEEDS CLARIFICATION resolved (5 clarifications documented)
- [x] Complexity deviations documented (none - clean design)

**Artifacts Generated**:
- [x] `/specs/003-add-built-in/research.md` (10 technical decisions documented)
- [x] `/specs/003-add-built-in/data-model.md` (5 entities, 13 validation rules)
- [x] `/specs/003-add-built-in/contracts/plugin-interface.md` (4 methods, 30+ test scenarios)
- [x] `/specs/003-add-built-in/quickstart.md` (6 validation scenarios, examples, troubleshooting)
- [x] `/home/alexis/Projects/Streamy/AGENTS.md` (updated with new tech context)

---

## 🔧 Interface Alignment Resolution (2025-10-04)

**Context**: During T001-T006 implementation, discovered spec assumed new plugin interface (`Name/Validate/Execute/DryRun`) but codebase uses existing interface (`Metadata/Check/Apply/DryRun`).

**Decision**: Adapt `line_in_file` to existing plugin interface (no refactoring needed) ✅

**Existing Interface** (`internal/plugin/interface.go`):
```go
type Plugin interface {
    Metadata() Metadata               // Returns Name, Version, Type
    Schema() interface{}              // Returns config struct
    Check(ctx, step) (bool, error)    // Idempotency check
    Apply(ctx, step) (*StepResult, error)    // Execute changes
    DryRun(ctx, step) (*StepResult, error)   // Preview changes
}
```

**Mapping Strategy**:
| Spec Contract | Existing Method | Implementation Notes |
|---------------|----------------|---------------------|
| `Name()` | `Metadata().Type` | Return `"line_in_file"` in Type field |
| `Validate()` | Built-in to `Apply()`/`DryRun()` | Validate config at method start |
| `Execute()` | `Apply()` | Direct 1:1 mapping |
| `DryRun()` | `DryRun()` | Direct 1:1 mapping |
| N/A (bonus) | `Check()` | Efficient idempotency: returns true if file already in desired state |

**Configuration Changes Required**:
1. Add `LineInFileStep` struct to `internal/config/types.go` with all fields (file, line, state, match, etc.)
2. Add `LineInFile *LineInFileStep` field to `Step` struct (inline YAML)
3. Update `Step.Type` validation: change `oneof=package repo symlink copy command template` → add `line_in_file`
4. Update `Step.UnmarshalYAML()` switch: add `case "line_in_file"` to decode into `step.LineInFile`

**Error Handling**: Use existing `streamyerrors` package:
- `streamyerrors.NewValidationError(stepID, message, cause)` for config errors
- `streamyerrors.NewExecutionError(stepID, err)` for runtime errors

**No Breaking Changes**: All 6 existing plugins (command, copy, package, repo, symlink, template) continue working unchanged.

**Impact on Tasks**: 
- T004-T006 need minor adjustments (use `Metadata()` instead of `Name()`, validate in `Apply()`)
- T007-T019 remain largely unchanged (test `Apply()` and `DryRun()` behavior)
- T020-T037 adapt to existing interface
- T038 unchanged (registration pattern matches existing plugins)

---
*Based on Constitution v2.1.1 - See `/memory/constitution.md`*
