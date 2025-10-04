# Plugin Interface Contract: line_in_file

**Feature**: Add Built-In Plugin: line_in_file  
**Date**: October 4, 2025  
**Purpose**: Define the contract for line_in_file plugin implementation

---

## Overview

The `line_in_file` plugin implements the `internal/plugin.Plugin` interface to provide declarative, idempotent text file line management within Streamy's DAG execution engine.

---

## Interface Contract

### Plugin Interface (from `internal/plugin/interface.go`)

```go
type Plugin interface {
    // Name returns the plugin type identifier (must be unique)
    Name() string
    
    // Validate checks configuration during parse phase
    // Returns error if config is invalid
    Validate(ctx context.Context, step config.Step) error
    
    // Execute performs the plugin action
    // Returns StepResult with changed status and optional error
    Execute(ctx context.Context, step config.Step, logger logger.Logger) (model.StepResult, error)
    
    // DryRun simulates execution without making changes
    // Returns preview of what would change
    DryRun(ctx context.Context, step config.Step, logger logger.Logger) (model.StepResult, error)
}
```

---

## Method Contracts

### 1. Name() → string

**Purpose**: Return unique plugin type identifier for registry lookup

**Contract**:
- **Inputs**: None
- **Outputs**: `"line_in_file"` (string constant)
- **Side Effects**: None
- **Idempotency**: Pure function, always returns same value
- **Error Conditions**: Cannot error

**Test Cases**:
```go
func TestLineInFile_Name(t *testing.T) {
    plugin := &LineInFilePlugin{}
    assert.Equal(t, "line_in_file", plugin.Name())
}
```

---

### 2. Validate(ctx, step) → error

**Purpose**: Validate configuration during config parse phase, before DAG execution

**Contract**:
- **Inputs**: 
  - `ctx`: Context for cancellation (timeout: 5s for validation phase)
  - `step`: Parsed step configuration containing plugin-specific fields
- **Outputs**: 
  - `nil` if configuration valid
  - `ConfigError` if configuration invalid (with field, value, reason)
- **Side Effects**: None (pure validation, no I/O)
- **Idempotency**: Pure function for same inputs
- **Error Conditions**:
  - `file` field empty → `ConfigError{Field: "file", Reason: "required"}`
  - `line` field empty → `ConfigError{Field: "line", Reason: "required"}`
  - `state` not in `[present, absent]` → `ConfigError{Field: "state", Reason: "must be present or absent"}`
  - `state: absent` and `match` empty → `ConfigError{Field: "match", Reason: "required when state is absent"}`
  - `match` not valid regex → `ConfigError{Field: "match", Reason: "invalid regex: {details}"}`
  - `on_multiple_matches` not in `[first, all, error, prompt]` → `ConfigError{...}`
  - `encoding` not supported → `ConfigError{Field: "encoding", Reason: "unsupported encoding"}`

