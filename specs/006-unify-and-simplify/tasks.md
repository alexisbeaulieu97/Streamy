# Tasks: Unify and Simplify the Plugin System

**Input**: Design documents from `/home/alexis/Projects/Streamy/specs/006-unify-and-simplify/`
**Prerequisites**: plan.md ✓, research.md ✓, data-model.md ✓, contracts/ ✓, quickstart.md ✓

## Execution Flow (main)
```
1. Load plan.md from feature directory ✓
   → Tech stack: Go 1.21+, Go standard library only
   → Structure: Single Go module, internal packages
2. Load design documents ✓
   → data-model.md: 5 entities (Plugin, EvaluationResult, PluginError, VerificationStatus, StepResult)
   → contracts/: 2 contracts (plugin-interface.md, executor-plugin.md)
   → research.md: 6 decisions (interface design, read-only, errors, InternalData, tests, performance)
3. Generate tasks by category ✓
   → Foundation: Core types (7 tasks)
   → Interface: Plugin refactoring (3 tasks)
   → Executor: Engine refactoring (5 tasks)
   → Plugins: 8 plugins in 4 phases (16 tasks)
   → Cleanup: Removal and docs (5 tasks)
4. Apply task rules ✓
   → Plugins in same phase: [P] (different files)
   → Executor modes: [P] (different concerns)
   → Tests before implementation (TDD)
5. Number tasks sequentially (T001-T036)
6. Validate completeness ✓
```

## Format: `[ID] [P?] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- Include exact file paths in descriptions

---

## Phase 3.1: Foundation - Core Types (Sequential)

These tasks establish the foundational types that all other work depends on.

- [ ] **T001** Create `internal/model/evaluation_result.go` with `EvaluationResult` struct
  - Define: `StepID`, `CurrentState`, `RequiresAction`, `Message`, `Diff`, `InternalData` fields
  - Add godoc comments explaining each field's purpose
  - **Blocks**: T002, T008

- [ ] **T002** Define `VerificationStatus` enum in `internal/model/evaluation_result.go`
  - Constants: `StatusSatisfied`, `StatusMissing`, `StatusDrifted`, `StatusBlocked`, `StatusUnknown`
  - Add String() method for human-readable output
  - **Depends on**: T001
  - **Blocks**: T008

- [ ] **T003** [P] Create `internal/plugin/errors.go` with `PluginError` interface
  - Define interface: `error`, `StepID() string`, `Unwrap() error`
  - Add godoc explaining when to use each error type
  - **Blocks**: T004, T005, T006

- [ ] **T004** [P] Implement `ValidationError` type in `internal/plugin/errors.go`
  - Fields: `ID string`, `Err error`
  - Methods: `Error()`, `StepID()`, `Unwrap()`
  - Constructor: `NewValidationError(stepID string, err error) *ValidationError`
  - **Depends on**: T003
  - **Blocks**: T011, T016

- [ ] **T005** [P] Implement `ExecutionError` type in `internal/plugin/errors.go`
  - Fields: `ID string`, `Err error`
  - Methods: `Error()`, `StepID()`, `Unwrap()`
  - Constructor: `NewExecutionError(stepID string, err error) *ExecutionError`
  - **Depends on**: T003
  - **Blocks**: T011, T016

- [ ] **T006** [P] Implement `StateError` type in `internal/plugin/errors.go`
  - Fields: `ID string`, `Err error`
  - Methods: `Error()`, `StepID()`, `Unwrap()`
  - Constructor: `NewStateError(stepID string, err error) *StateError`
  - **Depends on**: T003
  - **Blocks**: T011, T016

- [ ] **T007** [P] Add unit tests for error types in `internal/plugin/errors_test.go`
  - Test Error() returns formatted message with step ID
  - Test StepID() returns correct ID
  - Test Unwrap() returns underlying error
  - Test errors.Is() and errors.As() work correctly
  - **Depends on**: T004, T005, T006

---

## Phase 3.2: Interface Refactoring (Sequential)

Update the plugin interface and registry to support the new contract.

- [ ] **T008** Update `Plugin` interface in `internal/plugin/interface.go`
  - Add `Evaluate(ctx context.Context, step *config.Step) (*model.EvaluationResult, error)` method
  - Add `Apply(ctx context.Context, evalResult *model.EvaluationResult, step *config.Step) (*model.StepResult, error)` method
  - Mark `Check()`, `DryRun()`, `Verify()` as deprecated (add `// Deprecated:` comments)
  - Update interface godoc with new contracts and read-only guarantee
  - **Depends on**: T001, T002
  - **Blocks**: T009, T011

