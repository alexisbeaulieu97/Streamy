# Research & Technical Decisions

**Feature**: Extend Plugin Contract with Verify Lifecycle  
**Date**: October 4, 2025

## Executive Summary

This document captures the technical research, architectural decisions, and alternatives considered for implementing a verification lifecycle in Streamy's plugin contract. The verification system enables read-only state inspection, drift detection, and intelligent step skipping during apply operations.

---

## 1. Verification Status Model

### Decision: Five-Status Enum
We adopt a five-value verification status model: `satisfied`, `missing`, `drifted`, `blocked`, and `unknown`.

### Rationale
- **satisfied**: Clear positive signal that step can be safely skipped
- **missing**: Resource doesn't exist, requires creation (unambiguous)
- **drifted**: Exists but differs from expected state, needs correction
- **blocked**: Cannot verify due to external factors (permission, dependency failure)
- **unknown**: Verification not possible/deterministic (e.g., command without check clause)

The five-status model provides sufficient granularity for all plugin types while remaining simple and mutually exclusive.

### Alternatives Considered
1. **Four-status model (satisfied/missing/drifted/blocked)**
   - Rejected: No safe default for non-verifiable operations like arbitrary commands
   - Would force plugins to choose between "satisfied" (unsafe, skips needed work) or "drifted" (misleading semantics)

2. **Three-status model (ok/changed/error)**
   - Rejected: Insufficient granularity for dependency handling and drift detection
   - Cannot distinguish "doesn't exist" from "exists but wrong"

3. **Seven-status model adding "partial" and "corrupted"**
   - Rejected: Over-engineering; "drifted" adequately covers both cases
   - Adds complexity without clear user benefit

---

## 2. Plugin Interface Extension

### Decision: Add `Verify()` Method to Existing Plugin Interface

```go
type Plugin interface {
    Metadata() Metadata
    Schema() interface{}
    Check(ctx context.Context, step *config.Step) (bool, error)  // Existing
    Apply(ctx context.Context, step *config.Step) (*model.StepResult, error)
    DryRun(ctx context.Context, step *config.Step) (*model.StepResult, error)
    Verify(ctx context.Context, step *config.Step) (*model.VerificationResult, error)  // NEW
}
```

### Rationale
- Maintains consistency with existing methods (same signature pattern)
- Context-aware for cancellation and timeout support
- Returns structured result type for richer reporting (not just boolean)
- Backward compatible: existing plugins fail compilation (intentional) until updated

### Alternatives Considered
1. **Reuse existing `Check()` method**
   - Rejected: `Check()` returns `(bool, error)` which is insufficient for five-status model
   - Would require breaking change to `Check()` signature affecting all existing code
   - `Check()` semantics are idempotency-focused, not state inspection

2. **Separate optional interface (e.g., `Verifiable`)**
   - Rejected: Creates two plugin tiers and complicates registry logic
   - Users would have unclear expectations about which plugins support verification
   - Core architectural principle: all plugins should support all lifecycle phases

3. **Make verification external to plugins (inspection framework)**
   - Rejected: Plugins have domain knowledge needed for accurate verification
   - Would require duplicating plugin logic in separate verification layer
   - Violates plugin-centric architecture principle

---

## 3. Verification Result Structure

### Decision: New `VerificationResult` Type

```go
type VerificationResult struct {
    StepID    string
    Status    VerificationStatus  // satisfied/missing/drifted/blocked/unknown
    Message   string              // Human-readable explanation
    Details   string              // Full diff output for drifted status
    Error     error               // Non-nil for blocked status
    Duration  time.Duration
    Timestamp time.Time
}
```

### Rationale
- Parallel structure to existing `StepResult` for consistency
- `Details` field supports full diff output requirement (FR-013a)
- Separates short message (for tables) from detailed diff (for verbose mode)
- Duration tracking enables performance monitoring per verification type

### Alternatives Considered
1. **Reuse `StepResult` with status mapping**
   - Rejected: Conflates verification and execution results
   - `StepResult` has execution-specific fields (e.g., `StatusRunning`) that don't apply
   - Separate types improve type safety and code clarity

