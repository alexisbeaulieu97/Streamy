# Data Model: Plugin Dependency Registry

**Feature**: 005-add-plugin-dependency  
**Date**: October 6, 2025  
**Status**: Complete

## Overview
This document defines the core data structures for the plugin dependency registry system. All types are in-memory structures with no persistence requirements.

---

## Core Entities

### 1. PluginRegistry

**Purpose**: Central repository for all registered plugins with dependency resolution capabilities.

**Attributes**:
```go
type PluginRegistry struct {
    mu                sync.RWMutex                    // Thread-safe access
    plugins           map[string]Plugin               // name → plugin instance
    dependencyGraph   *DependencyGraph                 // Dependency relationships
    statefulInstances map[string]map[string]Plugin    // [pluginName][dependentName] → instance
    logger            *logger.Logger                   // Structured logging
    config            *RegistryConfig                  // Configuration
}
```

**Relationships**:
- Contains 0..N `Plugin` instances
- Owns 1 `DependencyGraph`
- References external `logger.Logger`

**Validation Rules**:
- Plugin names must be unique within registry
- All plugin dependencies must exist in registry before initialization completes
- No circular dependencies allowed

**State Transitions**:
1. **Empty** → Plugins registered via `Register()`
2. **Registered** → Dependencies validated via `ValidateDependencies()`
3. **Validated** → Plugins initialized via `Init()` in topological order
4. **Ready** → Plugins available for lookup via `Get()`

---

### 2. PluginMetadata

**Purpose**: Descriptive information about a plugin including version and dependencies.

**Attributes**:
```go
type PluginMetadata struct {
    Name         string       // Unique plugin identifier (e.g., "line_in_file")
    Version      string       // Semantic version (e.g., "1.2.3")
    APIVersion   string       // Core plugin API version (e.g., "1.x")
    Dependencies []Dependency // List of required plugins
    Stateful     bool         // If true, create per-dependent instances
    Description  string       // Human-readable description
}
```

**Relationships**:
- Each `Plugin` has exactly 1 `PluginMetadata` (via `Metadata()` method)
- Contains 0..N `Dependency` declarations

**Validation Rules**:
- `Name` must be non-empty and unique
- `Version` must be valid semantic version format (`X.Y.Z`)
- `APIVersion` must be valid major version format (`X.x`)
- `Dependencies` list must not include self-reference

---

### 3. Dependency

**Purpose**: Declaration of a required plugin dependency with optional version constraint.

**Attributes**:
```go
type Dependency struct {
    Name              string             // Required plugin name
    VersionConstraint *VersionConstraint // Optional version requirement (nil = any version)
}
```

**Relationships**:
- Belongs to 1 `PluginMetadata`
- References 1 plugin by name (must exist in registry)
- May have 0..1 `VersionConstraint`

**Validation Rules**:
- `Name` must reference an existing plugin in registry
- If `VersionConstraint` is specified, referenced plugin's version must satisfy it

---

### 4. VersionConstraint

**Purpose**: Specification of acceptable version range using major version matching.

**Attributes**:
```go
type VersionConstraint struct {
    MajorVersion int // Required major version (e.g., 1 for "1.x")
}
```

**Methods**:
```go
// Parse version constraint from string (e.g., "1.x" → VersionConstraint{MajorVersion: 1})
func ParseVersionConstraint(s string) (*VersionConstraint, error)

// Check if a semantic version satisfies this constraint
func (vc *VersionConstraint) Satisfies(version string) bool
```

**Validation Rules**:
- `MajorVersion` must be non-negative integer
- Only major version matching supported (format: `N.x`)

**Examples**:
- `1.x` matches `1.0.0`, `1.5.3`, `1.99.99` but not `2.0.0`
- `2.x` matches `2.0.0`, `2.1.0` but not `1.9.9` or `3.0.0`

---

### 5. DependencyGraph

**Purpose**: Directed acyclic graph (DAG) representing plugin dependency relationships.

