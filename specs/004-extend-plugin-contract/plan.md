
# Implementation Plan: Extend Plugin Contract with Verify Lifecycle

**Branch**: `004-extend-plugin-contract` | **Date**: October 4, 2025 | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification from `/specs/004-extend-plugin-contract/spec.md`

## Execution Flow (/plan command scope)
```
1. Load feature spec from Input path
   → ✓ Loaded successfully
2. Fill Technical Context (scan for NEEDS CLARIFICATION)
   → ✓ No NEEDS CLARIFICATION markers (all resolved via clarification phase)
   → ✓ Detected Project Type: single (Go-based CLI tool)
   → ✓ Structure Decision: Single project with plugin architecture
3. Fill the Constitution Check section
   → ✓ Completed based on constitution.md requirements
4. Evaluate Constitution Check section below
   → ✓ All principles satisfied (no violations)
   → ✓ Updated Progress Tracking: Initial Constitution Check PASS
5. Execute Phase 0 → research.md
   → ✓ Generated research.md with technical decisions
6. Execute Phase 1 → contracts, data-model.md, quickstart.md, AGENTS.md
   → ✓ Generated data-model.md
   → ✓ Generated contracts/plugin-verify-contract.md
   → ✓ Generated contracts/cli-verify-contract.md
   → ✓ Generated quickstart.md
   → ✓ Updated AGENTS.md
7. Re-evaluate Constitution Check section
   → ✓ No new violations introduced
   → ✓ Updated Progress Tracking: Post-Design Constitution Check PASS
8. Plan Phase 2 → Describe task generation approach
   → ✓ Completed below
9. STOP - Ready for /tasks command
```

**IMPORTANT**: The /plan command STOPS at step 8. Phases 2-4 are executed by other commands:
- Phase 2: /tasks command creates tasks.md
- Phase 3-4: Implementation execution (manual or via tools)

## Summary

**Primary Requirement**: Extend Streamy's plugin interface with a `Verify()` method that performs read-only state inspection to determine configuration alignment.

**Technical Approach**: 
- Add new `Verify()` method to `Plugin` interface returning `VerificationResult` with five-status enum (satisfied/missing/drifted/blocked/unknown)
- Implement verification logic in all existing plugins (package, symlink, template, command, repo, lineinfile, copy)
- Create standalone `streamy verify` CLI command with DAG-based execution
- Support optional `--verify-first` optimization in future apply workflow
- Generate unified diff output for drifted status
- Enforce 30-second default timeout (configurable per-step)

## Technical Context

**User-Provided Context**:
```
Implement Verify Lifecycle Across All Plugins

Updated Plugin Interface with Verify() method
VerificationStatus Enum (satisfied/drifted/missing/blocked/unknown)
CLI Integration: verify command, --verify-first flag
Plugin-specific verification logic for all built-in plugins
Testing: unit, integration, contract tests
```

**Language/Version**: Go 1.25+  
**Primary Dependencies**: Existing Streamy dependencies (no new external deps)  
**Storage**: N/A (stateless verification, in-memory results)  
**Testing**: Go testing framework (`go test`), table-driven tests, contract tests  
**Target Platform**: Linux, macOS, Windows (cross-platform via Go compilation)  
**Project Type**: Single (CLI tool with plugin architecture)  
**Performance Goals**: 
  - <100ms for simple verifications (file existence, symlink read)
  - <1s for medium verifications (template render, package query)
  - <30s default timeout for complex verifications (large files, network)
  - Target <5s total for typical 50-step config
**Constraints**:
  - Read-only guarantee (no state modification during verification)
  - Context cancellation must propagate correctly
  - Verification results must be deterministic (idempotent calls)
**Scale/Scope**: 
  - 7 built-in plugins to update
  - 3 new model types (VerificationStatus, VerificationResult, VerificationSummary)
  - 1 new CLI command
  - ~15-20 implementation tasks

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**I. Onboarding First**
- [x] Feature requires no additional dependencies beyond compiled binary
  - *Verification uses existing plugin infrastructure*
- [x] No system packages, language runtimes, or external tools needed
  - *Pure Go implementation, no external tools*
- [x] If dependencies required, implemented as optional plugin
  - *N/A - no new dependencies*
- [x] First-run experience documented and tested
  - *Quickstart guide provided, contract tests ensure correctness*

**II. Schema Clarity & Fun**
- [x] Configuration uses flat flags for common options
  - *CLI flags: --verbose, --json, --timeout*