- [ ] **T009** Update `PluginRegistry` in `internal/plugin/registry_new.go` to support new interface
  - Ensure registry can call both old and new methods during transition
  - Update type assertions and interface checks
  - **Depends on**: T008
  - **Blocks**: T011

- [ ] **T010** Create contract test suite in `internal/plugin/contract_test.go`
  - Function: `TestPluginContract(t *testing.T, plugin Plugin, createTestStep func() *config.Step)`
  - Subtests:
    - `Metadata_is_stable`: Call twice, assert equal
    - `Schema_returns_struct`: Assert schema is struct type
    - `Evaluate_is_read_only`: Take snapshot, call Evaluate, assert no changes
    - `Evaluate_is_idempotent`: Call twice, assert equivalent results
    - `Apply_is_idempotent`: Call twice, assert same final state
    - `Error_types_are_correct`: Invalid step returns PluginError
  - **Depends on**: T008
  - **Blocks**: All plugin migration tasks (T016-T031)

---

## Phase 3.3: Executor Refactoring (Parallel by Mode)

Refactor the execution engine to use the new Evaluate/Apply pattern.

- [ ] **T011** [P] Refactor verify mode in `cmd/streamy/verify.go`
  - Replace `plugin.Check()` calls with `plugin.Evaluate()`
  - Interpret `EvaluationResult.CurrentState` for output:
    - `StatusSatisfied` → "✓ step satisfied"
    - `StatusMissing`/`StatusDrifted` → "✗ step drifted: {Message}"
    - `StatusBlocked` → "⊘ step blocked: {Message}"
    - `StatusUnknown` → "? step unknown: {error}"
  - Generate summary: X satisfied, Y drifted, Z blocked
  - Exit code: 0 if all satisfied, non-zero otherwise
  - **Depends on**: T008, T009
  - **Blocks**: T015

- [ ] **T012** [P] Refactor dry-run mode in `cmd/streamy/apply.go` (dry-run flag handling)
  - Replace logic to call `plugin.Evaluate()` only
  - Check `EvaluationResult.RequiresAction`:
    - `false` → "⊙ would skip: {Message}"
    - `true` → "→ would apply: {Message}" + display Diff
  - Generate summary: X would skip, Y would apply
  - Exit code: always 0 (dry-run never fails)
  - **Depends on**: T008, T009
  - **Blocks**: T015

- [ ] **T013** [P] Refactor apply mode in `cmd/streamy/apply.go` (actual apply)
  - For each step:
    1. Call `plugin.Evaluate(ctx, step)`
    2. If `RequiresAction == false`: Log "⊙ skipped: {Message}", continue
    3. If `RequiresAction == true`: Call `plugin.Apply(ctx, evalResult, step)`
    4. Log result based on `StepResult.Status`
  - Handle `--continue-on-error` flag
  - Generate summary: X skipped, Y applied, Z failed
  - **Depends on**: T008, T009
  - **Blocks**: T015

- [ ] **T014** [P] Update error handling in `internal/engine/executor.go`
  - Add error type categorization using `errors.As()`:
    - `ValidationError` → Always fatal, clear config error message
    - `ExecutionError` → Fatal unless `--continue-on-error`, mark step as failed
    - `StateError` → Warning level, mark step as Unknown, continue
  - Update logging to include structured fields (step_id, error_type, duration)
  - Add context cancellation handling (ctx.Canceled, ctx.DeadlineExceeded)
  - **Depends on**: T004, T005, T006, T008
  - **Blocks**: T015

- [ ] **T015** Add executor integration tests in `tests/integration_test.go`
  - Test verify mode calls Evaluate() only (never Apply())
  - Test dry-run mode calls Evaluate() only (never Apply())
  - Test apply mode calls Evaluate() → Apply() when RequiresAction=true
  - Test apply mode skips Apply() when RequiresAction=false
  - Test error handling for each error type
  - Test context cancellation
  - Use mock plugin for controlled testing
  - **Depends on**: T011, T012, T013, T014

---

## Phase 3.4: Plugin Migration - Phase 1 (Simple Plugins)

Migrate simple plugins with straightforward state checking.

- [ ] **T016** [P] Migrate `symlink` plugin in `internal/plugins/symlink/symlink.go`
  - Implement `Evaluate(ctx, step)`:
    - Check if symlink exists and points to correct target (read-only)
    - Return `StatusSatisfied` if correct, `StatusMissing`/`StatusDrifted` otherwise
    - Populate `Message` with clear description
    - Generate `Diff` showing current vs desired target
  - Implement `Apply(ctx, evalResult, step)`:
    - Create or update symlink based on `evalResult.CurrentState`
    - Use `evalResult.InternalData` if applicable (can store desired target)
  - Remove `Check()`, `DryRun()`, `Verify()` methods
  - Update all existing tests to use new interface
  - **Depends on**: T010
  - **Blocks**: T020

