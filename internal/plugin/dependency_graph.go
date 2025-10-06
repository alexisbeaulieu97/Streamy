package plugin

import (
	"fmt"
	"sort"
)

// DependencyGraph tracks plugin relationships for validation and initialization ordering.
type DependencyGraph struct {
	nodes    map[string]struct{}
	incoming map[string]map[string]struct{}
	outgoing map[string]map[string]struct{}
}

// NewDependencyGraph creates an empty dependency graph.
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes:    make(map[string]struct{}),
		incoming: make(map[string]map[string]struct{}),
		outgoing: make(map[string]map[string]struct{}),
	}
}

// AddNode ensures the plugin exists within the graph.
func (g *DependencyGraph) AddNode(name string) {
	if g.nodes == nil {
		g.nodes = make(map[string]struct{})
	}
	if g.incoming == nil {
		g.incoming = make(map[string]map[string]struct{})
	}
	if g.outgoing == nil {
		g.outgoing = make(map[string]map[string]struct{})
	}

	if _, exists := g.nodes[name]; exists {
		return
	}

	g.nodes[name] = struct{}{}
	g.incoming[name] = make(map[string]struct{})
	g.outgoing[name] = make(map[string]struct{})
}

// AddEdge records a dependency edge between two plugins.
func (g *DependencyGraph) AddEdge(dependent, dependency string) {
	g.AddNode(dependent)
	g.AddNode(dependency)

	g.outgoing[dependent][dependency] = struct{}{}
	g.incoming[dependency][dependent] = struct{}{}
}

// DetectCycles returns one cycle if present or nil when the graph is acyclic.
func (g *DependencyGraph) DetectCycles() ([]string, error) {
	visited := make(map[string]bool)
	stack := make(map[string]bool)
	path := []string{}

	var cycle []string
	var dfs func(node string) bool

	dfs = func(node string) bool {
		visited[node] = true
		stack[node] = true
		path = append(path, node)

		for dependency := range g.outgoing[node] {
			if !visited[dependency] {
				if dfs(dependency) {
					return true
				}
			} else if stack[dependency] {
				// Extract cycle starting from dependency position in path.
				idx := len(path) - 1
				for idx >= 0 && path[idx] != dependency {
					idx--
				}
				if idx >= 0 {
					cycle = append([]string{}, path[idx:]...)
					return true
				}
			}
		}

		stack[node] = false
		path = path[:len(path)-1]
		return false
	}

	// Evaluate nodes in deterministic order for consistent results.
	nodes := g.sortedNodes()
	for _, node := range nodes {
		if !visited[node] {
			if dfs(node) {
				break
			}
		}
	}

	return cycle, nil
}

// TopologicalSort returns nodes in dependency order (dependencies first).
func (g *DependencyGraph) TopologicalSort() ([]string, error) {
	remaining := make(map[string]int, len(g.nodes))
	for node := range g.nodes {
		remaining[node] = len(g.outgoing[node])
	}

	queue := make([]string, 0, len(g.nodes))
	for node, deps := range remaining {
		if deps == 0 {
			queue = append(queue, node)
		}
	}
	sort.Strings(queue)

	result := make([]string, 0, len(g.nodes))
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		dependents := g.GetDependents(current)
		for _, dependent := range dependents {
			remaining[dependent]--
			if remaining[dependent] == 0 {
				queue = append(queue, dependent)
				sort.Strings(queue)
			}
		}
	}

	if len(result) != len(g.nodes) {
		cycle, err := g.DetectCycles()
		if err != nil {
			return nil, err
		}
		if len(cycle) > 0 {
			return nil, ErrCircularDependency{Cycle: cycle}
		}
		return nil, fmt.Errorf("dependency graph contains unresolved nodes")
	}

	return result, nil
}

// GetDependencies returns the list of dependencies for a node.
func (g *DependencyGraph) GetDependencies(node string) []string {
	depsMap, ok := g.outgoing[node]
	if !ok {
		return nil
	}
	deps := make([]string, 0, len(depsMap))
	for dep := range depsMap {
		deps = append(deps, dep)
	}
	sort.Strings(deps)
	return deps
}

// GetDependents returns all nodes that rely on the supplied node.
func (g *DependencyGraph) GetDependents(node string) []string {
	dependentsMap, ok := g.incoming[node]
	if !ok {
		return nil
	}
	dependents := make([]string, 0, len(dependentsMap))
	for dep := range dependentsMap {
		dependents = append(dependents, dep)
	}
	sort.Strings(dependents)
	return dependents
}

// HasNode reports if the node exists in the graph.
func (g *DependencyGraph) HasNode(node string) bool {
	if g == nil {
		return false
	}
	_, ok := g.nodes[node]
	return ok
}

func (g *DependencyGraph) sortedNodes() []string {
	nodes := make([]string, 0, len(g.nodes))
	for node := range g.nodes {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)
	return nodes
}
