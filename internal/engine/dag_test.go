package engine

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

func TestBuildDAG_GeneratesLevels(t *testing.T) {
	t.Parallel()

	steps := []config.Step{
		stepWithConfig(t, config.Step{ID: "install_git", Type: "package", Enabled: true}, config.PackageStep{Packages: []string{"git"}}),
		stepWithConfig(t, config.Step{ID: "clone_repo", Type: "repo", Enabled: true, DependsOn: []string{"install_git"}}, config.RepoStep{URL: "https://example.com/repo.git", Destination: "/tmp/repo"}),
		stepWithConfig(t, config.Step{ID: "configure", Type: "command", Enabled: true, DependsOn: []string{"clone_repo"}}, config.CommandStep{Command: "./setup.sh"}),
	}

	graph, err := BuildDAG(steps)
	require.NoError(t, err)
	require.NotNil(t, graph)

	require.Len(t, graph.Levels, 3)
	require.ElementsMatch(t, []string{"install_git"}, graph.Levels[0])
	require.ElementsMatch(t, []string{"clone_repo"}, graph.Levels[1])
	require.ElementsMatch(t, []string{"configure"}, graph.Levels[2])
}

func TestBuildDAG_AllowsParallelSteps(t *testing.T) {
	t.Parallel()

	steps := []config.Step{
		stepWithConfig(t, config.Step{ID: "install_git", Type: "package", Enabled: true}, config.PackageStep{Packages: []string{"git"}}),
		stepWithConfig(t, config.Step{ID: "install_curl", Type: "package", Enabled: true}, config.PackageStep{Packages: []string{"curl"}}),
		stepWithConfig(t, config.Step{ID: "clone_repo", Type: "repo", Enabled: true, DependsOn: []string{"install_git", "install_curl"}}, config.RepoStep{URL: "https://example.com/repo.git", Destination: "/tmp/repo"}),
	}

	graph, err := BuildDAG(steps)
	require.NoError(t, err)

	require.Len(t, graph.Levels, 2)
	require.ElementsMatch(t, []string{"install_git", "install_curl"}, graph.Levels[0])
	require.ElementsMatch(t, []string{"clone_repo"}, graph.Levels[1])
}

func TestBuildDAG_DetectsCycles(t *testing.T) {
	t.Parallel()

	steps := []config.Step{
		stepWithConfig(t, config.Step{ID: "a", Type: "command", Enabled: true, DependsOn: []string{"c"}}, config.CommandStep{Command: "echo a"}),
		stepWithConfig(t, config.Step{ID: "b", Type: "command", Enabled: true, DependsOn: []string{"a"}}, config.CommandStep{Command: "echo b"}),
		stepWithConfig(t, config.Step{ID: "c", Type: "command", Enabled: true, DependsOn: []string{"b"}}, config.CommandStep{Command: "echo c"}),
	}

	graph, err := BuildDAG(steps)
	require.Error(t, err)
	require.Nil(t, graph)

	var validationErr *streamyerrors.ValidationError
	require.ErrorAs(t, err, &validationErr)
	require.Contains(t, validationErr.Message, "cycle")
}

func TestBuildDAG_TopologicalOrderIsStable(t *testing.T) {
	t.Parallel()

	steps := []config.Step{
		stepWithConfig(t, config.Step{ID: "a", Type: "command", Enabled: true}, config.CommandStep{Command: "echo a"}),
		stepWithConfig(t, config.Step{ID: "b", Type: "command", Enabled: true}, config.CommandStep{Command: "echo b"}),
	}

	graph, err := BuildDAG(steps)
	require.NoError(t, err)

	require.ElementsMatch(t, []string{"a", "b"}, graph.Levels[0])
}

func TestBuildDAG_SkipsDisabledSteps(t *testing.T) {
	t.Parallel()

	steps := []config.Step{
		stepWithConfig(t, config.Step{ID: "disabled", Type: "command", Enabled: false}, config.CommandStep{Command: "echo skip"}),
		stepWithConfig(t, config.Step{ID: "active", Type: "command", Enabled: true}, config.CommandStep{Command: "echo run"}),
	}

	graph, err := BuildDAG(steps)
	require.NoError(t, err)
	require.Len(t, graph.Levels, 1)
	require.ElementsMatch(t, []string{"active"}, graph.Levels[0])
}

func TestBuildDAG_ErrorsWhenDependencyIsDisabled(t *testing.T) {
	t.Parallel()

	steps := []config.Step{
		stepWithConfig(t, config.Step{ID: "disabled", Type: "command", Enabled: false}, config.CommandStep{Command: "echo skip"}),
		stepWithConfig(t, config.Step{ID: "active", Type: "command", Enabled: true, DependsOn: []string{"disabled"}}, config.CommandStep{Command: "echo run"}),
	}

	graph, err := BuildDAG(steps)
	require.Error(t, err)
	require.Nil(t, graph)
}

func TestBuildDAG_ErrorsWhenDependencyMissing(t *testing.T) {
	t.Parallel()

	steps := []config.Step{
		stepWithConfig(t, config.Step{ID: "first", Type: "command", Enabled: true, DependsOn: []string{"missing"}}, config.CommandStep{Command: "echo"}),
	}

	graph, err := BuildDAG(steps)
	require.Error(t, err)
	require.Nil(t, graph)
}
