# Research & Technical Decisions: Interactive Dashboard

**Phase**: 0 (Outline & Research)  
**Date**: October 8, 2025  
**Feature**: Interactive Dashboard for Pipeline Management

## Overview

This document captures research findings, technology choices, and design patterns for implementing the interactive dashboard feature. All technical unknowns from the plan's Technical Context have been investigated and resolved.

## Technology Choices

### 1. TUI Framework: Bubble Tea (Retained)

**Decision**: Continue using `github.com/charmbracelet/bubbletea` v1.3.10

**Rationale**:
- Already integrated in Streamy's codebase (`internal/tui/` package)
- Follows the Elm Architecture (Model-Update-View) pattern - proven for complex interactive UIs
- Excellent async operation support via `tea.Cmd` - critical for verify/apply operations
- Active maintenance and strong ecosystem (Lipgloss, Bubbles components)
- Zero runtime dependencies (compiles into binary - satisfies Constitution Principle I)

**Alternatives Considered**:
- **tview**: More widget-heavy, but less flexible for custom layouts. Bubble Tea's composability better suits Streamy's needs.
- **termui**: Primarily for dashboards with graphs/charts. Overkill for text-based list navigation.
- **Raw termbox-go**: Too low-level. Would require reimplementing event handling, layout, and async patterns.

**Best Practices**:
- Separate concerns: Model (state), Update (business logic), View (rendering)
- Use `tea.Cmd` for all I/O operations (file reads, subprocess execution)
- Batch commands with `tea.Batch()` for concurrent operations
- Immutable updates: return new model copies, never mutate in-place
- Small, focused message types for clarity

### 2. Layout & Styling: Lipgloss

**Decision**: Use `github.com/charmbracelet/lipgloss` v1.1.0 for all styling and layout

**Rationale**:
- Declarative CSS-like API for terminal styling
- Built-in layout primitives (Join, Place, Align) for responsive design
- Handles ANSI color code generation - cross-platform compatibility
- Border, padding, margin utilities reduce boilerplate
- Already used in existing TUI components - consistency maintained

**Best Practices**:
- Define style constants upfront (colors, borders, spacing)
- Use `lipgloss.Width()` and `lipgloss.Height()` for accurate measurements
- Leverage `JoinHorizontal()` and `JoinVertical()` for complex layouts
- Cache styled strings when content doesn't change (performance optimization)
- Use `MaxWidth()` and `MaxHeight()` for responsive truncation

### 3. Reusable Components: Bubbles

**Decision**: Use `github.com/charmbracelet/bubbles` v0.21.0 for spinner, list, and viewport

**Rationale**:
- **Spinner**: Ideal for showing async operation progress (verify/apply in background)
- **List**: Built-in keyboard navigation, filtering, pagination - matches dashboard requirements
- **Viewport**: Scrollable content area for pipelines exceeding terminal height
- Pre-built, battle-tested components reduce implementation time
- Consistent UX with other Bubble Tea applications

**Component Selection**:
- **List** for main pipeline view (supports item selection, custom rendering)
- **Spinner** for loading states and operation progress
- **Viewport** fallback if List doesn't meet scrolling needs (unlikely)

**Best Practices**:
- Embed Bubbles components in custom model structs
- Forward relevant messages to component's Update() method
- Wrap component's View() output in custom styling
- Use `list.Item` interface for pipeline entries

### 4. Registry Persistence: JSON File Storage

**Decision**: Use `~/.streamy/registry.json` for pipeline metadata, `~/.streamy/status-cache.json` for status

**Rationale**:
- Simple, human-readable format for debugging
- No external database dependencies (Constitution Principle I)
- Standard library `encoding/json` sufficient (no third-party libs needed)
- File-based storage is fast enough for target scale (1-1000 pipelines)
- Atomic writes via temp file + rename pattern ensure consistency

