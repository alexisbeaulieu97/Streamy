# Tasks: Extend Plugin Contract with Verify Lifecycle

**Input**: Design documents from `/specs/004-extend-plugin-contract/`  
**Prerequisites**: plan.md ✓, research.md ✓, data-model.md ✓, contracts/ ✓, quickstart.md ✓

## Execution Flow (main)
```
1. Load plan.md from feature directory
   → ✓ Loaded successfully
   → ✓ Tech stack: Go 1.25+, no new external dependencies
   → ✓ Structure: Single project (CLI tool with plugin architecture)
2. Load optional design documents:
   → ✓ data-model.md: 5 entities (VerificationStatus, VerificationResult, VerificationSummary, Plugin extension, Step extension)
   → ✓ contracts/: 2 contracts (plugin-verify-contract.md, cli-verify-contract.md)
   → ✓ research.md: 13 technical decisions documented
   → ✓ quickstart.md: User workflows and examples
3. Generate tasks by category:
   → Setup: Go module management, linting
   → Tests: Contract tests (read-only, timeout, status accuracy), integration tests
   → Core: Models, plugin interface extension, plugin implementations
   → Integration: Executor verification logic, CLI command
   → Polish: Unit tests, performance validation, documentation
4. Apply task rules:
   → Different files = mark [P] for parallel
   → Same file = sequential (no [P])
   → Tests before implementation (TDD)
5. Number tasks sequentially (T001, T002...)
   → ✓ 46 tasks generated
6. Generate dependency graph
   → ✓ Foundation → Tests → Plugin implementations → Executor → CLI → Polish
7. Create parallel execution examples
   → ✓ 7 plugin implementations can run in parallel
   → ✓ Multiple contract tests can run in parallel
8. Validate task completeness:
   → ✓ All contracts have tests
   → ✓ All entities have implementation tasks
   → ✓ All plugins updated with Verify()
9. Return: SUCCESS (tasks ready for execution)
```

## Format: `[ID] [P?] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- Include exact file paths in descriptions

## Path Conventions
Single project structure (CLI tool):
- `cmd/streamy/` - CLI command implementations
- `internal/model/` - Data models and types
- `internal/plugin/` - Plugin interface and registry
- `internal/plugins/` - Plugin implementations
- `internal/engine/` - Execution engine
- `internal/config/` - Configuration structures
- `pkg/diff/` - Utility packages
- `tests/` - Integration tests

---

## Phase 3.1: Setup & Foundation

### Setup
- [x] **T001** - Verify Go module dependencies are up to date
  - Run `go mod tidy` to ensure clean dependency state
  - Verify no new external dependencies needed (constitution principle I)
  - **Acceptance**: `go mod tidy` completes without changes

- [x] **T002** [P] - Run gofmt and existing linters
  - Execute `go fmt ./...` to format all Go files
  - Run any existing linters configured in the project
  - **Acceptance**: No formatting or lint errors

### Foundation Models (All Parallel)
- [x] **T003** [P] - Create VerificationStatus type in `internal/model/verification_result.go`
  - Define string-based enum with 5 constants: StatusSatisfied, StatusMissing, StatusDrifted, StatusBlocked, StatusUnknown
  - Add validation helper method `IsValid() bool`
  - **Acceptance**: Type compiles, constants defined per data-model.md

- [x] **T004** [P] - Create VerificationResult struct in `internal/model/verification_result.go`
  - Define struct with 7 fields: StepID (string), Status (VerificationStatus), Message (string), Details (string), Error (error), Duration (time.Duration), Timestamp (time.Time)
  - **Acceptance**: Struct compiles with all required fields per data-model.md

- [x] **T005** [P] - Create VerificationSummary struct in `internal/model/verification_result.go`
  - Define struct with summary fields: TotalSteps, Satisfied, Missing, Drifted, Blocked, Unknown counts, Results slice, Duration
  - Add helper methods: `AllSatisfied() bool`, `NeedsApply() bool`, `ExitCode() int`
  - **Acceptance**: Struct compiles with helper methods per data-model.md

- [x] **T006** [P] - Create diff utility package in `pkg/diff/diff.go`
  - Implement `GenerateUnifiedDiff(expected, actual []byte, expectedLabel, actualLabel string) string`
  - Use unified diff format (compatible with git diff)
  - Include 3 lines of context before/after changes
  - Truncate diffs >10,000 lines with "... truncated ..." marker
  - **Acceptance**: Function compiles, returns valid unified diff format

