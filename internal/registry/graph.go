package registry

import (
	"errors"
	"fmt"
	"slices"
)

var (
	// ErrCircularDependency indicates that the dependency graph contains a cycle.
	ErrCircularDependency = errors.New("circular dependency")
	// ErrMissingDependency indicates that a referenced pipeline was not found in the registry.
	ErrMissingDependency = errors.New("missing dependency")
)

// DependencyGraph represents the directed acyclic graph of pipelines reachable from a root pipeline.
type DependencyGraph struct {
	Root       string
	edges      map[string][]string
	dependents map[string][]string
	nodes      map[string]struct{}
}

// NewDependencyGraph builds a dependency graph rooted at the provided pipeline identifier.
func NewDependencyGraph(reg *Registry, rootPipelineID string) (*DependencyGraph, error) {
	if rootPipelineID == "" {
		return nil, fmt.Errorf("root pipeline id cannot be empty")
	}

	pipelines := reg.PipelinesByID()

	graph := &DependencyGraph{
		Root:       rootPipelineID,
		edges:      make(map[string][]string),
		dependents: make(map[string][]string),
		nodes:      make(map[string]struct{}),
	}

	visiting := make(map[string]bool)
	if err := graph.build(pipelines, rootPipelineID, visiting); err != nil {
		return nil, err
	}

	for id := range graph.edges {
		slices.Sort(graph.edges[id])
	}

	for id := range graph.dependents {
		slices.Sort(graph.dependents[id])
	}

	return graph, nil
}

func (g *DependencyGraph) build(pipelines map[string]Pipeline, pipelineID string, visiting map[string]bool) error {
	pipeline, ok := pipelines[pipelineID]
	if !ok {
		return fmt.Errorf("%w: %s", ErrMissingDependency, pipelineID)
	}

	if visiting[pipelineID] {
		return fmt.Errorf("%w: %s", ErrCircularDependency, pipelineID)
	}

	if _, seen := g.nodes[pipelineID]; seen {
		return nil
	}

	visiting[pipelineID] = true
	defer delete(visiting, pipelineID)

	g.nodes[pipelineID] = struct{}{}

	dependencies := append([]string(nil), pipeline.Dependencies...)
	g.edges[pipelineID] = dependencies

	for _, depID := range dependencies {
		g.dependents[depID] = append(g.dependents[depID], pipelineID)

		if err := g.build(pipelines, depID, visiting); err != nil {
			return err
		}
	}

	return nil
}

// TopologicalSort returns pipeline identifiers grouped by execution level using Kahn's algorithm.
func (g *DependencyGraph) TopologicalSort() ([][]string, error) {
	inDegree := make(map[string]int, len(g.nodes))

	for node := range g.nodes {
		inDegree[node] = len(g.edges[node])
	}

	var levels [][]string
	for len(inDegree) > 0 {
		var level []string

		for node, degree := range inDegree {
			if degree == 0 {
				level = append(level, node)
			}
		}

		if len(level) == 0 {
			return nil, fmt.Errorf("%w: unresolved cycle in dependency graph", ErrCircularDependency)
		}

		slices.Sort(level)
		levels = append(levels, level)

		for _, node := range level {
			delete(inDegree, node)

			dependents := g.dependents[node]

			for _, dependent := range dependents {
				if _, ok := inDegree[dependent]; ok {
					inDegree[dependent]--
				}
			}
		}
	}

	return levels, nil
}

// GetTransitiveDependents returns all pipelines that transitively depend on the provided pipeline.
func (g *DependencyGraph) GetTransitiveDependents(pipelineID string) []string {
	visited := make(map[string]struct{})

	var dfs func(string)

	dfs = func(id string) {
		dependents := g.dependents[id]

		for _, dependent := range dependents {
			if _, ok := visited[dependent]; ok {
				continue
			}

			visited[dependent] = struct{}{}
			dfs(dependent)
		}
	}

	dfs(pipelineID)

	results := make([]string, 0, len(visited))
	for id := range visited {
		results = append(results, id)
	}

	slices.Sort(results)

	return results
}

// DetectCycle attempts to find a cycle in the graph and returns the cycle path if one exists.
func (g *DependencyGraph) DetectCycle() ([]string, error) {
	visiting := make(map[string]bool)
	visited := make(map[string]bool)
	parent := make(map[string]string)

	nodes := make([]string, 0, len(g.nodes))
	for node := range g.nodes {
		nodes = append(nodes, node)
	}
	slices.Sort(nodes)

	for _, node := range nodes {
		if visited[node] {
			continue
		}

		if path := g.detectCycleFrom(node, visiting, visited, parent); path != nil {
			return path, ErrCircularDependency
		}
	}

	return nil, nil
}

func (g *DependencyGraph) detectCycleFrom(node string, visiting, visited map[string]bool, parent map[string]string) []string {
	visiting[node] = true

	for _, dep := range g.edges[node] {
		if visiting[dep] {
			return g.buildCyclePath(node, dep, parent)
		}

		if !visited[dep] {
			parent[dep] = node
			if path := g.detectCycleFrom(dep, visiting, visited, parent); path != nil {
				return path
			}
		}
	}

	visiting[node] = false
	visited[node] = true

	return nil
}

func (g *DependencyGraph) buildCyclePath(start, end string, parent map[string]string) []string {
	path := []string{start}

	for node := parent[start]; node != ""; node = parent[node] {
		path = append(path, node)
		if node == end {
			break
		}
	}

	slices.Reverse(path)
	path = append(path, end)

	return path
}

// Dependencies returns a copy of the dependencies registered for the provided pipeline identifier.
func (g *DependencyGraph) Dependencies(pipelineID string) []string {
	deps := g.edges[pipelineID]
	return append([]string(nil), deps...)
}

// Nodes returns the list of pipeline identifiers present in the graph.
func (g *DependencyGraph) Nodes() []string {
	nodes := make([]string, 0, len(g.nodes))
	for id := range g.nodes {
		nodes = append(nodes, id)
	}

	slices.Sort(nodes)

	return nodes
}
