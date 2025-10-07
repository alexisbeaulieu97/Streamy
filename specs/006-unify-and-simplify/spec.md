# Feature Specification: Unify and Simplify the Plugin System

**Feature Branch**: `006-unify-and-simplify`  
**Created**: October 7, 2025  
**Status**: Draft  
**Input**: User description: "Unify and Simplify the Plugin System"

## Execution Flow (main)
```
1. Parse user description from Input
   → Feature description provided
2. Extract key concepts from description
   → Plugin interface refactoring, evaluate pattern, eliminate redundancy
3. For each unclear aspect:
   → Marked with [NEEDS CLARIFICATION] where appropriate
4. Fill User Scenarios & Testing section
   → User flows defined for plugin developers and maintainers
5. Generate Functional Requirements
   → Each requirement is testable
6. Identify Key Entities (if data involved)
   → Plugin interface, evaluation state, execution modes
7. Run Review Checklist
   → Check for implementation details and clarity
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
As a Plugin Developer, I want a single, clear interface to implement so that I can focus on my plugin's core logic without worrying about boilerplate or implementing multiple, similar methods for checking state.

As a Core Maintainer, I want a lean plugin contract so that the execution engine can make intelligent decisions about Verify, DryRun, and Apply without trusting each plugin to implement that logic correctly.

### Acceptance Scenarios

1. **Given** a new plugin developer wants to create a plugin, **When** they review the plugin interface documentation, **Then** they should see a single primary method for state evaluation with clear guidance on what it should return.

2. **Given** an existing plugin with the old interface, **When** a maintainer reviews the plugin code, **Then** they should identify duplicate logic across Check, Apply, DryRun, and Verify methods.

3. **Given** a plugin implements the new unified interface, **When** the execution engine runs the plugin in different modes (verify, dry-run, apply), **Then** the engine should correctly interpret the evaluation result without requiring mode-specific plugin methods.

4. **Given** a plugin evaluates system state, **When** the evaluation completes, **Then** the result should clearly indicate: current state, desired state, whether changes are needed, and what actions would be taken.

5. **Given** multiple plugins exist in the system, **When** all plugins are migrated to the new interface, **Then** the codebase should demonstrate reduced duplication and consistent evaluation patterns across all plugins.

### Edge Cases

- What happens when a plugin's evaluation logic encounters an error during state assessment?
  → The evaluation result must distinguish between "no changes needed" and "unable to determine state" using structured error types (ValidationError, ExecutionError, StateError)

- How does the system handle plugins that have non-idempotent operations?
  → The evaluation pattern must make it obvious when an operation cannot safely determine state without side effects; evaluation is strictly read-only

- What happens when a plugin needs to return different information for verify vs. apply modes?
  → The evaluation result structure must support providing preview information (diffs, change descriptions) separate from the changed/unchanged determination

- How does the system ensure backward compatibility during migration?
  → No backward compatibility provided; this is a breaking change with all 8 built-in plugins migrating simultaneously in a single release

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The plugin system MUST provide a single primary method for plugins to evaluate system state against desired configuration
  
- **FR-002**: The plugin evaluation method MUST return a result that includes:
  - Whether the current state matches the desired state
  - A description of what changes are needed (if any)
  - Information suitable for displaying in dry-run mode (preview of changes)
  - Sufficient context for the engine to generate appropriate user messages

- **FR-003**: The execution engine MUST be able to determine from the evaluation result alone whether to skip, apply changes, or report compliance without calling additional plugin methods

- **FR-004**: Plugin developers MUST NOT be required to implement separate logic for checking state, previewing changes, applying changes, and verifying results. The unified evaluation pattern MUST naturally encourage declarative, idempotent plugin logic by making the "golden path" (single Evaluate() + simple Apply()) the simplest implementation path.

- **FR-005**: The plugin interface MUST eliminate redundant metadata methods that duplicate information already available through other plugin methods

- **FR-006**: The evaluation result MUST support idempotent operations by clearly separating state assessment from state modification

- **FR-007**: Plugins MUST be able to report evaluation errors distinctly from "state is incorrect" outcomes

- **FR-008**: The system MUST provide clear guidance and examples showing plugin developers how to structure their evaluation logic following the new pattern

- **FR-009**: All built-in plugins MUST be refactored to use the unified interface as reference implementations

- **FR-010**: The plugin system MUST support the execution engine making decisions about verify, dry-run, and apply modes based on evaluation results without delegating mode-specific behavior to plugins

- **FR-011**: Plugin evaluation results MUST include enough information for the system to generate meaningful diffs or change previews without plugins needing mode-specific formatting logic

- **FR-012**: The plugin evaluation method MUST be strictly read-only and MUST NOT modify any system state (including temporary files); all state mutations occur only during apply operations

- **FR-013**: Plugin error reporting MUST use structured error types (ValidationError, ExecutionError, StateError) with error codes and contextual information

- **FR-014**: All 8 built-in plugins MUST be migrated to the unified interface in a single release with no backward compatibility layer. Plugins will be refactored in complexity order (but shipped together):
  - **Phase 1 (Simple)**: symlink, copy - straightforward state checking
  - **Phase 2 (Medium)**: lineinfile, template - content manipulation logic
  - **Phase 3 (Complex)**: package, repo - external system interaction
  - **Phase 4 (Meta)**: command, internalexec - user-provided command execution

- **FR-015**: The unified evaluation method MAY have up to 20% performance overhead compared to the current Check() method, prioritizing correctness and diagnostic capability over raw speed

### Non-Functional Requirements

- **NFR-001**: The new plugin interface should reduce the typical plugin implementation size by eliminating boilerplate
  
- **NFR-002**: The refactoring should improve maintainability by centralizing execution mode logic in the engine rather than distributing it across plugins

- **NFR-003**: Documentation must clearly explain the migration path from the old interface to the new interface

- **NFR-004**: The new interface should make it difficult for plugin developers to accidentally create non-idempotent plugins

### Key Entities *(include if feature involves data)*

- **Plugin Interface**: The contract that all plugins must satisfy, defining how plugins communicate their capabilities and evaluate system state

- **Evaluation Result**: A structured outcome from state assessment that includes:
  - Current state information
  - Whether changes are needed
  - Description of required changes
  - Preview/diff information for display purposes
  - Any errors encountered during evaluation (using structured error types)
  - Must be producible through strictly read-only operations

- **Execution Mode**: The context in which a plugin runs (verify-only, dry-run preview, actual apply), controlled by the execution engine rather than individual plugins

- **Plugin Metadata**: Identity and capability information about a plugin (name, version, dependencies, statefulness) - to be reviewed for potential consolidation with evaluation interface

- **Error Types**: Structured error categories for evaluation failures:
  - **ValidationError**: Configuration or input validation failures
  - **ExecutionError**: System command or operation failures during state assessment
  - **StateError**: Unable to determine current system state

- **Migration Scope**: All 8 built-in plugins requiring refactoring in complexity order:
  - **Simple** (Phase 1): symlink, copy
  - **Medium** (Phase 2): lineinfile, template
  - **Complex** (Phase 3): package, repo
  - **Meta** (Phase 4): command, internalexec
  
  Note: All phases ship together in one release; ordering is for implementation risk management only.

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

### Constitutional Alignment
- **Principle I (Minimize External Dependencies)**: ✅ This refactoring is internal and removes complexity rather than adding dependencies
- **Principle II (Clear Configuration)**: ✅ Simplifying the plugin interface makes the system more intuitive for plugin developers
- **Principle IV (Safety Defaults)**: ✅ The evaluation pattern naturally separates state assessment from modification, improving safety
- **Principle V (Performance Expectations)**: ✅ Consolidating duplicate evaluation logic should improve performance

---

## Execution Status
*Updated by main() during processing*

- [x] User description parsed
- [x] Key concepts extracted
- [x] Ambiguities marked
- [x] User scenarios defined
- [x] Requirements generated
- [x] Entities identified
- [x] Review checklist evaluated

---

## Clarifications

### Session 2025-10-07

- Q: What is the migration approach for transitioning existing plugins to the new unified interface? → A: **Big bang migration** - Breaking change requiring all built-in plugins to migrate simultaneously in one release; no backward compatibility
- Q: Can the evaluation method modify system state, or must it be strictly read-only? → A: **Strictly read-only** - Evaluation MUST NOT modify any system state; only read current state and compute diffs
- Q: How detailed should plugin error reporting be when evaluation fails? → A: **Structured error types** - Categorized errors (ValidationError, ExecutionError, StateError) with codes and context
- Q: What is the acceptable performance impact of calling the unified evaluate method compared to the current separate Check() calls? → A: **Modest overhead acceptable** - Up to 20% slower acceptable if it improves correctness and maintainability
- Q: Which plugins must be refactored as part of this feature? → A: **All 8 built-in plugins** - command, copy, internalexec, lineinfile, package, repo, symlink, template

---

## Additional Context

### Current State Analysis

The existing plugin interface requires implementations to provide:
- `Check()` - determine if changes are needed
- `Apply()` - make changes and return results  
- `DryRun()` - preview what would change
- `Verify()` - confirm expected state
- `Metadata()` - return plugin identity

This leads to:
1. **Code Duplication**: Most plugins evaluate state in Check(), then re-evaluate in Apply(), DryRun(), and Verify()
2. **Inconsistency Risk**: Plugins can have different logic paths for different modes, breaking idempotency guarantees
3. **Boilerplate Burden**: Plugin developers must implement 4+ methods with overlapping concerns
4. **Trust Issues**: The engine must trust each plugin to correctly implement mode-specific behavior

### The Evaluate Pattern (from line_in_file)

The line_in_file plugin demonstrates a better approach:
- Single `evaluate()` method assesses current vs. desired state (strictly read-only)
- Returns rich result including: state info, whether changes are needed, what would change, diffs for preview
- Check(), Apply(), DryRun() all call evaluate() and interpret the result differently
- Engine could make these interpretations instead of delegating to plugins
- Evaluation may be up to 20% slower than simple checks but provides comprehensive diagnostics

### Value Proposition

**For Plugin Developers**:
- Write less code (one evaluation method instead of four)
- Impossible to have inconsistent behavior between modes
- Clear "golden path" that naturally produces idempotent plugins
- Focus on domain logic rather than framework mechanics

**For Core Maintainers**:
- Centralized execution mode logic in one place (the engine)
- Easier to ensure correct behavior across all plugins
- Reduced surface area for bugs
- Clearer architectural boundaries

**For End Users**:
- More reliable, consistent plugin behavior
- Better error messages (engine controls all messaging)
- Higher confidence in dry-run predictions matching actual results

### Migration Strategy

**Approach**: Big bang migration with no backward compatibility
- All 8 built-in plugins refactored simultaneously in a single release
- Old plugin interface completely removed from codebase
- No compatibility layer, adapter pattern, or dual-interface support

**Rationale**:
- Project is at formative stage with no external plugin ecosystem
- Ensures complete validation of new interface against all real use cases
- Eliminates permanent technical debt from supporting dual interfaces
- Forces architectural consistency across entire plugin system
- Adapter pattern is fundamentally flawed (cannot reconstruct rich EvaluationResult from simple bool)

**Execution Plan**: Migrate plugins in complexity order (simple → complex) but ship all together:
1. Simple plugins: symlink, copy
2. Medium complexity: lineinfile, package
3. Complex plugins: command, repo, template, internalexec

**Risk Mitigation**:
- All refactored plugins must pass existing test suites
- Integration tests validate cross-plugin consistency
- lineinfile serves as reference implementation
- Comprehensive documentation updated before release
