# Feature Specification: Interactive Dashboard for Pipeline Management

**Feature Branch**: `007-build-an-interactive`  
**Created**: October 8, 2025  
**Status**: Draft  
**Input**: User description: "Build an interactive dashboard that serves as Streamy's main entry point when users run the `streamy` command with no subcommands. The dashboard displays all registered pipelines, showing their status (🟢 satisfied, 🟡 drifted, 🔴 failed, ⚪ unknown), last run time, and description. Users can select a pipeline to open its details, run verification or apply actions interactively, and refresh statuses. The goal is to make Streamy feel like a central workspace for managing environment setups rather than a one-off command runner. This improves clarity, visual feedback, and daily usability by allowing users to quickly see which pipelines need attention, reapply configurations, or troubleshoot drifts—all from a single TUI interface."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - View All Pipeline Statuses at a Glance (Priority: P1)

As a developer managing multiple environment configurations, I want to see all my registered pipelines and their current states when I run `streamy`, so I can immediately identify which configurations need attention without running individual commands.

**Why this priority**: This is the core value proposition of the dashboard - providing visibility into system state. Without this, users cannot benefit from any other dashboard features. It replaces the need to manually run verify commands on each pipeline individually.

**Independent Test**: Can be fully tested by registering 3-5 pipelines with different configurations, running `streamy` with no arguments, and verifying that all pipelines appear with correct status indicators and metadata. Delivers immediate value by showing system state at a glance.

**Acceptance Scenarios**:

1. **Given** I have 3 registered pipelines (one satisfied, one drifted, one failed), **When** I run `streamy` with no arguments, **Then** I see a dashboard listing all 3 pipelines with color-coded status indicators (🟢, 🟡, 🔴) and their descriptions
2. **Given** I have pipelines with different last run times, **When** the dashboard loads, **Then** each pipeline shows its last verification time (e.g., "2 hours ago", "Never run")
3. **Given** I have just installed Streamy with no pipelines, **When** I run `streamy`, **Then** I see a friendly empty state message explaining how to register pipelines
4. **Given** the dashboard is displayed, **When** I look at the pipeline list, **Then** pipelines are sorted with failed/drifted items at the top to prioritize attention

---

### User Story 2 - Navigate and Select Pipelines Interactively (Priority: P2)

As a user viewing the dashboard, I want to navigate through the pipeline list using keyboard controls and select a pipeline to see more details, so I can investigate specific configurations without typing commands.

**Why this priority**: Builds on P1 by adding interactivity. Users can now drill down into specific pipelines, but this requires the basic listing from P1 to be functional first. Delivers exploratory capability and detail inspection.

**Independent Test**: Can be tested by registering 2+ pipelines, opening the dashboard, using arrow keys to navigate, and pressing Enter on a selected pipeline to view its details. Success is verified when detail view shows pipeline-specific information.

**Acceptance Scenarios**:

1. **Given** the dashboard shows multiple pipelines, **When** I press up/down arrow keys, **Then** the selection highlight moves between pipelines with visual feedback
2. **Given** a pipeline is selected, **When** I press Enter, **Then** I see a detail view showing the pipeline's full configuration path, complete status breakdown, and available actions
3. **Given** I am viewing pipeline details, **When** I press Esc or 'q', **Then** I return to the main dashboard list
4. **Given** I am on the dashboard, **When** I press a number key corresponding to a pipeline (1-9), **Then** that pipeline is selected directly

---

### User Story 3 - Run Verification from Dashboard (Priority: P3)

As a user investigating a pipeline's status, I want to trigger a verification check directly from the dashboard, so I can get up-to-date status information without leaving the interface.

**Why this priority**: Adds action capability to the dashboard. Depends on P1 (listing) and P2 (selection) but provides significant value by making the dashboard actionable rather than just informational.

**Independent Test**: Can be tested independently by opening a pipeline's detail view and triggering verification. Success is measured by observing real-time progress indication and updated status display upon completion.

**Acceptance Scenarios**:

1. **Given** I am viewing a pipeline's details, **When** I press 'v' for verify, **Then** the system runs verification and shows progress indicators (spinner, step counts)
2. **Given** verification is running, **When** the process completes, **Then** the dashboard updates the status indicator and last run time automatically
3. **Given** verification encounters errors, **When** the process fails, **Then** I see detailed error messages and the failed status is reflected in the dashboard
4. **Given** verification is in progress, **When** I press Esc, **Then** a confirmation dialog appears asking "Cancel verification? (y/n)" to prevent accidental interruption
5. **Given** the cancellation confirmation is shown, **When** I press 'y', **Then** verification is cancelled and I return to the dashboard with the previous status intact
6. **Given** the cancellation confirmation is shown, **When** I press 'n' or Esc again, **Then** the confirmation closes and verification continues

