# Data Model: line_in_file Plugin

**Feature**: Add Built-In Plugin: line_in_file  
**Date**: October 4, 2025  
**Purpose**: Define data structures, validation rules, and state transitions

---

## 1. Configuration Structure

### LineInFileConfig
Primary configuration struct for the plugin step.

**Fields**:

| Field | Type | Required | Default | Validation Rules |
|-------|------|----------|---------|------------------|
| `file` | string | ✅ | — | Must be non-empty; tilde expansion applied; absolute or relative path |
| `line` | string | ✅ | — | Must be non-empty; exact text content to ensure/remove |
| `state` | string | ❌ | `"present"` | Must be `"present"` or `"absent"` |
| `match` | string | ❌ | — | Valid regex pattern; required if `state: absent` (FR-008) |
| `on_multiple_matches` | string | ❌ | `"prompt"` | Must be `"first"`, `"all"`, `"error"`, or `"prompt"` |
| `backup` | bool | ❌ | `false` | — |
| `backup_dir` | string | ❌ | — | If specified, must be valid directory path; created if missing |
| `encoding` | string | ❌ | `"utf-8"` | Must be supported encoding name (utf-8, latin-1, ascii, etc.) |

**Validation Rules** (applied during config parse phase):
1. If `state: absent` and `match` is empty → **ERROR**: "state: absent requires match pattern"
2. If `match` is provided and not valid regex → **ERROR**: "invalid match pattern: {regex error}"
3. If `on_multiple_matches` not in allowed values → **ERROR**: "on_multiple_matches must be one of: first, all, error, prompt"
4. If `encoding` not supported → **ERROR**: "unsupported encoding: {encoding}"

**Example**:
```yaml
- id: set_shell_path
  type: line_in_file
  file: "~/.bashrc"
  line: 'export PATH="$PATH:~/bin"'
  match: '^export PATH='
  state: present
  on_multiple_matches: first
  backup: true
  backup_dir: "~/.config/backups"
  encoding: "utf-8"
```

---

## 2. Runtime State Entities

### FileState
Represents the current state of the target file.

**Fields**:
- `Path` (string): Resolved absolute path after tilde expansion
- `Exists` (bool): Whether file exists on filesystem
- `Permissions` (os.FileMode): Unix permission bits (e.g., 0644)
- `Content` ([]byte): Raw file content
- `Lines` ([]string): Content split by newlines
- `LineCount` (int): Number of lines in file

**Lifecycle**:
1. Created during Execute phase by reading target file
2. If file doesn't exist: `Exists = false`, other fields zero-valued
3. Used for change detection and diff generation

---

### MatchResult
Result of applying regex pattern to file content.

**Fields**:
- `Matched` (bool): Whether any lines matched the pattern
- `LineNumbers` ([]int): 0-indexed positions of matched lines
- `MatchedLines` ([]string): Content of matched lines
- `MatchCount` (int): Total number of matches

**Usage**:
- For `state: present` with `match`: Determines replacement behavior
- For `state: absent`: Identifies lines to remove
- For `on_multiple_matches` decision logic

**Example**:
```go
MatchResult{
    Matched: true,
    LineNumbers: []int{5, 12, 23},
    MatchedLines: []string{"export PATH=/usr/bin", "export PATH=/usr/local/bin", "export PATH=/opt/bin"},
    MatchCount: 3,
}
```

---

### ChangeSet
Records planned or applied modifications for reporting.

**Fields**:
- `Action` (string): `"append"`, `"replace"`, `"remove"`, or `"none"`
- `LinesAdded` ([]string): New lines added to file
- `LinesRemoved` ([]string): Lines removed from file
- `LinesModified` ([]LineModification): Replaced lines (before/after pairs)
- `Changed` (bool): Whether file content changed
- `BackupCreated` (string): Path to backup file (empty if no backup)

**LineModification Sub-Type**:
```go
type LineModification struct {
    LineNumber int
    Before     string
    After      string
}
```

**Usage**:
- Populated during dry-run to show preview
- Populated during execute to report changes
- Drives logging output and step result status

---

### StepResult
Returned by Execute and DryRun methods.

**Fields**:
- `Changed` (bool): Whether file was modified (false = idempotent)
- `Message` (string): Human-readable summary (e.g., "Added 1 line", "No changes needed")
- `DiffOutput` (string): Unified diff format (only for --verbose)
- `Error` (error): Any error encountered (nil on success)

**State Mapping**:
- `Changed = false` → Idempotent success (FR-011, FR-020)
- `Changed = true` → File modified successfully
- `Error != nil` → Operation failed (permission, validation, etc.)

---

## 3. State Transitions

### Presence State Machine

