# Feature Specification: Registry Management CLI Commands

**Feature Branch**: `008-extend-streamy-with`  
**Created**: October 9, 2025  
**Status**: Draft  
**Input**: User description: "Extend Streamy with a registry management feature that allows users to register, unregister, list, and refresh pipelines directly from the command line. The registry represents a persistent index of all environment configurations managed by Streamy and serves as the data source for the interactive dashboard. The goal is to let users easily build and maintain a collection of Streamy pipelines without manually editing files. This feature improves discoverability and consistency across machines while keeping the dashboard always in sync with reality. Users can register new pipelines, remove obsolete ones, and refresh statuses to see which environments are up to date—all from a single, intuitive command set."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Register New Pipeline (Priority: P1)

A system administrator receives a new Streamy configuration file for staging environment setup. They want to add it to their tracked pipelines so it appears in the dashboard and can be managed alongside other environments. Using a single command, they register the pipeline by pointing to the configuration file path, providing a friendly name and optional description.

**Why this priority**: Core functionality that enables users to build their pipeline collection. Without this, users cannot add new pipelines to the registry, making all other features unusable. This is the foundation of the registry management system.

**Independent Test**: Can be fully tested by running the register command with a valid configuration file and verifying the pipeline appears in the registry file and subsequent list output. Delivers immediate value by making pipelines visible in the dashboard.

**Acceptance Scenarios**:

1. **Given** no existing registry file, **When** user registers their first pipeline with a valid configuration path, **Then** registry file is created and pipeline is added with unique identifier, name, path, and registration timestamp
2. **Given** an existing registry with 3 pipelines, **When** user registers a new pipeline with a unique name, **Then** the new pipeline is added to the registry and appears in subsequent listings
3. **Given** a valid configuration file, **When** user registers a pipeline without providing a description, **Then** the pipeline is registered successfully with an empty description field
4. **Given** an existing pipeline with ID "dev-env", **When** user attempts to register another pipeline with the same ID, **Then** registration fails with clear error message indicating duplicate ID

---

### User Story 2 - List All Registered Pipelines (Priority: P1)

A DevOps engineer wants to see all environment configurations they manage across multiple projects. They run a list command that displays all registered pipelines with their names, paths, current status, and last verification time in an organized, readable format.

**Why this priority**: Essential for discoverability and situational awareness. Users need to know what pipelines exist before they can manage them. This is required functionality for CLI-only workflows and validation that registrations worked correctly.

**Independent Test**: Can be fully tested by registering multiple pipelines and running the list command to verify all pipelines appear with correct information formatted clearly. Delivers value as a standalone command for pipeline discovery.

**Acceptance Scenarios**:

1. **Given** an empty registry, **When** user runs the list command, **Then** a friendly message indicates no pipelines are registered yet
2. **Given** a registry with 5 pipelines of varying statuses, **When** user runs the list command, **Then** all pipelines are displayed with ID, name, status indicator, and path in a table or structured format
3. **Given** registered pipelines with long names and paths, **When** user runs the list command in a narrow terminal, **Then** output remains readable with appropriate text wrapping or truncation
4. **Given** a registry with pipelines that have been verified recently and ones not verified, **When** user runs the list command, **Then** each pipeline shows its last run timestamp or indicates it has never been verified

---

### User Story 3 - Remove Obsolete Pipeline (Priority: P2)

A team lead needs to clean up their pipeline registry after a project is decommissioned. They identify the pipeline by name or ID and remove it using an unregister command. The system confirms the removal and updates the registry file immediately.

**Why this priority**: Important for maintenance and keeping the registry clean, but not required for initial setup. Users can work around this by manually editing the registry file if needed, though it's inconvenient.

**Independent Test**: Can be fully tested by registering a pipeline, unregistering it by ID, and verifying it no longer appears in the list. Delivers value by enabling registry cleanup without file editing.

**Acceptance Scenarios**:

1. **Given** a pipeline with ID "old-project" exists in registry, **When** user runs unregister command with that ID, **Then** pipeline is removed from registry and no longer appears in list output
2. **Given** a pipeline that does not exist, **When** user attempts to unregister it, **Then** command fails with clear error message indicating the pipeline was not found
3. **Given** a pipeline being used by the dashboard, **When** user unregisters it, **Then** the pipeline is removed and dashboard automatically updates on next refresh
4. **Given** a registry with one pipeline, **When** user unregisters it, **Then** registry file is updated to show empty pipeline list but registry file remains valid