2. **Simple struct with just status and error**
   - Rejected: Insufficient for reporting requirements
   - Cannot provide diff output (FR-013a) or user-friendly messages

---

## 4. Timeout Implementation Strategy

### Decision: Per-Step Configurable Timeout (Default 30s)

Add optional `verify_timeout` field to step configuration:

```yaml
- id: large-repo-check
  type: repo
  verify_timeout: 120s  # Override for slow operations
```

### Rationale
- Default 30s balances responsiveness and coverage for typical operations
- Per-step override accommodates edge cases (large files, network operations)
- Context deadline propagation already implemented in existing plugin methods
- Timeout violations result in "blocked" status with timeout error

### Alternatives Considered
1. **Fixed global timeout**
   - Rejected: No flexibility for legitimately slow operations
   - Would force either too-short (many false failures) or too-long (poor UX) timeout

2. **Plugin-specific timeout configuration**
   - Rejected: Users want control at step level, not plugin level
   - Two repo verification steps might need different timeouts

3. **No timeout (rely on context cancellation)**
   - Rejected: Verification should complete quickly; hanging operations indicate problems
   - Safety mechanism to prevent runaway verification checks

---

## 5. CLI Command Structure

### Decision: Standalone `verify` Command

```bash
streamy verify config.yaml [--verbose] [--json]
```

### Rationale
- Parallel to `apply` command for consistency
- Clear separation of concerns: verify = read-only audit, apply = state changes
- `--verbose` flag shows detailed diff output (FR-013a)
- `--json` flag enables automation/scripting
- Exit code 0 only when all steps satisfied (FR-014)

### Alternatives Considered
1. **Flag on apply command (`streamy apply --dry-run-verify`)**
   - Rejected: Conflates two distinct operations
   - Users want verification without implying apply intent
   - Dry-run already has specific semantics (preview changes)

2. **Separate `check` subcommand**
   - Rejected: "check" term already used in existing `Check()` method
   - Terminology confusion between verification and idempotency checks

3. **Always verify before apply (automatic)**
   - Rejected: Adds latency to every apply operation
   - Users should explicitly opt into verification-first workflow
   - Future: Consider `--verify-first` flag on apply command

---

## 6. Apply Integration Strategy

### Decision: Optional Verification Phase Before Apply

Introduce `--verify-first` flag (future enhancement):

```bash
streamy apply --verify-first config.yaml
```

Behavior: Run verification, skip satisfied steps, apply only missing/drifted/unknown/blocked.

### Rationale
- Opt-in for backward compatibility
- Reduces redundant work on repeated apply operations
- Verification results cached in executor for same-run apply decisions
- Clear performance benefit: skip satisfied steps immediately

### Alternatives Considered
1. **Always verify before apply (automatic)**
   - Deferred: Adds latency; should be opt-in initially
   - Future: Could become default with `--no-verify` escape hatch

2. **Separate verify + apply commands (manual workflow)**
   - Rejected: Poor UX for common "fix drift" workflow
   - Verification results not reused, duplicating inspection work

3. **Implicit verification via existing `Check()` method**
   - Rejected: `Check()` is idempotency test, not state inspection
   - Would require breaking changes to `Check()` semantics

---

## 7. Diff Output Format (Drifted Status)

### Decision: Unified Diff Format (Similar to `git diff`)

Example output for drifted template file:

```diff
--- expected: templates/app.conf.tmpl (rendered)
+++ actual: /etc/app.conf
@@ -1,3 +1,3 @@
 APP_NAME=Streamy
-ENVIRONMENT=production
+ENVIRONMENT=development
 DEBUG_MODE=false
```

### Rationale
- Familiar format for developers (git, patch, diff tools)
- Clear indication of expected vs actual state
- Line-by-line granularity for precise drift identification
- Works for text files (templates, line-in-file, config files)

### Alternatives Considered
1. **Simple "differs" message**
   - Rejected: Insufficient detail for debugging (FR-013a requirement)
   - Users need to know *what* changed, not just *that* it changed

2. **JSON structured diff**
   - Rejected: Less readable for human consumption
   - Reserve JSON output for `--json` flag

3. **Custom domain-specific formats per plugin**
   - Rejected: Inconsistent UX across plugins
   - Unified format provides predictability

