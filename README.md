# Streamy

Streamy is a declarative environment setup tool. Describe packages, repositories, symlinks, file copies, and shell commands in YAML, then register those pipelines with `streamy registry add <file>` to reproduce complete dependency graphs with dry-run previews, automatic status reconciliation, and a Bubbletea-powered TUI. Plugins read configuration exclusively through `step.DecodeConfig(&config.<Type>Step{})`, and helpers/tests populate step payloads with `step.SetConfig(config.<Type>Step{...})`.

## Features

- 🧩 **DAG Execution Engine** – Automatically orders steps based on `depends_on` relationships and executes independent steps in parallel.
- 🕸️ **Pipeline Orchestration** – Register multiple pipelines, visualize dependency trees, and execute graphs with `streamy run --registry <id>@<version>` (with safety prompts for forced overrides).
- 🔌 **Plugin Architecture** – Built-in plugins for package, repo, symlink, copy, and command steps; easily extensible for new step types.
- 🛡️ **Safety & Idempotency** – Per-step `Check` methods, dry-run mode, and post-execution validations keep runs predictable.
- 📊 **Interactive TUI** – Rich terminal UI shows live progress; falls back to plain output when running in non-interactive contexts.
- 🧪 **Extensive Testing** – Unit and integration tests cover core flows, error conditions, and validation behaviours.

## Installation

Build locally (Go 1.25.1 or newer):

```bash
go build ./cmd/streamy
```

After Phase 3.14 is complete you can use the install/build scripts in `scripts/` or CI releases for prebuilt binaries.

## Quick Start

1. Create a config (see `testdata/configs/simple.yaml`), making sure it includes a canonical `id`, `version`, and (optionally) a `dependencies` array:

   ```yaml
   id: simple-example
   version: "1.0"
   name: "Simple Example"
   steps:
     - id: say_hello
       type: command
       command: "echo hello"
     - id: say_goodbye
       type: command
       depends_on:
         - say_hello
       command: "echo goodbye"
   validations:
     - type: command_exists
       command: echo
   ```

2. Register the pipeline (Streamy stores it in `~/.streamy/registry.json`):

   ```bash
   streamy registry add simple.yaml
   ```

   Use canonical IDs of the form `<id>@<version>` everywhere inside the registry. If you reference other pipelines in the `dependencies` list, make sure each of those pipelines is registered as well.

3. Inspect the registry:

   ```bash
   streamy registry list --tree
   ```

   Missing upstream pipelines automatically mark dependents as **Blocked** (`🟠`). Once you add the missing pipelines, Streamy reconciles the status and shows them as **Ready** (`🟢`).

4. Run the registered pipeline (and its graph) end to end:

   ```bash
   streamy run --registry simple@1.0
   ```

   Add `--dry-run` for a safe preview, `--non-interactive` for CI output, and `--force --yes` if you really need to override upstream failures (you’ll be prompted in interactive sessions).

   Tip: append `@latest` (e.g. `streamy run --registry setup-pnpm@latest`) to resolve the highest registered version at runtime. Streamy prints the concrete version it executes so you keep an audit trail.

To execute a single YAML file without registering it, use:

```bash
streamy run --file simple.yaml
```

   ```bash
   streamy apply --config simple.yaml
   ```

   Use `--dry-run` for a safe preview and `--verbose` for detailed logging.

## Dashboard

Streamy includes an interactive TUI dashboard for managing multiple pipeline configurations:

```bash
streamy dashboard
```

### Features

- **Pipeline Overview**: View all registered pipelines with real-time status indicators
- **Status Tracking**: Visual status icons (🟢 satisfied, 🟡 drifted, 🔴 failed, ⚪ unknown)
- **Smart Sorting**: Pipelines automatically sorted by priority (failed > drifted > satisfied > unknown)
- **Operations**:
  - **Verify**: Check if pipeline configuration matches actual system state
  - **Apply**: Modify system to match configuration (with confirmation)
  - **Refresh**: Re-verify all pipelines or a single pipeline
- **Status Caching**: Fast startup with cached statuses from previous runs
- **Keyboard Navigation**: Efficient keyboard-driven interface

### Dashboard Usage

1. **Register pipelines**:
   ```bash
   streamy registry add ./configs/dev-env.yaml
   streamy registry add ./configs/prod-env.yaml
   ```

2. **Launch dashboard**:
   ```bash
   streamy dashboard
   ```