- [x] **T007** [P] - Add diff utility tests in `pkg/diff/diff_test.go`
  - Test identical content returns empty diff
  - Test single line change produces correct diff
  - Test multi-line changes with context
  - Test truncation for large diffs
  - **Acceptance**: All tests pass with `go test ./pkg/diff`

### Plugin Interface Extension
- [x] **T008** - Extend Plugin interface in `internal/plugin/interface.go`
  - Add `Verify(ctx context.Context, step *config.Step) (*model.VerificationResult, error)` method signature
  - Import necessary packages (context, model)
  - **Acceptance**: Interface compiles; existing plugins will fail compilation (expected)

- [x] **T009** - Add verify_timeout field to Step in `internal/config/types.go`
  - Add `VerifyTimeout time.Duration` field with `yaml:"verify_timeout,omitempty"` tag
  - **Acceptance**: Field compiles, YAML tag correct per schema

---

## Phase 3.2: Tests First (TDD) ⚠️ MUST COMPLETE BEFORE 3.3

**CRITICAL: These tests MUST be written and MUST FAIL before ANY plugin implementation**

### Plugin Contract Tests (All Parallel)
- [x] **T010** [P] - Contract test: Read-only verification in `internal/plugins/contract_test.go`
  - Test that Verify() calls do not modify filesystem, packages, or system state
  - Capture state before/after verification for all plugin types
  - **Acceptance**: Test exists and fails (no Verify() implemented yet), tests BR-001 from plugin contract

- [x] **T011** [P] - Contract test: Context cancellation in `internal/plugins/contract_test.go`
  - Test that cancelled context causes Verify() to return immediately with context.Canceled error
  - Test for all plugin types
  - **Acceptance**: Test exists and fails, tests BR-002 from plugin contract

- [x] **T012** [P] - Contract test: Timeout handling in `internal/plugins/contract_test.go`
  - Test that Verify() respects context deadline and returns within timeout
  - Use 1ms deadline to force timeout
  - **Acceptance**: Test exists and fails, tests BR-002 from plugin contract

- [x] **T013** [P] - Contract test: Status accuracy in `internal/plugins/contract_test.go`
  - Test all 5 status returns (satisfied/missing/drifted/blocked/unknown) for appropriate scenarios
  - Create test fixtures for each status condition
  - **Acceptance**: Test exists and fails, tests BR-003 from plugin contract

- [x] **T014** [P] - Contract test: Message clarity in `internal/plugins/contract_test.go`
  - Test that VerificationResult.Message is never empty and >10 characters
  - Test for all plugin types and all statuses
  - **Acceptance**: Test exists and fails, tests BR-004 from plugin contract

- [x] **T015** [P] - Contract test: Idempotency in `internal/plugins/contract_test.go`
  - Test that calling Verify() twice produces identical results (assuming no system state change)
  - Test for all plugin types
  - **Acceptance**: Test exists and fails, tests BR-008 from plugin contract

### Integration Tests (All Parallel)
- [x] **T016** [P] - Integration test: All satisfied scenario in `tests/integration_verify_test.go`
  - Create config with 3 satisfied steps
  - Run verification, assert all return StatusSatisfied
  - Assert exit code 0
  - **Acceptance**: Test exists and fails (no verify command yet)

- [x] **T017** [P] - Integration test: Missing steps scenario in `tests/integration_verify_test.go`
  - Create config with steps pointing to non-existent resources
  - Run verification, assert StatusMissing returned
  - Assert exit code 1
  - **Acceptance**: Test exists and fails

- [x] **T018** [P] - Integration test: Drifted steps scenario in `tests/integration_verify_test.go`
  - Create config with steps where actual state differs from expected
  - Run verification, assert StatusDrifted with diff in Details field
  - Assert exit code 1
  - **Acceptance**: Test exists and fails

- [x] **T019** [P] - Integration test: Blocked steps scenario in `tests/integration_verify_test.go`
  - Create config with steps that will encounter permission errors
  - Run verification, assert StatusBlocked with error details
  - Assert exit code 1
  - **Acceptance**: Test exists and fails

