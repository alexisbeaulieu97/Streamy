# Implementation Plan: Interactive Dashboard for Pipeline Management

**Branch**: `007-build-an-interactive` | **Date**: October 8, 2025 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/007-build-an-interactive/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Build an interactive TUI dashboard that serves as Streamy's main entry point when users run `streamy` with no subcommands. The dashboard displays all registered pipelines with color-coded status indicators (🟢 satisfied, 🟡 drifted, 🔴 failed, ⚪ unknown), last run times, and descriptions. Users can navigate with keyboard controls, select pipelines to view details, and trigger verify/apply operations interactively. The goal is to transform Streamy from a one-off command runner into a central workspace for managing environment configurations.

**Technical Approach**: Implement using Bubble Tea framework (already in use for existing TUI components) with Lipgloss for styling and Bubbles for reusable components. The dashboard integrates with existing pipeline registry, verification, and apply logic. State management follows the Elm Architecture pattern (Model-Update-View) with asynchronous operations handled via tea.Cmd. Pipeline metadata is loaded from `~/.streamy/registry.json` and status is determined by invoking existing verify logic.

## Technical Context

**Language/Version**: Go 1.25.1  
**Primary Dependencies**: 
- `github.com/charmbracelet/bubbletea` v1.3.10 (TUI framework)
- `github.com/charmbracelet/lipgloss` v1.1.0 (styling/layout)
- `github.com/charmbracelet/bubbles` v0.21.0 (reusable components: list, spinner, viewport)
- `github.com/spf13/cobra` v1.10.1 (CLI framework)

**Storage**: File-based
- Pipeline registry: `~/.streamy/registry.json` (pipeline metadata: id, path, description)
- Pipeline status cache: `~/.streamy/status-cache.json` (last run time, last known status)
- Pipeline configurations: User-specified YAML files referenced by registry

**Testing**: 
- Standard library `testing` package
- `github.com/stretchr/testify` v1.11.1 for assertions
- Unit tests for model logic, integration tests for verify/apply invocation

**Target Platform**: Cross-platform CLI (Linux, macOS, Windows) - single statically-linked binary  

**Project Type**: Single Go module with integrated TUI components

**Performance Goals**: 
- Dashboard startup: <500ms with 50 registered pipelines
- Keyboard navigation: <16ms latency (60fps equivalent)
- Status refresh: <3s for 10 pipelines verified in parallel
- Memory: <50MB resident with 100 pipelines loaded

**Constraints**: 
- Must work in any terminal supporting ANSI colors (minimum 16 colors)
- Terminal width minimum: 80 columns (graceful degradation for narrower terminals)
- No external dependencies beyond Go binary (constitution principle I)
- Must handle terminal resize events without crashing
- Async operations must not block UI updates

**Scale/Scope**: 
- Target: 1-50 pipelines (typical user), support up to 1000 pipelines
- Single-user local operation (no concurrent access to registry)
- Dashboard screens: main list view + per-pipeline detail view + help overlay
- ~5-10 keyboard commands (navigation, actions, quit)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Principle I: Onboarding First (NON-NEGOTIABLE) ✅ PASS
- **Requirement**: Zero dependencies besides compiled binary, works on fresh machine
- **Assessment**: Dashboard is integrated into existing Streamy binary with no new external dependencies. Uses Bubble Tea (already required), file-based registry (already implemented). No runtime dependencies added.
- **Verdict**: COMPLIANT - Dashboard enhances onboarding by providing visual feedback without adding setup complexity.

### Principle II: Schema Clarity & Fun ✅ PASS
- **Requirement**: YAML configuration minimal, declarative, enjoyable to write
- **Assessment**: Dashboard does not introduce new configuration schema. It consumes existing pipeline registry and config files. UI/UX improvements make existing schema more discoverable through visual feedback.
- **Verdict**: COMPLIANT - Improves schema usability by surfacing validation errors and status visually.

### Principle III: Plugin-Centric Architecture ✅ PASS
- **Requirement**: Core lightweight, plugins handle domain logic
- **Assessment**: Dashboard is UI layer built on existing core (DAG execution, plugin lifecycle). No new domain logic in core. Invokes existing verify/apply logic through established interfaces.
- **Verdict**: COMPLIANT - Dashboard is presentation layer, does not violate plugin boundaries.

### Principle IV: Safety by Default (NON-NEGOTIABLE) ✅ PASS
- **Requirement**: Idempotent operations, dry-run support, destructive actions require confirmation
- **Assessment**: 
  - Dashboard invokes existing verify (read-only, safe) and apply operations
  - Apply actions require explicit user confirmation (FR-008: confirmation prompt)
  - Verification can be cancelled via confirmation dialog (prevents accidental long operations)
  - No new destructive operations introduced
