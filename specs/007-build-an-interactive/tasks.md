# Tasks: Interactive Dashboard for Pipeline Management

**Input**: Design documents from `/specs/007-build-an-interactive/`  
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Feature Branch**: `007-build-an-interactive`  
**Organization**: Tasks grouped by user story for independent implementation and testing

## Format: `[ID] [P?] [Story] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1-US5, Setup, Foundation, Polish)
- Exact file paths included in descriptions

## Path Conventions (Single Go Project)
- Source: `internal/`, `cmd/` at repository root
- Tests: `tests/` at repository root
- Testdata: `testdata/` at repository root

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and test data fixtures for all user stories

- [X] **T001** [P] [Setup] Create test registry fixtures in `testdata/registry/empty.json`
- [X] **T002** [P] [Setup] Create test registry fixtures in `testdata/registry/single-pipeline.json`
- [X] **T003** [P] [Setup] Create test registry fixtures in `testdata/registry/multiple-pipelines.json`
- [X] **T004** [P] [Setup] Create test cache fixtures in `testdata/cache/empty-cache.json`
- [X] **T005** [P] [Setup] Create test cache fixtures in `testdata/cache/populated-cache.json`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story implementation

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Registry Abstraction Layer

- [X] **T006** [P] [Foundation] Create `internal/registry/types.go` with Pipeline, RegistryFile, CachedStatus structs
- [X] **T007** [Foundation] Implement Registry struct in `internal/registry/registry.go` with Load/Save/List/Get/Add/Remove methods
- [X] **T008** [Foundation] Implement StatusCache struct in `internal/registry/cache.go` with Get/Set/Invalidate methods
- [X] **T009** [P] [Foundation] Add unit tests for Registry in `internal/registry/registry_test.go`
- [X] **T010** [P] [Foundation] Add unit tests for StatusCache in `internal/registry/cache_test.go`

### Engine Integration Extraction

- [X] **T011** [Foundation] Extract verify core logic from `cmd/streamy/verify.go` to `internal/engine/verify.go` (VerifyPipeline function)
- [X] **T012** [Foundation] Extract apply core logic from `cmd/streamy/apply.go` to `internal/engine/apply.go` (ApplyPipeline function)
- [X] **T013** [Foundation] Update `cmd/streamy/verify.go` to call extracted `internal/engine/verify.go`
- [X] **T014** [Foundation] Update `cmd/streamy/apply.go` to call extracted `internal/engine/apply.go`

### Dashboard TUI Foundation

- [X] **T015** [P] [Foundation] Create `internal/tui/dashboard/messages.go` with all message type definitions (PipelineSelectedMsg, BackToListMsg, Verify/Apply messages, etc.)
- [X] **T016** [P] [Foundation] Create `internal/tui/dashboard/styles.go` with Lipgloss style definitions (title, item, selected, error, footer styles)
- [X] **T017** [Foundation] Create `internal/tui/dashboard/model.go` with DashboardModel struct and NewModel constructor
- [X] **T018** [Foundation] Implement Init() method in `internal/tui/dashboard/model.go`

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - View All Pipeline Statuses at a Glance (Priority: P1) 🎯 MVP

**Goal**: Display all registered pipelines with status indicators, descriptions, and last run times in a list view

**Independent Test**: Register 3-5 pipelines with different statuses, run `streamy` with no arguments, verify all appear with correct indicators and sorted by priority (failed > drifted > satisfied > unknown)

### Implementation for User Story 1

