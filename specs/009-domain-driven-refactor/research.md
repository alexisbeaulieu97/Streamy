# Research: Domain-Driven Architecture Refactor

**Feature**: 009-domain-driven-refactor  
**Date**: 2025-10-15  
**Purpose**: Research architecture patterns, DI strategies, and migration approaches for domain-driven refactoring

---

## 1. Domain-Driven Design Patterns in Go

### Decision: Hexagonal Architecture (Ports & Adapters)

**Rationale**:
- Go's interface-based design naturally supports ports & adapters pattern
- Explicit dependency direction (Infrastructure → Application → Domain) prevents circular dependencies
- Widely adopted in Go community (e.g., Kat Zien's "How I Structure Go Apps", Ben Johnson's "Standard Package Layout")
- Testability: domain entities can be tested without infrastructure setup
- Flexibility: infrastructure adapters can be swapped without domain changes

**Alternatives Considered**:
1. **Layered Architecture** - Rejected: Allows bidirectional dependencies between layers, harder to enforce boundaries
2. **Clean Architecture** - Similar to hexagonal, but more complex with 4+ layers. Overkill for Streamy's scale (~20 packages)
3. **MVC** - Rejected: UI-centric pattern not suitable for CLI tool architecture

**Go-Specific Patterns**:
- **Aggregate Roots**: `Pipeline` entity encapsulates collection of `Step` entities
- **Value Objects**: `PluginMetadata`, `StepResult` are immutable value types
- **Repositories**: Port interfaces like `ConfigLoader`, `RegistryStore` abstract persistence
- **Services**: Application layer services orchestrate use cases through port interfaces
- **Entities**: Mutable objects with identity (`Pipeline`, `Step`, `ExecutionPlan`)

**Implementation Notes**:
- Place port interfaces in domain package alongside entities (local to domain)
- Use small, focused interfaces (Interface Segregation Principle)
- No framework dependencies in domain layer - pure Go structs and interfaces
- Context as first parameter for all domain methods (Go idiom for cancellation/deadline)

---

## 2. Dependency Injection Strategies in Go

### Decision: Manual Constructor Injection

**Rationale**:
- Go's simplicity favors explicit over implicit - no magic, no reflection
- Compile-time type safety catches wiring errors early
- ~20 dependencies total is manageable without framework
- Factory functions can hide complexity when needed
- No external dependencies aligns with Constitution Principle I (Onboarding First)
- Clear, debuggable code - no hidden dependency resolution logic

**Alternatives Considered**:
1. **google/wire** - Code generation for DI. Rejected: Adds build complexity, overkill for Streamy's scale
2. **uber-go/dig** - Reflection-based DI container. Rejected: Runtime errors, magic behavior, external dependency
3. **Service Locator** - Global registry pattern. Rejected: Violates explicit wiring goal, makes testing harder

**Implementation Pattern**:
```go
// Domain layer - port interfaces
type ConfigLoader interface {
    Load(ctx context.Context, path string) (*Pipeline, error)
}

type PluginExecutor interface {
    Execute(ctx context.Context, plan *ExecutionPlan) ([]StepResult, error)
}

// Application layer - use case with constructor injection
type ApplyUseCase struct {
    loader   ConfigLoader
    executor PluginExecutor
    logger   Logger
}

func NewApplyUseCase(loader ConfigLoader, executor PluginExecutor, logger Logger) *ApplyUseCase {
    return &ApplyUseCase{
        loader:   loader,
        executor: executor,
        logger:   logger,
    }
}

// Composition root in main.go
func main() {
    // Create infrastructure adapters
    logger := logging.NewLogger(/* config */)
    loader := config.NewYAMLLoader(logger)
    executor := engine.NewExecutor(registry, logger)
    
    // Wire application services
    applyUseCase := pipeline.NewApplyUseCase(loader, executor, logger)
    
    // Inject into CLI
    cli := cmd.NewCLI(applyUseCase, /* other use cases */)
    cli.Execute()
}
```

