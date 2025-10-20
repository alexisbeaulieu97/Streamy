# Plugin Development Guide (Ports Architecture)

Streamy executes environment steps through plugins that implement the
`ports.Plugin` interface. Plugins run inside the domain-driven architecture and
interact with use cases only through strongly typed interfaces. This guide
describes the contract, lifecycle, and best practices for authoring built-in or
custom plugins.

## Overview

- Plugins reside under `internal/plugins/<name>`.
- Each plugin exports a `New()` constructor returning a `ports.Plugin`.
- Plugins operate on domain types (`pipeline.Step`, `pipeline.EvaluationResult`,
  `pipeline.StepResult`) and must return domain errors (`*pipeline.DomainError`)
  when something goes wrong.
- Metadata drives registration and dependency validation via the infrastructure
  plugin registry (`internal/infrastructure/plugin`).
- Contract tests in `internal/plugins/contracts_test.go` ensure every plugin
  respects metadata and cancellation conventions.

## Core Interfaces

```go
type Plugin interface {
    Metadata() plugin.Metadata
    Evaluate(ctx context.Context, step pipeline.Step) (*pipeline.EvaluationResult, error)
    Apply(ctx context.Context, evaluation *pipeline.EvaluationResult, step pipeline.Step) (*pipeline.StepResult, error)
}
```

- **Metadata**: Returns identity, type, version, optional dependencies, and
  description. Use the helper constructors defined in `internal/domain/plugin`.
- **Evaluate**: Read-only assessment of system state. Must honour context
  cancellation and return structured domain errors if validation or inspection
  fails.
- **Apply**: Mutates the system to achieve the desired state, typically using
  internal data provided by `Evaluate`. Must be idempotent and honour context.

The plugin registry port exposes:

```go
type PluginRegistry interface {
    Register(p Plugin) error
    Get(stepType plugin.Type) (Plugin, error)
    List() []Plugin
}
```

Infrastructure adapters populate the registry during CLI start-up using
`RegisterPortsPlugins` (`cmd/streamy/plugins_import.go`).

## Building a Plugin

1. **Create the package**
   ```bash
   mkdir internal/plugins/myplugin
   ```

2. **Implement `New()` and `Metadata()`**
   ```go
   func New() ports.Plugin { return &Plugin{} }

   func (Plugin) Metadata() plugin.Metadata {
       return plugin.Metadata{
           ID:          "myplugin",
           Name:        "myplugin",
           Version:     "1.0.0",
           Type:        plugin.Type("my_plugin"),
           Description: "Explains what the plugin does.",
       }
   }
   ```

3. **Decode configuration from `pipeline.Step`**
   ```go
   cfg, err := decodeConfig(step.Config)
   if err != nil {
       return nil, pipeline.NewValidationError("invalid configuration", map[string]any{"step_id": step.ID})
   }
   ```
   Keep decoding helpers private to the plugin package.

4. **Implement `Evaluate`**
   - Must not modify system state.
   - Return an `EvaluationResult` containing:
     - `RequiresAction` flag.
     - `CurrentState` string (use domain constants such as
       `pipeline.VerificationSatisfied`).
     - Optional `Diff` text for display.
     - Optional `InternalData` struct to carry expensive computations to `Apply`.

5. **Implement `Apply`**
   - Reuse data from `evaluation.InternalData` if present.
   - Handle cancellation by checking `ctx.Err()`.
   - Return a `StepResult` with status (`pipeline.StatusSuccess`, `StatusFailure`,
     etc.) and message. Set `Changed` when the operation modifies state.

6. **Register the plugin**
   Add the constructor to `portPluginFactories` in
   `cmd/streamy/plugins_import.go`. Custom builds can register additional plugins
   by calling the registry directly before executing commands.

7. **Add tests**
   - Unit tests in `internal/plugins/myplugin/plugin_test.go`.
   - Use `tests/harness.go` or `internal/infrastructure/engine` integration tests
     when the plugin affects multi-step flows.

## Error Handling

- Wrap operational issues with `pipeline.NewDomainError` or specialized helpers
  such as `NewExecutionError`. Always include `step_id` and `plugin_type` in the
  context map.
- Validation errors should use the `ErrCodeValidation` code; missing resources
  typically use `ErrCodeNotFound`.
- Contract tests assert that cancelled contexts bubble up as `ErrCodeCancelled`.

## Observability

- Plugins automatically inherit logging, metrics, and tracing from the executor:
  - Use `ports.Logger` when logging within Apply/Evaluate if the plugin stores a
    logger.
  - Record durations in the returned `StepResult.Duration` where applicable.
- Prefer human-readable yet precise messages; the CLI renders them in tables,
  verbose logs, or JSON.

## Cross-Plugin Dependencies

- Metadata exposes dependencies through the `Dependencies` field on
  `plugin.Metadata`. For built-ins, dependencies are minimal; however, you can
  state relationships such as:
  ```go
  Dependencies: []string{"line_in_file"},
  ```
- The registry validates dependencies at startup and provides lookups via
  `GetForDependent` (see infrastructure registry implementation). Keep
  dependency usage within infrastructure adapters; plugins themselves should
  avoid global lookups to preserve testability.

## Testing Checklist

- `go test ./internal/plugins/<name>` – unit tests.
- `go test ./internal/plugins` – contract suite.
- `go test ./tests -run Plugin` – integration scenarios.
- Update `tests/harness.go` if new plugins must be registered during tests.

## Migration Notes

- Legacy `internal/plugin` and `internal/model` packages are removed. All new
  plugins must rely exclusively on the ports and domain types documented here.
- Configuration decoding happens through the infrastructure YAML loader, so
  plugins no longer interact with legacy `config.Step` structs.
- Contract tests ensure future refactors keep metadata and cancellation behaviour
  consistent.
