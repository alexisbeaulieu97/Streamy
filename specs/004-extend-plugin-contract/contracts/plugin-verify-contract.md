# Plugin.Verify() Method Contract

**Feature**: Extend Plugin Contract with Verify Lifecycle  
**Contract Version**: 1.0.0

## Purpose

This contract defines the behavioral requirements for the `Verify()` method that all Streamy plugins must implement. The method performs read-only inspection of system state to determine alignment with declared configuration.

---

## Method Signature

```go
Verify(ctx context.Context, step *config.Step) (*model.VerificationResult, error)
```

### Parameters

**`ctx context.Context`**
- Standard Go context for cancellation and timeout propagation
- MUST be respected by implementations
- Plugins MUST check for cancellation during long-running operations
- Timeout deadline enforced by executor based on step's `verify_timeout` configuration

**`step *config.Step`**
- Parsed step configuration; plugin-specific validation occurs during `DecodeConfig`
- Contains plugin-specific configuration via `rawConfig` (decode with `step.DecodeConfig(&config.PackageStep{})`, `step.DecodeConfig(&config.SymlinkStep{})`, etc.)
- MUST NOT be modified by verification logic
- All fields accessible per plugin type

### Return Values

**`*model.VerificationResult`**
- MUST be non-nil on successful verification execution
- Contains verification status, message, optional diff, and timing
- Structure defined in [data-model.md](../data-model.md)

**`error`**
- MUST be nil when verification completes (even if status is blocked)
- MUST be non-nil only for:
  - Context cancellation (`context.Canceled`)
  - Timeout/deadline exceeded (`context.DeadlineExceeded`)
  - Unexpected infrastructure failures (panic recovery, null pointer, etc.)
- Expected verification failures (missing resources, permission errors, decode/validation errors) MUST be represented via `VerificationResult.Status` and `VerificationResult.Error` while returning a nil Go error

---

## Behavioral Requirements

### BR-001: Read-Only Guarantee (CRITICAL)

**Requirement**: The `Verify()` method MUST NOT modify any system state.

**Forbidden Operations**:
- Writing, creating, or deleting files
- Installing or removing packages
- Modifying symlinks, permissions, or ownership
- Executing commands that change state
- Writing to databases or external services

**Allowed Operations**:
- Reading files (content, permissions, ownership)
- Querying package installation status
- Reading symlink targets
- Checking file/directory existence
- Running read-only commands (e.g., `git status`, `systemctl is-active`)
- Network queries (version checks, API reads)

**Enforcement**: Integration tests MUST verify no side effects.

---

### BR-002: Context Respect (REQUIRED)

**Requirement**: Implementations MUST respect context cancellation and deadlines.

**Behavior**:
```go
select {
case <-ctx.Done():
    return nil, ctx.Err()
case result := <-verificationCompleted:
    return result, nil
}
```

**Timeout Handling**:
- If `ctx.Deadline()` exceeded, return immediately
- Cleanup any in-flight operations before returning
- Do not continue verification after cancellation

**Testing**: Contract tests MUST verify cancellation handling.

---

### BR-003: Status Accuracy (REQUIRED)

**Requirement**: Return status MUST accurately reflect system state.

**Status Determination**:

| Status | When to Return |
|--------|----------------|
| `satisfied` | Current state **exactly** matches expected configuration |
| `missing` | Required resource **does not exist** (file, package, repo) |
| `drifted` | Resource **exists but differs** (wrong content, version, branch) |
| `blocked` | **Cannot determine** status due to error (permission denied, network timeout, dependency failure) |
| `unknown` | **Verification not possible** (e.g., command plugin without verify clause) |

**Precision Requirements**:
- Do not return `satisfied` when state is merely "close enough"
- Use `drifted` for any detectable difference, no matter how small
- Reserve `blocked` for genuine errors, not missing resources
- Use `unknown` only when verification is fundamentally impossible

---

### BR-004: Message Clarity (REQUIRED)

**Requirement**: `VerificationResult.Message` MUST provide human-readable explanation.

**Message Guidelines**:
- Concise (1-2 sentences, <100 characters preferred)
- State-focused (describe what was found, not what was expected)
- Actionable (suggest next step when status is not satisfied)

**Examples**:
- ✅ `"package git is installed (version 2.39.0)"`
- ✅ `"repository not found at /opt/myrepo"`
- ✅ `"symlink points to /usr/bin/python3.9 (expected /usr/bin/python3.11)"`
- ✅ `"permission denied reading /etc/shadow"`
- ✅ `"no verification command specified for this step"`
- ❌ `"OK"` (too vague)
- ❌ `"failed"` (not descriptive)

---

### BR-005: Diff Generation (REQUIRED for drifted status)