- [x] Complex configs use clear nested structures with examples
  - *Optional verify_timeout field per step*
- [x] `id` and `name` fields used appropriately (machine vs. human)
  - *Verification results include step_id (machine) and message (human)*
- [x] JSON schema provided for validation
  - *JSON output format documented in CLI contract*
- [x] Error messages include file/line context and fix suggestions
  - *Blocked status includes error details, messages suggest next steps*

**III. Plugin-Centric Architecture**
- [x] Core logic limited to DAG execution, logging, validation
  - *Executor orchestrates verification; plugins implement domain logic*
- [x] Domain-specific logic implemented in plugins
  - *Each plugin verifies its own resource type (package, symlink, etc.)*
- [x] Plugin interfaces versioned and backward compatible
  - *Extends existing Plugin interface (breaking change acceptable pre-1.0)*
- [x] Plugin contract tests included
  - *Contract test requirements documented in plugin-verify-contract.md*

**IV. Safety by Default**
- [x] Dry-run mode supported for preview
  - *Verification IS the safe preview mode (read-only by design)*
- [x] Destructive operations require explicit flags/confirmation
  - *N/A - verification never modifies state*
- [x] Operations are idempotent (safe to run multiple times)
  - *Multiple verify calls produce same results (BR-008)*
- [x] Rollback/recovery procedures documented
  - *N/A - read-only operation requires no rollback*
- [x] Parallel execution defaults are safe
  - *Verification respects DAG dependencies, parallel where safe*

**V. Performance & Reliability**
- [x] Dry-run completes in <1s for typical configs
  - *Verification target <5s for 50 steps*
- [x] Structured logging shows task timing and dependencies
  - *Duration tracked per step, logged with structured fields*
- [x] Error messages include context, cause, and remediation
  - *VerificationResult.Message provides actionable guidance*
- [x] Resource limits declared for scheduling
  - *Timeout configured per step (default 30s)*
- [x] Timeouts configured for long operations
  - *verify_timeout field, --timeout CLI flag*

**VI. Extensibility & Composability**
- [x] Feature works in simple and complex scenarios
  - *Handles single-step to 100+ step configs*
- [x] No breaking changes to existing configs
  - *Optional verify_timeout field, existing configs work as-is*
- [x] Supports composition (imports, groups, conditionals where relevant)
  - *Verification respects existing DAG composition*
- [x] Backward compatible within major version
  - *Pre-1.0: breaking plugin interface change acceptable*

**VII. Ecosystem Consistency**
- [x] Follows plugin naming conventions (`id`, `name`, `enabled`, `depends_on`)
  - *Uses existing step.id, respects depends_on for DAG*
- [x] Structured error handling implemented
  - *VerificationResult captures errors in structured format*
- [x] Documentation includes schema, examples, troubleshooting
  - *Quickstart, contracts, and data model docs provided*
- [x] Version compatibility declared explicitly
  - *Contract version 1.0.0, stability marked as unstable pre-1.0*

## Project Structure

### Documentation (this feature)
```
specs/004-extend-plugin-contract/
├── plan.md                           # This file (/plan command output)
├── research.md                       # Phase 0 output (/plan command)
├── data-model.md                     # Phase 1 output (/plan command)
├── quickstart.md                     # Phase 1 output (/plan command)
├── contracts/                        # Phase 1 output (/plan command)
│   ├── plugin-verify-contract.md     # Plugin.Verify() behavioral contract
│   └── cli-verify-contract.md        # CLI command contract
└── tasks.md                          # Phase 2 output (/tasks command - NOT created yet)
```

