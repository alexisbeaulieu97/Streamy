# Plugin Development Guide

Streamy executes environment setup steps via plugins. Each step type must provide a plugin that implements the standard interface and registers itself with the global registry. This guide explains how to build, test, and register a new plugin.

## Plugin Interface

All plugins satisfy `internal/plugin.Plugin`:

```go
type Plugin interface {
    Metadata() Metadata
    Schema() interface{}
    Check(ctx context.Context, step *config.Step) (bool, error)
    Apply(ctx context.Context, step *config.Step) (*model.StepResult, error)
    DryRun(ctx context.Context, step *config.Step) (*model.StepResult, error)
}
```

- **Metadata**: Provides name, semantic version, and step type string.
- **Schema**: Returns a struct used for documentation or code-generation; optional but recommended.
- **Check**: Determines idempotency (`true` = step can be skipped). Should be fast and side-effect free.
- **Apply**: Performs the actual work and returns a `StepResult` detailing status, messages, and errors.
- **DryRun**: Returns a preview `StepResult` without performing side effects.

All methods receive the fully populated `config.Step`, so type-specific fields are available (e.g., `step.Package` for the package plugin).

## Registration

Plugins register themselves in an `init()` function with the registry:

```go
func init() {
    if err := plugin.RegisterPlugin("command", New()); err != nil {
        panic(err)
    }
}
```

The registry stores one plugin per type string; attempts to register duplicates return a `PluginError`.

## Example: Command Plugin

Located at `internal/plugins/command/command.go`, this plugin:
- Executes shell commands using `exec.CommandContext`.
- Supports optional `Check` commands for idempotency.
- Handles environment variables, working directory, and shell detection.
- Returns `StepResult` instances that drive the TUI and logging layers.

## Implementation Checklist

1. **Define Plugin Struct & Constructor**
   - Provide a `New()` function returning `plugin.Plugin` implementation.

2. **Implement Interface Methods**
   - Use the data model from `internal/config` to access type-specific fields.
   - Prefer context-aware system calls (`exec.CommandContext` / `os` APIs).
   - Ensure `Apply` populates `StepResult` with `StepID`, `Status`, human-readable `Message`, and non-nil `Error` on failure.
   - Ensure `DryRun` returns `StatusSkipped` with a clear message.

3. **Handle Idempotency**
   - `Check` should detect whether work is necessary (hash comparisons, file existence, package installation checks, etc.).
   - Return `(true, nil)` if the resource is already provisioned.

4. **Support Dry-Run**
   - Ensure `Apply` is safe to call after `DryRun`. The executor will call `DryRun` when `--dry-run` is set.

5. **Wrap Errors**
   - Use `pkg/errors` helpers (`NewPluginError`, `NewExecutionError`) or `fmt.Errorf("context: %w", err)` to provide context.

6. **Register Plugin**
   - Call `plugin.RegisterPlugin("<type>", New())` in `init`. Use lowercase step type names.

7. **Add Tests**
   - Place tests beside the plugin (`package/command_test.go`).
   - Mock system interactions by writing temporary files/scripts (`t.TempDir()`) or using in-memory structures.
   - Ensure tests cover `Check`, `Apply`, `DryRun`, and error conditions.

8. **Update Docs**
   - Document new step type usage in `docs/schema.md` and `README.md` once implemented.

## Testing Plugins

- Unit tests (`go test ./internal/plugins/<type>`) ensure interface compliance.
- Integration tests (`tests/integration_test.go`) should include scenarios exercising the new plugin to verify orchestration.

## Adding a New Step Type

1. Update `internal/config/types.go` to include type-specific struct and add it to `Step` inline fields.
2. Extend `ValidateStep` in `internal/config/validator.go` to handle the new type.
3. Create plugin implementation and tests under `internal/plugins/<type>/`.
4. Update documentation (`docs/schema.md`, README) with examples and validation rules.
5. Add integration tests where appropriate.

Following this guide ensures new plugins integrate seamlessly with Streamy's execution engine, validations, and user interfaces.

## Template Plugin (`type: template`)

The template plugin renders destination files from Go `text/template` sources with variable substitution. Use it when you need reproducible configuration files that vary per environment or developer.

### Key Features

- **Variable Resolution**: Inline `vars` take precedence over environment variables (`env: true` enables access). Missing variables trigger failures unless `allow_missing: true` is set.
- **Idempotency**: `Check` and `Apply` compare SHA-256 hashes of rendered output versus the destination file to skip unchanged files.
- **Dry-Run Support**: `DryRun` reports `would_create` or `would_update` without touching the filesystem.
- **Permissions**: Copy permissions from the template source by default, or supply an explicit `mode`.
- **Error Reporting**: Template parse/runtime errors include file name and line/column information for fast debugging.

### Configuration Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `source` | string | Yes | Path to the `.tmpl` file rendered with Go template syntax |
| `destination` | string | Yes | File path to write the rendered content |
| `vars` | map[string]string | No | Inline variables; override environment values |
| `env` | bool (default `true`) | No | Enable environment variable lookups |
| `allow_missing` | bool (default `false`) | No | Skip errors for undefined variables (render as empty string) |
| `mode` | octal (e.g., `0600`) | No | Explicit destination file permissions |

