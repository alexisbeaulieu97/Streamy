# Tasks: Plugin Dependency Registry for Composable Plugins

**Input**: Design documents from `/home/alexis/Projects/Streamy/specs/005-add-plugin-dependency/`
**Prerequisites**: plan.md ✅, research.md ✅, data-model.md ✅, contracts/ ✅, quickstart.md ✅

## Execution Flow (main)
```
1. Load plan.md from feature directory ✅
   → Extracted: Go 1.21+, zero dependencies, single project structure
2. Load optional design documents ✅
   → data-model.md: Extracted entities (PluginRegistry, PluginMetadata, Dependency, etc.)
   → contracts/: registry-api.md → contract test tasks
   → research.md: Extracted decisions (Kahn's algorithm, version matching, policies)
   → quickstart.md: Extracted test scenarios (shell_profile example)
3. Generate tasks by category ✅
   → Setup: Error types, data structures
   → Tests: Contract tests, integration tests
   → Core: Registry operations, dependency graph, version constraints
   → Integration: Policy system, plugin updates, main wiring
   → Polish: Documentation, performance validation
4. Apply task rules ✅
   → Different files = marked [P] for parallel
   → Same file = sequential (no [P])
   → Tests before implementation (TDD)
5. Number tasks sequentially (T001-T035) ✅
6. Generate dependency graph ✅
7. Create parallel execution examples ✅
8. Validate task completeness ✅
   → All contracts have tests ✅
   → All entities have implementation tasks ✅
   → Constitution principles covered ✅
9. Return: SUCCESS (35 tasks ready for execution)
```

## Format: `[ID] [P?] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- Include exact file paths in descriptions

## Path Conventions
**Single Project Structure** (from plan.md):
- Core: `internal/plugin/` for registry implementation
- Plugins: `internal/plugins/*/` for plugin updates
- Main: `cmd/streamy/` for application entry point
- Tests: `tests/` for integration tests
- Docs: `docs/` for documentation updates

---

## Phase 3.1: Setup & Foundation (T001-T005)

- [ ] **T001** [P] Create error types in `internal/plugin/errors.go`
  - Define `ErrPluginNotFound`, `ErrCircularDependency`, `ErrVersionConflict`, `ErrUndeclaredDependency`, `ErrMissingDependency`
  - Implement `Error()` methods with actionable messages per Constitution Principle II
  - Include remediation hints in error messages

- [ ] **T002** [P] Create metadata types in `internal/plugin/metadata.go`
  - Define `PluginMetadata` struct with Name, Version, APIVersion, Dependencies, Stateful, Description
  - Define `Dependency` struct with Name and VersionConstraint
  - Add validation methods for metadata fields

- [ ] **T003** [P] Create version constraint in `internal/plugin/version.go`
  - Define `VersionConstraint` struct with MajorVersion field
  - Implement `ParseVersionConstraint(s string)` for "N.x" format
  - Implement `Satisfies(version string) bool` method
  - Add unit tests in `internal/plugin/version_test.go`

- [ ] **T004** [P] Create configuration types in `internal/plugin/config.go`
  - Define `RegistryConfig` with DependencyPolicy and AccessPolicy
  - Define policy constants: PolicyStrict, PolicyGraceful, AccessStrict, AccessWarn, AccessOff
  - Implement `DefaultConfig()` with environment detection (CI vs interactive)
  - Test environment detection logic with CI env vars: CI, CONTINUOUS_INTEGRATION, GITHUB_ACTIONS, GITLAB_CI, JENKINS_HOME
  - Test that each env var triggers strict policy mode
  - Test that absence of CI vars triggers graceful policy mode

- [ ] **T005** [P] Create dependency graph structure in `internal/plugin/dependency_graph.go`
  - Define `DependencyGraph` struct with nodes, incoming, outgoing maps
  - Implement `AddNode(name string)` method
  - Implement `AddEdge(dependent, dependency string)` method
  - Skeleton for cycle detection and topological sort (implementation in T011-T012)

---

## Phase 3.2: Tests First (TDD) ⚠️ MUST COMPLETE BEFORE 3.3
**CRITICAL: These tests MUST be written and MUST FAIL before ANY implementation**

