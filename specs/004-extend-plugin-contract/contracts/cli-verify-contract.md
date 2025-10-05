# CLI Command Contract: `streamy verify`

**Feature**: Extend Plugin Contract with Verify Lifecycle  
**Contract Version**: 1.0.0

## Purpose

This contract defines the behavior, inputs, outputs, and exit codes for the `streamy verify` command. The command orchestrates verification across all steps in a configuration file and reports results.

---

## Command Syntax

```bash
streamy verify <config-file> [flags]
```

### Arguments

**`<config-file>`** (required)
- Path to YAML configuration file
- Can be relative or absolute
- Must be readable and valid YAML
- Must conform to Streamy configuration schema

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--verbose`, `-v` | bool | false | Show detailed output including diffs for drifted steps |
| `--json` | bool | false | Output results in JSON format (machine-readable) |
| `--timeout` | duration | 30s | Global default timeout for verification (per-step override in config) |

### Examples

```bash
# Basic verification
streamy verify config.yaml

# Verbose mode (show diffs)
streamy verify config.yaml --verbose

# JSON output for automation
streamy verify config.yaml --json

# Custom global timeout
streamy verify config.yaml --timeout=60s
```

---

## Behavior Contract

### BC-001: Configuration Loading

**Requirement**: Load and validate configuration file before verification begins.

**Process**:
1. Parse YAML file
2. Validate against schema
3. Check for circular dependencies
4. Exit with error if invalid (exit code 2)

**Error Handling**:
- File not found: `Error: configuration file not found: {path}`
- Invalid YAML: `Error: invalid YAML syntax at line {n}: {message}`
- Schema violation: `Error: invalid configuration: {validation error}`
- Circular dependency: `Error: circular dependency detected: {cycle path}`

**Exit Code**: 2 (configuration error)

---

### BC-002: DAG-Based Execution

**Requirement**: Verify steps in dependency order, respecting DAG structure.

**Process**:
1. Build dependency graph from `depends_on` fields
2. Topologically sort steps
3. Verify steps in order (parallel where safe)
4. If step A blocks, dependent steps (B, C) marked as blocked without verification

**Parallelization**:
- Independent steps verified concurrently
- Respects DAG constraints (prerequisite verified before dependent)
- Bounded by available CPU cores (GOMAXPROCS)

**Blocking Propagation**:
- If step status is `missing`, `drifted`, `blocked`, or `unknown`, dependent steps MAY be marked `blocked` without verification attempt
- Alternative: Attempt verification of dependent steps regardless (current state might satisfy requirements)

---

### BC-003: Progress Indication (Non-JSON Mode)

**Requirement**: Display progress during verification for user feedback.

**Output** (example):
```
Verifying configuration: config.yaml (5 steps)

[1/5] install-git        ✔ satisfied (120ms)
[2/5] clone-repo         ✖ missing (50ms)
[3/5] render-config      ⚠ drifted (230ms)
[4/5] create-symlink     ✔ satisfied (80ms)
[5/5] run-setup          ? unknown (10ms)

────────────────────────────────────────────────
Summary: 5 steps verified in 490ms
  ✔ 2 satisfied
  ✖ 1 missing
  ⚠ 1 drifted
  🚫 0 blocked
  ? 1 unknown
```

**Symbols**:
- ✔ = satisfied (green)
- ✖ = missing (red)
- ⚠ = drifted (yellow)
- 🚫 = blocked (red)
- ? = unknown (yellow)

---

### BC-004: Verbose Output (--verbose flag)

**Requirement**: When `--verbose` flag set, include detailed explanations and diffs.

**Output** (example):
```
[3/5] render-config      ⚠ drifted (230ms)
      Message: file content differs (3 lines changed)
      
      --- expected: templates/app.conf.tmpl (rendered)
      +++ actual: /etc/app.conf
      @@ -1,5 +1,5 @@
       APP_NAME=Streamy
      -ENVIRONMENT=production
      +ENVIRONMENT=development
       DEBUG_MODE=false