### Source Code (repository root)
```
# Single project structure (CLI tool)
cmd/
└── streamy/
    ├── main.go                       # Entry point
    ├── verify.go                     # NEW: verify command implementation
    └── ...existing commands...

internal/
├── model/
│   ├── step_result.go                # Existing
│   └── verification_result.go        # NEW: VerificationResult, VerificationStatus, VerificationSummary
├── plugin/
│   ├── interface.go                  # UPDATE: Add Verify() to Plugin interface
│   └── registry.go                   # Existing
├── plugins/
│   ├── package/
│   │   ├── package.go                # UPDATE: Implement Verify()
│   │   └── package_test.go           # UPDATE: Add verification tests
│   ├── symlink/
│   │   ├── symlink.go                # UPDATE: Implement Verify()
│   │   └── symlink_test.go           # UPDATE: Add verification tests
│   ├── template/
│   │   ├── template.go               # UPDATE: Implement Verify()
│   │   └── template_test.go          # UPDATE: Add verification tests
│   ├── command/
│   │   ├── command.go                # UPDATE: Implement Verify()
│   │   └── command_test.go           # UPDATE: Add verification tests
│   ├── repo/
│   │   ├── repo.go                   # UPDATE: Implement Verify()
│   │   └── repo_test.go              # UPDATE: Add verification tests
│   ├── lineinfile/
│   │   ├── lineinfile.go             # UPDATE: Implement Verify()
│   │   └── lineinfile_test.go        # UPDATE: Add verification tests
│   └── copy/
│       ├── copy.go                   # UPDATE: Implement Verify()
│       └── copy_test.go              # UPDATE: Add verification tests
├── engine/
│   ├── executor.go                   # UPDATE: Add verification execution logic
│   ├── executor_test.go              # UPDATE: Add verification executor tests
│   └── ...existing engine files...
└── config/
    └── types.go                      # UPDATE: Add verify_timeout field to Step

tests/
├── integration_test.go               # UPDATE: Add verification integration tests
└── integration_verify_test.go        # NEW: Dedicated verification integration tests

pkg/
└── diff/                             # NEW: Unified diff generation utilities
    ├── diff.go
    └── diff_test.go
```

**Structure Decision**: Single project structure appropriate for CLI tool. Plugins are internal packages within the main repository. No web/mobile components. All verification logic fits within existing architecture pattern.

## Phase 0: Outline & Research

### Research Completed ✓

Generated `research.md` with comprehensive technical decisions covering:

1. **Verification Status Model**: Five-status enum (satisfied/missing/drifted/blocked/unknown)
2. **Plugin Interface Extension**: Add `Verify()` method with `VerificationResult` return type
3. **Verification Result Structure**: New type with status, message, details, error, timing
4. **Timeout Strategy**: 30s default, per-step configurable
5. **CLI Command Structure**: Standalone `verify` command with --verbose, --json flags
6. **Apply Integration**: Optional `--verify-first` flag (future)
7. **Diff Output Format**: Unified diff (git-style) for text resources
8. **Plugin-Specific Logic**: Detailed verification approaches for each plugin type
9. **Error Handling**: Fail-fast with blocked status, continue remaining steps
10. **Performance**: Parallel verification respecting DAG dependencies
11. **Backward Compatibility**: Breaking change acceptable pre-1.0
12. **Testing Strategy**: Unit, integration, and contract tests

**Key Decisions**:
- Chose five-status model over simpler alternatives for adequate granularity
- Extended Plugin interface rather than creating optional separate interface
- Created dedicated VerificationResult type (separate from StepResult)
- Adopted fail-fast error handling without automatic retries
- Unified diff format for consistency across plugins

**Output**: [research.md](./research.md) ✓

---

## Phase 1: Design & Contracts

### Design Artifacts Generated ✓

**1. Data Model** ([data-model.md](./data-model.md))
   - `VerificationStatus` enum with 5 values
   - `VerificationResult` struct with 7 fields
   - `VerificationSummary` aggregation structure
   - Extended `Plugin` interface with `Verify()` method
   - Configuration extension: `verify_timeout` field
   - Entity relationship diagram
   - State transition diagrams
   - JSON serialization formats
   - Validation rules for all entities

**2. Contracts** ([contracts/](./contracts/))
   
   **Plugin Contract** ([plugin-verify-contract.md](./contracts/plugin-verify-contract.md)):
   - 10 behavioral requirements (BR-001 through BR-010)
   - Read-only guarantee (critical)
   - Context respect and timeout handling
   - Status accuracy determination rules
   - Message clarity guidelines
   - Diff generation requirements
   - Error propagation patterns
   - Performance bounds and optimization strategies
   - Idempotency requirements
   - Plugin-specific verification logic for each type
   - 6 contract test requirements
   
   **CLI Contract** ([cli-verify-contract.md](./contracts/cli-verify-contract.md)):
   - Command syntax and arguments
   - 10 behavioral contracts (BC-001 through BC-010)
   - Configuration loading and validation
   - DAG-based execution
   - Progress indication (non-JSON mode)
   - Verbose output format with diffs
   - JSON output schema
   - Exit code definitions (0/1/2/3)
   - Error handling and categorization
   - Timeout enforcement hierarchy
   - Signal handling (SIGINT/SIGTERM)
   - Logging integration
   - 6 contract test requirements

