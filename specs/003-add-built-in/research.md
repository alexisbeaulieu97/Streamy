# Research: line_in_file Plugin

**Date**: October 4, 2025  
**Feature**: Add Built-In Plugin: line_in_file  
**Purpose**: Research technical decisions, patterns, and best practices for implementing idempotent file line management

---

## 1. File Modification Patterns in Go

### Decision: Atomic Write via Temp File + Rename
**Rationale**: 
- Prevents file corruption if process crashes mid-write
- POSIX rename is atomic operation
- Preserves original file if write fails
- Standard pattern used in config management tools (Ansible, Puppet)

**Implementation Pattern**:
```go
// Pseudo-code
1. Read original file into memory
2. Perform line modifications in memory
3. Write modified content to temp file in same directory
4. Set permissions/ownership on temp file to match original
5. Rename temp file to original path (atomic)
```

**Alternatives Considered**:
- Direct file modification (in-place editing) → Rejected: not atomic, corruption risk
- Copy-on-write with hardlinks → Rejected: not portable across filesystems
- Database-style write-ahead log → Rejected: too complex for this use case

### Decision: Stream Processing for Large Files
**Rationale**:
- Files can be any size (FR-021: no arbitrary limits)
- Loading multi-GB files into memory is wasteful
- Line-by-line processing keeps memory bounded

**Implementation Pattern**:
- Use `bufio.Scanner` for line-by-line reading
- For presence checks: scan once to detect if line exists
- For replacements: scan + buffer modified lines, write to temp file
- Trade-off: Two passes for some operations vs. memory efficiency

**Alternatives Considered**:
- Memory-map entire file → Rejected: doesn't work well for line-oriented text operations
- Load entire file always → Rejected: violates FR-021, scales poorly

---

## 2. Regular Expression Matching

### Decision: Use Go's `regexp` Package with Compiled Patterns
**Rationale**:
- Standard library, no external dependencies
- RE2 syntax (safe, predictable performance)
- Compile once per step execution, reuse for all line checks

**Implementation Pattern**:
```go
// During Validate phase
compiled, err := regexp.Compile(config.Match)
if err != nil {
    return fmt.Errorf("invalid match pattern: %w", err)
}
// Store compiled regex in plugin state
```

**Performance Considerations**:
- Regex compilation is expensive (~microseconds to milliseconds)
- Cache compiled pattern in plugin instance
- FR-018: Validate patterns during config parsing phase

**Alternatives Considered**:
- Simple string matching only → Rejected: doesn't meet requirement for pattern-based replacement
- External regex library (PCRE) → Rejected: adds dependency, overkill for this use case

---

## 3. Encoding Support

### Decision: UTF-8 Default + `golang.org/x/text/encoding` for Other Encodings
**Rationale**:
- FR-001a: Support configurable encoding
- UTF-8 is default for most modern configs
- `x/text/encoding` provides comprehensive charset support (latin-1, ascii, windows-1252, etc.)
- Minimal dependency (official Go extended library)

**Implementation Pattern**:
```go
import (
    "golang.org/x/text/encoding"
    "golang.org/x/text/encoding/charmap"
    "golang.org/x/text/encoding/unicode"
)

func getDecoder(encodingName string) (*encoding.Decoder, error) {
    switch strings.ToLower(encodingName) {
    case "", "utf-8", "utf8":
        return unicode.UTF8.NewDecoder(), nil
    case "latin-1", "latin1", "iso-8859-1":
        return charmap.ISO8859_1.NewDecoder(), nil
    // ... other encodings
    }
}
```

**Alternatives Considered**:
- UTF-8 only → Rejected: doesn't meet FR-001a requirement
- Auto-detect encoding → Rejected: unreliable, adds complexity, explicit is better

---

## 4. Interactive Prompts for Runtime Decisions

### Decision: Use `bufio.Reader` with TTY Detection
**Rationale**:
- FR-004a: `on_multiple_matches: prompt` requires interactive input
- Must work in TTY (human user) and non-TTY (CI/CD) contexts
- Fail gracefully in non-interactive mode

