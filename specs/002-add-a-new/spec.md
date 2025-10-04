# Feature Specification: Template Plugin for Dynamic File Rendering

**Feature Branch**: `002-add-a-new`  
**Created**: 2025-10-04  
**Status**: Draft  
**Input**: User description: "Add a new built-in plugin to Streamy: `template` - Purpose: Allow users to render files from templates with variables, making onboarding configs dynamic and team-friendly. MVP Scope: Render from a source template file to a destination file. Support variable substitution from inline `vars` or environment variables. Support idempotency: if rendered file matches existing, skip."

## Execution Flow (main)
```
1. Parse user description from Input
   ✓ Feature: Template plugin for dynamic file rendering with variable substitution
2. Extract key concepts from description
   ✓ Actors: Streamy users creating onboarding configurations
   ✓ Actions: Render template files, substitute variables, write output files
   ✓ Data: Template files, variables (inline or environment), destination files
   ✓ Constraints: Idempotency (skip if output matches), MVP scope (simple substitution)
3. For each unclear aspect:
   ✓ Template syntax format to be used
   ✓ Behavior when variable is missing
   ✓ File permission handling for rendered output
4. Fill User Scenarios & Testing section
   ✓ Completed below
5. Generate Functional Requirements
   ✓ Completed below
6. Identify Key Entities (if data involved)
   ✓ Completed below
7. Run Review Checklist
   → Ready for review
8. Return: SUCCESS (spec ready for planning)
```

---

## ⚡ Quick Guidelines
- ✅ Focus on WHAT users need and WHY
- ❌ Avoid HOW to implement (no tech stack, APIs, code structure)
- 👥 Written for business stakeholders, not developers

---

## User Scenarios & Testing *(mandatory)*

### Primary User Story
As a Streamy user configuring team onboarding, I need to render configuration files from templates with team-specific or environment-specific variables, so that I can maintain a single template source while generating customized files for different contexts (e.g., different developers, environments, or projects).

### Acceptance Scenarios

1. **Given** a template file exists at `/templates/config.tmpl` with variable placeholders, and I define inline variables in my Streamy step, **When** Streamy executes the template step, **Then** a rendered file is created at the specified destination with all variables substituted from the inline variables.

2. **Given** a template file exists with variable placeholders, and some variables are defined as environment variables, **When** Streamy executes the template step, **Then** the rendered file contains values from environment variables where inline variables are not provided.

3. **Given** a template file exists and a rendered destination file already exists with identical content to what would be rendered, **When** Streamy executes the template step in Check mode, **Then** the step reports as already satisfied (idempotent skip).

4. **Given** a template file exists and a rendered destination file already exists with different content, **When** Streamy executes the template step, **Then** the existing file is overwritten with newly rendered content.

5. **Given** a template file references a variable that is not defined in inline vars or environment variables, **When** Streamy executes the template step, **Then** the step fails with a clear error message indicating which variable is missing.

6. **Given** the destination directory does not exist, **When** Streamy executes the template step, **Then** the system creates necessary parent directories before writing the rendered file.

7. **Given** I run Streamy in dry-run mode with a template step, **When** the template step is processed, **Then** the system reports what would be rendered without actually creating or modifying files.

### Edge Cases

- What happens when the template source file does not exist?
  → Step fails with clear error indicating the missing source file path.

- What happens when the user lacks write permissions to the destination path?
  → Step fails with clear error indicating permission denied and the destination path.

- What happens when a variable value contains special characters or multi-line content?
  → System renders the variable value exactly as provided, preserving special characters and newlines.

- What happens when the destination file exists but is a directory instead of a file?
  → Step fails with clear error indicating type mismatch.

- What happens when inline variables and environment variables both define the same variable?
  → Inline variables take precedence over environment variables (explicit over implicit).

- What happens when the template file is empty?
  → System creates an empty destination file (valid use case).

- What happens when the rendered output would be identical to the template source (no variables to substitute)?
  → System creates the destination file with identical content (valid use case).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST accept a template source file path as input for the template step.

- **FR-002**: System MUST accept a destination file path where the rendered output will be written.

- **FR-003**: System MUST accept an optional inline variables map (key-value pairs) for variable substitution.

- **FR-004**: System MUST substitute variables in the template with values from inline variables when provided.

- **FR-005**: System MUST substitute variables in the template with values from environment variables when not provided in inline variables.

- **FR-006**: System MUST give precedence to inline variables over environment variables for the same variable name.

- **FR-007**: System MUST fail with a descriptive error when a template references a variable that is undefined in both inline vars and environment variables.