3. **Keyboard Shortcuts**:
   - **List View**:
     - `↑`/`↓` or `k`/`j`: Navigate pipelines
     - `Enter`: View pipeline details
     - `1`-`9`: Jump to pipeline by number
     - `r`: Refresh all pipelines
     - `?`: Show help
     - `q`: Quit
   
   - **Detail View**:
     - `v`: Verify pipeline
     - `a`: Apply changes (requires confirmation)
     - `r`: Refresh this pipeline
     - `Esc`: Back to list
     - `?`: Show help
     - `q`: Quit
   
   - **Help View**:
     - `?`/`Esc`/`q`: Close help
   
   - **Confirmation Dialog**:
     - `y`: Confirm action
     - `n`/`Esc`: Cancel

4. **Status Indicators**:
   - 🟢 **Satisfied**: System matches configuration
   - 🟡 **Drifted**: System differs from configuration
   - 🔴 **Failed**: Verification or apply failed
   - ⚪ **Unknown**: Not yet verified
   - ⚙️ (spinner): Operation in progress

### Pipeline Management

```bash
# Register a pipeline
streamy registry add <config-path> [--description "Pipeline description"]

# List registered pipelines
streamy registry list

# Unregister a pipeline
streamy registry remove <pipeline-id>

# Verify a single pipeline (CLI)
streamy verify <pipeline-id>
```

The dashboard provides a real-time view of all registered pipelines with interactive operations. Status information is cached in `~/.streamy/status-cache.json` for fast startup.

## Configuration Reference

- **Root fields**: `version`, `name`, `description`, `settings`, `steps`, `validations`.
- **Settings**: `parallel` (1-32), `timeout` seconds (1-3600), `continue_on_error`, `dry_run`, `verbose`.
- **Steps**: Each requires `id`, `type`, optional `depends_on`. See [docs/schema.md](docs/schema.md) for type-specific fields.
- **Validations**: `command_exists`, `file_exists`, `path_contains` (post-execution).

## CLI Usage

```bash
streamy run --file path/to/config.yaml [--dry-run]
streamy run --registry pipeline@1.0 [--dry-run] [--force --yes]
streamy version
```

- `streamy run --file`: Parses and validates a local config, builds the execution plan, runs steps via registered plugins, and displays progress.
- `streamy run --registry`: Resolves the registered pipeline, executes its dependency graph, updates the status cache, and enforces graph-level safeguards (`--force` requires `--yes` in CI).
- `streamy version`: Prints build metadata (version, commit, build date) injected via `-ldflags`.

## Architecture Overview

- **Domain (`internal/domain`)** – Pipeline aggregate, steps, execution plan, results, validation definitions, domain errors.
- **Ports (`internal/ports`)** – Interfaces for config loading, execution, logging, metrics, tracing, plugins, and events.
- **Application (`internal/application`)** – Use cases (prepare/apply/verify) and validation service coordinating domain logic via ports.
- **Infrastructure (`internal/infrastructure`)** – Adapters implementing ports plus built-in plugins (`internal/plugins`). Includes YAML loader, DAG builder/executor, observability, registry.
- **CLI (`cmd/streamy`)** – Composition root wiring infrastructure adapters into application services; exposes Cobra commands and observability harness.
- **UI (`internal/tui`)** – Bubbletea models/views for the dashboard.

See [docs/architecture.md](docs/architecture.md) for diagrams and detailed data flow.

## Development

```bash
go fmt ./...
go test ./...
```

Use `go test ./... -run Integration` to focus on integration tests under `tests/`. The project expects `goimports` formatting and follows standard Go module layout.

### Testing

The project maintains **85.5% test coverage** on core business logic (see [docs/testing-guide.md](docs/testing-guide.md)):

```bash
# Run all tests including integration tests
go test ./...

# Run tests with coverage for core packages
go test ./internal/... ./pkg/... -coverprofile=coverage.out -covermode=atomic

# View coverage report
go tool cover -html=coverage.out
```

CI enforces 80% minimum coverage on `internal/` and `pkg/` packages. The `cmd/` package (CLI layer) is excluded as it's a thin wrapper around tested business logic.

## Extending Streamy

1. Define new step fields in `internal/infrastructure/config/schema.go` (and companion validation helpers) and extend domain validation as needed.
2. Implement a plugin under `internal/plugins/<type>/` and register it (see [docs/adding-plugins.md](docs/adding-plugins.md)).
3. Add fixtures/tests to `tests/` and documentation to `docs/` + README. Consult [docs/testing-guide.md](docs/testing-guide.md) for recommended coverage.

Refer to [docs/plugins.md](docs/plugins.md) for a plugin development checklist.

## Roadmap

- Additional package manager plugins (brew, choco, winget).
- Enhanced logging sinks and JSON output.
- Config composition/inheritance.
- Optional rollback hooks per plugin.

## License

TBD (add your preferred license file).
