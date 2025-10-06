package plugin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDependencyGraphDetectCycles(t *testing.T) {
	graph := NewDependencyGraph()
	graph.AddEdge("A", "B")
	graph.AddEdge("B", "C")
	graph.AddEdge("C", "A")

	cycle, err := graph.DetectCycles()
	require.NoError(t, err)
	require.Len(t, cycle, 3)
	require.ElementsMatch(t, []string{"A", "B", "C"}, cycle)

	acyclic := NewDependencyGraph()
	acyclic.AddEdge("A", "B")
	acyclic.AddEdge("B", "C")

	none, err := acyclic.DetectCycles()
	require.NoError(t, err)
	require.Nil(t, none)
}

func TestDependencyGraphTopologicalSort(t *testing.T) {
	graph := NewDependencyGraph()
	graph.AddEdge("B", "A")
	graph.AddEdge("C", "B")
	graph.AddEdge("C", "A")

	order, err := graph.TopologicalSort()
	require.NoError(t, err)
	require.Equal(t, []string{"A", "B", "C"}, order)

	cyclic := NewDependencyGraph()
	cyclic.AddEdge("A", "B")
	cyclic.AddEdge("B", "A")

	_, err = cyclic.TopologicalSort()
	require.Error(t, err)
	var cycle ErrCircularDependency
	require.ErrorAs(t, err, &cycle)
	require.NotEmpty(t, cycle.Cycle)
}

func TestDependencyGraphUtilities(t *testing.T) {
	graph := NewDependencyGraph()
	graph.AddEdge("shell_profile", "line_in_file")
	graph.AddEdge("shell_profile", "file_utils")
	graph.AddEdge("line_in_file", "file_utils")

	deps := graph.GetDependencies("shell_profile")
	require.Equal(t, []string{"file_utils", "line_in_file"}, deps)

	dependents := graph.GetDependents("file_utils")
	require.Equal(t, []string{"line_in_file", "shell_profile"}, dependents)

	require.True(t, graph.HasNode("shell_profile"))
	require.False(t, graph.HasNode("missing"))
}
