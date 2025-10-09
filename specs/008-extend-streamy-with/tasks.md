# Tasks: Registry Management CLI Commands

**Input**: Design documents from `/specs/008-extend-streamy-with/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Unit and integration tests are included as specified in the feature requirements.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

**Total Tasks**: 76 (75 original + 1 concurrent access test)

## Format: `[ID] [P?] [Story] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4, US5)
- Include exact file paths in descriptions

## Path Conventions
- Repository root: `/home/alexis/Projects/Streamy`
- Commands: `cmd/streamy/`
- Internal packages: `internal/registry/`
- Tests: `tests/` and `cmd/streamy/*_test.go`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create helper utilities that all commands will use

- [ ] T001 [P] Create `internal/registry/helpers.go` with `GeneratePipelineID(path string) string` function that sanitizes filename to valid ID
- [ ] T002 [P] Add `ValidatePipelineID(id string) error` to `internal/registry/helpers.go` that enforces regex `^[a-z0-9][a-z0-9-]*[a-z0-9]$`
- [ ] T003 [P] Add `SanitizeFilename(name string) string` helper to `internal/registry/helpers.go` for path-to-ID conversion
- [ ] T004 [P] Create unit tests in `internal/registry/helpers_test.go` for ID generation with edge cases (special chars, long names, empty strings)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Parent command structure that all subcommands attach to

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T005 Create `cmd/streamy/registry.go` with parent command `streamy registry` that groups subcommands
- [ ] T006 Add help text and usage examples to registry parent command
- [ ] T007 Register registry command group in `cmd/streamy/root.go` by adding `cmd.AddCommand(newRegistryCmd())`

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Register New Pipeline (Priority: P1) 🎯 MVP

**Goal**: Enable users to add configuration files to the registry with automatic ID generation and validation

**Independent Test**: Run `streamy register <config-path>` and verify pipeline appears in `~/.streamy/registry.json` and in list output

### Tests for User Story 1

**NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T008 [P] [US1] Create `cmd/streamy/register_test.go` with test for successful registration with valid config
- [ ] T009 [P] [US1] Add test to `cmd/streamy/register_test.go` for duplicate ID error (FR-009)
- [ ] T010 [P] [US1] Add test to `cmd/streamy/register_test.go` for invalid config file rejection (FR-002)
- [ ] T011 [P] [US1] Add test to `cmd/streamy/register_test.go` for ID generation from filename

### Implementation for User Story 1

- [ ] T012 [US1] Create `cmd/streamy/register.go` with command struct, flags (--id, --name, --description), and argument validation (exactly 1 path)
- [ ] T013 [US1] Implement `validateAndNormalizePath()` in `register.go` that checks file existence and converts to absolute path (FR-010)
- [ ] T014 [US1] Implement config validation in `register.go` using `config.ParseAndValidate()` from `internal/config` (FR-002)
- [ ] T015 [US1] Implement registration flow in `register.go`: generate/validate ID, create Pipeline struct, call `Registry.Add()`, call `Registry.Save()` (FR-001, FR-003, FR-004)
- [ ] T016 [US1] Add error handling with structured error messages and suggestions (FR-014)
- [ ] T017 [US1] Add verbose logging controlled by `--verbose` flag showing validation and save steps
- [ ] T018 [US1] Register register command as subcommand in `cmd/streamy/registry.go`

**Checkpoint**: At this point, User Story 1 should be fully functional - users can register pipelines

---

## Phase 4: User Story 2 - List All Registered Pipelines (Priority: P1) 🎯 MVP

**Goal**: Enable users to view all registered pipelines with status indicators in human-readable table or JSON format

**Independent Test**: Register multiple pipelines, run `streamy list`, verify all appear with correct formatting

### Tests for User Story 2

- [ ] T019 [P] [US2] Create `cmd/streamy/list_test.go` with test for table format output with multiple pipelines
- [ ] T020 [P] [US2] Add test to `cmd/streamy/list_test.go` for JSON format validation with schema check
- [ ] T021 [P] [US2] Add test to `cmd/streamy/list_test.go` for empty registry friendly message
- [ ] T022 [P] [US2] Add test to `cmd/streamy/list_test.go` for status icon display (Unicode and ASCII fallback)

### Implementation for User Story 2