**Attributes**:
```go
type DependencyGraph struct {
    nodes    map[string]bool            // Set of plugin names
    incoming map[string][]string        // node → list of nodes that depend on it
    outgoing map[string][]string        // node → list of nodes it depends on
}
```

**Methods**:
```go
// Add a plugin node to the graph
func (g *DependencyGraph) AddNode(name string)

// Add a dependency edge from dependent to dependency
func (g *DependencyGraph) AddEdge(dependent, dependency string)

// Detect circular dependencies, returns cycle if found
func (g *DependencyGraph) DetectCycles() ([]string, error)

// Return plugins in dependency order (dependencies before dependents)
func (g *DependencyGraph) TopologicalSort() ([]string, error)
```

**Validation Rules**:
- Graph must be acyclic (no circular dependencies)
- All edges must reference existing nodes
- Topological sort must produce a total ordering

**Algorithms**:
- **Cycle Detection**: DFS with recursion stack tracking
- **Topological Sort**: Kahn's algorithm (BFS with in-degree tracking)

---

### 6. RegistryConfig

**Purpose**: Configuration options for registry behavior.

**Attributes**:
```go
type RegistryConfig struct {
    DependencyPolicy DependencyPolicy // strict | graceful
    AccessPolicy     AccessPolicy     // strict | warn | off
}

type DependencyPolicy string
const (
    PolicyStrict   DependencyPolicy = "strict"   // Abort on dependency errors
    PolicyGraceful DependencyPolicy = "graceful" // Skip affected plugins, continue
)

type AccessPolicy string
const (
    AccessStrict AccessPolicy = "strict" // Error on undeclared dependency access
    AccessWarn   AccessPolicy = "warn"   // Log warning, allow access
    AccessOff    AccessPolicy = "off"    // No enforcement
)
```

**Default Values**:
- `DependencyPolicy`: `graceful` for CLI/TUI, `strict` for CI/automation (environment-detected)
- `AccessPolicy`: `strict` for CI/automation, `warn` for CLI/TUI/debug

---

## Error Types

### ErrPluginNotFound
```go
type ErrPluginNotFound struct {
    Name string
}
```
**Trigger**: Registry lookup for non-existent plugin  
**Resolution**: Verify plugin is registered, check spelling

### ErrCircularDependency
```go
type ErrCircularDependency struct {
    Cycle []string // Ordered list of plugins in cycle
}
```
**Trigger**: Dependency graph contains cycle  
**Resolution**: Remove or refactor dependencies to break cycle

### ErrVersionConflict
```go
type ErrVersionConflict struct {
    Plugin        string
    RequiredBy    map[string]string // dependent → constraint
    ActualVersion string
}
```
**Trigger**: Multiple plugins require incompatible versions of same dependency  
**Resolution**: Update plugin versions to satisfy all constraints

### ErrUndeclaredDependency
```go
type ErrUndeclaredDependency struct {
    Caller     string
    Dependency string
    Hint       string
}
```
**Trigger**: Plugin attempts to access dependency not declared in metadata  
**Resolution**: Add dependency to `Metadata().Dependencies`

### ErrMissingDependency
```go
type ErrMissingDependency struct {
    Plugin     string
    Dependency string
}
```
**Trigger**: Plugin declares dependency that doesn't exist in registry  
**Resolution**: Register missing plugin or remove invalid dependency declaration

---

## Interface Extensions

### Plugin Interface (Updated)

**Original**:
```go
type Plugin interface {
    Metadata() PluginMetadata
    Schema() PluginSchema
    Apply(step StepConfig) (Result, error)
    Verify(step StepConfig) (VerificationStatus, error)
}
```

**Updated** (backward compatible via optional interface):
```go
// Optional interface for plugins that need dependencies
type PluginInitializer interface {
    Init(registry *PluginRegistry) error
}
```

