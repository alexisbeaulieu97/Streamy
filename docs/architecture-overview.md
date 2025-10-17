# Streamy Architecture Overview

**Last Updated**: 2025-10-15  
**Related**: [ADR-001: Domain-Driven Refactor](adr/001-domain-driven-refactor.md) | [Feature Spec](../specs/009-domain-driven-refactor/spec.md)

## Purpose

This document describes Streamy's domain-driven architecture following the refactoring initiated in feature `009-domain-driven-refactor`. It serves as the north-star reference for understanding layer responsibilities, dependency flow, and design principles.

**Implementation Status**

| Layer | Status | Notes |
|-------|--------|-------|
| Domain | COMPLETE (2025-10-16) | Entities, value objects, and validation rules extracted into `internal/domain` with zero infrastructure imports. |
| Ports | COMPLETE (2025-10-16) | Port interfaces relocated to `internal/ports`, maintaining a pure domain and preventing import cycles. |

## Architecture Principles

### 1. Unidirectional Dependency Flow

```
Infrastructure Layer
    ↓ (implements)
Application Layer
    ↓ (uses)
Domain Layer
```

**Rules**:
- Domain has **zero dependencies** on application or infrastructure
- Application depends **only** on domain (entities + port interfaces)
- Infrastructure implements port interfaces defined in domain/application
- No circular dependencies between layers

**Enforcement**:
- CI checks package import graphs
- Code review validates new imports
- Static analysis tools (e.g., `go mod graph`, custom scripts)

### 2. Port and Adapter Pattern

**Ports** (interfaces) are defined in the layer that **uses** them:
- Domain layer defines ports for infrastructure it needs (ConfigLoader, PluginExecutor, Logger)
- Application layer defines ports for additional services (RegistryStore, ValidationService)

**Adapters** (implementations) live in the infrastructure layer:
- `infrastructure/config/yaml_loader.go` implements `domain/ports.ConfigLoader`
- `infrastructure/engine/executor.go` implements `domain/ports.PluginExecutor`
- `infrastructure/logging/logger.go` implements `domain/ports.Logger`

**Benefits**:
- Domain logic isolated from implementation details
- Infrastructure components swappable without domain changes
- Testing uses mock implementations of ports

### 3. Explicit Dependency Injection

All wiring happens in the **composition root** (`cmd/streamy/main.go`):

```go
// Create infrastructure adapters
logger := logging.NewLogger(logConfig)
configLoader := config.NewYAMLLoader(yamlConfig, logger)
pluginExecutor := engine.NewExecutor(executorConfig, logger)

// Wire application use cases
applyUseCase := pipeline.NewApplyUseCase(
    configLoader,
    pluginExecutor,
    logger,
)

// Wire CLI commands
applyCmd := cmd.NewApplyCommand(applyUseCase, logger)
```

**Rules**:
- Constructor injection with interface parameters
- No global variables for dependencies
- No service locator pattern
- No reflection-based DI frameworks

---

## Layer Responsibilities

### Domain Layer (`internal/domain/`)

**Purpose**: Pure business logic with zero infrastructure dependencies.

**Responsibilities**:
- Define core entities (Pipeline, Step, ExecutionPlan, StepResult)
- Implement business rules and invariants (validation, state transitions)
- Define port interfaces for infrastructure needs
- Represent domain errors with business context

**Rules**:
- ✅ Can import: stdlib, domain packages
- ❌ Cannot import: application, infrastructure, external frameworks
- ✅ All entities are serializable (for testing/state management)
- ✅ All methods are context-aware (first parameter: `context.Context`)

**Example Entity** (`domain/pipeline/pipeline.go`):
```go
type Pipeline struct {
    Name       string
    Version    string
    Steps      []Step
    Validations []Validation
    Settings   Settings
}

// Validate checks business invariants
func (p *Pipeline) Validate() error {
    if err := p.validateUniqueStepIDs(); err != nil {
        return err
    }
    if err := p.validateDependencies(); err != nil {
        return err
    }
    return nil
}
```

**Example Port** (`domain/ports/ports.go`):
```go
type ConfigLoader interface {
    Load(ctx context.Context, path string) (*pipeline.Pipeline, error)
    Validate(ctx context.Context, path string) error
}
```

### Application Layer (`internal/application/`)

**Purpose**: Orchestrate use cases and coordinate domain entities.

