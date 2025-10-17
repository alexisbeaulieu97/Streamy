# ADR-001: Domain-Driven Architecture Refactor

**Date**: 2025-10-15  
**Status**: Accepted  
**Related**: [Feature Spec](../../specs/009-domain-driven-refactor/spec.md) | [Architecture Overview](../architecture-overview.md)

## Context

Streamy's initial architecture emerged organically during rapid MVP development. The codebase exhibits common symptoms of organic growth:

1. **Mixed Concerns**: Domain entities (`Pipeline`, `Step`) co-located with infrastructure code (YAML parsing, file I/O)
2. **Circular Dependencies**: Domain layer imports infrastructure packages (`internal/domain/pipeline` → `internal/config`, `internal/engine`, `internal/logger`)
3. **Tight Coupling**: Domain service directly depends on concrete `*plugin.PluginRegistry` type rather than interfaces
4. **Testing Challenges**: Domain tests require infrastructure setup (file I/O, plugin loading) making them slow and brittle
5. **Limited Extensibility**: Adding new infrastructure implementations (e.g., JSON config loader, RPC-based plugins) requires modifying domain layer

### Specific Technical Debt

**`internal/domain/pipeline/service.go`**:
```go
import (
    "github.com/.../streamy/internal/config"      // ❌ Domain → Infrastructure
    "github.com/.../streamy/internal/engine"      // ❌ Domain → Infrastructure  
    "github.com/.../streamy/internal/logger"      // ❌ Domain → Infrastructure
    "github.com/.../streamy/internal/plugin"      // ❌ Domain → Infrastructure
)

type PipelineService struct {
    registry *plugin.PluginRegistry  // ❌ Concrete type, not interface
    logger   *logger.Logger           // ❌ Concrete type, not interface
}
```

**`internal/config/types.go`**:
```go
// MIXED: Domain entity + infrastructure parsing logic in same file
type Config struct {
    Version string `yaml:"version"`  // Infrastructure concern
    Steps   []Step                    // Domain entity
}
```

### Business Drivers

1. **Maintainability**: As features grow (imports, groups, conditionals), coupled architecture will become increasingly difficult to maintain
2. **Testability**: Slow integration-only tests hinder TDD and rapid iteration
3. **Extensibility**: Users should be able to add plugins and customize infrastructure without touching core business logic
4. **Team Scaling**: Clear boundaries enable parallel development without stepping on each other's toes

**Plugin Ecosystem Status**: As of 2025-10-15, **zero external plugins exist**. All 7 built-in plugins (package, repo, symlink, copy, command, template, lineinfile) are maintained in-tree under `internal/plugins/`. This internal-only ecosystem eliminates external compatibility constraints, allowing clean plugin interface redesign during the refactor. All plugins will migrate simultaneously in Phase 4 (US2). Pre-1.0 status (v0.x) explicitly allows breaking changes to plugin API.

## Decision

We will refactor Streamy into a **three-layer domain-driven architecture** with strict dependency direction: Infrastructure → Application → Domain.

### Architecture Pattern: Hexagonal Architecture (Ports & Adapters)

**Layer Responsibilities**:

1. **Domain Layer** (`internal/domain/`):
   - Pure business logic: entities, validation rules, business invariants
   - Port interfaces defining infrastructure needs
   - Zero dependencies on frameworks, libraries, or other layers
   - Testable in isolation with zero infrastructure setup

2. **Application Layer** (`internal/application/`):
   - Use case orchestration: ApplyPipeline, VerifyPipeline, ValidateConfiguration
   - Coordinates domain entities through port interfaces
   - Manages transactions, error aggregation, event emission
   - Depends only on domain (entities + ports)

3. **Infrastructure Layer** (`internal/infrastructure/`):
   - Adapter implementations for all ports (ConfigLoader, PluginExecutor, Logger)
   - External system integrations (file I/O, external processes, logging libraries)
   - TUI/CLI frameworks (Bubbletea, Cobra)
   - Implements port interfaces defined in domain/application

