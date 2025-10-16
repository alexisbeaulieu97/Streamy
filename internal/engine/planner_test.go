package engine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
)

func TestGeneratePlan(t *testing.T) {
	t.Parallel()

	steps := []config.Step{
		stepWithConfig(t, config.Step{ID: "install_git", Type: "package", Enabled: true}, config.PackageStep{Packages: []string{"git"}}),
		stepWithConfig(t, config.Step{ID: "install_curl", Type: "package", Enabled: true}, config.PackageStep{Packages: []string{"curl"}}),
		stepWithConfig(t, config.Step{ID: "clone_repo", Type: "repo", Enabled: true, DependsOn: []string{"install_git"}}, config.RepoStep{URL: "https://example.com/repo.git", Destination: "/tmp/repo"}),
		stepWithConfig(t, config.Step{ID: "configure", Type: "command", Enabled: true, DependsOn: []string{"clone_repo", "install_curl"}}, config.CommandStep{Command: "./configure"}),
	}

	graph, err := BuildDAG(steps)
	require.NoError(t, err)

	plan, err := GeneratePlan(graph)
	require.NoError(t, err)
	require.NotNil(t, plan)

	require.Len(t, plan.Levels, 3)
	require.ElementsMatch(t, []string{"install_git", "install_curl"}, plan.Levels[0].StepIDs)
	require.ElementsMatch(t, []string{"clone_repo"}, plan.Levels[1].StepIDs)
	require.ElementsMatch(t, []string{"configure"}, plan.Levels[2].StepIDs)

	for _, level := range plan.Levels {
		require.Greater(t, level.EstimatedDuration, time.Duration(0))
	}
}

func TestGeneratePlan_String(t *testing.T) {
	t.Parallel()

	steps := []config.Step{
		stepWithConfig(t, config.Step{ID: "a", Type: "command", Enabled: true}, config.CommandStep{Command: "echo a"}),
		stepWithConfig(t, config.Step{ID: "b", Type: "command", Enabled: true, DependsOn: []string{"a"}}, config.CommandStep{Command: "echo b"}),
	}

	graph, err := BuildDAG(steps)
	require.NoError(t, err)

	plan, err := GeneratePlan(graph)
	require.NoError(t, err)

	summary := plan.String()
	require.Contains(t, summary, "Level 0")
	require.Contains(t, summary, "a")
	require.Contains(t, summary, "Level 1")
	require.Contains(t, summary, "b")
}