---

### User Story 4 - Refresh Pipeline Statuses (Priority: P2)

A site reliability engineer wants to check which environments have drifted from their desired state after making manual changes during an incident. They run a refresh command that runs verification checks on all or selected pipelines and updates their status indicators to reflect current reality.

**Why this priority**: Very useful for monitoring and drift detection, but users can alternatively run verify commands individually or rely on dashboard auto-refresh. It's a convenience feature that improves workflow efficiency.

**Independent Test**: Can be fully tested by registering pipelines, making changes that cause drift, running refresh command, and verifying status indicators update correctly. Delivers value as a batch status update mechanism.

**Acceptance Scenarios**:

1. **Given** 3 registered pipelines with unknown status, **When** user runs refresh command without arguments, **Then** all pipelines are verified and status cache is updated with current satisfaction state
2. **Given** a specific pipeline ID, **When** user runs refresh command with that ID as argument, **Then** only that pipeline's status is verified and updated
3. **Given** a pipeline whose configuration file has been moved, **When** refresh command attempts to verify it, **Then** status is marked as failed with error indicating file not found
4. **Given** 10 registered pipelines, **When** user runs refresh command, **Then** progress indication shows which pipeline is being verified and total progress percentage

---

### User Story 5 - Show Pipeline Details (Priority: P3)

An engineer wants to see comprehensive information about a specific pipeline including its full path, registration date, recent execution history, and detailed status breakdown. They use an info or show command with the pipeline ID to display this detailed view.

**Why this priority**: Nice-to-have enhancement for debugging and detailed inspection. Users can get most of this information from the list command or by opening the configuration file directly. This improves usability but isn't critical for core workflows.

**Independent Test**: Can be fully tested by registering a pipeline, running verify, then displaying pipeline info to confirm all details are shown correctly. Delivers value as a detailed inspection tool.

**Acceptance Scenarios**:

1. **Given** a pipeline with ID "production", **When** user runs show command with that ID, **Then** full pipeline details are displayed including path, description, registration date, last run time, and current status
2. **Given** a pipeline that has never been verified, **When** user views its details, **Then** last run information indicates "never verified" or similar
3. **Given** a pipeline with recent execution history, **When** user views its details, **Then** last execution result is shown including success/failure, duration, and error messages if any
4. **Given** an invalid pipeline ID, **When** user runs show command, **Then** error message clearly indicates the pipeline was not found

---

### Edge Cases

