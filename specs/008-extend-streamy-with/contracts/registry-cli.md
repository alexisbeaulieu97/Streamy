# Command Contracts: Registry Management CLI

**Phase**: 1 - Design & Contracts  
**Date**: October 9, 2025  
**Status**: Complete

## Overview

This document defines the command-line interface contracts for registry management commands. Each contract specifies command syntax, arguments, flags, output formats, exit codes, and error handling.

---

## 1. Register Command

### Syntax

```bash
streamy register <config-path> [flags]
```

### Description

Registers a new Streamy configuration file in the pipeline registry, making it available for management via CLI and visible in the interactive dashboard.

### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `config-path` | string | Yes | Path to Streamy YAML configuration file (relative or absolute) |

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--id` | `-i` | string | auto-generated | Unique identifier for the pipeline (must match `^[a-z0-9][a-z0-9-]*[a-z0-9]$`) |
| `--name` | `-n` | string | filename | Human-readable name for the pipeline |
| `--description` | `-d` | string | "" | Optional description of the pipeline's purpose |
| `--verbose` | `-v` | bool | false | Enable verbose logging (inherited from root) |

### Behavior

1. Validates `config-path` exists and is readable
2. Parses config file to ensure valid Streamy YAML
3. Generates ID from filename if `--id` not provided (sanitize: lowercase, alphanumeric+hyphens)
4. Checks for duplicate ID in registry
5. Converts relative path to absolute path
6. Creates Pipeline struct with current timestamp
7. Adds to registry and saves atomically
8. Prints success message with pipeline ID

### Output Examples

**Success**:
```
✓ Registered pipeline 'dev-setup' (Development Environment)
  Path: /home/user/configs/dev.yaml
  ID:   dev-setup

Run 'streamy refresh dev-setup' to verify its current status.
```

**Success (verbose)**:
```
→ Validating config file: /home/user/configs/dev.yaml
→ Parsing YAML configuration...
✓ Configuration is valid (15 steps found)
→ Generating pipeline ID from filename: dev-setup
→ Checking for duplicate ID...
→ Adding pipeline to registry...
→ Saving registry to /home/user/.streamy/registry.json
✓ Registered pipeline 'dev-setup' (Development Environment)
  Path: /home/user/configs/dev.yaml
  ID:   dev-setup

Run 'streamy refresh dev-setup' to verify its current status.
```

### Error Cases

| Error | Exit Code | Output | Suggestion |
|-------|-----------|--------|------------|
| Config file not found | 1 | `Error: config file not found: /path/to/config.yaml` | Check the path and try again. Use absolute path or relative path from current directory. |
| Config parse error | 1 | `Error: invalid config: yaml: line 15: mapping values are not allowed in this context` | Fix the YAML syntax error at the indicated line. |
| Invalid config (no steps) | 1 | `Error: config must contain at least one step` | Add steps to your configuration file. See https://streamy.dev/docs for examples. |
| Duplicate ID | 1 | `Error: pipeline with ID 'dev-setup' already exists` | Use --id flag to specify a different ID, or unregister the existing pipeline first. |
| Permission denied | 1 | `Error: cannot write to registry: permission denied` | Check write permissions for ~/.streamy/ directory. |
| Invalid ID format | 1 | `Error: invalid pipeline ID 'Dev-Setup': must be lowercase alphanumeric with hyphens` | Use only lowercase letters, numbers, and hyphens. ID must start and end with alphanumeric character. |

### Exit Codes

- `0`: Success
- `1`: Validation or execution error
- `2`: Invalid arguments

---

## 2. Unregister Command

### Syntax

```bash
streamy unregister <pipeline-id> [flags]
```

### Description

Removes a pipeline from the registry. The configuration file is not deleted, only the registry entry.

### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `pipeline-id` | string | Yes | Unique identifier of the pipeline to remove |

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--force` | `-f` | bool | false | Skip confirmation prompt |
| `--verbose` | `-v` | bool | false | Enable verbose logging (inherited from root) |

### Behavior