- [ ] T023 [US2] Create `cmd/streamy/list.go` with command struct, flags (--format, --json), and no-arguments validation (FR-005)
- [ ] T024 [US2] Implement table output in `list.go` using `text/tabwriter` with columns: ID, Name, Status, Last Run, Path
- [ ] T025 [US2] Add status icon formatting in `list.go` with Unicode icons (🟢🟡🔴⚪) and ASCII fallback ([OK][!!][XX][??])
- [ ] T026 [US2] Implement relative timestamp formatting in `list.go` ("2 hours ago", "1 day ago", "never")
- [ ] T027 [US2] Implement JSON output in `list.go` marshaling registry + status cache with version and count fields
- [ ] T028 [US2] Add empty registry handler in `list.go` with friendly message and hint to register
- [ ] T029 [US2] Add TTY detection in `list.go` to disable colors/icons when output is piped
- [ ] T030 [US2] Register list command as subcommand in `cmd/streamy/registry.go`

**Checkpoint**: At this point, User Stories 1 AND 2 work together - users can register and view pipelines

---

## Phase 5: User Story 3 - Remove Obsolete Pipeline (Priority: P2)

**Goal**: Enable users to remove pipelines from registry with confirmation prompt for safety

**Independent Test**: Register a pipeline, unregister it, verify it no longer appears in list

### Tests for User Story 3

- [ ] T031 [P] [US3] Create `cmd/streamy/unregister_test.go` with test for successful removal and registry save
- [ ] T032 [P] [US3] Add test to `cmd/streamy/unregister_test.go` for pipeline-not-found error (FR-014)
- [ ] T033 [P] [US3] Add test to `cmd/streamy/unregister_test.go` for confirmation prompt logic with --force bypass

### Implementation for User Story 3

- [ ] T034 [US3] Create `cmd/streamy/unregister.go` with command struct, flags (--force), and argument validation (exactly 1 ID) (FR-006)
- [ ] T035 [US3] Implement confirmation prompt in `unregister.go` using `bufio.Scanner` with "Remove pipeline 'X'? [y/N]" message
- [ ] T036 [US3] Add TTY detection in `unregister.go` to require --force flag in non-interactive environments
- [ ] T037 [US3] Implement removal flow in `unregister.go`: call `Registry.Get()` to verify exists, prompt confirmation, call `Registry.Remove()`, call `Registry.Save()`
- [ ] T038 [US3] Add status cache cleanup in `unregister.go` by optionally calling `StatusCache.Delete(id)` and `StatusCache.Save()`
- [ ] T039 [US3] Add success message in `unregister.go` indicating pipeline removed and config file not deleted
- [ ] T040 [US3] Register unregister command as subcommand in `cmd/streamy/registry.go`

**Checkpoint**: At this point, User Stories 1, 2, AND 3 work independently - full register/list/unregister lifecycle

---

## Phase 6: User Story 4 - Refresh Pipeline Statuses (Priority: P2)

**Goal**: Enable users to batch-verify pipelines and update status cache concurrently

**Independent Test**: Register pipelines, run `streamy refresh`, verify status cache updated and list shows new statuses

### Tests for User Story 4

- [ ] T041 [P] [US4] Create `cmd/streamy/refresh_test.go` with test for single pipeline refresh with status update
- [ ] T042 [P] [US4] Add test to `cmd/streamy/refresh_test.go` for bulk refresh with concurrency using worker pool pattern
- [ ] T043 [P] [US4] Add test to `cmd/streamy/refresh_test.go` for missing config file graceful handling (FR-011)
- [ ] T044 [P] [US4] Add test to `cmd/streamy/refresh_test.go` for progress indicator output during bulk refresh (FR-019)

### Implementation for User Story 4

- [ ] T045 [US4] Create `cmd/streamy/refresh.go` with command struct, flags (--concurrency), and optional pipeline-id argument (FR-007, FR-008)
- [ ] T046 [US4] Implement single pipeline refresh in `refresh.go`: load config, execute verify via engine, update status cache, print result
- [ ] T047 [US4] Implement worker pool in `refresh.go` with semaphore channel for concurrent refresh (default concurrency: 5)
- [ ] T048 [US4] Add progress indicators in `refresh.go` showing "[N/M] pipeline-id... ✓ status" for each verification (FR-019)
- [ ] T049 [US4] Implement missing config file handling in `refresh.go` that marks status as failed without halting entire refresh (FR-011)
- [ ] T050 [US4] Add summary report in `refresh.go` showing counts: satisfied, drifted, failed
- [ ] T051 [US4] Add mutex-protected status cache updates in `refresh.go` for thread-safe concurrent writes
- [ ] T052 [US4] Add dry-run support in `refresh.go` showing which pipelines would be refreshed without executing
- [ ] T053 [US4] Register refresh command as subcommand in `cmd/streamy/registry.go`