**Implementation Pattern**:
```go
func promptUser(question string, options []string) (string, error) {
    if !terminal.IsTerminal(int(os.Stdin.Fd())) {
        return "", fmt.Errorf("interactive prompt required but not in TTY context")
    }
    
    fmt.Fprintf(os.Stderr, "%s\nOptions: %s\n", question, strings.Join(options, ", "))
    reader := bufio.NewReader(os.Stdin)
    response, err := reader.ReadString('\n')
    // ... validate and return
}
```

**Non-TTY Behavior**:
- Return error clearly stating interactive input required
- User must specify `on_multiple_matches` explicitly in config

**Alternatives Considered**:
- Always default to "first" match → Rejected: violates FR-004a requirement for user choice
- Use GUI dialog → Rejected: not appropriate for CLI tool

---

## 5. Backup File Management

### Decision: ISO 8601 Timestamps in Filename
**Rationale**:
- FR-010b: Format `<filename>.<timestamp>.bak`
- ISO 8601 (YYYY-MM-DDTHH-MM-SS) sorts chronologically
- Unambiguous, no timezone complexity needed (use local time)

**Implementation Pattern**:
```go
timestamp := time.Now().Format("20060102T150405") // Go's time format
backupPath := fmt.Sprintf("%s.%s.bak", originalPath, timestamp)
```

**Backup Directory Logic**:
- FR-010a: Optional `backup_dir` field
- If specified: resolve absolute path, create if missing
- If not specified: use directory of original file
- Preserve filename in backup name for easy identification

**Cleanup Strategy**:
- Plugin does NOT auto-delete old backups (user responsibility)
- Document in quickstart.md: users should manage backup retention

**Alternatives Considered**:
- Unix timestamp → Rejected: less human-readable
- UUID in filename → Rejected: loses temporal ordering
- Auto-cleanup old backups → Rejected: surprising behavior, user should control retention

---

## 6. Idempotency Implementation

### Decision: Content-Based Change Detection
**Rationale**:
- FR-011: Running same config twice produces no changes
- Compare final state to current state before writing
- Skip write operation if content unchanged

**Implementation Pattern**:
```go
// After computing new file content
if bytes.Equal(originalContent, newContent) {
    return StepResult{Changed: false, Message: "No changes needed"}
}
// Otherwise proceed with atomic write
```

**Optimization**:
- For `state: present` without `match`: Quick scan to check if line already exists
- For `state: absent`: Quick scan to check if match pattern doesn't exist
- Avoid unnecessary file writes to preserve mtime

**Alternatives Considered**:
- Always write and let filesystem deduplicate → Rejected: changes mtime, triggers unnecessary rebuilds
- Hash-based comparison → Rejected: overkill, byte comparison is fast for typical files

---

## 7. Dry-Run Mode Implementation

### Decision: Generate Unified Diff Format
**Rationale**:
- FR-012: Integrate with `--dry-run` mode
- FR-013: Show diffs with `--verbose`
- Unified diff is standard, human-readable format

**Implementation Pattern**:
```go
// Use standard library or simple custom implementation
func generateDiff(original, modified string) string {
    originalLines := strings.Split(original, "\n")
    modifiedLines := strings.Split(modified, "\n")
    
    // Generate unified diff with +/- markers
    // Show context around changes (3 lines before/after)
}
```

**Dry-Run Execution Path**:
- Plugin's `DryRun()` method: Perform all logic except actual write
- Return preview message with diff
- No temp files created, no renames performed

**Alternatives Considered**:
- External diff tool → Rejected: adds dependency, not portable
- Side-by-side diff → Rejected: doesn't work well in narrow terminals
- No diff, just summary → Rejected: doesn't meet FR-013 requirement

---

## 8. Error Handling Strategy

