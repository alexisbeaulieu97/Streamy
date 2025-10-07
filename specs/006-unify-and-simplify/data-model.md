# Data Model: Unified Plugin Interface

**Date**: October 7, 2025  
**Feature**: Plugin System Refactoring

## Core Entities

### 1. Plugin Interface

**Purpose**: The contract all plugins must satisfy to participate in Streamy's execution engine.

**Methods**:
- `Metadata() PluginMetadata` - Returns plugin identity and capabilities
- `Schema() interface{}` - Returns a struct defining the plugin's YAML configuration schema
- `Evaluate(ctx context.Context, step *config.Step) (*model.EvaluationResult, error)` - **Read-only** state assessment
- `Apply(ctx context.Context, evalResult *model.EvaluationResult, step *config.Step) (*model.StepResult, error)` - State mutation

**Validation Rules**:
- Metadata() must return consistent values across calls
- Schema() must return a struct type suitable for JSON schema generation
- Evaluate() **MUST NOT** modify any system state
- Apply() must be idempotent (safe to call multiple times with same input)

**Relationships**:
- Registered with `PluginRegistry` during application startup
- Invoked by `Executor` during DAG traversal
- Returns `EvaluationResult` consumed by engine decision logic

---

### 2. EvaluationResult

**Purpose**: Rich data structure transferred from plugin to engine containing state assessment.

**Fields**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `StepID` | `string` | Yes | Unique identifier of the evaluated step |
| `CurrentState` | `VerificationStatus` | Yes | Enum: `Satisfied`, `Missing`, `Drifted`, `Blocked`, `Unknown` |
| `RequiresAction` | `bool` | Yes | `true` if Apply() should be called; `false` if state matches desired |
| `Message` | `string` | Yes | Human-readable summary (e.g., "File content differs") |
| `Diff` | `string` | No | Optional formatted diff (unified diff, JSON diff, etc.) |
| `InternalData` | `interface{}` | No | Opaque data passed from Evaluate() to Apply() to avoid recomputation |

**Validation Rules**:
- `StepID` must match the step being evaluated
- `RequiresAction` must be `true` when `CurrentState` is `Missing` or `Drifted`
- `RequiresAction` must be `false` when `CurrentState` is `Satisfied`
- `Message` must not be empty
- `Diff` should be populated when `RequiresAction` is `true` and changes are previewable

**State Transitions**:
- `Unknown` → (Evaluate) → `Satisfied` | `Missing` | `Drifted` | `Blocked`
- `Missing` | `Drifted` → (Apply) → `Satisfied`
- `Satisfied` → (no action) → `Satisfied`
- `Blocked` → (no action) → `Blocked` (requires external resolution)

**Relationships**:
- Created by plugin's `Evaluate()` method
- Consumed by `Executor` to decide whether to call `Apply()`
- Passed to plugin's `Apply()` method (includes `InternalData` to avoid redundant work)

**Example**:
```go
&model.EvaluationResult{
    StepID:         "install-vim",
    CurrentState:   model.StatusMissing,
    RequiresAction: true,
    Message:        "Package vim is not installed",
    Diff:           "Would install: vim (version 8.2.3995)",
    InternalData:   nil,
}
```

---

### 3. PluginError Hierarchy

**Purpose**: Structured error categorization for engine decision-making and user feedback.

#### Base Interface: PluginError

```go
type PluginError interface {
    error
    StepID() string
    Unwrap() error
}
```

**Validation Rules**:
- `StepID()` must return the step identifier where the error occurred
- `Unwrap()` must return the underlying error (may be nil)
- Error message (from `Error()`) must be human-readable and actionable

#### Error Types

##### ValidationError

**Purpose**: Configuration or input validation failures.

**Fields**:
- `ID` (string): Step identifier
- `Err` (error): Underlying validation error

**When to Use**:
- YAML configuration is malformed
- Required fields are missing
- Field values fail validation rules (e.g., invalid path, malformed regex)

**Example**:
```go
&ValidationError{
    ID:  "configure-symlink",
    Err: fmt.Errorf("source path must be absolute, got: './relative/path'"),
}
```

##### ExecutionError

**Purpose**: External operation failures during state assessment or application.

**Fields**:
- `ID` (string): Step identifier
- `Err` (error): Underlying execution error

**When to Use**:
- Shell command execution fails
- File I/O errors (read/write/permissions)
- Network operations fail (git clone, package download)
- External tool not found or inaccessible

**Example**:
```go
&ExecutionError{
    ID:  "clone-repo",
    Err: fmt.Errorf("git clone failed: exit code 128 (authentication required)"),
}
```

##### StateError

**Purpose**: Unable to determine current system state.

**Fields**:
- `ID` (string): Step identifier
- `Err` (error): Underlying state detection error

**When to Use**:
- Cannot read current file permissions
- Package manager returns ambiguous status
- System state is inconsistent or corrupted

**Example**:
```go
&StateError{
    ID:  "check-package",
    Err: fmt.Errorf("dpkg database is locked by another process"),
}
```

**Relationships**:
- Returned by plugin's `Evaluate()` or `Apply()` methods
- Caught by `Executor` for categorized error handling
- Logged with appropriate severity based on error type
- Displayed to user with context-specific guidance

---

### 4. VerificationStatus

**Purpose**: Enum representing the current state of a resource relative to desired state.

**Values**:

