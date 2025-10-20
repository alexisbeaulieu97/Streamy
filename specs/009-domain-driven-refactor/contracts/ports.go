//go:build ignore
// +build ignore

// Port Interfaces for Domain-Driven Architecture
// Package: internal/ports
//
// These interfaces define the contracts that infrastructure adapters must satisfy.
// They are placed at the APPLICATION BOUNDARY (not in domain layer) to preserve
// a truly pure domain core with zero infrastructure dependencies.
//
// Architecture: Infrastructure → Application → Domain
//   - Domain layer: Pure business logic (entities, value objects, business rules)
//   - Application layer: Use case orchestration (depends on domain + ports)
//   - Infrastructure layer: Technical implementations (implements ports)
//   - Ports package: Interface contracts (defines what application needs)
//
// All port methods accept context.Context as the first parameter for:
// - Cancellation support
// - Deadline propagation
// - Correlation ID tracking

package ports

import (
	"context"
	"time"
)

// ConfigLoader loads and parses pipeline configurations from external sources.
//
// Implementations: internal/infrastructure/config/yaml_loader.go
//
// Example:
//
//	loader := config.NewYAMLLoader(logger)
//	pipeline, err := loader.Load(ctx, "/path/to/config.yaml")
type ConfigLoader interface {
	// Load reads and parses a pipeline configuration from the specified path.
	//
	// Returns:
	//   - *Pipeline: Fully validated pipeline entity
	//   - error: Domain error with code:
	//     - ErrCodeNotFound: File not found
	//     - ErrCodeValidation: Invalid YAML or schema validation failed
	//     - ErrCodeCancelled: Context cancelled during load
	Load(ctx context.Context, path string) (*Pipeline, error)

	// Validate checks if a config file is syntactically valid without fully loading.
	// Useful for pre-flight checks and CLI validation commands.
	//
	// Returns:
	//   - error: Domain error with validation details if invalid
	Validate(ctx context.Context, path string) error
}

// PluginExecutor executes plugins against steps to mutate system state or verify state.
//
// Implementations: internal/infrastructure/engine/executor.go
//
// Example:
//
//	executor := engine.NewExecutor(registry, logger)
//	results, err := executor.Execute(ctx, plan, pipeline)
type PluginExecutor interface {
	// Execute runs the execution plan, dispatching steps to plugins in parallel where safe.
	//
	// The executor:
	// - Respects execution level ordering (steps in same level run concurrently)
	// - Passes context for cancellation (checks ctx.Err() between levels)
	// - Records step results and aggregates them
	// - Stops on first error unless ContinueOnError is true
	//
	// Parameters:
	//   - ctx: Context with deadline/cancellation, correlation ID
	//   - plan: Execution plan with level-based step ordering
	//   - pipeline: Pipeline entity with step definitions
	//
	// Returns:
	//   - []StepResult: Results for all executed steps (in plan order)
	//   - error: Domain error if execution failed:
	//     - ErrCodeExecution: Plugin execution failed
	//     - ErrCodeTimeout: Step exceeded timeout
	//     - ErrCodeCancelled: Context cancelled mid-execution
	Execute(ctx context.Context, plan *ExecutionPlan, pipeline *Pipeline) ([]StepResult, error)

	// Verify checks if steps are in desired state without mutating system.
	// Calls plugin.Evaluate() for each step but never plugin.Apply().
	//
	// Returns:
	//   - []VerificationResult: State assessment for each step
	//   - error: Domain error if verification failed
	Verify(ctx context.Context, pipeline *Pipeline) ([]VerificationResult, error)
}

// Logger provides structured logging with context propagation.
//
// Implementations: internal/infrastructure/logging/logger.go
//
// The logger automatically extracts correlation ID from context and includes it
// in all log entries for tracing.
//
// Example:
//
//	logger := logging.NewLogger(config)
//	logger.Info(ctx, "pipeline started", "name", pipeline.Name)
type Logger interface {
	// Debug logs debug-level message with optional key-value fields.
	// Only logged if debug level is enabled.
	//
	// Parameters:
	//   - ctx: Context with correlation ID
	//   - msg: Log message
	//   - fields: Key-value pairs (must be even number of args)
	Debug(ctx context.Context, msg string, fields ...interface{})

	// Info logs informational message.
	Info(ctx context.Context, msg string, fields ...interface{})

	// Warn logs warning message.
	Warn(ctx context.Context, msg string, fields ...interface{})

	// Error logs error message. Should include error in fields.
	//
	// Example:
	//   logger.Error(ctx, "step failed", "step_id", step.ID, "error", err)
	Error(ctx context.Context, msg string, fields ...interface{})

	// With returns a child logger with additional fields pre-set.
	// Useful for component-scoped logging.
	//
	// Example:
	//   stepLogger := logger.With("step_id", step.ID, "plugin", "repo")
	//   stepLogger.Info(ctx, "cloning repository", "url", repoURL)
	With(fields ...interface{}) Logger
}

