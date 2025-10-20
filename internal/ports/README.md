# Ports Layer

Interfaces under `internal/ports` define the contracts that the application layer depends on. This layer lives at the **application boundary**, not inside the domain, to keep the domain completely unaware of I/O concerns.

## Responsibilities

- Declare interfaces for configuration, execution, logging, observability, and plugin coordination.
- Encode error and cancellation expectations (`context.Context` is always the first parameter).
- Provide documentation for each interface explaining usage patterns and expected error codes.

## Rules

1. **No Domain Imports**: Ports can reference domain types using forward declarations or dedicated DTOs, but they must not import concrete domain packages to avoid cycles.
2. **Small Interfaces**: Keep interfaces focused (2-5 methods). Split responsibilities rather than growing god interfaces.
3. **Context First**: Enforce `ctx context.Context` as the first parameter for all operations.
4. **Structured Errors**: Specify which domain error codes apply to each method.
5. **Documentation**: Every interface should include GoDoc describing the contract, failure modes, and relevant success criteria.

## Directory Layout

```
internal/ports/
├── config.go          # ConfigLoader interface
├── execution.go       # PluginExecutor, DAGBuilder, ExecutionPlanner
├── logging.go         # Logger interface with correlation helpers
├── observability.go   # MetricsCollector, Tracer, Span interfaces
├── plugins.go         # Plugin, PluginRegistry ports
├── events.go          # EventPublisher, Subscription contracts
└── registry.go        # RegistryStore, ValidationService ports
```

## When to Add a Port

- A use case requires data or behavior from outside the domain layer.
- The functionality has multiple potential implementations (e.g., different config sources).
- Tests need to mock or fake the dependency.

If the functionality is purely internal and not required by the application layer, keep it inside the infrastructure adapter instead of adding a port.

## Testing

- Use mock implementations located under `internal/application/.../testutil/` to verify interactions.
- Infrastructure adapters must implement these interfaces and include compile-time assertions: `var _ ports.ConfigLoader = (*YAMLLoader)(nil)`.

