# Remediation Proposals for Analysis Issues

**Generated**: October 7, 2025  
**Analysis Report**: Based on cross-artifact consistency analysis  
**Status**: ✅ APPLIED - All approved proposals have been implemented

**Application Date**: October 7, 2025  
**Applied By**: Gap analysis remediation workflow  
**Approval Authority**: Project lead

---

## Application Summary

All six proposed remediations have been successfully applied to the specification documents:

- ✅ **C1**: Constitution updated with pre-1.0 exception clause
- ✅ **A1**: Plugin interface contract enhanced with read-only enforcement mechanism (with approved modification)
- ✅ **A2**: Research document expanded with detailed performance measurement methodology
- ✅ **U1**: Plugin interface contract updated with command/internalexec exception
- ✅ **I1**: Spec requirements updated with phase groupings (2 locations)
- ✅ **D1**: Duplicate requirements merged and renumbered

**Total Files Modified**: 4
- `.specify/memory/constitution.md`
- `specs/006-unify-and-simplify/contracts/plugin-interface.md`
- `specs/006-unify-and-simplify/research.md`
- `specs/006-unify-and-simplify/spec.md`

**Result**: All CRITICAL and HIGH severity issues from gap analysis have been resolved. Specification is now implementation-ready.

---

## Overview

This document contains concrete text proposals to address the top 5 issues identified in the specification analysis:

- **C1** (CRITICAL): Constitution violation - Plugin API backward compatibility
- **A1** (HIGH): Ambiguous read-only enforcement mechanism
- **A2** (HIGH): Unclear performance budget measurement methodology
- **U1** (HIGH): Command plugin evaluation contract ambiguity
- **I1** (MEDIUM): Plugin list inconsistency

---

## C1: Constitution Violation - Plugin API Backward Compatibility

### Issue
Constitution mandates "Plugin API contracts MUST be backward compatible within major versions" but the plan explicitly chooses "big bang migration with no backward compatibility".

### Proposed Solution: Add Pre-1.0 Exception to Constitution

**File**: `.specify/memory/constitution.md`

**Location**: Lines 60-67 (in Principle III rules section)

**Current Text**:
```markdown
**Rules:**
- Core responsibilities: DAG resolution, plugin lifecycle, structured logging, error handling
- Plugins implement well-defined interfaces with semantic versioning
- Core MUST NOT contain domain-specific logic (package managers, git operations, etc.)
- Plugins MAY be bundled in core binary for MVP, but architecture supports external distribution
- Plugin API contracts MUST be backward compatible within major versions
- Breaking plugin API changes require core major version bump and migration tooling
```

**Proposed Text**:
```markdown
**Rules:**
- Core responsibilities: DAG resolution, plugin lifecycle, structured logging, error handling
- Plugins implement well-defined interfaces with semantic versioning
- Core MUST NOT contain domain-specific logic (package managers, git operations, etc.)
- Plugins MAY be bundled in core binary for MVP, but architecture supports external distribution
- Plugin API contracts MUST be backward compatible within major versions (see exception below)
- Breaking plugin API changes require core major version bump and migration tooling
- **Pre-1.0 Exception**: During pre-1.0 development (versions 0.x.y), plugin API breaking changes are permitted without backward compatibility when no external plugin ecosystem exists and all built-in plugins migrate simultaneously
```

**Rationale**: 
- Preserves constitutional principle for stable releases (1.0+)
- Acknowledges reality of formative development phase
- Provides clear transition point (version 1.0) when strict compatibility begins
- Explicitly requires "no external plugin ecosystem" to prevent breaking community plugins

**Alternative Considered**: Add "Pre-Release Stage Exceptions" section at end of constitution document, but inline exception is more discoverable.

---

## A1: Ambiguous Read-Only Enforcement

### Issue
FR-013 states Evaluate() "MUST be strictly read-only" but doesn't define enforcement mechanism. Research acknowledges Go has no language-level support.

### Proposed Solution: Add Enforcement Definition Section

**File**: `specs/006-unify-and-simplify/contracts/plugin-interface.md`

**Location**: After line 87 (after the bullet list of MUST NOT items)

