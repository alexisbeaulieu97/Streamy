# Contract: Dashboard Model Interface

**Phase**: 1 (Design & Contracts)  
**Date**: October 8, 2025  
**Purpose**: Define the public interface and behavior contracts for the DashboardModel

## Overview

The `DashboardModel` is the core state container for the dashboard TUI. It implements the Bubble Tea `tea.Model` interface and manages all dashboard state, including pipelines, UI state, and async operations.

## Interface Contract

### Required Bubble Tea Interface

```go
type Model interface {
    Init() tea.Cmd
    Update(tea.Msg) (tea.Model, tea.Cmd)
    View() string
}
```

### DashboardModel Implementation

```go
// DashboardModel implements tea.Model for the dashboard
type DashboardModel struct {
    // (fields from data-model.md)
}

// Init initializes the dashboard and returns initial commands
func (m DashboardModel) Init() tea.Cmd

// Update processes messages and returns updated model + commands
func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd)

// View renders the current state as a string
func (m DashboardModel) View() string
```

## Constructor Contract

### NewDashboardModel

**Signature**:
```go
func NewDashboardModel(
    pipelines []Pipeline,
    registry *registry.Registry,
    cache *registry.StatusCache,
) DashboardModel
```

**Preconditions**:
- `pipelines` may be empty (triggers empty state display per FR-014)
- `registry` must not be nil
- `cache` must not be nil (may be empty cache if none exists)

**Postconditions**:
- Model is fully initialized with default state
- `viewMode` is `ViewList`
- `cursor` is 0 (if pipelines non-empty)
- All maps are initialized (non-nil)
- Bubble Tea components (list, spinner) are initialized
- Terminal dimensions are set to defaults (will be updated by first WindowSizeMsg)

**Invariants Established**:
- `cursor ∈ [0, len(pipelines)-1]` if `len(pipelines) > 0`
- All pipelines have initial status from cache or `StatusUnknown`
- `loading` map is empty (no operations in progress)
- `errors` map is empty

## Init() Contract

**Purpose**: Return initial commands to execute on dashboard startup

**Signature**:
```go
func (m DashboardModel) Init() tea.Cmd
```

**Behavior**:
1. If pipelines exist and have unknown status, dispatch background refresh
2. Return spinner tick command if any pipelines are loading
3. Return nil if no initial actions needed

**Returns**:
```go
// Option 1: Background status refresh
tea.Batch(
    m.spinner.Tick,
    loadInitialStatusCmd(m.pipelines),
)

// Option 2: No initial work
nil
```

**Functional Requirements**: Supports FR-003 (display status on load)

## Update() Contract

### Signature

```go
func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd)
```

### Behavior Contract

**Preconditions**:
- `msg` is a valid Bubble Tea message (any type)
- Model is in valid state (invariants hold)

**Postconditions**:
- Returned model has all invariants maintained
- Returned model is a copy (original unchanged per Elm Architecture)
- Returned `tea.Cmd` will eventually send a message back to Update (or nil)

**Invariants Preserved**:
- `cursor` stays in valid range after navigation
- `viewMode` transitions are valid (see state machine below)
- `loading[id]` is true iff an operation is active for pipeline `id`
- `selectedID` is valid Pipeline.ID when `viewMode == ViewDetail`

### View Mode State Machine

```
ViewList ──┬──> ViewDetail (PipelineSelectedMsg)
           │
           └──> ViewHelp (KeyMsg: '?')
           │
           └──> ViewConfirm (ConfirmActionMsg)

ViewDetail ──┬──> ViewList (BackToListMsg)
             │
             └──> ViewConfirm (ConfirmActionMsg for apply)

ViewHelp ──> ViewList (KeyMsg: '?' or Esc)

ViewConfirm ──> ViewDetail (ConfirmResponseMsg)
```

**Illegal Transitions**: None defined. All transitions are explicit via messages.

### Message Handling Contract

Each message type MUST be handled according to its contract in `messages.md`. Key behaviors:

#### Navigation Messages

- **PipelineSelectedMsg**: Transition to `ViewDetail`, load detailed status
- **BackToListMsg**: Transition to `ViewList`, clear detail state
- **KeyMsg (Up/Down)**: Update cursor, maintain bounds [0, len(pipelines)-1]

#### Operation Messages

- **VerifyStartedMsg**: Set `loading[id] = true`, start spinner
- **VerifyCompleteMsg**: Update pipeline status, clear loading, save cache
- **VerifyErrorMsg**: Update status to failed, set error message, clear loading
- (Same pattern for Apply messages)