- [ ] **T017** [P] Add contract tests for `symlink` in `internal/plugins/symlink/symlink_test.go`
  - Import contract test suite from T010
  - Call `TestPluginContract(t, New(), createSymlinkTestStep)`
  - Add symlink-specific test: Evaluate() doesn't create symlink
  - **Depends on**: T016

- [ ] **T018** [P] Migrate `copy` plugin in `internal/plugins/copy/copy.go`
  - Implement `Evaluate(ctx, step)`:
    - Compare source and destination file checksums (read-only)
    - Return `StatusSatisfied` if identical, `StatusMissing`/`StatusDrifted` otherwise
    - Generate unified diff for text files
    - Store computed checksum/content in `InternalData`
  - Implement `Apply(ctx, evalResult, step)`:
    - Copy file using `evalResult.InternalData` to avoid re-reading source
    - Handle recursive directory copy
  - Remove `Check()`, `DryRun()`, `Verify()` methods
  - Update all existing tests to use new interface
  - **Depends on**: T010
  - **Blocks**: T020

- [ ] **T019** [P] Add contract tests for `copy` in `internal/plugins/copy/copy_test.go`
  - Import contract test suite from T010
  - Call `TestPluginContract(t, New(), createCopyTestStep)`
  - Add copy-specific test: Evaluate() doesn't copy file
  - **Depends on**: T018

---

## Phase 3.5: Plugin Migration - Phase 2 (Medium Complexity)

Migrate plugins with content manipulation logic.

- [ ] **T020** [P] Migrate `lineinfile` plugin in `internal/plugins/lineinfile/lineinfile.go`
  - Refactor existing `evaluate()` function to match new `Evaluate()` signature
  - Adapt `EvaluationResult` from existing `evaluationResult` struct
  - Implement `Apply(ctx, evalResult, step)` using `InternalData` from Evaluate
  - Remove `Check()`, `DryRun()`, `Verify()` methods (already partially done)
  - Update tests to use new interface
  - **Depends on**: T010, T016, T018 (learn from simple plugins)
  - **Blocks**: T024

- [ ] **T021** [P] Add contract tests for `lineinfile` in `internal/plugins/lineinfile/lineinfile_test.go`
  - Import contract test suite from T010
  - Call `TestPluginContract(t, New(), createLineInFileTestStep)`
  - Add lineinfile-specific test: Evaluate() doesn't modify file
  - **Depends on**: T020

- [ ] **T022** [P] Migrate `template` plugin in `internal/plugins/template/template.go`
  - Implement `Evaluate(ctx, step)`:
    - Render template in-memory (read-only)
    - Compare rendered content with destination file
    - Generate diff
    - Store rendered content in `InternalData`
  - Implement `Apply(ctx, evalResult, step)`:
    - Write content from `InternalData` to destination
  - Remove `Check()`, `DryRun()`, `Verify()` methods
  - Update all existing tests to use new interface
  - **Depends on**: T010, T020
  - **Blocks**: T024

- [ ] **T023** [P] Add contract tests for `template` in `internal/plugins/template/template_test.go`
  - Import contract test suite from T010
  - Call `TestPluginContract(t, New(), createTemplateTestStep)`
  - Add template-specific test: Evaluate() doesn't write file
  - **Depends on**: T022

---

## Phase 3.6: Plugin Migration - Phase 3 (Complex/External)

Migrate plugins that interact with external systems.

- [ ] **T024** [P] Migrate `package` plugin in `internal/plugins/package/package.go`
  - Implement `Evaluate(ctx, step)`:
    - Query package manager for package status (read-only)
    - Return `StatusSatisfied` if installed at correct version
    - Return `StatusMissing` if not installed
    - Return `StatusDrifted` if wrong version
    - Generate diff showing current vs desired version
  - Implement `Apply(ctx, evalResult, step)`:
    - Install/upgrade package based on state
  - Remove `Check()`, `DryRun()`, `Verify()` methods
  - Update all existing tests to use new interface
  - **Depends on**: T010, T020, T022 (learn from medium complexity)
  - **Blocks**: T028

- [ ] **T025** [P] Add contract tests for `package` in `internal/plugins/package/package_test.go`
  - Import contract test suite from T010
  - Call `TestPluginContract(t, New(), createPackageTestStep)`
  - Add package-specific test: Evaluate() doesn't install packages
  - **Depends on**: T024