**Schema**:
```json
// ~/.streamy/registry.json
{
  "version": "1.0",
  "pipelines": [
    {
      "id": "dev-env",
      "path": "/home/user/configs/dev.yaml",
      "description": "Development environment setup",
      "registered_at": "2025-10-08T12:00:00Z"
    }
  ]
}

// ~/.streamy/status-cache.json
{
  "version": "1.0",
  "statuses": {
    "dev-env": {
      "status": "satisfied",
      "last_run": "2025-10-08T14:30:00Z",
      "summary": "All 5 steps passed"
    }
  }
}
```

**Alternatives Considered**:
- **SQLite**: Overkill for simple key-value lookups. Adds binary dependency complexity.
- **YAML**: Less ergonomic for programmatic access. JSON marshaling is faster.
- **Embedded DB (BoltDB)**: Better for high-write workloads. Registry is read-heavy.

**Best Practices**:
- Validate JSON schema version on load (handle migration paths)
- Use file locks for concurrent access safety
- Atomic writes: write to `.tmp` file, then rename
- Graceful degradation if cache is corrupted (regenerate on next verify)

### 5. Async Operation Patterns

**Decision**: Use `tea.Cmd` with goroutines for verify/apply operations, communicate via custom messages

**Rationale**:
- Bubble Tea's architecture: Update() must return quickly, I/O in commands
- Goroutines allow non-blocking operations while UI remains responsive
- Custom message types (`VerifyCompleteMsg`, `ApplyCompleteMsg`) update state on completion
- Spinner component provides visual feedback during async work

**Pattern**:
```go
// Command constructor
func runVerifyCmd(pipelineID string) tea.Cmd {
    return func() tea.Msg {
        result := performVerify(pipelineID) // blocking call in goroutine
        return VerifyCompleteMsg{PipelineID: pipelineID, Result: result}
    }
}

// In Update()
case VerifyCompleteMsg:
    m.pipelines[msg.PipelineID].Status = msg.Result.Status
    m.loading = false
    return m, nil
```

**Best Practices**:
- Always return a message, even on errors (error messages update UI)
- Use `tea.Batch()` for parallel operations (refresh all pipelines)
- Avoid long-running work in Update() - move to commands
- Use context.Context for cancellable operations

### 6. State Management Strategy

**Decision**: Single centralized model with nested state for views (list vs. detail)

**Rationale**:
- Single source of truth simplifies reasoning about state
- View mode enum (`ViewMode` type) controls which view to render
- Navigation stack for detail view → main list transitions
- Immutable updates prevent race conditions

**Model Structure**:
```go
type DashboardModel struct {
    // Core data
    pipelines      []Pipeline
    registry       *registry.Registry
    statusCache    *registry.StatusCache
    
    // UI state
    viewMode       ViewMode // list, detail, help
    cursor         int
    selectedID     string
    
    // Component state
    list           list.Model
    spinner        spinner.Model
    
    // Operation state
    loading        map[string]bool // pipelineID -> isLoading
    errors         map[string]string
    
    // Dimensions
    width, height  int
}
```

**Best Practices**:
- Group related state fields together (data, UI, components, operations)
- Use maps for pipeline-specific state (loading, errors) - O(1) lookups
- Store terminal dimensions for responsive layout recalculation
- Separate concerns: model holds state, view renders it, update modifies it

## Integration Patterns

### 1. Verify/Apply Invocation

**Decision**: Reuse existing `cmd/streamy/verify.go` and `apply.go` logic via internal package extraction

**Approach**:
- Extract core verify logic from `cmd/streamy/verify.go` to `internal/engine/verify.go`
- Extract core apply logic from `cmd/streamy/apply.go` to `internal/engine/apply.go`
- Both CLI and dashboard call same internal functions
- Dashboard wraps calls in `tea.Cmd` for async execution

**Benefits**:
- No code duplication
- CLI and dashboard behavior stays synchronized
- Easier testing (internal package has no cobra dependencies)

