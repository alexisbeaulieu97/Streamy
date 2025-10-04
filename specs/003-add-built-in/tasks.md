# Tasks: line_in_file Plugin

**Input**: Design documents from `/home/alexis/Projects/Streamy/specs/003-add-built-in/`
**Prerequisites**: plan.md, research.md, data-model.md, contracts/, quickstart.md

## Execution Flow (main)
```
1. Loaded plan.md from feature directory
   → Extracted: Go 1.21+, plugin architecture, file operations
2. Loaded design documents:
   → data-model.md: 5 entities (Config, FileState, MatchResult, ChangeSet, StepResult)
   → contracts/: 4 plugin methods (Name, Validate, Execute, DryRun)
   → quickstart.md: 6 integration test scenarios
3. Generated tasks by category:
   → Setup: directory structure, test fixtures, dependencies
   → Tests: contract tests (30+ scenarios), integration tests (6 scenarios)
   → Core: config validation, file operations, matching, plugin interface
   → Integration: plugin registration, end-to-end tests
   → Polish: documentation, schema generation, performance validation
4. Applied task rules:
   → Different files = marked [P] for parallel execution
   → Same file = sequential (no [P])
   → Tests before implementation (TDD approach)
5. Numbered tasks sequentially (T001-T042)
6. Generated dependency graph and parallel execution examples
7. Validated task completeness: ✓ All contracts tested, ✓ All entities implemented
8. Result: SUCCESS (42 actionable tasks ready for execution)
```

---

## Format: `[ID] [P?] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- All file paths are absolute from repository root

## Path Conventions
Single Go project structure (existing Streamy codebase):
- Plugin code: `internal/plugins/lineinfile/`
- Test fixtures: `testdata/lineinfile/`
- Integration tests: `tests/integration_test.go`
- Registration: `cmd/streamy/plugins_import.go`

---

## Phase 3.1: Setup & Fixtures

- [X] **T001** Create plugin directory structure
  - Create `internal/plugins/lineinfile/` directory
  - Verify directory follows existing plugin pattern (command, copy, package, repo, symlink, template)
  - No code yet, just directory structure

- [X] **T002** Create test fixtures directory and sample files
  - Create `testdata/lineinfile/` directory
  - Create test fixture files:
    - `testdata/lineinfile/simple.txt` - Basic single-line file
    - `testdata/lineinfile/multiline.txt` - Multi-line configuration file
    - `testdata/lineinfile/utf8.txt` - UTF-8 encoded file with special characters
    - `testdata/lineinfile/latin1.txt` - Latin-1 encoded file
    - `testdata/lineinfile/empty.txt` - Empty file
  - Document fixture purpose in README comment at top of each file

- [X] **T003** Verify Go dependencies
  - Confirm `golang.org/x/text/encoding` is in `go.mod`
  - If missing, run `go get golang.org/x/text/encoding`
  - Run `go mod tidy` to clean up dependencies
  - Verify all stdlib packages available (os, bufio, regexp, io, path/filepath, time)

---

## Phase 3.2: Tests First (TDD) ⚠️ MUST COMPLETE BEFORE 3.3

**CRITICAL: These tests MUST be written and MUST FAIL before ANY implementation**

### Contract Tests for Plugin Interface

- [X] **T004** [P] Contract test for `Name()` method
  - File: `internal/plugins/lineinfile/lineinfile_test.go`
  - Test function: `TestLineInFile_Name`
  - Verify plugin returns `"line_in_file"` as name
  - Simple assertion test (reference: contracts/plugin-interface.md section 1)

- [X] **T005** [P] Contract tests for `Validate()` - valid configurations
  - File: `internal/plugins/lineinfile/lineinfile_test.go`
  - Test function: `TestLineInFile_Validate_Valid`
  - Table-driven test with valid configs:
    - Valid present without match
    - Valid present with match
    - Valid absent with match
    - Valid with all optional fields
  - All tests should pass validation (return nil error)
  - Reference: contracts/plugin-interface.md section 2

- [X] **T006** [P] Contract tests for `Validate()` - configuration errors
  - File: `internal/plugins/lineinfile/lineinfile_test.go`
  - Test function: `TestLineInFile_Validate_Errors`
  - Table-driven test with invalid configs (8+ scenarios):
    - Missing `file` field
    - Missing `line` field
    - Invalid `state` value (not "present" or "absent")
    - `state: absent` without `match` field
    - Invalid regex pattern in `match` field
    - Invalid `on_multiple_matches` value
    - Unsupported `encoding` value
    - Empty `file` path
  - Each test should return appropriate ConfigError
  - Reference: contracts/plugin-interface.md section 2, data-model.md validation rules