**Dependency Injection**:
- Manual constructor injection (no frameworks like `wire` or `dig`)
- All wiring in single composition root (`cmd/streamy/main.go`)
- Interfaces as constructor parameters, never concrete types
- Compile-time type safety

**Migration Strategy**:
- Strangler Pattern: new architecture alongside old, validate outputs match, migrate incrementally
- Layer-by-layer: Domain → Application → Infrastructure → Wiring → Cleanup
- Validation after each phase: all integration tests must pass unchanged

### Technology Choices

1. **Logging**: Migrate from `zerolog` to `charmbracelet/log`
   - Better alignment with existing Charm stack (Bubbletea, Lipgloss)
   - Structured logging with clean API
   - Adapter wrapper for smooth migration

2. **Context Propagation**: `context.Context` as first parameter in all methods
   - Enables cancellation, timeouts, correlation IDs
   - Industry-standard Go pattern

3. **Error Handling**: Typed domain errors with context preservation
   - `DomainError` with error codes, messages, cause chain, metadata
   - Wrapping at each layer to add context
   - User-friendly messages at CLI layer

## Consequences

### Positive

1. **Isolation & Testability**
   - Domain tests run in <100ms without infrastructure
   - Application tests use mocks (90%+ coverage achievable)
   - Infrastructure tests use real or test I/O
   - Integration tests validate end-to-end behavior

2. **Flexibility**
   - Swap infrastructure implementations without touching domain/application
   - Add new plugins by implementing adapters
   - Support multiple config formats (YAML, JSON, TOML) via different adapters
   - Enable future features: RPC-based plugins, database-backed registry, cloud storage

3. **Maintainability**
   - Clear boundaries: obvious where code belongs
   - Unidirectional dependencies prevent circular issues
   - Compile-time type safety catches wiring errors early
   - Documentation aligns with code structure

4. **Team Scaling**
   - Parallel development on different layers
   - New developers understand architecture by reading domain first
   - Code reviews focus on appropriate layer concerns

5. **Compliance with Constitution**
   - Enhances Principle VI (Extensibility & Composability)
   - Maintains Principle IV (Safety by Default) with improved context cancellation
   - Supports Principle V (Performance & Reliability) with faster tests

### Negative

1. **Initial Complexity**
   - More files and packages (domain/, application/, infrastructure/)
   - Indirection through port interfaces
   - Learning curve for developers unfamiliar with DDD

2. **Upfront Investment**
   - Significant refactoring effort (5-7 phases)
   - Risk of regressions during migration
   - Requires comprehensive test coverage to validate behavior preservation

3. **Verbosity**
   - Constructor injection can be verbose for services with many dependencies
   - Port interfaces add boilerplate (mitigated by clear benefits)

4. **Performance (Negligible)**
   - Interface calls have minimal overhead (<5% per Go benchmarks)
   - Context propagation adds ~nanoseconds per call
   - Overall performance impact: negligible for typical workloads (10-500 steps)

### Risk Mitigation

1. **Large Scope**: Break into phases with clear validation points
2. **Regressions**: Strangler pattern with output comparison, all integration tests must pass
3. **Team Adoption**: Provide quickstart guide, pair programming, code review emphasis
4. **Performance**: Benchmark after each phase, enforce SC-008 (build <10s, tests <20% slower)

## Alternatives Considered

### Alternative 1: Keep Current Architecture

**Pros**:
- No refactoring effort
- No risk of regressions
- Team already familiar

**Cons**:
- Technical debt compounds with each feature
- Testing remains difficult and slow
- Extensibility limited by tight coupling
- Violates DDD principles

**Rejected**: Current architecture is viable for MVP but will hinder growth as features increase.

### Alternative 2: Clean Architecture (Uncle Bob)

**Pros**:
- Similar benefits to Hexagonal Architecture
- Well-documented pattern
- Strong community support

**Cons**:
- More layers (4-5 vs 3)
- Use case classes can be overkill for simple operations
- Boundary interfaces can proliferate