| Value | Meaning | RequiresAction |
|-------|---------|----------------|
| `Satisfied` | Current state matches desired state exactly | `false` |
| `Missing` | Resource does not exist and should be created | `true` |
| `Drifted` | Resource exists but differs from desired state | `true` |
| `Blocked` | Cannot proceed due to external condition (e.g., dependency failed) | `false` |
| `Unknown` | Unable to determine state (typically an error condition) | `false` |

**Validation Rules**:
- Only one status per evaluation result
- Status must accurately reflect state comparison
- `Satisfied` implies idempotency (running again would be no-op)

**State Machine**:
```
Initial: Unknown
  ↓ (Evaluate)
  ├─→ Satisfied (no action)
  ├─→ Missing (Apply creates)
  ├─→ Drifted (Apply updates)
  ├─→ Blocked (external resolution)
  └─→ Unknown (error condition)
```

---

### 5. StepResult (Existing, Unchanged)

**Purpose**: Result of executing a step's Apply operation.

**Fields**:
- `StepID` (string): Identifier of the step
- `Status` (string): Outcome status (`Success`, `Skipped`, `Failed`, `WouldCreate`, `WouldUpdate`)
- `Message` (string): Human-readable description
- `Error` (error): Error if status is `Failed`

**Relationship to New Interface**:
- `Apply()` returns `*StepResult` (unchanged from current interface)
- Engine sets `Skipped` status when `EvaluationResult.RequiresAction` is `false`
- Engine calls `Apply()` only when `RequiresAction` is `true`

---

## Entity Relationship Diagram

```
┌─────────────────┐
│     Executor    │
│   (in engine)   │
└────────┬────────┘
         │ calls
         ↓
┌─────────────────┐
│     Plugin      │◄──────────┐
│   (interface)   │            │
└────────┬────────┘            │
         │                     │
         │ Evaluate()          │ registered with
         ├──────────→┌────────────────────┐
         │           │ EvaluationResult   │
         │           ├────────────────────┤
         │           │ StepID             │
         │           │ CurrentState       │───→ VerificationStatus
         │           │ RequiresAction     │     (enum)
         │           │ Message            │
         │           │ Diff               │
         │           │ InternalData       │
         │           └────────────────────┘
         │                     │
         │                     │ passed to
         │ Apply()             │
         ├──────────→┌────────────────────┐
         │           │   StepResult       │
         │           ├────────────────────┤
         │           │ StepID             │
         │           │ Status             │
         │           │ Message            │
         │           │ Error              │
         │           └────────────────────┘
         │                     │
         │ on error            │
         └──────────→┌────────────────────┐
                     │   PluginError      │
                     │   (interface)      │
                     └──────┬─────────────┘
                            │
                ┌───────────┼───────────┐
                │           │           │
         ┌──────▼──────┐ ┌─▼──────────┐ ┌──▼──────────┐
         │Validation   │ │Execution   │ │State        │
         │Error        │ │Error       │ │Error        │
         └─────────────┘ └────────────┘ └─────────────┘
```

---

## Implementation Notes

### Type Definitions (Go)

```go
// internal/plugin/interface.go
type Plugin interface {
    Metadata() PluginMetadata
    Schema() interface{}
    Evaluate(ctx context.Context, step *config.Step) (*model.EvaluationResult, error)
    Apply(ctx context.Context, evalResult *model.EvaluationResult, step *config.Step) (*model.StepResult, error)
}

// internal/model/evaluation_result.go
type EvaluationResult struct {
    StepID         string
    CurrentState   VerificationStatus
    RequiresAction bool
    Message        string
    Diff           string
    InternalData   interface{}
}

type VerificationStatus string

const (
    StatusSatisfied VerificationStatus = "satisfied"
    StatusMissing   VerificationStatus = "missing"
    StatusDrifted   VerificationStatus = "drifted"
    StatusBlocked   VerificationStatus = "blocked"
    StatusUnknown   VerificationStatus = "unknown"
)

// internal/plugin/errors.go (or pkg/errors/errors.go)
type PluginError interface {
    error
    StepID() string
    Unwrap() error
}

type ValidationError struct {
    ID  string
    Err error
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error in step %s: %v", e.ID, e.Err)
}

func (e *ValidationError) StepID() string { return e.ID }
func (e *ValidationError) Unwrap() error  { return e.Err }

// Similar implementations for ExecutionError and StateError
```

### Migration Considerations

- Existing `model.StepResult` and `model.VerificationResult` remain unchanged
- `VerificationStatus` enum reuses existing status constants where possible
- `PluginMetadata` (from existing codebase) remains unchanged
- Error types extend existing `pkg/errors` patterns

---

## Validation & Testing Strategy

### Entity Validation
- Unit tests for each error type (Error(), StepID(), Unwrap() methods)
- Table-driven tests for VerificationStatus transitions
- Validation that EvaluationResult constraints are enforced

### Integration Testing
- Verify Evaluate() returns EvaluationResult with correct fields
- Verify Apply() receives and uses InternalData correctly
- Verify error types are returned in appropriate scenarios
- Verify read-only contract: Evaluate() does not mutate state

### Contract Testing
Each plugin must pass a standard contract test suite:
1. Evaluate() is idempotent (multiple calls return same result)
2. Evaluate() is read-only (no filesystem/state changes)
3. Apply() is idempotent (multiple calls safe)
4. Error cases return appropriate error types
5. EvaluationResult.Diff is populated when RequiresAction is true