---

### User Story 4 - Apply Pipeline Configuration Interactively (Priority: P4)

As a user who has identified drifted or failed pipelines, I want to apply configuration changes directly from the dashboard, so I can remediate issues without switching back to the command line.

**Why this priority**: Completes the action loop by allowing remediation. While valuable, it's lower priority because users can always exit and run `streamy apply` manually. However, it's essential for the "central workspace" vision.

**Independent Test**: Can be tested by selecting a drifted pipeline and triggering apply action. Success is verified when configuration is applied and status updates to satisfied.

**Acceptance Scenarios**:

1. **Given** I am viewing a drifted pipeline's details, **When** I press 'a' for apply, **Then** the system prompts for confirmation before proceeding
2. **Given** I confirm the apply action, **When** the process runs, **Then** I see real-time progress with step-by-step feedback
3. **Given** apply completes successfully, **When** the process finishes, **Then** the pipeline status updates to satisfied (🟢) and the dashboard refreshes
4. **Given** apply fails on a step, **When** the error occurs, **Then** I see which step failed, the error message, and the option to retry or return to dashboard

---

### User Story 5 - Refresh Dashboard Status (Priority: P5)

As a user monitoring pipeline states, I want to manually refresh all pipeline statuses, so I can get current information after making external changes or after some time has passed.

**Why this priority**: Quality-of-life feature that enhances usability but isn't critical for basic functionality. Users can restart the dashboard to refresh.

**Independent Test**: Can be tested by modifying system state externally (e.g., changing a file that affects pipeline verification), then pressing refresh and verifying status indicators update.

**Acceptance Scenarios**:

1. **Given** I am viewing the dashboard, **When** I press 'r' for refresh, **Then** all pipeline statuses are re-verified and indicators update
2. **Given** refresh is triggered, **When** multiple pipelines are being verified, **Then** I see progress indication (e.g., "Refreshing... 2/5 complete")
3. **Given** some pipelines fail during refresh, **When** the process completes, **Then** the dashboard still displays with partial results and indicates which verifications failed

---

### Edge Cases

- **No pipelines registered**: Dashboard should show a helpful empty state explaining how to add pipelines or pointing to documentation
- **Pipeline configuration file deleted**: Dashboard should indicate the pipeline is registered but configuration is missing, with option to unregister
- **Very long pipeline descriptions**: Text should wrap or truncate gracefully to fit within the terminal width
- **Terminal resize during dashboard operation**: Interface should reflow and maintain current selection state
- **Concurrent pipeline modifications**: If a pipeline's configuration changes while the dashboard is open, status may be stale until refresh
- **Permission errors**: If verification or apply fails due to permissions, error message should clearly indicate the permission issue
- **Large number of pipelines (50+)**: Dashboard should support scrolling and potentially search/filter capabilities
- **Pipeline names with special characters or emojis**: Display should handle Unicode safely without breaking layout
- **Extremely fast status changes**: If a pipeline status changes during display, the next refresh should show the current state
- **Background processes**: Dashboard should not interfere with other Streamy operations running in different terminals

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST launch an interactive TUI dashboard when `streamy` command is executed with no subcommands
- **FR-002**: Dashboard MUST display all registered pipelines with their current status indicators (🟢 satisfied, 🟡 drifted, 🔴 failed, ⚪ unknown)
- **FR-003**: Dashboard MUST show each pipeline's description and last verification time (with human-readable relative time format)
- **FR-004**: System MUST support keyboard navigation using arrow keys (up/down) to move between pipelines
- **FR-005**: System MUST allow pipeline selection via Enter key or numeric shortcuts (1-9 for first 9 pipelines)
- **FR-006**: Dashboard MUST display a detail view for selected pipelines showing full configuration path, complete status breakdown, and available actions
- **FR-007**: Users MUST be able to trigger pipeline verification directly from the detail view
- **FR-008**: Users MUST be able to trigger pipeline apply actions directly from the detail view with confirmation prompt
- **FR-009**: System MUST display real-time progress indicators during verification and apply operations (spinner, step counts, current action)
- **FR-010**: Dashboard MUST automatically update status indicators and metadata when verification or apply operations complete
- **FR-011**: Users MUST be able to return to the main dashboard list from detail views using Esc or 'q' keys
- **FR-012**: Dashboard MUST support manual refresh of all pipeline statuses via 'r' key
- **FR-013**: System MUST handle terminal resize events and adjust layout dynamically
- **FR-014**: Dashboard MUST show a helpful empty state message when no pipelines are registered
- **FR-015**: System MUST sort pipelines with failed/drifted states at the top to prioritize attention
- **FR-016**: Dashboard MUST display error messages clearly when verification or apply operations fail, including which step failed
- **FR-017**: System MUST persist and load pipeline status metadata (last run time, last known status) across dashboard sessions
- **FR-018**: Dashboard MUST support scrolling when the number of pipelines exceeds terminal height
- **FR-019**: System MUST provide a help overlay (accessed via '?' key) showing available keyboard commands
- **FR-020**: Dashboard MUST gracefully handle missing configuration files by indicating the pipeline is registered but configuration is unavailable
- **FR-021**: System MUST display a confirmation dialog when user attempts to cancel running verification or apply operations via Esc key
- **FR-022**: System MUST allow confirmed cancellation of verification or apply operations, returning user to dashboard with previous status preserved