// MetricsCollector records execution metrics for observability.
//
// Implementations:
//   - internal/infrastructure/metrics/noop_collector.go (default for dev/test)
//   - internal/infrastructure/metrics/prometheus_collector.go (future)
//
// Example:
//
//	collector := metrics.NewNoOpCollector()
//	collector.RecordStepDuration(step.ID, duration)
type MetricsCollector interface {
	// RecordStepDuration records how long a step took to execute.
	RecordStepDuration(stepID string, duration time.Duration)

	// RecordStepStatus records step outcome (success, failure, skipped, etc.).
	RecordStepStatus(stepID string, status ResultStatus)

	// RecordPipelineExecution records overall pipeline metrics.
	RecordPipelineExecution(pipelineName string, duration time.Duration, success bool)
}

// Tracer provides distributed tracing spans for observability.
//
// Implementations:
//   - internal/infrastructure/tracing/noop_tracer.go (default for dev/test)
//   - internal/infrastructure/tracing/otel_tracer.go (future)
//
// Example:
//
//	tracer := tracing.NewNoOpTracer()
//	span := tracer.StartSpan(ctx, "execute_step", "step_id", step.ID)
//	defer span.End()
type Tracer interface {
	// StartSpan creates a new tracing span with optional attributes.
	// Returns a Span that must be closed with End() when operation completes.
	StartSpan(ctx context.Context, name string, attributes ...interface{}) Span

	// Extract retrieves trace context from context.Context.
	// Used for continuing traces across process boundaries.
	Extract(ctx context.Context) (interface{}, error)

	// Inject adds trace context to context.Context.
	// Used for propagating traces to called services.
	Inject(ctx context.Context, traceCtx interface{}) context.Context
}

// Span represents a single operation in a distributed trace.
type Span interface {
	// End completes the span and records its duration.
	End()

	// SetError marks the span as errored with error details.
	SetError(err error)

	// SetAttribute adds a key-value attribute to the span.
	SetAttribute(key string, value interface{})
}

// DAGBuilder constructs a directed acyclic graph from steps for execution planning.
//
// Implementations: internal/infrastructure/engine/dag_builder.go
//
// Example:
//
//	builder := engine.NewDAGBuilder()
//	graph, err := builder.Build(ctx, pipeline.Steps)
type DAGBuilder interface {
	// Build constructs a DAG from steps, validating dependencies and detecting cycles.
	//
	// Returns:
	//   - *Graph: Internal graph representation
	//   - error: Domain error:
	//     - ErrCodeDependency: Missing dependency or circular reference
	//     - ErrCodeValidation: Invalid step definitions
	Build(ctx context.Context, steps []Step) (*Graph, error)
}

// ExecutionPlanner generates an execution plan from a DAG.
//
// Implementations: internal/infrastructure/engine/planner.go
//
// Example:
//
//	planner := engine.NewPlanner()
//	plan, err := planner.GeneratePlan(ctx, graph)
type ExecutionPlanner interface {
	// GeneratePlan performs topological sort to produce level-based execution plan.
	//
	// Steps in the same level can execute in parallel (no dependencies between them).
	// Steps in level N may only depend on steps in levels < N.
	//
	// Returns:
	//   - *ExecutionPlan: Plan with level-based step ordering
	//   - error: Domain error if plan cannot be generated
	GeneratePlan(ctx context.Context, graph *Graph) (*ExecutionPlan, error)
}

// Graph represents a directed acyclic graph of step dependencies.
// Internal representation used by DAGBuilder and ExecutionPlanner.
type Graph struct {
	Nodes  map[string]*Node
	Levels [][]string
}

// Node represents a single step in the dependency graph.
type Node struct {
	StepID     string
	DependsOn  []string
	Dependents []string
}

// Types referenced in interfaces (defined in domain entities)
type Pipeline struct{}           // Defined in internal/domain/pipeline/pipeline.go
type Step struct{}               // Defined in internal/domain/pipeline/step.go
type ExecutionPlan struct{}      // Defined in internal/domain/pipeline/plan.go
type StepResult struct{}         // Defined in internal/domain/pipeline/result.go
type VerificationResult struct{} // Defined in internal/domain/pipeline/result.go
type ResultStatus string         // Defined in internal/domain/pipeline/result.go