- [x] **T020** [P] - Integration test: Unknown steps scenario in `tests/integration_verify_test.go`
  - Create config with command steps lacking verify clause
  - Run verification, assert StatusUnknown returned
  - Assert exit code 1
  - **Acceptance**: Test exists and fails

- [x] **T021** [P] - Integration test: Dependency blocking in `tests/integration_verify_test.go`
  - Create config where step B depends on step A (via depends_on)
  - Make step A return missing/blocked
  - Run verification, assert step B is blocked or skipped appropriately
  - **Acceptance**: Test exists and fails

- [x] **T022** [P] - Integration test: Verbose output format in `tests/integration_verify_test.go`
  - Run verify with --verbose flag on drifted step
  - Assert output includes unified diff in Details field
  - **Acceptance**: Test exists and fails

- [x] **T023** [P] - Integration test: JSON output format in `tests/integration_verify_test.go`
  - Run verify with --json flag
  - Parse JSON output and validate schema matches cli-verify-contract.md
  - Assert all required fields present
  - **Acceptance**: Test exists and fails

---

## Phase 3.3: Core Implementation (ONLY after tests are failing)

### Plugin Implementations (7 Plugins - All Parallel)

- [x] **T024** [P] - Implement Verify() in symlink plugin (`internal/plugins/symlink/symlink.go`)
  - Check if symlink exists via `os.Lstat()`
  - If not exists: return StatusMissing
  - If exists: read target via `os.Readlink()`
  - Compare target to expected source
  - If match: return StatusSatisfied
  - If differ: return StatusDrifted with message showing actual vs expected
  - If permission error: return StatusBlocked with error
  - **Acceptance**: Symlink contract tests pass, behavior matches plugin-verify-contract.md

- [x] **T025** [P] - Implement Verify() in package plugin (`internal/plugins/package/package.go`)
  - Query system package manager (apt, brew, etc.) for each package
  - If not installed: return StatusMissing
  - If installed but wrong version (when version specified): return StatusDrifted
  - If all packages match: return StatusSatisfied
  - Handle query errors: return StatusBlocked
  - **Acceptance**: Package contract tests pass, behavior matches plugin-verify-contract.md

- [x] **T026** [P] - Implement Verify() in template plugin (`internal/plugins/template/template.go`)
  - Render template in-memory using configured variables
  - Read destination file content
  - If destination doesn't exist: return StatusMissing
  - If read error (permission): return StatusBlocked
  - Compare rendered vs actual byte-by-byte
  - If identical: return StatusSatisfied
  - If differ: generate unified diff using pkg/diff, return StatusDrifted with diff in Details
  - **Acceptance**: Template contract tests pass, diff output correct

- [x] **T027** [P] - Implement Verify() in command plugin (`internal/plugins/command/command.go`)
  - Check if `step.Command.Verify` field is specified
  - If not specified: return StatusUnknown with message "no verification command specified"
  - If specified: execute verify command with timeout
  - If exit code 0: return StatusSatisfied
  - If exit code non-zero: return StatusMissing
  - If execution error (timeout, not found): return StatusBlocked
  - **Acceptance**: Command contract tests pass, unknown status for missing verify clause

- [x] **T028** [P] - Implement Verify() in repo plugin (`internal/plugins/repo/repo.go`)
  - Check if directory exists at configured path
  - If not exists: return StatusMissing
  - If exists but not a git repo (.git missing): return StatusBlocked
  - Query remote URL: `git config --get remote.origin.url`
  - Query current branch: `git rev-parse --abbrev-ref HEAD`
  - If remote or branch differs from config: return StatusDrifted with details
  - If all match: return StatusSatisfied
  - Handle git command errors: return StatusBlocked
  - **Acceptance**: Repo contract tests pass

- [x] **T029** [P] - Implement Verify() in lineinfile plugin (`internal/plugins/lineinfile/lineinfile.go`)
  - Read destination file content
  - If file doesn't exist: return StatusMissing
  - If read error: return StatusBlocked
  - Check if expected line exists (state=present) or absent (state=absent)
  - If line matches expectation: return StatusSatisfied
  - If line differs: return StatusDrifted with details
  - **Acceptance**: Lineinfile contract tests pass

