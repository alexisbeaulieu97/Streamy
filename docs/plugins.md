# Plugin Development Guide

Streamy executes environment setup steps through plugins. Each step type provides a plugin that implements the core interfaces and registers itself with the runtime registry. This guide explains the updated dependency-aware registry, describes the plugin contracts, and walks through building, testing, and documenting a new plugin.

## Plugin Interfaces

Every plugin must satisfy `internal/plugin.Plugin` and should expose the richer metadata via `MetadataProvider`. Plugins that need registry access during startup can optionally implement `PluginInitializer`.

```go
// Core plugin interface with unified Evaluate/Apply model
type Plugin interface {
    Metadata() Metadata
    Schema() interface{}

    // Evaluate performs a STRICTLY READ-ONLY assessment of the system's
    // current state against the desired state defined in the step configuration.
    //
    // CRITICAL CONTRACT: This method MUST NOT mutate any system state.
    // It should only read current state and compute what changes (if any)
    // would be needed to reach the desired state.
    //
    // Returns:
    //   - EvaluationResult: Rich state assessment including current status,
    //     whether action is required, human-readable message, optional diff,
    //     and optional internal data to pass to Apply()
    //   - error: PluginError (ValidationError, ExecutionError, StateError)
    //     if evaluation cannot be completed
    Evaluate(ctx context.Context, step *config.Step) (*model.EvaluationResult, error)

    // Apply mutates the system to match the desired state defined in the
    // step configuration. This method is ONLY called by the engine if
    // Evaluate() reported RequiresAction = true.
    //
    // The evalResult parameter contains the result from the previous
    // Evaluate() call, including InternalData that can be used to avoid
    // redundant computation.
    //
    // This method MUST be idempotent: calling it multiple times with the
    // same inputs should produce the same final state.
    //
    // Returns:
    //   - StepResult: Outcome of the apply operation (success, failure, etc.)
    //   - error: PluginError if the operation fails
    Apply(ctx context.Context, evalResult *model.EvaluationResult, step *config.Step) (*model.StepResult, error)
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
2. **Implement the Evaluate/Apply interface methods.** Call `step.DecodeConfig(&typedConfig)` to access plugin-specific configuration.
3. **Design EvaluationResult with InternalData.** Store computation results in InternalData to avoid redundant work in Apply().
4. **Ensure Evaluate() is strictly read-only.** This method MUST NOT mutate system state.
5. **Make Apply() idempotent.** Use the evalResult parameter to avoid recomputation and ensure consistent results.
6. **Expose dependency metadata.** Implement `PluginMetadata()` and use `plugin.MustParseVersionConstraint("1.x")` when pinning versions.
7. **Optional:** Implement `Init(*PluginRegistry)` to capture the registry or eagerly resolve dependencies.
8. **Wrap errors** using helpers from `internal/plugin/errors` to provide structured error types.
9. **Add unit tests** alongside the plugin. Include contract tests that verify read-only behavior and idempotency.
10. **Add integration coverage** under `tests/` when introducing new dependency patterns.
11. **Update documentation** (`docs/schema.md`, this guide, and feature-specific docs) with usage examples.

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

## Evaluate/Apply Interface Patterns

The new unified interface provides clear separation of concerns between read-only evaluation and stateful application:

### Read-Only Evaluation Pattern

```go
func (p *myPlugin) Evaluate(ctx context.Context, step *config.Step) (*model.EvaluationResult, error) {
    // Read current state ONLY - no mutations
    currentState := readCurrentState(step)

    // Determine what action is needed
    requiresAction := !isStateCorrect(currentState, step)

    // Store computation results for Apply() to reuse
    internalData := &myEvaluationData{
        currentState: currentState,
        computedValue: expensiveComputation(step),
    }

    return &model.EvaluationResult{
        StepID:         step.ID,
        CurrentState:   mapToVerificationStatus(currentState),
        RequiresAction: requiresAction,
        Message:        generateEvaluationMessage(currentState, step),
        Diff:           generateDiff(currentState, step),
        InternalData:   internalData,
    }, nil
}
```

### Efficient Apply Pattern

```go
func (p *myPlugin) Apply(ctx context.Context, evalResult *model.EvaluationResult, step *config.Step) (*model.StepResult, error) {
    // Skip if no action needed
    if !evalResult.RequiresAction {
        return &model.StepResult{
            StepID:  step.ID,
            Status:  model.StatusSkipped,
            Message: "no changes needed",
        }, nil
    }

    // Use pre-computed data from Evaluate()
    data := evalResult.InternalData.(*myEvaluationData)

    // Apply changes using efficient data
    err := applyChanges(ctx, step, data)
    if err != nil {
        return nil, plugin.NewExecutionError(step.ID, err)
    }

    return &model.StepResult{
        StepID:  step.ID,
        Status:  model.StatusSuccess,
        Message: "changes applied successfully",
    }, nil
}
```

### Contract Testing Pattern

```go
func TestMyPlugin_Contract(t *testing.T) {
    t.Run("Evaluate is read-only", func(t *testing.T) {
        plugin := New()
        step := createTestStep()

        // Take snapshot before evaluation
        before := captureSystemState()

        result, err := plugin.Evaluate(context.Background(), step)
        require.NoError(t, err)

        // Verify system unchanged
        after := captureSystemState()
        assert.Equal(t, before, after, "Evaluate() should not modify system")
    })

    t.Run("Apply uses evaluation data", func(t *testing.T) {
        plugin := New()
        step := createTestStep()

        // First evaluate
        evalResult, err := plugin.Evaluate(context.Background(), step)
        require.NoError(t, err)

        // Then apply
        stepResult, err := plugin.Apply(context.Background(), evalResult, step)
        require.NoError(t, err)

        // Verify success
        assert.Equal(t, model.StatusSuccess, stepResult.Status)
    })
}
```

## Migration Notes

The interface has been simplified from the old Check/DryRun/Verify methods to the unified Evaluate/Apply model:

### From Old Interface to New

**Old methods (removed):**
- `Check(ctx, step) (bool, error)` → Use `Evaluate()` and check `CurrentState == StatusSatisfied`
- `DryRun(ctx, step) (*StepResult, error)` → Use `Evaluate()` and check `RequiresAction`
- `Verify(ctx, step) (*VerificationResult, error)` → Use `Evaluate()` and use returned fields directly
- `Apply(ctx, step) (*StepResult, error)` → Now requires `evalResult` parameter

**Migration steps:**
1. **Replace Check() with Evaluate()**: Convert boolean return to VerificationStatus enum
2. **Remove DryRun()**: Use Evaluate() to determine what would change
3. **Remove Verify()**: Use Evaluate() for all read-only assessments
4. **Update Apply()**: Add evalResult parameter and use InternalData for efficiency
5. **Add contract tests**: Verify read-only behavior and idempotency

### Benefits of New Interface

1. **Clear separation**: Read-only evaluation vs stateful application
2. **Efficiency**: InternalData avoids recomputation between Evaluate and Apply
3. **Rich feedback**: EvaluationResult provides detailed state information
4. **Consistent error handling**: All operations use structured PluginError types
5. **Better testing**: Contract tests validate interface guarantees

Legacy plugins that only implement `Metadata()` continue to work; the registry synthesizes default metadata with no dependencies. To migrate:
1. Add `PluginMetadata()` returning the enriched metadata.
2. Replace global registration with explicit inclusion in `RegisterPlugins()`.
3. Update Check/DryRun/Verify methods to Evaluate/Apply as shown above.
4. Optionally add an `Init` hook if the plugin needs orchestrated startup.

Adhering to these guidelines keeps plugins composable, makes dependencies explicit, and allows Streamy to validate and initialise the graph before executing user workflows.