```

**Included Information**:
- Full verification message
- Diff output for drifted steps (from `VerificationResult.Details`)
- Error details for blocked steps
- Explanation for unknown steps

---

### BC-005: JSON Output (--json flag)

**Requirement**: When `--json` flag set, output machine-readable JSON.

**JSON Schema**:
```json
{
  "config_file": "config.yaml",
  "summary": {
    "total_steps": 5,
    "satisfied": 2,
    "missing": 1,
    "drifted": 1,
    "blocked": 0,
    "unknown": 1,
    "duration_ms": 490
  },
  "results": [
    {
      "step_id": "install-git",
      "status": "satisfied",
      "message": "package git is installed (version 2.39.0)",
      "duration_ms": 120,
      "timestamp": "2025-10-04T14:30:00Z"
    },
    {
      "step_id": "clone-repo",
      "status": "missing",
      "message": "repository not found at /opt/myrepo",
      "duration_ms": 50,
      "timestamp": "2025-10-04T14:30:00Z"
    },
    {
      "step_id": "render-config",
      "status": "drifted",
      "message": "file content differs (3 lines changed)",
      "details": "--- expected...\n+++ actual...\n@@ -1,5 +1,5 @@\n...",
      "duration_ms": 230,
      "timestamp": "2025-10-04T14:30:01Z"
    },
    {
      "step_id": "create-symlink",
      "status": "satisfied",
      "message": "symlink /usr/bin/python points to /usr/bin/python3.11",
      "duration_ms": 80,
      "timestamp": "2025-10-04T14:30:01Z"
    },
    {
      "step_id": "run-setup",
      "status": "unknown",
      "message": "no verification command specified for this step",
      "duration_ms": 10,
      "timestamp": "2025-10-04T14:30:01Z"
    }
  ]
}
```

**JSON Requirements**:
- Valid JSON (no trailing commas, proper escaping)
- UTF-8 encoding
- Pretty-printed (indented) for readability
- No color codes or terminal escape sequences
- Always include `summary` and `results` fields

---

### BC-006: Exit Codes

**Requirement**: Command must exit with appropriate status code.

| Exit Code | Meaning | When to Use |
|-----------|---------|-------------|
| 0 | Success, all satisfied | All steps returned `satisfied` status |
| 1 | Verification found issues | Any step is `missing`, `drifted`, `blocked`, or `unknown` |
| 2 | Configuration error | Invalid config file, parse error, schema violation |
| 3 | Execution error | Unexpected failure (plugin crash, IO error) |

**Examples**:
```bash
# All satisfied
$ streamy verify config.yaml
# ... output ...
$ echo $?
0

# Some steps need work
$ streamy verify config.yaml
# ... output ...
$ echo $?
1

# Invalid config
$ streamy verify invalid.yaml
Error: circular dependency detected: step-a -> step-b -> step-a
$ echo $?
2
```

---

### BC-007: Error Handling

**Requirement**: Handle errors gracefully with clear messages.

**Error Categories**:

1. **Configuration Errors** (exit 2)
   - File not found, invalid YAML, schema validation failures
   - Format: `Error: {category}: {message}`
   - Example: `Error: configuration file not found: /path/to/config.yaml`

2. **Plugin Errors** (exit 3)
   - Plugin registration failures, unexpected panics
   - Format: `Error: plugin '{name}' failed: {message}`
   - Example: `Error: plugin 'package' failed: unexpected nil pointer`

3. **Execution Errors** (exit 3)
   - DAG build failures, unexpected IO errors
   - Format: `Error: {message}`
   - Example: `Error: failed to build dependency graph: {reason}`

**Error Output**: All errors written to stderr.

---

### BC-008: Timeout Enforcement

**Requirement**: Respect global and per-step timeout configuration.

**Timeout Hierarchy**:
1. Per-step `verify_timeout` (highest priority)
2. Global `--timeout` flag (command-line override)
3. Default 30s (fallback)

**Timeout Behavior**:
- If verification exceeds timeout, step returns `blocked` status
- Message includes timeout duration: "verification timed out after 30s"
- Verification continues for remaining steps

**Example**:
```yaml
- id: slow-check
  type: template
  source: large-template.tmpl
  destination: /etc/large-config
  verify_timeout: 120s  # Override default 30s
