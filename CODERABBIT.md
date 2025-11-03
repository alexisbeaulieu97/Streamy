# CodeRabbit Findings

This document contains potential issues identified by CodeRabbit analysis that require verification and potential action.

# Process

- Go through each reported issue identified.
- For each, investigate the code to verify the claim and determine if it is valid.
- If invalid, update the issue to document why it is invalid (false positive, not applicable, etc.).
- If valid, implement the suggested fix or an improved solution.
- Test the fix to ensure it resolves the issue without introducing new problems.
- After validation, mark the issue as resolved `- [x]` and add a brief note about what was done.

---

## [ ] Missing timestamp fields in PipelineExecutionSummary domain struct
**File:** `specs/010-pipeline-dependencies/quickstart.md`
**Lines:** 760-767
**Reported Issue:** PipelineExecutionSummary is missing StartedAt and FinishedAt timestamp fields used by persistence code
**Suggested Fix:** Update the struct to include StartedAt and FinishedAt (type time.Time) so the persistence layer can read/write timestamps, and keep Duration as-is (or derive Duration from FinishedAt-StartedAt)
**Verification Needed:** Confirm the persistence code references these timestamp fields and the struct definition is missing them

---

## [ ] Type mismatch in ExecutionResult.BlockedBy field
**File:** `specs/010-pipeline-dependencies/quickstart.md`
**Lines:** 64-70
**Reported Issue:** ExecutionResult.BlockedBy is declared as a single string but the add command builds a []string and assigns it to pipeline.BlockedBy
**Suggested Fix:** Choose a single consistent type (either string for one blocker or []string for multiple) and update the ExecutionResult type, JSON tag, registry/storage schema, the add command variable declaration and assignments, and all places that read/write pipeline.BlockedBy
**Verification Needed:** Verify the type mismatch exists and determine which type is more appropriate for the use case

---

## [ ] Non-existent method call in test harness
**File:** `specs/010-pipeline-dependencies/quickstart.md`
**Lines:** 1370-1376
**Reported Issue:** Test harness calls a non-existent h.useCase.Execute(...) method
**Suggested Fix:** Replace that call with the correct method on OrchestrateUseCase (e.g., h.useCase.Orchestrate(context.Background(), pipelineID, dryRun)) or add an Execute wrapper that forwards to Orchestrate
**Verification Needed:** Check the actual OrchestrateUseCase interface and confirm the correct method signature

---

## [ ] Incorrect field references in persistence code
**File:** `specs/010-pipeline-dependencies/quickstart.md`
**Lines:** 1005-1047 (notably line 1033)
**Reported Issue:** Persistence code references non-existent or misnamed fields and a missing helper
**Suggested Fix:** Replace result.Plan.Levels with result.ExecutionLevels, replace result.FinishedAt with result.EndTime and result.StartedAt with result.StartTime, remove or replace any use of result.FailedCount, and add a new flattenLevels(helper) that accepts the ExecutionLevels structure
**Verification Needed:** Verify the actual field names in the domain types and confirm the mismatches

---

## [ ] Missing ReconcileStatuses method on registry interface
**File:** `specs/010-pipeline-dependencies/quickstart.md`
**Lines:** 326 (also applies to lines 663-680)
**Reported Issue:** Call to op.registry.ReconcileStatuses(op.statusCache) refers to a method not declared on the registry interface
**Suggested Fix:** Add a ReconcileStatuses(ctx context.Context, statusCache StatusCache) ([]PipelineID, error) method to the registry interface or move this into a dedicated ReconciliationService interface
**Verification Needed:** Check the registry interface definition and determine if this method should be added or if a different service should handle reconciliation

---

## [x] Import organization issues in test files
**File:** `cmd/streamy/list_tree_test.go`
**Lines:** 3-14
**Reported Issue:** Import block should be reorganized into three groups with blank lines and redundant alias removed
**Suggested Fix:** Reorder imports into three groups (standard library, third-party, local packages) separated by blank lines, remove redundant alias, and run goimports -w . to apply grouping
**Verification Needed:** Confirm the import organization issues and run goimports to fix
**Resolution:** Imports reordered and redundant alias removed in `cmd/streamy/list_tree_test.go`; gofmt/goimports applied.