**3. User Documentation** ([quickstart.md](./quickstart.md))
   - "What is Verification?" introduction
   - Quick example with 3-step config
   - Verification status explanations with symbols
   - 4 common workflows:
     * Pre-apply audit
     * Drift detection
     * Compliance auditing
     * CI/CD integration
   - Command reference with examples
   - Plugin-specific verification behavior
   - Troubleshooting Q&A section
   - Summary table of common actions

**4. Agent Context Update**
   - Updated `AGENTS.md` with verification lifecycle information
   - Preserved existing context and manual additions
   - Added project structure and recent changes

**Output**: 
- ✓ data-model.md
- ✓ contracts/plugin-verify-contract.md
- ✓ contracts/cli-verify-contract.md
- ✓ quickstart.md
- ✓ AGENTS.md (updated)

---

## Phase 2: Task Planning Approach

*This section describes what the /tasks command will do - DO NOT execute during /plan*

### Task Generation Strategy

**Source Documents**:
- Phase 1 contracts (plugin-verify-contract.md, cli-verify-contract.md)
- Data model (data-model.md)
- Research decisions (research.md)
- Functional requirements from spec.md (FR-001 through FR-023)

**Task Categories**:

1. **Foundation Tasks** (models, interfaces)
   - Create `VerificationStatus` enum
   - Create `VerificationResult` struct
   - Create `VerificationSummary` struct
   - Extend `Plugin` interface with `Verify()` method
   - Add `verify_timeout` field to `config.Step`
   - Create diff generation utilities (`pkg/diff`)

2. **Plugin Implementation Tasks** (7 plugins × verify logic + tests)
   - Package plugin: Implement `Verify()` + tests
   - Symlink plugin: Implement `Verify()` + tests
   - Template plugin: Implement `Verify()` + tests
   - Command plugin: Implement `Verify()` + tests
   - Repo plugin: Implement `Verify()` + tests
   - Line-in-file plugin: Implement `Verify()` + tests
   - Copy plugin: Implement `Verify()` + tests

3. **Executor Tasks** (orchestration)
   - Add verification execution to executor
   - Implement DAG-based verification traversal
   - Add timeout enforcement
   - Add parallel verification logic
   - Write executor verification tests

4. **CLI Tasks** (command interface)
   - Create `verify` command implementation
   - Add flag parsing (--verbose, --json, --timeout)
   - Implement table output formatter
   - Implement JSON output formatter
   - Implement verbose output with diffs
   - Add exit code handling
   - Write CLI integration tests

5. **Contract Test Tasks** (ensure compliance)
   - Write plugin contract tests (read-only, cancellation, timeout, status accuracy, idempotency)
   - Write CLI contract tests (exit codes, JSON schema, timeout enforcement)

6. **Integration Test Tasks** (end-to-end)
   - Test all-satisfied scenario
   - Test missing-steps scenario
   - Test drifted-steps scenario
   - Test blocked-steps scenario
   - Test unknown-steps scenario
   - Test dependency blocking propagation
   - Test timeout behavior
   - Test verbose output
   - Test JSON output

7. **Documentation Tasks**
   - Update plugin development guide with `Verify()` requirements
   - Update configuration schema docs with `verify_timeout`
   - Add verification examples to README

### Ordering Strategy

**Phase-Based Ordering**:
1. **Foundation first**: Models and interfaces (no dependencies)
2. **Plugin implementations**: Can be done in parallel after foundation
3. **Executor logic**: Depends on models and at least one plugin
4. **CLI command**: Depends on executor
5. **Contract tests**: Alongside corresponding implementations
6. **Integration tests**: After CLI is functional
7. **Documentation**: Final polish

**Dependency Rules**:
- Models → Plugin implementations
- Models → Executor
- Executor → CLI
- Plugin implementations + Executor → Integration tests
- All code → Documentation

**Parallelization Markers** ([P]):
- All 7 plugin implementation tasks can run in parallel
- Foundation model tasks can run in parallel
- Contract test tasks can run alongside implementation
- Documentation tasks can run alongside testing

### Estimated Task Count

**Breakdown**:
- Foundation: 6 tasks
- Plugin implementations: 14 tasks (7 plugins × 2 subtasks each)
- Executor: 5 tasks
- CLI: 7 tasks
- Contract tests: 2 tasks
- Integration tests: 8 tasks
- Documentation: 3 tasks

