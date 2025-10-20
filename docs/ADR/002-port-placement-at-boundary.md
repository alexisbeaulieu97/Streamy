# ADR 002: Port Placement at Application Boundary

- **Status**: Accepted
- **Date**: 2025-10-15
- **Context**: Domain-driven refactor (Spec 009)

## Problem

Legacy Streamy intertwined domain entities and infrastructure concerns. Step
definitions lived in `internal/config`, executor logic in `internal/engine`, and
plugins required knowledge of both packages. This caused cyclic dependencies and
made testing difficult. Ports (interfaces describing infrastructure behaviour)
needed a consistent home that kept the domain pure while allowing the
application layer to orchestrate workflows.

## Decision

Ports are defined at the **application boundary** (`internal/ports`). The domain
layer (`internal/domain`) remains pure and unaware of infrastructure. The
application layer (`internal/application`) depends on domain types and ports to
describe its collaborators.

```
Infrastructure ─▶ Ports ─▶ Application ─▶ Domain
```

Key implications:
- Domain layer exports aggregates, value objects, and domain errors only.
- Ports package contains interfaces for configuration loading, execution,
  logging, metrics, tracing, events, and plugin registry.
- Infrastructure implements the ports; application services accept interfaces in
  their constructors (manual dependency injection).
- Plugins implement `ports.Plugin` and use domain types exclusively.

## Consequences

**Positive**
- Domain tests run without infrastructure dependencies (fast, deterministic).
- Application services are easy to mock for unit testing.
- Infrastructure adapters are replaceable, enabling future integrations.
- Clear dependency direction; `golangci-lint` import rules enforce boundaries.

**Negative**
- Requires explicit wiring in `cmd/streamy/main.go`.
- Legacy packages (`internal/config`, `internal/engine`, `internal/plugin`) had
  to be migrated and deleted.
- Additional boilerplate to thread ports through application constructors.

## References

- Spec 009 Implementation Plan and Tasks.
- `internal/ports` package documentation.
- `cmd/streamy/plugins_import.go` for registry wiring.
