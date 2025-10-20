// Package engine builds and executes pipeline dependency graphs.
package engine

import (
	"context"
	"sort"
	"strings"

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
	active, err := collectActiveSteps(ctx, steps)
	if err != nil {
		return nil, err
	}

	indegree, adjacency, err := initializeGraph(ctx, active)
	if err != nil {
		return nil, err
	}

	if err := populateGraph(ctx, active, indegree, adjacency); err != nil {
		return nil, err
	}

	levels, processed, err := topologicalLevels(ctx, indegree, adjacency)
	if err != nil {
		return nil, err
	}

	if err := ensureNoCycles(indegree, active, processed); err != nil {
		return nil, err
	}

	return &pipeline.ExecutionPlan{
		Levels:     levels,
		TotalSteps: len(active),
	}, nil
}

func collectActiveSteps(ctx context.Context, steps []pipeline.Step) (map[string]pipeline.Step, error) {
	active := make(map[string]pipeline.Step)

	for idx, step := range steps {
		if err := contextCancelled(ctx, "collect_steps"); err != nil {
			return nil, err
		}

		if !step.Enabled {
			continue
		}

		if strings.TrimSpace(step.ID) == "" {
			return nil, pipeline.NewValidationError("step id must be set", map[string]interface{}{
				"step_index":  idx,
				"plugin_type": step.Type,
			})
		}

		if _, exists := active[step.ID]; exists {
			return nil, pipeline.NewDuplicateError(step.ID).WithContext(map[string]interface{}{
				"step_index":  idx,
				"plugin_type": step.Type,
			})
		}

		active[step.ID] = step
	}

	return active, nil
}

func initializeGraph(ctx context.Context, active map[string]pipeline.Step) (map[string]int, map[string][]string, error) {
	indegree := make(map[string]int, len(active))
	adjacency := make(map[string][]string, len(active))

	for id := range active {
		if err := contextCancelled(ctx, "init_graph"); err != nil {
			return nil, nil, err
		}

		indegree[id] = 0
	}

	return indegree, adjacency, nil
}

func populateGraph(ctx context.Context, active map[string]pipeline.Step, indegree map[string]int, adjacency map[string][]string) error {
	for id, step := range active {
		seen := make(map[string]struct{}, len(step.DependsOn))

		for _, dep := range step.DependsOn {
			if err := contextCancelled(ctx, "build_graph"); err != nil {
				return err
			}

			if _, duplicate := seen[dep]; duplicate {
				continue
			}

			seen[dep] = struct{}{}

			if dep == id {
				return pipeline.NewDependencyError("step cannot depend on itself", map[string]interface{}{"step_id": id})
			}

			if _, ok := active[dep]; !ok {
				return pipeline.NewDependencyError("dependency not found", map[string]interface{}{"step_id": id, "missing_dependency": dep})
			}

			indegree[id]++
			adjacency[dep] = append(adjacency[dep], id)
		}
	}

	return nil
}

func topologicalLevels(ctx context.Context, indegree map[string]int, adjacency map[string][]string) ([]pipeline.ExecutionLevel, int, error) {
	queue := zeroIndegreeQueue(indegree)
	levels := make([]pipeline.ExecutionLevel, 0)
	processed := 0

	for len(queue) > 0 {
		if err := contextCancelled(ctx, "topological_sort"); err != nil {
			return nil, 0, err
		}

		current := append([]string(nil), queue...)
		sort.Strings(current)

		level := pipeline.ExecutionLevel{
			Level:   len(levels),
			StepIDs: current,
		}
		levels = append(levels, level)

		next := make([]string, 0)

		for _, id := range current {
			if err := contextCancelled(ctx, "topological_sort"); err != nil {
				return nil, 0, err
			}

			processed++

			for _, dep := range adjacency[id] {
				if err := contextCancelled(ctx, "topological_sort"); err != nil {
					return nil, 0, err
				}

				indegree[dep]--
				if indegree[dep] == 0 {
					next = append(next, dep)
				}
			}
		}

		sort.Strings(next)
		queue = next
	}

	return levels, processed, nil
}

func ensureNoCycles(indegree map[string]int, active map[string]pipeline.Step, processed int) error {
	if processed == len(active) {
		return nil
	}

	cycle := make([]string, 0, len(active)-processed)
	for id, step := range active {
		if indegree[id] > 0 {
			cycle = append(cycle, step.ID)
		}
	}

	sort.Strings(cycle)

	return pipeline.NewCycleError(cycle)
}

func zeroIndegreeQueue(indegree map[string]int) []string {
	queue := make([]string, 0)

	for id, deg := range indegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	sort.Strings(queue)

	return queue
}

func contextCancelled(ctx context.Context, phase string) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return pipeline.NewDomainError(pipeline.ErrCodeCancelled, "build cancelled", ctxErr, map[string]interface{}{"phase": phase})
	}

	return nil
}
