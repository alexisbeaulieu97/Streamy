# Implementation Plan: Registry Management CLI Commands

**Branch**: `008-extend-streamy-with` | **Date**: October 9, 2025 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/008-extend-streamy-with/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

This feature extends Streamy with CLI commands for managing a persistent registry of pipeline configurations. Users can register new pipelines, list all registered pipelines with their current status, unregister obsolete configurations, and refresh verification statuses in batch. The registry serves as the single source of truth for the interactive dashboard, enabling full lifecycle management without manual file editing.

**Technical Approach**: Leverage existing `internal/registry` package which already provides thread-safe Registry and StatusCache types with atomic file operations. Add four new Cobra commands (register, unregister, list, refresh) that manipulate the registry at `~/.streamy/registry.json` and status cache at `~/.streamy/status.json`. Commands will integrate with existing verify/apply infrastructure and update the dashboard's data source automatically.

## Technical Context

**Language/Version**: Go 1.25.1  
**Primary Dependencies**: 
  - `github.com/spf13/cobra` (CLI framework, already in use)
  - `encoding/json` (standard library for serialization)
  - Existing internal packages: `internal/registry`, `internal/config`, `internal/engine`
  
**Storage**: 
  - Registry file: `~/.streamy/registry.json` (pipeline metadata)
  - Status cache: `~/.streamy/status.json` (runtime status)
  - Both use atomic writes (temp file + rename pattern)
  
**Testing**: Standard Go testing (`go test`), existing test patterns in `cmd/streamy/*_test.go`  
**Target Platform**: Linux, macOS, Windows (cross-platform via Go)  
**Project Type**: Single CLI application with TUI dashboard integration  
**Performance Goals**: 
  - Register command: <100ms for validation + persistence
  - List command: <1s for 50 pipelines
  - Refresh command: <30s for 10 pipelines (concurrent verification)
  
**Constraints**: 
  - Zero breaking changes to existing registry data structures
  - Backward compatible with dashboard's registry loading
  - Thread-safe for concurrent operations
  
**Scale/Scope**: Support 5-100 registered pipelines per user

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### ✅ I. Onboarding First (NON-NEGOTIABLE)
**Status**: PASS  
**Analysis**: Feature adds optional commands to existing binary. No new dependencies beyond what's already compiled in. User downloads binary → can immediately use `streamy register` without installing anything.

**Post-Design Verification**: ✅ CONFIRMED - Implementation uses only standard library (`encoding/json`, `text/tabwriter`, `bufio`) and existing Cobra dependency. Zero new external dependencies.

### ✅ II. Schema Clarity & Fun
**Status**: PASS  
**Analysis**: Commands use simple, intuitive arguments (e.g., `streamy register <path> --description "text"`). Registry JSON schema is already defined in `internal/registry/types.go`. No new configuration syntax introduced—commands manipulate existing well-defined structures.

**Post-Design Verification**: ✅ CONFIRMED - Command contracts documented in `contracts/registry-cli.md` with clear examples. Error messages include actionable suggestions. JSON output schema is stable and versioned.

### ✅ III. Plugin-Centric Architecture
**Status**: PASS  
**Analysis**: Feature is core CLI functionality, not plugin logic. Commands orchestrate existing registry and engine components. No plugin API changes required. Aligns with core responsibility of "DAG resolution, plugin lifecycle, structured logging, error handling."

**Post-Design Verification**: ✅ CONFIRMED - No changes to plugin API. Commands are pure orchestration layer using existing `internal/registry` and `internal/config` packages.

### ✅ IV. Safety by Default (NON-NEGOTIABLE)
**Status**: PASS with Notes  
**Analysis**: 
  - `register`: Validates config exists and parses before adding (safe)
  - `unregister`: Prompts for confirmation unless `--force` (safe)
  - `list`: Read-only (inherently safe)
  - `refresh`: Dry-run mode already exists in verify command, reuse pattern (safe)
  - Registry uses atomic writes (existing implementation prevents corruption)
  
**Note**: Spec requires unregister confirmation prompt—implementation MUST enforce this unless `--force` flag is provided.