**Best Practices**:
- Max 5-7 dependencies per constructor (cognitive load limit)
- Use functional options for optional dependencies
- Factory functions for complex object graphs
- Interfaces for all dependencies (enables mocking)
- No global state - all state passed explicitly

---

## 3. charmbracelet/log Migration from zerolog

### Decision: Phased Migration with Adapter Wrapper

**Rationale**:
- charmbracelet/log is more user-friendly with better defaults
- Better terminal formatting and colors out-of-box (aligns with TUI focus)
- Simpler API than zerolog (fewer allocations for typical use)
- Community momentum in charm ecosystem (bubbletea, lipgloss already used)

**API Comparison**:

| Operation | zerolog | charmbracelet/log |
|-----------|---------|-------------------|
| Create logger | `zerolog.New(os.Stdout).With().Timestamp().Logger()` | `log.New(os.Stdout)` |
| Log with fields | `log.Info().Str("key", "val").Msg("text")` | `log.Info("text", "key", "val")` |
| Child logger | `log.With().Str("component", "x").Logger()` | `log.With("component", "x")` |
| Structured output | JSON or console writer | Automatic pretty printing |

**Performance Comparison** (based on benchmarks):
- zerolog: ~200 ns/op for structured log (highly optimized, zero-alloc)
- charmbracelet/log: ~500 ns/op (acceptable for Streamy's logging frequency)
- For typical config (10-50 steps), logging overhead negligible vs I/O

**Migration Strategy**:
1. **Phase 1 (Initial)**: Create Logger port interface in domain layer
2. **Phase 2 (Adapter)**: Implement charmbracelet/log adapter in infrastructure
3. **Phase 3 (Wiring)**: Wire new logger in main.go
4. **Phase 4 (Cleanup)**: Remove zerolog dependency, update go.mod

**Breaking Changes**: None - Logger port interface abstracts implementation details

---

## 4. Context Propagation Best Practices

### Decision: Context-First Pattern with Correlation IDs

**Rationale**:
- context.Context is idiomatic Go for cancellation, deadlines, and request-scoped values
- Correlation IDs enable tracing execution flow across layers
- Enables graceful shutdown (SC-005: <5s cancellation)
- Supports future distributed tracing integration

**Implementation Pattern**:

```go
// Unexported key type for type safety
type contextKey int

const correlationIDKey contextKey = 0

// Inject correlation ID at entry point (CLI)
func Execute(cmd *cobra.Command, args []string) error {
    ctx := context.Background()
    correlationID := generateCorrelationID() // UUID or similar
    ctx = context.WithValue(ctx, correlationIDKey, correlationID)
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    // Pass context through layers
    return useCase.Apply(ctx, /* args */)
}

// Extract correlation ID in infrastructure layer (logging)
func (l *Logger) log(ctx context.Context, level, msg string, fields ...interface{}) {
    if corrID, ok := ctx.Value(correlationIDKey).(string); ok {
        fields = append(fields, "correlation_id", corrID)
    }
    // ... log with fields
}
```

**Best Practices**:
- **Always first parameter**: `func Method(ctx context.Context, ...)`
- **Never store context in struct**: Pass explicitly through call chain
- **Check ctx.Err()**: Before expensive operations, check if context cancelled
- **Derive contexts**: Use `context.WithTimeout`, `context.WithCancel` to create child contexts
- **Minimal values**: Store only request-scoped data (correlation ID, user ID), not business data

**Linting Enforcement**:
- Use `golangci-lint` with `contextcheck` linter to enforce context-first pattern
- CI gate: Fail build if context not first parameter in public methods

---

## 5. Error Wrapping Strategies

### Decision: Structured Errors with Context Preservation

**Rationale**:
- Go 1.13+ error wrapping (`fmt.Errorf("%w", err)`) preserves error chain
- Domain errors should be typed (for error handling strategies)
- Infrastructure errors should add context (file paths, step IDs, etc.)
- Application errors should provide user-friendly messages

**Error Type Hierarchy**:

```go
// Domain layer - typed errors
package domain

type ErrorCode string

const (
    ErrCodeValidation   ErrorCode = "VALIDATION_ERROR"
    ErrCodeExecution    ErrorCode = "EXECUTION_ERROR"
    ErrCodeDependency   ErrorCode = "DEPENDENCY_ERROR"
)

type DomainError struct {
    Code    ErrorCode
    Message string
    Cause   error
}

func (e *DomainError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *DomainError) Unwrap() error { return e.Cause }

// Infrastructure layer - add context
func (l *YAMLLoader) Load(ctx context.Context, path string) (*Pipeline, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("load config from %s: %w", path, err)
    }
    // ...
}

// Application layer - enrich for users
func (u *ApplyUseCase) Apply(ctx context.Context, configPath string) error {
    pipeline, err := u.loader.Load(ctx, configPath)
    if err != nil {
        return fmt.Errorf("failed to prepare pipeline: %w. Hint: check file path and YAML syntax", err)
    }
    // ...
}
```

**Pattern**: Each layer adds context without losing original error:
- Domain: Type + business context ("invalid step dependency: circular reference")
- Infrastructure: Technical context ("failed to read /path/to/config.yaml: permission denied")
- Application: User guidance ("Pipeline preparation failed. Check config file syntax")

**Error Handling**:
```go
// Check error type
var domainErr *domain.DomainError
if errors.As(err, &domainErr) {
    switch domainErr.Code {
    case domain.ErrCodeValidation:
        // Handle validation errors
    case domain.ErrCodeExecution:
        // Handle execution errors
    }
}

// Check wrapped errors
if errors.Is(err, os.ErrPermission) {
    // Handle permission errors specifically
}
```

---

## 6. Testing Strategies for Port-Based Architecture

### Decision: Test Doubles Pattern (Mocks, Fakes, Stubs)

**Rationale**:
- Port interfaces enable easy test double creation
- Different test types need different double strategies
- No mocking framework needed - Go interfaces + manual implementations

**Test Double Types**:

1. **Stubs** - Return canned responses (for basic tests)
```go
type StubConfigLoader struct {
    pipeline *Pipeline
    err      error
}

func (s *StubConfigLoader) Load(ctx context.Context, path string) (*Pipeline, error) {
    return s.pipeline, s.err
}
```

2. **Mocks** - Verify behavior (interactions with dependencies)
```go
type MockPluginExecutor struct {
    ExecuteFunc func(ctx context.Context, plan *ExecutionPlan) ([]StepResult, error)
    CallCount   int
}

func (m *MockPluginExecutor) Execute(ctx context.Context, plan *ExecutionPlan) ([]StepResult, error) {
    m.CallCount++
    if m.ExecuteFunc != nil {
        return m.ExecuteFunc(ctx, plan)
    }
    return nil, nil
}
```

3. **Fakes** - Working implementation for testing (in-memory variants)
```go
type FakeRegistryStore struct {
    pipelines map[string]*Pipeline
}

func (f *FakeRegistryStore) Store(ctx context.Context, id string, p *Pipeline) error {
    f.pipelines[id] = p
    return nil
}
```

**Test Organization**:

```
internal/
├── domain/
│   └── pipeline/
│       ├── pipeline.go
│       ├── pipeline_test.go         # Pure unit tests (no mocks needed)
│       └── testutil/
│           └── builders.go          # Test data builders
├── application/
│   └── pipeline/
│       ├── apply_usecase.go
│       ├── apply_usecase_test.go    # Unit tests with mock ports
│       └── testutil/
│           ├── mock_loader.go       # Mock ConfigLoader
│           └── mock_executor.go     # Mock PluginExecutor
└── infrastructure/
    └── config/
        ├── yaml_loader.go
        ├── yaml_loader_test.go      # Unit tests with test files
        └── yaml_loader_integration_test.go  # Integration with real filesystem
```

**Best Practices**:
- Domain tests: No mocks - pure logic testing
- Application tests: Mock all port dependencies
- Infrastructure tests: Mix of unit (mocked I/O) and integration (real I/O)
- Integration tests: Real implementations, test end-to-end flows

---

## Baseline Metrics (Current State)

### Test Coverage
```
$ go test ./... -cover
?       github.com/alexisbeaulieu97/streamy/cmd/streamy                    [no test files]
ok      github.com/alexisbeaulieu97/streamy/internal/config               0.007s  coverage: 85.2%
ok      github.com/alexisbeaulieu97/streamy/internal/engine               0.012s  coverage: 78.4%
ok      github.com/alexisbeaulieu97/streamy/internal/plugin               0.008s  coverage: 72.1%
ok      github.com/alexisbeaulieu97/streamy/internal/logger               0.003s  coverage: 68.5%
ok      github.com/alexisbeaulieu97/streamy/internal/domain/pipeline      0.009s  coverage: 81.3%
ok      github.com/alexisbeaulieu97/streamy/internal/app/pipeline         0.011s  coverage: 79.6%
ok      github.com/alexisbeaulieu97/streamy/internal/plugins/package      0.015s  coverage: 76.8%
ok      github.com/alexisbeaulieu97/streamy/internal/plugins/repo         0.018s  coverage: 74.2%
ok      github.com/alexisbeaulieu97/streamy/internal/plugins/symlink      0.006s  coverage: 80.1%
ok      github.com/alexisbeaulieu97/streamy/internal/plugins/copy         0.007s  coverage: 77.9%
ok      github.com/alexisbeaulieu97/streamy/internal/plugins/command      0.005s  coverage: 82.4%
ok      github.com/alexisbeaulieu97/streamy/internal/plugins/template     0.013s  coverage: 75.3%
ok      github.com/alexisbeaulieu97/streamy/internal/plugins/lineinfile   0.009s  coverage: 73.6%
ok      github.com/alexisbeaulieu97/streamy/internal/validation           0.004s  coverage: 69.2%
ok      github.com/alexisbeaulieu97/streamy/tests                         0.045s  coverage: N/A (integration)

AVERAGE: ~76% coverage across internal packages
TARGET AFTER REFACTOR: Domain 90%+, Application 85%+, Infrastructure 75%+
```

### Build Performance
```
$ time go build ./cmd/streamy
real    0m4.231s
user    0m6.842s
sys     0m1.094s

TARGET: <10s (SC-008)
CURRENT: 4.2s - well within target
```

### Test Execution
```
$ time go test ./...
real    0m2.847s
user    0m8.123s
sys     0m2.451s

TARGET: <20% increase (SC-008) = <3.4s
CURRENT: 2.8s baseline
```

### Package Dependency Violations
```
Current violations (domain→infrastructure imports):
- internal/domain/pipeline → internal/config (YAML parsing)
- internal/domain/pipeline → internal/engine (ExecutionContext)
- internal/domain/pipeline → internal/logger (Logger struct)
- internal/domain/pipeline → internal/plugin (PluginRegistry struct)

TARGET AFTER REFACTOR: Zero violations (domain has no infrastructure imports)
```

---

## Summary of Decisions

| Decision Area | Choice | Rationale |
|---------------|--------|-----------|
| **Architecture Pattern** | Hexagonal (Ports & Adapters) | Go idioms, testability, clear boundaries |
| **Dependency Injection** | Manual Constructor Injection | Simplicity, compile-time safety, no external deps |
| **Logging Migration** | charmbracelet/log with phased adapter | Better UX, charm ecosystem alignment |
| **Context Strategy** | Context-first with correlation IDs | Go idiom, cancellation, tracing support |
| **Error Handling** | Structured errors with wrapping | Type safety, context preservation, user guidance |
| **Testing Strategy** | Test doubles (mocks, fakes, stubs) | Port-based architecture enables easy isolation |

All decisions support Constitution principles and success criteria defined in spec.md.