```
Initial State: File + Line Check
    ↓
[state: present, no match]
    ↓
    Line exists? → YES → DONE (no change)
    ↓ NO
    Append line → Changed = true → DONE

[state: present, with match]
    ↓
    Match exists? → NO → Append line → Changed = true → DONE
    ↓ YES
    Multiple matches?
        ↓ NO (single match)
        Replace match → Changed = true → DONE
        ↓ YES (multiple matches)
        Check on_multiple_matches:
            ↓ "first"
            Replace first match → Changed = true → DONE
            ↓ "all"
            Replace all matches → Changed = true → DONE
            ↓ "error"
            ERROR → DONE
            ↓ "prompt"
            Ask user → [first|all|error] → Recurse with chosen strategy

[state: absent, with match]
    ↓
    Match exists? → NO → DONE (no change)
    ↓ YES
    Remove all matching lines → Changed = true → DONE
```

### Backup Creation Logic

```
[backup: true AND Changed = true]
    ↓
    backup_dir specified?
        ↓ YES
        Resolve backup_dir path
        Create dir if missing
        ↓ NO
        Use directory of original file
    ↓
    Generate backup path: <filename>.<timestamp>.bak
    Copy original file to backup path
    Record backup path in ChangeSet
    ↓
    Continue with file modification
```

---

## 4. Validation Rules Summary

### Config Parse Phase (Before DAG Execution)
- **V-001**: `file` field must be non-empty
- **V-002**: `line` field must be non-empty
- **V-003**: `state` must be `"present"` or `"absent"`
- **V-004**: If `state: absent`, `match` field must be non-empty
- **V-005**: If `match` provided, must compile as valid regex (FR-018)
- **V-006**: `on_multiple_matches` must be one of: `first`, `all`, `error`, `prompt`
- **V-007**: `encoding` must be supported charset name

### Execute Phase (During DAG Execution)
- **V-008**: Target file directory must exist (if file doesn't exist, parent dir must)
- **V-009**: Must have read permission on target file (if exists)
- **V-010**: Must have write permission on target file directory
- **V-011**: If `backup_dir` specified, must be writable
- **V-012**: If `on_multiple_matches: prompt`, must be in TTY context
- **V-013**: File content must be valid for specified encoding

---

## 5. Error Types

### Configuration Errors (Validation Phase)
```go
type ConfigError struct {
    Field   string  // e.g., "match", "state"
    Value   string  // Invalid value provided
    Reason  string  // Why it's invalid
}
```

**Examples**:
- `ConfigError{Field: "state", Value: "maybe", Reason: "must be 'present' or 'absent'"}`
- `ConfigError{Field: "match", Value: "[invalid(", Reason: "error parsing regexp: missing closing )"}`

### Execution Errors (Runtime Phase)
```go
type ExecutionError struct {
    Operation string  // e.g., "read file", "write backup", "apply changes"
    FilePath  string  // File being operated on
    Cause     error   // Underlying error (permission denied, etc.)
}
```

**Examples**:
- `ExecutionError{Operation: "read file", FilePath: "/etc/hosts", Cause: os.ErrPermission}`
- `ExecutionError{Operation: "write backup", FilePath: "/tmp/backup/hosts.20251004.bak", Cause: fs.ErrNotExist}`

### Interactive Errors (Prompt Phase)
```go
type InteractiveError struct {
    Reason string  // e.g., "not in TTY context", "invalid user input"
}
```

---

## 6. Encoding Mapping

Map of supported encoding names to Go encoding types:

| Config Value | Go Encoding | Description |
|--------------|-------------|-------------|
| `utf-8`, `utf8` | `unicode.UTF8` | Default, most common |
| `ascii` | `charmap.Windows1252` | 7-bit ASCII |
| `latin-1`, `iso-8859-1` | `charmap.ISO8859_1` | Western European |
| `latin-2`, `iso-8859-2` | `charmap.ISO8859_2` | Central European |
| `windows-1252` | `charmap.Windows1252` | Windows Western European |
| `utf-16` | `unicode.UTF16(BE/LE)` | 16-bit Unicode |

**Validation**: If user provides encoding not in this table → ConfigError

---

## 7. Relationships & Dependencies

### Plugin → Streamy Core
- Implements `internal/plugin.Plugin` interface
- Uses `internal/logger.Logger` for structured logging
- Registered via `internal/plugin.Registry`

### Plugin → File System
- Reads from `file` path
- Writes to temp file in same directory
- Optionally writes to `backup_dir`
- Preserves permissions via `os.Chmod`

### Plugin → User (Interactive)
- Prompts via `os.Stdin` when `on_multiple_matches: prompt`
- Outputs via `os.Stderr` (prompts) and logger (results)

### Plugin → DAG Execution
- Can declare `depends_on` to run after file creation steps
- Participates in parallel execution (no shared state between instances)
- Returns `StepResult` for DAG progress tracking

---

## Summary

**Total Entities**: 5 primary (Config, FileState, MatchResult, ChangeSet, StepResult)  
**Total Validation Rules**: 13 (7 parse-phase, 6 execute-phase)  
**Total State Transitions**: 3 main branches (present no match, present with match, absent)  
**Total Error Types**: 3 (ConfigError, ExecutionError, InteractiveError)

**Key Invariants**:
1. File content never modified without passing validation
2. Atomic writes guarantee no partial updates
3. Idempotency enforced by content comparison
4. All errors include file path context
5. Backup created before any destructive change

**Ready for Contract Generation**: ✅