- [X] **T019** [P] [US1] Implement helper functions in `internal/tui/dashboard/model.go`: GetPipelineStatus, SortPipelines, CountByStatus
- [X] **T020** [P] [US1] Implement loadInitialStatusCmd in `internal/tui/dashboard/commands.go` to load cached statuses on startup
- [X] **T021** [US1] Implement renderListView in `internal/tui/dashboard/view.go` with header, body, footer layout
- [X] **T022** [US1] Implement renderHeader in `internal/tui/dashboard/view.go` with title and status summary
- [X] **T023** [US1] Implement renderPipelineList in `internal/tui/dashboard/view.go` with pipeline items rendering
- [X] **T024** [US1] Implement renderPipelineItem in `internal/tui/dashboard/view.go` with status icon, name, description, and last run time
- [X] **T025** [US1] Implement renderFooter in `internal/tui/dashboard/view.go` with keyboard hints
- [X] **T026** [US1] Implement empty state handling in renderPipelineList (FR-014: show friendly message when no pipelines)
- [X] **T027** [US1] Add FormatLastRun helper in `internal/tui/dashboard/view.go` for human-readable timestamps ("2 hours ago", "Never run")
- [X] **T028** [US1] Implement View() method in `internal/tui/dashboard/model.go` to dispatch to renderListView
- [X] **T029** [US1] Handle InitialStatusLoadedMsg in Update() method to populate pipeline statuses from cache
- [X] **T030** [US1] Handle tea.WindowSizeMsg in Update() method to adjust layout dimensions (FR-013)
- [X] **T031** [US1] Create `cmd/streamy/dashboard.go` with newDashboardCmd and runDashboard functions
- [X] **T032** [US1] Modify `cmd/streamy/root.go` to launch dashboard when no subcommands provided
- [X] **T033** [P] [US1] Add unit test for SortPipelines in `internal/tui/dashboard/model_test.go`
- [X] **T034** [P] [US1] Add unit test for FormatLastRun in `internal/tui/dashboard/view_test.go`
- [X] **T035** [US1] Add integration test in `tests/integration_dashboard_test.go`: TestDashboardDisplaysPipelines (verify 3 pipelines appear)
- [X] **T036** [US1] Add integration test in `tests/integration_dashboard_test.go`: TestDashboardEmptyState (verify empty state message)
- [X] **T037** [US1] Add integration test in `tests/integration_dashboard_test.go`: TestDashboardSortsByPriority (verify failed/drifted appear first)

**Checkpoint**: User Story 1 complete - Dashboard displays all pipelines with status at a glance ✅

---

## Phase 4: User Story 2 - Navigate and Select Pipelines Interactively (Priority: P2)

**Goal**: Enable keyboard navigation (up/down arrows, number keys) and pipeline selection to view details

**Independent Test**: Open dashboard with 2+ pipelines, use arrow keys to navigate, press Enter to select, verify detail view shows pipeline-specific information, press Esc to return

### Implementation for User Story 2

- [X] **T038** [P] [US2] Implement renderDetailView in `internal/tui/dashboard/view.go` with pipeline details layout
- [X] **T039** [US2] Implement handleDetailKeys in `internal/tui/dashboard/update.go` for detail view (Esc, v, a, r, ?)
- [X] **T040** [US2] Handle Enter key in handleListKeys to select pipeline and switch to detail view
- [X] **T041** [US2] Handle number keys (1-9) in handleListKeys to directly select pipeline by index
- [X] **T042** [US2] Add integration test in `tests/integration_dashboard_test.go`: TestDashboardNavigateToDetail
- [X] **T043** [US2] Add integration test in `tests/integration_dashboard_test.go`: TestDashboardBackToList
- [X] **T044** [US2] Add integration test in `tests/integration_dashboard_test.go`: TestDashboardDirectSelection
- [X] **T045** [US2] Add integration test in `tests/integration_dashboard_test.go`: TestDashboardDetailViewWithResult

**Checkpoint**: User Story 2 complete - Navigation and selection working ✅

---

## Phase 5: User Story 3 - Run Verification from Dashboard (Priority: P3)

**Goal**: Trigger verification from detail view, show progress indicators, update status on completion, support cancellation

**Independent Test**: Open pipeline detail view, press 'v', observe spinner and progress, verify status updates when complete. Test cancellation by pressing Esc during verification.

### Implementation for User Story 3

