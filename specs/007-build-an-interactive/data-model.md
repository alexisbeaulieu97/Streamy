# Data Model: Interactive Dashboard

**Phase**: 1 (Design & Contracts)  
**Date**: October 8, 2025  
**Feature**: Interactive Dashboard for Pipeline Management

## Overview

This document defines the data structures, state management, and entity relationships for the interactive dashboard. All types are designed to support the functional requirements from the feature specification.

## Core Entities

### 1. Pipeline

Represents a registered pipeline configuration with runtime metadata.

```go
// Pipeline represents a registered Streamy pipeline
type Pipeline struct {
    // Identification
    ID          string    `json:"id"`           // Unique identifier (e.g., "dev-env")
    Name        string    `json:"name"`         // Human-readable name (for display)
    
    // Configuration
    Path        string    `json:"path"`         // Absolute path to YAML config file
    Description string    `json:"description"`  // User-provided description
    
    // Metadata
    RegisteredAt time.Time `json:"registered_at"` // When pipeline was registered
    
    // Runtime state (not persisted in registry)
    Status       PipelineStatus `json:"-"` // Current verification status
    LastRun      time.Time      `json:"-"` // Last verification/apply time
    LastResult   *ExecutionResult `json:"-"` // Cached result summary
}
```

**Validation Rules** (from FR-002, FR-003, FR-020):
- `ID` must be unique within registry
- `Path` must exist and be readable at registration time
- `Description` is optional, defaults to empty string
- `Status` defaults to `StatusUnknown` on initial load
- Missing config files detected and displayed appropriately

**State Transitions**:
- `StatusUnknown` → `StatusVerifying` → `StatusSatisfied | StatusDrifted | StatusFailed`
- `StatusSatisfied` can transition to `StatusApplying` → `StatusSatisfied | StatusFailed`

### 2. PipelineStatus

Enumeration of possible pipeline states.

```go
// PipelineStatus represents the verification state of a pipeline
type PipelineStatus string

const (
    StatusUnknown   PipelineStatus = "unknown"   // Not yet verified, or cache stale
    StatusSatisfied PipelineStatus = "satisfied" // All steps passed verification
    StatusDrifted   PipelineStatus = "drifted"   // Some steps need changes (non-critical)
    StatusFailed    PipelineStatus = "failed"    // Verification or apply failed
    StatusVerifying PipelineStatus = "verifying" // Verification in progress
    StatusApplying  PipelineStatus = "applying"  // Apply operation in progress
)

// Icon returns the Unicode icon for the status (FR-002)
func (s PipelineStatus) Icon() string {
    switch s {
    case StatusSatisfied:
        return "🟢"
    case StatusDrifted:
        return "🟡"
    case StatusFailed:
        return "🔴"
    default:
        return "⚪"
    }
}

// Color returns the Lipgloss color for the status
func (s PipelineStatus) Color() lipgloss.Color {
    switch s {
    case StatusSatisfied:
        return lipgloss.Color("42") // green
    case StatusDrifted:
        return lipgloss.Color("226") // yellow
    case StatusFailed:
        return lipgloss.Color("196") // red
    default:
        return lipgloss.Color("250") // light gray
    }
}
```

**Display Mapping** (FR-002):
- 🟢 `satisfied`: Green - all checks passed
- 🟡 `drifted`: Yellow - changes needed but not critical
- 🔴 `failed`: Red - errors during verification/apply
- ⚪ `unknown`: Gray - not yet checked or cache expired

### 3. ExecutionResult

Summary of a verification or apply operation.

```go
// ExecutionResult captures the outcome of a verify or apply operation
type ExecutionResult struct {
    // Identification
    PipelineID  string    `json:"pipeline_id"`
    Operation   string    `json:"operation"` // "verify" or "apply"
    
    // Outcome
    Status      PipelineStatus `json:"status"`
    Success     bool           `json:"success"`
    
    // Details
    StepResults []StepResult   `json:"step_results"` // Per-step outcomes
    Duration    time.Duration  `json:"duration"`
    CompletedAt time.Time      `json:"completed_at"`
    
    // Errors
    Error       *ErrorDetail   `json:"error,omitempty"` // Overall error if failed
}

// StepResult represents the outcome of a single step
type StepResult struct {
    StepID      string    `json:"step_id"`
    Status      string    `json:"status"` // "pending", "running", "success", "failed", "skipped"
    Message     string    `json:"message,omitempty"`
    Duration    time.Duration `json:"duration"`
    Error       *ErrorDetail  `json:"error,omitempty"`
}

// ErrorDetail provides structured error information (FR-016)
type ErrorDetail struct {
    Code       string   `json:"code"`        // Error category (e.g., "CONFIG_INVALID")
    Message    string   `json:"message"`     // Human-readable description
    Context    string   `json:"context"`     // Additional context (file, line, etc.)
    Suggestion string   `json:"suggestion"`  // Recommended fix
    Stacktrace []string `json:"stacktrace,omitempty"` // Debug info
}
```

