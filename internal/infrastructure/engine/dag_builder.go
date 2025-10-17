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
			return nil, &pipeline.DomainError{Code: pipeline.ErrCodeCancelled, Message: "build cancelled", Cause: ctxErr}
		}
		if !step.Enabled {
			continue
		}
		active[step.ID] = step
	}

	indegree := make(map[string]int, len(active))
	adjacency := make(map[string][]string, len(active))

	for id := range active {
		indegree[id] = 0
	}

	for id, step := range active {
		for _, dep := range step.DependsOn {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, &pipeline.DomainError{Code: pipeline.ErrCodeCancelled, Message: "build cancelled", Cause: ctxErr}
			}
			if dep == id {
				return nil, &pipeline.DomainError{
					Code:    pipeline.ErrCodeDependency,
					Message: "step cannot depend on itself",
					Context: map[string]interface{}{"step_id": id},
				}
			}
			if _, ok := active[dep]; !ok {
				return nil, &pipeline.DomainError{
					Code:    pipeline.ErrCodeDependency,
					Message: "dependency not found",
					Context: map[string]interface{}{"step_id": id, "missing_dependency": dep},
				}
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
			return nil, &pipeline.DomainError{Code: pipeline.ErrCodeCancelled, Message: "build cancelled", Cause: ctxErr}
		}
		current := append([]string(nil), queue...)
		sort.Strings(current)
		level := pipeline.ExecutionLevel{Level: len(levels), StepIDs: current}
		levels = append(levels, level)

		next := make([]string, 0)
		for _, id := range current {
			processed++
			for _, dep := range adjacency[id] {
				indegree[dep]--
				if indegree[dep] == 0 {
					next = append(next, dep)
				}
			}
		}
		sort.Strings(next)
		queue = next
	}

	if processed != len(active) {
		return nil, &pipeline.DomainError{Code: pipeline.ErrCodeCycle, Message: "circular dependency detected"}
	}

	plan := &pipeline.ExecutionPlan{
		Levels:     levels,
		TotalSteps: len(active),
	}

	return plan, nil
}
