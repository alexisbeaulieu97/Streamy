# Feature Specification: Pipeline Dependencies

**Feature Branch**: `010-pipeline-dependencies`  
**Created**: October 28, 2025  
**Status**: Draft  
**Input**: User description: "Streamy users often maintain multiple configs that must run in a specific order; today they have to chain them by hand. Extending the registry so pipelines can declare dependencies would let Streamy orchestrate end-to-end environments with one command, giving teams a shared source of truth for complex setups. Users gain clearer visibility (e.g., dashboard, status cache) into how higher-level workflows depend on one another, plus safer automation: when an upstream pipeline fails, downstream runs can be skipped or marked blocked instead of drifting out of sync."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Declare Pipeline Dependencies (Priority: P1)

A DevOps engineer managing a multi-tier application environment needs to declare that their "app-deployment" pipeline depends on "database-setup" and "network-config" pipelines. They add a `dependencies` field to their pipeline YAML configuration so the relationships live in source control and are auto-imported when the pipeline is registered.

**Why this priority**: This is the foundation of the feature. Without the ability to declare dependencies in config, no source-controlled orchestration is possible. This enables teams to version dependency relationships alongside their pipeline definitions.

**Independent Test**: Can be fully tested by adding a `dependencies` list to a pipeline YAML, running `streamy add <config-file>`, and verifying that the dependency relationships are correctly imported into the registry and queryable. Delivers immediate value by documenting pipeline relationships in version control.

**Acceptance Scenarios**:

1. **Given** a pipeline config with top-level `dependencies` field listing upstream pipeline IDs, **When** a user runs "streamy add <config-file>", **Then** the registry imports the dependency relationships
2. **Given** a pipeline config that depends on unregistered pipelines, **When** added to the registry, **Then** Streamy accepts the registration with a warning and marks the pipeline as "Blocked" (reason: unregistered dependencies)
3. **Given** a pipeline with dependencies, **When** queried via "streamy registry list --tree", **Then** the system displays the dependency tree with visual status indicators (🟢 Ready, 🟠 Blocked with unresolved IDs listed)
4. **Given** a pipeline configuration with circular dependencies (A→B→C→A), **When** loaded into the registry, **Then** Streamy rejects the configuration with a clear error message showing the cycle path

---

### User Story 2 - Automatic Dependency Resolution and Execution (Priority: P2)

A platform engineer runs a single command to apply a high-level pipeline that orchestrates an entire environment setup. Streamy automatically resolves all dependencies, determines the correct execution order, and runs upstream pipelines before downstream ones.

**Why this priority**: This delivers the core automation value. Users can replace manual chaining scripts with a single command, reducing errors and cognitive load.

**Independent Test**: Can be fully tested by creating a dependency graph (A→B, A→C, B→D, C→D) and running the leaf pipeline (D). Streamy should automatically execute A, then B and C in parallel, then D. Delivers the primary value proposition: automated orchestration with one command.

**Acceptance Scenarios**:

1. **Given** a pipeline with resolved dependencies, **When** a user runs `streamy run --registry <id>@<version>`, **Then** Streamy executes all upstream dependencies in topologically sorted order before executing the target pipeline
2. **Given** a dependency graph where multiple pipelines can run in parallel (no inter-dependencies), **When** executing a downstream pipeline, **Then** Streamy runs independent upstream pipelines concurrently
3. **Given** a complex dependency tree (5+ levels deep), **When** executing the leaf pipeline, **Then** Streamy correctly resolves and executes all ancestors in the proper order
4. **Given** a pipeline already executed successfully in the current session, **When** another pipeline depends on it, **Then** Streamy skips re-execution unless explicitly requested

---

### User Story 3 - Failure Handling and Blocked State Propagation (Priority: P3)

When an upstream pipeline fails during orchestrated execution, the DevOps team needs downstream pipelines to be automatically skipped or marked as blocked, preventing inconsistent or broken states from propagating through the environment.

**Why this priority**: This adds safety and reliability to automation. While P1 and P2 enable orchestration, this ensures that automation is safe and predictable when things go wrong.

**Independent Test**: Can be fully tested by creating a dependency chain (A→B→C) where B is configured to fail. Running C should execute A, then B (which fails), then skip C and mark it as "blocked due to upstream failure". Delivers production-ready reliability for automated workflows.

**Acceptance Scenarios**:

1. **Given** a pipeline execution where an upstream dependency fails, **When** the failure occurs, **Then** all downstream pipelines are marked as "blocked" and not executed
2. **Given** a dependency graph where one branch fails (A→B fails, A→C succeeds), **When** D depends on both B and C, **Then** D is marked as "blocked" due to B's failure
3. **Given** a failed pipeline execution, **When** a user views the status, **Then** the system shows which upstream failure caused the block
4. **Given** a blocked pipeline, **When** a user provides a "--force" flag, **Then** the system allows execution despite upstream failures with a warning
5. **Given** a pipeline D that depends on both B and C, **When** B fails and C succeeds, **Then** D remains blocked until every listed dependency reports success (no partial/OR semantics)