**Rejected**: Hexagonal Architecture achieves same goals with fewer layers, better fit for Streamy's complexity level.

### Alternative 3: Traditional Layered Architecture

```
Presentation Layer
    ↓
Business Logic Layer
    ↓
Data Access Layer
```

**Pros**:
- Simpler mental model
- Widely understood pattern

**Cons**:
- Allows business logic to depend on data access layer
- Harder to test business logic in isolation
- Doesn't emphasize port/adapter pattern

**Rejected**: Doesn't achieve sufficient isolation between business logic and infrastructure.

### Alternative 4: Dependency Injection Framework (wire, dig)

**Pros**:
- Less boilerplate in wiring code
- Automatic dependency graph construction
- Compile-time generation (wire) or runtime (dig)

**Cons**:
- Added complexity and learning curve
- Magic/implicit behavior (especially dig's runtime reflection)
- Overkill for ~20 dependencies
- Constitution Principle I (Onboarding First) favors zero external dependencies

**Rejected**: Manual constructor injection is sufficient for Streamy's scale. Simple, explicit, and maintainable.

## Implementation Plan

See [plan.md](../../specs/009-domain-driven-refactor/plan.md) for detailed phase breakdown:

1. **Phase 0**: Research & baseline (architecture patterns, DI strategy, logging migration)
2. **Phase 1**: Design & contracts (data model, port interfaces, quickstart guide)
3. **Phase 2**: Task planning (generate detailed implementation tasks)
4. **Phase 3**: Domain layer (entities, ports, pure business logic)
5. **Phase 4**: Application layer (use cases with mock-based tests)
6. **Phase 5**: Infrastructure adapters (config, executor, logging, metrics)
7. **Phase 6**: Wiring & integration (DI in main.go, CLI updates)
8. **Phase 7**: Migration & cleanup (strangler completion, legacy removal)

## Validation

### Success Criteria (from spec.md)

- **SC-001**: Domain tests run in <100ms without infrastructure setup ✅
- **SC-002**: Application tests achieve 90%+ coverage with mocks ✅
- **SC-003**: DI is compile-time type-checked (no interface{} or reflection) ✅
- **SC-004**: 95%+ of logs use charmbracelet/log structured format ✅
- **SC-005**: Context cancellation causes graceful shutdown within 5 seconds ✅
- **SC-006**: Add plugins without modifying domain/application layers ✅
- **SC-007**: All existing integration tests pass unchanged ✅ (critical)
- **SC-008**: Build <10s, test suite <20% slower ✅
- **SC-009**: Error messages include full context chain ✅
- **SC-010**: Architecture understandable by reading domain first ✅

### Constitution Check

All 7 constitution principles pass (see plan.md for detailed analysis).

### Rollback Plan

If any phase fails validation:
1. Keep legacy code intact until new code proven
2. Revert specific phase if integration tests fail
3. Strangler pattern enables incremental rollback
4. Maximum risk contained to single phase

## References

- **Hexagonal Architecture**: Alistair Cockburn, "Ports and Adapters" (2005)
- **Domain-Driven Design**: Eric Evans, "Domain-Driven Design" (2003)
- **Go Package Layout**: Ben Johnson, "Standard Package Layout" (2017)
- **Go Dependency Injection**: Kat Zien, "How I Structure Go Apps" (2018)
- **charmbracelet/log**: https://github.com/charmbracelet/log

## Timeline

- **Start Date**: 2025-10-15
- **Phase 0-1**: ~1-2 weeks (research + design) ✅ Complete
- **Phase 3-5**: ~3-4 weeks (implementation)
- **Phase 6-7**: ~1-2 weeks (integration + cleanup)
- **Total Estimate**: 5-8 weeks for complete migration

## Approval

- **Proposed by**: Development team
- **Reviewed by**: Architecture review (implicit in feature specification approval)
- **Approved by**: Product owner (implicit in feature prioritization)
- **Date**: 2025-10-15