**Responsibilities**:
- Implement use cases (ApplyPipeline, VerifyPipeline, PreparePipeline)
- Coordinate multiple domain operations
- Manage transactions and error aggregation
- Translate domain errors into user-friendly messages
- Emit structured events at key workflow points

**Rules**:
- ✅ Can import: domain packages (entities + ports)
- ❌ Cannot import: infrastructure (only interfaces, not implementations)
- ✅ All services accept port interfaces via constructor injection
- ✅ Context passed through all operations

**Example Use Case** (`application/pipeline/apply_usecase.go`):
```go
type ApplyUseCase struct {
    loader    ports.ConfigLoader
    builder   ports.DAGBuilder
    planner   ports.ExecutionPlanner
    executor  ports.PluginExecutor
    validator ports.ValidationService
    logger    ports.Logger
    metrics   ports.MetricsCollector
}

func (u *ApplyUseCase) Apply(ctx context.Context, configPath string, dryRun bool) error {
    // Load configuration
    pip, err := u.loader.Load(ctx, configPath)
    if err != nil {
        return fmt.Errorf("load config: %w", err)
    }
    
    // Build execution plan
    plan, err := u.buildExecutionPlan(ctx, pip)
    if err != nil {
        return fmt.Errorf("build plan: %w", err)
    }
    
    // Execute (with dry-run support)
    results, err := u.executor.Execute(ctx, plan, dryRun)
    if err != nil {
        return fmt.Errorf("execute plan: %w", err)
    }
    
    // Validate results
    return u.validator.RunValidations(ctx, pip, results)
}
```

### Infrastructure Layer (`internal/infrastructure/`)

**Purpose**: Implement adapters for all ports and handle external systems.

**Responsibilities**:
- Implement port interfaces (ConfigLoader, PluginExecutor, Logger, etc.)
- Handle file I/O, network calls, external processes
- Manage context cancellation and resource cleanup
- Translate infrastructure errors into domain errors where appropriate

**Rules**:
- ✅ Can import: domain (to implement ports), application (for app-level ports), external libraries
- ❌ Infrastructure components should not directly depend on each other (use ports)
- ✅ Each adapter has clear lifecycle (construction, operation, cleanup)
- ✅ Context cancellation honored in all blocking operations

**Example Adapter** (`infrastructure/config/yaml_loader.go`):
```go
type YAMLLoader struct {
    config YAMLLoaderConfig
    logger ports.Logger
}

func NewYAMLLoader(config YAMLLoaderConfig, logger ports.Logger) *YAMLLoader {
    return &YAMLLoader{
        config: config,
        logger: logger,
    }
}

func (l *YAMLLoader) Load(ctx context.Context, path string) (*pipeline.Pipeline, error) {
    l.logger.Debug(ctx, "loading YAML config", "path", path)
    
    // Check context before expensive operation
    if err := ctx.Err(); err != nil {
        return nil, &pipeline.DomainError{
            Code:    pipeline.ErrCodeCancelled,
            Message: "load cancelled",
            Cause:   err,
        }
    }
    
    // Read and parse YAML...
    // Validate domain invariants...
    
    return pip, nil
}
```

---

## Dependency Graph

### Current State (Before Refactor)

```
┌─────────────────────────────────────────────────────┐
│                   cmd/streamy                       │
│                   (main.go)                         │
└──────────────┬─────────────────────────────────────┘
               │
               ├─→ internal/domain/pipeline ────┐
               │         ↓ (violates DDD)        │
               │         ↓                        │
               ├─→ internal/config ──────────────┤
               │         ↓                        │
               ├─→ internal/engine ──────────────┤
               │         ↓                        │
               ├─→ internal/logger ──────────────┤
               │         ↓                        │
               └─→ internal/plugin               │
                         ↑                        │
                         └────────────────────────┘
                    (Circular dependencies!)
```

**Problems**:
- Domain imports infrastructure (config, engine, logger, plugin)
- Circular dependencies between packages
- Impossible to test domain without infrastructure
- Tightly coupled - can't swap implementations

### Target State (After Refactor)

