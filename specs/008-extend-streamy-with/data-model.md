# Data Model: Registry Management CLI Commands

**Phase**: 1 - Design & Contracts  
**Date**: October 9, 2025  
**Status**: Complete

## Overview

This document defines the data structures and relationships for the registry management CLI feature. All data structures already exist in the codebase—this document serves as reference documentation showing how they're used by the new commands.

## Core Entities

### 1. Pipeline

**Location**: `internal/registry/types.go`  
**Purpose**: Represents a registered Streamy configuration file with metadata

**Structure**:
```go
type Pipeline struct {
    ID           string    `json:"id"`            // Unique identifier (alphanumeric + hyphens)
    Name         string    `json:"name"`          // Human-readable display name
    Path         string    `json:"path"`          // Absolute path to config file
    Description  string    `json:"description"`   // Optional user-provided description
    RegisteredAt time.Time `json:"registered_at"` // Timestamp of registration
    
    // Runtime fields (not persisted in registry.json)
    Status     PipelineStatus   `json:"-"` // Current verification status
    LastRun    time.Time        `json:"-"` // Last verify/apply execution
    LastResult *ExecutionResult `json:"-"` // Detailed execution result
}
```

**Validation Rules**:
- `ID`: Must match regex `^[a-z0-9][a-z0-9-]*[a-z0-9]$`, length 1-64 characters
- `Name`: Required, max 100 characters
- `Path`: Must be absolute path, file must exist at registration time
- `Description`: Optional, max 500 characters
- `RegisteredAt`: Automatically set to current time during registration

**Relationships**:
- One-to-one with configuration file on filesystem
- One-to-one with status cache entry (keyed by ID)
- One-to-many with execution results (historical)

**Lifecycle**:
```
[User runs register] → Pipeline created with status=unknown
                    → Persisted to registry.json
                    → Dashboard discovers on next poll

[User runs refresh]  → Pipeline.Status updated from verify
                    → Status cached in status.json

[User runs unregister] → Pipeline removed from registry.json
                       → Status cache entry optionally cleared
```

### 2. PipelineStatus

**Location**: `internal/registry/types.go`  
**Purpose**: Enum representing pipeline verification state

**Values**:
```go
type PipelineStatus string

const (
    StatusUnknown   PipelineStatus = "unknown"   // Not yet verified
    StatusSatisfied PipelineStatus = "satisfied" // All requirements met
    StatusDrifted   PipelineStatus = "drifted"   // Requirements not met
    StatusFailed    PipelineStatus = "failed"    // Verification error
    StatusVerifying PipelineStatus = "verifying" // In progress (transient)
    StatusApplying  PipelineStatus = "applying"  // Apply in progress (transient)
)
```

**Display Representation**:
- Unicode icons: 🟢 (satisfied), 🟡 (drifted), 🔴 (failed), ⚪ (unknown)
- ASCII fallback: `[OK]`, `[!!]`, `[XX]`, `[??]`
- Color codes: Green (42), Yellow (226), Red (196), Gray (250)

**State Transitions**:
```
unknown → verifying → (satisfied | drifted | failed)
satisfied → verifying → (satisfied | drifted | failed)
drifted → applying → (satisfied | failed)
```

### 3. RegistryFile

**Location**: `internal/registry/types.go`  
**Purpose**: JSON file format for persistent registry storage

**Structure**:
```go
type RegistryFile struct {
    Version   string     `json:"version"`   // Schema version (currently "1.0")
    Pipelines []Pipeline `json:"pipelines"` // Array of registered pipelines
}
```

**File Location**: `~/.streamy/registry.json` (expandable via `$HOME`)

**Example JSON**:
```json
{
  "version": "1.0",
  "pipelines": [
    {
      "id": "dev-setup",
      "name": "Development Environment",
      "path": "/home/user/configs/dev.yaml",
      "description": "Local development tools and configs",
      "registered_at": "2025-10-08T15:20:00Z"
    },
    {
      "id": "staging-env",
      "name": "Staging Environment",
      "path": "/home/user/configs/staging.yaml",
      "description": "",
      "registered_at": "2025-10-09T09:45:00Z"
    }
  ]
}
```