**Pattern**:
```go
// internal/engine/verify.go
func VerifyPipeline(ctx context.Context, configPath string) (*VerifyResult, error) {
    // Existing verify logic
}

// internal/tui/dashboard/commands.go
func verifyCmd(configPath string) tea.Cmd {
    return func() tea.Msg {
        ctx := context.Background()
        result, err := engine.VerifyPipeline(ctx, configPath)
        if err != nil {
            return VerifyErrorMsg{Error: err}
        }
        return VerifyCompleteMsg{Result: result}
    }
}
```

### 2. Registry Integration

**Decision**: Create `internal/registry` package as abstraction layer over JSON files

**Responsibilities**:
- Load/save registry.json and status-cache.json
- CRUD operations for pipeline entries
- Thread-safe access via mutex (single-process safety)
- Cache invalidation and refresh logic

**API Design**:
```go
type Registry struct {
    path        string
    pipelines   []Pipeline
    mu          sync.RWMutex
}

func Load(path string) (*Registry, error)
func (r *Registry) List() []Pipeline
func (r *Registry) Get(id string) (Pipeline, error)
func (r *Registry) Add(p Pipeline) error
func (r *Registry) Remove(id string) error
func (r *Registry) Save() error
```

### 3. Existing TUI Component Reuse

**Decision**: Embed existing step execution TUI for detail view operations

**Approach**:
- When user triggers verify/apply from detail view, launch existing TUI in nested mode
- Existing `internal/tui/model.go` handles step-by-step progress display
- Dashboard temporarily yields control, resumes on completion
- Use `tea.WindowSizeMsg` to ensure nested TUI gets correct dimensions

**Benefit**: Reuse proven step execution visualization, maintain consistency

## Terminal Compatibility

### Color Support Detection

**Decision**: Detect terminal capabilities at startup, fallback to 16-color mode

**Implementation**:
- Use `lipgloss.ColorProfile()` to detect 24-bit, 256-color, or 16-color support
- Define color palettes for each mode
- Status icons work with basic colors (green, yellow, red, white)

**Graceful Degradation**:
- Unicode icons (🟢🟡🔴⚪) fallback to ASCII (`[OK]`, `[!!]`, `[XX]`, `[??]`)
- Box drawing characters fallback to ASCII art (e.g., `+---+` instead of `┌───┐`)
- Use `term.IsTerminal()` to detect non-TTY environments (log JSON instead)

### Minimum Terminal Size

**Decision**: 80x24 minimum, graceful degradation for smaller

**Handling**:
- Display warning message if terminal < 80 columns
- Truncate long descriptions with ellipsis
- Reduce padding/margins in narrow mode
- Hide help footer if height < 10 lines (show "press ? for help" only)

## Performance Optimizations

### 1. Lazy Status Loading

**Decision**: Load pipeline list immediately, status on-demand or background refresh

**Approach**:
- Initial load: show pipelines with "unknown" status (⚪)
- Background: dispatch async commands to verify each pipeline
- Update UI as results arrive (incremental refresh)
- Cache results for next dashboard session

**Benefit**: Fast startup, avoid blocking on slow verify operations

### 2. Parallel Status Refresh

**Decision**: Use `tea.Batch()` to verify multiple pipelines concurrently

**Implementation**:
```go
func refreshAllCmd(pipelines []Pipeline) tea.Cmd {
    cmds := make([]tea.Cmd, len(pipelines))
    for i, p := range pipelines {
        cmds[i] = verifyCmd(p.Path)
    }
    return tea.Batch(cmds...)
}
```

**Constraint**: Respect DAG dependencies if pipelines reference each other (future consideration)

### 3. View Rendering Optimization

**Decision**: Cache static strings, only re-render on state changes

