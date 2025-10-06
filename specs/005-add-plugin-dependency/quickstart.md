# Quickstart: Plugin Dependency Registry

**Feature**: 005-add-plugin-dependency  
**Date**: October 6, 2025  
**Audience**: Plugin developers and Streamy core contributors

## Overview
This quickstart demonstrates how to use the Plugin Dependency Registry to create composable plugins. You'll learn how to declare dependencies, use dependent plugins, and handle errors.

---

## Prerequisites
- Streamy repository cloned
- Go 1.21+ installed
- Familiarity with Streamy plugin interface

---

## Scenario: Create a Shell Profile Plugin

We'll create a `shell_profile` plugin that depends on the existing `line_in_file` plugin to manage shell configuration files.

### Step 1: Define Plugin Metadata with Dependencies

Create `internal/plugins/shellprofile/plugin.go`:

```go
package shellprofile

import (
    "github.com/alexisbeaulieu97/streamy/internal/plugin"
)

type ShellProfilePlugin struct {
    registry *plugin.PluginRegistry
}

// Metadata declares dependencies
func (p *ShellProfilePlugin) Metadata() plugin.PluginMetadata {
    return plugin.PluginMetadata{
        Name:       "shell_profile",
        Version:    "1.0.0",
        APIVersion: "1.x",
        Dependencies: []plugin.Dependency{
            {
                Name: "line_in_file",
                VersionConstraint: plugin.MustParseVersionConstraint("1.x"),
            },
        },
        Stateful:    false,
        Description: "Manages shell profile configuration files",
    }
}
```

**Key Points**:
- `Dependencies` field lists required plugins
- `VersionConstraint` ensures compatibility (major version `1.x`)
- `Stateful: false` means shared singleton instance (default)

---

### Step 2: Implement Plugin Initializer

Add the `Init()` method to receive the registry:

```go
// Init stores registry reference for dependency access
func (p *ShellProfilePlugin) Init(registry *plugin.PluginRegistry) error {
    p.registry = registry
    
    // Verify dependency is available
    _, err := registry.Get("line_in_file")
    if err != nil {
        return fmt.Errorf("required dependency missing: %w", err)
    }
    
    return nil
}
```

**Key Points**:
- Store `registry` reference for later use
- Optionally validate dependencies are accessible
- Return error if critical dependencies missing

---

### Step 3: Use Dependencies in Apply()

Implement `Apply()` using the dependent plugin:

```go
func (p *ShellProfilePlugin) Apply(step config.StepConfig) (model.Result, error) {
    // Get dependent plugin via registry
    lineInFile, err := p.registry.GetForDependent("shell_profile", "line_in_file")
    if err != nil {
        return model.Result{}, fmt.Errorf("failed to access line_in_file: %w", err)
    }
    
    // Compose configuration for dependent plugin
    bashrcPath := step.Params["profile_path"].(string)
    exportLine := fmt.Sprintf("export PATH=\"$PATH:%s\"", step.Params["path"])
    
    lineConfig := config.StepConfig{
        ID:     step.ID + "_line",
        Plugin: "line_in_file",
        Params: map[string]interface{}{
            "file":  bashrcPath,
            "line":  exportLine,
            "state": "present",
        },
    }
    
    // Invoke dependent plugin
    return lineInFile.Apply(lineConfig)
}
```

**Key Points**:
- Use `GetForDependent()` to enforce dependency declarations
- Compose configuration for dependent plugin
- Delegate execution to dependent plugin's `Apply()`

---

### Step 4: Implement Schema and Verify

Complete the plugin interface:

```go
func (p *ShellProfilePlugin) Schema() plugin.PluginSchema {
    return plugin.PluginSchema{
        Params: map[string]plugin.ParamSpec{
            "profile_path": {Type: "string", Required: true, Description: "Path to shell profile file"},
            "path":         {Type: "string", Required: true, Description: "Directory to add to PATH"},
        },
    }
}

func (p *ShellProfilePlugin) Verify(step config.StepConfig) (model.VerificationStatus, error) {
    // Delegate verification to line_in_file
    lineInFile, err := p.registry.GetForDependent("shell_profile", "line_in_file")
    if err != nil {
        return model.VerificationFailed, err
    }
    
    // Build line_in_file verification config
    lineConfig := config.StepConfig{
        ID:     step.ID + "_line",
        Plugin: "line_in_file",
        Params: map[string]interface{}{
            "file":  step.Params["profile_path"],
            "line":  fmt.Sprintf("export PATH=\"$PATH:%s\"", step.Params["path"]),
            "state": "present",
        },
    }
    
    return lineInFile.Verify(lineConfig)
}
```

---

### Step 5: Register Plugin in Core

Update `cmd/streamy/plugins_import.go`:

```go
import (
    // ... existing imports
    "github.com/alexisbeaulieu97/streamy/internal/plugins/shellprofile"
)

func RegisterPlugins(registry *plugin.PluginRegistry) error {
    plugins := []plugin.Plugin{
        // ... existing plugins
        &shellprofile.ShellProfilePlugin{},
    }
    
    for _, p := range plugins {
        if err := registry.Register(p); err != nil {
            return fmt.Errorf("failed to register plugin %s: %w", p.Metadata().Name, err)
        }
    }
    
    return nil
}
```

---

### Step 6: Initialize Registry in Main

Update `cmd/streamy/main.go`:

```go
func main() {
    logger := logger.New()
    
    // Create registry with environment-aware defaults
    registryConfig := plugin.DefaultConfig()
    registry := plugin.NewPluginRegistry(registryConfig, logger)
    
    // Register all plugins
    if err := RegisterPlugins(registry); err != nil {
        logger.Fatal().Err(err).Msg("Plugin registration failed")
    }
    
    // Validate dependencies
    if err := registry.ValidateDependencies(); err != nil {
        logger.Fatal().Err(err).Msg("Dependency validation failed")
    }
    
    // Initialize plugins in dependency order
    if err := registry.InitializePlugins(); err != nil {
        logger.Fatal().Err(err).Msg("Plugin initialization failed")
    }
    
    logger.Info().Msgf("Loaded %d plugins", len(registry.List()))
    
    // ... rest of application logic
}
```

---

## Testing Your Plugin

### Unit Test

Create `internal/plugins/shellprofile/plugin_test.go`:

```go
package shellprofile

import (
    "testing"
    "github.com/alexisbeaulieu97/streamy/internal/plugin"
    "github.com/stretchr/testify/assert"
)

func TestMetadata(t *testing.T) {
    p := &ShellProfilePlugin{}
    meta := p.Metadata()
    
    assert.Equal(t, "shell_profile", meta.Name)
    assert.Equal(t, 1, len(meta.Dependencies))
    assert.Equal(t, "line_in_file", meta.Dependencies[0].Name)
}

func TestInit(t *testing.T) {
    registry := plugin.NewPluginRegistry(plugin.DefaultConfig(), testLogger)
    
    // Register dependencies first
    registry.Register(&lineinfile.LineInFilePlugin{})
    registry.ValidateDependencies()
    registry.InitializePlugins()
    
    // Register and init our plugin
    p := &ShellProfilePlugin{}
    err := registry.Register(p)
    assert.NoError(t, err)
    
    err = p.Init(registry)
    assert.NoError(t, err)
    assert.NotNil(t, p.registry)
}
```

### Integration Test

Create `tests/plugin_composition_test.go`:

```go
package tests

import (
    "testing"
    "github.com/alexisbeaulieu97/streamy/internal/config"
    "github.com/alexisbeaulieu97/streamy/internal/plugin"
    "github.com/stretchr/testify/assert"
)

func TestShellProfileComposition(t *testing.T) {
    // Setup registry with both plugins
    registry := setupTestRegistry(t)
    
    // Get shell_profile plugin
    shellProfile, err := registry.Get("shell_profile")
    assert.NoError(t, err)
    
    // Create test step
    step := config.StepConfig{
        ID:     "add_path",
        Plugin: "shell_profile",
        Params: map[string]interface{}{
            "profile_path": "/tmp/test_bashrc",
            "path":         "/opt/custom/bin",
        },
    }
    
    // Apply should delegate to line_in_file
    result, err := shellProfile.Apply(step)
    assert.NoError(t, err)
    assert.True(t, result.Changed)
    
    // Verify should confirm line exists
    status, err := shellProfile.Verify(step)
    assert.NoError(t, err)
    assert.Equal(t, model.VerificationPassed, status)
}
```