**Persistence Guarantees**:
- Atomic writes (temp file + rename)
- UTF-8 encoding
- Pretty-printed with 2-space indentation
- File permissions: 0644 (user read/write, group/other read-only)

### 4. CachedStatus

**Location**: `internal/registry/types.go`  
**Purpose**: Runtime status information cached separately from registry

**Structure**:
```go
type CachedStatus struct {
    Status      PipelineStatus `json:"status"`        // Current verification status
    LastRun     time.Time      `json:"last_run"`      // Timestamp of last verify/apply
    Summary     string         `json:"summary"`       // Brief result description
    StepCount   int            `json:"step_count"`    // Total steps in pipeline
    FailedSteps []string       `json:"failed_steps"`  // IDs of failed steps (if any)
}
```

**File Location**: `~/.streamy/status.json`

**Rationale for Separation**:
- Registry stores static metadata (registration info)
- Status cache stores dynamic runtime state (verification results)
- Allows status updates without modifying registry history
- Reduces registry file size and write frequency

### 5. ExecutionResult

**Location**: `internal/registry/types.go`  
**Purpose**: Detailed outcome of verify or apply operation

**Structure**:
```go
type ExecutionResult struct {
    PipelineID  string         `json:"pipeline_id"`
    Operation   string         `json:"operation"`    // "verify" or "apply"
    Status      PipelineStatus `json:"status"`
    Success     bool           `json:"success"`
    StepResults []StepResult   `json:"step_results"`
    Duration    time.Duration  `json:"duration"`
    CompletedAt time.Time      `json:"completed_at"`
    Error       *ErrorDetail   `json:"error,omitempty"`
}
```

**Usage**:
- Populated by verify/apply engine
- Stored in status cache
- Referenced by list/show commands for display
- Not persisted long-term (only most recent result cached)

## Data Flow Diagrams

### Register Command Flow

```
User Input (path, description)
    ↓
Validate file exists
    ↓
Parse & validate config (internal/config)
    ↓
Generate/validate pipeline ID
    ↓
Create Pipeline struct
    ↓
Registry.Add(pipeline)
    ↓
Registry.Save() → atomic write to registry.json
    ↓
Success message to user
```

### List Command Flow

```
User executes list command
    ↓
Registry.Load() → read registry.json
    ↓
StatusCache.Load() → read status.json
    ↓
Merge: Pipeline + CachedStatus
    ↓
Format output (table or JSON)
    ↓
Display to stdout
```

### Refresh Command Flow

```
User executes refresh command [pipeline-id]
    ↓
Registry.Load() → get pipeline(s)
    ↓
For each pipeline (concurrent):
    ↓
    engine.Verify() executes config check
    ↓
    ExecutionResult produced
    ↓
    StatusCache.Set(id, status)
    ↓
StatusCache.Save() → atomic write to status.json
    ↓
Summary report to user
```

### Unregister Command Flow

```
User Input (pipeline-id, --force)
    ↓
Registry.Get(id) → verify exists
    ↓
Prompt confirmation (unless --force)
    ↓
User confirms
    ↓
Registry.Remove(id)
    ↓
Registry.Save() → atomic write
    ↓
StatusCache.Delete(id) [optional]
    ↓
StatusCache.Save() [if modified]
    ↓
Success message to user
```

## Concurrency Model

### Thread Safety

**Registry Operations**:
- Uses `sync.RWMutex` for read/write locking
- Multiple readers allowed simultaneously
- Single writer blocks all readers
- Methods: `Add`, `Remove`, `Update` acquire write lock
- Methods: `Get`, `List` acquire read lock

**StatusCache Operations**:
- Same mutex pattern as Registry
- Independent lock (not shared with Registry)
- Safe for concurrent refresh operations

**Atomic File Writes**:
```go
// Pattern used by Registry.Save() and StatusCache.Save():
1. Write to temp file: registry.json.tmp
2. Rename temp to target: mv registry.json.tmp registry.json
3. OS guarantees atomic rename (prevents partial reads)
```

### Concurrent Refresh

