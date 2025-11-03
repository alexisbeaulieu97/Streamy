package registry

import (
	"testing"

	assert "github.com/stretchr/testify/assert"
	require "github.com/stretchr/testify/require"
)

func TestDependencyGraphBuildsAndSorts(t *testing.T) {
	t.Parallel()

	reg := newInMemoryRegistry(t)

	require.NoError(t, reg.Add(Pipeline{
		ID:           "base@1.0",
		Name:         "Base",
		Dependencies: nil,
	}))

	require.NoError(t, reg.Add(Pipeline{
		ID:           "network@1.0",
		Name:         "Network",
		Dependencies: nil,
	}))

	require.NoError(t, reg.Add(Pipeline{
		ID:           "db@1.0",
		Name:         "Database",
		Dependencies: []string{"base@1.0"},
	}))

	require.NoError(t, reg.Add(Pipeline{
		ID:           "infra@1.0",
		Name:         "Infrastructure",
		Dependencies: []string{"network@1.0", "db@1.0"},
	}))

	graph, err := NewDependencyGraph(reg, "infra@1.0")
	require.NoError(t, err)
	require.NotNil(t, graph)

	assert.ElementsMatch(t, []string{"db@1.0", "network@1.0"}, graph.Dependencies("infra@1.0"))
	assert.ElementsMatch(t, []string{"base@1.0"}, graph.Dependencies("db@1.0"))

	levels, err := graph.TopologicalSort()
	require.NoError(t, err)
	require.Equal(t, [][]string{
		{"base@1.0", "network@1.0"},
		{"db@1.0"},
		{"infra@1.0"},
	}, levels)

	dependents := graph.GetTransitiveDependents("base@1.0")
	assert.Equal(t, []string{"db@1.0", "infra@1.0"}, dependents)
}

func TestDependencyGraphMissingDependency(t *testing.T) {
	t.Parallel()

	reg := newInMemoryRegistry(t)

	require.NoError(t, reg.Add(Pipeline{
		ID:           "app@1.0",
		Dependencies: []string{"missing@1.0"},
	}))

	graph, err := NewDependencyGraph(reg, "app@1.0")
	require.Error(t, err)
	assert.Nil(t, graph)
	assert.ErrorIs(t, err, ErrMissingDependency)
}

func TestDependencyGraphDetectsCycle(t *testing.T) {
	t.Parallel()

	reg := newInMemoryRegistry(t)

	require.NoError(t, reg.Add(Pipeline{
		ID:           "frontend@1.0",
		Dependencies: []string{"backend@1.0"},
	}))

	require.NoError(t, reg.Add(Pipeline{
		ID:           "backend@1.0",
		Dependencies: []string{"frontend@1.0"},
	}))

	graph, err := NewDependencyGraph(reg, "frontend@1.0")
	require.Error(t, err)
	assert.Nil(t, graph)
	assert.ErrorIs(t, err, ErrCircularDependency)

	// Manually construct graph to exercise DetectCycle helper
	cyclic := &DependencyGraph{
		edges: map[string][]string{
			"frontend@1.0": {"backend@1.0"},
			"backend@1.0":  {"frontend@1.0"},
		},
		dependents: map[string][]string{
			"backend@1.0":  {"frontend@1.0"},
			"frontend@1.0": {"backend@1.0"},
		},
		nodes: map[string]struct{}{
			"frontend@1.0": {},
			"backend@1.0":  {},
		},
	}

	path, cycleErr := cyclic.DetectCycle()
	require.Error(t, cycleErr)
	assert.ErrorIs(t, cycleErr, ErrCircularDependency)
	assert.NotEmpty(t, path)
}

func newInMemoryRegistry(t *testing.T) *Registry {
	t.Helper()

	registryPath := t.TempDir() + "/registry.json"
	reg, err := NewRegistry(registryPath)
	require.NoError(t, err)

	return reg
}