- What happens when registry file is corrupted or has invalid JSON? System should detect corruption, report clear error, and offer to rebuild from backup or start fresh
- What happens when two users register the same pipeline simultaneously on shared storage? File locking or atomic writes should prevent corruption; last write wins or conflict error is returned
- What happens when a registered pipeline's configuration file is deleted? List command should indicate missing file; refresh should mark status as failed with appropriate error
- What happens when registry file path is in a read-only location? Registration attempts should fail immediately with clear permission error message
- What happens when user tries to register a configuration file that doesn't exist? Command should validate file existence before registration and fail with file-not-found error
- What happens when user registers a file that exists but isn't a valid Streamy configuration? System should detect invalid format during registration and reject it with parsing error details
- What happens when registry has 100+ pipelines? List command should support pagination or filtering; refresh should support concurrent verification to maintain reasonable performance
- What happens when pipeline ID contains special characters or spaces? IDs should be validated during registration to ensure they're filesystem-safe and URL-safe
- What happens when user runs commands from different working directories? All paths should be stored as absolute paths to ensure consistency regardless of working directory
- What happens during concurrent operations on the registry? Registry should use file locking or atomic operations to prevent race conditions and data loss

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a "register" command that adds a new pipeline to the registry by accepting a configuration file path, friendly name, and optional description
- **FR-002**: System MUST validate that the configuration file exists and is a valid Streamy configuration before registering it
- **FR-003**: System MUST generate a unique identifier for each pipeline during registration if one is not explicitly provided
- **FR-004**: System MUST store registered pipelines in a persistent registry file that survives application restarts
- **FR-005**: System MUST provide a "list" command that displays all registered pipelines with their ID, name, current status, and last verification time
- **FR-006**: System MUST provide an "unregister" command that removes a pipeline from the registry by ID (pipeline identifiers are the primary key for all registry operations)
- **FR-007**: System MUST provide a "refresh" command that updates the status of all pipelines by running verification checks
- **FR-008**: System MUST support refreshing a single pipeline's status by specifying its ID as a command argument
- **FR-009**: System MUST prevent registration of duplicate pipelines based on unique identifiers
- **FR-010**: System MUST convert relative configuration file paths to absolute paths during registration for consistency
- **FR-011**: System MUST handle missing or moved configuration files gracefully by marking status as failed with descriptive error
- **FR-012**: System MUST update the dashboard's data source automatically when registry changes are made
- **FR-013**: System MUST persist pipeline registration timestamp for audit and tracking purposes
- **FR-014**: System MUST provide clear error messages when operations fail due to file permissions, invalid IDs, or missing pipelines
- **FR-015**: System MUST support empty description fields when registering pipelines
- **FR-016**: System MUST validate pipeline IDs during registration to ensure they are filesystem-safe and contain no special characters that could cause issues
- **FR-017**: System MUST use atomic file operations when updating the registry to prevent corruption during concurrent access
- **FR-018**: System MUST provide a "show" or "info" command that displays detailed information about a specific pipeline including full path, description, registration date, and execution history
- **FR-019**: System MUST indicate progress during bulk refresh operations showing which pipeline is being processed
- **FR-020**: System MUST create registry file and parent directories automatically if they don't exist on first registration

### Key Entities

- **Pipeline Registration**: Represents a tracked environment configuration with unique identifier, friendly name, absolute file path, optional description, and registration timestamp. Links configuration files to registry for management and dashboard display.
- **Registry File**: Persistent storage of all pipeline registrations in structured format. Contains version metadata and array of pipeline entries. Serves as single source of truth for dashboard and CLI operations.
- **Pipeline Status**: Runtime state of a pipeline indicating whether it's satisfied, drifted, failed, or unknown. Updated by verification checks and cached separately from registration data. Used for visual indicators in list and dashboard views.
- **Execution History**: Record of past verify and apply operations for each pipeline including success/failure, duration, timestamps, and error details. Used for debugging and tracking environment changes over time.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can register a new pipeline in under 10 seconds using a single command with file path
- **SC-002**: List command displays all registered pipelines in under 1 second for registries containing up to 50 pipelines
- **SC-003**: Refresh command completes status updates for 10 pipelines in under 30 seconds on typical hardware
- **SC-004**: 100% of registry file modifications are atomic and corruption-free even during concurrent access
- **SC-005**: Users can identify pipeline status (satisfied/drifted/failed) at a glance from list output within 3 seconds
- **SC-006**: Zero manual file edits required for common registry operations (register, unregister, list, refresh)
- **SC-007**: Dashboard reflects registry changes within 2 seconds of command completion without requiring manual refresh
- **SC-008**: 95% of users successfully register their first pipeline without consulting documentation
- **SC-009**: Error messages for failed operations include actionable next steps in 100% of cases
- **SC-010**: Registry survives unexpected termination or power loss with zero data loss when using atomic writes

## Assumptions

- Users have read and write permissions to the directory where the registry file will be stored
- Pipeline configuration files are stored on the same machine or accessible filesystem as the registry
- Users understand basic command-line interface conventions and can provide file paths
- The dashboard component already exists and has a data loading mechanism that can be adapted to read from the registry
- Status verification is performed by existing verify command functionality that can be programmatically invoked
- Pipeline IDs will use alphanumeric characters with hyphens and underscores as allowed separators
- Registry file format will be human-readable (JSON or YAML) to enable manual inspection and emergency editing
- Concurrent access to registry will be rare enough that simple file locking or atomic writes provide adequate protection
- Most users will manage between 5-20 pipelines, with power users potentially tracking up to 100
- Pipeline paths will remain stable most of the time; moved files are exceptional cases requiring user intervention