```

```bash
streamy verify config.yaml --timeout=60s  # Global override
```

---

### BC-009: Signal Handling

**Requirement**: Handle interrupt signals (SIGINT, SIGTERM) gracefully.

**Behavior**:
- On signal receipt: cancel all in-progress verifications
- Wait briefly (1s) for clean shutdown
- Print partial results if available
- Exit with code 130 (SIGINT) or 143 (SIGTERM)

**Example**:
```bash
$ streamy verify config.yaml
Verifying configuration: config.yaml (100 steps)
[1/100] step-1  ✔ satisfied
[2/100] step-2  ✖ missing
^C
Verification interrupted by user (2/100 steps completed)
$ echo $?
130
```

---

### BC-010: Logging Integration

**Requirement**: Emit structured logs for debugging and observability.

**Log Levels**:
- INFO: Command start/end, summary
- DEBUG: Per-step verification start/end, duration
- WARN: Timeouts, recoverable errors
- ERROR: Configuration errors, plugin failures

**Structured Fields**:
- `config_file`: Path to configuration
- `step_id`: Current step being verified
- `status`: Verification status
- `duration_ms`: Verification duration
- `error`: Error message (if any)

**Example Log Output** (JSON format, controlled via logger config):
```json
{"level":"info","msg":"starting verification","config_file":"config.yaml","steps":5,"ts":"2025-10-04T14:30:00Z"}
{"level":"debug","msg":"verifying step","step_id":"install-git","plugin":"package","ts":"2025-10-04T14:30:00Z"}
{"level":"debug","msg":"step verified","step_id":"install-git","status":"satisfied","duration_ms":120,"ts":"2025-10-04T14:30:00Z"}
{"level":"info","msg":"verification complete","summary":{"satisfied":2,"missing":1,"drifted":1,"blocked":0,"unknown":1},"duration_ms":490,"ts":"2025-10-04T14:30:01Z"}
```

---

## Integration with Apply Command (Future)

### Future Enhancement: `--verify-first` Flag on Apply

**Proposed Behavior**:
```bash
streamy apply config.yaml --verify-first
```

1. Run verification phase for all steps
2. Skip steps with `satisfied` status
3. Apply only `missing`, `drifted`, `unknown`, `blocked` steps
4. Report verification results + apply results

**Benefits**:
- Reduces redundant work on repeated applies
- Clear visibility into what was skipped vs executed
- Performance optimization for idempotent configs

**Exit Codes**:
- 0: All steps satisfied (nothing applied) or apply succeeded
- 1: Apply failed for one or more steps

---

## Contract Test Requirements

### Test: All Satisfied Exit Code
```go
func TestVerifyAllSatisfied(t *testing.T) {
    // Setup config with all steps satisfied
    exitCode := runVerify("satisfied-config.yaml")
    assert.Equal(t, 0, exitCode)
}
```

### Test: Missing Step Exit Code
```go
func TestVerifyMissingStep(t *testing.T) {
    // Setup config with one missing step
    exitCode := runVerify("missing-config.yaml")
    assert.Equal(t, 1, exitCode)
}
```

### Test: Invalid Config Exit Code
```go
func TestVerifyInvalidConfig(t *testing.T) {
    exitCode := runVerify("invalid-config.yaml")
    assert.Equal(t, 2, exitCode)
}
```

### Test: JSON Output Schema
```go
func TestVerifyJSONOutput(t *testing.T) {
    output := runVerify("config.yaml", "--json")
    var result VerificationOutput
    err := json.Unmarshal(output, &result)
    require.NoError(t, err)
    assert.NotEmpty(t, result.Summary)
    assert.NotEmpty(t, result.Results)
}
```

### Test: Verbose Output Includes Diffs
```go
func TestVerifyVerboseOutput(t *testing.T) {
    output := runVerify("drifted-config.yaml", "--verbose")
    assert.Contains(t, output, "--- expected")
    assert.Contains(t, output, "+++ actual")
}
```

### Test: Timeout Enforcement
```go
func TestVerifyTimeout(t *testing.T) {
    // Setup step that takes >1s to verify
    start := time.Now()
    exitCode := runVerify("slow-config.yaml", "--timeout=1s")
    duration := time.Since(start)
    
    assert.Equal(t, 1, exitCode) // blocked status
    assert.Less(t, duration, 2*time.Second) // completes fast
}
```

---

## Version Compatibility

**Current Version**: 1.0.0  
**Stability**: Unstable (pre-1.0 Streamy release)

**Future Changes**:
- May add filtering flags (`--step`, `--tag`) for selective verification
- May add `--verify-first` integration with apply command
- May add `--watch` mode for continuous verification

---

**Contract Status**: Complete  
**Last Updated**: October 4, 2025  
**Next**: Implement verification executor and CLI command