- [X] **T057** [P] [US3] Implement verifyCmd in `internal/tui/dashboard/commands.go` to run verification asynchronously
- [X] **T058** [US3] Handle 'v' key in handleDetailKeys to dispatch verifyCmd
- [X] **T059** [US3] Handle VerifyStartedMsg in Update() to set loading[pipelineID] = true and start spinner
- [X] **T060** [US3] Handle VerifyCompleteMsg in Update() to update pipeline status, clear loading, save to cache
- [X] **T061** [US3] Handle VerifyErrorMsg in Update() to set status to failed, display error banner
- [X] **T062** [US3] Implement saveStatusCacheCmd in `internal/tui/dashboard/commands.go` to persist status updates
- [X] **T063** [US3] Add spinner rendering in renderPipelineItem when loading[id] == true (FR-009)
- [X] **T064** [US3] Add spinner.TickMsg handling in Update() to animate spinner
- [X] **T065** [US3] Implement renderConfirmView in `internal/tui/dashboard/view.go` for cancellation confirmation dialog
- [X] **T066** [US3] Handle Esc during verification to show ConfirmActionMsg (FR-021: "Cancel verification? (y/n)")
- [X] **T067** [US3] Implement handleConfirmKeys in `internal/tui/dashboard/update.go` for y/n/Esc responses
- [X] **T068** [US3] Handle ConfirmResponseMsg in Update() to call cancelOperationCmd if confirmed
- [X] **T069** [P] [US3] Implement cancelOperationCmd in `internal/tui/dashboard/commands.go` using context cancellation
- [X] **T070** [US3] Update verifyCmd to accept context.Context for cancellation support
- [X] **T071** [US3] Update `internal/engine/verify.go` VerifyPipeline to accept and respect context.Context
- [X] **T072** [US3] Add progress indicators in renderDetailView (show "Verifying... step 3/8")
- [X] **T073** [P] [US3] Add unit test for VerifyCompleteMsg handling in `internal/tui/dashboard/update_test.go`
- [X] **T074** [P] [US3] Add unit test for VerifyErrorMsg handling in `internal/tui/dashboard/update_test.go`
- [X] **T075** [P] [US3] Add unit test for ConfirmResponseMsg handling in `internal/tui/dashboard/update_test.go`
- [X] **T076** [US3] Add integration test in `tests/integration_dashboard_test.go`: TestDashboardVerify (trigger verify, check status updates)
- [X] **T077** [US3] Add integration test in `tests/integration_dashboard_test.go`: TestDashboardVerifyError (verify error display on failure)
- [X] **T078** [US3] Add integration test in `tests/integration_dashboard_test.go`: TestDashboardVerifyCancellation (test Esc → confirmation → cancel)

**Checkpoint**: User Story 3 complete - Verification from dashboard working with cancellation ✅

---

## Phase 6: User Story 4 - Apply Pipeline Configuration Interactively (Priority: P4)

**Goal**: Trigger apply operations from detail view with confirmation prompt, show progress, update status on completion

**Independent Test**: Select a drifted pipeline, press 'a', confirm prompt, observe progress, verify status updates to satisfied when complete

### Implementation for User Story 4

- [X] **T079** [P] [US4] Implement applyCmd in `internal/tui/dashboard/commands.go` to run apply asynchronously
- [X] **T080** [US4] Handle 'a' key in handleDetailKeys to show ConfirmActionMsg for apply operation
- [X] **T081** [US4] Update renderConfirmView to show apply-specific confirmation message ("Apply changes? This will modify your system.")
- [X] **T082** [US4] Handle ConfirmResponseMsg for apply action to dispatch applyCmd if confirmed
- [X] **T083** [US4] Handle ApplyStartedMsg in Update() to set loading[pipelineID] = true
- [X] **T084** [US4] Handle ApplyCompleteMsg in Update() to update status to satisfied, clear loading, save cache
- [X] **T085** [US4] Handle ApplyErrorMsg in Update() to display error with failed step information (FR-016)
- [X] **T086** [US4] Update applyCmd to accept context.Context for cancellation support
- [X] **T087** [US4] Update `internal/engine/apply.go` ApplyPipeline to accept and respect context.Context
- [X] **T088** [US4] Add apply progress indicators in renderDetailView (show "Applying... step 5/12")
- [X] **T089** [US4] Add retry option in error display for failed apply operations
- [X] **T090** [US4] Implement auto-verify after successful apply in ApplyCompleteMsg handler (dispatch verifyCmd)
- [X] **T091** [P] [US4] Add unit test for ApplyCompleteMsg handling in `internal/tui/dashboard/update_test.go`
- [X] **T092** [P] [US4] Add unit test for ApplyErrorMsg handling in `internal/tui/dashboard/update_test.go`
- [X] **T093** [US4] Add integration test in `tests/integration_dashboard_test.go`: TestDashboardApply (trigger apply, check status updates)
- [X] **T094** [US4] Add integration test in `tests/integration_dashboard_test.go`: TestDashboardApplyConfirmation (verify confirmation required)
- [X] **T095** [US4] Add integration test in `tests/integration_dashboard_test.go`: TestDashboardApplyError (verify error display on failure)