**Worker Pool Pattern**:
```
Main goroutine:
  - Creates semaphore channel (capacity = concurrency limit)
  - Spawns worker goroutines
  - Waits for all workers via WaitGroup

Worker goroutines:
  - Acquire semaphore slot
  - Execute engine.Verify() on assigned pipeline
  - Write result to preallocated slice
  - Release semaphore slot
  - Signal completion to WaitGroup
```

**Safety Guarantees**:
- No shared mutable state between workers (write to unique slice indices)
- StatusCache updates serialized via mutex
- Registry not modified during refresh command (read-only)

## Validation Rules

### Pipeline ID Validation

**Format**: `^[a-z0-9][a-z0-9-]*[a-z0-9]$`

**Rules**:
- Must start and end with alphanumeric character
- Can contain hyphens in the middle
- Lowercase only
- Length: 1-64 characters

**Invalid Examples**:
- `-dev-setup` (starts with hyphen)
- `dev-setup-` (ends with hyphen)
- `Dev-Setup` (uppercase)
- `dev_setup` (underscore not allowed)
- `dev.setup` (dot not allowed)

**Valid Examples**:
- `dev-setup`
- `staging-env`
- `prod`
- `my-app-2024`

### Path Validation

**Rules**:
- Must be absolute path (starts with `/` on Unix, `C:\` on Windows)
- File must exist at registration time
- Must be readable by current user
- Must be valid Streamy YAML config

**Auto-conversion**:
- Relative paths converted to absolute using `filepath.Abs()`
- Tilde (`~`) expanded to user's home directory

### Description Validation

**Rules**:
- Optional (empty string allowed)
- Max length: 500 characters
- Any UTF-8 characters allowed
- No special processing (stored as-is)

## Schema Versioning

**Current Version**: `1.0`

**Version Field Purpose**:
- Enables future schema migrations
- Commands check version on load
- Unsupported versions trigger clear error message

**Future Compatibility**:
- New optional fields: Add with default values (minor version bump)
- New required fields: Require migration tool (major version bump)
- Field removals: Deprecated in version N, removed in N+1 (major bump)

**Migration Strategy** (when needed):
```go
func migrateRegistry(file RegistryFile) (RegistryFile, error) {
    switch file.Version {
    case "1.0":
        return file, nil // Current version, no migration
    case "0.9": // Hypothetical old version
        return migrateFrom0_9To1_0(file)
    default:
        return file, fmt.Errorf("unsupported registry version: %s", file.Version)
    }
}
```

## Error Handling

### Error Types

**ConfigNotFoundError**:
- Trigger: File doesn't exist at provided path
- Suggestion: "Check the path and try again"

**ConfigParseError**:
- Trigger: YAML syntax error or schema violation
- Suggestion: Include line number and fix hint from parser

**DuplicateIDError**:
- Trigger: Pipeline with same ID already registered
- Suggestion: "Use --id flag to specify a different ID or unregister the existing pipeline first"

**PipelineNotFoundError**:
- Trigger: Attempting to unregister/show non-existent pipeline
- Suggestion: "Run 'streamy list' to see registered pipelines"

**PermissionError**:
- Trigger: Cannot write to registry directory
- Suggestion: "Check file permissions for ~/.streamy/"

## Query Operations

### List Filtering (Future Enhancement)

**Potential Flags**:
- `--status satisfied|drifted|failed|unknown`: Filter by status
- `--path-contains <substring>`: Filter by path pattern
- `--sort-by id|name|status|last-run`: Sort criteria

**Implementation Note**: Not in MVP scope, but data model supports it.

### Pagination (Future Enhancement)

**Potential Flags**:
- `--limit N`: Show only N pipelines
- `--offset M`: Skip first M pipelines

**Implementation Note**: Add when user reports >100 registered pipelines.

## Summary

All required data structures already exist in the codebase. The new CLI commands will:
- **Read** from Registry and StatusCache using existing methods
- **Write** via existing atomic persistence layer
- **Display** by formatting existing types into human/machine-readable output
- **Validate** using existing config parser and custom ID validation

No schema changes required. Zero breaking changes to existing data structures.