- [ ] **T006** [P] Contract test: Register and Get in `internal/plugin/registry_test.go`
  - Test registry creation with `NewPluginRegistry()`
  - Test plugin registration with `Register()`
  - Test plugin retrieval with `Get()`
  - Verify error when getting non-existent plugin
  - **MUST FAIL** initially (no implementation yet)

- [ ] **T007** [P] Contract test: Missing dependency detection in `internal/plugin/registry_test.go`
  - Test plugin with dependency on non-existent plugin
  - Call `ValidateDependencies()` expecting error
  - Verify `ErrMissingDependency` type
  - **MUST FAIL** initially

- [ ] **T008** [P] Contract test: Circular dependency detection in `internal/plugin/registry_test.go`
  - Create 3 plugins in circular dependency (A→B→C→A)
  - Register all three plugins
  - Call `ValidateDependencies()` expecting error
  - Verify `ErrCircularDependency` with cycle list
  - **MUST FAIL** initially

- [ ] **T009** [P] Contract test: Initialization order in `internal/plugin/registry_test.go`
  - Create plugin A with no dependencies, plugin B depending on A
  - Track initialization order
  - Call `InitializePlugins()`
  - Verify A initialized before B
  - **MUST FAIL** initially

- [ ] **T010** [P] Contract test: Version constraint validation in `internal/plugin/version_test.go`
  - Test `ParseVersionConstraint("1.x")` success
  - Test `ParseVersionConstraint("invalid")` error
  - Test `Satisfies("1.2.3")` returns true for "1.x" constraint
  - Test `Satisfies("2.0.0")` returns false for "1.x" constraint
  - **MUST FAIL** initially

---

## Phase 3.3: Dependency Graph Implementation (T011-T013)

- [ ] **T011** Implement cycle detection in `internal/plugin/dependency_graph.go`
  - Implement `DetectCycles() ([]string, error)` using DFS with recursion stack
  - Return ordered list of nodes in cycle if found
  - Add comprehensive test cases in `internal/plugin/dependency_graph_test.go`
  - Verify T008 contract test now passes

- [ ] **T012** Implement topological sort in `internal/plugin/dependency_graph.go`
  - Implement `TopologicalSort() ([]string, error)` using Kahn's algorithm
  - Calculate in-degrees, process queue, detect remaining nodes
  - Return ordered list of plugins (dependencies before dependents)
  - Add test cases for various graph structures
  - Verify T009 contract test now passes

- [ ] **T013** [P] Add graph utility methods in `internal/plugin/dependency_graph.go`
  - Implement `GetDependencies(node string) []string`
  - Implement `GetDependents(node string) []string`
  - Implement `HasNode(node string) bool`
  - Add unit tests for utilities

---

## Phase 3.4: Core Registry Implementation (T014-T021)

- [ ] **T014** Create registry structure in `internal/plugin/registry.go`
  - Define `PluginRegistry` struct with mu (RWMutex), plugins map, dependencyGraph, statefulInstances, logger, config
  - Implement `NewPluginRegistry(config *RegistryConfig, logger *logger.Logger) *PluginRegistry`
  - Initialize maps and dependency graph
  - Verify T006 contract test progresses

- [ ] **T015** Implement Register method in `internal/plugin/registry.go`
  - Implement `Register(p Plugin) error` with mutex lock
  - Validate metadata (non-empty name, valid version)
  - Check for duplicate plugin names
  - Add plugin to registry map
  - Add dependencies to dependency graph
  - Log backward compatibility warnings for missing metadata
  - Verify T006, T007 contract tests progress

- [ ] **T016** Implement ValidateDependencies in `internal/plugin/registry.go`
  - Implement `ValidateDependencies() error`
  - Check all declared dependencies exist in registry
  - Check version constraints are satisfied
  - Call `dependencyGraph.DetectCycles()`
  - Apply policy (strict vs graceful) to handle errors
  - Disable affected plugins in graceful mode
  - Log clear error messages with remediation hints
  - Verify T007, T008 contract tests now pass

- [ ] **T017** Implement InitializePlugins in `internal/plugin/registry.go`
  - Implement `InitializePlugins() error`
  - Call `dependencyGraph.TopologicalSort()` to get ordered list
  - For each plugin in order, check if implements `PluginInitializer`
  - Call `plugin.Init(registry)` if supported
  - Log initialization progress with timing
  - Propagate initialization errors
  - Verify T009 contract test now passes