**Checkpoint**: User Story 4 complete - Apply operations from dashboard working ✅

---

## Phase 7: User Story 5 - Refresh Dashboard Status (Priority: P5)

**Goal**: Allow manual refresh of all pipeline statuses in parallel with progress indication

**Independent Test**: Modify a config file externally, press 'r' in dashboard, observe progress counter, verify all statuses update

### Implementation for User Story 5

- [X] **T096** [US5] Implement refreshAllCmd in `internal/tui/dashboard/commands.go` to batch verify all pipelines
- [X] **T097** [US5] Handle 'r' key in handleListKeys to dispatch refreshAllCmd
- [X] **T098** [US5] Handle RefreshStartedMsg in Update() to set refreshing state
- [X] **T099** [US5] Handle RefreshProgressMsg in Update() to update progress counter
- [X] **T100** [US5] Handle RefreshCompleteMsg in Update() to update all statuses and clear refreshing state
- [X] **T101** [US5] Update refreshAllCmd to send progress updates as individual verifications complete
- [X] **T102** [US5] Add refresh progress indicator in renderHeader (show "Refreshing... 3/10")
- [X] **T103** [US5] Add global spinner in header during refresh operations
- [X] **T104** [US5] Handle partial failures during refresh (some verifications fail, others succeed)
- [X] **T105** [US5] Implement saveAllStatusCmd in `internal/tui/dashboard/commands.go` to batch-save all statuses
- [X] **T106** [P] [US5] Add unit test for RefreshCompleteMsg handling in `internal/tui/dashboard/update_test.go`
- [X] **T107** [P] [US5] Add unit test for refreshAllCmd parallelization in `internal/tui/dashboard/commands_test.go`
- [X] **T108** [US5] Add integration test in `tests/integration_dashboard_test.go`: TestDashboardRefresh (trigger refresh, verify all update)
- [X] **T109** [US5] Add integration test in `tests/integration_dashboard_test.go`: TestDashboardRefreshProgress (verify progress counter)
- [X] **T110** [US5] Add integration test in `tests/integration_dashboard_test.go`: TestDashboardRefreshPartialFailure (verify graceful handling)

**Checkpoint**: User Story 5 complete - Refresh functionality working ✅

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Improvements affecting multiple user stories and final quality assurance

### Help Overlay & Error Handling

- [X] **T111** [P] [Polish] Create renderHelpView in `internal/tui/dashboard/view.go` with keyboard shortcuts reference
- [X] **T112** [P] [Polish] Handle '?' key to toggle help overlay (ViewHelp mode)
- [X] **T113** [P] [Polish] Handle 'q' and Ctrl+C to quit dashboard from any view
- [X] **T114** [Polish] Implement renderErrorBanner in `internal/tui/dashboard/view.go` for error display (FR-016)
- [X] **T115** [Polish] Handle ErrorMsg to show error banner with context and suggestions
- [X] **T116** [Polish] Handle ClearErrorMsg to dismiss error banner (Esc or auto-dismiss after 5s)

### Edge Cases & Validation

- [X] **T117** [P] [Polish] Add missing config file handling in loadDetailCmd (FR-020)
- [X] **T118** [P] [Polish] Add terminal size validation with warning for <80 columns (research.md requirement)
- [X] **T119** [P] [Polish] Add text truncation for long pipeline descriptions (edge case handling)
- [X] **T120** [P] [Polish] Add Unicode fallback for terminals without emoji support (use [OK], [!!], [XX], [??])
- [X] **T121** [Polish] Add graceful degradation for narrow terminals (<80 columns)

### Performance & Optimization

- [X] **T122** [P] [Polish] Cache static strings (header, footer) in renderListView
- [X] **T123** [P] [Polish] Optimize View() rendering to avoid unnecessary Lipgloss calls
- [X] **T124** [P] [Polish] Add debouncing for status cache writes during batch operations

### Documentation & Testing