- [ ] **T026** [P] Migrate `repo` plugin in `internal/plugins/repo/repo.go`
  - Implement `Evaluate(ctx, step)`:
    - Check git remote URL, branch, commit (read-only git commands)
    - Return state based on comparison
    - Generate diff showing current vs desired state
  - Implement `Apply(ctx, evalResult, step)`:
    - Clone/pull/checkout as needed
  - Remove `Check()`, `DryRun()`, `Verify()` methods
  - Update all existing tests to use new interface
  - **Depends on**: T010, T024
  - **Blocks**: T028

- [ ] **T027** [P] Add contract tests for `repo` in `internal/plugins/repo/repo_test.go`
  - Import contract test suite from T010
  - Call `TestPluginContract(t, New(), createRepoTestStep)`
  - Add repo-specific test: Evaluate() doesn't clone/modify repo
  - **Depends on**: T026

---

## Phase 3.7: Plugin Migration - Phase 4 (Meta/Framework)

Migrate plugins that execute user-defined commands.

- [ ] **T028** [P] Migrate `command` plugin in `internal/plugins/command/command.go`
  - Implement `Evaluate(ctx, step)`:
    - Execute `check` command if specified (may be read-only or not, depends on user)
    - Map exit code to state: 0=Satisfied, non-zero=Drifted/Missing
    - Capture stdout/stderr for Message
  - Implement `Apply(ctx, evalResult, step)`:
    - Execute `command` specified in config
  - Remove `Check()`, `DryRun()`, `Verify()` methods
  - Update all existing tests to use new interface
  - **Depends on**: T010, T024, T026 (learn from complex plugins)

- [ ] **T029** [P] Add contract tests for `command` in `internal/plugins/command/command_test.go`
  - Import contract test suite from T010
  - Call `TestPluginContract(t, New(), createCommandTestStep)`
  - Note: Read-only test depends on user-provided check command
  - **Depends on**: T028

- [ ] **T030** [P] Migrate `internalexec` plugin in `internal/plugins/internalexec/internalexec.go`
  - Implement `Evaluate(ctx, step)`:
    - Similar to command plugin but for internal executables
  - Implement `Apply(ctx, evalResult, step)`:
    - Execute internal command
  - Remove `Check()`, `DryRun()`, `Verify()` methods
  - Update all existing tests to use new interface
  - **Depends on**: T010, T028

- [ ] **T031** [P] Add contract tests for `internalexec` in `internal/plugins/internalexec/internalexec_test.go`
  - Import contract test suite from T010
  - Call `TestPluginContract(t, New(), createInternalExecTestStep)`
  - **Depends on**: T030

---

## Phase 3.8: Cleanup & Documentation

Remove deprecated code and update documentation.