#### Refresh Messages

- **RefreshStartedMsg**: Set refresh state, dispatch parallel verify commands
- **RefreshProgressMsg**: Update progress counter
- **RefreshCompleteMsg**: Update all statuses, clear refresh state

#### System Messages

- **WindowSizeMsg**: Update `width`, `height`, recalculate layout

### Error Handling in Update()

**Rule**: Update() MUST NOT panic. All errors are converted to ErrorMsg.

```go
func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    defer func() {
        if r := recover(); r != nil {
            log.Error().Interface("panic", r).Msg("Dashboard panic in Update")
            // Return model with error state
            m.showError = true
            m.errorMsg = "Internal error occurred"
        }
    }()
    
    // ... handle messages
}
```

## View() Contract

**Purpose**: Render the current model state as a string for terminal display

**Signature**:
```go
func (m DashboardModel) View() string
```

**Behavior Contract**:

**Preconditions**:
- Model is in valid state
- Terminal dimensions (`m.width`, `m.height`) are set

**Postconditions**:
- Returns a string that fits within terminal dimensions
- String contains ANSI escape codes for colors/styling
- String ends with newline

**Rendering Rules**:
1. **View Mode Dispatch**: Render based on `m.viewMode`
   - `ViewList` → render pipeline list
   - `ViewDetail` → render pipeline detail view
   - `ViewHelp` → render help overlay
   - `ViewConfirm` → render confirmation dialog

2. **Layout Structure** (ViewList):
   ```
   ┌─────────────────────────────────────┐
   │ Header (title, summary, status)     │
   ├─────────────────────────────────────┤
   │                                     │
   │ Body (pipeline list, scrollable)    │
   │                                     │
   ├─────────────────────────────────────┤
   │ Footer (key hints)                  │
   └─────────────────────────────────────┘
   ```

3. **Error Banner**: If `m.showError == true`, render error at top:
   ```
   ┌─────────────────────────────────────┐
   │ ❌ ERROR: message here              │ (red background)
   ├─────────────────────────────────────┤
   │ [rest of normal view]               │
   ```

4. **Loading Indicators**: Pipelines with `loading[id] == true` show spinner

5. **Status Icons**: Use emoji or ASCII fallback based on terminal capabilities

**Performance Requirements**:
- View() should complete in <16ms (60fps target)
- Cache static strings (header, footer) when possible
- Only re-render changed sections (use Lipgloss efficiently)

**Functional Requirements**: Supports all display requirements (FR-002, FR-003, FR-006, FR-009, FR-014, FR-016, FR-019)

## Helper Methods Contract

### Navigation Helpers

```go
// MoveUp moves cursor up one position (FR-004)
func (m *DashboardModel) MoveUp()

// MoveDown moves cursor down one position (FR-004)
func (m *DashboardModel) MoveDown()

// SelectPipeline selects the pipeline at cursor position (FR-005)
func (m *DashboardModel) SelectPipeline() (Pipeline, error)
```

**Postconditions**: Cursor stays in valid range [0, len(pipelines)-1]

### Status Helpers

```go
// GetPipelineStatus returns the current status for a pipeline
func (m DashboardModel) GetPipelineStatus(id string) (PipelineStatus, error)

// SetPipelineStatus updates a pipeline's status (internal use)
func (m *DashboardModel) SetPipelineStatus(id string, status PipelineStatus) error

// IsLoading checks if a pipeline has an operation in progress
func (m DashboardModel) IsLoading(id string) bool
```

### Sorting Helpers

```go
// SortPipelines sorts pipelines by priority (failed > drifted > satisfied > unknown) (FR-015)
func (m *DashboardModel) SortPipelines()
```

**Postcondition**: Pipelines are sorted in priority order, cursor adjusted to maintain selection

## Invariants

The model MUST maintain these invariants at all times:

### Cursor Invariants

```go
// When pipelines exist
if len(m.pipelines) > 0 {
    assert(m.cursor >= 0)
    assert(m.cursor < len(m.pipelines))
}

// When no pipelines
if len(m.pipelines) == 0 {
    assert(m.cursor == 0)
}
```

### View Mode Invariants

```go
// Detail view requires selected ID
if m.viewMode == ViewDetail {
    assert(m.selectedID != "")
    assert(pipelineExists(m.selectedID))
}

// List view clears selected ID
if m.viewMode == ViewList {
    assert(m.selectedID == "")
}
```

