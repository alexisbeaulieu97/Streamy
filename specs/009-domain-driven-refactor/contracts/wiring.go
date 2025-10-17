//go:build ignore
// +build ignore

// Dependency Injection Wiring Patterns
//
// This file documents the patterns and practices for wiring dependencies
// in Streamy's domain-driven architecture.
//
// PRINCIPLES:
// 1. Manual Constructor Injection (no frameworks, no reflection)
// 2. Interfaces for all dependencies (enables testing with mocks)
// 3. Composition root in cmd/streamy/main.go
// 4. Compile-time type safety (errors caught at build time)
// 5. Max 5-7 dependencies per constructor (cognitive load limit)

package wiring

import (
	"context"
	"fmt"
)

// ============================================================================
// PATTERN 1: Use Case with Port Dependencies
// ============================================================================

// Application layer use cases accept port interfaces via constructor injection.
// This enables testing with mocks and keeps use cases decoupled from infrastructure.

// Example: ApplyUseCase orchestrates pipeline application
type ApplyUseCase struct {
	loader    ConfigLoader    // Domain port: loads config
	builder   DAGBuilder      // Domain port: builds DAG
	planner   ExecutionPlanner // Domain port: generates plan
	executor  PluginExecutor  // Domain port: executes plan
	validator ValidationService // App port: runs validations
	logger    Logger          // Domain port: structured logging
	metrics   MetricsCollector // Domain port: records metrics
}

// NewApplyUseCase creates a use case with all dependencies injected.
//
// Benefits:
// - All dependencies explicit (no hidden coupling)
// - Compile-time type checking (wrong types = build error)
// - Easy to test (inject mocks for each interface)
// - Clear at call site what the use case needs
func NewApplyUseCase(
	loader ConfigLoader,
	builder DAGBuilder,
	planner ExecutionPlanner,
	executor PluginExecutor,
	validator ValidationService,
	logger Logger,
	metrics MetricsCollector,
) *ApplyUseCase {
	return &ApplyUseCase{
		loader:    loader,
		builder:   builder,
		planner:   planner,
		executor:  executor,
		validator: validator,
		logger:    logger,
		metrics:   metrics,
	}
}