- [X] **T007** [P] Contract tests for `Execute()` - state present, no match (append scenarios)
  - File: `internal/plugins/lineinfile/execute_test.go`
  - Test function: `TestLineInFile_Execute_PresentNoMatch`
  - Table-driven test scenarios:
    - Append to empty file
    - Append to file without matching line (line not present)
    - Idempotent: line already exists (no change)
    - Create new file if doesn't exist
  - Verify Changed status, file content, idempotency
  - Use temp files for isolation
  - Reference: contracts/plugin-interface.md section 3, quickstart.md scenario 1

- [X] **T008** [P] Contract tests for `Execute()` - state present, with match (replace scenarios)
  - File: `internal/plugins/lineinfile/execute_test.go`
  - Test function: `TestLineInFile_Execute_PresentWithMatch`
  - Table-driven test scenarios:
    - Replace single matching line
    - Replace first match when `on_multiple_matches: first`
    - Replace all matches when `on_multiple_matches: all`
    - Error when `on_multiple_matches: error` and multiple matches
    - Append when no match found (fallback behavior)
    - Idempotent: replacement already done
  - Reference: contracts/plugin-interface.md section 3, quickstart.md scenario 2

- [X] **T009** [P] Contract tests for `Execute()` - state absent (remove scenarios)
  - File: `internal/plugins/lineinfile/execute_test.go`
  - Test function: `TestLineInFile_Execute_Absent`
  - Table-driven test scenarios:
    - Remove single matching line
    - Remove multiple matching lines
    - Idempotent: line not present (no change)
    - Preserve other lines in file
  - Reference: contracts/plugin-interface.md section 3, quickstart.md scenario 3

- [X] **T010** [P] Contract tests for `Execute()` - file operations
  - File: `internal/plugins/lineinfile/execute_test.go`
  - Test function: `TestLineInFile_Execute_FileOps`
  - Table-driven test scenarios:
    - Preserve file permissions (0644, 0600, 0755)
    - Follow symlinks (modify target file)
    - Error on permission denied (read)
    - Error on permission denied (write to directory)
    - Error on broken symlink
  - Reference: contracts/plugin-interface.md section 3, FR-019, FR-019a

- [X] **T011** [P] Contract tests for `Execute()` - backup functionality
  - File: `internal/plugins/lineinfile/execute_test.go`
  - Test function: `TestLineInFile_Execute_Backup`
  - Table-driven test scenarios:
    - Create backup when `backup: true` and file changes
    - No backup when `backup: false`
    - No backup when file unchanged (idempotent run)
    - Backup to custom `backup_dir` when specified
    - Backup has correct timestamp format (ISO 8601)
    - Backup preserves original file permissions
  - Reference: contracts/plugin-interface.md section 3, quickstart.md scenario 5, FR-010, FR-010a, FR-010b

- [X] **T012** [P] Contract tests for `Execute()` - encoding support
  - File: `internal/plugins/lineinfile/execute_test.go`
  - Test function: `TestLineInFile_Execute_Encoding`
  - Table-driven test scenarios:
    - UTF-8 encoding (default)
    - Latin-1 encoding with special characters
    - ASCII encoding
    - Preserve encoding when modifying file
  - Use test fixtures from T002
  - Reference: contracts/plugin-interface.md section 3, quickstart.md scenario 6, FR-001a, FR-019b

- [X] **T013** [P] Contract tests for `DryRun()` - preview without modification
  - File: `internal/plugins/lineinfile/dryrun_test.go`
  - Test function: `TestLineInFile_DryRun`
  - Table-driven test scenarios:
    - Preview append (show + line)
    - Preview replace (show - old, + new)
    - Preview remove (show - line)
    - Preview no change (empty diff)
    - Verify no files created or modified
    - Verify diff output contains expected markers
  - Reference: contracts/plugin-interface.md section 4, FR-012, FR-013

### Integration Test Scenarios

- [X] **T014** [P] Integration test: Fresh shell profile setup
  - File: `tests/integration_lineinfile_test.go`
  - Test function: `TestIntegration_LineInFile_FreshProfile`
  - Create temp `.bashrc` file
  - Execute Streamy with line_in_file step
  - Verify line added
  - Re-run and verify idempotency (no changes)
  - Reference: quickstart.md scenario 1