**Test Cases**:
```go
func TestLineInFile_Validate(t *testing.T) {
    tests := []struct {
        name      string
        step      config.Step
        expectErr bool
        errField  string
    }{
        {"valid_present", validPresentStep, false, ""},
        {"valid_absent", validAbsentStep, false, ""},
        {"missing_file", stepWithoutFile, true, "file"},
        {"missing_line", stepWithoutLine, true, "line"},
        {"absent_without_match", absentWithoutMatch, true, "match"},
        {"invalid_regex", invalidRegexStep, true, "match"},
        {"invalid_state", invalidStateStep, true, "state"},
        {"invalid_on_multiple", invalidOnMultiple, true, "on_multiple_matches"},
        {"unsupported_encoding", unsupportedEncoding, true, "encoding"},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := plugin.Validate(ctx, tt.step)
            if tt.expectErr {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tt.errField)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

---

### 3. Execute(ctx, step, logger) → (StepResult, error)

**Purpose**: Perform idempotent file line management according to configuration

**Contract**:
- **Inputs**:
  - `ctx`: Context for cancellation and timeout (user-configurable, default 30s)
  - `step`: Validated step configuration
  - `logger`: Structured logger for operation tracking
- **Outputs**:
  - `StepResult` with:
    - `Changed: true` if file modified, `false` if idempotent
    - `Message`: Human-readable summary (e.g., "Added 1 line", "Replaced 2 lines", "No changes needed")
    - `DiffOutput`: Empty (populated only in DryRun with --verbose)
    - `Error`: nil on success
  - `error`: ExecutionError if operation failed
- **Side Effects**:
  - Reads target file from filesystem
  - Writes modified content to temp file
  - Renames temp file to target file (atomic)
  - Creates backup file if `backup: true` and content changed
  - Logs operation start, duration, result via logger
  - May prompt user if `on_multiple_matches: prompt` and TTY available
- **Idempotency**: 
  - Running twice with same config and file state returns `Changed: false` on second run
  - No backup created on idempotent run
- **Error Conditions**:
  - File not readable → `ExecutionError{Operation: "read file", Cause: os.ErrPermission}`
  - Directory not writable → `ExecutionError{Operation: "write temp file", Cause: os.ErrPermission}`
  - Invalid encoding in file → `ExecutionError{Operation: "decode file", Cause: encoding error}`
  - Backup dir not writable → `ExecutionError{Operation: "write backup", Cause: ...}`
  - Interactive prompt required but not TTY → `InteractiveError{Reason: "not in TTY context"}`
  - Context cancelled → `ctx.Err()`

**Preconditions**:
- `Validate()` must have returned nil
- Config fields populated and valid

**Postconditions** (on success):
- If `Changed: true`:
  - Target file contains expected modifications
  - File permissions preserved from original (or 0644 if new file)
  - Backup file exists if `backup: true` (with timestamp in name)
- If `Changed: false`:
  - Target file unchanged
  - No backup created
  - No temp files left behind

**Test Cases**:
```go
func TestLineInFile_Execute(t *testing.T) {
    tests := []struct {
        name            string
        config          Config
        existingContent string
        expectedContent string
        expectChanged   bool
        expectError     bool
        errorType       string
    }{
        // State: present, no match
        {"append_new_line", presentNoMatch, "", "new line\n", true, false, ""},
        {"idempotent_existing", presentNoMatch, "new line\n", "new line\n", false, false, ""},
        
        // State: present, with match
        {"replace_single_match", presentWithMatch, "old=1\n", "new=1\n", true, false, ""},
        {"replace_first_of_many", presentMatchFirst, "old=1\nold=2\n", "new=1\nold=2\n", true, false, ""},
        {"replace_all_matches", presentMatchAll, "old=1\nold=2\n", "new=1\nnew=2\n", true, false, ""},
        {"error_on_multiple", presentMatchError, "old=1\nold=2\n", "", false, true, "ExecutionError"},
        
        // State: absent
        {"remove_matched_line", absentWithMatch, "keep\nremove\nkeep\n", "keep\nkeep\n", true, false, ""},
        {"idempotent_not_present", absentWithMatch, "keep\nkeep\n", "keep\nkeep\n", false, false, ""},
        
        // File creation
        {"create_new_file", presentNoMatch, fileNotExists, "new line\n", true, false, ""},
        
        // Permissions
        {"preserve_permissions", presentNoMatch, contentWithPerm0600, "...", true, false, ""},
        {"permission_denied_read", presentNoMatch, fileUnreadable, "", false, true, "ExecutionError"},
        {"permission_denied_write", presentNoMatch, dirUnwritable, "", false, true, "ExecutionError"},
        
        // Backup
        {"backup_created", presentWithBackup, "old\n", "new\n", true, false, ""},
        {"no_backup_if_unchanged", presentWithBackup, "line\n", "line\n", false, false, ""},
        
        // Encoding
        {"utf8_default", presentUTF8, utf8Content, utf8Modified, true, false, ""},
        {"latin1_explicit", presentLatin1, latin1Content, latin1Modified, true, false, ""},
        
        // Symlinks
        {"follow_symlink", presentNoMatch, symlinkedFile, "...", true, false, ""},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup test file
            // Execute plugin
            // Assert result
        })
    }
}
```

---

### 4. DryRun(ctx, step, logger) → (StepResult, error)

**Purpose**: Simulate execution to preview changes without modifying filesystem

**Contract**:
- **Inputs**: Same as Execute()
- **Outputs**:
  - `StepResult` with:
    - `Changed: true/false` (same logic as Execute)
    - `Message`: Same as Execute would return
    - `DiffOutput`: Unified diff showing additions/removals/replacements (populated if --verbose)
    - `Error`: nil on success
  - `error`: Same error conditions as Execute validation phase (file read, encoding, etc.)
- **Side Effects**:
  - Reads target file from filesystem (read-only)
  - Logs dry-run operation via logger
  - May prompt user if `on_multiple_matches: prompt` (same as Execute)
  - **NO writes**: No temp files, no backups, no file modifications
- **Idempotency**: Pure function for same inputs (file content, config)
- **Error Conditions**: Same as Execute, except no write-permission errors

**Preconditions**: Same as Execute()

**Postconditions**:
- No files created or modified
- No backup files created
- Diff output shows what Execute() would do

**Test Cases**:
```go
func TestLineInFile_DryRun(t *testing.T) {
    tests := []struct {
        name            string
        config          Config
        existingContent string
        expectChanged   bool
        expectDiff      string
    }{
        {"preview_append", presentNoMatch, "", true, "+new line\n"},
        {"preview_replace", presentWithMatch, "old=1\n", true, "-old=1\n+new=1\n"},
        {"preview_remove", absentWithMatch, "keep\nremove\n", true, "-remove\n"},
        {"preview_no_change", presentNoMatch, "line\n", false, ""},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := plugin.DryRun(ctx, tt.step, logger)
            require.NoError(t, err)
            assert.Equal(t, tt.expectChanged, result.Changed)
            if tt.expectDiff != "" {
                assert.Contains(t, result.DiffOutput, tt.expectDiff)
            }
            // Verify no files were modified
        })
    }
}
```

---

## Integration with Streamy Core

### Registration

```go
// cmd/streamy/plugins_import.go
import (
    "github.com/alexisbeaulieu97/streamy/internal/plugins/lineinfile"
)

