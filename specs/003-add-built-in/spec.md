# Feature Specification: Add Built-In Plugin: line_in_file

**Feature Branch**: `003-add-built-in`  
**Created**: October 4, 2025  
**Status**: Draft  
**Input**: User description: "Add Built-In Plugin: line_in_file - Introduce a new built-in plugin that provides a generic, idempotent way to ensure specific lines exist or are removed in text files."

## Execution Flow (main)
```
1. Parse user description from Input
   → Feature clearly defined: line_in_file plugin for idempotent text file line management
2. Extract key concepts from description
   → Actors: Streamy users configuring systems declaratively
   → Actions: ensure lines present/absent, regex matching, backup files
   → Data: text files, lines, regex patterns
   → Constraints: idempotency, dry-run support, integration with DAG execution
3. For each unclear aspect:
   → All key aspects specified in user description
4. Fill User Scenarios & Testing section
   → Clear user flows identified for present/absent states
5. Generate Functional Requirements
   → All requirements testable and specific
6. Identify Key Entities
   → Step configuration, file state, match results
7. Run Review Checklist
   → No implementation details leaked
   → All requirements testable
8. Return: SUCCESS (spec ready for planning)
```

---

## ⚡ Quick Guidelines
- ✅ Focus on WHAT users need and WHY
- ❌ Avoid HOW to implement (no tech stack, APIs, code structure)
- 👥 Written for business stakeholders, not developers

---

## Clarifications

### Session 2025-10-04
- Q: When `state: present` with a `match` pattern finds multiple matching lines in a file, what should the behavior be? → A: Provide a plugin option to specify behavior in advance; if not specified, prompt the user at runtime
- Q: What is the maximum file size (in MB or lines) that `line_in_file` should support before reporting an error or warning? → A: No limit (handle files of any size)
- Q: When creating a backup file with `backup: true`, where should the backup be stored and what should the naming format be? → A: Configurable backup directory with default being same directory as original. Format named <filename>.<timestamp>.bak
- Q: How should the plugin handle symbolic links when the target file path points to a symlink? → A: Follow symlink and modify the target file
- Q: What file encoding should the plugin assume and how should it handle files with different encodings? → A: UTF-8 by default with configurable `encoding` option

---

## User Scenarios & Testing

### Primary User Story
A developer is provisioning a new development environment using Streamy. They need to ensure that specific environment variables are present in their shell profile, that certain configuration lines exist in application config files, and that outdated or conflicting lines are removed—all in a declarative, repeatable way without manual editing or error-prone shell commands.

### Acceptance Scenarios

1. **Given** a file `~/.bashrc` without a specific PATH export, **When** a step with `type: line_in_file`, `state: present`, and `line: 'export PATH="$PATH:~/bin"'` is executed, **Then** the line is appended to the file and subsequent runs make no further changes.

2. **Given** a file `/etc/app.conf` containing the line `debug=true`, **When** a step with `type: line_in_file`, `match: '^debug='`, `line: 'debug=false'`, and `state: present` is executed, **Then** the matching line is replaced with `debug=false`.

3. **Given** a file `~/.profile` containing the line `export OLD_VAR=value`, **When** a step with `type: line_in_file`, `match: '^export OLD_VAR='`, and `state: absent` is executed, **Then** the matching line is removed from the file.

4. **Given** a file that does not exist, **When** a step with `type: line_in_file` and `state: present` is executed, **Then** the file is created with the specified line.

5. **Given** any configuration file, **When** a step is executed with `backup: true`, **Then** a timestamped backup of the original file is created in the same directory (or custom `backup_dir` if specified) with the naming format `<filename>.<timestamp>.bak` before any modifications.

6. **Given** a step with `state: present` and a line already present in the file, **When** the step is executed in `--dry-run` mode, **Then** no changes are made and the system reports "no changes required".

### Edge Cases

- What happens when the file is read-only or lacks write permissions? System reports an error and halts execution for that step.
- What happens when multiple lines match the regex pattern? System behavior is controlled by the `on_multiple_matches` field: replace first, replace all, fail with error, or prompt user interactively.
- What happens when `state: absent` is specified but no match pattern is provided? System reports a validation error during configuration parsing.
- What happens when the file contains the line multiple times and `state: present` is used without a match pattern? System ensures at least one occurrence exists (idempotent).
- What happens when running the same configuration multiple times? System detects no changes needed and reports success without modifying the file.
- What happens when the target file path is a symbolic link? System follows the symlink and modifies the target file.
- What happens when a file has a non-UTF-8 encoding? System uses the encoding specified in the `encoding` field or defaults to UTF-8.

---

## Requirements

