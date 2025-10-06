# Feature Specification: Plugin Dependency Registry for Composable Plugins

**Feature Branch**: `005-add-plugin-dependency`  
**Created**: October 6, 2025  
**Status**: Draft  
**Input**: User description: "Add Plugin Dependency Registry for Composable Plugins"

## Execution Flow (main)
```
1. Parse user description from Input
   → Feature description provided: "Add Plugin Dependency Registry for Composable Plugins"
2. Extract key concepts from description
   → Actors: Plugin developers, Streamy core runtime
   → Actions: Register plugins, declare dependencies, resolve dependencies, validate compatibility
   → Data: Plugin metadata, dependency graphs, version constraints
   → Constraints: No circular dependencies, version compatibility, safe resolution
3. For each unclear aspect:
   → All aspects sufficiently detailed in user input
4. Fill User Scenarios & Testing section
   → User flows identified for plugin composition
5. Generate Functional Requirements
   → All requirements testable and aligned with composability goals
6. Identify Key Entities
   → PluginRegistry, PluginMetadata, DependencyGraph
7. Run Review Checklist
   → No implementation details in requirements
   → All requirements testable
8. Return: SUCCESS (spec ready for planning)
```

---

## ⚡ Quick Guidelines
- ✅ Focus on WHAT users need and WHY
- ❌ Avoid HOW to implement (no tech stack, APIs, code structure)
- 👥 Written for business stakeholders, not developers

### Section Requirements
- **Mandatory sections**: Must be completed for every feature
- **Optional sections**: Include only when relevant to the feature
- When a section doesn't apply, remove it entirely (don't leave as "N/A")

### For AI Generation
When creating this spec from a user prompt:
1. **Mark all ambiguities**: Use [NEEDS CLARIFICATION: specific question] for any assumption you'd need to make
2. **Don't guess**: If the prompt doesn't specify something (e.g., "login system" without auth method), mark it
3. **Think like a tester**: Every vague requirement should fail the "testable and unambiguous" checklist item
4. **Common underspecified areas**:
   - User types and permissions
   - Data retention/deletion policies  
   - Performance targets and scale
   - Error handling behaviors
   - Integration requirements
   - Security/compliance needs
5. **Constitution alignment**: Consider onboarding impact, schema clarity, safety defaults
   - Will this require external dependencies? (Principle I)
   - Is the configuration intuitive? (Principle II)
   - Are destructive operations identified and guarded? (Principle IV)
   - Are performance expectations clear? (Principle V)

---

## Clarifications

### Session 2025-10-06
- Q: Version Constraint Syntax - Which version constraint syntax should the system support? → A: Major version matching (`1.x`) - allows minor/patch updates only
- Q: Plugin Instance State Isolation - Should dependency plugin state be isolated per dependent or shared? → A: Configurable via metadata - shared singleton by default (stateless-by-contract), with optional per-dependent isolation for declared stateful plugins
- Q: Dependency Validation Failure Behavior - What happens when dependency validation fails? → A: Configurable via settings.dependency_policy - defaults to graceful (skip affected plugins) for CLI/TUI, strict (abort execution) for automation/CI
- Q: Runtime Undeclared Dependency Access - How should undeclared dependency access be enforced? → A: Environment-aware enforcement - strict (fail on access) for CI/automation, warn (log and continue) for CLI/interactive/debug, optional off mode for internal testing
- Q: Backward Compatibility for Existing Plugins - Should existing plugins without dependency metadata continue to work? → A: Graceful migration - existing plugins work without modification, missing metadata treated as no dependencies with warning logged, strict requirement deferred to next major release

---

## User Scenarios & Testing *(mandatory)*

### Primary User Story
A plugin developer wants to create a `shell_profile` plugin that manages shell configuration files. Instead of reimplementing line-in-file logic, they want to reuse the existing `line_in_file` plugin as a dependency. The system must allow the plugin to discover, declare, and safely use the `line_in_file` plugin's functionality while preventing circular dependencies and ensuring version compatibility.

### Acceptance Scenarios

1. **Given** a plugin declares a dependency on another registered plugin, **When** the plugin is initialized, **Then** the system resolves the dependency and provides access to the dependent plugin's functionality.

2. **Given** a plugin declares a dependency on a non-existent plugin, **When** the system initializes, **Then** the system reports a missing dependency error and prevents the plugin from loading.

3. **Given** two plugins declare circular dependencies on each other, **When** the system initializes, **Then** the system detects the cycle and reports an error preventing both plugins from loading.

4. **Given** a plugin declares a dependency with version constraints, **When** the system resolves dependencies, **Then** the system verifies version compatibility and loads only compatible versions.

5. **Given** a plugin successfully resolves its dependencies, **When** the plugin executes its apply or verify operation, **Then** the plugin can invoke the dependent plugin's operations and receive correct results.

6. **Given** multiple plugins depend on the same base plugin, **When** all plugins are loaded, **Then** the system shares a single instance of the base plugin across all dependents.

### Edge Cases

- What happens when a plugin's dependency declares its own dependencies (transitive dependencies)?
  → System must resolve the entire dependency chain and validate all transitive dependencies.

- What happens when two plugins require incompatible versions of the same dependency?
  → System must detect the version conflict and report an error identifying which plugins have conflicting requirements.

- How does the system handle plugins that are registered after dependent plugins attempt to resolve them?
  → System must validate all dependencies are registered before allowing any plugin to complete initialization.

- What happens when a plugin attempts to use a dependency at runtime that wasn't declared in its metadata?
  → System should prevent undeclared runtime dependencies or log warnings about undeclared usage.

