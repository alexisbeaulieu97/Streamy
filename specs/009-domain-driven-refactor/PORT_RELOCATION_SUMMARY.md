# Port Relocation Summary

**Date**: 2025-10-15  
**Branch**: `009-domain-driven-refactor`  
**Purpose**: Document the architectural decision to move port interfaces from domain packages to application boundary

---

## Motivation

The initial design placed port interfaces within domain packages (`internal/domain/pipeline/ports.go`, `internal/domain/plugin/ports.go`). This created a conceptual ambiguity:

**Problem**: While the domain layer had no implementation dependencies on infrastructure, it still **defined the contracts** that infrastructure must satisfy. This blurred the "truly pure domain" boundary.

**Peak Architecture Goal**: A pure domain should contain **only business logic** (entities, value objects, business rules) with **zero knowledge** of external contracts or infrastructure needs.

---

## Solution: Port Interfaces at Application Boundary

Port interfaces now live in `internal/ports/` as a separate package at the **application boundary**:

```
internal/
├── ports/                       # Port interfaces (application boundary)
│   ├── config.go                # ConfigLoader
│   ├── execution.go             # PluginExecutor, DAGBuilder, ExecutionPlanner
│   ├── logging.go               # Logger
│   ├── observability.go         # MetricsCollector, Tracer
│   ├── plugins.go               # Plugin, PluginRegistry
│   ├── events.go                # EventPublisher, EventHandler, DomainEvent
│   └── registry.go              # RegistryStore, ValidationService
│
├── domain/                      # Pure domain (zero port knowledge)
│   ├── pipeline/
│   │   ├── pipeline.go          # Domain entities only
│   │   ├── step.go
│   │   └── errors.go
│   └── plugin/
│       ├── plugin.go
│       └── metadata.go
│
├── application/                 # Uses domain + ports
│   ├── pipeline/
│   │   ├── apply_usecase.go
│   │   └── verify_usecase.go
│   └── registry/
│       └── add_usecase.go
│
└── infrastructure/              # Implements ports
    ├── config/
    │   └── yaml_loader.go       # implements ports.ConfigLoader
    ├── engine/
    │   └── executor.go          # implements ports.PluginExecutor
    └── logging/
        └── logger.go            # implements ports.Logger
```

---

## Dependency Flow

**Before** (ports in domain):
```
Infrastructure → implements → Domain Ports
Application    → uses      → Domain Entities + Domain Ports
Domain         → defines   → Entities + Ports (mixed concern)
```

**After** (ports at boundary):
```
Infrastructure → implements → Ports (application boundary)
Application    → uses      → Domain Entities & Ports
Ports          → depends on→ Domain (for types in contracts)
Domain         → contains  → Pure business logic (no port knowledge)
```

---

## Dependency Inversion Principle

This follows the **Dependency Inversion Principle** correctly:

1. **High-level policy** (Application) defines what it needs (Ports)
2. **Low-level details** (Infrastructure) implement those contracts
3. **Business core** (Domain) remains pure and unaware of external dependencies

The application layer **owns the contracts** it needs to orchestrate use cases. The domain layer provides the business rules those use cases operate on.

---

## Changes Made

### 1. File Structure Updates

**plan.md** (lines 132-165):
- Removed `internal/domain/pipeline/ports.go`
- Removed `internal/domain/plugin/ports.go`
- Removed `internal/application/pipeline/ports.go`
- Added `internal/ports/` directory with 7 files

**data-model.md** (lines 407-520):
- Changed section title from "Port Interfaces (Domain Layer)" to "Port Interfaces (Application Boundary)"
- Added explanation of port placement rationale
- Added directory structure diagram
- Updated all port definitions with `Location: internal/ports/<file>.go`

### 2. Contract Updates

**contracts/domain-ports.go** (lines 1-18):
- Updated package comment from "Domain Port Interfaces" to "Port Interfaces for Domain-Driven Architecture"
- Changed package path from `internal/domain/ports` to `internal/ports`
- Added explanation of port placement at application boundary
- Clarified architecture: Infrastructure → Application → Domain (with Ports at boundary)

**contracts/app-ports.go** (lines 1-18):
- Updated package comment to clarify all ports live in `internal/ports/`
- Changed package path from `internal/application/ports` to `internal/ports`
- Added port organization listing (config.go, execution.go, logging.go, etc.)

### 3. Task Updates