**Requirement**: When `Status == StatusDrifted` and resource is text-based, `Details` field MUST contain unified diff.

**Diff Format**:
```diff
--- expected: /path/to/source
+++ actual: /path/to/destination
@@ -line,count +line,count @@
 context
-removed line
+added line
 context
```

**Diff Requirements**:
- Use unified diff format (compatible with `patch` tool)
- Include 3 lines of context before/after changes
- Label sources clearly (expected vs actual)
- Truncate diff if >10,000 lines (add "... truncated ..." marker)

**Non-Text Resources**:
- For binary files: Message includes "binary files differ"
- For symbolic links: Message shows "points to X (expected Y)"
- For packages: Message shows "version X installed (expected Y)"

**Exemption**: If diff generation fails or resource is non-textual, populate `Details` with explanatory message.

---

### BR-006: Error Propagation (REQUIRED for blocked status)

**Requirement**: When `Status == StatusBlocked`, `VerificationResult.Error` MUST be populated with underlying error.

**Error Wrapping**:
```go
return &VerificationResult{
    StepID:  step.ID,
    Status:  StatusBlocked,
    Message: "permission denied reading /etc/shadow",
    Error:   fmt.Errorf("stat /etc/shadow: %w", err),
}, nil
```

**Error Context**:
- Wrap original error with context
- Include file path, operation, or resource identifier
- Preserve error chain for debugging

**Testing**: Error messages MUST be tested for clarity.

---

### BR-007: Performance Bounds (REQUIRED)

**Requirement**: Verification MUST complete within configured timeout or context deadline.

**Default Timeout**: 30 seconds (configurable per-step via `verify_timeout`)

**Performance Guidelines**:
- Target <100ms for simple checks (file existence, symlink read)
- Target <1s for medium checks (template rendering, package queries)
- Target <30s for complex checks (large file comparisons, network operations)

**Optimization Strategies**:
- Use checksums/hashes instead of full content comparison when possible
- Stream large files rather than reading entirely into memory
- Short-circuit early on first difference detected
- Cache expensive queries within single verification run (e.g., `apt list --installed`)

**Timeout Violation**: If timeout exceeded, context cancellation triggers, plugin returns immediately with timeout error.

---

### BR-008: Idempotency (REQUIRED)

**Requirement**: Multiple `Verify()` calls with same inputs MUST produce same result (assuming system state unchanged).

**Determinism**:
- No randomness in verification logic
- No time-based checks (timestamps, "recently modified", etc.)
- No reliance on global mutable state

**Exceptions**:
- External state changes between calls (file modified, package updated) MAY change result
- Network resources MAY return different results due to external changes

**Testing**: Contract tests MUST call `Verify()` twice and assert result equality.

---

### BR-009: Dependency Awareness (RECOMMENDED)

**Requirement**: Plugins SHOULD account for dependency failures when determining status.

**Scenario**: Step B depends on Step A.
- If A verification returns `missing` or `blocked`, B verification MAY return `blocked` with message "dependency step-A failed verification".
- Alternatively, executor MAY skip B verification entirely and mark it `blocked`.

**Implementation**: Executor handles dependency logic; plugins focus on local state verification.

---

### BR-010: Logging (RECOMMENDED)

**Requirement**: Plugins SHOULD emit structured logs during verification.

**Log Levels**:
- DEBUG: Detailed verification steps ("checking file existence", "querying package manager")
- INFO: Verification outcome ("step satisfied", "step drifted")
- WARN: Unexpected but recoverable conditions ("file permissions looser than expected")
- ERROR: Verification failures ("permission denied", "timeout exceeded")

**Structured Fields**:
- `step_id`: Step being verified
- `plugin`: Plugin name
- `status`: Verification status
- `duration_ms`: Verification duration

**Example**:
```go
cfg := config.SymlinkStep{}
if err := step.DecodeConfig(&cfg); err != nil {
    return &model.VerificationResult{
        StepID:  step.ID,
        Status:  model.StatusBlocked,
        Message: "invalid symlink configuration",
        Error:   fmt.Errorf("decode config: %w", err),
    }, nil
}

log.Debug().
    Str("step_id", step.ID).
    Str("plugin", "symlink").
    Str("target", cfg.Target).
    Msg("checking symlink target")
```

---

## Plugin-Specific Contracts

### Package Plugin

**Verification Logic**:
1. Decode configuration: `var cfg config.PackageStep; err := step.DecodeConfig(&cfg)`
2. If decoding fails, return `blocked` with message `invalid package configuration` (include decode error in `Error`)
3. Query system package manager (`apt list --installed`, `brew list`, etc.)
4. For each package in `cfg.Packages`:
   - If not installed: return `missing`
   - If version specified and doesn't match: return `drifted` with message
