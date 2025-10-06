# Plugin Development Guide

Streamy executes environment setup steps through plugins. Each step type provides a plugin that implements the core interfaces and registers itself with the runtime registry. This guide explains the updated dependency-aware registry, describes the plugin contracts, and walks through building, testing, and documenting a new plugin.

## Plugin Interfaces

Every plugin must satisfy `internal/plugin.Plugin` and should expose the richer metadata via `MetadataProvider`. Plugins that need registry access during startup can optionally implement `PluginInitializer`.

```go
// Legacy surface used by the executor for backwards compatibility.
type Plugin interface {
    Metadata() Metadata
    Schema() interface{}
    Check(ctx context.Context, step *config.Step) (bool, error)
    Apply(ctx context.Context, step *config.Step) (*model.StepResult, error)
    DryRun(ctx context.Context, step *config.Step) (*model.StepResult, error)
    Verify(ctx context.Context, step *config.Step) (*model.VerificationResult, error)
}

// Dependency-aware metadata consumed by the PluginRegistry.
type MetadataProvider interface {
    PluginMetadata() PluginMetadata
}

// Optional hook for receiving the registry after validation.
type PluginInitializer interface {
    Init(registry *PluginRegistry) error
}
```

### PluginMetadata Structure

```go
type PluginMetadata struct {
    Name         string
    Version      string
    APIVersion   string
    Dependencies []Dependency
    Stateful     bool
    Description  string
}

type Dependency struct {
    Name              string
    VersionConstraint *VersionConstraint // e.g. MustParseVersionConstraint("1.x")
}
```

Key points:
- **Name** acts as the registry identifier. It should match the step `type` string.
- **Version** follows semantic versioning (`X.Y.Z`).
- **APIVersion** uses the major-version placeholder format (`N.x`) to express compatibility with the registry contract.
- **Dependencies** declare other plugins required at runtime. Use `VersionConstraint` to pin a major version.
- **Stateful** indicates whether dependents receive dedicated instances (`true`) or a shared singleton (`false`).
- **Description** appears in debugging/logging output.

Implement `PluginMetadata()` to supply these fields. The registry validates metadata, detects version conflicts, and computes initialization order automatically.

### Optional Initialisation

Plugins that need access to their dependencies during setup can implement `PluginInitializer`. The registry calls `Init(registry)` in topological order after dependency validation succeeds. Use `registry.GetForDependent("<caller>", "<dependency>")` inside `Init` to retrieve dependent plugins safely.

## Registration Workflow

At process startup (`cmd/streamy/main.go`) Streamy creates a `PluginRegistry`, registers all built-in plugins, validates dependencies, and calls `InitializePlugins()`:

```go
log, _ := logger.New(logger.Options{Level: "info", HumanReadable: true})
registry := plugin.NewPluginRegistry(plugin.DefaultConfig(), log)
if err := RegisterPlugins(registry, log); err != nil {
    panic(err)
}
```

`RegisterPlugins` (see `cmd/streamy/plugins_import.go`) instantiates each built-in plugin via its `New()` constructor and registers it with the registry. Custom builds can reuse this helper or extend it with additional plugins before executing commands.

When authoring new plugins:
1. Export a `New()` constructor returning the concrete plugin implementation.
2. Implement `PluginMetadata()` to supply dependency and version information.
3. Ensure `New()` does **not** mutate global state; construction should be side-effect free.
4. Wire the plugin into `RegisterPlugins()` (or a feature-specific registrar) so the CLI can register it during startup.

## Example: line_in_file Plugin

`internal/plugins/lineinfile/lineinfile.go` demonstrates the new metadata contract:

```go
func (p *lineInFilePlugin) PluginMetadata() plugin.PluginMetadata {
    return plugin.PluginMetadata{
        Name:         "line_in_file",
        Version:      "1.0.0",
        APIVersion:   "1.x",
        Dependencies: []plugin.Dependency{}, // no registry dependencies
        Stateful:     false,
        Description:  "Manages ensuring specific lines exist within files.",
    }
}
```

Because the plugin is stateless it returns a singleton instance. If a plugin maintains per-dependent state, set `Stateful: true`; the registry will hand each dependent a dedicated copy created via reflection (the plugin must be pointer-to-struct for this to work).

## Implementation Checklist

1. **Define the Plugin struct and `New()` constructor.** Keep construction free of side effects.
2. **Implement the interface methods.** Use the type-specific fields on `config.Step`.
3. **Expose dependency metadata.** Implement `PluginMetadata()` and use `plugin.MustParseVersionConstraint("1.x")` when pinning versions.
4. **Optional:** Implement `Init(*PluginRegistry)` to capture the registry or eagerly resolve dependencies.
5. **Handle idempotency.** `Check` and `DryRun` should make it safe to run `Apply` repeatedly.
6. **Wrap errors** using helpers from `pkg/errors` to provide remediation hints.
7. **Add unit tests** alongside the plugin. Table-driven tests should cover happy paths, idempotency, error handling, and (if present) `Init` behaviour.
8. **Add integration coverage** under `tests/` when introducing new dependency patterns.
9. **Update documentation** (`docs/schema.md`, this guide, and feature-specific docs) with usage examples.

## Testing Plugins

- **Unit**: `go test ./internal/plugins/<type>`
- **Registry Contract**: Extend `internal/plugin/registry_test.go` or create feature-specific tests using `internal/plugin/mock_plugin_test.go` for helpers.
- **Integration**: Add scenarios to `tests/integration_plugin_dependency_test.go` when validating cross-plugin behaviour.
- **Performance**: Benchmarks live in `internal/plugin/registry_perf_test.go` to guard lookup and validation overhead.

## Registry Access Patterns

Plugins should access peer plugins through the registry only after successful validation:

```go
func (p *shellProfilePlugin) Init(reg *plugin.PluginRegistry) error {
    dep, err := reg.GetForDependent("shell_profile", "line_in_file")
    if err != nil {
        return err
    }
    p.lineInFile = dep
    return nil
}
```

`GetForDependent` enforces declared dependencies and honours the configured access policy:
- **strict**: returns `ErrUndeclaredDependency`
- **warn**: logs a warning but returns the dependency
- **off**: bypasses enforcement (use sparingly)

## Migration Notes

Legacy plugins that only implement `Metadata()` continue to work; the registry synthesizes default metadata with no dependencies. To migrate:
1. Add `PluginMetadata()` returning the enriched metadata.
2. Replace global registration with explicit inclusion in `RegisterPlugins()`.
3. Optionally add an `Init` hook if the plugin needs orchestrated startup.

Adhering to these guidelines keeps plugins composable, makes dependencies explicit, and allows Streamy to validate and initialise the graph before executing user workflows.