---

### User Story 4 - Dependency Visibility and Status Tracking (Priority: P4)

> **Status**: Deferred. Dashboard work remains a future enhancement and is not included in the 010 implementation scope—Release 010 delivers CLI visibility only.

A team lead needs to understand the dependency relationships across all pipelines and track execution status across a complex orchestrated run. They access a dashboard or command that shows the dependency graph and real-time execution status of all related pipelines.

**Why this priority**: This enhances operational visibility and debugging. While the core orchestration works with P1-P3, this makes it observable and debuggable for teams.

**Independent Test**: Can be fully tested by setting up a dependency graph, running orchestrated execution, and querying status during/after execution. The dashboard should show the graph structure, which pipelines are running/completed/failed/blocked. Delivers operational transparency for complex workflows.

**Acceptance Scenarios**:

1. **Given** multiple pipelines with dependencies, **When** a user runs "streamy list --dependencies", **Then** the system displays a visual representation of all dependency relationships
2. _(Deferred – dashboard)_ **Given** an orchestrated execution in progress, **When** a user views the dashboard, **Then** real-time status indicators are shown for each pipeline (current level: `running`, completed levels: `ready`, blocked dependents: `blocked`)
3. _(Deferred – dashboard)_ **Given** a completed orchestrated execution, **When** a user queries historical status, **Then** the system retrieves and displays the cached execution results
4. **Given** a pipeline with many dependencies, **When** a user requests dependency information via CLI, **Then** the system shows both direct dependencies and the full transitive closure

---

### Edge Cases

- What happens when a pipeline is removed from the registry but other pipelines still depend on it? System should mark dependent pipelines as "Blocked" with reason "unregistered dependency: <pipeline-id>" and warn users before orchestrated execution.
- How does the system handle circular dependencies across more than 2 pipelines (A→B→C→D→A)? System must detect all cycles during `streamy add` and reject configurations with clear error messages indicating the cycle path.
- What if a dependency declaration references a pipeline ID that doesn't exist? System accepts registration but marks pipeline as "Blocked" with clear warning listing the unresolved pipeline IDs. When the missing dependency is later registered, status automatically transitions to "Ready".
- What happens when a pipeline is updated in the registry while orchestrated execution is in progress? System should use the pipeline definitions that were loaded at execution start (snapshot semantics) to prevent mid-flight inconsistencies.
- How does the system handle timeouts for long-running upstream dependencies? System should support per-pipeline timeout configurations and propagate timeout failures as "Blocked" status to downstream pipelines.
- How are different versions of a pipeline handled? Dependencies reference concrete registry IDs. If a team needs different variants, they register them as separate entries (e.g., "db-setup-v1", "db-setup-v2") and dependents choose the appropriate ID. No semver constraints or version resolution is performed—the registry is the single source of truth for which pipeline definitions are available. *(See Assumptions section for full versioning rationale.)*

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Pipeline configurations MUST support a top-level "dependencies" field containing a list of upstream pipeline IDs in canonical `<id>@<version>` format
- **FR-001a**: Pipeline configurations MUST include explicit `id` and `version` fields; configs without these fields MUST be rejected with a validation error
- **FR-001b**: The registry MUST store pipelines using the canonical identifier format `<id>@<version>` (e.g., `app-deploy@1.0`), enabling multiple versions of the same logical pipeline to coexist
- **FR-002**: The "streamy add" command MUST validate that every dependency entry is in canonical `<id>@<version>` format before importing it into the registry
- **FR-003**: The registry MUST accept pipelines with unregistered dependencies, marking them as "Blocked" with clear warnings listing unresolved dependency IDs
- **FR-004**: The registry MUST detect circular dependencies during import and reject any configuration that would create a cycle, showing the complete cycle path in the error message
- **FR-005**: When executing a pipeline with dependencies via `streamy run --registry <id>@<version>`, the system MUST automatically resolve the complete dependency graph (including transitive dependencies)
- **FR-006**: The system MUST execute pipelines in topologically sorted order, ensuring all upstream dependencies complete successfully before downstream pipelines begin
- **FR-007**: The system MUST support concurrent execution of independent pipelines (pipelines at the same dependency level with no inter-dependencies)
- **FR-008**: When an upstream pipeline fails, the system MUST mark all downstream dependents as "Blocked" and skip their execution
- **FR-008a**: Downstream pipelines MUST require that *all* listed dependencies complete successfully before they run; a single failure keeps the dependent pipeline blocked (AND semantics).
- **FR-009**: The system MUST track pipeline status in the registry using two states: Ready (all dependencies registered and resolvable), Blocked (unregistered dependencies OR upstream failure) and MUST attach `BlockedBy` metadata listing the blocking pipeline IDs or missing dependencies so the CLI can explain the reason
- **FR-010**: The "streamy registry list --tree" command MUST visualize dependency trees with colored status indicators (🟢 Ready, 🟠 Blocked with unresolved IDs listed)
- **FR-011**: Users MUST be able to view dependency graphs and execution status through CLI commands; dashboard support is deferred to a future milestone (010 backlog)
- **FR-012**: The system MUST cache execution status to provide visibility into orchestrated runs after completion (surfaceable via CLI; dashboard consumption deferred)
- **FR-013**: Users MUST be able to force execution of a pipeline despite upstream *failure* using an explicit override flag; forcing does not bypass validation for missing or unregistered dependencies.
- **FR-013a**: The `--force` flag MUST present an interactive confirmation listing the pipelines that will run despite being blocked; non-interactive environments MUST require an explicit `--yes` acknowledgment to proceed
- **FR-014**: The system MUST support skipping already-executed pipelines in the same orchestrated session unless explicitly requested to re-run. *Session* is defined as a single invocation of `streamy run --registry`; state does not persist across separate CLI executions.
- **FR-015**: Error messages for dependency issues (circular references, blocked pipelines) MUST clearly indicate the root cause and affected registry pipeline IDs
- **FR-016**: Dependencies MUST reference concrete registry pipeline IDs (no version ranges or semver constraints); the registry serves as the single source of truth for available pipeline definitions