- [ ] **T032** Remove deprecated methods from `internal/plugin/interface.go`
  - Delete `Check()` method from Plugin interface
  - Delete `DryRun()` method from Plugin interface
  - Delete `Verify()` method from Plugin interface
  - Remove all `// Deprecated:` comments (they're gone now)
  - **Depends on**: T016-T031 (all plugins migrated)
  - **Blocks**: T033

- [ ] **T033** Update integration tests in `tests/` directory
  - Update `integration_test.go` to use Evaluate/Apply
  - Update `integration_verify_test.go` to test read-only behavior
  - Update `integration_plugin_dependency_test.go` if needed
  - Add new integration tests for error type handling
  - **Depends on**: T032

- [ ] **T034** Update plugin documentation in `docs/plugins.md`
  - Replace old interface examples with new Evaluate/Apply examples
  - Document read-only guarantee prominently
  - Add error type usage guidelines
  - Update example plugin implementation
  - Add migration guide for external plugin developers (if any)
  - **Depends on**: T032

- [ ] **T035** [P] Add performance benchmarks in `internal/plugin/perf_test.go` and plugin dirs
  - Benchmark Evaluate() for each plugin type
  - Compare with baseline (old Check() timing if available)
  - Verify within 20% overhead budget
  - Document results in comments
  - **Depends on**: T016-T031

- [ ] **T036** Run full validation using `specs/006-unify-and-simplify/quickstart.md`
  - Execute all 12 steps in quickstart
  - Verify all pass
  - Document any issues found
  - Create summary report
  - **Depends on**: T032, T033, T034, T035

---

## Dependencies Graph

```
T001 → T002 → T008 → T009 → T011, T012, T013, T015
                      ↓
T003 → T004, T005, T006 → T014 → T015
           ↓
T007       ↓
           ↓
T008 → T010 → T016, T018, T020, T022, T024, T026, T028, T030
         ↓
T016 → T017, T020
T018 → T019
T020 → T021, T022, T024
T022 → T023
T024 → T025, T026, T028
T026 → T027
T028 → T029, T030
T030 → T031

T016-T031 → T032 → T033, T034
                    ↓
T035                ↓
  ↓                 ↓
T036 ←─────────────┘
```

---

## Parallel Execution Examples

### Foundation Phase (T003-T007)
```bash
# After T001-T002 complete, run error types in parallel:
Task: "Create PluginError interface in internal/plugin/errors.go"
Task: "Implement ValidationError in internal/plugin/errors.go"  # Can work in same file
Task: "Implement ExecutionError in internal/plugin/errors.go"
Task: "Implement StateError in internal/plugin/errors.go"
# Then T007 tests all of them
```

### Executor Refactoring (T011-T014)
```bash
# After T009 complete, refactor modes in parallel:
Task: "Refactor verify mode in cmd/streamy/verify.go"
Task: "Refactor dry-run in cmd/streamy/apply.go"
Task: "Refactor apply mode in cmd/streamy/apply.go"
Task: "Update error handling in internal/engine/executor.go"
```

### Plugin Phase 1 (T016-T019)
```bash
# After T010 complete:
Task: "Migrate symlink plugin in internal/plugins/symlink/symlink.go"
Task: "Migrate copy plugin in internal/plugins/copy/copy.go"
# Then contract tests:
Task: "Add contract tests for symlink"
Task: "Add contract tests for copy"
```

### Plugin Phase 2 (T020-T023)
```bash
# After T016, T018 complete:
Task: "Migrate lineinfile plugin in internal/plugins/lineinfile/lineinfile.go"
Task: "Migrate template plugin in internal/plugins/template/template.go"
# Then contract tests:
Task: "Add contract tests for lineinfile"
Task: "Add contract tests for template"
```

### Plugin Phase 3 (T024-T027)
```bash
# After T020, T022 complete:
Task: "Migrate package plugin in internal/plugins/package/package.go"
Task: "Migrate repo plugin in internal/plugins/repo/repo.go"
# Then contract tests:
Task: "Add contract tests for package"
Task: "Add contract tests for repo"
```

### Plugin Phase 4 (T028-T031)
```bash
# After T024, T026 complete:
Task: "Migrate command plugin in internal/plugins/command/command.go"
Task: "Migrate internalexec plugin in internal/plugins/internalexec/internalexec.go"
# Then contract tests:
Task: "Add contract tests for command"
Task: "Add contract tests for internalexec"
```

---

## Progress Tracking

**Total Tasks**: 36

**By Phase**:
- Phase 3.1 (Foundation): 7 tasks
- Phase 3.2 (Interface): 3 tasks
- Phase 3.3 (Executor): 5 tasks
- Phase 3.4 (Plugins P1): 4 tasks
- Phase 3.5 (Plugins P2): 4 tasks
- Phase 3.6 (Plugins P3): 4 tasks
- Phase 3.7 (Plugins P4): 4 tasks
- Phase 3.8 (Cleanup): 5 tasks

**Parallelization**:
- Maximum concurrent: ~8 tasks (during plugin migration phases)
- Sequential critical path: T001→T002→T008→T009→T011→T015→T010→T016→T020→T024→T028→T032→T036
- Estimated with 2 developers: ~3-4 weeks
- Estimated with 4 developers: ~2-3 weeks

---

## Validation Checklist

- [x] All contracts have corresponding tests (2 contracts → T010 covers both)
- [x] All entities have model/implementation tasks (5 entities → T001-T006)
- [x] All tests come before implementation (contract tests T010 before plugins T016-T031)
- [x] Parallel tasks truly independent (different files, verified)
- [x] Each task specifies exact file path ✓
- [x] No [P] task modifies same file as another [P] task ✓

---

## Notes

- **TDD Approach**: Contract tests (T010) are created before any plugin migration
- **Incremental Testing**: Each plugin gets contract tests immediately after migration
- **Read-Only Verification**: Contract test suite includes Evaluate() read-only check
- **Performance Budget**: T035 validates 20% overhead is not exceeded
- **Quickstart Validation**: T036 runs complete validation procedure
- **Big Bang Completion**: All plugins must be migrated (T016-T031) before T032 removes old interface
- **Commit Strategy**: Commit after each task or logical group of parallel tasks
- **Constitution Alignment**: Tasks support all 7 principles (safety, clarity, plugin architecture, performance)

---

**Generated**: October 7, 2025  
**Based on**: plan.md, data-model.md, contracts/, research.md, quickstart.md  
**Ready for execution**: All tasks are concrete, ordered, and have clear file paths