- How does the system handle dependency resolution order for complex graphs?
  → System must use topological sorting to ensure dependencies are initialized before dependents.


## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a centralized registry where all plugins can be registered with unique identifiers.

- **FR-002**: System MUST allow plugins to declare dependencies on other plugins through metadata. For backward compatibility, plugins without dependency metadata MUST continue to function (treated as having no dependencies), with a warning logged during initialization. Strict metadata requirements may be enforced in a future major release.

- **FR-003**: System MUST validate that all declared dependencies exist in the registry before plugin initialization completes. Validation failure behavior MUST be configurable via `RegistryConfig.DependencyPolicy` with environment-aware defaults: graceful mode (skip affected plugins, continue with valid ones) for interactive CLI/TUI contexts, and strict mode (abort entire execution) for automation/CI contexts.

- **FR-004**: System MUST detect circular dependencies between plugins and prevent their registration. Behavior follows the configurable `RegistryConfig.DependencyPolicy`: graceful mode skips circular dependency groups while allowing other plugins to load; strict mode aborts execution entirely.

- **FR-005**: System MUST support major version matching constraints (e.g., `1.x`, `2.x`) for plugin dependencies, allowing minor and patch updates within the same major version while preventing breaking changes across major version boundaries.

- **FR-006**: System MUST resolve transitive dependencies (dependencies of dependencies) automatically.

- **FR-007**: System MUST provide plugins with safe access to their declared dependencies during apply and verify operations.

- **FR-008**: System MUST enforce that plugins only access dependencies explicitly declared in their metadata. Enforcement level varies by environment: strict mode (return error on undeclared access) for CI/automation contexts, warn mode (log warning but allow access) for CLI/interactive/debug contexts, with optional off mode for internal testing. The core invariant is that plugins may only interact with declared dependencies.

- **FR-009**: System MUST detect version conflicts when multiple plugins depend on incompatible versions of the same plugin. Conflict resolution follows `RegistryConfig.DependencyPolicy`: graceful mode disables conflicting dependents while preserving the dependency; strict mode aborts execution with detailed conflict report.

- **FR-010**: System MUST initialize plugins in dependency order, ensuring dependencies are ready before dependents.

- **FR-011**: System MUST share single instances of plugins across multiple dependents by default (assuming stateless-by-contract behavior). Plugins that maintain internal state MUST be able to declare a stateful isolation preference in their metadata, triggering per-dependent instance creation to prevent unsafe state sharing.

- **FR-012**: System MUST report clear error messages identifying missing, circular, or incompatible dependencies. Error messages must include context, cause, and remediation suggestion.

- **FR-013**: System MUST allow plugins to reuse other plugins' apply operations to compose higher-level functionality.

- **FR-014**: System MUST allow plugins to reuse other plugins' verify operations to ensure consistent validation.

- **FR-015**: System MUST propagate errors from dependent plugins back to the calling plugin with proper context.

- **FR-016**: System MUST maintain backward compatibility with existing plugins that lack dependency metadata, treating them as having zero dependencies and logging a deprecation warning. This allows graceful migration without breaking existing plugin installations.

### Key Entities

- **PluginRegistry**: Central repository that maintains all registered plugins, provides lookup by unique identifier, manages initialization order, and validates dependency graphs. Contains version information for each plugin and tracks registration state.

- **PluginMetadata**: Descriptive information about a plugin including its unique name/identifier, semantic version, list of declared dependencies (with optional version constraints), human-readable description, and Stateful field. Plugins that maintain internal state can set Stateful to true to receive per-dependent instances instead of shared singletons. Used for validation and dependency resolution.

- **DependencyGraph**: Representation of relationships between plugins, used to detect circular dependencies, determine initialization order through topological sorting, and identify transitive dependencies. Validated during plugin registration.

- **VersionConstraint**: Specification of acceptable version ranges for dependencies using major version matching syntax (e.g., `1.x`, `2.x`). Allows minor and patch version updates within the same major version while preventing breaking changes across major versions. Used to validate compatibility between dependent and dependency plugins.


---

## Review & Acceptance Checklist
*GATE: Automated checks run during main() execution*

### Content Quality
- [x] No implementation details (languages, frameworks, APIs)
- [x] Focused on user value and business needs
- [x] Written for non-technical stakeholders
- [x] All mandatory sections completed

### Requirement Completeness
- [x] No [NEEDS CLARIFICATION] markers remain
- [x] Requirements are testable and unambiguous  
- [x] Success criteria are measurable
- [x] Scope is clearly bounded
- [x] Dependencies and assumptions identified

---

## Execution Status
*Updated by main() during processing*

- [x] User description parsed
- [x] Key concepts extracted
- [x] Ambiguities marked
- [x] User scenarios defined
- [x] Requirements generated
- [x] Entities identified
- [x] Review checklist passed

---

## Constitution Alignment Notes

**Principle I (Minimal External Dependencies)**: The dependency registry is an internal Streamy capability that allows plugins to depend on other plugins within the ecosystem, not external systems. This maintains the zero-dependency philosophy while enabling composability.

**Principle II (Configuration Clarity)**: Plugin dependency declarations must be explicit and visible in plugin metadata, making the dependency graph transparent and auditable. Users can understand which plugins depend on others.

**Principle IV (Safety by Default)**: The system validates dependency graphs before execution, preventing circular dependencies and missing dependencies that could lead to runtime failures. This ensures safe composition.

**Principle V (Performance Awareness)**: Shared plugin instances across dependents avoid duplication and reduce initialization overhead. Dependency resolution happens once at startup, not during execution.

