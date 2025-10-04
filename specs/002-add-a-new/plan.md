
# Implementation Plan: Template Plugin for Dynamic File Rendering

**Branch**: `002-add-a-new` | **Date**: 2025-10-04 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/home/alexis/Projects/Streamy/specs/002-add-a-new/spec.md`

## Execution Flow (/plan command scope)
```
1. Load feature spec from Input path
   → If not found: ERROR "No feature spec at {path}"
2. Fill Technical Context (scan for NEEDS CLARIFICATION)
   → Detect Project Type from file system structure or context (web=frontend+backend, mobile=app+api)
   → Set Structure Decision based on project type
3. Fill the Constitution Check section based on the content of the constitution document.
4. Evaluate Constitution Check section below
   → If violations exist: Document in Complexity Tracking
   → If no justification possible: ERROR "Simplify approach first"
   → Update Progress Tracking: Initial Constitution Check
5. Execute Phase 0 → research.md
   → If NEEDS CLARIFICATION remain: ERROR "Resolve unknowns"
6. Execute Phase 1 → contracts, data-model.md, quickstart.md, agent-specific template file (e.g., `CLAUDE.md` for Claude Code, `.github/copilot-instructions.md` for GitHub Copilot, `GEMINI.md` for Gemini CLI, `QWEN.md` for Qwen Code, or `AGENTS.md` for all other agents).
7. Re-evaluate Constitution Check section
   → If new violations: Refactor design, return to Phase 1
   → Update Progress Tracking: Post-Design Constitution Check
