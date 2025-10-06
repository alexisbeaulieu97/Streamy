# Data Model

**Feature**: Extend Plugin Contract with Verify Lifecycle  
**Date**: October 4, 2025

## Overview

This document defines the core data structures and their relationships for the verification lifecycle feature. All entities support the verification workflow: status representation, result aggregation, and plugin contract extension.

---

## Core Entities

### 1. VerificationStatus

**Purpose**: Enumeration representing the state match level for a single step verification.

**Type**: String enumeration (Go: `type VerificationStatus string`)

**Values**:

| Value | Meaning | Apply Behavior |
|-------|---------|----------------|
| `satisfied` | Current state exactly matches expected configuration | Skip step (optimization) |
| `missing` | Required resource/file/configuration not found | Execute step |
| `drifted` | Partial match or unexpected difference detected | Execute step |
| `blocked` | Cannot verify due to dependency failure or error | Execute step (may fail) |
| `unknown` | Verification status cannot be determined | Execute step (safe default) |

**Constants** (Go implementation):
```go
const (
    StatusSatisfied VerificationStatus = "satisfied"
    StatusMissing   VerificationStatus = "missing"
    StatusDrifted   VerificationStatus = "drifted"
    StatusBlocked   VerificationStatus = "blocked"
    StatusUnknown   VerificationStatus = "unknown"
)
```

**Validation Rules**:
- Must be one of the five defined values
- Case-sensitive (lowercase canonical form)
- Immutable once assigned to a result

**Relationships**:
- Used by: `VerificationResult` (status field)
- Returned by: `Plugin.Verify()` method

---

### 2. VerificationResult

**Purpose**: Contains the outcome of verifying a single step, including status, explanation, optional diff, and timing metadata.

**Fields**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `StepID` | string | Yes | Unique identifier of the verified step |
| `Status` | VerificationStatus | Yes | One of five status values |
| `Message` | string | Yes | Human-readable summary (e.g., "file not found", "symlink matches") |
| `Details` | string | No | Full diff output for drifted status; empty for other statuses |
| `Error` | error | No | Non-nil for blocked status; contains underlying error |
| `Duration` | time.Duration | Yes | Time taken to perform verification |
| `Timestamp` | time.Time | Yes | When verification completed |

**Structure** (Go):
```go
type VerificationResult struct {
    StepID    string
    Status    VerificationStatus
    Message   string
    Details   string             // Unified diff for drifted status
    Error     error              // Populated for blocked status
    Duration  time.Duration
    Timestamp time.Time
}
```

**Validation Rules**:
- `StepID` must match a configured step ID
- `Status` must be valid VerificationStatus value
- `Message` must not be empty (always provide explanation)
- `Details` populated only when `Status == StatusDrifted`
- `Error` populated only when `Status == StatusBlocked`
- `Duration` must be non-negative
- `Timestamp` must not be zero value

**Lifecycle**:
1. Created by plugin's `Verify()` method
2. Aggregated by executor for reporting
3. Used by apply logic to determine skip eligibility

**Relationships**:
- Produced by: `Plugin.Verify()` method
- Consumed by: Executor, CLI reporter, TUI (future)
- Distinct from: `StepResult` (used by Apply operations)

**VerificationResult vs StepResult**:
- `VerificationResult`: Returned by `Plugin.Verify()` during read-only state inspection. Describes current state alignment without modification.
- `StepResult`: Returned by `Plugin.Apply()` during state modification. Describes what changes were made.
- Usage: Verification happens before apply; apply may skip steps based on verification status (future integration).

---

### 3. VerificationSummary

**Purpose**: Aggregates verification results across all steps for reporting and decision-making.

**Fields**:

| Field | Type | Description |
|-------|------|-------------|
| `TotalSteps` | int | Total number of steps verified |
| `Satisfied` | int | Count of steps with satisfied status |
| `Missing` | int | Count of steps with missing status |
| `Drifted` | int | Count of steps with drifted status |
| `Blocked` | int | Count of steps with blocked status |
| `Unknown` | int | Count of steps with unknown status |
| `Results` | []VerificationResult | Full results for all steps |
| `Duration` | time.Duration | Total verification time |