- **FR-008**: System MUST read the template source file and render output to the destination file path.

- **FR-009**: System MUST create parent directories for the destination file if they do not exist.

- **FR-010**: System MUST support idempotent execution by comparing the rendered output content with existing destination file content before writing.

- **FR-011**: System MUST skip writing the destination file when the rendered content matches the existing file content (idempotency check passes).

- **FR-012**: System MUST overwrite the destination file when the rendered content differs from existing content.

- **FR-013**: System MUST report clear error messages when the source template file does not exist.

- **FR-014**: System MUST report clear error messages when write permissions are denied for the destination path.

- **FR-015**: System MUST support dry-run mode by previewing what would be rendered without modifying files.

- **FR-016**: System MUST preserve the template source file unchanged (read-only operation on source).

- **FR-017**: System MUST handle multi-line variable values and special characters in variable values without corruption.

- **FR-018**: System MUST validate that the destination path is not a directory when a file is expected.

- **FR-019**: System MUST handle empty template files as valid input (rendering to empty output file).

- **FR-020**: System MUST support a defined template syntax format for variable placeholders within template files. [NEEDS CLARIFICATION: specific template syntax not specified - Go templates, Mustache, simple ${VAR}, or other format?]

- **FR-021**: System MUST handle template syntax errors gracefully with clear error messages indicating the issue and location in the template. [NEEDS CLARIFICATION: error handling severity and detail level for malformed templates]

- **FR-022**: System MUST preserve or set file permissions on the rendered destination file. [NEEDS CLARIFICATION: file permission behavior not specified - copy from template source, use default umask, or allow explicit configuration?]

### Key Entities

- **Template Step Configuration**: Represents a single template rendering operation defined in a Streamy configuration file. Contains source template path, destination file path, optional inline variables map, and standard step metadata (ID, description, dependencies).

- **Template Source File**: An existing file containing static content with variable placeholders following a defined syntax. Remains unchanged during execution.

- **Rendered Destination File**: The output file created by substituting variables in the template with actual values. May be newly created or overwritten based on idempotency check.

- **Variable**: A named placeholder in a template that gets replaced with an actual value. Can be sourced from inline step configuration or environment variables, with inline taking precedence.

---

## Review & Acceptance Checklist
*GATE: Automated checks run during main() execution*

### Content Quality
- [x] No implementation details (languages, frameworks, APIs)
- [x] Focused on user value and business needs
- [x] Written for non-technical stakeholders
- [x] All mandatory sections completed

### Requirement Completeness
- [ ] No [NEEDS CLARIFICATION] markers remain (3 clarifications needed)
- [x] Requirements are testable and unambiguous (except marked items)
- [x] Success criteria are measurable
- [x] Scope is clearly bounded (MVP: simple template rendering with variable substitution)
- [x] Dependencies and assumptions identified (assumes file system access, environment variable access)

### Constitution Alignment
- **Principle I (Minimal Dependencies)**: ✓ Template rendering can use standard library capabilities, no external dependencies required for MVP
- **Principle II (Declarative Schema)**: ✓ Configuration is intuitive - source, destination, and variables are clear concepts
- **Principle IV (Safe Defaults)**: ⚠ Needs clarification on file permission behavior to ensure safe defaults
- **Principle V (Performance)**: ✓ Template rendering expected to be fast for typical config files (< 1MB)

---

## Execution Status
*Updated by main() during processing*

- [x] User description parsed
- [x] Key concepts extracted
- [x] Ambiguities marked (3 clarifications needed)
- [x] User scenarios defined
- [x] Requirements generated
- [x] Entities identified
- [ ] Review checklist passed (pending clarifications)

---

## Clarifications Needed Before Implementation

1. **Template Syntax**: Which template syntax should be used for variable placeholders? Options include Go text/template syntax, simple shell-style `${VAR}` or `$VAR`, Mustache/Handlebars `{{VAR}}`, or other formats. Recommendation: Go text/template for consistency with Go ecosystem, but `${VAR}` for simplicity in MVP.

2. **File Permissions**: How should file permissions be handled for rendered files? Options include: (a) use system default umask, (b) copy permissions from template source file, or (c) allow explicit permission configuration in step. Recommendation: Copy from source by default with optional explicit permission field.

3. **Template Error Handling**: What level of detail should be provided when template syntax is malformed? Should execution continue with warnings or fail fast? Recommendation: Fail fast with clear error messages showing line/column of syntax errors for rapid debugging.

---