**Migration Strategy**:
- Old plugins without `Init()` → continue to work, cannot use dependencies
- New plugins implement `PluginInitializer` → receive registry during initialization
- Type assertion determines if plugin supports dependencies: `if initializer, ok := p.(PluginInitializer); ok`

---

## Data Flow Diagrams

### Registration Flow
```
1. Plugin created → Metadata() called
2. Registry.Register(plugin)
3. Add to plugins map
4. Extract dependencies from metadata
5. Build dependency graph (add nodes and edges)
```

### Validation Flow
```
1. Registry.ValidateDependencies() called
2. For each plugin:
   a. Check all dependencies exist in registry
   b. Check version constraints satisfied
3. DependencyGraph.DetectCycles()
4. If errors and policy=strict → abort
5. If errors and policy=graceful → disable affected plugins, log warnings
```

### Initialization Flow
```
1. DependencyGraph.TopologicalSort() → ordered plugin list
2. For each plugin in order:
   a. If implements PluginInitializer:
      - Call plugin.Init(registry)
      - Plugin can now access dependencies via registry.Get()
   b. Log initialization success/failure
```

### Dependency Access Flow
```
1. Plugin A calls registry.Get("plugin_b", policy)
2. Check if plugin_b exists → ErrPluginNotFound if not
3. Check if plugin_b declared in plugin_a metadata:
   a. If not declared and policy=strict → ErrUndeclaredDependency
   b. If not declared and policy=warn → log warning, continue
   c. If not declared and policy=off → continue
4. Check if plugin_b is stateful:
   a. If stateful → return/create per-dependent instance
   b. If stateless → return shared singleton
5. Return plugin_b instance
```

---

## Memory Characteristics

### Space Complexity
- **Registry**: O(P) where P = number of plugins
- **DependencyGraph**: O(P + D) where D = number of dependency edges
- **Stateful Instances**: O(P × D) worst case (if all plugins are stateful with max dependencies)

### Typical Memory Usage (100 plugins)
- Registry map: ~8KB (pointers)
- Dependency graph: ~16KB (adjacency lists)
- Metadata: ~50KB (strings, version info)
- **Total**: ~75KB

### Performance Targets
- **Registration**: O(1) per plugin
- **Validation**: O(P + D) one-time at startup
- **Lookup**: O(1) average case (hash map)
- **Initialization**: O(P + D) one-time (topological sort)

---

## Thread Safety Model

### Concurrent Operations
- **Reads**: Multiple goroutines can call `Get()` concurrently (RLock)
- **Writes**: Registration and initialization are sequential (Lock)
- **Mixed**: Lookups during initialization blocked by write lock

### Synchronization Strategy
```go
// Registration (exclusive)
func (r *PluginRegistry) Register(p Plugin) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    // ... modify registry state
}

// Lookup (shared)
func (r *PluginRegistry) Get(name string) (Plugin, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    // ... read-only access
}
```

---

## Backward Compatibility

### Legacy Plugin Support
- **Missing Dependencies field**: Treated as empty list `[]Dependency{}`
- **No Init() method**: Plugin functions normally, cannot access dependencies
- **No APIVersion**: Treated as compatible with all versions (warning logged)

### Deprecation Path
1. **v1.x** (current): Optional dependencies, warnings for missing metadata
2. **v2.0** (future): Required dependency metadata, errors for missing fields
3. **Migration tool**: Automated generation of metadata stubs for old plugins

---

## Schema Evolution

### Additive Changes (Minor Version)
- Add optional fields to `PluginMetadata` (e.g., `Tags []string`)
- Add new error types
- Add new policy modes

### Breaking Changes (Major Version)
- Change dependency constraint syntax
- Make metadata fields required
- Change plugin interface signature

All breaking changes require migration guide and tooling support per Constitution.

---

## References
- Plugin interface: `internal/plugin/interface.go`
- Existing registry (basic): `internal/plugin/registry.go`
- Feature spec: `spec.md`
- Research decisions: `research.md`