- [X] **T125** [P] [Polish] Update main README.md with dashboard usage section and screenshots
- [X] **T126** [P] [Polish] Add code comments and godoc for all exported types in `internal/tui/dashboard/`
- [X] **T127** [P] [Polish] Run quickstart.md validation workflow (manual testing checklist)
- [X] **T128** [P] [Polish] Performance test with 100 pipelines (validate <500ms startup target)
- [X] **T129** [P] [Polish] Cross-platform testing on Linux, macOS, Windows terminals

### Integration Tests

- [X] **T130** [Polish] Add end-to-end integration test: TestDashboardFullWorkflow (list → select → verify → apply → back → refresh)
- [X] **T131** [Polish] Add integration test for terminal resize handling: TestDashboardResize
- [X] **T132** [Polish] Add integration test for concurrent operations: TestDashboardConcurrentVerify

**Checkpoint**: All polish tasks complete - Feature ready for release ✅

---

## Dependencies & Execution Order

### Phase Dependencies

1. **Setup (Phase 1)**: No dependencies - can start immediately
2. **Foundational (Phase 2)**: Depends on Setup - **BLOCKS all user stories**
3. **User Story 1 (Phase 3)**: Depends on Foundational - Can start once Phase 2 complete
4. **User Story 2 (Phase 4)**: Depends on User Story 1 (needs list view to navigate)
5. **User Story 3 (Phase 5)**: Depends on User Story 2 (needs detail view to trigger verify)
6. **User Story 4 (Phase 6)**: Depends on User Story 2 (needs detail view to trigger apply)
7. **User Story 5 (Phase 7)**: Depends on User Story 3 (uses verify infrastructure)
8. **Polish (Phase 8)**: Depends on desired user stories being complete

### User Story Dependencies

- **US1** (View statuses): Independent - only needs Foundation
- **US2** (Navigation): Depends on US1 (builds on list view)
- **US3** (Verify): Depends on US2 (needs detail view navigation)
- **US4** (Apply): Depends on US2 (needs detail view navigation)
- **US5** (Refresh): Depends on US3 (uses verify commands in parallel)

### Within Each User Story

1. Tests can run in parallel (all marked [P])
2. Helper functions and types before main implementation
3. Commands before Update handlers
4. View rendering before full integration
5. Unit tests alongside implementation
6. Integration tests after story completion

### Parallel Opportunities

**Setup Phase** (all can run in parallel):
- T001-T005 (all test fixtures)

**Foundational Phase** (groups can run in parallel):
- Group 1: T006-T010 (registry layer)
- Group 2: T011-T014 (engine extraction) - can run parallel with Group 1
- Group 3: T015-T018 (TUI foundation) - must wait for T006 (types dependency)

**User Story Phases** (within each story):
- Commands, helpers, and styles can be parallel
- View rendering functions can be parallel
- Unit tests can be parallel
- Integration tests can be sequential (share state)

**Polish Phase**:
- T111-T113, T117-T121, T125-T129 all parallelizable
- T130-T132 sequential (integration tests)

---

## Parallel Example: User Story 1

```bash
# After Foundation complete, launch these in parallel:
T019: Helper functions (GetPipelineStatus, SortPipelines, CountByStatus)
T020: loadInitialStatusCmd implementation

# Then these in parallel:
T021: renderListView skeleton
T022: renderHeader
T025: renderFooter
T027: FormatLastRun helper

# Then these in parallel:
T033: Unit test SortPipelines
T034: Unit test FormatLastRun

# Finally sequential (integration tests):
T035 → T036 → T037
```

---

## Parallel Example: Foundational Phase

```bash
# Registry layer (Group 1):
T006: types.go
↓
T007: registry.go (depends on T006)
T008: cache.go (depends on T006)
↓
T009: registry_test.go (depends on T007)
T010: cache_test.go (depends on T008)

# Engine extraction (Group 2 - parallel with Group 1):
T011: Extract verify
T012: Extract apply
↓
T013: Update verify CLI
T014: Update apply CLI

# TUI foundation (Group 3 - after T006):
T015: messages.go (can start after T006)
T016: styles.go (can start immediately)
↓
T017: model.go (depends on T006, T015)
T018: Init() (depends on T017)
```

---