**Structure** (Go):
```go
type VerificationSummary struct {
    TotalSteps int
    Satisfied  int
    Missing    int
    Drifted    int
    Blocked    int
    Unknown    int
    Results    []VerificationResult
    Duration   time.Duration
}
```

**Derived Properties**:
```go
func (s *VerificationSummary) AllSatisfied() bool {
    return s.Satisfied == s.TotalSteps
}

func (s *VerificationSummary) NeedsApply() bool {
    return s.Missing + s.Drifted + s.Unknown + s.Blocked > 0
}

func (s *VerificationSummary) ExitCode() int {
    if s.AllSatisfied() {
        return 0
    }
    return 1
}
```

**Lifecycle**:
1. Initialized at verification start
2. Updated as each step completes verification
3. Finalized after all steps verified
4. Reported to user via CLI/TUI

**Relationships**:
- Contains: Multiple `VerificationResult` instances
- Used by: CLI reporter, JSON output formatter

---

### 4. Plugin (Extended Interface)

**Purpose**: Contract that all Streamy plugins must satisfy, now including verification capability.

**Extended Interface** (Go):
```go
type Plugin interface {
    // Existing methods
    Metadata() Metadata
    Schema() interface{}
    Check(ctx context.Context, step *config.Step) (bool, error)
    Apply(ctx context.Context, step *config.Step) (*model.StepResult, error)
    DryRun(ctx context.Context, step *config.Step) (*model.StepResult, error)
    
    // NEW: Verification method
    Verify(ctx context.Context, step *config.Step) (*VerificationResult, error)
}
```

**Verify Method Contract**:
- **Input**: `context.Context` (for cancellation/timeout), `*config.Step` (configuration)
- **Output**: `*VerificationResult` (status and details), `error` (only for unexpected failures)
- **Guarantees**:
  - MUST NOT modify any system state
  - MUST respect context cancellation/timeout
  - MUST return within configured timeout (default 30s)
  - MUST populate result with appropriate status and message
  - MUST generate diff in `Details` field when status is `drifted`
  - MUST set `Error` field when status is `blocked`

**Error Handling**:
- Unexpected errors (panic, infrastructure failure): return `nil` result + error
- Expected verification failures (missing resource, permission denied): return result with appropriate status + `nil` error

**Relationships**:
- Implemented by: All plugin types (package, symlink, template, command, etc.)
- Invoked by: Executor during verification phase
- Returns: `VerificationResult`

---

### 5. Step Configuration Extension

**Purpose**: Per-step configuration options for verification behavior.

**New Fields** (added to `config.Step`):

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `verify_timeout` | duration | 30s | Maximum time allowed for verification of this step |

**Example YAML**:
```yaml
- id: large-repo-check
  type: repo
  path: /opt/large-monorepo
  verify_timeout: 120s  # Override default 30s timeout
```

**Validation Rules**:
- `verify_timeout` must be positive duration
- Parse error if invalid duration format
- If unspecified, use global default (30s)

**Relationships**:
- Used by: Executor to set context deadline for `Verify()` call
- Specified in: YAML configuration

---

## Entity Relationships Diagram

```
┌─────────────────────┐
│  Plugin Interface   │
│  (extends existing) │
└──────────┬──────────┘
           │
           │ implements
           ▼
   ┌───────────────────┐
   │ Verify() method   │
   └────────┬──────────┘
            │
            │ returns
            ▼
   ┌────────────────────────┐
   │  VerificationResult    │
   │  ┌──────────────────┐  │
   │  │ StepID           │  │
   │  │ Status  ◄────────┼──┼──── VerificationStatus enum
   │  │ Message          │  │
   │  │ Details          │  │
   │  │ Error            │  │
   │  │ Duration         │  │
   │  │ Timestamp        │  │
   │  └──────────────────┘  │
   └────────┬───────────────┘
            │
            │ aggregated into
            ▼
   ┌───────────────────────┐
   │ VerificationSummary   │
   │  ┌─────────────────┐  │
   │  │ TotalSteps      │  │
   │  │ Satisfied       │  │
   │  │ Missing         │  │
   │  │ Drifted         │  │
   │  │ Blocked         │  │
   │  │ Unknown         │  │
   │  │ Results[]       │  │
   │  │ Duration        │  │
   │  └─────────────────┘  │
   └────────┬──────────────┘
            │
            │ reported via
            ▼
   ┌───────────────────┐
   │  CLI / TUI        │
   │  Output           │
   └───────────────────┘
```