---

## 8. Plugin-Specific Verification Logic

### Package Plugin

**Decision**: Query system package manager

```go
func (p *PackagePlugin) Verify(ctx context.Context, step *config.Step) (*model.VerificationResult, error) {
    for _, pkg := range step.Package.Packages {
        installed, version := p.queryPackage(pkg.Name)
        if !installed {
            return &model.VerificationResult{Status: model.StatusMissing}, nil
        }
        if pkg.Version != "" && version != pkg.Version {
            return &model.VerificationResult{Status: model.StatusDrifted}, nil
        }
    }
    return &model.VerificationResult{Status: model.StatusSatisfied}, nil
}
```

**Rationale**: Existing package managers (apt, brew, etc.) provide query interfaces.

---

### Symlink Plugin

**Decision**: Compare `readlink` output to expected source

```go
func (p *SymlinkPlugin) Verify(ctx context.Context, step *config.Step) (*model.VerificationResult, error) {
    actual, err := os.Readlink(step.Symlink.Target)
    if os.IsNotExist(err) {
        return &model.VerificationResult{Status: model.StatusMissing}, nil
    }
    if err != nil {
        return &model.VerificationResult{Status: model.StatusBlocked, Error: err}, nil
    }
    if actual == step.Symlink.Source {
        return &model.VerificationResult{Status: model.StatusSatisfied}, nil
    }
    return &model.VerificationResult{Status: model.StatusDrifted}, nil
}
```

**Rationale**: Symlink verification is deterministic and fast (single syscall).

---

### Template Plugin

**Decision**: Render template in-memory, compare checksums

```go
func (p *TemplatePlugin) Verify(ctx context.Context, step *config.Step) (*model.VerificationResult, error) {
    rendered, err := p.renderTemplate(step.Template)
    if err != nil {
        return &model.VerificationResult{Status: model.StatusBlocked, Error: err}, nil
    }
    
    actual, err := os.ReadFile(step.Template.Destination)
    if os.IsNotExist(err) {
        return &model.VerificationResult{Status: model.StatusMissing}, nil
    }
    
    if bytes.Equal(rendered, actual) {
        return &model.VerificationResult{Status: model.StatusSatisfied}, nil
    }
    
    diff := generateUnifiedDiff(rendered, actual)
    return &model.VerificationResult{Status: model.StatusDrifted, Details: diff}, nil
}
```

**Rationale**: Full content comparison ensures accuracy; checksum alone insufficient for diff output.

---

### Command Plugin

**Decision**: Run optional `verify` command if specified, else return `unknown`

```yaml
- id: service-running
  type: command
  command: systemctl start myservice
  verify: systemctl is-active myservice  # Optional verification command
```

```go
func (p *CommandPlugin) Verify(ctx context.Context, step *config.Step) (*model.VerificationResult, error) {
    if step.Command.Verify == "" {
        return &model.VerificationResult{Status: model.StatusUnknown}, nil
    }
    
    err := p.runCommand(ctx, step.Command.Verify)
    if err != nil {
        return &model.VerificationResult{Status: model.StatusMissing}, nil
    }
    return &model.VerificationResult{Status: model.StatusSatisfied}, nil
}
```

**Rationale**: Commands are often non-idempotent; explicit verify command gives users control. Unknown status forces re-application for safety.

---

## 9. Error Handling & Transient Failures

### Decision: Fail Fast with "Blocked" Status

When verification encounters errors (permission denied, network timeout, file locks):
- Report step as `blocked` with error details
- Continue verifying remaining steps (FR-022a)
- Do not retry (user can re-run verify when issue resolves)

### Rationale
- Simple, predictable behavior
- Avoids masking real problems with automatic retries
- Users control retry timing (e.g., after fixing permissions)
- Consistent with "verification is read-only" principle

### Alternatives Considered
1. **Automatic retry with exponential backoff**
   - Rejected: Adds complexity and latency
   - Transient failures (file locks) often require external resolution

2. **Skip failed steps silently**
   - Rejected: Hides problems; user unaware of incomplete verification