- [x] **T030** [P] - Implement Verify() in copy plugin (`internal/plugins/copy/copy.go`)
  - Read source file content
  - Read destination file content
  - If destination doesn't exist: return StatusMissing
  - If read errors: return StatusBlocked
  - Compare checksums (SHA-256)
  - If match: return StatusSatisfied
  - If differ: return StatusDrifted (optionally include diff for text files)
  - **Acceptance**: Copy contract tests pass

### Executor Verification Logic

- [x] **T031** - Add verification execution to executor in `internal/engine/executor.go`
  - Create `VerifySteps(ctx context.Context, steps []*config.Step) (*model.VerificationSummary, error)` method
  - Traverse DAG in dependency order (reuse existing DAG logic)
  - For each step: call plugin's Verify() method with timeout context
  - Aggregate results into VerificationSummary
  - Handle context cancellation gracefully
  - **Acceptance**: Method compiles, basic verification traversal works

- [x] **T032** - Add parallel verification logic to executor
  - Identify independent steps (no dependencies)
  - Launch goroutines for parallel verification (bounded by GOMAXPROCS)
  - Wait for all to complete before proceeding to dependent steps
  - **Acceptance**: Parallel verification completes faster than serial for independent steps

- [x] **T033** - Add timeout enforcement to executor verification
  - Read `verify_timeout` from step config (default 30s if not specified)
  - Create context with deadline for each Verify() call
  - If timeout exceeded, plugin returns with blocked status
  - **Acceptance**: Timeout test passes, blocked status returned after timeout

- [x] **T034** - Add executor verification tests in `internal/engine/executor_test.go`
  - Test verification traverses DAG correctly
  - Test parallel verification for independent steps
  - Test timeout enforcement
  - Test summary aggregation (counts per status)
  - **Acceptance**: All executor verification tests pass

### CLI Verify Command

- [x] **T035** - Create verify command implementation in `cmd/streamy/verify.go`
  - Add `verifyCmd` to Cobra command structure
  - Parse config file argument
  - Add flags: --verbose, --json, --timeout
  - Load and validate configuration
  - Call executor.VerifySteps()
  - **Acceptance**: Command compiles and registers with CLI

- [x] **T036** - Implement table output formatter for verify command
  - Format VerificationSummary as human-readable table
  - Use symbols: ✔ (satisfied), ✖ (missing), ⚠ (drifted), 🚫 (blocked), ? (unknown)
  - Show step ID, status symbol, message, duration
  - Print summary line with counts
  - **Acceptance**: Table output matches format in cli-verify-contract.md

- [x] **T037** - Implement verbose output formatter for verify command
  - When --verbose flag set, include full Details field for drifted steps
  - Show unified diff output
  - Include error details for blocked steps
  - **Acceptance**: Verbose output includes diffs per quickstart.md examples

- [x] **T038** - Implement JSON output formatter for verify command
  - When --json flag set, marshal VerificationSummary to JSON
  - Use field names matching cli-verify-contract.md schema
  - Pretty-print JSON with indentation
  - **Acceptance**: JSON output validates against schema, no color codes

- [x] **T039** - Add exit code handling to verify command
  - Exit 0 if all steps satisfied
  - Exit 1 if any step missing/drifted/blocked/unknown
  - Exit 2 if configuration error (parse error, validation failure)
  - Exit 3 if execution error (plugin crash, unexpected failure)
  - **Acceptance**: Exit codes match cli-verify-contract.md specification

- [x] **T040** - Add CLI integration tests in `cmd/streamy/verify_test.go`
  - Test command parsing and flag handling
  - Test exit codes for various scenarios
  - Test output format switching (table vs JSON)
  - **Acceptance**: CLI integration tests pass

---

## Phase 3.4: Integration & Validation

### Integration
- [x] **T041** - Wire verify command into main CLI in `cmd/streamy/main.go`
  - Register verifyCmd with root command
  - Ensure command appears in help output
  - **Acceptance**: `streamy verify --help` shows command documentation

- [x] **T042** - Add structured logging to verification flow
  - Log verification start with config file and step count
  - Log per-step verification with step_id, status, duration
  - Log verification complete with summary
  - Use existing logger (zerolog) with structured fields
  - **Acceptance**: Logs include all required fields per cli-verify-contract.md

### Polish
- [x] **T043** [P] - Run full test suite
  - Execute `go test ./...` to run all tests
  - Verify all contract tests pass
  - Verify all integration tests pass
  - Fix any failing tests
  - **Acceptance**: Zero test failures, coverage >80%