**Relationships**:
- `ExecutionResult` has 0..N `StepResult` entries
- `StepResult` matches steps from pipeline configuration DAG
- `ErrorDetail` can appear in `ExecutionResult` or individual `StepResult`

## Dashboard State

### 4. DashboardModel

Main Bubble Tea model for dashboard state management.

```go
// DashboardModel is the Bubble Tea model for the dashboard UI
type DashboardModel struct {
    // ===== Core Data =====
    pipelines      []Pipeline            // All registered pipelines
    registry       *registry.Registry    // Registry I/O abstraction
    statusCache    *registry.StatusCache // Persistent status cache
    
    // ===== UI State =====
    viewMode       ViewMode              // Current view (list, detail, help)
    cursor         int                   // Selected index in list view
    selectedID     string                // ID of pipeline in detail view
    scrollOffset   int                   // Vertical scroll position
    
    // ===== Component State =====
    list           list.Model            // Bubbles list component
    spinner        spinner.Model         // Bubbles spinner component
    viewport       viewport.Model        // Scrollable detail content
    
    // ===== Operation State =====
    loading        map[string]bool       // pipelineID -> isLoading
    operations     map[string]Operation  // In-progress operations
    errors         map[string]string     // pipelineID -> error message
    showError      bool                  // Display error banner
    errorMsg       string                // Current error banner text
    
    // ===== Dimensions =====
    width          int                   // Terminal width
    height         int                   // Terminal height
    
    // ===== Configuration =====
    refreshInterval time.Duration        // Auto-refresh interval (0 = disabled)
    confirmations   bool                 // Require confirmations for destructive actions
}

// ViewMode determines which screen to render
type ViewMode int

const (
    ViewList   ViewMode = iota // Main pipeline list (default)
    ViewDetail                 // Single pipeline detail view
    ViewHelp                   // Keyboard shortcuts help overlay
    ViewConfirm                // Confirmation dialog (for apply)
)

// Operation tracks an in-progress async operation
type Operation struct {
    Type       OperationType // verify, apply, refresh
    PipelineID string
    StartedAt  time.Time
    Cmd        tea.Cmd
}

// OperationType categorizes async operations
type OperationType string

const (
    OpVerify OperationType = "verify"
    OpApply  OperationType = "apply"
    OpRefresh OperationType = "refresh"
)
```

**State Invariants**:
- `cursor` ∈ [0, len(pipelines)-1] when `len(pipelines) > 0`
- `selectedID` is valid Pipeline.ID when `viewMode == ViewDetail`
- `loading[id]` is true iff an operation is in progress for pipeline `id`
- `scrollOffset` + visible height ≤ content height

**Initialization** (from FR-001, FR-014):
1. Load registry from `~/.streamy/registry.json`
2. Load status cache from `~/.streamy/status-cache.json`
3. If registry empty, set `viewMode = ViewList` with empty state message
4. If registry non-empty, populate `pipelines`, set cursor to 0
5. Initialize Bubbles components (list, spinner, viewport)

### 5. Registry (Persistence Layer)

Abstraction for pipeline registry storage.

```go
// Registry manages the pipeline registry persistence
type Registry struct {
    path      string        // Path to registry.json
    mu        sync.RWMutex  // Thread-safe access
    version   string        // Schema version (for migrations)
    pipelines []Pipeline    // In-memory cache
}

// RegistryFile is the JSON file format
type RegistryFile struct {
    Version   string     `json:"version"`
    Pipelines []Pipeline `json:"pipelines"`
}

// Methods (from FR-017)
func NewRegistry(path string) (*Registry, error)
func (r *Registry) Load() error
func (r *Registry) Save() error
func (r *Registry) List() []Pipeline
func (r *Registry) Get(id string) (Pipeline, error)
func (r *Registry) Add(p Pipeline) error
func (r *Registry) Update(p Pipeline) error
func (r *Registry) Remove(id string) error
```

