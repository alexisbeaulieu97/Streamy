# Plugin Registry API Contract

**Feature**: 005-add-plugin-dependency  
**Date**: October 6, 2025

## Overview
This document defines the public API contract for the PluginRegistry system. All methods are part of the `internal/plugin` package and consumed by Streamy core and plugin implementations.

---

## PluginRegistry Interface

### Constructor

```go
// NewPluginRegistry creates a new plugin registry with the given configuration
func NewPluginRegistry(config *RegistryConfig, logger *logger.Logger) *PluginRegistry
```

**Parameters**:
- `config`: Registry configuration (dependency policy, access policy)
- `logger`: Structured logger for diagnostics

**Returns**: Initialized registry (empty, ready for registration)

**Preconditions**: None

**Postconditions**: Registry in Empty state, ready to accept plugin registrations

---

### Register

```go
// Register adds a plugin to the registry and updates the dependency graph
func (r *PluginRegistry) Register(p Plugin) error
```

**Parameters**:
- `p`: Plugin instance to register

**Returns**: 
- `nil` on success
- `error` if registration fails (duplicate name, invalid metadata)

**Preconditions**: 
- Plugin not already registered
- Plugin metadata valid (non-empty name, valid version)

**Postconditions**:
- Plugin added to registry
- Plugin's dependencies added to dependency graph

**Error Cases**:
- Duplicate plugin name → `ErrDuplicatePlugin`
- Invalid metadata → `ErrInvalidMetadata`

**Example**:
```go
registry := NewPluginRegistry(config, logger)
plugin := &MyPlugin{}
err := registry.Register(plugin)
if err != nil {
    log.Fatal(err)
}
```

---

### ValidateDependencies

```go
// ValidateDependencies checks all declared dependencies and detects cycles
func (r *PluginRegistry) ValidateDependencies() error
```

**Parameters**: None

**Returns**:
- `nil` if all dependencies valid
- `error` describing validation failures (missing deps, cycles, version conflicts)

**Preconditions**: All plugins registered

**Postconditions**:
- Dependency graph validated
- If policy=graceful: affected plugins disabled, warnings logged
- If policy=strict: error returned, registry unusable

**Error Cases**:
- Missing dependency → `ErrMissingDependency`
- Circular dependency → `ErrCircularDependency`
- Version conflict → `ErrVersionConflict`

**Behavior by Policy**:
- `PolicyStrict`: Return first error encountered, stop validation
- `PolicyGraceful`: Collect all errors, disable affected plugins, log warnings, return nil

**Example**:
```go
// After registering all plugins
err := registry.ValidateDependencies()
if err != nil {
    log.Fatalf("Dependency validation failed: %v", err)
}
```

---

### InitializePlugins

```go
// InitializePlugins calls Init() on all plugins in dependency order
func (r *PluginRegistry) InitializePlugins() error
```

**Parameters**: None

**Returns**:
- `nil` if all initializations successful
- `error` if any plugin initialization fails

**Preconditions**: ValidateDependencies() completed successfully

**Postconditions**:
- All plugins initialized in topological order
- Plugins can now access dependencies via registry
- Registry in Ready state

**Error Cases**:
- Plugin Init() returns error → propagate error
- Topological sort fails → `ErrCircularDependency` (should be caught by ValidateDependencies)

**Example**:
```go
err := registry.InitializePlugins()
if err != nil {
    log.Fatalf("Plugin initialization failed: %v", err)
}
```

---

### Get

```go
// Get retrieves a plugin by name
func (r *PluginRegistry) Get(name string) (Plugin, error)
```

**Parameters**:
- `name`: Unique plugin identifier

**Returns**:
- Plugin instance
- `ErrPluginNotFound` if plugin doesn't exist

**Preconditions**: Registry in Ready state (after InitializePlugins)

**Postconditions**: None (read-only operation)

**Thread Safety**: Safe for concurrent calls (uses RLock)

**Example**:
```go
plugin, err := registry.Get("line_in_file")
if err != nil {
    return err
}
result, err := plugin.Apply(stepConfig)
```

---

### GetForDependent

```go
// GetForDependent retrieves a plugin with enforcement of declared dependencies
func (r *PluginRegistry) GetForDependent(dependentName, pluginName string) (Plugin, error)
```

**Parameters**:
- `dependentName`: Name of plugin requesting the dependency
- `pluginName`: Name of dependency plugin

**Returns**:
- Plugin instance (singleton or per-dependent based on Stateful flag)
- `ErrPluginNotFound` if plugin doesn't exist
- `ErrUndeclaredDependency` if access policy is strict and dependency not declared

