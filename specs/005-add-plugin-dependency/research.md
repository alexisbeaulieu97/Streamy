# Research: Plugin Dependency Registry

**Feature**: 005-add-plugin-dependency  
**Date**: October 6, 2025  
**Status**: Complete

## Overview
This document consolidates research findings for implementing a plugin dependency registry system in Streamy's Go codebase. The registry enables composable plugin architecture where plugins can depend on and invoke other plugins.

---

## Decision 1: Dependency Graph Algorithm

**Decision**: Use Kahn's algorithm for topological sorting with explicit cycle detection

**Rationale**:
- **Simplicity**: Kahn's algorithm is straightforward to implement without external dependencies
- **Cycle Detection**: Naturally detects cycles - if sort completes with remaining nodes, a cycle exists
- **Performance**: O(V + E) complexity where V = plugins, E = dependencies
- **No External Deps**: Can be implemented in ~50 lines of Go, aligns with Constitution Principle I

**Alternatives Considered**:
1. **DFS-based topological sort**: More complex to implement correctly, harder to explain cycle detection
2. **External graph library** (`golang.org/x/tools`): Violates zero-dependency principle, overkill for simple DAG
3. **No ordering (runtime resolution)**: Would require complex lazy initialization, harder to debug

**Implementation Notes**:
```go
// Pseudocode for Kahn's algorithm
func (g *DependencyGraph) TopologicalSort() ([]string, error) {
    inDegree := make(map[string]int)
    queue := []string{}
    result := []string{}
    
    // Calculate in-degrees
    for node := range g.nodes {
        inDegree[node] = len(g.incoming[node])
        if inDegree[node] == 0 {
            queue = append(queue, node)
        }
    }
    
    // Process queue
    for len(queue) > 0 {
        current := queue[0]
        queue = queue[1:]
        result = append(result, current)
        
        for _, neighbor := range g.outgoing[current] {
            inDegree[neighbor]--
            if inDegree[neighbor] == 0 {
                queue = append(queue, neighbor)
            }
        }
    }
    
    // If not all nodes processed, cycle exists
    if len(result) != len(g.nodes) {
        return nil, ErrCircularDependency
    }
    
    return result, nil
}
```

---

## Decision 2: Version Constraint Syntax

**Decision**: Major version matching using `X.x` format (e.g., `1.x`, `2.x`)

**Rationale** (from clarification session):
- **Simplicity**: Easy to parse and validate with string comparison
- **Semantic Versioning Alignment**: Follows semver major version boundaries
- **Clear Compatibility**: `1.x` means "any 1.y.z version", preventing breaking changes
- **User-Friendly**: Intuitive syntax that developers understand immediately

**Alternatives Considered**:
1. **Full semver ranges** (`^1.0.0`, `~1.2.0`): More complex to parse, requires semver library
2. **Exact versions only** (`1.0.0`): Too restrictive, forces updates for patch/minor releases
3. **Minimum version** (`>=1.0.0`): Allows breaking changes, defeats purpose of constraints

**Implementation**:
```go
type VersionConstraint struct {
    MajorVersion int    // e.g., 1 for "1.x"
}

func ParseVersionConstraint(s string) (*VersionConstraint, error) {
    // Parse "1.x", "2.x", etc.
    parts := strings.Split(s, ".")
    if len(parts) != 2 || parts[1] != "x" {
        return nil, fmt.Errorf("invalid version constraint: %s (expected format: N.x)", s)
    }
    major, err := strconv.Atoi(parts[0])
    if err != nil {
        return nil, fmt.Errorf("invalid major version: %s", parts[0])
    }
    return &VersionConstraint{MajorVersion: major}, nil
}

func (vc *VersionConstraint) Satisfies(version string) bool {
    // Parse "X.Y.Z" version
    parts := strings.Split(version, ".")
    if len(parts) < 1 {
        return false
    }
    major, err := strconv.Atoi(parts[0])
    if err != nil {
        return false
    }
    return major == vc.MajorVersion
}
```

---

## Decision 3: Plugin State Isolation Strategy

**Decision**: Shared singleton instances by default, with optional per-dependent isolation via metadata flag

**Rationale** (from clarification session):
- **Performance**: Single instance per plugin minimizes memory and initialization overhead
- **Stateless-by-Contract**: Most Streamy plugins are stateless (Apply/Verify are pure functions)
- **Escape Hatch**: Plugins that need state can declare `Stateful: true` in metadata
- **Go Idioms**: Aligns with Go's "share memory by communicating" principle

**Alternatives Considered**:
1. **Always isolate**: Wasteful for stateless plugins, 10-20x memory overhead
2. **Always share**: Unsafe if plugin maintains mutable state
3. **Auto-detect statefulness**: Complex, error-prone, hard to debug

