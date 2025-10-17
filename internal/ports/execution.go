package ports

import (
	"context"

	domain "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
)

// PluginExecutor drives step execution while enforcing ordering guarantees,
// parallelism constraints, context cancellation, and domain error semantics.
// Implementations must:
//   - Execute steps level-by-level, running siblings concurrently (see FR-017).
//   - Respect ctx cancellation between levels and before dispatching each step.
//   - Translate infrastructure failures into domain.ErrCodeExecution,
//     domain.ErrCodeTimeout, or domain.ErrCodeCancelled as appropriate.
//   - Emit observability signals via injected ports (metrics, tracing, events).
type PluginExecutor interface {
	// Execute applies all steps in the plan and returns results in execution order.
	Execute(ctx context.Context, plan *domain.ExecutionPlan, pipeline *domain.Pipeline) ([]domain.StepResult, error)

	// Verify performs a read-only evaluation across all steps, never mutating
	// system state. Implementations typically delegate to Plugin.Evaluate.
	Verify(ctx context.Context, pipeline *domain.Pipeline) ([]domain.VerificationResult, error)
}

// DAGBuilder constructs a dependency-aware execution plan from raw steps. It
// is responsible for cycle detection, duplicate guarding, and ensuring only
// enabled steps are scheduled. Returned plans must satisfy Pipeline.Validate()
// invariants.
type DAGBuilder interface {
	Build(ctx context.Context, steps []domain.Step) (*domain.ExecutionPlan, error)
}

// ExecutionPlanner transforms a dependency graph into an execution plan. This
// separation allows alternative planners (e.g., different batching heuristics)
// while reusing DAG builders.
type ExecutionPlanner interface {
	GeneratePlan(ctx context.Context, graph *ExecutionGraph) (*domain.ExecutionPlan, error)
}

// ExecutionGraph represents a directed acyclic graph of step dependencies.
// DAG builders populate the structure, planners consume it.
type ExecutionGraph struct {
	Nodes map[string]*ExecutionNode
	Roots []string
}

// ExecutionNode captures the relationship metadata for a single step within an
// execution graph.
type ExecutionNode struct {
	Step       domain.Step
	DependsOn  []string
	Dependents []string
}