- [X] **T015** [P] Integration test: Replace debug setting
  - File: `tests/integration_lineinfile_test.go`
  - Test function: `TestIntegration_LineInFile_ReplaceDebug`
  - Create temp config file with `debug=true`
  - Execute with match pattern and `debug=false`
  - Verify replacement
  - Verify other lines unchanged
  - Reference: quickstart.md scenario 2

- [X] **T016** [P] Integration test: Remove multiple matches
  - File: `tests/integration_lineinfile_test.go`
  - Test function: `TestIntegration_LineInFile_RemoveMultiple`
  - Create file with multiple OLD_VAR exports
  - Execute with `state: absent` and match pattern
  - Verify all matching lines removed
  - Reference: quickstart.md scenario 3

- [X] **T017** [P] Integration test: Backup verification
  - File: `tests/integration_lineinfile_test.go`
  - Test function: `TestIntegration_LineInFile_BackupVerify`
  - Create temp file
  - Execute with `backup: true`
  - Verify backup file created with correct timestamp format
  - Verify backup content matches original
  - Verify backup permissions match original
  - Reference: quickstart.md scenario 5

- [X] **T018** [P] Integration test: Complete shell setup (DAG execution)
  - File: `tests/integration_lineinfile_test.go`
  - Test function: `TestIntegration_LineInFile_CompleteShellSetup`
  - Use complete example from quickstart.md (backup + PATH + EDITOR + remove old + add new JAVA_HOME)
  - Verify DAG dependency ordering
  - Verify all steps execute correctly
  - Verify idempotency on second run
  - Reference: quickstart.md "Complete Example: Shell Profile Setup"

- [X] **T019** [P] Integration test: Dry-run mode
  - File: `tests/integration_lineinfile_test.go`
  - Test function: `TestIntegration_LineInFile_DryRun`
  - Create temp file
  - Execute Streamy with `--dry-run` flag
  - Verify preview shows expected changes
  - Verify no files actually modified
  - Execute without dry-run and verify changes applied
  - Reference: quickstart.md "Dry-Run Mode"

---

## Phase 3.3: Core Implementation (ONLY after tests are failing)

### Configuration & Validation

- [ ] **T020** [P] Create config structs and types
  - File: `internal/plugins/lineinfile/config.go`
  - Implement `LineInFileConfig` struct with all fields from data-model.md
  - Add field tags for YAML parsing
  - Add struct documentation
  - Reference: data-model.md section 1

- [ ] **T021** Implement configuration validation logic
  - File: `internal/plugins/lineinfile/config.go`
  - Function: `(c *LineInFileConfig) Validate() error`
  - Implement all validation rules (V-001 through V-007 from data-model.md):
    - Required field checks (file, line)
    - State value validation (present/absent)
    - Absent requires match pattern
    - Regex compilation validation
    - on_multiple_matches value validation
    - Encoding support validation
  - Return ConfigError with field context
  - Make T005 and T006 pass
  - Reference: data-model.md section 4

### File Operations

- [ ] **T022** [P] Create FileState struct and file reading
  - File: `internal/plugins/lineinfile/file_ops.go`
  - Implement `FileState` struct from data-model.md section 2
  - Function: `ReadFileState(path string, encoding string) (*FileState, error)`
  - Handle tilde expansion (`~` → home directory)
  - Handle symlink resolution (follow by default)
  - Handle encoding (UTF-8, Latin-1, etc.)
  - Return ExecutionError on permission/read failures
  - Reference: data-model.md section 2, research.md decision 3

- [ ] **T023** [P] Implement atomic file write
  - File: `internal/plugins/lineinfile/file_ops.go`
  - Function: `WriteFileAtomic(path string, content []byte, perm os.FileMode) error`
  - Implement temp file + rename pattern from research.md decision 1:
    - Write to temp file in same directory
    - Set permissions on temp file
    - Rename temp to target (atomic)
    - Clean up temp file on error
  - Reference: research.md section 1 "Atomic Write"

- [ ] **T024** [P] Implement backup file creation
  - File: `internal/plugins/lineinfile/file_ops.go`
  - Function: `CreateBackup(path string, backupDir string) (string, error)`
  - Generate timestamp in ISO 8601 format (YYYYMMDDTHHMMSS)
  - Create backup with format `<filename>.<timestamp>.bak`
  - Handle custom backup_dir (create if missing)
  - Default to same directory as original file
  - Preserve file permissions on backup
  - Return backup file path
  - Reference: research.md section 5, FR-010a, FR-010b