- [ ] **T018** Implement Get method in `internal/plugin/registry.go`
  - Implement `Get(name string) (Plugin, error)` with RLock
  - Look up plugin in map
  - Return `ErrPluginNotFound` if not exists
  - Thread-safe for concurrent access
  - Verify T006 contract test fully passes

- [ ] **T019** Implement GetForDependent in `internal/plugin/registry.go`
  - Implement `GetForDependent(dependentName, pluginName string) (Plugin, error)`
  - Check if dependency declared in dependent's metadata
  - Apply access policy (strict, warn, off)
  - Return error or log warning for undeclared access
  - Check if plugin is stateful
  - Return singleton or create per-dependent instance
  - Add unit tests for access policy enforcement

- [ ] **T020** [P] Implement List method in `internal/plugin/registry.go`
  - Implement `List() []string` with RLock
  - Return sorted list of plugin names
  - Add unit test

- [ ] **T021** [P] Add registry helper methods in `internal/plugin/registry.go`
  - Implement `isDependencyDeclared(caller Plugin, depName string) bool`
  - Implement `disableAffectedPlugins(err error)`
  - Implement `createPluginInstance(name string) Plugin` for stateful plugins
  - Add internal helper tests

---

## Phase 3.5: Plugin Interface Updates (T022-T024)

- [ ] **T022** Add PluginInitializer interface in `internal/plugin/interface.go`
  - Define optional `PluginInitializer` interface with `Init(registry *PluginRegistry) error`
  - Document backward compatibility (type assertion checks)
  - Update interface documentation with examples
  - Add contract test for optional interface pattern

- [ ] **T023** Update Plugin interface documentation in `internal/plugin/interface.go`
  - Document `Metadata()` return expectations including Dependencies
  - Add examples of declaring dependencies
  - Document stateful vs stateless plugins
  - Include migration guide comments for existing plugins

- [ ] **T024** Create mock plugin for testing in `internal/plugin/mock_plugin_test.go`
  - Implement `MockPlugin` with configurable metadata
  - Support optional Init() implementation
  - Track method calls for verification
  - Add factory functions for common test scenarios

---

## Phase 3.6: Integration Tests (T025-T028)

- [ ] **T025** [P] Integration test: Shell profile composition in `tests/integration_plugin_dependency_test.go`
  - Implement test from quickstart.md scenario
  - Create mock `line_in_file` and `shell_profile` plugins
  - Register both plugins with dependency declaration
  - Validate dependencies and initialize
  - Call `shell_profile.Apply()` and verify delegation
  - Assert `line_in_file` receives correct configuration

- [ ] **T026** [P] Integration test: Transitive dependencies in `tests/integration_plugin_dependency_test.go`
  - Create 3-plugin chain: A → B → C
  - Register all plugins
  - Validate and initialize
  - Verify initialization order (C, B, A)
  - Test that A can transitively access C through B

- [ ] **T027** [P] Integration test: Policy modes in `tests/integration_plugin_dependency_test.go`
  - Test strict mode with missing dependency (should abort)
  - Test graceful mode with missing dependency (should skip affected, continue)
  - Test strict mode with circular dependency (should abort)
  - Test graceful mode with undeclared access (should warn)
  - Test strict mode with undeclared access (should error)

- [ ] **T028** [P] Integration test: Backward compatibility in `tests/integration_plugin_dependency_test.go`
  - Create legacy plugin without Dependencies field
  - Create legacy plugin without Init() method
  - Register alongside new dependency-aware plugins
  - Verify legacy plugins work normally
  - Verify warnings are logged
  - Verify legacy plugins cannot access dependencies

---

## Phase 3.7: Main Integration (T029-T030)

- [ ] **T029** Update main.go to create registry in `cmd/streamy/main.go`
  - Import plugin registry package
  - Create `RegistryConfig` with `DefaultConfig()`
  - Create `PluginRegistry` with config and logger
  - Store registry reference for plugin registration
  - Add error handling with clear messages

- [ ] **T030** Update plugin registration in `cmd/streamy/plugins_import.go`
  - Modify `RegisterPlugins()` to accept registry parameter
  - Call `registry.Register()` for each plugin
  - Call `registry.ValidateDependencies()` after all registrations
  - Call `registry.InitializePlugins()` to inject dependencies
  - Log summary of loaded plugins
  - Add startup timing logs for performance validation