**Preconditions**: Registry in Ready state

**Postconditions**:
- If dependency is stateful: per-dependent instance created if not exists
- If access policy=warn and undeclared: warning logged

**Behavior by Access Policy**:
- `AccessStrict`: Return error if dependency not declared in dependent's metadata
- `AccessWarn`: Log warning if undeclared, allow access
- `AccessOff`: Allow all access without checks

**Example**:
```go
// In a plugin's Apply() method
func (p *ShellProfilePlugin) Apply(step StepConfig) (Result, error) {
    lineInFile, err := p.registry.GetForDependent("shell_profile", "line_in_file")
    if err != nil {
        return Result{}, err
    }
    return lineInFile.Apply(stepConfig)
}
```

---

### List

```go
// List returns all registered plugin names
func (r *PluginRegistry) List() []string
```

**Parameters**: None

**Returns**: Slice of plugin names (sorted alphabetically)

**Preconditions**: None

**Postconditions**: None (read-only)

**Thread Safety**: Safe for concurrent calls

**Example**:
```go
plugins := registry.List()
fmt.Printf("Registered plugins: %v\n", plugins)
```

---

## PluginMetadata Interface

```go
// Metadata returns plugin metadata including dependencies
func (p Plugin) Metadata() PluginMetadata
```

**Returns**: `PluginMetadata` struct with:
- `Name`: Unique plugin identifier
- `Version`: Semantic version string
- `APIVersion`: Core API version compatibility
- `Dependencies`: List of required plugins
- `Stateful`: Whether plugin maintains mutable state
- `Description`: Human-readable description

**Preconditions**: None

**Postconditions**: None (read-only)

**Example**:
```go
func (p *ShellProfilePlugin) Metadata() PluginMetadata {
    return PluginMetadata{
        Name:    "shell_profile",
        Version: "1.0.0",
        APIVersion: "1.x",
        Dependencies: []Dependency{
            {Name: "line_in_file", VersionConstraint: ParseVersionConstraint("1.x")},
        },
        Stateful: false,
        Description: "Manages shell profile configuration",
    }
}
```

---

## PluginInitializer Interface (Optional)

```go
// Init is called after all plugins are registered and validated
// Registry can be stored for later dependency access
func (p Plugin) Init(registry *PluginRegistry) error
```

**Parameters**:
- `registry`: Reference to plugin registry for dependency lookup

**Returns**:
- `nil` if initialization successful
- `error` if initialization fails (propagated to caller)

**Preconditions**: 
- Registry has validated all dependencies
- All dependency plugins are registered

**Postconditions**:
- Plugin can access dependencies via registry
- Plugin ready for Apply/Verify operations

**Example**:
```go
type ShellProfilePlugin struct {
    registry *PluginRegistry
}

func (p *ShellProfilePlugin) Init(registry *PluginRegistry) error {
    p.registry = registry
    
    // Verify dependencies are available
    _, err := registry.Get("line_in_file")
    if err != nil {
        return fmt.Errorf("required dependency missing: %w", err)
    }
    
    return nil
}
```

---

## Dependency Struct

```go
type Dependency struct {
    Name              string
    VersionConstraint *VersionConstraint
}
```

**Fields**:
- `Name`: Required plugin name
- `VersionConstraint`: Optional version requirement (nil = any version)

**Example**:
```go
dependencies := []Dependency{
    {Name: "line_in_file", VersionConstraint: ParseVersionConstraint("1.x")},
    {Name: "template"},  // No version constraint
}
```

---

## VersionConstraint

```go
// ParseVersionConstraint parses a version constraint string
func ParseVersionConstraint(s string) (*VersionConstraint, error)

// Satisfies checks if a version satisfies this constraint
func (vc *VersionConstraint) Satisfies(version string) bool
```

**Supported Format**: `N.x` (major version matching)

**Examples**:
- `ParseVersionConstraint("1.x")` → matches `1.0.0`, `1.5.3`, `1.99.99`
- `ParseVersionConstraint("2.x")` → matches `2.0.0`, `2.1.0`

**Error Cases**:
- Invalid format → `ErrInvalidVersionConstraint`
- Non-numeric major version → `ErrInvalidVersionConstraint`

---

## Error Types

### ErrPluginNotFound
```go
type ErrPluginNotFound struct {
    Name string
}

func (e ErrPluginNotFound) Error() string
```

**When**: Registry lookup for non-existent plugin

**Message**: `"plugin 'NAME' not found in registry"`

---