**File Location**: `~/.streamy/registry.json`

**Concurrency**: `sync.RWMutex` ensures thread-safety for single-process access

**Atomic Writes**: Save() writes to `.tmp` file, then renames (prevents corruption)

### 6. StatusCache (Performance Layer)

Caches pipeline status to avoid re-verification on every dashboard launch.

```go
// StatusCache persists pipeline status between sessions
type StatusCache struct {
    path     string
    mu       sync.RWMutex
    version  string
    statuses map[string]CachedStatus
}

// CachedStatus stores status metadata
type CachedStatus struct {
    Status      PipelineStatus   `json:"status"`
    LastRun     time.Time        `json:"last_run"`
    Summary     string           `json:"summary"`     // Brief result description
    StepCount   int              `json:"step_count"`  // Total steps in pipeline
    FailedSteps []string         `json:"failed_steps,omitempty"` // Failed step IDs
}

// StatusCacheFile is the JSON file format
type StatusCacheFile struct {
    Version  string                  `json:"version"`
    Statuses map[string]CachedStatus `json:"statuses"`
}

// Methods (from FR-017, FR-003)
func NewStatusCache(path string) (*StatusCache, error)
func (c *StatusCache) Load() error
func (c *StatusCache) Save() error
func (c *StatusCache) Get(pipelineID string) (CachedStatus, bool)
func (c *StatusCache) Set(pipelineID string, status CachedStatus) error
func (c *StatusCache) Invalidate(pipelineID string) error
```

**File Location**: `~/.streamy/status-cache.json`

**Invalidation**: Clear cache entry when:
- Config file modified (detected by mtime)
- User triggers manual refresh (FR-012)
- Verification or apply operation completes

**TTL**: No automatic expiration - cache is valid until explicitly invalidated

## Message Types (Bubble Tea)

### 7. Dashboard Messages

Custom message types for dashboard state updates.

```go
// ===== Navigation Messages =====

// PipelineSelectedMsg indicates a pipeline was selected in list view (FR-005)
type PipelineSelectedMsg struct {
    Pipeline Pipeline
}

// BackToListMsg indicates return to list from detail view (FR-011)
type BackToListMsg struct{}

// ===== Operation Messages =====

// VerifyStartedMsg indicates verification started (FR-007)
type VerifyStartedMsg struct {
    PipelineID string
}

// VerifyCompleteMsg carries verification results (FR-007, FR-010)
type VerifyCompleteMsg struct {
    PipelineID string
    Result     ExecutionResult
}

// VerifyErrorMsg reports verification failure (FR-016)
type VerifyErrorMsg struct {
    PipelineID string
    Error      ErrorDetail
}

// ApplyStartedMsg indicates apply started (FR-008)
type ApplyStartedMsg struct {
    PipelineID string
}

// ApplyCompleteMsg carries apply results (FR-008, FR-010)
type ApplyCompleteMsg struct {
    PipelineID string
    Result     ExecutionResult
}

// ApplyErrorMsg reports apply failure (FR-016)
type ApplyErrorMsg struct {
    PipelineID string
    Error      ErrorDetail
}

// RefreshStartedMsg indicates full refresh initiated (FR-012)
type RefreshStartedMsg struct {
    Count int // Number of pipelines being refreshed
}

// RefreshProgressMsg reports refresh progress
type RefreshProgressMsg struct {
    Completed int
    Total     int
}

// RefreshCompleteMsg indicates all pipelines refreshed
type RefreshCompleteMsg struct {
    Results map[string]ExecutionResult // pipelineID -> result
}

// ===== Confirmation Messages =====

// ConfirmActionMsg prompts user for confirmation (FR-008)
type ConfirmActionMsg struct {
    Action     string    // "apply", "delete", etc.
    PipelineID string
    Message    string    // Confirmation prompt text
}

// ConfirmResponseMsg carries user's confirmation response
type ConfirmResponseMsg struct {
    Confirmed bool
}

// ===== Error Messages =====

// ErrorMsg displays an error banner (FR-016)
type ErrorMsg struct {
    Error ErrorDetail
}

// ClearErrorMsg dismisses the error banner
type ClearErrorMsg struct{}
```

**Message Flow Examples**:

**Verify Operation**:
1. User presses 'v' → `Update()` returns `verifyCmd(pipelineID)`
2. `verifyCmd` dispatches `VerifyStartedMsg` → Update sets `loading[id] = true`
3. Verify runs in goroutine → completes → sends `VerifyCompleteMsg`
4. `Update()` receives `VerifyCompleteMsg` → updates `pipelines[id].Status`, sets `loading[id] = false`

**Apply with Confirmation**:
1. User presses 'a' → `Update()` returns `ConfirmActionMsg`
2. `Update()` switches to `ViewConfirm` mode, displays prompt
3. User presses 'y' → `Update()` receives `ConfirmResponseMsg{Confirmed: true}`
4. `Update()` returns `applyCmd(pipelineID)` → same flow as verify

## Data Flow Diagrams

### Startup Flow

```
main() 
  → LoadRegistry() 
  → LoadStatusCache() 
  → NewDashboardModel(pipelines, registry, cache)
  → tea.NewProgram(model).Run()
  → Init() returns background refresh commands
  → Update loop begins
```

### Verification Flow

```
User Input (key 'v')
  → Update() → verifyCmd(pipelineID)
  → goroutine: VerifyPipeline()
  → VerifyCompleteMsg
  → Update() updates model.pipelines[id].Status
  → SaveStatusCache()
  → View() re-renders with new status icon
```

### Refresh All Flow

```
User Input (key 'r')
  → Update() → refreshAllCmd(pipelineIDs)
  → tea.Batch([verifyCmd(id1), verifyCmd(id2), ...])
  → Multiple goroutines execute in parallel
  → Each sends VerifyCompleteMsg independently
  → Update() handles each completion incrementally
  → View() shows spinner with "Refreshing... 3/10"
  → All complete → View() shows updated list
```

## Validation & Business Rules

### Pipeline Sorting (FR-015)

```go
// SortPipelines orders pipelines by priority
func SortPipelines(pipelines []Pipeline) []Pipeline {
    // Priority: failed > drifted > satisfied > unknown
    sort.SliceStable(pipelines, func(i, j int) bool {
        return statusPriority(pipelines[i].Status) > statusPriority(pipelines[j].Status)
    })
    return pipelines
}

func statusPriority(s PipelineStatus) int {
    switch s {
    case StatusFailed:   return 3
    case StatusDrifted:  return 2
    case StatusSatisfied: return 1
    default:             return 0 // unknown
    }
}
```

### Human-Readable Timestamps (FR-003)

```go
// FormatLastRun converts timestamp to relative time
func FormatLastRun(t time.Time) string {
    if t.IsZero() {
        return "Never run"
    }
    
    elapsed := time.Since(t)
    
    switch {
    case elapsed < time.Minute:
        return "Just now"
    case elapsed < time.Hour:
        return fmt.Sprintf("%d minutes ago", int(elapsed.Minutes()))
    case elapsed < 24*time.Hour:
        return fmt.Sprintf("%d hours ago", int(elapsed.Hours()))
    case elapsed < 7*24*time.Hour:
        return fmt.Sprintf("%d days ago", int(elapsed.Hours()/24))
    default:
        return t.Format("Jan 2, 2006")
    }
}
```

### Terminal Size Constraints (research.md: 80x24 minimum)

```go
// ValidateTerminalSize checks if terminal meets minimum requirements
func ValidateTerminalSize(width, height int) error {
    if width < 80 {
        return fmt.Errorf("terminal too narrow (minimum 80 columns, got %d)", width)
    }
    if height < 10 {
        return fmt.Errorf("terminal too short (minimum 10 lines, got %d)", height)
    }
    return nil
}
```

## Persistence Schema

### registry.json

```json
{
  "version": "1.0",
  "pipelines": [
    {
      "id": "dev-env",
      "name": "Development Environment",
      "path": "/home/user/.config/streamy/dev.yaml",
      "description": "Sets up local dev tools and configs",
      "registered_at": "2025-10-08T10:00:00Z"
    },
    {
      "id": "prod-deploy",
      "name": "Production Deployment",
      "path": "/home/user/.config/streamy/prod.yaml",
      "description": "Deploys app to production servers",
      "registered_at": "2025-10-08T11:30:00Z"
    }
  ]
}
```

### status-cache.json

```json
{
  "version": "1.0",
  "statuses": {
    "dev-env": {
      "status": "satisfied",
      "last_run": "2025-10-08T14:25:00Z",
      "summary": "All 8 steps passed",
      "step_count": 8,
      "failed_steps": []
    },
    "prod-deploy": {
      "status": "failed",
      "last_run": "2025-10-08T13:10:00Z",
      "summary": "Failed at step 3/12",
      "step_count": 12,
      "failed_steps": ["deploy-backend", "verify-health"]
    }
  }
}
```