---

## [x] Import organization issues in orchestration test
**File:** `internal/application/orchestration/usecase_test.go`
**Lines:** 3-15
**Reported Issue:** Imports are not organized per goimports convention and named aliases for testify's assert and require are redundant
**Suggested Fix:** Reorder imports into three groups with blank lines, remove explicit aliases, and run goimports/gofmt to ensure formatting is correct
**Verification Needed:** Check current import organization and apply goimports fix
**Resolution:** Updated import groups and removed testify aliases in `internal/application/orchestration/usecase_test.go`; gofmt/goimports applied.

---

## [x] Dependency validation missing whitespace trimming
**File:** `internal/infrastructure/config/schema.go`
**Lines:** 45-52
**Reported Issue:** Dependency strings are validated without trimming, unlike ID and Version earlier
**Suggested Fix:** Trim each dependency with strings.TrimSpace before calling registry.ValidateCanonicalPipelineID and use the trimmed value in validation and error payload
**Verification Needed:** Confirm that ID and Version validation includes trimming but dependencies do not
**Resolution:** Dependencies are normalized with `strings.TrimSpace` before validation and persistence in `internal/infrastructure/config/schema.go`.

---

## [x] Conditional completion log missing in non-interactive runs
**File:** `cmd/streamy/orchestrate.go`
**Lines:** 98-100
**Reported Issue:** Current condition prevents emitting the completion log in non-interactive runs
**Suggested Fix:** Change the if to log regardless of opts.NonInteractive by removing the !opts.NonInteractive check so it only checks app.Logger != nil && logger != nil
**Verification Needed:** Determine if the current behavior is intentional or if logs should always be emitted
**Resolution:** Removed the non-interactive guard so completion logs are emitted consistently in `cmd/streamy/orchestrate.go`.

---

## [x] Incorrect indentation in status cache
**File:** `internal/registry/status_cache.go`
**Lines:** 144-157
**Reported Issue:** If block starting at line 149 has incorrect indentation (spaces instead of tabs) which breaks gofmt
**Suggested Fix:** Re-indent that block using tabs to match surrounding code and run gofmt and goimports
**Verification Needed:** Confirm the indentation issue and apply gofmt fix
**Resolution:** Re-indented the orchestration history trimming block and ran gofmt on `internal/registry/status_cache.go`.

---

## [x] Non-deterministic map iteration in cycle detection
**File:** `internal/registry/graph.go`
**Lines:** 161-178
**Reported Issue:** Iteration over the map g.nodes is non-deterministic which can return different cycle paths across runs
**Suggested Fix:** Collect all node keys into a slice, sort the slice, then iterate over the sorted slice so DetectCycle() visits nodes in a deterministic order
**Verification Needed:** Verify the map iteration and implement deterministic ordering
**Resolution:** Gathered node keys into a sorted slice before traversal and aligned helper sorting with `slices.Sort` in `internal/registry/graph.go`.

---

## [ ] Mixed pipeline ID formats in tests (refactor suggestion)
**File:** `specs/010-pipeline-dependencies/quickstart.md`
**Lines:** 1252-1289
**Reported Issue:** Tests mix bare pipeline IDs and canonical IDs with versions
**Suggested Fix:** Change all pipeline identifiers to the canonical "@" format, add top-of-file constants for each pipeline ID, and replace literal strings with those constants
**Verification Needed:** Review test consistency and implement canonical ID format throughout

---

## [x] Ambiguous wording in MEMORY.md status line
**File:** `MEMORY.md`
**Lines:** 4
**Reported Issue:** Phrase "CodeRabbit clean-up wrapped" is ambiguous about completion status
**Suggested Fix:** Update to use clear, unambiguous wording indicating status (e.g., "CodeRabbit clean-up completed" or "CodeRabbit clean-up in progress")
**Verification Needed:** Determine the actual status and update with clear, consistent wording
**Resolution:** Reworded the status line to "CodeRabbit clean-up completed" in `MEMORY.md`.