1. Validates pipeline exists in registry
2. Prompts for confirmation (unless `--force`)
3. Waits for user input: `y`/`yes` confirms, anything else cancels
4. Removes pipeline from registry
5. Saves registry atomically
6. Optionally removes status cache entry
7. Prints success message

### Output Examples

**Success (with prompt)**:
```
Remove pipeline 'dev-setup' from registry? [y/N]: y
✓ Unregistered pipeline 'dev-setup'

The configuration file at /home/user/configs/dev.yaml was not deleted.
```

**Success (with --force)**:
```
✓ Unregistered pipeline 'dev-setup'

The configuration file at /home/user/configs/dev.yaml was not deleted.
```

**Cancelled by user**:
```
Remove pipeline 'dev-setup' from registry? [y/N]: n
Cancelled.
```

### Error Cases

| Error | Exit Code | Output | Suggestion |
|-------|-----------|--------|------------|
| Pipeline not found | 1 | `Error: pipeline 'unknown-id' not found in registry` | Run 'streamy list' to see registered pipelines. |
| Permission denied | 1 | `Error: cannot write to registry: permission denied` | Check write permissions for ~/.streamy/ directory. |
| No TTY (interactive mode required) | 1 | `Error: cannot prompt for confirmation: not a terminal` | Use --force flag when running in non-interactive environments. |

### Exit Codes

- `0`: Success (or user cancelled)
- `1`: Execution error
- `2`: Invalid arguments

---

## 3. List Command

### Syntax

```bash
streamy list [flags]
```

### Description

Displays all registered pipelines with their current status, last run time, and metadata. Supports both human-readable table format and machine-readable JSON output.

### Arguments

None

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--format` | | string | "table" | Output format: `table` or `json` |
| `--json` | | bool | false | Shorthand for --format=json |
| `--verbose` | `-v` | bool | false | Enable verbose logging (inherited from root) |

### Behavior

1. Loads registry from disk
2. Loads status cache from disk
3. Merges pipeline metadata with runtime status
4. Formats output according to `--format` flag
5. Writes to stdout

### Output Examples

**Table format (default)**:
```
ID             NAME                     STATUS          LAST RUN         PATH
dev-setup      Development Environment  🟢 satisfied    2 hours ago      /home/user/configs/dev.yaml
staging-env    Staging Environment      🟡 drifted      1 day ago        /home/user/configs/staging.yaml
prod-deploy    Production Deploy        ⚪ unknown      never            /home/user/configs/prod.yaml
```

**Table format (ASCII fallback, no Unicode support)**:
```
ID             NAME                     STATUS          LAST RUN         PATH
dev-setup      Development Environment  [OK] satisfied  2 hours ago      /home/user/configs/dev.yaml
staging-env    Staging Environment      [!!] drifted    1 day ago        /home/user/configs/staging.yaml
prod-deploy    Production Deploy        [??] unknown    never            /home/user/configs/prod.yaml
```

**Empty registry**:
```
No pipelines registered yet.

Run 'streamy register <config-path>' to add your first pipeline.
```

**JSON format**:
```json
{
  "version": "1.0",
  "count": 3,
  "pipelines": [
    {
      "id": "dev-setup",
      "name": "Development Environment",
      "path": "/home/user/configs/dev.yaml",
      "description": "Local development tools and configs",
      "status": "satisfied",
      "last_run": "2025-10-09T12:30:00Z",
      "registered_at": "2025-10-08T15:20:00Z"
    },
    {
      "id": "staging-env",
      "name": "Staging Environment",
      "path": "/home/user/configs/staging.yaml",
      "description": "",
      "status": "drifted",
      "last_run": "2025-10-08T10:15:00Z",
      "registered_at": "2025-10-07T09:00:00Z"
    },
    {
      "id": "prod-deploy",
      "name": "Production Deploy",
      "path": "/home/user/configs/prod.yaml",
      "description": "Production environment setup",
      "status": "unknown",
      "last_run": null,
      "registered_at": "2025-10-09T14:00:00Z"
    }
  ]
}
```

### Error Cases

| Error | Exit Code | Output | Suggestion |
|-------|-----------|--------|------------|
| Registry file corrupted | 1 | `Error: failed to parse registry: invalid JSON` | Registry file may be corrupted. Restore from backup or recreate by re-registering pipelines. |
| Permission denied | 1 | `Error: cannot read registry: permission denied` | Check read permissions for ~/.streamy/registry.json |
| Invalid format flag | 2 | `Error: invalid format 'csv': must be 'table' or 'json'` | Use --format=table or --format=json |

### Exit Codes

- `0`: Success (including empty registry)
- `1`: Execution error
- `2`: Invalid arguments

---

## 4. Refresh Command

### Syntax

```bash
streamy refresh [pipeline-id] [flags]
```

### Description

Re-runs verification checks on registered pipelines and updates their status in the cache. If no pipeline ID is provided, refreshes all pipelines.

### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `pipeline-id` | string | No | Specific pipeline to refresh (default: all pipelines) |

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--concurrency` | `-c` | int | 5 | Number of pipelines to verify concurrently (when refreshing all) |
| `--verbose` | `-v` | bool | false | Enable verbose logging (inherited from root) |
| `--dry-run` | | bool | false | Show what would be verified without executing (inherited from root) |