### Loading State Invariants

```go
// Loading implies operation exists
if m.loading[id] {
    assert(m.operations[id] != nil)
}

// Operation exists implies loading
if m.operations[id] != nil {
    assert(m.loading[id] == true)
}
```

### Pipeline Status Invariants

```go
// Status transitions are valid
// Can only transition:
//   Unknown → Verifying → (Satisfied | Drifted | Failed)
//   Satisfied → Applying → (Satisfied | Failed)

// Loading states are temporary
if p.Status == StatusVerifying || p.Status == StatusApplying {
    assert(m.loading[p.ID] == true)
}
```

### Dimension Invariants

```go
// Terminal dimensions are positive
assert(m.width > 0)
assert(m.height > 0)

// Component dimensions fit within terminal
assert(m.list.Width <= m.width)
assert(m.list.Height <= m.height)
```

## Testing Contract

### Unit Test Requirements

1. **Constructor Test**:
   ```go
   func TestNewDashboardModel(t *testing.T) {
       m := NewDashboardModel(testPipelines, testRegistry, testCache)
       assert.Equal(t, ViewList, m.viewMode)
       assert.Equal(t, 0, m.cursor)
       assert.NotNil(t, m.loading)
   }
   ```

2. **Navigation Test**:
   ```go
   func TestNavigationBounds(t *testing.T) {
       m := NewDashboardModel(testPipelines, testRegistry, testCache)
       // Move down beyond bounds
       for i := 0; i < len(testPipelines)+5; i++ {
           m.MoveDown()
       }
       assert.Equal(t, len(testPipelines)-1, m.cursor)
   }
   ```

3. **Message Handling Test** (for each message type):
   ```go
   func TestVerifyCompleteMsg(t *testing.T) {
       // Test from messages.md contract
   }
   ```

4. **Invariant Test** (after every Update call):
   ```go
   func assertInvariants(t *testing.T, m DashboardModel) {
       // Check all invariants hold
       if len(m.pipelines) > 0 {
           assert.True(t, m.cursor >= 0 && m.cursor < len(m.pipelines))
       }
       // ... check other invariants
   }
   ```

### Integration Test Requirements

1. **Full Workflow Test**:
   ```go
   func TestVerifyApplyWorkflow(t *testing.T) {
       // Test: navigate → select → verify → apply → back to list
   }
   ```

2. **Async Operation Test**:
   ```go
   func TestAsyncVerifyCompletes(t *testing.T) {
       // Test: trigger verify → wait for completion → check status updated
   }
   ```

3. **Error Handling Test**:
   ```go
   func TestVerifyError(t *testing.T) {
       // Test: trigger verify on invalid config → check error displayed
   }
   ```

## Thread Safety

**Important**: DashboardModel is NOT thread-safe by design (Bubble Tea single-threaded model).

**Rules**:
- Only Update() method modifies model state
- Async operations (tea.Cmd goroutines) MUST NOT mutate model directly
- Communication is one-way: goroutines → messages → Update()
- Registry and StatusCache handle their own thread safety internally

## Performance Considerations

### View() Optimization

```go
// Cache static strings
var (
    headerCache string
    headerCacheDirty bool
)

func (m DashboardModel) View() string {
    if headerCacheDirty || headerCache == "" {
        headerCache = renderHeader(m)
        headerCacheDirty = false
    }
    
    // Use cached header
    body := renderBody(m)
    footer := renderFooter() // Always static
    
    return lipgloss.JoinVertical(lipgloss.Left, headerCache, body, footer)
}
```

### Update() Optimization

- **Fast Path**: Common messages (key navigation) return quickly
- **Batch Commands**: Use `tea.Batch()` for multiple operations
- **Avoid Allocations**: Reuse slices where possible

### Profiling Requirements

- Profile View() with `go test -bench` on 100-pipeline dataset
- Ensure no O(n²) operations in Update()
- Monitor memory allocations (should be constant after initialization)

## Summary

The DashboardModel interface contract ensures:
1. **Predictability**: All state transitions are explicit via messages
2. **Testability**: Pure functions, no hidden side effects
3. **Correctness**: Invariants maintained at all times
4. **Performance**: View() renders in <16ms, Update() returns immediately
5. **Safety**: No panics, all errors converted to messages

This contract is the foundation for implementing a robust, maintainable dashboard TUI that integrates seamlessly with Bubble Tea's architecture.