---

## [x] Contradictory test/lint status statements in MEMORY.md
**File:** `MEMORY.md`
**Lines:** 14 and 20
**Reported Issue:** File contains contradictory statements about test/lint status
**Suggested Fix:** Verify current CI state and make both lines consistent, stating either they pass with date/commit reference or fail with known issues
**Verification Needed:** Run golangci-lint and go test locally to determine actual status
**Resolution:** Consolidated the test/lint messaging into a single "Verification Status" entry that notes outstanding baseline failures in `MEMORY.md`.

---

## [x] Missing functional requirement for force flag safety
**File:** `specs/010-pipeline-dependencies/spec.md`
**Lines:** 110
**Reported Issue:** Missing requirement for --force to require explicit --yes flag in non-interactive environments
**Suggested Fix:** Add FR-013b stating that when running with --force in non-interactive environments the command MUST require an explicit --yes flag to proceed
**Verification Needed:** Add the new functional requirement and update related text
**Resolution:** Requirement already captured in FR-013a (explicit `--yes` needed for non-interactive `--force` runs); verified wording in `specs/010-pipeline-dependencies/spec.md`.

---

## [x] Incomplete command example in spec
**File:** `specs/010-pipeline-dependencies/spec.md`
**Lines:** 37
**Reported Issue:** Command example uses incomplete pipeline identifier format
**Suggested Fix:** Update example to use canonical id@version format (e.g., streamy run --registry app-deploy@1.0)
**Verification Needed:** Update the example to consistently show full required format
**Resolution:** Acceptance criteria now reference `streamy run --registry <id>@<version>` to reinforce canonical identifiers in `specs/010-pipeline-dependencies/spec.md`.

---

## [x] Unclear BlockedBy field lifecycle and persistence
**File:** `specs/010-pipeline-dependencies/DESIGN_RESOLUTION.md`
**Lines:** 28, 78, 181, 220-222
**Reported Issue:** Lifecycle and persistence of BlockedBy field is ambiguous regarding registration-time vs runtime blocks
**Suggested Fix:** Clarify that registration-time BlockedBy is persisted in registry.json while runtime upstream-failure blocks use separate cache, and describe reconciliation algorithm
**Verification Needed:** Add explicit lifecycle documentation with persistence strategy
**Resolution:** Added an explicit persistence note distinguishing registry-time dependency blocks from runtime status cache reasons in `specs/010-pipeline-dependencies/DESIGN_RESOLUTION.md`.

---

## [ ] Incomplete implementation checklist
**File:** `specs/010-pipeline-dependencies/DESIGN_RESOLUTION.md`
**Lines:** 417-431
**Reported Issue:** Implementation Checklist is incomplete and contains vague merged items
**Suggested Fix:** Expand with explicit items for DependencyGraph, status reconciliation, orchestration cache, Executor interface, and split blocked-execution behavior into distinct checkboxes
**Verification Needed:** Add precise, actionable checklist items for each component

---

## [ ] Ambiguous pipeline persistence model
**File:** `specs/010-pipeline-dependencies/DESIGN_RESOLUTION.md`
**Lines:** 170-185
**Reported Issue:** Pipeline struct's persistence model is ambiguous about which fields are persisted vs runtime-only
**Suggested Fix:** Clearly list persisted fields (ID, Name, Path, Description, RegisteredAt, Dependencies) vs runtime-only (Status, BlockedBy, LastRun, LastResult) and describe startup reconstruction
**Verification Needed:** Define persistence model with explicit field categorization

---

## [ ] Missing mock executor interface in test harness
**File:** `specs/010-pipeline-dependencies/quickstart.md`
**Lines:** 1349-1370
**Reported Issue:** Test harness refers to h.executor.failures but no mock executor type or contract is defined
**Suggested Fix:** Add a mock executor interface with Execute and SetFailure methods, provide concrete implementation with failures map and mutex
**Verification Needed:** Define proper mock executor interface and update harness to use it

---

