# Quickstart: Domain-Driven Architecture

**Feature**: 009-domain-driven-refactor  
**Audience**: Developers working on or extending Streamy's refactored architecture  
**Purpose**: Practical guide for adding features, implementing adapters, and testing in the new architecture

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Adding a New Use Case](#adding-a-new-use-case)
3. [Implementing a New Adapter](#implementing-a-new-adapter)
4. [Testing Strategies](#testing-strategies)
5. [Common Patterns](#common-patterns)
6. [Troubleshooting](#troubleshooting)

---

## Architecture Overview

Streamy uses a **three-layer domain-driven architecture** with strict dependency direction:

```
Infrastructure Layer (adapters, I/O, external systems)
        ↓ implements ports
Application Layer (use cases, orchestration)
        ↓ uses entities + ports
Domain Layer (entities, business logic - pure, no port knowledge)
        ← defined by ports (application boundary)
```

**Key Principles**:
- Domain has **zero dependencies** on infrastructure, application, OR ports (truly pure)
- Ports are defined at **application boundary** (`internal/ports/`) not in domain
- Application depends on domain entities + port interfaces
- Infrastructure implements port interfaces
- All wiring happens in `cmd/streamy/main.go` (composition root)

**Package Structure**:
```
internal/
├── ports/           # Port interfaces at application boundary
│   ├── config.go        # ConfigLoader
│   ├── execution.go     # PluginExecutor, DAGBuilder, ExecutionPlanner
│   ├── logging.go       # Logger
│   ├── observability.go # MetricsCollector, Tracer
│   ├── plugins.go       # Plugin, PluginRegistry
│   ├── events.go        # EventPublisher, EventHandler
│   └── registry.go      # RegistryStore, ValidationService
├── domain/          # Pure business logic (no infrastructure, no port knowledge)
│   ├── pipeline/    # Pipeline entities (Pipeline, Step, ExecutionPlan, etc.)
│   └── plugin/      # Plugin domain interfaces
├── application/     # Use cases (orchestration via ports)
│   ├── pipeline/    # Apply, Verify, Prepare use cases
│   └── validation/  # Validation orchestration
└── infrastructure/  # Adapters implementing ports
    ├── config/      # YAML parser (implements ConfigLoader)
    ├── engine/      # Executor (implements PluginExecutor)
    ├── logging/     # Logger adapter (charmbracelet/log)
    └── persistence/ # Registry storage
```

---

## Adding a New Use Case

**Example**: Add a "DiffPipeline" use case that shows what would change without applying.

### Step 1: Define Use Case in Application Layer

Create `internal/application/pipeline/diff_usecase.go`:

```go
package pipeline

import (
	"context"
	"github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

// DiffUseCase compares current state with desired state without applying changes.
type DiffUseCase struct {
	loader   ports.ConfigLoader
	builder  ports.DAGBuilder
	planner  ports.ExecutionPlanner
	executor ports.PluginExecutor  // Will call Evaluate only, not Apply
	logger   ports.Logger
}

// NewDiffUseCase creates the use case with injected dependencies.
func NewDiffUseCase(
	loader ports.ConfigLoader,
	builder ports.DAGBuilder,
	planner ports.ExecutionPlanner,
	executor ports.PluginExecutor,
	logger ports.Logger,
) *DiffUseCase {
	return &DiffUseCase{
		loader:   loader,
		builder:  builder,
		planner:  planner,
		executor: executor,
		logger:   logger,
	}
}

// Diff generates a diff showing what would change.
func (u *DiffUseCase) Diff(ctx context.Context, configPath string) (*pipeline.DiffResult, error) {
	u.logger.Info(ctx, "generating diff", "path", configPath)

	// Load pipeline
	pip, err := u.loader.Load(ctx, configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// Build execution plan
	graph, err := u.builder.Build(ctx, pip.Steps)
	if err != nil {
		return nil, fmt.Errorf("build DAG: %w", err)
	}

	plan, err := u.planner.GeneratePlan(ctx, graph)
	if err != nil {
		return nil, fmt.Errorf("generate plan: %w", err)
	}

	// Evaluate only (no Apply)
	evaluations, err := u.executor.Verify(ctx, pip)
	if err != nil {
		return nil, fmt.Errorf("evaluate steps: %w", err)
	}

	// Aggregate results into diff
	return buildDiff(evaluations), nil
}
```

### Step 2: Add Tests with Mocks

Create `internal/application/pipeline/diff_usecase_test.go`:

```go
package pipeline_test

import (
	"context"
	"testing"

	"github.com/alexisbeaulieu97/streamy/internal/application/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/application/pipeline/testutil"
	"github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
)

func TestDiffUseCase_Success(t *testing.T) {
	// Arrange: Create test doubles
	mockLoader := &testutil.MockConfigLoader{
		Pipeline: &domain.Pipeline{
			Name:  "test-pipeline",
			Steps: []domain.Step{{ID: "step1", Type: "command"}},
		},
	}
	mockBuilder := &testutil.MockDAGBuilder{}
	mockPlanner := &testutil.MockPlanner{}
	mockExecutor := &testutil.MockExecutor{
		VerifyResults: []domain.VerificationResult{
			{Status: "drifted", Message: "step1 needs changes"},
		},
	}
	mockLogger := &testutil.MockLogger{}

	useCase := pipeline.NewDiffUseCase(
		mockLoader,
		mockBuilder,
		mockPlanner,
		mockExecutor,
		mockLogger,
	)

	// Act
	result, err := useCase.Diff(context.Background(), "/fake/config.yaml")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if mockLoader.LoadCallCount != 1 {
		t.Errorf("expected Load called once, got %d", mockLoader.LoadCallCount)
	}
}
```

### Step 3: Wire in main.go

Update `cmd/streamy/main.go`:

```go
func main() {
	// ... existing adapter creation ...

	// Wire new use case
	diffUseCase := pipeline.NewDiffUseCase(
		configLoader,
		dagBuilder,
		planner,
		executor,
		logger,
	)

	// Wire into CLI
	diffCmd := cmd.NewDiffCommand(diffUseCase, logger)

	// Add to root command
	rootCmd := cmd.NewRootCommand(applyCmd, verifyCmd, diffCmd, dashboardCmd)
	rootCmd.Execute()
}
```

### Step 4: Create CLI Command

Create `cmd/streamy/diff.go`:

```go
package main

import (
	"context"
	"github.com/spf13/cobra"
)

func NewDiffCommand(useCase *pipeline.DiffUseCase, logger ports.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "diff <config-path>",
		Short: "Show what would change without applying",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			result, err := useCase.Diff(ctx, args[0])
			if err != nil {
				return err
			}
			// Print diff...
			return nil
		},
	}
}
```

---

## Implementing a New Adapter

**Example**: Implement a JSON config loader (alternative to YAML).

### Step 1: Port Interface Already Exists

The port interface is already defined in `internal/ports/config.go`:

```go
type ConfigLoader interface {
	Load(ctx context.Context, path string) (*Pipeline, error)
	Validate(ctx context.Context, path string) error
}
```

### Step 2: Implement Adapter

Create `internal/infrastructure/config/json_loader.go`:

```go
package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

// JSONLoader implements ports.ConfigLoader for JSON files.
type JSONLoader struct {
	logger ports.Logger
}

// NewJSONLoader creates a JSON config loader.
func NewJSONLoader(logger ports.Logger) *JSONLoader {
	return &JSONLoader{
		logger: logger,
	}
}

// Verify interface implementation at compile time
var _ ports.ConfigLoader = (*JSONLoader)(nil)

// Load reads and parses a JSON config file.
func (l *JSONLoader) Load(ctx context.Context, path string) (*pipeline.Pipeline, error) {
	l.logger.Debug(ctx, "loading JSON config", "path", path)

	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return nil, &pipeline.DomainError{
			Code:    pipeline.ErrCodeCancelled,
			Message: "load cancelled",
			Cause:   err,
		}
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &pipeline.DomainError{
				Code:    pipeline.ErrCodeNotFound,
				Message: fmt.Sprintf("config file not found: %s", path),
				Cause:   err,
			}
		}
		return nil, fmt.Errorf("read file: %w", err)
	}

	// Parse JSON
	var pip pipeline.Pipeline
	if err := json.Unmarshal(data, &pip); err != nil {
		return nil, &pipeline.DomainError{
			Code:    pipeline.ErrCodeValidation,
			Message: "invalid JSON syntax",
			Cause:   err,
			Context: map[string]interface{}{
				"file": path,
			},
		}
	}

	// Validate domain invariants
	if err := pip.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	l.logger.Info(ctx, "loaded config", "name", pip.Name, "steps", len(pip.Steps))
	return &pip, nil
}

// Validate checks JSON syntax without fully loading.
func (l *JSONLoader) Validate(ctx context.Context, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	var pip pipeline.Pipeline
	if err := json.Unmarshal(data, &pip); err != nil {
		return &pipeline.DomainError{
			Code:    pipeline.ErrCodeValidation,
			Message: "invalid JSON",
			Cause:   err,
		}
	}

	return pip.Validate()
}
```

### Step 3: Add Tests

Create `internal/infrastructure/config/json_loader_test.go`:

```go
package config_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/alexisbeaulieu97/streamy/internal/infrastructure/config"
	"github.com/alexisbeaulieu97/streamy/internal/infrastructure/logging"
)

func TestJSONLoader_Load_Success(t *testing.T) {
	// Arrange: Create temp JSON file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	configData := `{
		"version": "1.0",
		"name": "test",
		"steps": [
			{"id": "step1", "type": "command", "command": "echo test"}
		]
	}`
	os.WriteFile(configPath, []byte(configData), 0644)

	logger := logging.NewNoOpLogger()
	loader := config.NewJSONLoader(logger)

	// Act
	pipeline, err := loader.Load(context.Background(), configPath)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if pipeline == nil {
		t.Fatal("expected pipeline, got nil")
	}
	if pipeline.Name != "test" {
		t.Errorf("expected name 'test', got %s", pipeline.Name)
	}
}

func TestJSONLoader_Load_FileNotFound(t *testing.T) {
	logger := logging.NewNoOpLogger()
	loader := config.NewJSONLoader(logger)

	_, err := loader.Load(context.Background(), "/nonexistent/file.json")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Check for ErrCodeNotFound...
}
```

### Step 4: Wire Conditionally in main.go

```go
func main() {
	var loader ports.ConfigLoader

	// Choose loader based on file extension or flag
	if *jsonMode {
		loader = config.NewJSONLoader(logger)
	} else {
		loader = config.NewYAMLLoader(yamlConfig, logger)
	}

	// Use loader in use cases
	applyUseCase := pipeline.NewApplyUseCase(loader, /* ... */)
}
```

---

## Testing Strategies

### Domain Layer Tests (Pure Unit Tests)

**Goal**: Test business logic without any infrastructure.

```go
// internal/domain/pipeline/pipeline_test.go
func TestPipeline_Validate_UniqueIDs(t *testing.T) {
	// No mocks needed - pure logic test
	pip := &Pipeline{
		Steps: []Step{
			{ID: "step1", Type: "command"},
			{ID: "step1", Type: "command"}, // Duplicate!
		},
	}

	err := pip.Validate()

	if err == nil {
		t.Fatal("expected validation error for duplicate IDs")
	}
}
```

**Characteristics**:
- ✅ No mocks or test doubles
- ✅ Fast (<1ms per test)
- ✅ Tests business rules and invariants
- ✅ No file I/O, no network, no external dependencies

### Application Layer Tests (Mock-Based Unit Tests)

**Goal**: Test use case orchestration with mocked dependencies.

```go
// internal/application/pipeline/apply_usecase_test.go
func TestApplyUseCase_LoadFailure(t *testing.T) {
	// Mock that returns error
	mockLoader := &testutil.MockConfigLoader{
		LoadError: errors.New("file not found"),
	}
	mockLogger := &testutil.MockLogger{}

	useCase := NewApplyUseCase(mockLoader, /* ... */, mockLogger)

	err := useCase.Apply(context.Background(), "/fake/path.yaml", false)

	// Verify error handling
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Verify logger was called with error
	if mockLogger.ErrorCallCount != 1 {
		t.Errorf("expected Error called once, got %d", mockLogger.ErrorCallCount)
	}
}
```

**Characteristics**:
- ✅ Mocks for all port dependencies
- ✅ Tests orchestration logic
- ✅ Verifies interactions between components
- ✅ Fast (<10ms per test)

### Infrastructure Layer Tests (Real and Mocked I/O)

**Goal**: Test adapter implementations with real or test I/O.

```go
// internal/infrastructure/config/yaml_loader_test.go
func TestYAMLLoader_Load_RealFile(t *testing.T) {
	// Real file I/O with temp directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	yaml := `version: "1.0"\nname: test\nsteps:\n  - id: step1\n    type: command`
	os.WriteFile(configPath, []byte(yaml), 0644)

	logger := logging.NewNoOpLogger()
	loader := NewYAMLLoader(YAMLLoaderConfig{}, logger)

	pipeline, err := loader.Load(context.Background(), configPath)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if pipeline.Name != "test" {
		t.Errorf("expected name 'test', got %s", pipeline.Name)
	}
}
```

**Characteristics**:
- ✅ Mix of real and mocked I/O
- ✅ Use `t.TempDir()` for filesystem tests
- ✅ Slower than domain/app tests (~50-100ms)
- ✅ Tests actual adapter behavior

### Integration Tests (End-to-End)

**Goal**: Validate entire system with real adapters.

```go
// tests/integration_test.go
func TestApplyPipeline_EndToEnd(t *testing.T) {
	// Real adapters, real config file
	logger := logging.NewLogger(logging.Config{Level: "error"})
	loader := config.NewYAMLLoader(config.YAMLLoaderConfig{}, logger)
	// ... create all real adapters ...

	useCase := pipeline.NewApplyUseCase(loader, builder, planner, executor, validator, logger, metrics)

	err := useCase.Apply(context.Background(), "testdata/configs/simple.yaml", false)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Verify side effects (files created, commands run, etc.)
}
```

**Characteristics**:
- ✅ Real implementations (no mocks)
- ✅ Tests user workflows end-to-end
- ✅ Validates all layers work together
- ✅ Slow (1-10s per test)
- ✅ Run after each phase to validate SC-007

---

## Common Patterns

### Pattern 1: Context Propagation

Always pass `context.Context` as the first parameter:

```go
// ✅ GOOD: Context first
func (u *UseCase) Apply(ctx context.Context, path string) error {
	// Check cancellation before expensive operations
	if err := ctx.Err(); err != nil {
		return &DomainError{Code: ErrCodeCancelled, Cause: err}
	}

	// Pass context to all dependencies
	pipeline, err := u.loader.Load(ctx, path)
	// ...
}

// ❌ BAD: No context
func (u *UseCase) Apply(path string) error {
	// Can't cancel, no timeout support
}
```

### Pattern 2: Error Wrapping with Context

Preserve error chain while adding context at each layer:

```go
// Domain layer: Typed errors
return &DomainError{
	Code:    ErrCodeValidation,
	Message: "step dependency not found",
	Context: map[string]interface{}{
		"step_id": step.ID,
		"missing": dep,
	},
}

// Infrastructure layer: Add technical context
if err != nil {
	return fmt.Errorf("read config from %s: %w", path, err)
}

// Application layer: Add user guidance
if err != nil {
	return fmt.Errorf("failed to prepare pipeline: %w. Check config file syntax", err)
}
```

### Pattern 3: Logger with Fields

Use structured logging with key-value fields:

```go
// Create child logger for component
stepLogger := logger.With("step_id", step.ID, "plugin", pluginName)

// Log with additional fields
stepLogger.Info(ctx, "executing step", "timeout", timeout, "dry_run", dryRun)

// Log errors with full context
stepLogger.Error(ctx, "step failed", "error", err, "duration", duration)
```

### Pattern 4: Graceful Cancellation

Respect context cancellation:

```go
func (e *Executor) Execute(ctx context.Context, plan *ExecutionPlan) error {
	for _, level := range plan.Levels {
		// Check cancellation before each level
		select {
		case <-ctx.Done():
			return &DomainError{
				Code:    ErrCodeCancelled,
				Message: "execution cancelled",
				Cause:   ctx.Err(),
			}
		default:
		}

		// Execute level...
	}
	return nil
}
```

---

## Troubleshooting

### Problem: "cannot use concrete type as interface"

```
cannot use loader (type *YAMLLoader) as type ConfigLoader in argument to NewApplyUseCase
```

**Solution**: Verify interface implementation with compile-time check:

```go
var _ ports.ConfigLoader = (*YAMLLoader)(nil)
```

### Problem: "import cycle detected"

```
import cycle not allowed
internal/domain/pipeline → internal/infrastructure/config → internal/domain/pipeline
```

**Solution**: Domain should not import infrastructure. Move interface to domain/ports.

### Problem: "mock doesn't implement interface"

```
cannot use &MockLoader{} (type *MockLoader) as type ConfigLoader in argument
```

**Solution**: Implement all interface methods in mock:

```go
type MockLoader struct{}

func (m *MockLoader) Load(ctx context.Context, path string) (*Pipeline, error) {
	return nil, nil
}

func (m *MockLoader) Validate(ctx context.Context, path string) error {
	return nil
}

var _ ports.ConfigLoader = (*MockLoader)(nil) // Verify at compile time
```

### Problem: "nil pointer dereference in test"

```
panic: runtime error: invalid memory address or nil pointer dereference
```

**Solution**: Initialize all mocks properly:

```go
// ❌ BAD: Nil mock
mockLogger := &MockLogger{}
mockLogger.InfoFunc = nil // Will panic!

// ✅ GOOD: Provide default implementation
mockLogger := &MockLogger{
	InfoFunc: func(ctx context.Context, msg string, fields ...interface{}) {
		// No-op or record call
	},
}
```

### Problem: "tests pass individually but fail when run together"

**Solution**: Tests are sharing state. Ensure:
- Each test creates its own mocks
- Use `t.TempDir()` for filesystem tests
- Don't rely on execution order
- Clean up resources in `t.Cleanup()`

---

## Next Steps

1. **Read the data model** (`data-model.md`) to understand all entities
2. **Review port interfaces** (contracts in `specs/009-domain-driven-refactor/contracts/`; implementations will be in `internal/ports/`)
3. **Study wiring patterns** (`contracts/wiring.go`)
4. **Look at existing tests** in `internal/domain/pipeline/` for examples
5. **Start implementing** Phase 3 tasks (domain layer)

**Need Help?**
- Check `docs/architecture-overview.md` for architecture diagrams
- Review `research.md` for design decisions and rationale
- Look at `specs/001-create-streamy-mvp/` for original MVP patterns