### Key Entities

- **Pipeline Registry Entry**: Represents a registered pipeline with its name, description, configuration file path, current status, last verification time, and last known result summary
- **Dashboard State**: Represents the current view state including selected pipeline index, scroll position, active view (list vs detail), and cached pipeline statuses
- **Pipeline Status**: Represents the verification state of a pipeline including overall status (satisfied/drifted/failed/unknown), individual step statuses, error messages, and timestamp
- **Action Result**: Represents the outcome of a verification or apply operation including success status, affected steps, error details, and execution time

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can identify all pipeline statuses within 2 seconds of launching the dashboard (no need to run separate commands)
- **SC-002**: Users can navigate from dashboard launch to viewing a specific pipeline's details in under 5 seconds using keyboard controls
- **SC-003**: Users can trigger verification or apply actions and see completion within the same interface session without switching to command line
- **SC-004**: Dashboard refreshes and displays updated pipeline statuses within 3 seconds of completing a verification or apply action
- **SC-005**: 90% of dashboard operations (navigation, selection, action triggers) complete within 1 second of user input
- **SC-006**: Dashboard handles terminal resize events without crashing or corrupting the display
- **SC-007**: Empty state is displayed correctly when 0 pipelines are registered, guiding users on next steps
- **SC-008**: Dashboard can display and manage at least 50 pipelines without performance degradation
- **SC-009**: Error messages from failed operations are displayed with sufficient context for users to understand and resolve issues
- **SC-010**: Users can perform complete workflow (view status → verify → apply → confirm success) without leaving the dashboard interface

## Assumptions

- Users have registered at least one pipeline configuration before expecting meaningful dashboard content
- Terminal supports ANSI colors and Unicode characters for status indicators and visual elements
- Pipeline verification and apply operations can be executed synchronously within the dashboard process
- Users are familiar with basic keyboard navigation patterns (arrow keys, Enter, Esc)
- Dashboard will be the default behavior when running `streamy` with no arguments (may affect existing workflows)
- Pipeline status can be cached and displayed immediately, with background refresh optional
- The existing TUI infrastructure in `internal/tui/` can be extended to support this dashboard functionality
- Pipeline registry state is already tracked (based on `registry_state.go` presence in codebase)
- Verification and apply logic can be invoked programmatically from the dashboard (not just via CLI commands)
- Status determination (satisfied/drifted/failed/unknown) follows existing Streamy verification logic
- "Last run time" can be persisted to disk or in-memory state storage
- Maximum reasonable pipeline count is under 1000 (for UI performance considerations)
- Dashboard will integrate with existing logger infrastructure for operation feedback

## Dependencies

- Existing `internal/tui/` package provides foundational TUI components and patterns
- `cmd/streamy/registry_state.go` contains pipeline registry management logic
- `cmd/streamy/verify.go` and `cmd/streamy/apply.go` contain verification and apply logic that must be callable from dashboard
- Terminal UI framework (appears to be Bubble Tea based on existing code structure)
- Pipeline configuration parser (`internal/config/`) for loading and validating pipeline definitions
- DAG executor (`internal/engine/`) for running verification and apply operations

## Out of Scope

- Search or filter functionality for large pipeline lists (future enhancement)
- Pipeline creation or editing from within the dashboard (use configuration files)
- Real-time monitoring with automatic background status updates
- Pipeline scheduling or automated apply actions
- Multi-user or remote dashboard access
- Pipeline execution history or detailed audit logs within the dashboard
- Comparison views between pipeline states at different points in time
- Bulk operations (verify or apply multiple pipelines simultaneously)
- Custom dashboard themes or color schemes
- Export of dashboard state or pipeline status reports
- Integration with external monitoring or alerting systems