### Decision: Wrapped Errors with Context
**Rationale**:
- FR-014: Clear errors for permission issues
- FR-018: Clear errors for regex syntax
- Use `fmt.Errorf` with `%w` to preserve error chains
- Add file path and operation context to all errors

**Implementation Pattern**:
```go
if err := os.WriteFile(tempPath, content, perm); err != nil {
    return fmt.Errorf("failed to write temp file %s: %w", tempPath, err)
}
```

**Error Categories**:
1. Configuration errors (validation phase): Regex syntax, missing required fields
2. Permission errors (execution phase): File read/write access denied
3. File system errors (execution phase): File not found, disk full
4. Encoding errors (execution phase): Invalid byte sequences

**Recovery Strategy**:
- Config errors: Fail fast during validation, before DAG execution starts
- Permission errors: Report and halt step, continue with remaining DAG if possible
- Atomic write failures: Temp file cleaned up, original file unchanged

**Alternatives Considered**:
- Silent failures with warnings → Rejected: violates "Safety by Default"
- Retry logic for transient failures → Rejected: can be added later if needed

---

## 9. Symlink Handling

### Decision: Follow Symlinks by Default
**Rationale**:
- Clarification answer: Follow symlink and modify target file
- Use `os.Stat()` instead of `os.Lstat()` to resolve symlinks
- Consistent with most Unix tools (cat, less, vim)

**Implementation Pattern**:
```go
// os.Open() and os.Stat() follow symlinks automatically
// No special handling needed
```

**Edge Case**:
- Broken symlinks: Will fail with clear "file not found" error
- Symlink loops: Protected by OS (ELOOP error)

**Alternatives Considered**:
- Configurable `follow_symlinks` option → Rejected: clarification specified default behavior, no request for configurability
- Detect and error on symlinks → Rejected: not user requirement

---

## 10. Testing Strategy

### Decision: Table-Driven Unit Tests + Integration Tests
**Rationale**:
- Go idiom for comprehensive test coverage
- Test matrix: (state) × (match) × (existing content) × (encoding)
- Integration tests validate full Streamy execution flow

**Test Structure**:
```go
func TestLineInFile_Execute(t *testing.T) {
    tests := []struct {
        name           string
        config         Config
        existingContent string
        expectedContent string
        expectChanged  bool
        expectError    bool
    }{
        // Table entries for each scenario
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Run test case
        })
    }
}
```

**Test Scenarios** (from spec.md):
1. Add new line (append)
2. Replace matched line
3. Remove line
4. Idempotent re-run (no changes)
5. Dry-run preview
6. File permissions preserved
7. Backup file creation
8. Multiple matches handling (each `on_multiple_matches` value)
9. Encoding support (UTF-8, Latin-1)
10. Symlink following
11. Error cases (invalid regex, permission denied, broken symlink)

**Alternatives Considered**:
- Only integration tests → Rejected: too slow, poor failure localization
- Mocking file system → Rejected: prefer real file operations in temp dir

---

## Summary of Technical Decisions

| Area | Decision | Key Rationale |
|------|----------|---------------|
| File Operations | Atomic write via temp + rename | Safety, prevents corruption |
| Memory Management | Stream processing for large files | Unbounded file size support |
| Regex Engine | Go stdlib `regexp` (RE2) | No dependencies, safe performance |
| Encoding | UTF-8 default + x/text library | Meet FR-001a with minimal deps |
| Interactive Prompts | bufio.Reader with TTY detection | Support FR-004a prompt behavior |
| Backup Format | ISO 8601 timestamp in filename | Chronological sorting, clarity |
| Idempotency | Content comparison before write | Minimize unnecessary filesystem changes |
| Dry-Run | Unified diff generation | Standard, readable format |
| Error Handling | Wrapped errors with context | Clear diagnostics per FR-014, FR-018 |
| Symlinks | Follow by default (no config) | Per clarification decision |
| Testing | Table-driven + integration | Comprehensive coverage, Go idiom |

---

**All NEEDS CLARIFICATION Resolved**: ✅  
**Ready for Phase 1**: ✅
