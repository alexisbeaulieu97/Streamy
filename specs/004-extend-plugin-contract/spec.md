# Feature Specification: Extend Plugin Contract with Verify Lifecycle

**Feature Branch**: `004-extend-plugin-contract`  
**Created**: October 4, 2025  
**Status**: Draft  
**Input**: User description: "Extend Plugin Contract: Add Verify Lifecycle"

## Execution Flow (main)
```
1. Parse user description from Input
   → ✓ Parsed: "Extend Plugin Contract: Add Verify Lifecycle"
2. Extract key concepts from description
   → ✓ Identified: verification, plugin lifecycle, state checking, drift detection
3. For each unclear aspect:
   → ✓ No major ambiguities (detailed description provided)
4. Fill User Scenarios & Testing section
   → ✓ Clear user flows: verify command, apply optimization, audit mode
5. Generate Functional Requirements
   → ✓ Each requirement is testable
6. Identify Key Entities
   → ✓ VerificationStatus, VerificationResult
7. Run Review Checklist
   → ✓ No implementation details included
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
- Q: When the command plugin has no verification command specified, what should the default verification behavior be? → A: A new fifth status "unknown" alongside satisfied/missing/drifted/blocked, with re-apply as default behavior
- Q: What's the expected timeout threshold for verification checks mentioned in FR-021? → A: Configurable per-step with global default of 30 seconds
- Q: Should the verify command support filtering/selecting specific steps, or always verify the entire configuration? → A: Always verify all steps (simple, comprehensive)
- Q: When reporting "drifted" status, how much detail should the verify output include about the difference? → A: Show full diff output (like git diff)
- Q: How should verification handle transient failures (e.g., temporary network issues, file locks)? → A: Fail immediately, report as "blocked"

---

## User Scenarios & Testing

### Primary User Story
As a **DevOps engineer or developer** managing environment configurations with Streamy, I want to **verify that my system matches the declared configuration** without applying any changes, so that I can:
- Understand what needs to change before running apply
- Detect configuration drift on existing environments
- Skip steps that are already satisfied during re-runs
- Audit compliance across multiple machines or deployments

### Acceptance Scenarios

1. **Given** a system with some configuration steps already applied, **When** I run the verify command, **Then** the system reports which steps are satisfied, missing, or drifted without modifying any files or settings.

2. **Given** a fresh machine with no configuration, **When** I run verify, **Then** all steps are reported as "missing" and I see a summary count of what needs to be applied.

3. **Given** a partially configured system where some resources have changed manually, **When** I run verify, **Then** steps with mismatches are reported as "drifted" with details about the difference.

4. **Given** a fully configured system, **When** I run apply after verify shows all satisfied, **Then** the apply command skips all steps and completes immediately with no changes.

5. **Given** a configuration with dependencies (e.g., step B depends on step A), **When** step A is missing during verify, **Then** step B is reported as "blocked" because its prerequisite is not satisfied.

6. **Given** a verify command running on a system, **When** any plugin's verify logic executes, **Then** no files, packages, or system state are modified (read-only operation guaranteed).

### Edge Cases
- What happens when a verification check encounters a permission error (e.g., cannot read a file)? → Report as "blocked" with error details
- How does verify handle steps with no clear verification logic (e.g., arbitrary commands without check clauses)? → Return "unknown" status; system will re-apply by default for safety
- What if dependency chains create circular references? → DAG validation catches this before verify runs (existing behavior)
- How does verify handle expensive checks (e.g., large file comparisons)? → Must complete within reasonable time; plugins should optimize checks
- How should verification handle transient failures (e.g., network issues, file locks)? → Fail immediately and report as "blocked" with error details; user can re-run verify when issue resolves

---

## Requirements

### Functional Requirements

**Core Verification Capabilities**
- **FR-001**: System MUST allow users to run a dedicated verify command that checks configuration state without modifying resources.
- **FR-002**: Every plugin MUST provide a verification method that inspects current system state and returns one of five statuses: satisfied, drifted, missing, blocked, or unknown.
- **FR-003**: Verification checks MUST be read-only operations that never modify files, install packages, or change system settings.
- **FR-004**: System MUST execute verification in dependency order (DAG-based), verifying prerequisites before dependent steps.

**Status Reporting**
- **FR-005**: System MUST report "satisfied" when current state exactly matches the declared configuration for a step.
- **FR-006**: System MUST report "missing" when a required resource, file, or configuration does not exist.
- **FR-007**: System MUST report "drifted" when partial matches or unexpected differences are detected between current and expected state.
- **FR-008**: System MUST report "blocked" when a step cannot be verified because a dependency failed verification or prerequisites are unsatisfied.
- **FR-008a**: System MUST report "unknown" when verification status cannot be determined (e.g., command plugin with no verification command specified).

**Integration with Apply** *(Phase 2 - Future Scope)*
- **FR-009** *(Phase 2)*: During normal apply execution, system SHOULD use verification results to skip steps already in the satisfied state.
- **FR-009a** *(Phase 2)*: During apply execution, system SHOULD re-apply steps with "unknown", "missing", "drifted", or "blocked" status (only "satisfied" triggers skip).
- **FR-010** *(Phase 2)*: System SHOULD log or display which steps were skipped due to verification showing satisfied state.
- **FR-011** *(Phase 2)*: Users SHOULD be able to force re-application of satisfied steps via command flag (override verification optimization).

**Note**: FR-009 through FR-011 describe apply command optimization using verification results. Phase 1 (current implementation) focuses on standalone `streamy verify` command. Apply integration will be addressed in Phase 2.

**Output and Reporting**
- **FR-012**: Verify command MUST display a human-readable summary showing step names, verification statuses, and a final count of satisfied/missing/drifted/blocked/unknown steps.
- **FR-013**: System MUST provide detailed output mode showing why each step received its verification status (e.g., file not found, hash mismatch).
- **FR-013a**: For "drifted" status, system MUST display full diff output showing the differences between expected and actual state in unified diff format.
- **FR-014**: Verify command MUST exit with status code 0 when all steps are satisfied, and non-zero when any step is missing, drifted, blocked, or unknown.

**Plugin-Specific Verification**
- **FR-015**: Symlink plugin MUST verify that the target path exists and points to the expected source path.
- **FR-016**: Package plugin MUST verify that specified packages are installed at the correct version or any version if not specified.
- **FR-017**: Repository plugin MUST verify that the repository exists at the expected path and is on the correct branch.
- **FR-018**: Template plugin MUST verify that the destination file matches the rendered template output (content comparison).
- **FR-019**: Line-in-file plugin MUST verify that expected lines exist or are absent as configured.
- **FR-020**: Command plugin MUST run an optional verification command if specified; if no verification command is provided, MUST return "unknown" status.

**Safety and Performance**
- **FR-021**: Verification checks MUST complete within a configurable timeout (default 30 seconds per step) to prevent long-running operations from blocking user workflows.
- **FR-021a**: Users MUST be able to override verification timeout on a per-step basis for operations requiring longer execution (e.g., large file comparisons, network operations).
- **FR-022**: System MUST handle verification errors gracefully and report them without failing the entire verification run.
- **FR-022a**: When verification encounters transient failures (network issues, file locks, permission errors), system MUST report the step as "blocked" with error details and continue verifying remaining steps.
- **FR-023**: System MUST maintain consistent verification behavior regardless of the number of steps in the configuration.

### Key Entities

- **VerificationStatus**: Represents the state match level for a single step. Possible values: `satisfied` (matches expected), `missing` (resource not found), `drifted` (partial match or difference), `blocked` (dependency failed or prerequisite unsatisfied), `unknown` (verification status cannot be determined). Used by plugins to communicate verification outcome.

- **VerificationResult**: Contains the outcome of verifying a single step, including step identifier, verification status, human-readable message explaining the status, any error encountered, and metadata like duration. Used by the executor to aggregate and report results.

- **VerifyCommand**: User-facing command that orchestrates verification across all steps in a configuration file. Accepts configuration path, verbosity flags, and output format preferences. Always verifies all steps in dependency order. Returns aggregated verification summary.

---

## Alignment with Streamy Principles

**Principle I (Self-Contained)**: Verification adds no external dependencies; uses existing plugin infrastructure and extends the current contract.

**Principle II (Configuration-First)**: Verification output clearly maps to declared configuration, helping users understand state alignment without inspecting system manually.

**Principle III (Reproducibility)**: Verification enables deterministic re-runs by identifying which steps need work, improving idempotency and reducing unnecessary changes.

**Principle IV (Safety)**: Verification is explicitly read-only and never modifies state, providing a safe audit mechanism before applying changes. Supports "understand before act" workflow.

**Principle V (Performance)**: Verification optimization allows apply to skip satisfied steps, reducing execution time on subsequent runs and minimizing unnecessary system operations.

---

## Review & Acceptance Checklist

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

- [x] User description parsed
- [x] Key concepts extracted
- [x] Ambiguities marked (none found)
- [x] User scenarios defined
- [x] Requirements generated
- [x] Entities identified
- [x] Review checklist passed

---