**tasks.md** (lines 136-158):
- **Section Rename**: "Port Interfaces (Domain Layer)" → "Port Interfaces (Application Boundary)"
- **T033**: Changed from "Create `internal/domain/pipeline/ports.go` with ConfigLoader" to "Create `internal/ports/` directory"
- **T034**: Changed to "Create `internal/ports/config.go` with ConfigLoader interface"
- **T035**: Changed to "Create `internal/ports/execution.go` with PluginExecutor, DAGBuilder, ExecutionPlanner"
- **T036**: Changed to "Create `internal/ports/logging.go` with Logger interface"
- **T037**: Changed to "Create `internal/ports/observability.go` with MetricsCollector and Tracer"
- **T038**: Changed to "Create `internal/ports/plugins.go` with Plugin and PluginRegistry"
- **T039**: Changed to "Create `internal/ports/events.go` with EventPublisher, EventHandler, DomainEvent"
- **T040**: Changed to "Create `internal/ports/registry.go` with RegistryStore and ValidationService"
- **Removed T043**: "Create `internal/domain/plugin/ports.go`" (consolidated into T038)
- **T044-T049**: Renumbered from T045-T050 (removed one task)
- **T047**: Updated to verify domain has zero imports from application/infrastructure **and** no knowledge of ports

### 4. Quickstart Guide Updates

**quickstart.md** (lines 17-52):
- Updated architecture diagram showing ports at boundary (not in domain)
- Added explanation: "Domain has zero dependencies on infrastructure, application, OR ports (truly pure)"
- Added new principle: "Ports are defined at **application boundary** (`internal/ports/`) not in domain"
- Updated package structure to show `internal/ports/` directory with 7 files
- Updated all code examples to import from `internal/ports` instead of `internal/domain/ports`

**quickstart.md** (line 262):
- Changed port reference from `internal/domain/ports/ports.go` to `internal/ports/config.go`

**quickstart.md** (line 753):
- Updated "Review port interfaces" guidance to mention contracts in specs and implementations in `internal/ports/`

---

## Validation

After port relocation, the following must hold true:

### Domain Purity Check
```bash
# Domain layer should have zero imports from ports, application, or infrastructure
go list -f '{{.Imports}}' ./internal/domain/...
# Should output only: context, time, fmt, errors, strings (stdlib only)
```

### Ports Dependency Check
```bash
# Ports should only depend on domain and stdlib
go list -f '{{.Imports}}' ./internal/ports/...
# Should only output domain and stdlib packages
```

### Application Layer Check
```bash
# Application should import both domain AND ports (not domain ports)
grep -r "internal/ports" internal/application/
grep -r "internal/domain" internal/application/
# Both should return results
```

### Infrastructure Layer Check
```bash
# Infrastructure should implement ports
grep -r "ports\." internal/infrastructure/
# Should show adapter structs implementing port interfaces
```

---

## Benefits of Port Relocation

1. **True Domain Purity**: Domain layer has **zero knowledge** of external contracts
2. **Clear Ownership**: Application layer owns the contracts it needs to orchestrate
3. **Dependency Inversion**: High-level policy (application) defines contracts, low-level details (infrastructure) implement them
4. **Testability**: Domain can be tested with **zero mocking** (pure business logic)
5. **Single Responsibility**: Domain focused solely on business rules, not infrastructure contracts
6. **Future-Proof**: Adding new infrastructure (webhooks, external services) doesn't touch domain
7. **Conceptual Clarity**: "Where do ports belong?" has a clear answer: application boundary

---

## References

- **Architecture Overview**: `docs/architecture-overview.md`
- **ADR**: `docs/adr/001-domain-driven-refactor.md`
- **User Feedback**: Original concern from architectural review 2025-10-15
  > "Keeping ports in the domain package blurs the intended 'pure core' boundary. Even in a maximal design, it's cleaner to define ports at the application boundary (e.g., `internal/ports` or `internal/application/ports`). Domain entities stay dependency-free, application orchestrators own the ports, and infrastructure implements them."

---

## Implementation Status

| Component | Status | Notes |
|-----------|--------|-------|
| Port interfaces (`internal/ports/`) | ✅ Complete | Contracts defined at application boundary |
| Config loader adapter (`internal/infrastructure/config/yaml_loader.go`) | ✅ Complete | Context-aware loading, detailed error mapping |
| Config loader tests (`internal/infrastructure/config/yaml_loader_test.go`) | ✅ Complete | Covers success, not found, parse errors, validation errors, cancellation |
| DAG builder adapter (`internal/infrastructure/engine/dag_builder.go`) | ✅ Complete | Builds level-based plans, handles cycles and missing deps |
| DAG builder tests (`internal/infrastructure/engine/dag_builder_test.go`) | ✅ Complete | Validates ordering, cycle detection, disabled steps, cancellation |

**Status**: Adapter implementations for YAML config loading and DAG building are now complete and verified with unit tests.