### ErrCircularDependency
```go
type ErrCircularDependency struct {
    Cycle []string
}

func (e ErrCircularDependency) Error() string
```

**When**: Dependency graph contains cycle

**Message**: `"circular dependency detected: A -> B -> C -> A"`

---

### ErrVersionConflict
```go
type ErrVersionConflict struct {
    Plugin        string
    RequiredBy    map[string]string
    ActualVersion string
}

func (e ErrVersionConflict) Error() string
```

**When**: Incompatible version constraints

**Message**:
```
version conflict for plugin 'line_in_file' (version 2.0.0):
  shell_profile requires 1.x
  bash_config requires 1.x
Suggestion: Update plugin versions to satisfy all constraints
```

---

### ErrUndeclaredDependency
```go
type ErrUndeclaredDependency struct {
    Caller     string
    Dependency string
    Hint       string
}

func (e ErrUndeclaredDependency) Error() string
```

**When**: Plugin accesses undeclared dependency (strict mode)

**Message**:
```
plugin 'shell_profile' attempted to access undeclared dependency 'line_in_file'
Add 'line_in_file' to Dependencies in Metadata()
```

---

### ErrMissingDependency
```go
type ErrMissingDependency struct {
    Plugin     string
    Dependency string
}

func (e ErrMissingDependency) Error() string
```

**When**: Declared dependency doesn't exist in registry

**Message**: `"plugin 'shell_profile' declares dependency 'missing_plugin' which is not registered"`

---

## Configuration

### RegistryConfig
```go
type RegistryConfig struct {
    DependencyPolicy DependencyPolicy
    AccessPolicy     AccessPolicy
}
```

**Defaults** (environment-detected):
```go
func DefaultConfig() *RegistryConfig {
    policy := PolicyGraceful
    access := AccessWarn
    
    if isCI() {  // Check CI env vars
        policy = PolicyStrict
        access = AccessStrict
    }
    
    return &RegistryConfig{
        DependencyPolicy: policy,
        AccessPolicy:     access,
    }
}
```

---

## Contract Tests

All contract tests must pass before implementation is considered complete.

### Test: Register and Get
```go
func TestRegisterAndGet(t *testing.T) {
    registry := NewPluginRegistry(DefaultConfig(), logger)
    plugin := &MockPlugin{name: "test"}
    
    err := registry.Register(plugin)
    assert.NoError(t, err)
    
    retrieved, err := registry.Get("test")
    assert.NoError(t, err)
    assert.Equal(t, plugin, retrieved)
}
```

### Test: Missing Dependency Detection
```go
func TestMissingDependency(t *testing.T) {
    registry := NewPluginRegistry(DefaultConfig(), logger)
    plugin := &MockPlugin{
        name: "dependent",
        deps: []Dependency{{Name: "missing"}},
    }
    registry.Register(plugin)
    
    err := registry.ValidateDependencies()
    assert.Error(t, err)
    assert.IsType(t, ErrMissingDependency{}, err)
}
```

### Test: Circular Dependency Detection
```go
func TestCircularDependency(t *testing.T) {
    registry := NewPluginRegistry(DefaultConfig(), logger)
    
    pluginA := &MockPlugin{name: "A", deps: []Dependency{{Name: "B"}}}
    pluginB := &MockPlugin{name: "B", deps: []Dependency{{Name: "C"}}}
    pluginC := &MockPlugin{name: "C", deps: []Dependency{{Name: "A"}}}
    
    registry.Register(pluginA)
    registry.Register(pluginB)
    registry.Register(pluginC)
    
    err := registry.ValidateDependencies()
    assert.Error(t, err)
    assert.IsType(t, ErrCircularDependency{}, err)
}
```

### Test: Initialization Order
```go
func TestInitializationOrder(t *testing.T) {
    registry := NewPluginRegistry(DefaultConfig(), logger)
    order := []string{}
    
    pluginA := &MockPlugin{name: "A", onInit: func() { order = append(order, "A") }}
    pluginB := &MockPlugin{name: "B", deps: []Dependency{{Name: "A"}}, onInit: func() { order = append(order, "B") }}
    
    registry.Register(pluginA)
    registry.Register(pluginB)
    registry.ValidateDependencies()
    registry.InitializePlugins()
    
    assert.Equal(t, []string{"A", "B"}, order)
}
```

---

## Versioning

**API Version**: `1.x`  
**Breaking Changes**: Require major version bump and migration guide  
**Additive Changes**: Allowed in minor versions

---

## References
- Data Model: `data-model.md`
- Research: `research.md`
- Feature Spec: `spec.md`