### Behavior

**Single pipeline refresh**:
1. Validates pipeline exists in registry
2. Loads pipeline configuration
3. Executes verify operation
4. Updates status cache with result
5. Saves status cache atomically
6. Prints result

**All pipelines refresh**:
1. Loads all pipelines from registry
2. Creates worker pool with `--concurrency` limit
3. Spawns goroutines to verify each pipeline
4. Collects results as they complete
5. Updates status cache with all results
6. Saves status cache atomically
7. Prints summary report

### Output Examples

**Single pipeline**:
```
Refreshing pipeline 'dev-setup'...
✓ dev-setup: satisfied (15/15 steps passed)

Run 'streamy list' to see updated status.
```

**All pipelines**:
```
Refreshing 3 pipelines...

[1/3] dev-setup...     ✓ satisfied
[2/3] staging-env...   ⚠ drifted (2 steps failed)
[3/3] prod-deploy...   ✗ failed (config file not found)

Summary:
  ✓ Satisfied: 1
  ⚠ Drifted:   1
  ✗ Failed:    1

Run 'streamy list' to see detailed status.
```

**Verbose (single pipeline)**:
```
Refreshing pipeline 'dev-setup'...
→ Loading config: /home/user/configs/dev.yaml
→ Parsing configuration...
→ Building execution DAG...
→ Executing 15 steps...
  [1/15] install-git...        ✓ satisfied
  [2/15] install-docker...     ✓ satisfied
  [3/15] clone-repo...         ✓ satisfied
  ...
  [15/15] setup-aliases...     ✓ satisfied
✓ dev-setup: satisfied (15/15 steps passed)

Run 'streamy list' to see updated status.
```

**Dry-run**:
```
Dry-run: Would refresh the following pipelines:
  - dev-setup (Development Environment)
  - staging-env (Staging Environment)
  - prod-deploy (Production Deploy)

No changes made.
```

### Error Cases

| Error | Exit Code | Output | Suggestion |
|-------|-----------|--------|------------|
| Pipeline not found | 1 | `Error: pipeline 'unknown-id' not found in registry` | Run 'streamy list' to see registered pipelines. |
| Config file missing | 1 | `Error: config file not found for pipeline 'dev-setup': /path/to/config.yaml` | Update the registry with the new path or unregister the pipeline. |
| Verify execution error | 1 | `Error: failed to verify pipeline 'dev-setup': <error details>` | Check the pipeline configuration and system state. Run 'streamy verify <config-path>' for details. |
| Permission denied | 1 | `Error: cannot write to status cache: permission denied` | Check write permissions for ~/.streamy/status.json |

### Exit Codes

- `0`: Success (all pipelines verified, regardless of their status)
- `1`: Execution error (cannot verify due to system error)
- `2`: Invalid arguments

**Note**: A pipeline being "drifted" or "failed" is not an execution error—refresh succeeds even if pipelines are not satisfied.

---

## 5. Show Command (Future Enhancement)