**Implementation**:
```go
type PluginMetadata struct {
    Name         string
    Version      string
    Dependencies []Dependency
    Stateful     bool   // NEW: If true, create per-dependent instances
}

type PluginRegistry struct {
    plugins          map[string]Plugin         // Singleton instances
    statefulInstances map[string]map[string]Plugin // [pluginName][dependentName]Plugin
}

func (r *PluginRegistry) GetForDependent(pluginName, dependentName string) (Plugin, error) {
    plugin, exists := r.plugins[pluginName]
    if !exists {
        return nil, ErrPluginNotFound{Name: pluginName}
    }
    
    meta := plugin.Metadata()
    if !meta.Stateful {
        return plugin, nil  // Shared singleton
    }
    
    // Create or retrieve per-dependent instance
    if r.statefulInstances[pluginName] == nil {
        r.statefulInstances[pluginName] = make(map[string]Plugin)
    }
    if instance, ok := r.statefulInstances[pluginName][dependentName]; ok {
        return instance, nil
    }
    
    // Clone plugin for this dependent (requires plugin factory pattern)
    newInstance := r.createPluginInstance(pluginName)
    r.statefulInstances[pluginName][dependentName] = newInstance
    return newInstance, nil
}
```

---

## Decision 4: Failure Policy Configuration

**Decision**: Environment-aware defaults via `settings.dependency_policy` configuration

**Rationale** (from clarification session):
- **Context-Appropriate**: CI needs strict validation, developers need flexibility
- **Deterministic in CI**: Automation pipelines fail fast on dependency issues
- **Developer-Friendly**: Local development allows partial execution with warnings
- **Explicit Override**: Users can opt into stricter or more lenient modes

**Modes**:
- **Strict**: Abort execution on any dependency validation failure (missing, circular, version conflict)
- **Graceful**: Skip affected plugins, continue with valid ones, log warnings

**Defaults**:
- CLI/TUI (interactive): `graceful`
- CI/Automation (detected via CI env vars): `strict`

**Implementation**:
```go
type DependencyPolicy string

const (
    PolicyStrict   DependencyPolicy = "strict"
    PolicyGraceful DependencyPolicy = "graceful"
)

type Config struct {
    DependencyPolicy DependencyPolicy `yaml:"dependency_policy"`
}

func DetectDefaultPolicy() DependencyPolicy {
    // Check common CI environment variables
    ciEnvVars := []string{"CI", "CONTINUOUS_INTEGRATION", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_HOME"}
    for _, envVar := range ciEnvVars {
        if os.Getenv(envVar) != "" {
            return PolicyStrict
        }
    }
    return PolicyGraceful
}

func (r *PluginRegistry) ValidateDependencies(policy DependencyPolicy) error {
    errors := r.collectDependencyErrors()
    
    if len(errors) == 0 {
        return nil
    }
    
    if policy == PolicyStrict {
        return fmt.Errorf("dependency validation failed: %v", errors)
    }
    
    // PolicyGraceful: log warnings, disable affected plugins
    for _, err := range errors {
        r.logger.Warn().Err(err).Msg("Dependency issue detected, disabling affected plugins")
        r.disableAffectedPlugins(err)
    }
    
    return nil
}
```

---

## Decision 5: Runtime Dependency Access Enforcement

**Decision**: Environment-aware enforcement with typed error returns

**Rationale** (from clarification session):
- **Core Invariant**: Plugins may only access declared dependencies
- **Strict in Production**: CI/automation enforces contract via errors
- **Lenient in Development**: Warnings guide developers without blocking
- **Optional Off**: Internal testing can bypass for debugging

**Enforcement Levels**:
- **Strict** (CI/automation): `Get()` returns error for undeclared dependency
- **Warn** (CLI/interactive/debug): `Get()` logs warning, allows access
- **Off** (internal testing): No enforcement, unrestricted access

**Implementation**:
```go
type AccessPolicy string

const (
    AccessStrict AccessPolicy = "strict"
    AccessWarn   AccessPolicy = "warn"
    AccessOff    AccessPolicy = "off"
)

func (r *PluginRegistry) Get(callerName, dependencyName string, policy AccessPolicy) (Plugin, error) {
    plugin, exists := r.plugins[dependencyName]
    if !exists {
        return nil, ErrPluginNotFound{Name: dependencyName}
    }
    
    // Check if caller declared this dependency
    caller := r.plugins[callerName]
    declared := r.isDependencyDeclared(caller, dependencyName)
    
    if !declared {
        switch policy {
        case AccessStrict:
            return nil, ErrUndeclaredDependency{
                Caller:     callerName,
                Dependency: dependencyName,
                Hint:       "Add '" + dependencyName + "' to Dependencies in Metadata()",
            }
        case AccessWarn:
            r.logger.Warn().
                Str("caller", callerName).
                Str("dependency", dependencyName).
                Msg("Undeclared dependency accessed - add to Metadata().Dependencies")
        case AccessOff:
            // No enforcement
        }
    }
    
    return plugin, nil
}
```

---

## Decision 6: Backward Compatibility Strategy

**Decision**: Graceful migration with deprecation warnings

**Rationale** (from clarification session):
- **Non-Breaking**: Existing plugins continue to work without modification
- **Clear Migration Path**: Warnings guide plugin authors to update metadata
- **Future Enforcement**: Next major version can require metadata
- **Ecosystem Health**: Gradual adoption prevents fragmentation

