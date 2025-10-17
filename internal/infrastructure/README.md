# Infrastructure Layer

Infrastructure packages implement the port interfaces defined in `internal/ports`. Adapters live here: configuration loaders, execution engines, logging, persistence, metrics, tracing, and event delivery.

## Responsibilities

- Bridge external systems (filesystem, OS commands, network) with the domain.
- Provide concrete implementations for ports used by the application layer.
- Handle serialization, logging frameworks, metrics libraries, and external APIs.

## Rules

1. **Implement Ports**: Every adapter must implement at least one interface from `internal/ports`. Add compile-time assertions (`var _ ports.ConfigLoader = (*YAMLLoader)(nil)`).
2. **No Domain Logic**: Keep business rules in the domain layer. Adapters should transform data and delegate to domain constructs.
3. **Context & Cancellation**: Respect `context.Context` for all operations. Check `ctx.Err()` before long-running or blocking work.
4. **Error Wrapping**: Include technical details (file paths, command output) while wrapping errors with domain codes where appropriate.
5. **Configuration via Constructors**: Use explicit constructor functions that accept configuration structs and dependencies.
6. **Testing Strategy**: Provide unit tests using `t.TempDir()` or fakes. Where necessary, offer integration tests under `tests/`.

## Directory Layout

```
internal/infrastructure/
├── config/        # YAML/JSON loaders implementing ConfigLoader
├── engine/        # Execution adapters (DAG builder, executor)
├── logging/       # charmbracelet/log adapter implementing Logger
├── plugin/        # Registry implementations
├── metrics/       # MetricsCollector adapters
├── tracing/       # Tracer implementations
└── events/        # EventPublisher implementations
```

## Adding an Adapter

1. Identify the port interface that the adapter must satisfy.
2. Create a package under the appropriate subdirectory.
3. Implement the interface with dependency injection via constructors.
4. Add tests covering success and failure scenarios; use mocks for other ports.
5. Update wiring in `cmd/streamy/main.go` to instantiate the adapter during startup.

## Observability

- Logging adapters should extract correlation IDs from context and add structured fields.
- Metrics and tracing adapters provide no-op implementations for development to keep tests lightweight.

## Validation

Run infrastructure tests with:

```
go test ./internal/infrastructure/... -cover
```

Adapters must not introduce dependencies from infrastructure back into domain or application packages to prevent import cycles.