## [ ] Race condition in dependency validation
**File:** `specs/010-pipeline-dependencies/quickstart.md`
**Lines:** 605-634
**Reported Issue:** Code temporarily mutates registry during validation which can race with concurrent operations
**Suggested Fix:** Avoid mutating shared state by constructing in-memory view for validation, or acquire write lock for entire validation window
**Verification Needed:** Implement race-free validation approach

---

## [ ] Regex validation contradiction for pipeline IDs
**File:** `specs/010-pipeline-dependencies/quickstart.md`
**Lines:** 220-224
**Reported Issue:** pipelineIDPattern regex omits @ but validation message claims IDs may include @
**Suggested Fix:** Update regex to allow optional version suffix ^[a-zA-Z0-9_-]+(@[a-zA-Z0-9_.]+)?$ or move @validation into ensureCanonicalDependencies
**Verification Needed:** Fix regex to match validation message and adjust tests

---

## [ ] Ignored error in pipeline lookup
**File:** `specs/010-pipeline-dependencies/quickstart.md`
**Lines:** 962-975
**Reported Issue:** Call to uc.registry.GetPipeline(id) ignores returned error which can lead to ambiguous failures
**Suggested Fix:** Capture and check error immediately, return clear contextual error if pipeline not found
**Verification Needed:** Add proper error handling for pipeline lookup

---

## [x] Resource leak in context creation
**File:** `cmd/streamy/run.go`
**Lines:** 64
**Reported Issue:** Code discards cancel function returned by app.CommandContext which can leak resources
**Suggested Fix:** Capture cancel function and immediately defer its call after obtaining context
**Verification Needed:** Add proper resource cleanup with defer cancel()
**Resolution:** `executeRun` now derives a cancellable execution context and defers the cancel function before invoking apply/orchestrate flows (`cmd/streamy/run.go`).

---

## [x] Cross-platform chmod issue
**File:** `internal/registry/fileio.go`
**Lines:** 22-25
**Reported Issue:** tmpFile.Chmod(...) has limited cross-platform effect (notably on Windows)
**Suggested Fix:** Close temp file, rename to final path, then call os.Chmod on final path to set permissions
**Verification Needed:** Update flow to work cross-platform with proper error handling
**Resolution:** Permission changes now occur after the atomic rename via `os.Chmod` on the final path in `internal/registry/fileio.go`.

---

## [x] Inconsistent sort usage in graph.go
**File:** `internal/registry/graph.go`
**Lines:** 3-8, 45-50, 111-112, 156-157, 231-231
**Reported Issue:** File mixes sort.Strings with slices.Sort
**Suggested Fix:** Replace all sort.Strings calls with slices.Sort and remove unused "sort" import
**Verification Needed:** Standardize on slices.Sort and clean up imports
**Resolution:** Replaced all `sort.Strings` calls with `slices.Sort` and removed the redundant import in `internal/registry/graph.go`.

---

## [x] Manual flag parsing in main.go
**File:** `cmd/streamy/main.go`
**Lines:** 27-32
**Reported Issue:** Manual os.Args parsing instead of using Cobra's persistent flags
**Suggested Fix:** Register persistent verbose flag on root command and move log-level initialization into PersistentPreRun
**Verification Needed:** Refactor to use proper Cobra flag handling
**Resolution:** Swapped to a preflight `pflag.FlagSet` for persistent verbose detection and moved runtime level updates into root command lifecycle hooks (`cmd/streamy/main.go`, `cmd/streamy/root.go`).

---

## [ ] Incomplete orchestration aggregate-count rules in data model
**File:** `specs/010-pipeline-dependencies/data-model.md`
**Lines:** 294-298 (and Validation Summary table at ~576)
**Reported Issue:** Orchestration aggregate-count rules are incomplete
**Suggested Fix:** Add explicit validations that ReadyCount equals number of PipelineResults entries with Success == true; FailedCount equals number with Success == false AND BlockedBy == ""; and BlockedCount equals number with BlockedBy != ""
**Verification Needed:** Update Validation Rules list and Validation Summary table with O(1) counter definitions