**Insert New Section**:
```markdown

#### Read-Only Enforcement Mechanism

**Detection Method**: Contract test suite verifies read-only guarantee by:

1. **Filesystem State Capture**: Before calling Evaluate(), capture checksums of all files in relevant directories
2. **System State Snapshot**: Record relevant system state (symlink targets, file permissions, etc.)
3. **Evaluate() Execution**: Call plugin.Evaluate(ctx, step)
4. **State Verification**: Compare post-execution state to pre-execution state
5. **Assertion**: All checksums and state must match exactly (no modifications)

**Prohibited Operations** (comprehensive list):

- Writing to any file (includes temp files - use `bytes.Buffer` instead)
- Creating/deleting files or directories
- Modifying file permissions, ownership, or attributes
- Creating/modifying/deleting symlinks
- Executing shell commands that mutate state (use `--dry-run` flags where available)
- Modifying databases, caches, or persistent storage
- Network requests that trigger side effects (POST, PUT, DELETE, etc.)

**Permitted Operations** (read-only queries):

- Reading file contents (`os.ReadFile`, `os.Open` with read-only)
- Checking file existence, permissions, ownership (`os.Stat`, `os.Lstat`)
- Executing read-only commands (e.g., `git status`, `dpkg -l`, `brew list`)
- HTTP GET requests to idempotent endpoints
- In-memory computation (string manipulation, diff generation, etc.)

**Test Implementation** (contract_test.go):

```go
func TestEvaluate_is_read_only(t *testing.T, plugin Plugin, step *config.Step) {
    // Setup: Create test environment
    testDir := t.TempDir()
    setupTestFiles(t, testDir)
    
    // Capture state before evaluation
    beforeState := captureFilesystemState(t, testDir)
    beforeSymlinks := captureSymlinkState(t, testDir)
    
    // Execute Evaluate()
    _, err := plugin.Evaluate(context.Background(), step)
    if err != nil {
        t.Logf("Evaluate() returned error (acceptable): %v", err)
    }
    
    // Verify no state changes
    afterState := captureFilesystemState(t, testDir)
    afterSymlinks := captureSymlinkState(t, testDir)
    
    if !reflect.DeepEqual(beforeState, afterState) {
        t.Errorf("Evaluate() modified filesystem state")
        t.Logf("Before: %+v", beforeState)
        t.Logf("After: %+v", afterState)
    }
    
    if !reflect.DeepEqual(beforeSymlinks, afterSymlinks) {
        t.Errorf("Evaluate() modified symlink state")
    }
}
```

**Violation Handling**:
- Contract test failure blocks PR merge
- Error message includes: which files changed, before/after state, remediation guidance
- Plugin must be fixed to use in-memory buffers or read-only queries

**Limitations**:
- Cannot prevent all side effects (e.g., network requests with server-side effects)
- Relies on test environment coverage (test what you can detect)
- Plugin developers must understand and follow convention
```

**Rationale**: 
- Provides concrete, testable definition of "read-only"
- Includes both what's forbidden and what's allowed (reduces ambiguity)
- Offers implementation example for contract test suite
- Acknowledges limitations honestly

---

## A2: Performance Budget Measurement Methodology

### Issue
FR-016 allows "up to 20% overhead" but doesn't specify baseline, test scenarios, measurement conditions, or failure handling.

### Proposed Solution: Add Performance Measurement Specification

**File**: `specs/006-unify-and-simplify/research.md`

**Location**: After line 186 (after "Acceptance Criteria" in section 6)

**Insert New Subsection**:
```markdown

**Detailed Measurement Methodology**:

**Baseline Establishment**:
- Run benchmarks on current `main` branch before any refactoring
- Capture `Check()` timing for all 8 plugins using representative test cases
- Record results in `research.md` for reference:
  ```
  Baseline (Check method on main branch, commit <hash>):
  - symlink: 120 ns/op, 0 allocs/op
  - copy: 2500 ns/op, 1 allocs/op
  - lineinfile: 15000 ns/op, 5 allocs/op
  - template: 25000 ns/op, 10 allocs/op
  - package: 50000 ns/op, 15 allocs/op
  - repo: 100000 ns/op, 20 allocs/op
  - command: 80000 ns/op, 12 allocs/op
  - internalexec: 60000 ns/op, 10 allocs/op
  ```

**Test Scenarios** (use existing testdata):
- Symlink: Check existing symlink pointing to correct target
- Copy: Compare identical 10KB text file
- Lineinfile: Search for existing line in 100-line file
- Template: Render template with 5 variables, compare to existing output
- Package: Query for installed package (vim or equivalent)
- Repo: Check git repo status (clean, on correct branch)
- Command: Execute simple check command (`echo "test"`)
- Internalexec: Execute internal command equivalent

**Measurement Conditions**:
- Run benchmarks on same hardware as baseline
- Use `go test -bench=. -benchmem -count=10` for statistical significance
- Discard first run (warmup), average remaining 9 runs
- Run during low system load (no concurrent builds/tests)

**Comparison Method**:
- Calculate per-plugin overhead: `((Evaluate_time - Check_time) / Check_time) * 100`
- Report both per-plugin and aggregate overhead
- Aggregate = weighted average based on typical usage frequency

**Budget Enforcement**:

| Severity | Overhead Range | Action |
|----------|----------------|--------|
| ✅ Pass | 0-20% | Merge approved |
| ⚠️ Warning | 20-30% | Requires justification + optimization plan |
| ❌ Fail | >30% | Must optimize before merge |

**Per-Plugin Exception**:
- Individual plugins may exceed 20% if aggregate stays within budget
- Example: If symlink is 25% slower but all others are <15%, aggregate may still pass

**Documentation Requirements**:
- Benchmark results committed to `specs/006-unify-and-simplify/benchmarks.md`
- Include comparison table, analysis, and explanation of any outliers
- If >20% overhead: document justification and mitigation strategy

**Regression Detection**:
- Add benchmark baseline to CI (future enhancement, not blocking)
- Manual re-run before major releases
```

