package engine

import (
	"context"
	"sort"

	"github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
)

// DAGBuilder implements the ports.DAGBuilder interface by constructing
// execution plans from domain steps using a topological sort.
type DAGBuilder struct{}

// NewDAGBuilder creates a DAGBuilder instance.
func NewDAGBuilder() *DAGBuilder {
	return &DAGBuilder{}
}

// Build constructs a level-based execution plan for the provided steps.
func (b *DAGBuilder) Build(ctx context.Context, steps []pipeline.Step) (*pipeline.ExecutionPlan, error) {
	active := make(map[string]pipeline.Step)
	for _, step := range steps {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, pipeline.NewDomainError(pipeline.ErrCodeCancelled, "build cancelled", ctxErr, map[string]interface{}{"phase": "collect_steps"})
		}
		if !step.Enabled {
			continue
		}
		active[step.ID] = step
	}

	indegree := make(map[string]int, len(active))
	adjacency := make(map[string][]string, len(active))

	for id := range active {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, pipeline.NewDomainError(pipeline.ErrCodeCancelled, "build cancelled", ctxErr, map[string]interface{}{"phase": "init_graph"})
		}
		indegree[id] = 0
	}

	for id, step := range active {
		for _, dep := range step.DependsOn {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, pipeline.NewDomainError(pipeline.ErrCodeCancelled, "build cancelled", ctxErr, map[string]interface{}{"phase": "build_graph"})
			}
			if dep == id {
				return nil, pipeline.NewDependencyError("step cannot depend on itself", map[string]interface{}{"step_id": id})
			}
			if _, ok := active[dep]; !ok {
				return nil, pipeline.NewDependencyError("dependency not found", map[string]interface{}{"step_id": id, "missing_dependency": dep})
			}
			indegree[id]++
			adjacency[dep] = append(adjacency[dep], id)
		}
	}

	var queue []string
	for id, deg := range indegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue)

	processed := 0
	levels := make([]pipeline.ExecutionLevel, 0)

	for len(queue) > 0 {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, pipeline.NewDomainError(pipeline.ErrCodeCancelled, "build cancelled", ctxErr, map[string]interface{}{"phase": "topological_sort"})
		}
		current := append([]string(nil), queue...)
		sort.Strings(current)
		level := pipeline.ExecutionLevel{Level: len(levels), StepIDs: current}
		levels = append(levels, level)

		next := make([]string, 0)
		for _, id := range current {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, pipeline.NewDomainError(pipeline.ErrCodeCancelled, "build cancelled", ctxErr, map[string]interface{}{"phase": "topological_sort"})
			}
			processed++
			for _, dep := range adjacency[id] {
				if ctxErr := ctx.Err(); ctxErr != nil {
					return nil, pipeline.NewDomainError(pipeline.ErrCodeCancelled, "build cancelled", ctxErr, map[string]interface{}{"phase": "topological_sort"})
				}
				indegree[dep]--
				if indegree[dep] == 0 {
					next = append(next, dep)
				}
			}
		}
		sort.Strings(next)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, pipeline.NewDomainError(pipeline.ErrCodeCancelled, "build cancelled", ctxErr, map[string]interface{}{"phase": "topological_sort"})
		}
		queue = next
	}

	if processed != len(active) {
		cycle := make([]string, 0, len(active)-processed)
		for id, step := range active {
			if indegree[id] > 0 {
				cycle = append(cycle, step.ID)
			}
		}
		sort.Strings(cycle)
		return nil, pipeline.NewCycleError(cycle)
	}

	plan := &pipeline.ExecutionPlan{
		Levels:     levels,
		TotalSteps: len(active),
	}

	return plan, nil
}
