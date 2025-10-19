# Adding a Plugin

Use this guide to add a new built-in plugin (for example, a `service` plugin
that ensures a systemd unit is enabled). The process applies to custom builds as
well—simply register the plugin before running Streamy commands.

## 1. Plan the Step Schema

- Choose a unique step `type` string (e.g. `service`).
- Enumerate configuration fields and default behaviour.
- Document expected outcomes for Evaluate (satisfied/missing/drifted) and Apply
  (success/failure).

## 2. Create the Package

```bash
mkdir internal/plugins/service
```

Add `plugin.go` with a minimal skeleton:

```go
package serviceplugin

type Plugin struct{}

func New() ports.Plugin { return &Plugin{} }
```

## 3. Implement Metadata

```go
func (Plugin) Metadata() plugin.Metadata {
    return plugin.Metadata{
        ID:          "service",
        Name:        "service",
        Version:     "1.0.0",
        Type:        plugin.Type("service"),
        Description: "Manages system services via systemctl",
    }
}
```

Keep metadata consistent with the step type string and use semantic versions.

## 4. Decode Configuration

Add helper functions to translate `pipeline.Step.Config` into typed structs:

```go
type config struct {
    Name   string
    Action string // start/stop/restart
    Enable bool
}

func decode(step pipeline.Step) (config, error) {
    // Validate required keys, convert to strings/bools, return domain errors.
}
```

Reuse domain error helpers (`pipeline.NewValidationError`, etc.) so the caller
receives consistent messaging.

## 5. Implement Evaluate

- Inspect current system state (e.g. `systemctl is-active`).
- Honour `ctx.Err()` before expensive operations.
- Populate `EvaluationResult`:
  - `RequiresAction`: true if the service is not in the desired state.
  - `CurrentState`: use `pipeline.VerificationSatisfied/Failed/Unknown`.
  - `Diff`: optional human-readable summary.
  - `InternalData`: store any data needed for Apply (e.g., desired action).

## 6. Implement Apply

- Use the internal data from Evaluate when possible to avoid recomputation.
- Perform the necessary mutations (start/stop/enable service).
- Return a `StepResult` with status (`pipeline.StatusSuccess`, etc.), message,
  and `Changed` flag.
- Wrap operational errors with `pipeline.NewExecutionError` or similar.

## 7. Register the Plugin

Modify `cmd/streamy/plugins_import.go` to include the constructor:

```go
var portPluginFactories = []portPluginFactory{
    // ... existing entries
    {name: "service", factory: serviceplugin.New},
}
```

The CLI will now load the plugin at startup.

## 8. Write Tests

- **Unit tests** in `internal/plugins/service/plugin_test.go` covering:
  - Config decoding and validation errors.
  - Evaluate scenarios (service running, stopped, missing).
  - Apply outcomes (success, failure, idempotency).
- **Contract test**: `go test ./internal/plugins` automatically executes the
  shared cancellation + metadata suite.
- **Integration test** (optional) under `tests/` when behaviour interacts with
  multiple plugins or orchestrations.

## 9. Update Harness (if needed)

If integration tests or custom builds require extra registration, update
`tests/harness.go` to include the plugin in `builtinPortPlugins()`.

## 10. Document Usage

- Add an example to the README or docs if the plugin will ship in the CLI by
  default.
- Provide sample YAML under `testdata/` if helpful.

## Checklist Recap

- [ ] New package under `internal/plugins/<type>` with `New()` constructor.
- [ ] `Metadata()` returns valid `plugin.Metadata`.
- [ ] Config decoding converts `map[string]any` to typed structs with domain
      errors.
- [ ] `Evaluate` is read-only and honours context cancellation.
- [ ] `Apply` is idempotent, returns structured results, and handles
      cancellation.
- [ ] Unit tests cover happy path + edge cases.
- [ ] Plugin registered in the CLI/bootstrapper.
- [ ] Integration tests/harness updated if necessary.
- [ ] Documentation updated to describe configuration and behaviour.

With these steps, the new plugin will participate in the ports-native registry
and inherit observability (logging, metrics, tracing) from the executor.