## Implementation Strategy

### MVP First (User Stories 1-2 Only)

**Minimum Viable Product delivers:**
- View all pipelines with status indicators (US1)
- Navigate and select pipelines (US2)
- Basic detail view display

**Steps:**
1. Complete Phase 1: Setup (T001-T005) - ~1 hour
2. Complete Phase 2: Foundational (T006-T018) - ~4-6 hours
3. Complete Phase 3: User Story 1 (T019-T037) - ~6-8 hours
4. Complete Phase 4: User Story 2 (T038-T056) - ~4-6 hours
5. **STOP and VALIDATE**: Test US1+US2 independently
6. Deploy MVP if validated

**Total MVP Effort**: ~15-21 hours (2-3 days solo, 1 day with team)

### Incremental Delivery

**Delivery 1: MVP (US1 + US2)**
- Users can view and navigate pipelines
- Value: Replaces manual `streamy verify` checking

**Delivery 2: MVP + Verification (US3)**
- Users can trigger verification from dashboard
- Value: Actionable dashboard, reduces context switching

**Delivery 3: MVP + Verification + Apply (US4)**
- Users can apply configurations interactively
- Value: Complete workflow without leaving dashboard

**Delivery 4: Full Feature (US5 + Polish)**
- Users can refresh all statuses, help overlay, error handling
- Value: Production-ready central workspace

### Parallel Team Strategy

With 3 developers after Foundational phase completes:

- **Developer A**: User Story 1 (T019-T037) - ~2 days
- **Developer B**: Start User Story 3 commands (T057, T062, T069) in parallel - ~1 day
- **Developer C**: Start User Story 4 commands (T079, T086) in parallel - ~1 day

Once US1 complete:
- **Developer A**: User Story 2 (T038-T056) - ~2 days
- **Developer B**: Continue US3 integration
- **Developer C**: Continue US4 integration

---

## Task Statistics

**Total Tasks**: 132 tasks

**By Phase**:
- Setup: 5 tasks
- Foundational: 13 tasks (~10% of total) - **BLOCKS all stories**
- User Story 1 (P1): 19 tasks (~14%)
- User Story 2 (P2): 19 tasks (~14%)
- User Story 3 (P3): 22 tasks (~17%)
- User Story 4 (P4): 17 tasks (~13%)
- User Story 5 (P5): 15 tasks (~11%)
- Polish: 22 tasks (~17%)

**Parallelizable Tasks**: 47 tasks marked [P] (~36%)

**Test Tasks**: 24 unit tests + 12 integration tests = 36 test tasks (~27%)

**MVP Scope (US1+US2)**: 51 tasks (~39% of total)

**Independent Test Criteria**:
- US1: Register 3-5 pipelines, verify display with status icons and sorting ✅
- US2: Navigate with arrows, select with Enter, verify detail view ✅
- US3: Press 'v', verify status updates, test cancellation ✅
- US4: Press 'a', confirm prompt, verify apply success ✅
- US5: Press 'r', verify all statuses refresh in parallel ✅

---

## Notes

- **[P] tasks**: Different files, no dependencies - safe for parallel execution
- **[Story] label**: Maps task to user story for traceability and independent testing
- **Foundational phase is critical**: Must complete before ANY user story work begins
- Each user story is independently testable and deliverable
- Stop at any checkpoint to validate story works standalone
- Commit after each task or logical group
- MVP (US1+US2) delivers immediate value - consider deploying before continuing
- Tests are included but can be skipped if time-constrained (add later)
- Follow quickstart.md for development workflow and debugging tips

---

## Success Criteria

✅ **Phase 2 Complete**: Registry, engine extraction, TUI foundation all working
✅ **US1 Complete**: Dashboard displays pipelines with status at a glance
✅ **US2 Complete**: Navigation and selection working, detail view accessible
✅ **US3 Complete**: Verification runs from dashboard with cancellation
✅ **US4 Complete**: Apply operations work with confirmation
✅ **US5 Complete**: Refresh updates all statuses in parallel
✅ **Polish Complete**: Help, errors, edge cases, performance optimized
✅ **All Integration Tests Pass**: End-to-end workflows validated
✅ **Performance Targets Met**: <500ms startup, <16ms navigation, <3s refresh
