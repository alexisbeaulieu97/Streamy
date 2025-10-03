package engine

import (
	"fmt"
	"sort"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

// BuildDAG constructs the execution graph from the provided steps.
func BuildDAG(steps []config.Step) (*Graph, error) {
	graph := NewGraph()
	stepMap := make(map[string]*config.Step, len(steps))

	for i := range steps {
		step := &steps[i]
		if !step.Enabled {
			continue
		}
		if _, err := graph.AddNode(step); err != nil {
			return nil, err
		}
		stepMap[step.ID] = step
	}

	for _, step := range steps {
		if !step.Enabled {
			continue
		}
		if len(step.DependsOn) == 0 {
			continue
		}
		for _, dependency := range step.DependsOn {
			if _, ok := stepMap[dependency]; !ok {
				return nil, streamyerrors.NewValidationError("steps", fmt.Sprintf("step %q depends on unknown step %q", step.ID, dependency), nil)
			}
			if err := graph.AddEdge(dependency, step.ID); err != nil {
				return nil, err
			}
		}
	}

	if err := graph.TopologicalSort(); err != nil {
		return nil, err
	}

	// Ensure every step appears in levels even when no dependencies.
	ensureLevelsContainAll(graph, steps)

	return graph, nil
}

func ensureLevelsContainAll(graph *Graph, steps []config.Step) {
	seen := make(map[string]struct{})
	for _, level := range graph.Levels {
		for _, id := range level {
			seen[id] = struct{}{}
		}
	}

	var missing []string
	for _, step := range steps {
		if !step.Enabled {
			continue
		}
		if _, ok := seen[step.ID]; !ok {
			missing = append(missing, step.ID)
		}
	}

	if len(missing) == 0 {
		return
	}

	sort.Strings(missing)
	graph.Levels = append([][]string{missing}, graph.Levels...)
}