8. Plan Phase 2 → Describe task generation approach (DO NOT create tasks.md)
9. STOP - Ready for /tasks command
```

**IMPORTANT**: The /plan command STOPS at step 7. Phases 2-4 are executed by other commands:
- Phase 2: /tasks command creates tasks.md
- Phase 3-4: Implementation execution (manual or via tools)

## Summary

Add a new built-in plugin to Streamy that renders files from templates with variable substitution. The template plugin enables users to maintain single template sources while generating customized configuration files for different contexts (developers, environments, projects). Uses Go's `text/template` package for rendering with support for inline variables and environment variable fallback. Implements idempotency checks to skip rendering when output matches existing files.

**Primary Requirements**:
- Render template files to destination files with variable substitution
- Support inline `vars` map and environment variable fallback (inline takes precedence)
- Implement idempotency: skip writing if rendered output matches existing file
- Use Go `text/template` syntax (`{{.VAR}}`) with support for conditionals and loops
- Fail fast on missing variables unless `allow_missing: true` is set
- Support explicit file permissions with source file fallback
- Provide dry-run mode showing diffs without writing files

**Technical Approach**: Implement as a standard Streamy plugin following the existing plugin architecture (interface implementation, registry registration, Check/Apply/DryRun methods). Leverage Go's `text/template` package from standard library for template parsing and rendering. Implement content comparison using SHA-256 hashing for idempotency checks.

## Technical Context
**Language/Version**: Go 1.21+ (matches existing Streamy codebase)
**Primary Dependencies**: Go standard library only (`text/template`, `os`, `io`, `crypto/sha256`, `path/filepath`)
**Storage**: File system operations for template reading and destination writing
**Testing**: Go standard testing with `testing` package, table-driven tests, `t.TempDir()` for isolation
**Target Platform**: Cross-platform (Linux, macOS, Windows) - same as core Streamy
**Project Type**: Single project (plugin within existing Streamy monorepo)
**Performance Goals**: Template rendering <100ms for typical config files (<1MB), dry-run <50ms
**Constraints**: 
  - Must support both inline vars and environment variable substitution
  - If a variable is missing, must fail with clear error unless `allow_missing: true` is set
  - Idempotency required: re-running must not overwrite if identical
  - Respect dry-run (show diff without writing)
**Scale/Scope**: Single plugin implementation (~300-400 lines), 5-7 test scenarios, follows existing plugin patterns in `internal/plugins/`

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**I. Onboarding First**
- [x] Feature requires no additional dependencies beyond compiled binary (uses Go stdlib only)
- [x] No system packages, language runtimes, or external tools needed
- [x] If dependencies required, implemented as optional plugin (N/A - built-in plugin)
- [x] First-run experience documented and tested (standard plugin registration)

**II. Schema Clarity & Fun**
- [x] Configuration uses flat flags for common options (`source`, `destination`, `vars`)
- [x] Complex configs use clear nested structures with examples (`vars` map is optional)
- [x] `id` and `name` fields used appropriately (standard Step fields)
- [x] JSON schema provided for validation (TemplateStep struct with yaml tags)
- [x] Error messages include file/line context and fix suggestions (template syntax errors show line/column)

**III. Plugin-Centric Architecture**
- [x] Core logic limited to DAG execution, logging, validation (plugin handles all template logic)
- [x] Domain-specific logic implemented in plugins (template rendering is plugin responsibility)
- [x] Plugin interfaces versioned and backward compatible (implements standard plugin.Plugin interface)
- [x] Plugin contract tests included (Check, Apply, DryRun all tested)

**IV. Safety by Default**
- [x] Dry-run mode supported for preview (DryRun method shows what would be rendered)
- [x] Destructive operations require explicit flags/confirmation (overwrites handled by idempotency check)
- [x] Operations are idempotent (safe to run multiple times - content comparison before write)
- [x] Rollback/recovery procedures documented (file writes are atomic, original preserved on failure)
- [x] Parallel execution defaults are safe (no shared state between template steps)

**V. Performance & Reliability**
- [x] Dry-run completes in <1s for typical configs (<50ms per template step)
- [x] Structured logging shows task timing and dependencies (StepResult with status and messages)
- [x] Error messages include context, cause, and remediation (missing vars show variable name, template errors show line/column)
- [x] Resource limits declared for scheduling (file size limits in validation)
- [x] Timeouts configured for long operations (template parsing has sensible limits)

**VI. Extensibility & Composability**
- [x] Feature works in simple and complex scenarios (simple var substitution to complex conditionals)
- [x] No breaking changes to existing configs (new plugin type, doesn't affect existing steps)
- [x] Supports composition (imports, groups, conditionals where relevant - vars can reference env vars)
- [x] Backward compatible within major version (new feature, no existing API changes)

**VII. Ecosystem Consistency**
- [x] Follows plugin naming conventions (`id`, `name`, `enabled`, `depends_on` - standard Step fields)
- [x] Structured error handling implemented (uses pkg/errors helpers, returns StepResult)
- [x] Documentation includes schema, examples, troubleshooting (will be added to docs/plugins.md and docs/schema.md)
- [x] Version compatibility declared explicitly (Metadata includes version)

## Project Structure

### Documentation (this feature)
```
specs/[###-feature]/
├── plan.md              # This file (/plan command output)
├── research.md          # Phase 0 output (/plan command)
├── data-model.md        # Phase 1 output (/plan command)
├── quickstart.md        # Phase 1 output (/plan command)
├── contracts/           # Phase 1 output (/plan command)
└── tasks.md             # Phase 2 output (/tasks command - NOT created by /plan)
```

### Source Code (repository root)
```
internal/
├── config/
│   └── types.go           # Add TemplateStep struct
├── plugins/
│   └── template/
│       ├── template.go    # Plugin implementation (New, Metadata, Schema, Check, Apply, DryRun)
│       └── template_test.go # Unit tests with table-driven scenarios
└── plugin/
    └── interface.go       # Existing interface (no changes)

cmd/
└── streamy/
    └── plugins_import.go  # Register template plugin

docs/
├── plugins.md             # Add template plugin documentation
└── schema.md              # Add TemplateStep schema reference

testdata/
└── configs/
    └── template.yaml      # Example config with template step
```

**Structure Decision**: Single project structure (existing Streamy monorepo). The template plugin follows the established pattern in `internal/plugins/` with co-located tests. Configuration type extends `internal/config/types.go`, and plugin registration happens in `cmd/streamy/plugins_import.go`.

## Phase 0: Outline & Research

**Status**: ✅ COMPLETE

**Output**: `research.md`

**Decisions Made**:
1. **Template Engine**: Go `text/template` package (stdlib, zero deps)
2. **Variable Resolution**: Two-tier (env vars + inline, inline takes precedence)
3. **Idempotency**: SHA-256 hash comparison (fast, reliable)
4. **File Permissions**: Copy from source by default, allow explicit override
5. **Missing Variables**: Fail fast by default, opt-in tolerance via `allow_missing`
6. **Dry-Run**: Render to memory, show diff, skip write
7. **Error Handling**: Line/column precision from Go template parser
8. **Plugin Pattern**: Follow existing plugins (copy, command, package)

**Key Findings**:
- All unknowns resolved during research
- No external dependencies required
- Performance targets achievable (<100ms for typical configs)
- All risks mitigated with documented approaches

---

## Phase 1: Design & Contracts

**Status**: ✅ COMPLETE

**Outputs**:
- `data-model.md` - TemplateStep struct, validation rules, state transitions
- `contracts/plugin-interface.md` - Plugin interface contract with method specifications
- `quickstart.md` - End-to-end verification scenarios
- `AGENTS.md` - Updated with template plugin context (to be generated)

**Design Summary**:

### Data Model
- **TemplateStep struct**: 6 fields (source, destination, vars, env, allow_missing, mode)
- **Validation**: YAML tags + custom validator for variable names
- **Integration**: Inline embedding in Step struct (follows existing pattern)

### Plugin Interface
- **Metadata**: Returns "template-renderer", "1.0.0", "template"
- **Schema**: Returns TemplateStep{} for documentation
- **Check**: Fast idempotency check via SHA-256 hash comparison (<50ms)
- **Apply**: Full render + write with parent dir creation (<100ms)
- **DryRun**: Render to memory, report changes without writing (<50ms)

### Error Handling
- **ValidationError**: Config parsing errors (invalid var names, missing fields)
- **PluginError**: Runtime errors (template syntax, missing vars, I/O failures)
- **Error Messages**: Include file path, line/column, actionable suggestions

### Testing Strategy
- **Unit Tests**: Table-driven tests for each method (Check, Apply, DryRun)
- **Integration Tests**: Full quickstart scenarios in `quickstart.md`
- **Edge Cases**: Empty templates, large files, special characters, concurrent execution
- **Coverage Target**: >80%

### Files to Create
1. `internal/config/types.go` - Add TemplateStep struct
2. `internal/plugins/template/template.go` - Plugin implementation
3. `internal/plugins/template/template_test.go` - Unit tests
4. `cmd/streamy/plugins_import.go` - Import template plugin for registration
5. `docs/plugins.md` - Add template plugin documentation
6. `docs/schema.md` - Add TemplateStep schema reference
7. `testdata/configs/template.yaml` - Example configuration

---

## Phase 2: Task Planning Approach
*This section describes what the /tasks command will do - DO NOT execute during /plan*

**Task Generation Strategy**:

The `/tasks` command will generate a complete, ordered task list for implementing the template plugin. Tasks will be derived from:

1. **Data Model (data-model.md)**: 
   - Task: Add TemplateStep struct to `internal/config/types.go`
   - Task: Add validation rules for TemplateStep
   - Task: Add inline embedding in Step struct

2. **Plugin Interface (contracts/plugin-interface.md)**:
   - Task: Create plugin package directory structure
   - Task: Implement Metadata() method
   - Task: Implement Schema() method
   - Task: Implement Check() method (idempotency)
   - Task: Implement Apply() method (rendering + writing)
   - Task: Implement DryRun() method (preview)
   - Task: Implement helper methods (renderTemplate, buildContext, hashFile)

3. **Registration**:
   - Task: Create init() with plugin registration
   - Task: Import template plugin in cmd/streamy/plugins_import.go

4. **Testing (contracts/plugin-interface.md + quickstart.md)**:
   - Task: Write table-driven tests for Check()
   - Task: Write table-driven tests for Apply()
   - Task: Write table-driven tests for DryRun()
   - Task: Write integration test for quickstart scenarios
   - Task: Add test fixtures (templates, configs)

5. **Documentation**:
   - Task: Add template plugin section to docs/plugins.md
   - Task: Add TemplateStep schema to docs/schema.md
   - Task: Create example config in testdata/configs/template.yaml

**Ordering Strategy**:

1. **Foundation First** (P = can run in parallel):
   - [P] Add TemplateStep struct to types.go
   - [P] Create plugin directory structure
   - Add Step.Template field (depends on TemplateStep struct)

2. **Interface Implementation** (sequential within, parallel across):
   - Implement Metadata() and Schema() [P]
   - Implement renderTemplate() helper (core logic)
   - Implement Check() (depends on renderTemplate)
   - Implement Apply() (depends on renderTemplate)
   - Implement DryRun() (depends on renderTemplate)

3. **Integration** (sequential):
   - Add init() registration
   - Import in plugins_import.go
   - Build and verify compilation

4. **Testing** (parallel where possible):
   - [P] Write tests for Check()
   - [P] Write tests for Apply()
   - [P] Write tests for DryRun()
   - [P] Create test fixtures
   - Run integration test (depends on all above)

5. **Documentation** (can run in parallel with testing):
   - [P] Add to docs/plugins.md
   - [P] Add to docs/schema.md
   - [P] Create testdata example

**Estimated Task Count**: 20-25 tasks

**Estimated Breakdown**:
- Data model: 3 tasks
- Plugin implementation: 8-10 tasks (methods + helpers)
- Registration: 2 tasks
- Testing: 6-8 tasks
- Documentation: 3 tasks

**Dependencies**:
- No external dependencies (Go stdlib only)
- Depends on existing plugin framework (no changes needed)
- Depends on existing error handling utilities (pkg/errors)

**Parallel Execution Opportunities**:
- Initial struct definitions (data model + plugin skeleton)
- Test writing (independent test files)
- Documentation (independent from implementation)
- Fixture creation (independent from code)

**Critical Path**:
1. TemplateStep struct → Step.Template field → plugin implementation → registration → integration test

**Risk Mitigation**:
- Start with simplest method (Metadata) to validate pattern
- Write tests incrementally alongside implementation
- Use existing plugins as reference (copy.go, command.go)
- Validate against quickstart.md scenarios continuously

**IMPORTANT**: This phase is executed by the /tasks command, NOT by /plan

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)  
**Phase 4**: Implementation (execute tasks.md following constitutional principles)  
**Phase 5**: Validation (run tests, execute quickstart.md, performance validation)

## Complexity Tracking

**No constitutional violations detected.**

The template plugin implementation follows all constitutional principles:
- Uses only Go standard library (no external dependencies)
- Follows existing plugin patterns (consistency with copy, command, package plugins)
- Implements safety defaults (fail-fast on errors, idempotent operations)
- Provides clear error messages with actionable suggestions
- Maintains performance targets (<100ms for typical configs)

No complexity deviations require justification.


## Progress Tracking
*This checklist is updated during execution flow*

**Phase Status**:
- [x] Phase 0: Research complete (/plan command)
- [x] Phase 1: Design complete (/plan command)
- [x] Phase 2: Task planning approach described (/plan command)
- [ ] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [x] Initial Constitution Check: PASS (all principles satisfied)
- [x] Post-Design Constitution Check: PASS (no violations)
- [x] All NEEDS CLARIFICATION resolved (clarification session completed)
- [x] Complexity deviations documented (none - straightforward plugin implementation)

---
*Based on Constitution v2.1.1 - See `/memory/constitution.md`*