- **Verdict**: COMPLIANT - Maintains existing safety guarantees, adds confirmation layers.

### Principle V: Performance & Reliability ✅ PASS
- **Requirement**: Concurrent execution where safe, <1s dry-run, clear logs, predictable errors
- **Assessment**:
  - Dashboard startup <500ms target (within <1s requirement)
  - Parallel status refresh supported (FR-012: refresh all pipelines)
  - Uses existing DAG execution engine (no changes to concurrency model)
  - Real-time progress indicators (FR-009) maintain visibility
- **Verdict**: COMPLIANT - Leverages existing performance characteristics, adds visual feedback.

### Principle VI: Extensibility & Composability ✅ PASS
- **Requirement**: Scale from minimal to complex, imports/groups/conditionals supported
- **Assessment**: Dashboard displays pipelines regardless of complexity. Handles 1-1000 pipelines (FR-018: scrolling support). Does not restrict or enhance composability - transparent to config structure.
- **Verdict**: COMPLIANT - Dashboard is a view layer, orthogonal to configuration composability.

### Principle VII: Ecosystem Consistency ✅ PASS
- **Requirement**: Consistent naming, structured errors, testing, documentation
- **Assessment**:
  - Follows existing TUI patterns (internal/tui package already established)
  - Uses standard Streamy error handling and logging
  - Will include unit tests for model logic (standard practice)
  - Keyboard commands follow common TUI conventions (arrows, Enter, Esc, q)
- **Verdict**: COMPLIANT - Extends existing TUI infrastructure with consistent patterns.

### Gate Result: ✅ ALL GATES PASSED

No constitutional violations detected. Dashboard is an additive UI feature that enhances existing functionality without compromising core principles. All safety, performance, and architecture requirements maintained.

## Project Structure

### Documentation (this feature)

```
specs/007-build-an-interactive/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
│   ├── messages.md      # Bubble Tea message contracts
│   ├── model.md         # Dashboard model interface
│   └── commands.md      # Tea.Cmd patterns for async operations
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```
# Single Go project structure (existing)
cmd/
└── streamy/
    ├── main.go              # Entry point - route to dashboard when no subcommands
    ├── root.go              # Root cobra command - check for subcommands
    ├── dashboard.go         # NEW: Dashboard command initialization
    ├── verify.go            # Existing verify logic - invokable from dashboard
    ├── apply.go             # Existing apply logic - invokable from dashboard
    └── registry_state.go    # Existing registry management

internal/
├── tui/
│   ├── model.go             # Existing TUI model for step execution
│   ├── update.go            # Existing update logic
│   ├── view.go              # Existing view rendering
│   ├── styles.go            # Existing styles
│   ├── dashboard/           # NEW: Dashboard-specific TUI components
│   │   ├── model.go         # Dashboard model (pipeline list state)
│   │   ├── update.go        # Dashboard update logic (handle navigation, actions)
│   │   ├── view.go          # Dashboard view rendering (list + detail views)
│   │   ├── detail.go        # Pipeline detail view component
│   │   ├── messages.go      # Dashboard-specific tea.Msg types
│   │   ├── commands.go      # Async command constructors (verify, apply, refresh)
│   │   └── styles.go        # Dashboard-specific styling
│   └── components/
│       └── ...              # Existing reusable components
├── registry/                # NEW: Registry management abstraction
│   ├── registry.go          # Registry CRUD operations
│   ├── types.go             # Pipeline, RegistryEntry structs
│   └── cache.go             # Status cache persistence
├── config/
│   └── ...                  # Existing config parsing (unchanged)
├── engine/
│   └── ...                  # Existing DAG executor (unchanged)
└── logger/
    └── ...                  # Existing logging (unchanged)

tests/
├── integration_dashboard_test.go  # NEW: Dashboard integration tests
├── integration_test.go            # Existing integration tests
└── ...

testdata/
├── configs/
│   └── ...                        # Existing test configs
└── registry/                      # NEW: Test registry fixtures
    ├── empty.json
    ├── single-pipeline.json
    └── multiple-pipelines.json
```

**Structure Decision**: Single Go project with modular internal packages. Dashboard implementation follows existing patterns in `internal/tui/` with a dedicated subdirectory for dashboard-specific logic. Registry operations extracted to `internal/registry/` for reusability between CLI commands and dashboard. No additional projects or services required - dashboard is integrated into the existing Streamy binary as a new TUI mode.

## Complexity Tracking

*No violations to track - all Constitution Check gates passed.*

The dashboard feature introduces no architectural complexity or constitutional violations. It's a straightforward additive UI layer leveraging existing infrastructure (Bubble Tea TUI framework, registry, verify/apply logic) without modifying core execution semantics.