### Functional Requirements

- **FR-001**: System MUST support a plugin type `line_in_file` that operates on text files specified by an absolute or tilde-expanded path.
- **FR-001a**: System MUST support an optional `encoding` field to specify file encoding (e.g., `utf-8`, `ascii`, `latin-1`); if not specified, default to UTF-8.
- **FR-002**: System MUST support a `state` field with values `present` (ensure line exists) or `absent` (ensure line is removed).
- **FR-003**: System MUST support a `line` field containing the exact text content to ensure or remove.
- **FR-004**: System MUST support an optional `match` field containing a regular expression pattern to identify existing lines for replacement or removal.
- **FR-004a**: System MUST support an optional `on_multiple_matches` field with values `first` (replace only first match), `all` (replace all matches), `error` (fail the step), or `prompt` (ask user at runtime). If not specified, default behavior is `prompt`.
- **FR-005**: When `state: present` and no `match` pattern is provided, system MUST append the line if it does not already exist in the file.
- **FR-006**: When `state: present` and a `match` pattern is provided, system MUST replace matching line(s) according to the `on_multiple_matches` setting or prompt the user if multiple matches are found and no preference is specified.
- **FR-007**: When `state: absent` and a `match` pattern is provided, system MUST remove all lines matching the pattern.
- **FR-008**: System MUST validate that `state: absent` includes a `match` pattern (cannot remove without specifying what to remove).
- **FR-009**: System MUST create the target file if it does not exist and `state: present`.
- **FR-010**: System MUST support an optional `backup` field (boolean) that creates a timestamped backup of the file before modification.
- **FR-010a**: System MUST support an optional `backup_dir` field to specify where backup files are stored; if not specified, backups are created in the same directory as the original file.
- **FR-010b**: System MUST name backup files using the format `<filename>.<timestamp>.bak` where timestamp is in ISO 8601 format (e.g., `bashrc.20251004T153045.bak`).
- **FR-011**: System MUST implement idempotency: running the same configuration multiple times produces the same file state without unnecessary writes.
- **FR-012**: System MUST integrate with Streamy's `--dry-run` mode to preview changes without applying them.
- **FR-013**: System MUST integrate with Streamy's `--verbose` mode to display diffs showing what lines were added, replaced, or removed.
- **FR-014**: System MUST report errors when file permissions prevent reading or writing.
- **FR-015**: System MUST handle multiple matching lines according to the `on_multiple_matches` configuration field or prompt the user if unspecified.
- **FR-016**: System MUST expand tilde (`~`) to the user's home directory in file paths.
- **FR-017**: System MUST support dependency ordering via Streamy's DAG execution (e.g., `line_in_file` steps can depend on file creation steps).
- **FR-018**: System MUST validate regex patterns in the `match` field during configuration parsing and report syntax errors.
- **FR-019**: System MUST preserve file permissions and ownership when modifying existing files.
- **FR-019a**: System MUST follow symbolic links and modify the target file when the specified path is a symlink.
- **FR-019b**: System MUST preserve the original file encoding when modifying files, using UTF-8 as default or the encoding specified in the `encoding` field.
- **FR-020**: System MUST report step success when file state matches the desired state without changes (idempotent success).
- **FR-021**: System MUST handle files of any size without imposing arbitrary limits, though performance may degrade for very large files.

### Key Entities

- **LineInFileStep**: A step configuration defining the target file, desired line content, state (present/absent), optional match pattern, backup preference, and optional backup directory.
- **FileState**: The current state of a text file including its contents, existence, and permissions.
- **MatchResult**: The result of applying a regex match pattern to a file, including matched lines and their positions.
- **BackupFile**: A timestamped copy of the original file created before modifications when `backup: true`.
- **ChangeSet**: A record of planned or applied modifications (additions, replacements, deletions) for reporting and dry-run preview.

---

## Review & Acceptance Checklist
*GATE: Automated checks run during main() execution*

### Content Quality
- [x] No implementation details (languages, frameworks, APIs)
- [x] Focused on user value and business needs
- [x] Written for non-technical stakeholders
- [x] All mandatory sections completed

### Requirement Completeness
- [x] No [NEEDS CLARIFICATION] markers remain
- [x] Requirements are testable and unambiguous  
- [x] Success criteria are measurable
- [x] Scope is clearly bounded
- [x] Dependencies and assumptions identified

---

## Execution Status
*Updated by main() during processing*

- [x] User description parsed
- [x] Key concepts extracted
- [x] Ambiguities marked (none found)
- [x] User scenarios defined
- [x] Requirements generated
- [x] Entities identified
- [x] Review checklist passed

---