func init() {
    registry.MustRegister(lineinfile.New())
}
```

### Execution Flow

```
1. Config Parser → Calls Validate() for each line_in_file step
2. DAG Builder → Adds line_in_file steps to dependency graph
3. Executor → Calls Execute() or DryRun() based on --dry-run flag
4. Logger → Receives structured log events from plugin
5. TUI → Displays step progress and results
```

### Context Handling

- Plugin MUST respect `ctx.Done()` for cancellation
- Long operations (large file processing) MUST check context periodically
- Interactive prompts MUST abort on context cancellation

### Logging Contract

Expected log events (via `logger.Logger`):

```go
logger.Info("line_in_file: starting", 
    "file", resolvedPath, 
    "state", config.State,
    "has_match", config.Match != "")

logger.Debug("line_in_file: file read", "size_bytes", len(content))

logger.Info("line_in_file: completed",
    "changed", result.Changed,
    "action", changeset.Action,
    "duration_ms", elapsed)

logger.Error("line_in_file: failed",
    "error", err,
    "operation", "read file")
```

---

## Performance Expectations

Based on Technical Context and FR-021:

| File Size | Operation | Expected Duration |
|-----------|-----------|-------------------|
| < 1 MB | Validate | < 1 ms |
| < 10 MB | Execute (append) | < 50 ms |
| < 10 MB | Execute (replace) | < 100 ms |
| < 10 MB | DryRun | < 100 ms |
| 100 MB | Execute (append) | < 2 s |
| 1 GB | Execute (replace) | < 30 s (streaming) |

**Degradation**: Performance degrades linearly with file size for streaming operations

---

## Error Handling Contract

All errors MUST include:
1. **Operation Context**: What was being attempted
2. **File Path**: Which file caused the error
3. **Root Cause**: Wrapped original error

Example error messages:
```
- "failed to read file /home/user/.bashrc: permission denied"
- "failed to compile regex pattern '^export PATH(': error parsing regexp: missing closing )"
- "multiple matches found for pattern '^debug=' but on_multiple_matches not specified and not in TTY context"
```

---

## Backward Compatibility

- Plugin interface is stable within Streamy major version
- Config schema additions must be backward compatible (new optional fields only)
- Default behavior must not change in minor versions

---

## Contract Tests

Minimum test coverage requirements:
- **Unit Tests**: 85%+ code coverage
- **Contract Tests**: All interface methods with table-driven tests
- **Integration Tests**: End-to-end Streamy execution with various configs
- **Error Path Tests**: Every error condition triggered and validated

**Contract Verification**:
```bash
go test ./internal/plugins/lineinfile -v -cover
go test ./tests -run TestIntegration_LineInFile
```

---

## Summary

**Interface Compliance**: Implements `internal/plugin.Plugin` ✅  
**Method Count**: 4 (Name, Validate, Execute, DryRun)  
**Test Scenarios**: 30+ covering success, idempotency, errors, edge cases  
**Performance Targets**: <100ms for typical configs (<10MB files)  
**Error Categories**: 3 (Config, Execution, Interactive)

**Ready for Implementation**: ✅