---

## Running the Tests

```bash
# Unit tests
go test ./internal/plugins/shellprofile/...

# Integration tests
go test ./tests/plugin_composition_test.go

# All tests
go test ./...
```

---

## Configuration Usage

Create a Streamy config using your new plugin:

```yaml
# streamy.yml
version: "1.0"

steps:
  - id: setup_dev_path
    plugin: shell_profile
    params:
      profile_path: ~/.bashrc
      path: ~/dev/bin
    depends_on: []
```

Run with:
```bash
./streamy apply streamy.yml
```

Expected output:
```
[INFO] Loaded 15 plugins
[INFO] Validating dependencies...
[INFO] Initializing plugins in dependency order...
[INFO] Running step: setup_dev_path
[INFO] shell_profile → line_in_file: Adding line to ~/.bashrc
[INFO] ✓ setup_dev_path completed (changed=true)
```

---

## Error Handling Examples

### Missing Dependency

If `line_in_file` is not registered:

```
[FATAL] Dependency validation failed: plugin 'shell_profile' declares dependency 'line_in_file' which is not registered
```

### Version Conflict

If `shell_profile` requires `1.x` but `line_in_file` is version `2.0.0`:

```
[FATAL] Dependency validation failed: version conflict for plugin 'line_in_file' (version 2.0.0):
  shell_profile requires 1.x
Suggestion: Update plugin versions to satisfy all constraints
```

### Undeclared Access (Strict Mode)

If plugin attempts to use dependency without declaring it:

```
[ERROR] plugin 'shell_profile' attempted to access undeclared dependency 'template'
Add 'template' to Dependencies in Metadata()
```

---

## Advanced: Stateful Plugins

For plugins that maintain state, set `Stateful: true`:

```go
func (p *StatefulPlugin) Metadata() plugin.PluginMetadata {
    return plugin.PluginMetadata{
        Name:     "stateful_example",
        Version:  "1.0.0",
        Stateful: true,  // Each dependent gets own instance
        // ...
    }
}
```

This creates per-dependent instances instead of shared singleton.

---

## Advanced: Transitive Dependencies

Plugins can depend on plugins that have dependencies:

```
shell_profile → line_in_file → file_utils
```

The registry automatically resolves the full chain and initializes in correct order:
```
1. file_utils.Init()
2. line_in_file.Init()
3. shell_profile.Init()
```

---

## Troubleshooting

### "Circular dependency detected"

**Symptom**: Registry fails validation with cycle error

**Solution**: Remove or refactor dependencies to break the cycle

```
A → B → C → A  (BAD)
A → B
C → B         (GOOD - both depend on B)
```

### "Required dependency missing"

**Symptom**: Plugin Init() fails

**Solution**: Ensure dependency is registered before dependent plugin

### "Undeclared dependency accessed"

**Symptom**: Warning or error when calling GetForDependent()

**Solution**: Add dependency to Metadata().Dependencies

---

## Best Practices

1. **Declare All Dependencies**: Add every plugin you use to Dependencies
2. **Version Constraints**: Use major version matching (`1.x`) for stability
3. **Stateless by Default**: Only set `Stateful: true` if plugin maintains mutable state
4. **Error Propagation**: Return detailed errors with context
5. **Composability**: Design plugins for reuse by other plugins
6. **Testing**: Test both standalone and composition scenarios

---

## Next Steps

- Review existing plugins for composition opportunities
- Update plugin documentation with dependency information
- Explore creating composite plugins for common workflows
- Contribute new composable plugins to Streamy ecosystem

---

## References
- API Contract: `contracts/registry-api.md`
- Data Model: `data-model.md`
- Feature Spec: `spec.md`
- Plugin Development Guide: `docs/plugins.md`