---

## [ ] Missing validation rules in data model summary table
**File:** `specs/010-pipeline-dependencies/data-model.md`
**Lines:** 576-586
**Reported Issue:** Validation Rules Summary table is missing several rules described elsewhere
**Suggested Fix:** Add rows for Pipeline dependencies format, Dependencies normalization, ExecutionResult consistency, OrchestrationResult count verification, and CachedStatus BlockedBy requirements
**Verification Needed:** Add missing rows to match existing table style with clear error messages

---

## [ ] Breaking change without migration guide
**File:** `specs/010-pipeline-dependencies/quickstart.md`
**Lines:** 188-196
**Reported Issue:** New required PipelineConfig.ID field is a breaking change for existing YAML pipeline files without migration guidance
**Suggested Fix:** Update migration guide or prerequisites to warn users and provide clear migration recipe with before/after examples
**Verification Needed:** Add explicit migration instructions for breaking ID field requirement

---

## [x] Status cache invalidation not persisted
**File:** `cmd/streamy/remove.go`
**Lines:** 168-172
**Reported Issue:** Status cache is invalidated in memory but not persisted to disk
**Suggested Fix:** After calling statusCache.Invalidate(pipelineID), call statusCache.Save() and handle its error with proper logging
**Verification Needed:** Ensure Save() is called when Invalidate succeeds and errors are properly handled
**Resolution:** Marked as false positive; `StatusCache.Invalidate` already persists the update by invoking `saveLocked`, so an additional `Save` call is unnecessary.

---

## [ ] Ignored error in dependency graph creation
**File:** `specs/010-pipeline-dependencies/quickstart.md`
**Lines:** 897
**Reported Issue:** Call to registry.NewDependencyGraph ignores returned error
**Suggested Fix:** Capture the error and handle it properly instead of discarding - either return error, log with context, or document why safe to ignore
**Verification Needed:** Add proper error handling for dependency graph creation

---

## [ ] Conflated status enums in specification
**File:** `specs/010-pipeline-dependencies/spec.md`
**Lines:** 127-130
**Reported Issue:** Pipeline Status enum conflates registration-time availability with execution-time outcomes
**Suggested Fix:** Split into two distinct enums: RegistrationStatus {Ready, Blocked} and ExecutionStatus {Pending, Running, Succeeded, Failed, BlockedByUpstream}
**Verification Needed:** Refactor status handling and update all references with state-transition notes

---

## [ ] Missing YAML schema definition in specification
**File:** `specs/010-pipeline-dependencies/spec.md`
**Lines:** 94-97
**Reported Issue:** Spec mentions dependencies field but lacks formal YAML schema definition
**Suggested Fix:** Define exact schema as YAML sequence of strings with canonical id@version regex, empty list validity, and duplicate handling rules
**Verification Needed:** Add formal schema definition with validation rules and examples

---

## [ ] Order-sensitive assertions in graph tests
**File:** `internal/registry/graph_test.go`
**Lines:** 48-52, 55
**Reported Issue:** Tests use order-sensitive assertions on unordered collections causing flaky tests
**Suggested Fix:** Replace assert.Equal with assert.ElementsMatch for unordered collections; use per-level ElementsMatch comparisons for levels slice
**Verification Needed:** Update tests to be order-insensitive while maintaining validation

---

## [x] Import formatting issue in registry
**File:** `internal/registry/registry.go`
**Lines:** 12-13
**Reported Issue:** Imports are incorrectly split with blank line before "slices" even though all are standard library
**Suggested Fix:** Remove blank line so all stdlib imports are in single group and run goimports -w . to format properly
**Verification Needed:** Fix import grouping and apply proper formatting
**Resolution:** `goimports` regrouped the standard library imports and removed the stray separator in `internal/registry/registry.go`.

---