**Post-Design Verification**: ✅ CONFIRMED - Contracts specify confirmation prompt with `bufio.Scanner`. Atomic writes via temp-file-then-rename pattern. All destructive operations require explicit confirmation or `--force` flag. Dry-run mode documented.

### ✅ V. Performance & Reliability
**Status**: PASS  
**Analysis**: 
  - Registry load/save uses efficient JSON encoding with RWMutex for concurrency
  - List command reads in-memory data (fast)
  - Refresh uses existing verify engine which supports concurrent execution via DAG
  - Error messages from existing registry package are descriptive with context
  
**Performance Targets Met**:
  - Dry-run planning: Existing verify completes quickly
  - Concurrent refresh: DAG engine already handles parallel verification

**Post-Design Verification**: ✅ CONFIRMED - Quickstart specifies performance targets (<100ms register, <1s list 50 pipelines, <30s refresh 10 pipelines). Worker pool pattern with semaphore limits concurrency. Structured error messages with suggestions documented in contracts.

### ✅ VI. Extensibility & Composability
**Status**: PASS  
**Analysis**: 
  - Commands integrate cleanly with existing CLI structure via Cobra
  - Registry schema versioning already implemented (`version: "1.0"` field)
  - Future enhancements (filters, pagination, export) can be added without breaking existing commands
  - Dashboard already reads from same registry file—zero breaking changes

**Post-Design Verification**: ✅ CONFIRMED - Data model includes schema versioning with migration strategy documented. Contracts define API stability promise. Future enhancements (show command, filters, pagination) identified in quickstart without requiring breaking changes.

### ✅ VII. Ecosystem Consistency
**Status**: PASS  
**Analysis**: 
  - Command naming follows Unix conventions: `register`, `unregister`, `list`, `refresh`
  - Flag naming consistent with existing commands: `--verbose`, `--force`, `--description`
  - Error handling follows established pattern from `cmd/streamy/apply.go` and `verify.go`
  - Registry entities already use consistent field names: `id`, `name`, `path`, `description`

**Post-Design Verification**: ✅ CONFIRMED - Contracts standardize output format (table/JSON), exit codes (0/1/2), error message structure (context + suggestion). Follows existing command patterns from apply.go/verify.go. All entities use consistent naming (`id`, `name`, `status`, `last_run`).

### Overall Gate Result: ✅ PASS - Design Complete, Ready for Implementation

**Post-Design Summary**: All constitutional principles validated against detailed design artifacts (data-model.md, contracts/registry-cli.md, quickstart.md). No violations introduced during design phase. Implementation can proceed with confidence.

## Project Structure

### Documentation (this feature)

```
specs/008-extend-streamy-with/
├── plan.md              # This file (/speckit.plan command output)
├── spec.md              # Feature specification (already created)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
│   └── registry-cli.md  # Command interface contracts
└── checklists/
    └── requirements.md  # Validation checklist (already created)
```

### Source Code (repository root)

```
cmd/streamy/
├── root.go              # Add new commands to root (modify)
├── registry.go          # NEW: Parent command for registry subcommands
├── register.go          # NEW: Register command implementation
├── unregister.go        # NEW: Unregister command implementation  
├── list.go              # NEW: List command implementation
├── refresh.go           # NEW: Refresh command implementation
├── register_test.go     # NEW: Register command tests
├── unregister_test.go   # NEW: Unregister command tests
├── list_test.go         # NEW: List command tests
└── refresh_test.go      # NEW: Refresh command tests

internal/registry/
├── registry.go          # Existing: Registry type with Add/Remove/List
├── cache.go             # Existing: StatusCache for runtime status
├── types.go             # Existing: Pipeline, PipelineStatus types
└── helpers.go           # NEW: Helper functions for CLI (ID generation, validation)

internal/config/
└── parser.go            # Existing: Config validation (used by register)

tests/
└── integration_registry_test.go  # NEW: End-to-end CLI tests
```

**Structure Decision**: Single project structure (existing). New CLI commands in `cmd/streamy/`, leveraging existing `internal/registry` package which already provides required persistence layer. Minimal new code—primarily command handlers and integration glue.

## Complexity Tracking

*No constitutional violations—table not required.*
