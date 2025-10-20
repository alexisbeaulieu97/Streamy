# Testing Guide

This document summarises how Streamy's automated tests are organised and how to
run them efficiently across the domain-driven architecture.

## Test Pyramid

| Layer               | Location                                   | Notes |
|---------------------|--------------------------------------------|-------|
| **Domain**          | `internal/domain/<feature>/*_test.go`      | Pure logic, table-driven tests, no infrastructure. |
| **Application**     | `internal/application/.../*_test.go`       | Use mocks from `internal/application/<feature>/testutil`. Targets use cases and validation services. |
| **Infrastructure**  | `internal/infrastructure/.../*_test.go`    | Adapter behaviour (YAML loader, executor, logging). |
| **Plugins**         | `internal/plugins/<name>/*_test.go`        | Unit coverage per plugin + shared contract suite (`internal/plugins/contracts_test.go`). |
| **End-to-End**      | `tests/*.go`                               | Drives application ports via the test harness. |
| **CLI**             | `cmd/streamy/*_test.go`                    | Cobra command wiring, error formatting, observability harness. |

System-level integration tests live under `/tests` and operate through the same
application use cases used in production, ensuring parity between the CLI and
programmatic flows.

## Running Tests

```bash
# Entire suite (default target for CI)
go test ./...

# Domain-only (fast feedback, <100ms total)
go test ./internal/domain/...

# Application use cases with coverage
go test ./internal/application/... -cover

# Infrastructure adapters (includes executor integration tests)
go test ./internal/infrastructure/...

# Plugin contract suite
go test ./internal/plugins

# Integration scenarios
go test ./tests
```

The integration harness (`tests/harness.go`) wires in-memory adapters, ports
plugins, and use cases. To add high-level scenarios, register any additional
plugins or adapters via the harness helper functions.

## Common Patterns

- **Table-driven tests**: preferred for domain and application layers. Use
  `t.Run(name, func(t *testing.T) { ... })` for clarity.
- **Context propagation**: when testing cancellation, create a context with
  timeout/WithCancel and assert `ErrCodeCancelled` on returned domain errors.
- **Mocks**: use the helpers in `internal/application/<feature>/testutil` to
  avoid reimplementing common port stubs.
- **Golden files**: stored under `testdata/` (e.g. YAML configs) for loader and
  executor assertions. Use `t.TempDir()` for temporary mutations.
- **Concurrent tests**: mark with `t.Parallel()` where safe to reduce runtime.

## Coverage & Reporting

```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

Success criteria tracked by the Phase 9 checklist:
- Domain tests complete in under 100ms.
- Application layer coverage ≥ 90%.
- Infrastructure layer coverage ≥ 75%.

CI executes `go test ./...` and `golangci-lint run` on every change; add the
coverage profile command locally before large refactors.

## Benchmarks & Performance

- DAG/executor benchmarks live in `internal/infrastructure/engine/*_test.go`
  (guarded by `Benchmark...` functions). Run with `go test ./internal/infrastructure/engine -bench=. -benchmem`.
- Plugin-specific performance tests should live alongside the plugin package if
  a new implementation introduces heavy computation.

## Observability Harness

`cmd/streamy/observability_test.go` runs selected Cobra commands (`streamy
verify`, `streamy apply`) end-to-end while capturing logs, events, and metrics.
Use it to assert correlation IDs, structured logging, and span propagation.

## Gotchas

- Do not create background contexts in tests. Always propagate a caller-provided
  context to ensure cancellation is testable.
- Avoid relying on global state (e.g., environment variables) unless the test
  restores original values with `t.Cleanup`.
- When manipulating file paths, prefer `filepath.Join` and `t.TempDir()` to keep
  tests cross-platform.

## Adding New Tests

When adding a feature:
1. Start with domain invariants.
2. Add application-level unit tests using mocks.
3. Extend infrastructure/adapter tests for I/O behaviours.
4. Add integration coverage if the feature spans multiple layers or introduces
   regressions in the end-to-end flow.
5. Update this guide if the testing strategy evolves.