---

## State Transitions

### Verification Status State Machine

Verification statuses are terminal (no transitions after assignment):

```
[Plugin Verification]
        │
        ├─► satisfied  (state matches)
        ├─► missing    (resource not found)
        ├─► drifted    (state differs)
        ├─► blocked    (error/dependency failure)
        └─► unknown    (cannot determine)
```

### Integration with Apply Workflow

```
[Verify Phase]
     │
     ├─► satisfied ──► SKIP (optimization)
     │
     ├─► missing   ──► APPLY
     │
     ├─► drifted   ──► APPLY
     │
     ├─► blocked   ──► APPLY (may fail)
     │
     └─► unknown   ──► APPLY (safe default)
```

---

## Serialization Formats

### JSON Output (for `--json` flag)

```json
{
  "summary": {
    "total_steps": 5,
    "satisfied": 2,
    "missing": 1,
    "drifted": 1,
    "blocked": 0,
    "unknown": 1,
    "duration_ms": 450
  },
  "results": [
    {
      "step_id": "install-git",
      "status": "satisfied",
      "message": "package git is installed (version 2.39.0)",
      "duration_ms": 120
    },
    {
      "step_id": "clone-repo",
      "status": "missing",
      "message": "repository not found at /opt/myrepo",
      "duration_ms": 50
    },
    {
      "step_id": "render-config",
      "status": "drifted",
      "message": "file content differs",
      "details": "--- expected\n+++ actual\n@@ -1,3 +1,3 @@\n APP_NAME=Streamy\n-ENVIRONMENT=production\n+ENVIRONMENT=development\n DEBUG_MODE=false\n",
      "duration_ms": 230
    },
    {
      "step_id": "run-setup",
      "status": "unknown",
      "message": "no verification command specified",
      "duration_ms": 10
    }
  ]
}
```

### Table Output (default CLI)

```
Step ID             Status      Message
──────────────────────────────────────────────────────────────
install-git         ✔ satisfied  package git is installed (version 2.39.0)
clone-repo          ✖ missing    repository not found at /opt/myrepo
render-config       ⚠ drifted    file content differs
run-setup           ? unknown    no verification command specified
──────────────────────────────────────────────────────────────
Summary: 5 steps checked — 2 satisfied, 1 missing, 1 drifted, 0 blocked, 1 unknown
```

---

## Validation Rules Summary

### VerificationStatus
- Must be one of five defined values
- Case-sensitive lowercase

### VerificationResult
- `StepID` must reference valid step
- `Status` must be valid enum value
- `Message` must not be empty
- `Details` only for drifted status
- `Error` only for blocked status
- `Duration` non-negative
- `Timestamp` not zero

### VerificationSummary
- Counts must sum to `TotalSteps`
- `Results` length equals `TotalSteps`
- `Duration` non-negative

### Plugin.Verify() Contract
- Must not modify state
- Must respect context cancellation
- Must return within timeout
- Must populate all required result fields

---

## Performance Characteristics

| Entity | Size (Typical) | Notes |
|--------|----------------|-------|
| VerificationStatus | 8-16 bytes | String enum, interned |
| VerificationResult | ~200 bytes | Without Details field |
| VerificationResult (with diff) | ~2-10 KB | Diff for 100-line file |
| VerificationSummary | O(n) results | n = number of steps |

**Memory Efficiency**: For 100-step config without diffs: ~20 KB total.

---

## Extensibility Considerations

### Future Enhancements
1. **Verification caching**: Store results for reuse across runs
2. **Partial verification**: Add filtering by step ID or tag
3. **Custom status extensions**: Plugin-specific sub-statuses (e.g., "partially satisfied")
4. **Verification hooks**: Pre/post-verification callbacks

### Backward Compatibility
- New status values can be added (enum extensibility)
- `VerificationResult` can add optional fields without breaking existing code
- Plugin interface extension requires all plugins update (acceptable pre-1.0)

---

**Document Status**: Complete  
**Next**: Generate contracts and quickstart documentation