- [x] **T016** [P] - Integration test: All satisfied scenario in `tests/integration_verify_test.go`
  - Create config with 3 satisfied steps
  - Run verification, assert all return StatusSatisfied
  - Assert exit code 0
  - **Acceptance**: Test exists and passes

- [x] **T017** [P] - Integration test: Missing steps scenario in `tests/integration_verify_test.go`
  - Create config with steps pointing to non-existent resources
  - Run verification, assert StatusMissing returned
  - Assert exit code 1
  - **Acceptance**: Test exists and passes

- [x] **T018** [P] - Integration test: Drifted steps scenario in `tests/integration_verify_test.go`
  - Create config with steps where actual state differs from expected
  - Run verification, assert StatusDrifted with diff in Details field
  - Assert exit code 1
  - **Acceptance**: Test exists and passes

- [x] **T019** [P] - Integration test: Blocked steps scenario in `tests/integration_verify_test.go`
  - Create config with steps that will encounter permission errors
  - Run verification, assert StatusBlocked with error details
  - Assert exit code 1
  - **Acceptance**: Test exists and passes

- [x] **T020** [P] - Integration test: Unknown steps scenario in `tests/integration_verify_test.go`
  - Create config with command steps lacking verify clause
  - Run verification, assert StatusUnknown returned
  - Assert exit code 1
  - **Acceptance**: Test exists and passes

- [x] **T021** [P] - Integration test: Dependency blocking in `tests/integration_verify_test.go`
  - Create config where step B depends on step A (via depends_on)
  - Make step A return missing/blocked
  - Run verification, assert step B is blocked or skipped appropriately
  - **Acceptance**: Test exists and passes

- [x] **T022** [P] - Integration test: Verbose output format in `tests/integration_verify_test.go`
  - Run verify with --verbose flag on drifted step
  - Assert output includes unified diff in Details field
  - **Acceptance**: Test exists and passes

- [x] **T023** [P] - Integration test: JSON output format in `tests/integration_verify_test.go`
  - Run verify with --json flag
  - Parse JSON output and validate schema matches cli-verify-contract.md
  - Assert all required fields present
  - **Acceptance**: Test exists and passes

- [x] **T044** [P] - Performance validation
  - Create config with 50 steps
  - Run verification and measure duration
  - Assert total time <5 seconds
  - Profile if necessary and optimize
  - **Acceptance**: Verification completes <5s for 50-step config

- [ ] **T045** [P] - Update plugin development documentation in `docs/plugins.md`
  - Add Verify() method requirements
  - Document contract test expectations
  - Add examples of verification implementations
  - Reference plugin-verify-contract.md
  - **Acceptance**: Documentation includes Verify() guidance

- [ ] **T046** [P] - Update configuration schema documentation
  - Add `verify_timeout` field to schema docs
  - Provide examples of timeout configuration
  - Document default timeout (30s)
  - **Acceptance**: Schema docs include verify_timeout

---

## Dependencies

### Critical Path
```
T001-T009 (Foundation) 
  → T010-T023 (Tests - MUST FAIL)
    → T024-T030 (Plugin Implementations - parallel)
      → T031-T034 (Executor)
        → T035-T040 (CLI)
          → T041-T042 (Integration)
            → T043-T046 (Polish - parallel)
```

### Blocking Relationships
- **T001-T009** must complete before any test writing (foundation needed)
- **T010-T023** must exist and fail before **T024-T046** (TDD requirement)
- **T008** (Plugin interface) blocks **T024-T030** (plugin implementations)
- **T003-T005** (Model types) block **T031** (Executor)
- **T031** (Executor) blocks **T035** (CLI command)
- **T024-T030** + **T031** block **T041** (Integration)

### Parallel Opportunities
- **T003-T007** (Foundation models) can run together (different concerns)
- **T010-T023** (All tests) can run together (test-only, no implementation conflicts)
- **T024-T030** (7 plugin implementations) can run together (different files)
- **T043-T046** (Polish tasks) can run together (different artifacts)

---

## Parallel Execution Example