// Apply implements the use case logic using injected dependencies.
func (u *ApplyUseCase) Apply(ctx context.Context, configPath string, dryRun bool) error {
	// Use logger (injected dependency)
	u.logger.Info(ctx, "applying pipeline", "path", configPath, "dry_run", dryRun)

	// Use loader (injected dependency)
	pipeline, err := u.loader.Load(ctx, configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Use builder and planner (injected dependencies)
	graph, err := u.builder.Build(ctx, pipeline.Steps)
	if err != nil {
		return fmt.Errorf("build DAG: %w", err)
	}

	plan, err := u.planner.GeneratePlan(ctx, graph)
	if err != nil {
		return fmt.Errorf("generate plan: %w", err)
	}

	// Use executor (injected dependency)
	results, err := u.executor.Execute(ctx, plan, pipeline)
	u.metrics.RecordPipelineExecution(pipeline.Name, /* duration */, err == nil)

	if err != nil {
		return fmt.Errorf("execute plan: %w", err)
	}

	// Use validator (injected dependency)
	summary, err := u.validator.RunValidations(ctx, pipeline.Validations)
	if err != nil {
		u.logger.Warn(ctx, "validation failed", "error", err)
	}

	return nil
}

// ============================================================================
// PATTERN 2: Adapter with Configuration
// ============================================================================

// Infrastructure adapters may need configuration in addition to dependencies.
// Use a Config struct to pass multiple configuration values cleanly.

// YAMLLoaderConfig holds configuration for YAML config loader.
type YAMLLoaderConfig struct {
	ValidateSchema bool   // Whether to validate YAML against JSON schema
	MaxFileSize    int64  // Maximum config file size in bytes
	CacheTTL       int    // How long to cache parsed configs (seconds)
}

// YAMLLoader implements ConfigLoader port.
type YAMLLoader struct {
	config YAMLLoaderConfig
	logger Logger
}

// NewYAMLLoader creates a YAML config loader with configuration and dependencies.
//
// Pattern: Config struct first, then dependency interfaces.
func NewYAMLLoader(config YAMLLoaderConfig, logger Logger) *YAMLLoader {
	return &YAMLLoader{
		config: config,
		logger: logger,
	}
}

// Load implements ConfigLoader interface.
func (l *YAMLLoader) Load(ctx context.Context, path string) (*Pipeline, error) {
	l.logger.Debug(ctx, "loading config", "path", path, "validate_schema", l.config.ValidateSchema)
	// ... implementation using l.config and l.logger
	return nil, nil
}

// ============================================================================
// PATTERN 3: Factory Functions for Complex Object Graphs
// ============================================================================

// When wiring gets complex (many dependencies), use factory functions to hide
// the complexity and provide sensible defaults.

// ExecutorFactory creates a plugin executor with all required dependencies.
type ExecutorFactory struct {
	registry PluginRegistry
	logger   Logger
	metrics  MetricsCollector
}

// NewExecutorFactory creates a factory for building executors.
func NewExecutorFactory(registry PluginRegistry, logger Logger, metrics MetricsCollector) *ExecutorFactory {
	return &ExecutorFactory{
		registry: registry,
		logger:   logger,
		metrics:  metrics,
	}
}

// CreateExecutor builds a fully configured executor.
//
// This hides internal complexity like creating worker pools, setting up
// cancellation handlers, etc.
func (f *ExecutorFactory) CreateExecutor(parallelism int) *DAGExecutor {
	return &DAGExecutor{
		registry:    f.registry,
		logger:      f.logger,
		metrics:     f.metrics,
		workerPool:  createWorkerPool(parallelism),
		cancelChan:  make(chan struct{}),
	}
}

// ============================================================================
// PATTERN 4: Composition Root in main.go
// ============================================================================

// All dependency wiring happens in cmd/streamy/main.go.
// This is the "composition root" - the only place where concrete types meet.

func main() {
	// 1. Create infrastructure adapters (concrete types)
	logger := logging.NewLogger(logging.Config{
		Level:  "info",
		Format: "pretty",
	})

	configLoader := config.NewYAMLLoader(
		config.YAMLLoaderConfig{
			ValidateSchema: true,
			MaxFileSize:    10 * 1024 * 1024, // 10MB
		},
		logger,
	)

	dagBuilder := engine.NewDAGBuilder(logger)
	planner := engine.NewPlanner(logger)

	pluginRegistry := plugin.NewRegistry(logger)
	// Register plugins...

	executor := engine.NewExecutor(pluginRegistry, logger, metrics.NewNoOpCollector())
	validator := validation.NewService(logger)
	registryStore := persistence.NewFileRegistryStore("/var/lib/streamy/registry", logger)

	// 2. Wire application use cases (inject port interfaces)
	applyUseCase := pipeline.NewApplyUseCase(
		configLoader,  // ConfigLoader interface
		dagBuilder,    // DAGBuilder interface
		planner,       // ExecutionPlanner interface
		executor,      // PluginExecutor interface
		validator,     // ValidationService interface
		logger,        // Logger interface
		metrics.NewNoOpCollector(), // MetricsCollector interface
	)

	verifyUseCase := pipeline.NewVerifyUseCase(
		configLoader,
		dagBuilder,
		planner,
		executor,
		validator,
		logger,
	)

	// 3. Wire CLI commands (inject use cases)
	applyCmd := cmd.NewApplyCommand(applyUseCase, logger)
	verifyCmd := cmd.NewVerifyCommand(verifyUseCase, logger)
	dashboardCmd := cmd.NewDashboardCommand(registryStore, verifyUseCase, applyUseCase, logger)

	// 4. Execute CLI
	rootCmd := cmd.NewRootCommand(applyCmd, verifyCmd, dashboardCmd)
	if err := rootCmd.Execute(); err != nil {
		logger.Error(context.Background(), "command failed", "error", err)
		os.Exit(1)
	}
}

// ============================================================================
// PATTERN 5: Testing with Mocks
// ============================================================================

// Because all dependencies are interfaces, testing is straightforward:
// create test doubles that implement the interface.

// MockConfigLoader implements ConfigLoader for testing.
type MockConfigLoader struct {
	LoadFunc     func(ctx context.Context, path string) (*Pipeline, error)
	ValidateFunc func(ctx context.Context, path string) error
	CallCount    int
}

func (m *MockConfigLoader) Load(ctx context.Context, path string) (*Pipeline, error) {
	m.CallCount++
	if m.LoadFunc != nil {
		return m.LoadFunc(ctx, path)
	}
	return &Pipeline{Name: "test"}, nil
}

func (m *MockConfigLoader) Validate(ctx context.Context, path string) error {
	if m.ValidateFunc != nil {
		return m.ValidateFunc(ctx, path)
	}
	return nil
}

// Example test using mock
func TestApplyUseCase_Success(t *testing.T) {
	// Arrange: Create mocks
	mockLoader := &MockConfigLoader{
		LoadFunc: func(ctx context.Context, path string) (*Pipeline, error) {
			return &Pipeline{Name: "test", Steps: []Step{}}, nil
		},
	}
	mockBuilder := &MockDAGBuilder{}  // Similar implementation
	mockPlanner := &MockPlanner{}     // Similar implementation
	mockExecutor := &MockExecutor{}   // Similar implementation
	mockValidator := &MockValidator{} // Similar implementation
	mockLogger := &MockLogger{}       // Similar implementation
	mockMetrics := &MockMetrics{}     // Similar implementation

	// Create use case with mocks
	useCase := NewApplyUseCase(
		mockLoader,
		mockBuilder,
		mockPlanner,
		mockExecutor,
		mockValidator,
		mockLogger,
		mockMetrics,
	)

	// Act: Execute use case
	err := useCase.Apply(context.Background(), "/fake/path.yaml", false)

	// Assert: Verify behavior
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mockLoader.CallCount != 1 {
		t.Errorf("expected Load called once, got %d", mockLoader.CallCount)
	}
}

// ============================================================================
// PATTERN 6: Functional Options for Optional Dependencies
// ============================================================================

// For components with many optional dependencies, use functional options pattern.

type ExecutorOption func(*ExecutorConfig)

type ExecutorConfig struct {
	parallelism int
	timeout     int
	metrics     MetricsCollector
	tracer      Tracer
}

// WithParallelism sets max concurrent steps.
func WithParallelism(n int) ExecutorOption {
	return func(c *ExecutorConfig) {
		c.parallelism = n
	}
}

// WithMetrics sets metrics collector.
func WithMetrics(m MetricsCollector) ExecutorOption {
	return func(c *ExecutorConfig) {
		c.metrics = m
	}
}

// WithTracer sets distributed tracer.
func WithTracer(t Tracer) ExecutorOption {
	return func(c *ExecutorConfig) {
		c.tracer = t
	}
}

// NewExecutorWithOptions creates executor with optional dependencies.
func NewExecutorWithOptions(registry PluginRegistry, logger Logger, opts ...ExecutorOption) *DAGExecutor {
	// Defaults
	config := ExecutorConfig{
		parallelism: 4,
		timeout:     300,
		metrics:     &NoOpMetrics{},
		tracer:      &NoOpTracer{},
	}

	// Apply options
	for _, opt := range opts {
		opt(&config)
	}

	return &DAGExecutor{
		registry:    registry,
		logger:      logger,
		parallelism: config.parallelism,
		timeout:     config.timeout,
		metrics:     config.metrics,
		tracer:      config.tracer,
	}
}

// Usage:
// executor := NewExecutorWithOptions(
//     registry,
//     logger,
//     WithParallelism(8),
//     WithMetrics(prometheusCollector),
// )

// ============================================================================
// ANTI-PATTERNS TO AVOID
// ============================================================================

// ❌ DON'T: Global variables
var globalLogger Logger // BAD: Makes testing hard, creates hidden dependencies

// ❌ DON'T: Service Locator pattern
type ServiceLocator struct {
	services map[string]interface{}
}
func (s *ServiceLocator) Get(name string) interface{} { /* ... */ } // BAD: Runtime errors, no type safety

// ❌ DON'T: Concrete dependencies in constructor
func NewUseCase(loader *YAMLLoader) *UseCase { /* ... */ } // BAD: Can't inject mocks for testing

// ❌ DON'T: Too many dependencies (cognitive overload)
func NewUseCase(a, b, c, d, e, f, g, h, i, j interface{}) {} // BAD: >7 deps = design smell

// ✅ DO: Interface dependencies
func NewUseCase(loader ConfigLoader) *UseCase { /* ... */ } // GOOD: Can inject any impl

// ✅ DO: Factory functions for complex wiring
func NewUseCaseFactory(common CommonDeps) *UseCaseFactory { /* ... */ } // GOOD: Hides complexity

// ============================================================================
// GUIDELINES
// ============================================================================

/*
1. CONSTRUCTOR INJECTION
   - All dependencies passed as constructor parameters
   - Constructors never fail (no error return)
   - Validate dependencies at construction time (panic if nil)

2. INTERFACE SEGREGATION
   - Keep interfaces small and focused (2-5 methods)
   - Don't depend on interfaces you don't use
   - Define interfaces in consumer package (port pattern)

3. DEPENDENCY LIMITS
   - Max 5-7 dependencies per constructor
   - If more needed, create factory or use functional options
   - Group related dependencies into aggregate interfaces

4. COMPOSITION ROOT
   - All wiring in main.go (single location)
   - Pass fully-wired dependencies down call chain
   - Never construct dependencies inside use cases

5. TESTING
   - Create mocks that implement interfaces
   - Use table-driven tests with different mock implementations
   - Verify interactions with mocks (call counts, parameters)

6. ERROR HANDLING
   - Constructors never fail (panic on nil dependencies)
   - Methods return errors wrapped with context
   - Use domain error types for business logic errors
*/

// ============================================================================
// MIGRATION STRATEGY
// ============================================================================

/*
Migrating existing code to dependency injection:

1. Extract interface for existing type
   OLD: func NewService() *Service
   NEW: type ServiceInterface interface { ... }
        func NewService() ServiceInterface

2. Change constructor to accept interfaces
   OLD: func NewUseCase(svc *Service) *UseCase
   NEW: func NewUseCase(svc ServiceInterface) *UseCase

3. Update main.go wiring
   OLD: svc := NewService()
        uc := NewUseCase(svc)
   NEW: svc := concrete.NewService() // implements ServiceInterface
        uc := NewUseCase(svc)         // accepts interface

4. Create mocks for tests
   NEW: type MockService struct { ... }
        func (m *MockService) Method() { ... }

5. Validate: All tests still pass, no behavior changes
*/

// Interface placeholders (actual definitions in respective packages)
type ConfigLoader interface{}
type DAGBuilder interface{}
type ExecutionPlanner interface{}
type PluginExecutor interface{}
type ValidationService interface{}
type Logger interface{}
type MetricsCollector interface{}
type Tracer interface{}
type PluginRegistry interface{}
type Pipeline struct{}
type Step struct{}
type DAGExecutor struct {
	registry    PluginRegistry
	logger      Logger
	metrics     MetricsCollector
	parallelism int
	timeout     int
	tracer      Tracer
	workerPool  interface{}
	cancelChan  chan struct{}
}

func createWorkerPool(parallelism int) interface{} { return nil }
type NoOpMetrics struct{}
type NoOpTracer struct{}
