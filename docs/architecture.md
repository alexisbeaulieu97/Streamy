# Streamy Architecture Overview

Streamy transforms declarative YAML configuration files into a dependency-aware execution plan that provisions a developer environment. The architecture is organized into layers that keep configuration parsing, validation, planning, execution, and presentation concerns isolated.

## High-Level Flow

1. **Configuration Parsing**: `internal/config` parses YAML into strongly typed Go structs (see `types.go`) and performs schema validation (required fields, dependency references, ID format, etc.).
2. **Graph Construction**: `internal/engine/dag.go` + `dag_builder.go` turn the list of steps into a directed acyclic graph (DAG) while detecting cycles and missing dependencies.
3. **Execution Planning**: `internal/engine/planner.go` performs a topological sort to produce level-based execution groups for parallelism and time estimation.
4. **Execution**: `internal/engine/executor.go` consumes the plan with a worker pool, dispatching steps to their registered plugins, capturing `StepResult` metadata and supporting dry-run mode.
5. **Validations**: `internal/validation` runs post-execution checks (command_exists, file_exists, path_contains) and aggregates results.
6. **Presentation**: `internal/tui` renders progress using Bubbletea, falling back to a textual summary when no TTY is available. CLI commands (`cmd/streamy`) orchestrate these steps.

## Key Packages

### internal/config
- Defines `Config`, `Step`, step-specific structs, and validation tags.
- `parser.go` loads YAML while annotating errors with file/line context.
- `validator.go` enforces structural and cross-step rules (unique IDs, dependency existence, cycle detection).

### internal/engine
- **DAG** (`dag.go`, `dag_builder.go`): Nodes consist of step metadata and edges represent `depends_on` relationships.
- **Planner** (`planner.go`): Produces `ExecutionPlan` with `ExecutionLevel` slices for parallel execution.
- **Executor** (`executor.go`): Runs each level with a worker pool, dispatching to plugins, respecting dry-run, timeouts, and cancellation.
- **Context** (`context.go`): Holds shared execution state (config, dry-run flag, worker semaphore, logger, results map).

### internal/plugin & internal/plugins
- `plugin/interface.go` defines the plugin contract (`Metadata`, `Schema`, `Check`, `Apply`, `DryRun`).
- `plugin/registry.go` keeps global plugin registrations.
- Concrete implementations under `internal/plugins/` (package, repo, symlink, copy, command) encapsulate system-specific behaviour and idempotency checks.

### internal/logger
- Wrapper around Zerolog for consistent structured logging with optional human-readable output.

### internal/validation
- `types.go` defines validation results.
- `validator.go` orchestrates validation runs; `checks.go` houses command/file/path helpers.

### internal/tui
- Bubbletea model/update/view for streaming execution progress.
- Components (progress bar, step list, summary) encapsulate presentation primitives.

### cmd/streamy
- Cobra CLI (`main.go`, `root.go`) exposes `streamy apply` and `streamy version` commands.
- `apply.go` wires together parsing, validation, execution, TUI display, and validation results.
- Flags (`flags.go`) enforce config presence and sensible defaults.

## Data Flow Diagram

```
YAML Config → config.ParseConfig → config.ValidateConfig → engine.BuildDAG
         ↘ validations (config.Validations)         ↘ engine.GeneratePlan
                                                   ↘ engine.Execute → plugin.Apply / DryRun
                                                   ↘ validation.RunValidations
                                                   ↘ tui.Model (Bubbletea) & logger.Logger
```

## Extensibility Points

- **Step Types**: Add new plugin implementations under `internal/plugins/<type>` and register them via `plugin.RegisterPlugin` in an `init()` block.
- **Validations**: Extend `internal/validation` by adding new helpers and wiring them into `RunValidations`.
- **CLI**: Additional commands/subcommands can be added in `cmd/streamy` while reusing core packages.

## Cross-Cutting Concerns

- **Logging**: Components accept a `logger.Logger` to ensure consistent metadata (step IDs, durations).
- **Dry-Run**: Executor passes `DryRun` flag through to plugins, effectively short-circuiting side effects while still showing intended operations.
- **Timeouts**: Steps inherit configurable timeout from `Config.Settings`. The executor uses context deadlines to enforce them.
- **Parallelism**: Worker pool size defaults to 4 but honors `settings.parallel` when present.
- **Testing**: Unit tests cover each layer (parsing, validation, DAG, executor, plugins, TUI). Integration tests (`tests/integration_test.go`) stitch modules together with fixture configs.

## Future Enhancements

- **Additional Plugins**: Extend `internal/plugins` with platform-specific package managers, service integrations, or infrastructure provisioning.
- **Observability**: Add structured event streaming for headless monitoring, integrate with logging sinks.
- **Config Includes**: Support config composition/imports for large environments.
- **Rollback Support**: Introduce optional rollback handlers per plugin for safer failure recovery.

This architecture balances separation of concerns with straightforward data flow, enabling contributors to reason about each phase—from configuration ingest to execution feedback—while keeping the system extensible for new step types, validations, and presentation layers.