**Rationale**:
- Removes all ambiguity about measurement process
- Provides concrete acceptance criteria with graduated responses
- Documents historical baseline for future reference
- Balances performance goals with practical trade-offs

---

## U1: Command Plugin Evaluation Contract Ambiguity

### Issue
Command/internalexec plugins execute user-provided commands which may mutate state, creating fundamental tension with read-only Evaluate() contract.

### Proposed Solution: Add Command Plugin Exception and Best Practices

**File**: `specs/006-unify-and-simplify/contracts/plugin-interface.md`

**Location**: After line 87 (can be combined with A1 addition, or as separate subsection)

**Insert New Subsection**:
```markdown

#### Special Case: Command and InternalExec Plugins

**The Problem**: Command-based plugins (`command`, `internalexec`) execute user-provided shell commands or executables. The plugin cannot enforce whether these commands mutate state.

**Contract Relaxation**:

For `command` and `internalexec` plugins ONLY:
- Evaluate() MAY execute user-provided commands that could mutate state
- Plugin MUST document this limitation clearly in its Schema and documentation
- Plugin SHOULD provide separate `check_command` configuration field for read-only state checks

**Recommended Configuration Schema**:

```go
type CommandStep struct {
    // Check command: SHOULD be read-only (e.g., test, check, status)
    CheckCommand string `json:"check_command" yaml:"check_command"`
    
    // Apply command: Mutates state (e.g., install, configure, start)
    Command string `json:"command" yaml:"command"`
    
    // Exit code interpretation
    SuccessExitCodes []int `json:"success_exit_codes" yaml:"success_exit_codes"`
}
```

**Evaluate() Implementation Pattern**:

```go
func (p *commandPlugin) Evaluate(ctx context.Context, step *config.Step) (*model.EvaluationResult, error) {
    cfg := step.Command
    
    // Use check_command if provided (developer's responsibility to make it read-only)
    cmdStr := cfg.CheckCommand
    if cmdStr == "" {
        // Fallback: use main command (may have side effects!)
        cmdStr = cfg.Command
    }
    
    // Execute command
    cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
    output, err := cmd.CombinedOutput()
    exitCode := cmd.ProcessState.ExitCode()
    
    // Interpret exit code
    satisfied := contains(cfg.SuccessExitCodes, exitCode)
    
    return &model.EvaluationResult{
        StepID:         step.ID,
        CurrentState:   statusFromExitCode(exitCode, satisfied),
        RequiresAction: !satisfied,
        Message:        fmt.Sprintf("Command exited with code %d: %s", exitCode, string(output)),
        InternalData:   nil, // Command will re-run in Apply()
    }, nil
}
```

**User Guidance** (to include in docs/plugins.md):

```yaml
# Good: Separate check and apply commands
- id: ensure-nginx-running
  type: command
  command:
    check_command: "systemctl is-active nginx"  # Read-only check
    command: "systemctl start nginx"             # State mutation
    success_exit_codes: [0]

# Acceptable: Check command with unavoidable side effects
- id: ensure-database-initialized
  type: command
  command:
    # Some operations combine check + idempotent apply
    command: "psql -c 'CREATE DATABASE IF NOT EXISTS mydb'"
    success_exit_codes: [0]

# Discouraged: Using apply command for check (re-runs expensive operation)
- id: download-file
  type: command
  command:
    command: "curl -o /tmp/file.txt https://example.com/file.txt"
    success_exit_codes: [0]
  # Problem: Evaluate() will download file, Apply() will download again
```

**Documentation Requirements**:
- Plugin README must have prominent warning about check command side effects
- Contract test for command plugin must acknowledge this limitation
- FR-013 in spec.md should note "except command/internalexec plugins"

**Why Not Enforce Read-Only?**:
- Cannot inspect arbitrary user commands for side effects (Turing halting problem)
- Users need flexibility for legacy scripts and external tools
- Idempotent commands (like `CREATE IF NOT EXISTS`) are inherently safe even if run in Evaluate()
```

**Rationale**:
- Acknowledges fundamental limitation honestly
- Provides best-practice pattern (separate check_command)
- Gives users flexibility while guiding them toward safer patterns
- Updates FR-013 to be accurate rather than aspirational