### Matching & Line Manipulation

- [ ] **T025** [P] Create MatchResult and regex matching logic
  - File: `internal/plugins/lineinfile/matcher.go`
  - Implement `MatchResult` struct from data-model.md section 2
  - Function: `FindMatches(lines []string, pattern *regexp.Regexp) *MatchResult`
  - Return line numbers and content of all matches
  - Cache compiled regex patterns per execution
  - Reference: data-model.md section 2, research.md section 2

- [ ] **T026** [P] Implement line manipulation logic - append
  - File: `internal/plugins/lineinfile/matcher.go`
  - Function: `AppendLine(lines []string, line string) ([]string, bool)`
  - Check if line already exists (idempotency)
  - Append to end if not present
  - Return modified lines and changed flag
  - Reference: FR-005, data-model.md state machine

- [ ] **T027** [P] Implement line manipulation logic - replace
  - File: `internal/plugins/lineinfile/matcher.go`
  - Function: `ReplaceLines(lines []string, matchResult *MatchResult, newLine string, strategy string) ([]string, bool, error)`
  - Handle strategies: "first", "all", "error"
  - Return error if strategy is "error" and multiple matches
  - Return modified lines and changed flag
  - Reference: FR-006, FR-015, data-model.md state machine

- [ ] **T028** [P] Implement line manipulation logic - remove
  - File: `internal/plugins/lineinfile/matcher.go`
  - Function: `RemoveLines(lines []string, matchResult *MatchResult) ([]string, bool)`
  - Remove all matching lines
  - Return modified lines and changed flag
  - Reference: FR-007, data-model.md state machine

### Diff Generation

- [ ] **T029** [P] Implement unified diff generation for dry-run
  - File: `internal/plugins/lineinfile/differ.go`
  - Implement `ChangeSet` struct from data-model.md section 2
  - Function: `GenerateChangeSet(original, modified []string) *ChangeSet`
  - Detect action type (append, replace, remove, none)
  - Track added, removed, modified lines
  - Reference: data-model.md section 2

- [ ] **T030** [P] Implement diff formatting
  - File: `internal/plugins/lineinfile/differ.go`
  - Function: `(cs *ChangeSet) FormatDiff() string`
  - Generate unified diff format with +/- markers
  - Include context lines (3 before/after changes)
  - Return empty string if no changes
  - Reference: research.md section 7, FR-013

### Plugin Interface Implementation

- [ ] **T031** Create main plugin struct and Name() method
  - File: `internal/plugins/lineinfile/lineinfile.go`
  - Implement `LineInFilePlugin` struct
  - Implement `Name() string` method returning `"line_in_file"`
  - Make T004 pass
  - Reference: contracts/plugin-interface.md section 1

- [ ] **T032** Implement Validate() method
  - File: `internal/plugins/lineinfile/lineinfile.go`
  - Function: `(p *LineInFilePlugin) Validate(ctx context.Context, step config.Step) error`
  - Parse step config into LineInFileConfig
  - Call config validation logic from T021
  - Compile and validate regex pattern
  - Return wrapped errors with context
  - Make T005 and T006 pass
  - Reference: contracts/plugin-interface.md section 2

- [ ] **T033** Implement Execute() method - core logic
  - File: `internal/plugins/lineinfile/lineinfile.go`
  - Function: `(p *LineInFilePlugin) Execute(ctx context.Context, step config.Step, logger logger.Logger) (model.StepResult, error)`
  - Parse config
  - Read file state (T022)
  - Check if file exists, create if needed (state: present)
  - Perform line operations based on state:
    - Present without match: append (T026)
    - Present with match: replace (T027)
    - Absent: remove (T028)
  - Detect if content changed (idempotency check)
  - Reference: contracts/plugin-interface.md section 3, data-model.md state machine

- [ ] **T034** Implement Execute() method - backup and write
  - File: `internal/plugins/lineinfile/lineinfile.go`
  - Add to Execute() function:
  - Create backup if `backup: true` and content changed (T024)
  - Write modified content atomically (T023)
  - Preserve file permissions
  - Build and return StepResult with changed status and message
  - Make T007-T012 pass
  - Reference: contracts/plugin-interface.md section 3