3. **Fail entire verify run on first error**
   - Rejected: User wants complete picture of system state
   - Single blocked step shouldn't prevent auditing remaining steps

---

## 10. Performance Considerations

### Decision: Parallel Verification Where Safe

- Verification respects DAG dependencies (verify prerequisites first)
- Independent steps verified in parallel (same as apply execution)
- Timeout per step prevents runaway checks
- Verification results cached for reuse in same process (future `--verify-first`)

### Rationale
- Leverages existing DAG parallelization infrastructure
- Fast feedback (target <5s for typical 50-step configs)
- Consistent with apply execution model

### Alternatives Considered
1. **Serial verification only**
   - Rejected: Unnecessarily slow for large configs
   - Verification is read-only, safe to parallelize

2. **Always parallel (ignore dependencies)**
   - Rejected: Blocked status depends on prerequisite verification results
   - Cannot determine if B is blocked without verifying A first

---

## 11. Backward Compatibility

### Decision: Breaking Change Requiring Plugin Updates

All existing plugins must implement `Verify()` to compile. This is acceptable because:
- Streamy is pre-1.0 (rapid iteration phase)
- All plugins are internal/built-in currently
- Clear migration path: implement per-plugin verification logic
- Temporary stub implementations possible (`return StatusUnknown`)

### Future Compatibility
- External plugins (post-1.0): semantic versioning enforced
- Breaking plugin API changes require major version bump
- Migration guide provided for plugin developers

---

## 12. Testing Strategy

### Unit Tests
- Each plugin: test all five status paths (satisfied/missing/drifted/blocked/unknown)
- Timeout handling: verify context cancellation propagates correctly
- Diff generation: verify unified diff format correctness

### Integration Tests
- End-to-end verify command with multi-step configs
- Dependency blocking: verify B blocked when A is missing
- Apply integration: verify satisfied steps skipped

### Contract Tests
- All plugins implement Plugin interface (compile-time guarantee)
- Verification result structure consistency

---

## 13. Documentation Requirements

### User Documentation
- `streamy verify` command reference
- Status meaning definitions (satisfied/missing/drifted/blocked/unknown)
- Per-plugin verification behavior explanations
- Troubleshooting guide for blocked statuses

### Developer Documentation
- Plugin developer guide: implementing `Verify()`
- Best practices: timeout tuning, diff formatting
- Contract test examples

---

## Summary of Key Decisions

| Decision Area | Choice | Primary Rationale |
|---------------|--------|-------------------|
| Status Model | 5 statuses (satisfied/missing/drifted/blocked/unknown) | Sufficient granularity, mutually exclusive |
| Interface Extension | Add `Verify()` method | Consistent with existing lifecycle methods |
| Result Type | New `VerificationResult` struct | Supports rich reporting (diff, messages, timing) |
| Timeout | 30s default, per-step configurable | Balance responsiveness and flexibility |
| CLI Command | Standalone `verify` command | Clear separation from apply |
| Apply Integration | Optional `--verify-first` flag (future) | Opt-in optimization, backward compatible |
| Diff Format | Unified diff (git-style) | Familiar, precise, text-friendly |
| Error Handling | Fail fast, report blocked, continue | Simple, transparent, user-controlled |
| Parallelization | Parallel within DAG constraints | Fast feedback, consistent with apply |
| Backward Compatibility | Breaking change pre-1.0 | Acceptable in rapid iteration phase |

---

## Open Questions / Future Enhancements

1. **Verification result caching**: Should verify results persist across runs for performance?
   - Deferred: In-memory caching sufficient for `--verify-first` workflow
   - Persistent cache adds complexity (invalidation, storage)

2. **Partial verification**: Should users be able to verify specific steps?
   - Deferred: Always-verify-all is simpler for MVP
   - Future: Add step filtering if strong user demand

3. **Verification hooks**: Should plugins support pre/post-verification hooks?
   - Deferred: No clear use case identified yet
   - Revisit if plugin developers request it

4. **TUI integration**: How should verification results display in interactive mode?
   - Deferred: TUI enhancement is separate feature
   - Future: Pre-select drifted/missing steps in interactive mode

---

**Document Status**: Complete  
**Next Phase**: Phase 1 - Design contracts and data model