```
┌─────────────────────────────────────────────────────┐
│                   cmd/streamy                       │
│                   (Composition Root)                │
│                                                     │
│  Creates all adapters, wires use cases, runs CLI   │
└──┬────────────────┬────────────────┬───────────────┘
   │                │                │
   │                │                └────────────────┐
   ↓                ↓                                 ↓
┌──────────────────────────┐            ┌──────────────────────────┐
│  Application Layer       │            │  Infrastructure Layer    │
│  internal/application/   │            │  internal/infrastructure/│
│                          │            │                          │
│  • Use cases             │◄───────────┤  • Config adapters       │
│  • Orchestration         │ implements │  • Plugin executor       │
│  • Error translation     │   ports    │  • Logger (charmbracelet)│
│  • Event emission        │            │  • Metrics, tracing      │
└──────────┬───────────────┘            │  • TUI, CLI handlers     │
           │                            └──────────────────────────┘
           │ depends on                              │
           ↓                                         │ implements
┌──────────────────────────┐                        │
│  Domain Layer            │◄───────────────────────┘
│  internal/domain/        │
│                          │
│  • Entities (pure logic) │
│  • Port interfaces       │
│  • Business rules        │
│  • Domain errors         │
└──────────────────────────┘
   Zero dependencies! ✅
```

**Benefits**:
- Domain completely isolated - testable without infrastructure
- Unidirectional dependencies (Infrastructure → Application → Domain)
- Infrastructure swappable via port interfaces
- Clear boundaries for where code belongs

---

## Context Propagation

All operations accept `context.Context` as the first parameter to enable:

1. **Cancellation**: Graceful shutdown on Ctrl+C or timeout
2. **Correlation IDs**: Track requests across layers
3. **Deadlines**: Enforce operation timeouts
4. **Request-scoped values**: Carry metadata (user ID, trace spans)

**Pattern**:
```go
func (s *Service) Operation(ctx context.Context, params Parameters) error {
    // Check cancellation before expensive work
    if err := ctx.Err(); err != nil {
        return &DomainError{Code: ErrCodeCancelled, Cause: err}
    }
    
    // Pass context to dependencies
    result, err := s.dependency.SubOperation(ctx, ...)
    
    // Check cancellation between steps
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    return nil
}
```

**Correlation ID Pattern**:
```go
// In main.go (entry point)
correlationID := uuid.New().String()
ctx := context.WithValue(ctx, correlationIDKey, correlationID)

// In logger adapter
func (l *Logger) Info(ctx context.Context, msg string, fields ...interface{}) {
    correlationID := ctx.Value(correlationIDKey)
    l.impl.Info().
        Str("correlation_id", correlationID).
        Fields(fields).
        Msg(msg)
}
```

---

## Error Handling

### Error Flow

```
Infrastructure Layer
  ↓ (wraps with technical context)
Application Layer  
  ↓ (wraps with user guidance)
Domain Layer
  ↓ (typed domain errors)
CLI/TUI
  ↓ (formats for display)
User
```

### Domain Errors

Domain defines typed error codes for business failures:

```go
type ErrorCode string

const (
    ErrCodeValidation    ErrorCode = "VALIDATION_ERROR"
    ErrCodeNotFound      ErrorCode = "NOT_FOUND"
    ErrCodeDependency    ErrorCode = "DEPENDENCY_ERROR"
    ErrCodeCancelled     ErrorCode = "CANCELLED"
    ErrCodeTimeout       ErrorCode = "TIMEOUT"
)

type DomainError struct {
    Code    ErrorCode
    Message string
    Cause   error
    Context map[string]interface{}
}
```

### Error Wrapping Example

```go
// Domain layer: Typed error with business context
if stepID not found {
    return &DomainError{
        Code:    ErrCodeNotFound,
        Message: "step not found",
        Context: map[string]interface{}{
            "step_id": stepID,
            "pipeline": pip.Name,
        },
    }
}

// Application layer: Add user guidance
if err != nil {
    return fmt.Errorf("failed to prepare pipeline: %w. Check that all step dependencies exist in the config", err)
}

// CLI layer: Format for display
if err != nil {
    fmt.Fprintf(os.Stderr, "Error: %v\n", err)
    // Optionally extract DomainError and show structured details
}
```

---

## Testing Strategy

### Domain Layer Tests

**Goal**: Test business logic in isolation (<100ms for entire domain layer).

**Approach**:
- No mocks needed - pure unit tests
- Test entities, validation rules, state transitions
- Use table-driven tests for comprehensive coverage

**Example**:
```go
func TestPipeline_Validate_UniqueIDs(t *testing.T) {
    pip := &Pipeline{
        Steps: []Step{
            {ID: "step1", Type: "command"},
            {ID: "step1", Type: "command"}, // Duplicate!
        },
    }
    
    err := pip.Validate()
    
    if err == nil {
        t.Fatal("expected validation error")
    }
}
```

