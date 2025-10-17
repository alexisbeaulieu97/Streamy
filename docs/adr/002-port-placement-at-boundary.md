# ADR-002: Port Placement at the Application Boundary

**Date**: 2025-10-16  
**Status**: Accepted  
**Related**: [ADR-001: Domain-Driven Refactor](001-domain-driven-refactor.md) | [Feature Spec](../../specs/009-domain-driven-refactor/spec.md)

## Context

During Phase 1 of the domain-driven refactor, the team defined the initial contract interfaces under `specs/009-domain-driven-refactor/contracts/`. Those documents describe the intent for how ports should behave, but they still left an open decision: **in which package should these interfaces live inside the Go module?**

Early drafts placed the port interfaces under `internal/domain/pipeline/ports.go`. That location matched the idea that the domain “owns” its abstractions, but it introduced two major issues:

1. **Import Cycles**: Infrastructure adapters located under `internal/infrastructure/...` need to implement these ports. While doing so they import domain entities. If the interfaces also live in the domain package, the adapter imports the domain package that now exposes interfaces referencing other domain subpackages, leading quickly to cycles as the domain grows.
2. **Leaky Domain Purity**: One of the primary goals of the refactor is to keep the domain layer completely unaware of infrastructure concerns. If ports live in the domain, then the domain layer explicitly knows about infrastructure requirements (e.g., logger APIs, metrics functions). That contradicts the North Star architecture from ADR-001.

Industry guidance (Clean/Hexagonal/Onion architectures) often suggests placing port interfaces in the **outer layer** that depends on them. In Streamy’s case, application services are the ones that depend on infrastructure behaviors. Therefore, placing ports in a distinct package that sits alongside the application layer accomplishes several goals:

- Maintains a pure domain (`internal/domain/`) with zero knowledge of infrastructure or adapter contracts.
- Provides a shared package (`internal/ports/`) that both application services and infrastructure adapters can import without forming cycles.
- Keeps the dependency direction consistent: Infrastructure → Ports → Application → Domain.

## Decision

Create a dedicated `internal/ports/` package housing **all** port interfaces required by the application layer. The package structure mirrors the conceptual groupings from the contracts' documentation:

```
internal/ports/
├── config.go          # ConfigLoader
├── execution.go       # PluginExecutor, DAGBuilder, ExecutionPlanner
├── logging.go         # Logger + correlation helpers
├── observability.go   # MetricsCollector, Tracer, Span
├── plugins.go         # Plugin, PluginRegistry
├── events.go          # EventPublisher, Subscription, DomainEvent
└── registry.go        # RegistryStore, ValidationService
```

Key characteristics of this decision:

- `internal/domain/` defines business entities, value objects, and domain errors only.
- `internal/application/` imports `internal/ports` to orchestrate use cases.
- `internal/infrastructure/` implements interfaces from `internal/ports` and may import domain entities where required by method signatures.
- No other package is allowed to declare new ports outside of `internal/ports/` without an ADR update.

## Consequences

### Positive

- **Enforced Purity**: Domain code cannot accidentally pull in port interfaces, because they live in a separate package. Static analysis (`depguard`) now forbids domain packages from importing `internal/ports`.
- **Cycle Prevention**: The central `internal/ports/` package breaks potential import cycles between domain/application and infrastructure.
- **Documentation Alignment**: The new package structure matches the contracts created in Phase 1 and the quickstart guidance. Engineers can quickly locate relevant interfaces.
- **Scalability**: As ports grow (logging upgrades, metrics expansion, etc.), changes are localized and don’t pollute the domain namespace.

### Negative

- **Extra Package**: Developers must familiarize themselves with an additional package alongside the domain and application layers.
- **Duplicated Types**: Some domain types need forward declarations in the ports package to avoid introducing dependencies. For example, `type Pipeline struct{}` placeholders exist until the actual domain implementation is imported by adapters.

## Status & Next Steps

1. Implement `internal/ports/README.md` describing the rationale and rules (completed as part of Phase 1 tasks T015).
2. Update `specs/009-domain-driven-refactor/tasks.md` to reflect the port relocation (task T020).
3. Enforce the decision via lint rules (completed in `.golangci.yml`).

## Alternatives Considered

1. **Ports inside domain**: Rejected due to import cycles and domain impurity.
2. **Ports inside application subpackages**: Would cause duplication of interfaces between features and more scattered definitions.
3. **Ports under `pkg/`**: Would make interfaces public and accessible outside the module, conflicting with the intention to limit the refactor to internal packages until the design stabilizes.

## References

- [ADR-001: Domain-Driven Refactor](001-domain-driven-refactor.md)
- [Architecture Overview](../architecture-overview.md)
- [Quickstart Guide](../../specs/009-domain-driven-refactor/quickstart.md)