## Migration Strategy

### Schema Version Handling

```go
// LoadWithMigration loads registry and migrates if needed
func LoadWithMigration(path string) (*Registry, error) {
    file := RegistryFile{}
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    
    if err := json.Unmarshal(data, &file); err != nil {
        return nil, err
    }
    
    // Check version and migrate
    switch file.Version {
    case "1.0":
        // Current version, no migration needed
    case "":
        // Legacy format (pre-versioning), migrate to 1.0
        file = migrateV0toV1(file)
    default:
        return nil, fmt.Errorf("unsupported registry version: %s", file.Version)
    }
    
    return &Registry{path: path, version: "1.0", pipelines: file.Pipelines}, nil
}
```

**Versioning Policy** (Constitution: Schema Evolution Rules):
- **Additive changes** (new fields): Keep version, make fields optional
- **Deprecations**: Warn on load, support for 1 major version
- **Breaking changes**: Bump version, provide migration function

## Performance Considerations

### Memory Footprint

**Target**: <50MB resident with 100 pipelines

**Breakdown**:
- 100 pipelines × ~500 bytes/pipeline = 50KB
- Status cache (100 entries × 200 bytes) = 20KB
- Execution results (cached for current view) = ~1-5MB
- Bubble Tea framework overhead = ~5-10MB
- Go runtime overhead = ~10-20MB
- **Total**: ~20-35MB (within target)

### Disk I/O

**Registry Load**: Read once on startup, ~10-100KB file → <1ms

**Status Cache Load**: Read once on startup, ~20-200KB file → <1ms

**Cache Save**: Write on every status change, atomic rename → <5ms

**Optimization**: Debounce cache writes (max 1 write/second during batch operations)

### Rendering Performance

**Target**: 60fps navigation (< 16ms frame time)

**Optimizations**:
- Cache static strings (header, footer, help text)
- Only re-render changed list items
- Use `lipgloss.NewStyle().Inline(true)` for non-layout styles
- Avoid expensive regexp or string operations in View()

**Profiling**: Use `go test -bench -cpuprofile` on 100-pipeline dataset

## Testing Data Fixtures

### testdata/registry/empty.json

```json
{"version": "1.0", "pipelines": []}
```

### testdata/registry/single-pipeline.json

```json
{
  "version": "1.0",
  "pipelines": [
    {
      "id": "test-pipeline",
      "name": "Test Pipeline",
      "path": "/tmp/test.yaml",
      "description": "Test description",
      "registered_at": "2025-01-01T00:00:00Z"
    }
  ]
}
```

### testdata/registry/multiple-pipelines.json

```json
{
  "version": "1.0",
  "pipelines": [
    {"id": "satisfied", "name": "Satisfied", "path": "/tmp/satisfied.yaml", "description": "Green", "registered_at": "2025-01-01T00:00:00Z"},
    {"id": "drifted", "name": "Drifted", "path": "/tmp/drifted.yaml", "description": "Yellow", "registered_at": "2025-01-02T00:00:00Z"},
    {"id": "failed", "name": "Failed", "path": "/tmp/failed.yaml", "description": "Red", "registered_at": "2025-01-03T00:00:00Z"},
    {"id": "unknown", "name": "Unknown", "path": "/tmp/unknown.yaml", "description": "Gray", "registered_at": "2025-01-04T00:00:00Z"}
  ]
}
```

## Summary

This data model supports all functional requirements from the specification:
- **FR-002**: Pipeline status with color-coded icons
- **FR-003**: Last run time with human-readable format
- **FR-007, FR-008**: Verify and apply operations with progress tracking
- **FR-009, FR-010**: Real-time progress and status updates
- **FR-012**: Refresh all statuses
- **FR-015**: Priority sorting (failed → drifted → satisfied → unknown)
- **FR-016**: Structured error messages with context
- **FR-017**: Persistent status cache across sessions
- **FR-018**: Scrolling support via viewport
- **FR-020**: Graceful handling of missing config files

The model follows Bubble Tea's Elm Architecture pattern with clear separation of concerns: core data entities (Pipeline, ExecutionResult), UI state (DashboardModel, ViewMode), and message-driven updates (async operations via tea.Cmd).