**Checkpoint**: At this point, all P1 and P2 user stories complete - MVP+ with batch operations

---

## Phase 7: User Story 5 - Show Pipeline Details (Priority: P3)

**Goal**: Enable users to view detailed pipeline information for debugging and inspection

**Independent Test**: Register and verify a pipeline, run `streamy show <id>`, verify all details displayed

### Tests for User Story 5 (OPTIONAL - P3 priority)

- [ ] T054 [P] [US5] Create `cmd/streamy/show_test.go` with test for detailed output including path, description, timestamps
- [ ] T055 [P] [US5] Add test to `cmd/streamy/show_test.go` for never-verified pipeline showing "never" last run
- [ ] T056 [P] [US5] Add test to `cmd/streamy/show_test.go` for pipeline-not-found error
- [ ] T057 [P] [US5] Add test to `cmd/streamy/show_test.go` for JSON output format with complete schema

### Implementation for User Story 5 (OPTIONAL - P3 priority)

- [ ] T058 [US5] Create `cmd/streamy/show.go` with command struct, flags (--json), and argument validation (exactly 1 ID) (FR-018)
- [ ] T059 [US5] Implement detailed display in `show.go`: pipeline metadata, status, last run, execution result, step count
- [ ] T060 [US5] Add execution history display in `show.go` showing recent step results with durations
- [ ] T061 [US5] Add JSON output format in `show.go` with complete pipeline + status + execution result
- [ ] T062 [US5] Register show command as subcommand in `cmd/streamy/registry.go`

**Checkpoint**: All user stories complete including nice-to-have P3 feature

---

## Phase 8: Integration & Polish

**Purpose**: End-to-end testing and cross-cutting improvements

- [ ] T063 Create `tests/integration_registry_test.go` with full workflow test: register → list → refresh → unregister
- [ ] T064 Add test to `tests/integration_registry_test.go` verifying registry file JSON content after operations
- [ ] T065 Add test to `tests/integration_registry_test.go` verifying status cache updates after refresh
- [ ] T066 Add test to `tests/integration_registry_test.go` with real config file from `testdata/configs/`
- [ ] T066a Add test to `tests/integration_registry_test.go` for concurrent registry operations using goroutines to simulate simultaneous register/unregister calls, verify no data loss and valid JSON after concurrent access (FR-017)
- [ ] T067 [P] Update `README.md` with registry commands section and usage examples
- [ ] T068 [P] Add command examples to `docs/` showing common workflows
- [ ] T069 [P] Update command help text with detailed descriptions and examples
- [ ] T070 Perform manual testing on Linux with 10+ registered pipelines for performance validation (SC-002, SC-003)
- [ ] T071 [P] Perform manual testing on macOS for cross-platform compatibility
- [ ] T072 [P] Perform manual testing on Windows for cross-platform compatibility
- [ ] T073 Verify dashboard auto-updates after register/unregister operations (FR-012, SC-007)
- [ ] T074 Run quickstart.md validation checklist to ensure all acceptance criteria met
- [ ] T075 Code review and refactoring pass for consistency with existing commands

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
  - All tasks marked [P] can run in parallel
  
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
  - T005-T007 must complete sequentially (same files)
  
- **User Stories (Phase 3-7)**: All depend on Foundational phase completion
  - **US1 (Phase 3)**: Can start after Phase 2 - No dependencies on other stories
  - **US2 (Phase 4)**: Can start after Phase 2 - Works with US1 but independently testable
  - **US3 (Phase 5)**: Can start after Phase 2 - Integrates with US1/US2 but independently testable
  - **US4 (Phase 6)**: Can start after Phase 2 - Uses verify engine, independently testable
  - **US5 (Phase 7)**: Can start after Phase 2 - Optional P3 feature, independently testable
  
- **Integration & Polish (Phase 8)**: Depends on desired user stories being complete
  - Should wait for at least US1 and US2 (MVP) before integration testing
  - Can complete with just P1/P2 stories; P3 is optional

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- All tests for a story marked [P] can run in parallel (different test files)
- Implementation tasks in same file must run sequentially
- Implementation tasks in different files can run in parallel

### Parallel Opportunities

**Phase 1 (Setup)**: All tasks can run in parallel
```
T001 [P] helpers.go - GeneratePipelineID
T002 [P] helpers.go - ValidatePipelineID  
T003 [P] helpers.go - SanitizeFilename
T004 [P] helpers_test.go - All unit tests
```