## [ ] Missing test harness method RegisterPipelineExpectError
**File:** `specs/010-pipeline-dependencies/quickstart.md`
**Lines:** 1318 and 1349-1376
**Reported Issue:** Test calls h.RegisterPipelineExpectError but that method is not defined in the test harness
**Suggested Fix:** Add the missing helper method to testHarness that builds a pipelineConfig from provided ID and options and returns h.registry.Add(cfg.pipeline), or modify test to call existing h.RegisterPipeline and assert expected error
**Verification Needed:** Implement missing test harness helper or update test to use existing methods

---

## [ ] Missing functional requirement for snapshot semantics
**File:** `specs/010-pipeline-dependencies/spec.md`
**Lines:** 86 and 120
**Reported Issue:** Snapshot semantics are only mentioned as an assumption/edge case without formal requirement
**Suggested Fix:** Add explicit functional requirement FR-017 stating system MUST capture snapshot of pipeline definitions, dependency relationships, and registry state at start of streamy run --registry execution
**Verification Needed:** Add formal requirement for snapshot semantics and update FR-005 to reference snapshot-based dependency resolution

---

## [ ] CLI command reference needs consolidation
**File:** `specs/010-pipeline-dependencies/spec.md`
**Lines:** 22-23, 74, 110, 139
**Reported Issue:** CLI command specs are scattered and need disambiguation with unified syntax
**Suggested Fix:** Add "CLI Command Reference" under Requirements that defines canonical commands and aliases, specifies --tree output format with status legend, clarifies visual representation, and documents --dry-run placement and semantics
**Verification Needed:** Create comprehensive CLI reference section with unified syntax and examples

---

## [ ] Nil pointer panic risk in orchestration
**File:** `specs/010-pipeline-dependencies/quickstart.md`
**Lines:** 959-990
**Reported Issue:** Call to uc.registry.GetPipeline ignores error and dereferences pipeline.Path causing possible nil pointer panic
**Suggested Fix:** Update goroutine to capture and check GetPipeline error, set PipelineExecutionSummary to failed/blocked state when pipeline is nil, skip Apply call, and only access pipeline.Path after confirming pipeline != nil
**Verification Needed:** Add proper error handling and nil checks to prevent panics

---

## [ ] Missing --dry-run flag wiring in run command
**File:** `specs/010-pipeline-dependencies/quickstart.md`
**Lines:** 1066, 1075-1097, 1120
**Reported Issue:** Run command is missing --dry-run flag and runOptions struct isn't wired for it
**Suggested Fix:** Add DryRun bool field to runOptions, add flag registration in newRunCmd, and update runOrchestrate invocation to pass opts.DryRun
**Verification Needed:** Implement complete --dry-run flag wiring from CLI flag to orchestration execution

---

## [x] Logger nil-check confusion in orchestrate.go
**File:** `cmd/streamy/orchestrate.go`
**Lines:** 98-100
**Reported Issue:** Nil-check uses both app.Logger and logger but body only references logger, which is confusing
**Suggested Fix:** Remove redundant app.Logger check and rely on incoming logger, or consistently derive logger from app.Logger and check only app.Logger
**Verification Needed:** Simplify nil-check logic to use consistent logger source
**Resolution:** Reduced the guard to a single `logger != nil` check before logging completion in `cmd/streamy/orchestrate.go`.

---

## [ ] Hard-coded stdin reading prevents unit testing
**File:** `cmd/streamy/orchestrate.go`
**Lines:** 301-318 and 129
**Reported Issue:** promptForceConfirmation reads directly from os.Stdin which hinders unit testing
**Suggested Fix:** Change function signature to accept input io.Reader, default to os.Stdin when provided in is nil, and update caller to pass desired reader
**Verification Needed:** Refactor to accept io.Reader parameter for testability

---

## [ ] Force flag scope needs clarification in specification
**File:** `specs/010-pipeline-dependencies/spec.md`
**Lines:** 57 and 109-113
**Reported Issue:** Scope of "--force" flag is ambiguous - unclear if it overrides missing dependencies or only upstream failures
**Suggested Fix:** Update FR-013 to state that "--force" only permits execution when upstream pipelines have failed, and does NOT override pipeline blocking caused by missing or unregistered dependencies
**Verification Needed:** Clarify distinction between "upstream failures" and "missing/unregistered dependencies" in user story and FR-013