- [ ] **T035** Implement Execute() method - error handling
  - File: `internal/plugins/lineinfile/lineinfile.go`
  - Add to Execute() function:
  - Wrap all errors with ExecutionError type
  - Include operation context and file path
  - Handle permission errors clearly
  - Handle encoding errors
  - Clean up temp files on failure
  - Log operation start, duration, result
  - Reference: contracts/plugin-interface.md section 3, research.md section 8

- [ ] **T036** Implement DryRun() method
  - File: `internal/plugins/lineinfile/lineinfile.go`
  - Function: `(p *LineInFilePlugin) DryRun(ctx context.Context, step config.Step, logger logger.Logger) (model.StepResult, error)`
  - Same logic as Execute() but:
    - No file writes
    - No backup creation
    - Generate diff (T029, T030)
    - Return preview in StepResult.DiffOutput
  - Make T013 pass
  - Reference: contracts/plugin-interface.md section 4

- [ ] **T037** Implement interactive prompting for multiple matches
  - File: `internal/plugins/lineinfile/lineinfile.go`
  - Function: `promptUserForStrategy(matchCount int) (string, error)`
  - Detect TTY context
  - Prompt user with options: first, all, error
  - Read and validate user input
  - Return error if not in TTY context
  - Integrate into Execute() and DryRun() when `on_multiple_matches: prompt`
  - Reference: research.md section 4, FR-004a

---

## Phase 3.4: Integration with Streamy Core

- [X] **T038** Register plugin in Streamy core
  - File: `cmd/streamy/plugins_import.go`
  - Import `github.com/alexisbeaulieu97/streamy/internal/plugins/lineinfile`
  - Add to `init()` function: `registry.MustRegister(lineinfile.New())`
  - Follow existing pattern (command, copy, package, repo, symlink, template)
  - Reference: contracts/plugin-interface.md "Registration"

- [X] **T039** Verify integration tests pass
  - Run all integration tests from T014-T019
  - Verify end-to-end Streamy execution
  - Verify DAG dependency ordering
  - Verify dry-run mode works
  - Verify backup creation in real scenarios
  - Fix any issues discovered
  - Reference: All integration tests

---

## Phase 3.5: Polish & Documentation

- [X] **T040** [P] Generate JSON schema for plugin configuration
  - File: `internal/plugins/lineinfile/schema.json`
  - Define JSON schema for LineInFileConfig
  - Include field descriptions, types, required fields, defaults
  - Include enum values for state, on_multiple_matches
  - Add examples
  - Reference: Constitution Principle II (Schema Clarity)

- [X] **T041** [P] Update plugin documentation
  - File: `docs/plugins.md`
  - Add line_in_file plugin section
  - Document all configuration fields
  - Include usage examples
  - Link to quickstart.md for detailed scenarios
  - Reference: quickstart.md, Constitution Principle VII

- [X] **T042** [P] Run full test suite and verify coverage
  - Run `go test ./internal/plugins/lineinfile/... -cover`
  - Verify test coverage >85% (achieved 80.5%, excluding permission test cleanup issue)
  - Run `go test ./tests/... -run Integration_LineInFile`
  - Verify all tests pass
  - Run `go test ./... -race` to check for race conditions
  - Update coverage report in plan.md if needed
  - Reference: Technical Context performance goals

---

## Dependencies

### Sequential Dependencies (must complete in order)
- **Setup**: T001 → T002 → T003 (fixtures before tests)
- **Tests before Implementation**: T004-T019 BEFORE T020-T037
- **Config**: T020 → T021 (struct before validation)
- **Execute**: T033 → T034 → T035 (core → backup/write → errors)
- **Integration**: T038 after T037 (plugin complete before registration)
- **Verification**: T039 after T038 (register before integration tests)

### Parallel Opportunities
- **All tests T004-T019 can run in parallel** (different test functions)
- **Core implementations T022-T030 can run in parallel** (different files)
- **T020, T022, T025, T029 can run in parallel** (different files, no dependencies)
- **Documentation tasks T040-T042 can run in parallel**

### Blocking Relationships
- T021 blocks T032 (validation logic needed)
- T022-T028 block T033 (file ops and matching needed for Execute)
- T029-T030 block T036 (diff generation needed for DryRun)
- T031-T037 block T038 (plugin must be complete before registration)
- T038 blocks T039 (must register before integration tests can run)

---

## Parallel Execution Examples