### Example

```yaml
- id: render-config
  type: template
  source: templates/app.conf.tmpl
  destination: config/app.conf
  vars:
    APP_NAME: Streamy
    ENVIRONMENT: production
    DEBUG_MODE: "false"
  mode: 0644

- id: render-secrets
  type: template
  source: templates/secret.env.tmpl
  destination: config/secret.env
  env: true           # pull API keys from the environment
  mode: 0600          # tighten permissions for secrets

- id: render-optional
  type: template
  source: templates/optional.conf.tmpl
  destination: config/optional.conf
  allow_missing: true # optional variables render as empty strings
```

### Best Practices

- Keep templates deterministic—avoid timestamps or random values that break idempotency.
- Validate variable names with Go identifier rules (`^[a-zA-Z_][a-zA-Z0-9_]*$`).
- Use dry-run (`streamy apply --dry-run`) to preview changes; look for `would_create` / `would_update` statuses.
- Pair each template with table-driven tests using `t.TempDir()` to guarantee portability.

## Line In File Plugin (`type: line_in_file`)

The line_in_file plugin provides declarative, idempotent management of text file lines. It ensures specific lines exist or are removed, supports pattern-based replacement, and respects Streamy's safety features (dry-run, backups, verbose output).

### Key Features

- **Idempotent Operations**: Content comparison ensures no unnecessary file modifications on repeated runs.
- **Pattern Matching**: Use regular expressions to find and replace specific lines or remove multiple matches.
- **Multiple Match Strategies**: Configure how to handle multiple regex matches (first, all, error, or interactive prompt).
- **Backup Support**: Automatic timestamped backups before destructive changes.
- **Encoding Support**: Handle various text encodings (UTF-8, Latin-1, ASCII, etc.).
- **Atomic Writes**: Safe file modifications using temp file + rename pattern.
- **Dry-Run Preview**: Unified diff format shows exactly what changes will be made.

### Configuration Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `file` | string | Yes | — | Path to target file (supports `~` expansion) |
| `line` | string | Yes | — | Exact line content to add, replace, or remove |
| `state` | string | No | `"present"` | `"present"` or `"absent"` - desired line state |
| `match` | string | No | — | Regular expression pattern (required when `state: absent`) |
| `on_multiple_matches` | string | No | `"prompt"` | `"first"`, `"all"`, `"error"`, or `"prompt"` |
| `backup` | boolean | No | `false` | Create backup file before changes |
| `backup_dir` | string | No | — | Directory for backup files (default: same as target) |
| `encoding` | string | No | `"utf-8"` | File encoding (`utf-8`, `latin-1`, `ascii`, etc.) |

### Usage Examples

#### Add Line to File
```yaml
- id: add_path_export
  type: line_in_file
  file: "~/.bashrc"
  line: 'export PATH="$PATH:$HOME/bin"'
  state: present
```

#### Replace Matched Line
```yaml
- id: disable_debug
  type: line_in_file
  file: "/etc/app/config.ini"
  line: "debug=false"
  match: '^debug='
  state: present
  on_multiple_matches: first
```

#### Remove Line Pattern
```yaml
- id: cleanup_old_vars
  type: line_in_file
  file: "~/.profile"
  match: '^export DEPRECATED_VAR='
  state: absent
```

#### Backup Before Changes
```yaml
- id: update_hosts
  type: line_in_file
  file: "/etc/hosts"
  line: "127.0.0.1 myapp.local"
  backup: true
  backup_dir: "/var/backups/streamy"
```

#### Handle Multiple Matches
```yaml
- id: update_all_paths
  type: line_in_file
  file: "/tmp/paths"
  line: 'export PATH="/opt/bin:$PATH"'
  match: '^export PATH='
  on_multiple_matches: all  # Options: first, all, error, prompt
```

#### Encoding Support
```yaml
- id: update_legacy_config
  type: line_in_file
  file: "/legacy/app.conf"
  line: "charset=utf-8"
  encoding: "latin-1"
  state: present
```

### Dry-Run Output

Use `--dry-run` to preview changes:
```bash
streamy apply config.yaml --dry-run --verbose
```

Example dry-run output:
```
⊙ add_path_export — Would add 1 line to /home/user/.bashrc
  + export PATH="$PATH:$HOME/bin"

✓ disable_debug — Would replace 1 line in /etc/app/config.ini
  - debug=true
  + debug=false
```

### Best Practices

- Always use `match` patterns when replacing existing lines to avoid duplicates.
- Test with `--dry-run` first to verify changes before applying.
- Use `backup: true` for critical system files.
- Specify `on_multiple_matches` explicitly in automated scripts (avoid `prompt` in CI/CD).
- Use anchored patterns (`^`, `$`) for faster and more precise matching.
- Combine with `depends_on` to ensure files exist before modification.

### Error Handling

- **Permission Denied**: Clear error messages with suggestions (run with sudo, check permissions)
- **Invalid Regex**: Detailed regex syntax errors during config validation
- **Encoding Errors**: Clear messages when files can't be decoded with specified encoding
- **Interactive Mode**: Graceful fallback when `prompt` strategy used in non-TTY environments

For complete examples and validation scenarios, see `specs/003-add-built-in/quickstart.md`.