**User Story 1 Tests**: All can run in parallel
```
T008 [P] register_test.go - Test valid registration
T009 [P] register_test.go - Test duplicate ID error
T010 [P] register_test.go - Test invalid config
T011 [P] register_test.go - Test ID generation
```

**User Story 2 Tests**: All can run in parallel
```
T019 [P] list_test.go - Test table format
T020 [P] list_test.go - Test JSON format
T021 [P] list_test.go - Test empty registry
T022 [P] list_test.go - Test status icons
```

**Multiple User Stories in Parallel** (if team capacity allows):
- After Phase 2 completes, assign different developers to US1, US2, US3, US4 simultaneously
- Each story is in a separate file (register.go, list.go, unregister.go, refresh.go)
- No file conflicts, can merge independently

**Documentation Tasks**: All can run in parallel
```
T067 [P] README.md
T068 [P] docs/
T069 [P] Command help text
```

---

## Parallel Example: Starting Multiple User Stories

```bash
# After Phase 2 (Foundational) completes, launch user stories in parallel:

# Developer A: User Story 1 (Register)
Task: "Create cmd/streamy/register.go with command struct and flags"
Task: "Implement validation logic and registration flow"

# Developer B: User Story 2 (List)  
Task: "Create cmd/streamy/list.go with command struct and flags"
Task: "Implement table and JSON output formatting"

# Developer C: User Story 3 (Unregister)
Task: "Create cmd/streamy/unregister.go with command struct and flags"
Task: "Implement confirmation prompt and removal flow"

# Developer D: User Story 4 (Refresh)
Task: "Create cmd/streamy/refresh.go with command struct and flags"
Task: "Implement worker pool and concurrent refresh"
```

---

## Implementation Strategy

### MVP First (User Stories 1 & 2 Only)

1. Complete Phase 1: Setup (T001-T004)
2. Complete Phase 2: Foundational (T005-T007) - BLOCKS all stories
3. Complete Phase 3: User Story 1 - Register (T008-T018)
4. Complete Phase 4: User Story 2 - List (T019-T030)
5. **STOP and VALIDATE**: Test register → list workflow independently
6. Complete Phase 8: Integration tests (T063-T066) and docs (T067-T069)
7. Deploy/demo MVP

### Full Feature (All P1 + P2 Stories)

1. Complete MVP (US1 + US2)
2. Complete Phase 5: User Story 3 - Unregister (T031-T040)
3. Complete Phase 6: User Story 4 - Refresh (T041-T053)
4. **VALIDATE**: Test full lifecycle: register → list → refresh → unregister
5. Complete remaining integration and polish tasks
6. Deploy/demo full feature

### With Optional P3 Feature

- Add Phase 7: User Story 5 - Show (T054-T062) after P1/P2 complete
- Fully optional - can skip for initial release

### Parallel Team Strategy

With 4 developers:

1. Everyone: Complete Setup + Foundational together (T001-T007)
2. Once Foundational done, split by user story:
   - Dev A: US1 Register (T008-T018) - 1-2 days
   - Dev B: US2 List (T019-T030) - 1 day
   - Dev C: US3 Unregister (T031-T040) - 1 day
   - Dev D: US4 Refresh (T041-T053) - 1-2 days
3. Everyone: Integration testing together (T063-T066)
4. Docs/Polish: Divide T067-T075 among team

**Timeline**: ~4 days for full P1+P2 feature with parallel work

---

## Success Metrics (from spec.md)

- **SC-001**: Register command completes in <10 seconds (validate with T070)
- **SC-002**: List command displays 50 pipelines in <1 second (validate with T070)
- **SC-003**: Refresh 10 pipelines completes in <30 seconds (validate with T070)
- **SC-004**: 100% atomic registry writes (validated by integration tests T063-T066)
- **SC-005**: Status visible at a glance (validated by list output in T024-T026)
- **SC-006**: Zero manual file edits (validated by full workflow test T063)
- **SC-007**: Dashboard updates within 2 seconds (validated with T073)

---

## Notes

- [P] tasks = different files, no dependencies, can run in parallel
- [Story] label maps task to specific user story for traceability and independent testing
- Each user story is independently completable and testable (per spec.md requirements)
- Tests MUST fail before implementation (TDD approach)
- Stop at any checkpoint to validate story works independently
- MVP = US1 + US2 (register + list) - delivers immediate value
- Full feature = US1 + US2 + US3 + US4 (all P1 + P2 stories)
- US5 (show command) is optional P3 enhancement
- Commit after each task or logical group
- Verify dashboard integration with manual test (T073) before final release