5. If all packages match: return `satisfied`

**Status Examples**:
- `satisfied`: "packages git, curl installed"
- `missing`: "package git not found"
- `drifted`: "package git version 2.39.0 installed (expected 2.40.0)"

---

### Symlink Plugin

**Verification Logic**:
1. Decode configuration: `var cfg config.SymlinkStep; err := step.DecodeConfig(&cfg)`
2. If decoding fails, return `blocked` with message `invalid symlink configuration` (include the decode error in `Error`)
3. Call `os.Readlink(cfg.Target)`
4. If error is `os.ErrNotExist`: return `missing`
5. If other error (permission, loop): return `blocked`
6. Compare readlink result to `cfg.Source`
7. If match: return `satisfied`; otherwise return `drifted`

**Status Examples**:
- `satisfied`: "symlink /usr/bin/python points to /usr/bin/python3.11"
- `missing`: "symlink /usr/bin/python does not exist"
- `drifted`: "symlink points to /usr/bin/python3.9 (expected /usr/bin/python3.11)"
- `blocked`: "permission denied reading symlink /etc/alternatives/editor"

---

### Template Plugin

**Verification Logic**:
1. Render template in-memory using configured variables
2. Read destination file content
3. If destination doesn't exist: return `missing`
4. If read error: return `blocked`
5. Compare rendered content to destination byte-by-byte
6. If identical: return `satisfied`
7. If differ: generate unified diff, return `drifted` with diff in `Details`

**Status Examples**:
- `satisfied`: "rendered template matches /etc/app.conf"
- `missing`: "destination file /etc/app.conf not found"
- `drifted`: "file content differs (3 lines changed)"
- `blocked`: "permission denied reading /etc/app.conf"

**Diff Example** (in `Details` field):
```diff
--- expected: templates/app.conf.tmpl (rendered)
+++ actual: /etc/app.conf
@@ -1,5 +1,5 @@
 APP_NAME=Streamy
-ENVIRONMENT=production
+ENVIRONMENT=development
 DEBUG_MODE=false
```

---

### Command Plugin

**Verification Logic**:
1. Decode configuration: `var cfg config.CommandStep; err := step.DecodeConfig(&cfg)`
2. If decoding fails, return `blocked` with message `invalid command configuration` (include decode error in `Error`)
3. Check if `cfg.Check` (or equivalent verify command) is specified
4. If not specified: return `unknown` with message "no verification command specified"
5. If specified: execute command with timeout
6. If exit code 0: return `satisfied`
7. If exit code non-zero: return `missing` (resource not in expected state)
8. If execution error (timeout, not found): return `blocked`

**Status Examples**:
- `satisfied`: "verification command succeeded (exit code 0)"
- `missing`: "verification command failed (exit code 1)"
- `unknown`: "no verification command specified for this step"
- `blocked`: "verification command timed out after 30s"

**Configuration Example**:
```yaml
- id: service-running
  type: command
  command: systemctl start myservice
  verify: systemctl is-active myservice --quiet
```

---

### Repo Plugin

**Verification Logic**:
1. Decode configuration: `var cfg config.RepoStep; err := step.DecodeConfig(&cfg)`
2. If decoding fails, return `blocked` with message `invalid repo configuration` (include decode error in `Error`)
3. Check if directory exists at `cfg.Destination`
4. If not exists: return `missing`
5. If exists but not a git repo: return `blocked` (or `drifted` depending on design)
6. Query remote URL: `git config --get remote.origin.url`
7. If doesn't match `cfg.URL`: return `drifted`
8. Query current branch: `git rev-parse --abbrev-ref HEAD`
9. If doesn't match `cfg.Branch`: return `drifted`
10. If all match: return `satisfied`

**Status Examples**:
- `satisfied`: "repository at /opt/myrepo on branch main (remote: github.com/user/repo)"
- `missing`: "directory /opt/myrepo does not exist"
- `drifted`: "repository on branch develop (expected main)"
- `blocked`: "permission denied accessing /opt/myrepo/.git"

---

## Contract Test Requirements

All plugins MUST pass the following contract tests:

### Test: Read-Only Verification
```go
func TestVerifyReadOnly(t *testing.T) {
    // Capture initial system state
    before := captureState()
    
    // Run verification
    plugin.Verify(ctx, step)
    
    // Assert state unchanged
    after := captureState()
    assert.Equal(t, before, after)
}
```

### Test: Context Cancellation
```go
func TestVerifyCancellation(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    cancel() // Cancel immediately
    
    result, err := plugin.Verify(ctx, step)
    assert.Error(t, err)
    assert.ErrorIs(t, err, context.Canceled)
}
```