### Syntax

```bash
streamy show <pipeline-id> [flags]
```

### Description

Displays detailed information about a specific pipeline including full metadata, configuration summary, execution history, and recent status.

**Note**: This command is specified in the feature spec but marked as P3 (lowest priority). Including here for completeness.

### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `pipeline-id` | string | Yes | Unique identifier of the pipeline |

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--json` | | bool | false | Output in JSON format |
| `--verbose` | `-v` | bool | false | Enable verbose logging (inherited from root) |

### Behavior

1. Validates pipeline exists in registry
2. Loads pipeline metadata from registry
3. Loads status from cache
4. Loads execution history (if available)
5. Formats comprehensive output
6. Writes to stdout

### Output Example

```
Pipeline: dev-setup
Name:     Development Environment
Path:     /home/user/configs/dev.yaml

Description:
  Local development tools and configs for the team.

Status:   🟢 satisfied
Last Run: 2025-10-09 12:30:00 (2 hours ago)
Duration: 45.3s

Configuration:
  Steps:     15
  Plugins:   package (apt, brew), repo (git), symlink, command
  Variables: 3

Recent Execution (verify):
  ✓ install-git           (0.5s)
  ✓ install-docker        (12.3s)
  ✓ clone-repo            (3.2s)
  ✓ setup-git-config      (0.1s)
  ... (11 more)

Registered: 2025-10-08 15:20:00 (1 day ago)

Run 'streamy verify /home/user/configs/dev.yaml' to re-verify this pipeline.
```

---

## Common Patterns

### Global Flags

Inherited from root command, available on all subcommands:

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--verbose` | `-v` | bool | false | Enable verbose logging |
| `--dry-run` | | bool | false | Preview actions without executing (where applicable) |

### Exit Code Summary

| Code | Meaning | Usage |
|------|---------|-------|
| 0 | Success | Command completed successfully |
| 1 | Execution error | Runtime error (file not found, permission denied, etc.) |
| 2 | Invalid arguments | Wrong number of arguments or invalid flag values |

### Output Conventions

**Success indicators**:
- Prefix: `✓` (Unicode checkmark) or `[OK]` (ASCII)
- Color: Green (when TTY supports colors)

**Warning indicators**:
- Prefix: `⚠` (Unicode warning) or `[!!]` (ASCII)
- Color: Yellow

**Error indicators**:
- Prefix: `✗` (Unicode cross) or `[XX]` (ASCII)
- Color: Red

**Progress indicators**:
- Format: `[N/M] pipeline-id... ✓ status`
- Only shown for batch operations (refresh all)

**Timestamps**:
- Absolute: ISO 8601 format (`2025-10-09T12:30:00Z`)
- Relative: Human-friendly (`2 hours ago`, `1 day ago`, `never`)

### Error Message Format

```
Error: <brief description>

<detailed context or underlying error>

Suggestion: <actionable next step>
```

### Non-Interactive Detection

Commands detect non-interactive environments (no TTY) and adjust behavior:
- Disable color codes
- Use ASCII instead of Unicode icons
- Disable confirmation prompts (require `--force`)
- Simplify progress output

---

## Contract Testing Checklist

For each command, verify:

- [ ] Command accepts specified arguments and flags
- [ ] Invalid arguments/flags trigger exit code 2
- [ ] Error conditions produce documented error messages
- [ ] Success conditions produce documented output format
- [ ] Exit codes match specification
- [ ] `--json` output is valid JSON with specified schema
- [ ] `--verbose` flag increases output detail
- [ ] `--dry-run` flag (where applicable) prevents side effects
- [ ] Non-interactive mode (no TTY) behaves correctly
- [ ] Unicode fallback works when locale doesn't support UTF-8

---

## API Stability

**Stability Promise**: 
- Command names, arguments, and flags are part of the public API
- Breaking changes require major version bump
- New optional flags are allowed in minor versions
- Output format changes (non-JSON) are not considered breaking
- JSON output schema changes follow semantic versioning

**Deprecation Process**:
1. Announce in release notes
2. Add deprecation warning for one minor version
3. Remove in next major version
