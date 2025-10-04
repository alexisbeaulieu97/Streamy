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