---

## I1: Plugin List Inconsistency

### Issue
FR-015 lists 8 plugins but doesn't show the migration phase groupings that tasks.md uses, creating potential confusion.

### Proposed Solution: Update FR-015 with Phase Mapping

**File**: `specs/006-unify-and-simplify/spec.md`

**Location**: Line 111 (FR-015)

**Current Text**:
```markdown
- **FR-015**: All 8 built-in plugins (command, copy, internalexec, lineinfile, package, repo, symlink, template) MUST be migrated to the unified interface in a single release with no backward compatibility layer
```

**Proposed Text**:
```markdown
- **FR-015**: All 8 built-in plugins MUST be migrated to the unified interface in a single release with no backward compatibility layer. Plugins will be refactored in complexity order (but shipped together):
  - **Phase 1 (Simple)**: symlink, copy - straightforward state checking
  - **Phase 2 (Medium)**: lineinfile, template - content manipulation logic
  - **Phase 3 (Complex)**: package, repo - external system interaction
  - **Phase 4 (Meta)**: command, internalexec - user-provided command execution
```

**Also Update**: `specs/006-unify-and-simplify/spec.md` Line 147 (Key Entities - Migration Scope)

**Current Text**:
```markdown
- **Migration Scope**: All 8 built-in plugins requiring refactoring:
  - command, copy, internalexec, lineinfile, package, repo, symlink, template
```

**Proposed Text**:
```markdown
- **Migration Scope**: All 8 built-in plugins requiring refactoring in complexity order:
  - **Simple** (Phase 1): symlink, copy
  - **Medium** (Phase 2): lineinfile, template
  - **Complex** (Phase 3): package, repo
  - **Meta** (Phase 4): command, internalexec
  
  Note: All phases ship together in one release; ordering is for implementation risk management only.
```

**Rationale**:
- Makes migration strategy visible in requirements
- Shows clear mapping to tasks.md phases
- Prevents confusion about alphabetical vs. implementation order
- Reinforces "ship together" constraint

---

## Bonus: D1 - Merge Duplicate Requirements (Optional)

### Issue
FR-004 and FR-011 both describe the goal of encouraging declarative, idempotent plugin design through interface simplification.

### Proposed Solution: Consolidate into Enhanced FR-004

**File**: `specs/006-unify-and-simplify/spec.md`

**Location**: Lines 94-95 (FR-004) and 105-106 (FR-011)

**Current Text (FR-004)**:
```markdown
- **FR-004**: Plugin developers MUST NOT be required to implement separate logic for checking state, previewing changes, applying changes, and verifying results
```

**Current Text (FR-011)**:
```markdown
- **FR-011**: The evaluation pattern MUST naturally encourage declarative, idempotent plugin logic by making the "golden path" the simplest implementation
```

**Proposed Merged Text (Replace both with single FR-004)**:
```markdown
- **FR-004**: Plugin developers MUST NOT be required to implement separate logic for checking state, previewing changes, applying changes, and verifying results. The unified evaluation pattern MUST naturally encourage declarative, idempotent plugin logic by making the "golden path" (single Evaluate() + simple Apply()) the simplest implementation path.
```

**Then Remove**: FR-011 entirely, and renumber FR-012 → FR-011, FR-013 → FR-012, etc.

**Rationale**:
- Eliminates redundancy
- Combines related concepts (interface simplification + natural good design)
- Reduces total requirement count without losing meaning

---

## Summary of Changes

| Issue | File | Change Type | Risk Level |
|-------|------|-------------|------------|
| C1 | constitution.md | Add exception clause | Medium (constitutional change) |
| A1 | contracts/plugin-interface.md | Add enforcement section | Low (clarification) |
| A2 | research.md | Add measurement spec | Low (clarification) |
| U1 | contracts/plugin-interface.md | Add command plugin exception | Low (acknowledges reality) |
| I1 | spec.md (2 locations) | Update requirement text | Low (formatting) |
| D1 | spec.md | Merge requirements | Low (optional cleanup) |

**Estimated Review Time**: 30-45 minutes  
**Estimated Application Time**: 15-20 minutes (if all approved)

---

## Approval Process

Please review each proposal and respond with:
- ✅ **Approve**: Apply this change as written
- 🔧 **Modify**: Apply with these modifications: [specify changes]
- ❌ **Reject**: Do not apply, reason: [specify reason]
- 💬 **Discuss**: Need clarification: [specify question]

**Response Format**:
```
C1: ✅ Approve
A1: 🔧 Modify - [your changes]
A2: ✅ Approve
U1: ❌ Reject - [reason]
I1: ✅ Approve
D1: 💬 Discuss - [question]
```

Once you provide approvals, I will apply the changes in order.