---

## [ ] Missing cycle detection algorithm details in design resolution
**File:** `specs/010-pipeline-dependencies/DESIGN_RESOLUTION.md`
**Lines:** 103, 414, 423
**Reported Issue:** Design references "Validates cycle detection" but lacks concrete algorithm, example error, and graph constraints
**Suggested Fix:** State a concrete cycle detection strategy (e.g., DFS-based topological sort with visited/recursion-stack or Kahn's algorithm), add example showing cyclic dependency with exact error string, and define dependency graph constraints (must be DAG, max node/edge expectations, handling of disconnected components)
**Verification Needed:** Add algorithm description, example error output, and graph constraints adjacent to existing validation lines

---

## [x] Ignored error in context creation
**File:** `cmd/streamy/run.go`
**Lines:** 64
**Reported Issue:** Call to app.CommandContext ignores returned error which can cause silent failures
**Suggested Fix:** Capture the error (ctx, err := app.CommandContext(...)), check if err != nil, and handle it by returning the error from the function or logging and returning a wrapped error so the command fails fast when context creation fails
**Verification Needed:** Confirm the current error handling and implement proper error capture and return
**Resolution:** Determined to be a false positive; `AppContext.CommandContext` returns only `(context.Context, ports.Logger)` without an error result, so there is nothing to capture in `cmd/streamy/run.go`.

---

## [x] FindUnregisteredDependencies returns unsorted duplicates
**File:** `internal/registry/registry.go`
**Lines:** 178-202
**Reported Issue:** Function can return duplicate dependency IDs and an unsorted slice
**Suggested Fix:** Change it to deduplicate and sort the missing dependencies before returning them by using a temporary map to collect unique missing IDs, then build a slice from that map, call sort.Strings on the slice, and return it
**Verification Needed:** Verify current function behavior and implement deduplication and sorting
**Resolution:** Updated `FindUnregisteredDependencies` to collect unique IDs in a set and return a sorted slice using `slices.Sort`.

---

## [x] Missing statusCache validation in orchestration use case constructor
**File:** `internal/application/orchestration/usecase.go`
**Lines:** 29-49
**Reported Issue:** Constructor validates registry and executor but not statusCache, making API contract unclear
**Suggested Fix:** Either enforce non-nil by adding a nil check that returns an error like "status cache is nil" to keep behavior consistent with other parameters, or if statusCache is intended to be optional, add a clear comment above the function explaining that nil is allowed and that persistSummary handles nil caches safely
**Verification Needed:** Determine if statusCache should be required or optional and implement appropriate validation or documentation
**Resolution:** Documented the intended optionality by extending the constructor comment to state that a nil cache disables persistence without error.

---

## [x] Inconsistent nil handling in status cache Load method
**File:** `internal/registry/status_cache.go`
**Lines:** 50-79
**Reported Issue:** Load method treats nil Statuses map by initializing an empty map but does not do the same for Orchestrations, leaving c.orchestrations nil when file.Orchestrations is nil
**Suggested Fix:** Explicitly set c.orchestrations to an empty slice when file.Orchestrations is nil, otherwise set it to a copy of file.Orchestrations (mirror the nil-check/initialization style used for Statuses)
**Verification Needed:** Confirm the inconsistent nil handling and implement proper initialization for both fields
**Resolution:** `StatusCache.Load` now initializes `c.orchestrations` to an empty slice when the file omits orchestration history, matching the status map behavior.

---

## [ ] BlockedBy metadata schema is ambiguous
**File:** `specs/010-pipeline-dependencies/spec.md`
**Lines:** 109
**Reported Issue:** BlockedBy metadata is ambiguous for multiple blocking reasons, lacks defined schema
**Suggested Fix:** Define explicit schema for BlockedBy as array of objects with fields type (missing|failed), id (dependency or pipeline id), reason (short string), details (optional), and add example outputs for common scenarios
**Verification Needed:** Implement deterministic schema for CLI/error formatting and ensure FR-009 and FR-013 consistency

---