### Group 1: Foundation Models (Parallel)
```bash
# Run these 5 tasks simultaneously:
go run task-agent.go "Create VerificationStatus type in internal/model/verification_result.go"
go run task-agent.go "Create VerificationResult struct in internal/model/verification_result.go"
go run task-agent.go "Create VerificationSummary struct in internal/model/verification_result.go"
go run task-agent.go "Create diff utility package in pkg/diff/diff.go"
go run task-agent.go "Add diff utility tests in pkg/diff/diff_test.go"
```

### Group 2: Contract Tests (Parallel - After Foundation)
```bash
# Run these 6 tests simultaneously (they will all fail initially - expected):
go run task-agent.go "Contract test: Read-only verification in internal/plugins/contract_test.go"
go run task-agent.go "Contract test: Context cancellation in internal/plugins/contract_test.go"
go run task-agent.go "Contract test: Timeout handling in internal/plugins/contract_test.go"
go run task-agent.go "Contract test: Status accuracy in internal/plugins/contract_test.go"
go run task-agent.go "Contract test: Message clarity in internal/plugins/contract_test.go"
go run task-agent.go "Contract test: Idempotency in internal/plugins/contract_test.go"
```

### Group 3: Plugin Implementations (Parallel - After Tests Fail)
```bash
# Run these 7 plugin implementations simultaneously:
go run task-agent.go "Implement Verify() in symlink plugin"
go run task-agent.go "Implement Verify() in package plugin"
go run task-agent.go "Implement Verify() in template plugin"
go run task-agent.go "Implement Verify() in command plugin"
go run task-agent.go "Implement Verify() in repo plugin"
go run task-agent.go "Implement Verify() in lineinfile plugin"
go run task-agent.go "Implement Verify() in copy plugin"
```

### Group 4: Polish (Parallel - After Integration Complete)
```bash
# Run these 4 tasks simultaneously:
go run task-agent.go "Run full test suite"
go run task-agent.go "Performance validation"
go run task-agent.go "Update plugin development documentation"
go run task-agent.go "Update configuration schema documentation"
```

---

## Notes

- **[P] tasks** = Different files, no dependencies, safe to parallelize
- **TDD Critical**: Verify all tests (T010-T023) fail before implementing (T024-T046)
- **Commit Strategy**: Commit after each task or logical group
- **Constitution Compliance**: All tasks align with principles (no external deps, safety first, plugin-centric)
- **Breaking Change**: Plugin interface extension (T008) will break compilation of existing plugins - this is intentional and acceptable pre-1.0

---

## Task Generation Rules Applied

1. **From Contracts**:
   - plugin-verify-contract.md → T010-T015 (6 contract tests)
   - cli-verify-contract.md → T016-T023 (8 integration tests)
   
2. **From Data Model**:
   - VerificationStatus → T003
   - VerificationResult → T004
   - VerificationSummary → T005
   - Plugin interface → T008
   - Step extension → T009

3. **From Plan Architecture**:
   - 7 plugins need Verify() → T024-T030
   - Executor needs verification → T031-T034
   - CLI needs verify command → T035-T040
   - Diff utility needed → T006-T007

4. **Ordering**:
   - Setup → Foundation → Tests → Implementation → Integration → Polish
   - Dependencies enforce sequential order where needed
   - [P] markers enable parallelization where safe

5. **Constitution-Driven Tasks**:
   - Onboarding: T001 (no new deps), T041 (command registration)
   - Schema: T009 (verify_timeout), T046 (schema docs)
   - Safety: T010 (read-only test), T015 (idempotency test)
   - Performance: T044 (5s target validation)
   - Plugin: T008 (interface), T024-T030 (implementations), T010-T015 (contract tests)

---

## Validation Checklist

- [x] All contracts have corresponding tests (T010-T023 cover both contracts)
- [x] All entities have implementation tasks (T003-T005, T008-T009)
- [x] All tests come before implementation (T010-T023 before T024-T046)
- [x] Parallel tasks truly independent (verified file paths)
- [x] Each task specifies exact file path
- [x] No task modifies same file as another [P] task
- [x] TDD workflow enforced (tests must fail first)
- [x] Constitution principles reflected in task design

---

**Tasks Status**: ✅ COMPLETE - 46 tasks generated, ready for execution  
**Estimated Timeline**: 9-14 days with parallelization (per plan.md estimate)  
**Next Command**: Begin execution with Phase 3.1 setup tasks (T001-T002)