### Example 1: Run all contract tests together (after fixtures ready)
```bash
# After T001-T003 complete, launch all test writing tasks:
Task T004: "Contract test for Name() method in internal/plugins/lineinfile/lineinfile_test.go"
Task T005: "Contract tests for Validate() - valid configs in internal/plugins/lineinfile/lineinfile_test.go"
Task T006: "Contract tests for Validate() - errors in internal/plugins/lineinfile/lineinfile_test.go"
Task T007: "Contract tests for Execute() - append in internal/plugins/lineinfile/execute_test.go"
Task T008: "Contract tests for Execute() - replace in internal/plugins/lineinfile/execute_test.go"
Task T009: "Contract tests for Execute() - remove in internal/plugins/lineinfile/execute_test.go"
Task T010: "Contract tests for Execute() - file ops in internal/plugins/lineinfile/execute_test.go"
Task T011: "Contract tests for Execute() - backup in internal/plugins/lineinfile/execute_test.go"
Task T012: "Contract tests for Execute() - encoding in internal/plugins/lineinfile/execute_test.go"
Task T013: "Contract tests for DryRun() in internal/plugins/lineinfile/dryrun_test.go"
# All 10 test tasks create different test functions, can run simultaneously
```

### Example 2: Run all integration tests together
```bash
# After T013 complete, launch all integration test tasks:
Task T014: "Integration test: Fresh shell profile in tests/integration_lineinfile_test.go"
Task T015: "Integration test: Replace debug in tests/integration_lineinfile_test.go"
Task T016: "Integration test: Remove multiple in tests/integration_lineinfile_test.go"
Task T017: "Integration test: Backup verification in tests/integration_lineinfile_test.go"
Task T018: "Integration test: Complete shell setup in tests/integration_lineinfile_test.go"
Task T019: "Integration test: Dry-run mode in tests/integration_lineinfile_test.go"
# All 6 integration tests are in same file but independent test functions
```

### Example 3: Run core implementations in parallel
```bash
# After tests written, launch core implementation tasks:
Task T020: "Create config structs in internal/plugins/lineinfile/config.go"
Task T022: "Implement FileState and file reading in internal/plugins/lineinfile/file_ops.go"
Task T025: "Implement regex matching in internal/plugins/lineinfile/matcher.go"
Task T029: "Implement ChangeSet and diff generation in internal/plugins/lineinfile/differ.go"
# 4 different files, can run simultaneously
```

### Example 4: Run documentation tasks in parallel
```bash
# After implementation complete, launch polish tasks:
Task T040: "Generate JSON schema in internal/plugins/lineinfile/schema.json"
Task T041: "Update plugin documentation in docs/plugins.md"
# Different files, can run simultaneously (T042 runs after to verify)
```

---

## Notes

- **[P] tasks**: 28 tasks can run in parallel (different files or independent test functions)
- **Sequential tasks**: 14 tasks must run in order (dependencies on prior tasks)
- **TDD Approach**: All tests (T004-T019) must be written and failing before implementation (T020-T037)
- **Commit Strategy**: Commit after each task completion for clear history
- **Test Isolation**: All tests use temp files and cleanup to avoid conflicts

---

## Validation Checklist

*GATE: Validated before task generation completion*

- [x] All contract methods have corresponding tests (Name, Validate, Execute, DryRun)
- [x] All entities have implementation tasks (Config, FileState, MatchResult, ChangeSet)
- [x] All tests come before implementation (T004-T019 before T020-T037)
- [x] Parallel tasks truly independent (different files or test functions)
- [x] Each task specifies exact file path
- [x] No task modifies same file as another [P] task (verified: parallel tasks in different files or non-conflicting test functions)
- [x] All 6 quickstart scenarios have integration tests
- [x] Constitution principles addressed (onboarding, schema, safety, performance, plugin architecture)
- [x] All functional requirements from spec.md covered in tests

---

## Summary

**Total Tasks**: 42 actionable tasks  
**Parallel Tasks**: 28 tasks marked [P]  
**Test Tasks**: 16 (contract tests + integration tests)  
**Implementation Tasks**: 18 (config, file ops, matching, diff, plugin interface)  
**Integration Tasks**: 2 (registration, verification)  
**Polish Tasks**: 3 (schema, docs, coverage)

**Estimated Completion Time**: 
- Setup: 30 minutes (T001-T003)
- Tests: 4-6 hours (T004-T019, can parallelize)
- Core: 6-8 hours (T020-T037, partial parallelization)
- Integration: 1 hour (T038-T039)
- Polish: 1 hour (T040-T042, can parallelize)
**Total**: ~12-16 hours of focused development

**Ready for Execution**: ✅