---

## Phase 3.8: Plugin Migration Examples (T031)

- [ ] **T031** [P] Add dependency declarations to example plugin in `internal/plugins/lineinfile/plugin.go`
  - Update `Metadata()` to include empty Dependencies list
  - Add APIVersion field
  - Add Description field
  - Demonstrate backward-compatible metadata update
  - Document pattern in code comments

---

## Phase 3.9: Performance & Polish (T032-T035)

- [ ] **T032** [P] Add performance tests in `internal/plugin/registry_perf_test.go`
  - Benchmark dependency resolution for 10, 50, 100 plugins
  - Benchmark registry lookup (Get) operations
  - Verify <50ms validation for 20-plugin graph
  - Verify <1μs lookup time
  - Test memory usage (O(n) verification)

- [ ] **T033** [P] Update plugin documentation in `docs/plugins.md`
  - Add "Plugin Dependencies" section
  - Document how to declare dependencies in metadata
  - Document Init() method pattern
  - Add examples from quickstart.md
  - Document version constraint syntax
  - Add troubleshooting section for common errors
  - Include migration guide for existing plugins

- [ ] **T034** [P] Update architecture documentation in `docs/architecture.md`
  - Add "Plugin Registry" section
  - Document dependency resolution flow
  - Include dependency graph diagram (text-based)
  - Document policy system (strict vs graceful)
  - Document thread safety guarantees
  - Add performance characteristics

- [ ] **T035** [P] Add registry JSON schema in `docs/schema.md`
  - Document PluginMetadata JSON schema
  - Document Dependency structure
  - Document VersionConstraint format
  - Include schema validation examples
  - Add error message format documentation

---

## Dependencies

### Critical Path
```
T001-T005 (Setup) → T006-T010 (Tests) → T011-T013 (Graph) → T014-T021 (Registry) → T029-T030 (Integration)
```

### Detailed Dependencies
- **T006-T010** (Tests) must complete before implementation tasks
- **T011-T012** (Graph algorithms) block **T016** (ValidateDependencies) and **T017** (InitializePlugins)
- **T014** (Registry structure) blocks **T015-T021** (Registry methods)
- **T015** (Register) blocks **T016** (ValidateDependencies)
- **T016** (ValidateDependencies) blocks **T017** (InitializePlugins)
- **T022-T024** (Interface updates) can proceed in parallel with registry implementation
- **T025-T028** (Integration tests) require **T021** (Registry complete)
- **T029-T030** (Main integration) require **T021** (Registry complete)
- **T032-T035** (Polish) can proceed once core implementation stable

### Parallelization Opportunities

**Wave 1: Foundation** (can run simultaneously)
```
T001 [P] - errors.go
T002 [P] - metadata.go  
T003 [P] - version.go
T004 [P] - config.go
T005 [P] - dependency_graph.go (structure only)
```

**Wave 2: Contract Tests** (can run simultaneously)
```
T006 [P] - registry_test.go (Register/Get tests)
T007 [P] - registry_test.go (Missing dep tests)
T008 [P] - registry_test.go (Circular dep tests)
T009 [P] - registry_test.go (Init order tests)
T010 [P] - version_test.go (Version constraint tests)
```

**Wave 3: Graph Implementation** (sequential due to dependencies)
```
T011 - DetectCycles() implementation
T012 - TopologicalSort() implementation  
T013 [P] - Graph utilities
```

**Wave 4: Registry Core** (sequential within group, due to shared file)
```
T014 - Registry structure
T015 - Register method
T016 - ValidateDependencies method
T017 - InitializePlugins method
T018 - Get method
T019 - GetForDependent method
T020 [P] - List method (can overlap with T019)
T021 [P] - Helper methods (can overlap with T019-T020)
```

**Wave 5: Interface & Integration Tests** (can run simultaneously)
```
T022 [P] - interface.go updates
T023 [P] - interface.go docs
T024 [P] - mock_plugin_test.go
T025 [P] - integration test: composition
T026 [P] - integration test: transitive
T027 [P] - integration test: policies
T028 [P] - integration test: backward compat
```

**Wave 6: Main Integration** (sequential)
```
T029 - main.go updates
T030 - plugins_import.go updates
```

