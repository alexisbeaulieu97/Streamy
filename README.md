# Streamy

Streamy is a declarative environment setup tool. Describe packages, repositories, symlinks, file copies, and shell commands in YAML, then run `streamy apply config.yaml` to reproduce the environment with dependency awareness, dry-run previews, and a Bubbletea-powered TUI.

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