**Total**: ~45 tasks (many parallelizable)

**Estimated Effort**: 
- Foundation: 1-2 days
- Plugins: 3-4 days (parallel)
- Executor: 1-2 days
- CLI: 1-2 days
- Testing: 2-3 days
- Documentation: 1 day

**Total**: 9-14 days (with parallelization)

---

## Phase 3+: Future Implementation

*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution
- /tasks command generates tasks.md with ordered, numbered tasks
- Each task includes clear acceptance criteria from contracts
- Tasks marked [P] for parallel execution where safe

**Phase 4**: Implementation
- Follow TDD approach: write contract/integration tests first
- Implement foundation (models, interface extension)
- Implement plugins in parallel
- Implement executor and CLI
- Ensure all contract tests pass

**Phase 5**: Validation
- Run full test suite (`go test ./...`)
- Execute quickstart examples manually
- Performance validation (verify completes <5s for 50 steps)
- Cross-platform testing (Linux, macOS, Windows)
- Update documentation with any discovered edge cases

---

## Complexity Tracking

*Fill ONLY if Constitution Check has violations that must be justified*

**No violations detected.** All constitutional principles satisfied:
- No external dependencies added
- Schema remains clear and simple
- Plugin-centric architecture maintained
- Safety guaranteed (read-only operations)
- Performance targets defined and achievable
- Backward compatibility handled appropriately (pre-1.0 breaking change)
- Ecosystem consistency maintained

---

## Progress Tracking

*This checklist is updated during execution flow*

**Phase Status**:
- [x] Phase 0: Research complete (/plan command)
- [x] Phase 1: Design complete (/plan command)
- [x] Phase 2: Task planning complete (/plan command - describe approach only)
- [ ] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [x] Initial Constitution Check: PASS
- [x] Post-Design Constitution Check: PASS
- [x] All NEEDS CLARIFICATION resolved (via /clarify command)
- [x] Complexity deviations documented (none required)

**Artifacts Generated**:
- [x] research.md
- [x] data-model.md
- [x] contracts/plugin-verify-contract.md
- [x] contracts/cli-verify-contract.md
- [x] quickstart.md
- [x] AGENTS.md (updated)
- [ ] tasks.md (awaiting /tasks command)

---

*Based on Constitution v2.1.1 - See `/memory/constitution.md`*

**Plan Status**: ✅ COMPLETE - Ready for /tasks command
backend/
├── src/
│   ├── models/
│   ├── services/
│   └── api/
└── tests/

frontend/
├── src/
│   ├── components/
│   ├── pages/
│   └── services/
└── tests/

# [REMOVE IF UNUSED] Option 3: Mobile + API (when "iOS/Android" detected)
api/
└── [same as backend above]

ios/ or android/
└── [platform-specific structure: feature modules, UI flows, platform tests]
```

**Structure Decision**: [Document the selected structure and reference the real
directories captured above]

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
- Load `.specify/templates/tasks-template.md` as base
- Generate tasks from Phase 1 design docs (contracts, data model, quickstart)
- Each contract → contract test task [P]
- Each entity → model creation task [P] 
- Each user story → integration test task
- Implementation tasks to make tests pass

**Ordering Strategy**:
- TDD order: Tests before implementation 
- Dependency order: Models before services before UI
- Mark [P] for parallel execution (independent files)

**Estimated Output**: 25-30 numbered, ordered tasks in tasks.md

**IMPORTANT**: This phase is executed by the /tasks command, NOT by /plan

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)  
**Phase 4**: Implementation (execute tasks.md following constitutional principles)  
**Phase 5**: Validation (run tests, execute quickstart.md, performance validation)

## Complexity Tracking
*Fill ONLY if Constitution Check has violations that must be justified*

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |


## Progress Tracking
*This checklist is updated during execution flow*

**Phase Status**:
- [ ] Phase 0: Research complete (/plan command)
- [ ] Phase 1: Design complete (/plan command)
- [ ] Phase 2: Task planning complete (/plan command - describe approach only)
- [ ] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [ ] Initial Constitution Check: PASS
- [ ] Post-Design Constitution Check: PASS
- [ ] All NEEDS CLARIFICATION resolved
- [ ] Complexity deviations documented

---
*Based on Constitution v2.1.1 - See `/memory/constitution.md`*