**Wave 7: Examples** (can run during or after Wave 6)
```
T031 [P] - lineinfile plugin update
```

**Wave 8: Polish** (can run simultaneously)
```
T032 [P] - registry_perf_test.go
T033 [P] - docs/plugins.md
T034 [P] - docs/architecture.md
T035 [P] - docs/schema.md
```

---

## Parallel Execution Examples

### Example 1: Foundation Phase
```bash
# Launch T001-T005 together
Task: "Create error types in internal/plugin/errors.go with ErrPluginNotFound, ErrCircularDependency, etc."
Task: "Create metadata types in internal/plugin/metadata.go with PluginMetadata and Dependency structs"
Task: "Create version constraint in internal/plugin/version.go with ParseVersionConstraint and Satisfies"
Task: "Create configuration types in internal/plugin/config.go with RegistryConfig and policy constants"
Task: "Create dependency graph structure in internal/plugin/dependency_graph.go"
```

### Example 2: Contract Tests Phase
```bash
# Launch T006-T010 together (all must fail initially)
Task: "Contract test Register and Get in internal/plugin/registry_test.go"
Task: "Contract test Missing dependency detection in internal/plugin/registry_test.go"
Task: "Contract test Circular dependency detection in internal/plugin/registry_test.go"
Task: "Contract test Initialization order in internal/plugin/registry_test.go"
Task: "Contract test Version constraint validation in internal/plugin/version_test.go"
```

### Example 3: Documentation Phase
```bash
# Launch T033-T035 together
Task: "Update plugin documentation in docs/plugins.md with dependency system guide"
Task: "Update architecture documentation in docs/architecture.md with registry design"
Task: "Add registry JSON schema in docs/schema.md"
```

---

## Task Validation Checklist
*GATE: Checked before task execution begins*

- [x] All contracts from registry-api.md have corresponding tests (T006-T010)
- [x] All entities from data-model.md have implementation tasks
  - [x] PluginRegistry (T014-T021)
  - [x] PluginMetadata (T002)
  - [x] Dependency (T002)
  - [x] VersionConstraint (T003)
  - [x] DependencyGraph (T005, T011-T013)
  - [x] RegistryConfig (T004)
  - [x] Error types (T001)
- [x] All tests come before implementation (T006-T010 before T011+)
- [x] Parallel tasks truly independent (different files or independent sections)
- [x] Each task specifies exact file path
- [x] No task modifies same file as another [P] task
- [x] Constitution principles addressed:
  - [x] Onboarding: T029-T030 (zero deps, binary-only)
  - [x] Schema: T002, T035 (clear metadata, JSON schema)
  - [x] Plugin-centric: T022-T024 (interface extensions)
  - [x] Safety: T016, T027 (validation, policy modes)
  - [x] Performance: T032 (timing tests, <50ms target)
  - [x] Composability: T025-T026 (plugin composition tests)
  - [x] Consistency: T001, T033 (error messages, docs)

---

## Notes

- **TDD Enforcement**: Tests T006-T010 must fail before proceeding to implementation
- **Commit Strategy**: Commit after each task completion for clean history
- **Test Verification**: Run `go test ./...` after each implementation task
- **Coverage Target**: Aim for >85% coverage on registry core
- **Performance Validation**: Run T032 benchmarks before marking complete
- **Documentation**: Update docs immediately after implementation (don't defer to end)

---

## Constitution Alignment

This task breakdown follows Streamy's constitutional principles:

1. **Onboarding First**: No external dependencies, pure Go implementation (T001-T035)
2. **Schema Clarity**: Clear metadata structures, JSON schema docs (T002, T035)
3. **Plugin-Centric**: Registry enables plugin composition (T014-T021)
4. **Safety by Default**: Validation before execution, policy modes (T016, T027)
5. **Performance**: <50ms validation target, benchmarks (T032)
6. **Extensibility**: Backward compatible, optional interfaces (T022-T024, T028)
7. **Consistency**: Structured errors, clear docs (T001, T033-T035)

---

**Total Tasks**: 35  
**Estimated Parallel Opportunities**: 18 tasks can run in parallel across 8 waves  
**Critical Path Length**: ~17 sequential tasks  
**Constitution Compliance**: 100% (all 7 principles addressed)
