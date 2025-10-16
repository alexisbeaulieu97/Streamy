# Streamy

Streamy is a declarative environment setup tool. Describe packages, repositories, symlinks, file copies, and shell commands in YAML, then run `streamy apply config.yaml` to reproduce the environment with dependency awareness, dry-run previews, and a Bubbletea-powered TUI. Plugins read configuration exclusively through `step.DecodeConfig(&config.<Type>Step{})`, and helpers/tests populate step payloads with `step.SetConfig(config.<Type>Step{...})`.

## Features

- 🧩 **DAG Execution Engine** – Automatically orders steps based on `depends_on` relationships and executes independent steps in parallel.
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

1. Create a config (see `testdata/configs/simple.yaml`):

   ```yaml
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

2. Run Streamy:

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
   streamy register dev-env ./configs/dev-env.yaml
   streamy register prod-env ./configs/prod-env.yaml
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
streamy register <name> <config-path> [--description "Pipeline description"]

# List registered pipelines
streamy list

# Unregister a pipeline
streamy unregister <name>

# Verify a single pipeline (CLI)
streamy verify <name>
```

The dashboard provides a real-time view of all registered pipelines with interactive operations. Status information is cached in `~/.streamy/status-cache.json` for fast startup.

## Configuration Reference

- **Root fields**: `version`, `name`, `description`, `settings`, `steps`, `validations`.
- **Settings**: `parallel` (1-32), `timeout` seconds (1-3600), `continue_on_error`, `dry_run`, `verbose`.
- **Steps**: Each requires `id`, `type`, optional `depends_on`. See [docs/schema.md](docs/schema.md) for type-specific fields.
- **Validations**: `command_exists`, `file_exists`, `path_contains` (post-execution).

## CLI Usage

```bash
streamy apply --config path/to/config.yaml [--dry-run] [--verbose]
streamy version
```

- `streamy apply`: Parses and validates the config, builds the execution plan, runs steps via registered plugins, and displays progress.
- `streamy version`: Prints build metadata (version, commit, build date) injected via `-ldflags`.

## Architecture Overview

- `internal/config`: YAML parsing and validation.
- `internal/engine`: DAG, planner, executor, execution context.
- `internal/plugin` and `internal/plugins/*`: Plugin interface and implementations.
- `internal/validation`: Post-execution checks.
- `internal/tui`: Bubbletea model/update/view and UI components.
- `cmd/streamy`: Cobra CLI wrapper around the engine.

See [docs/architecture.md](docs/architecture.md) for detailed flow diagrams and package responsibilities.

## Development

```bash
go fmt ./...
go test ./...
```

Use `go test ./... -run Integration` to focus on integration tests under `tests/`. The project expects `goimports` formatting and follows standard Go module layout.

### Testing

The project maintains **85.5% test coverage** on core business logic (see [docs/testing-strategy.md](docs/testing-strategy.md)):

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

1. Define new step fields in `internal/config/types.go` and extend validation.
2. Implement a plugin under `internal/plugins/<type>/` and register it.
3. Add fixtures/tests to `tests/` and documentation to `docs/` + README.

Refer to [docs/plugins.md](docs/plugins.md) for a plugin development checklist.

## Roadmap

- Additional package manager plugins (brew, choco, winget).
- Enhanced logging sinks and JSON output.
- Config composition/inheritance.
- Optional rollback hooks per plugin.

## License

TBD (add your preferred license file).