### Application Layer Tests

**Goal**: Test use case orchestration with mock dependencies (90%+ coverage).

**Approach**:
- Mock all port interfaces
- Test error handling, transaction coordination, event emission
- Verify correct interaction with dependencies

**Example**:
```go
func TestApplyUseCase_LoadFailure(t *testing.T) {
    mockLoader := &MockConfigLoader{
        LoadError: errors.New("file not found"),
    }
    mockLogger := &MockLogger{}
    
    useCase := NewApplyUseCase(mockLoader, /* ... */, mockLogger)
    
    err := useCase.Apply(context.Background(), "/fake/path.yaml", false)
    
    if err == nil {
        t.Fatal("expected error")
    }
    if mockLogger.ErrorCallCount != 1 {
        t.Errorf("expected Error called once, got %d", mockLogger.ErrorCallCount)
    }
}
```

### Infrastructure Layer Tests

**Goal**: Test adapter implementations with real or test I/O.

**Approach**:
- Mix of real file I/O (using `t.TempDir()`) and mocked external services
- Test adapter behavior, error handling, context cancellation
- Slower than domain/app tests (~50-100ms per test)

**Example**:
```go
func TestYAMLLoader_Load_RealFile(t *testing.T) {
    tmpDir := t.TempDir()
    configPath := filepath.Join(tmpDir, "config.yaml")
    yaml := `version: "1.0"\nname: test\nsteps:\n  - id: step1\n    type: command`
    os.WriteFile(configPath, []byte(yaml), 0644)
    
    loader := NewYAMLLoader(YAMLLoaderConfig{}, logger)
    
    pipeline, err := loader.Load(context.Background(), configPath)
    
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if pipeline.Name != "test" {
        t.Errorf("expected name 'test', got %s", pipeline.Name)
    }
}
```

### Integration Tests

**Goal**: Validate entire system with real adapters end-to-end.

**Approach**:
- Real implementations (no mocks)
- Test complete user workflows
- Validate SC-007 (all integration tests pass unchanged)
- Slower (1-10s per test)

**Example**:
```go
func TestApplyPipeline_EndToEnd(t *testing.T) {
    // Real adapters
    logger := logging.NewLogger(logging.Config{Level: "error"})
    loader := config.NewYAMLLoader(config.YAMLLoaderConfig{}, logger)
    // ... create all real adapters ...
    
    useCase := pipeline.NewApplyUseCase(loader, executor, logger, metrics)
    
    err := useCase.Apply(context.Background(), "testdata/configs/simple.yaml", false)
    
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    // Verify side effects...
}
```

---

## Migration Strategy (Strangler Pattern)

### Phase 3: Domain Layer
1. Create new `internal/domain/` packages
2. Extract entities from `internal/config` and `internal/model`
3. Define port interfaces
4. Test domain in isolation (SC-001: <100ms)

### Phase 4: Application Layer
1. Create `internal/application/` packages
2. Implement use cases using domain entities + port interfaces
3. Test with mock implementations (SC-002: 90%+ coverage)

### Phase 5: Infrastructure Adapters
1. Create `internal/infrastructure/` packages
2. Implement adapters for all ports
3. Migrate from zerolog to charmbracelet/log
4. Test adapters with real I/O

### Phase 6: Wiring & Integration
1. Wire dependencies in `cmd/streamy/main.go`
2. Update CLI commands to use new use cases
3. Run integration tests (SC-007: all pass)
4. Compare outputs with legacy implementation (strangler validation)

### Phase 7: Cleanup
1. Remove legacy packages once strangler validation passes
2. Update documentation
3. Final benchmarking (SC-008)

---

## References

- **Specification**: [spec.md](../specs/009-domain-driven-refactor/spec.md)
- **Implementation Plan**: [plan.md](../specs/009-domain-driven-refactor/plan.md)
- **Research Findings**: [research.md](../specs/009-domain-driven-refactor/research.md)
- **Data Model**: [data-model.md](../specs/009-domain-driven-refactor/data-model.md)
- **Quickstart Guide**: [quickstart.md](../specs/009-domain-driven-refactor/quickstart.md)
- **ADR**: [ADR-001: Domain-Driven Refactor](adr/001-domain-driven-refactor.md)