### Assumptions

- **Source Control Integration**: Dependencies are declared in pipeline configuration YAML files via the top-level `dependencies` field. The registry auto-imports these relationships during `streamy add`, making the registry a runtime mirror of source-controlled metadata.
- **Blocked State Workflow**: Pipelines can be registered even if their dependencies are not yet registered. The registry marks them as "Blocked" with clear warnings listing unresolved dependency IDs. When missing dependencies are later registered, dependent pipelines automatically transition to "Ready" status. Execution summaries use the `BlockedBy` field to record the immediate cause (missing dependency or upstream failure), ensuring the CLI communicates the reason.
- **Versioning**: The feature does not introduce per-dependency version constraints or semver resolution. Each registry entry represents one concrete pipeline definition. If teams need different variants of a pipeline, they register them as separate entries (e.g., "db-setup-v1", "db-setup-v2") and dependents explicitly choose the appropriate registry ID. This keeps orchestration simple and aligned with Streamy's existing registry semantics.
- **Registry Stability**: Dependencies are resolved at the start of orchestrated execution using a snapshot of the registry state. Updates to the registry during execution do not affect in-flight orchestration.
- **Execution Context**: Orchestrated execution tracking (status cache, session management) is scoped to a single invocation. Cross-invocation state is not persisted beyond what the existing registry and configuration files provide.

### Key Entities

- **Pipeline Identifier**: Canonical format is `<id>@<version>` (e.g., `app-deploy@1.0`). Built from required `id` and `version` fields in pipeline YAML. Enables multiple versions of the same logical pipeline to coexist in the registry.
- **Pipeline Dependency**: Declared in the YAML `dependencies` field and imported into the registry. Represents a relationship where one pipeline (dependent) requires another (dependency) to execute first. Stored as a list of upstream pipeline IDs in canonical `<id>@<version>` format.
- **Pipeline Status**: Enum with two states: Ready (all dependencies registered and last obstacle cleared), Blocked (unregistered dependencies OR upstream failure during execution). Blocked pipelines include a list of unresolved dependency IDs or blocking failure reason.
- **Orchestrated Execution**: Represents a single invocation of a pipeline with dependency resolution via `streamy run --registry`. Tracks: root registry pipeline ID, complete dependency graph, execution order, start time, completion time, overall status.
- **Pipeline Execution Result**: Represents the outcome of a single pipeline within an orchestrated execution. Attributes include: registry pipeline ID, status (ready/blocked), success boolean, blocked-by reason (if blocked), summary, duration.
- **Dependency Graph**: Represents the resolved directed acyclic graph (DAG) of all pipelines involved in an orchestrated execution. Contains nodes (registry pipeline IDs in `<id>@<version>` format) and edges (dependency relationships), along with topological ordering.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can declare dependencies for a pipeline and have them validated by the registry in under 5 seconds for graphs with up to 100 pipelines
- **SC-002**: The system correctly resolves and executes dependency graphs with up to 50 pipelines and 10 levels of depth without errors
- **SC-003**: When an upstream pipeline fails, downstream pipelines are marked as blocked within 1 second of the failure being detected
- **SC-004**: Users can visualize the complete dependency graph and execution status for all related pipelines via CLI outputs (`streamy registry list --tree`, `streamy run --registry ... --dry-run`) without relying on a dashboard
- **SC-005**: Orchestrated execution reduces user intervention time by 80% compared to manual chaining (measured by number of commands and context switches required)
- **SC-006**: 90% of users can successfully create and execute dependency-based pipeline orchestration without referring to documentation after viewing one example
- **SC-007**: The system prevents 100% of invalid configurations (circular dependencies, missing references) from being registered in the system
- **SC-008**: Independent pipelines at the same dependency level execute concurrently, reducing total orchestration time by at least 30% compared to sequential execution for typical multi-tier setups