**Approach**:
- Header (title, summary) cached until pipeline count changes
- Footer (key hints) is static, rendered once
- List items cached per pipeline, invalidated on status change
- Use `lipgloss.NewStyle().Render()` sparingly (profiling shows it's expensive)

**Measurement**: Profile with `go test -bench` on 100-pipeline dataset

## Error Handling Strategy

### User-Facing Errors

**Decision**: Display errors inline with context and suggested actions

**Pattern**:
- Configuration parse errors: show file path, line number, YAML snippet
- Verification failures: show failed step, error message, retry option
- Registry errors: show "Registry corrupted - press 'r' to rebuild"
- Permission errors: show "Cannot write to ~/.streamy/ - check permissions"

**UI Placement**: Red error banner at top of screen, dismissible with Esc

### Panic Recovery

**Decision**: Use `defer recover()` in Update() to catch panics, display error screen

**Implementation**:
```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    defer func() {
        if r := recover(); r != nil {
            log.Error().Interface("panic", r).Msg("Dashboard panic")
            // Show error screen with crash report option
        }
    }()
    // ... normal update logic
}
```

**Rationale**: Dashboard shouldn't crash Streamy binary - graceful degradation to CLI fallback

## Testing Strategy

### Unit Tests

**Scope**: Model logic, state transitions, message handling

**Approach**:
- Test Update() with synthetic messages
- Verify state transitions (list → detail → list)
- Test loading state changes on async operations
- Mock registry/cache I/O

**Example**:
```go
func TestNavigationDown(t *testing.T) {
    m := NewDashboardModel(testPipelines)
    m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
    assert.Equal(t, 1, m.cursor)
}
```

### Integration Tests

**Scope**: Full dashboard flow with real registry files

**Approach**:
- Use `testdata/registry/` fixtures
- Spawn dashboard in test mode (headless)
- Send key sequences, verify final state
- Test async operations complete correctly

### Manual Testing

**Scenarios**:
- Test on Linux, macOS, Windows terminals
- Verify resize handling (tmux, terminal window resize)
- Test with 1, 10, 100 pipelines (performance validation)
- Test with corrupted registry files (error handling)

## Open Questions & Future Research

### 1. Search/Filter for Large Pipeline Lists (Deferred)

**Status**: Out of scope for MVP (spec: "Out of Scope" section)

**Future Research**:
- Fuzzy search library (e.g., `github.com/sahilm/fuzzy`)
- Filter by status (show only drifted)
- Tag-based filtering (requires registry schema extension)

### 2. Real-Time Status Monitoring (Deferred)

**Status**: Out of scope for MVP (spec: "automatic background status updates")

**Future Research**:
- File watcher (fsnotify) for config changes
- Periodic background refresh (every N minutes)
- WebSocket/SSE for multi-user environments (far future)

### 3. Diff View Integration

**Status**: Mentioned in user input ("d: show diff") but not in spec

**Clarification Needed**: What should diff view show?
- Diff between current state and desired state?
- Diff between two verification runs?
- File-level diff of config changes?

**Recommendation**: Defer to future spec, focus on core verify/apply actions for MVP

## References

- Bubble Tea Documentation: https://github.com/charmbracelet/bubbletea
- Lipgloss Examples: https://github.com/charmbracelet/lipgloss/tree/master/examples
- Bubbles Components: https://github.com/charmbracelet/bubbles
- Go TUI Best Practices: https://charm.sh/blog/
- Streamy Constitution: `/.specify/memory/constitution.md`
- Feature Specification: `./spec.md`

## Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2025-10-08 | Use Bubble Tea (existing) | Already integrated, Elm Architecture fits needs |
| 2025-10-08 | JSON for registry persistence | Simple, no external deps, fast enough |
| 2025-10-08 | Extract verify/apply to internal/engine | Code reuse between CLI and dashboard |
| 2025-10-08 | Single model with view mode enum | Simpler than multiple models, single source of truth |
| 2025-10-08 | Lazy status loading with background refresh | Fast startup, responsive UI |
| 2025-10-08 | 80x24 minimum terminal size | Standard terminal default, graceful degradation |
| 2025-10-08 | Defer search/filter to future iteration | MVP focus on core navigation and actions |