**Behavior**:
- Plugins without `Dependencies` field → treated as zero dependencies
- Plugins without `Init()` method → no registry access, but still functional
- Warning logged once per plugin at initialization

**Implementation**:
```go
func (r *PluginRegistry) Register(p Plugin) error {
    meta := p.Metadata()
    
    // Backward compatibility: missing dependencies = empty list
    if meta.Dependencies == nil {
        r.logger.Warn().
            Str("plugin", meta.Name).
            Msg("Plugin missing Dependencies field in metadata (will be required in v2.0)")
    }
    
    // Backward compatibility: missing Init() is OK
    if initializer, ok := p.(PluginInitializer); ok {
        if err := initializer.Init(r); err != nil {
            return fmt.Errorf("plugin %s initialization failed: %w", meta.Name, err)
        }
    } else {
        r.logger.Debug().
            Str("plugin", meta.Name).
            Msg("Plugin does not implement Init() - cannot use dependencies")
    }
    
    r.plugins[meta.Name] = p
    return nil
}

// Optional interface for plugins that need dependencies
type PluginInitializer interface {
    Init(registry *PluginRegistry) error
}
```

---

## Decision 7: Error Types and Messages

**Decision**: Typed errors with actionable remediation hints

**Rationale**:
- **Constitution Principle II**: Error messages must include fix suggestions
- **Developer Experience**: Clear guidance reduces support burden
- **Programmatic Handling**: Typed errors allow conditional logic based on error type

**Error Types**:
```go
// Typed errors for structured handling
type ErrPluginNotFound struct {
    Name string
}

func (e ErrPluginNotFound) Error() string {
    return fmt.Sprintf("plugin '%s' not found in registry", e.Name)
}

type ErrCircularDependency struct {
    Cycle []string
}

func (e ErrCircularDependency) Error() string {
    return fmt.Sprintf("circular dependency detected: %s", strings.Join(e.Cycle, " -> "))
}

type ErrVersionConflict struct {
    Plugin       string
    RequiredBy   map[string]string // dependent -> constraint
    ActualVersion string
}

func (e ErrVersionConflict) Error() string {
    conflicts := []string{}
    for dependent, constraint := range e.RequiredBy {
        conflicts = append(conflicts, fmt.Sprintf("%s requires %s", dependent, constraint))
    }
    return fmt.Sprintf(
        "version conflict for plugin '%s' (version %s):\n  %s\nSuggestion: Update plugin versions to satisfy all constraints",
        e.Plugin,
        e.ActualVersion,
        strings.Join(conflicts, "\n  "),
    )
}

type ErrUndeclaredDependency struct {
    Caller     string
    Dependency string
    Hint       string
}

func (e ErrUndeclaredDependency) Error() string {
    return fmt.Sprintf(
        "plugin '%s' attempted to access undeclared dependency '%s'\n%s",
        e.Caller,
        e.Dependency,
        e.Hint,
    )
}
```

---

## Decision 8: Thread Safety

**Decision**: Mutex-protected registry with read-write lock

**Rationale**:
- **Concurrent Lookups**: Multiple plugins may look up dependencies simultaneously
- **Initialization Order**: Registration happens sequentially at startup, lookups during execution
- **Go Best Practices**: `sync.RWMutex` allows concurrent reads, exclusive writes

**Implementation**:
```go
type PluginRegistry struct {
    mu                sync.RWMutex
    plugins           map[string]Plugin
    dependencyGraph   *DependencyGraph
    statefulInstances map[string]map[string]Plugin
}

func (r *PluginRegistry) Register(p Plugin) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    // Registration logic
    // ...
}

func (r *PluginRegistry) Get(name string) (Plugin, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    plugin, exists := r.plugins[name]
    if !exists {
        return nil, ErrPluginNotFound{Name: name}
    }
    return plugin, nil
}
```

---

## Testing Strategy

### Unit Tests
1. **Registry Registration**: Test plugin registration with/without dependencies
2. **Cycle Detection**: Verify circular dependency detection with various graph structures
3. **Version Constraints**: Test constraint parsing and satisfaction checks
4. **Topological Sort**: Validate correct initialization order
5. **Error Scenarios**: Missing dependencies, version conflicts, undeclared access

### Integration Tests
1. **Plugin Composition**: `shell_profile` depends on `line_in_file`
2. **Transitive Dependencies**: A → B → C dependency chain
3. **Policy Modes**: Strict vs graceful failure handling
4. **Backward Compatibility**: Old plugins work alongside new dependency-aware plugins

### Performance Tests
1. **Dependency Resolution**: Benchmark topological sort for 10, 50, 100 plugins
2. **Lookup Performance**: Measure registry lookup overhead (target: <1μs)
3. **Memory Overhead**: Verify O(n) memory usage

---

## Open Questions
None - all critical decisions resolved during clarification session.

---

## References
- [Kahn's Algorithm](https://en.wikipedia.org/wiki/Topological_sorting#Kahn's_algorithm)
- [Semantic Versioning](https://semver.org/)
- [Go sync.RWMutex](https://pkg.go.dev/sync#RWMutex)
- Streamy Constitution: `.specify/memory/constitution.md`
- Feature Specification: `spec.md`
