package engine

import (
	"fmt"
	"sort"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

// Node represents a vertex in the execution DAG.
type Node struct {
	ID         string
	Step       *config.Step
	DependsOn  []*Node
	Dependents []*Node
}

// Graph encapsulates the DAG structure and topological levels.
type Graph struct {
	Nodes  map[string]*Node
	Levels [][]string
}

// NewGraph creates an empty graph.
func NewGraph() *Graph {
	return &Graph{Nodes: make(map[string]*Node)}
}

// AddNode inserts a step as a vertex in the graph.
func (g *Graph) AddNode(step *config.Step) (*Node, error) {
	if step == nil {
		return nil, streamyerrors.NewExecutionError("", fmt.Errorf("step cannot be nil"))
	}

	if g.Nodes == nil {
		g.Nodes = make(map[string]*Node)
	}

	if _, exists := g.Nodes[step.ID]; exists {
		return nil, streamyerrors.NewValidationError("steps", fmt.Sprintf("duplicate step id %q", step.ID), nil)
	}

	node := &Node{ID: step.ID, Step: step}
	g.Nodes[step.ID] = node
	return node, nil
}

// AddEdge connects dependency relationship between nodes.
func (g *Graph) AddEdge(from, to string) error {
	source, ok := g.Nodes[from]
	if !ok {
		return streamyerrors.NewValidationError("steps", fmt.Sprintf("unknown dependency %q", from), nil)
	}

	target, ok := g.Nodes[to]
	if !ok {
		return streamyerrors.NewValidationError("steps", fmt.Sprintf("unknown dependency target %q", to), nil)
	}

	source.Dependents = append(source.Dependents, target)
	target.DependsOn = append(target.DependsOn, source)
	return nil
}

// TopologicalSort computes the DAG levels using Kahn's algorithm.
func (g *Graph) TopologicalSort() error {
	indegree := make(map[string]int, len(g.Nodes))
	for id := range g.Nodes {
		indegree[id] = 0
	}

	for _, node := range g.Nodes {
		for _, dep := range node.Dependents {
			indegree[dep.ID]++
		}
	}

	var queue []string
	for id, degree := range indegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue)

	processed := 0
	var levels [][]string

	for len(queue) > 0 {
		currentLevel := queue
		sort.Strings(currentLevel)
		levels = append(levels, append([]string(nil), currentLevel...))

		var nextLevel []string
		for _, id := range currentLevel {
			processed++
			node := g.Nodes[id]
			for _, dependent := range node.Dependents {
				indegree[dependent.ID]--
				if indegree[dependent.ID] == 0 {
					nextLevel = append(nextLevel, dependent.ID)
				}
			}
		}

		sort.Strings(nextLevel)
		queue = nextLevel
	}

	if processed != len(g.Nodes) {
		return streamyerrors.NewValidationError("steps", "cycle detected while sorting graph", nil)
	}

	g.Levels = levels
	return nil
}
