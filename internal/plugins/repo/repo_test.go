package repoplugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
)

func TestRepoPlugin_Metadata(t *testing.T) {
	t.Parallel()

	p := New()
	meta := p.PluginMetadata()

	require.NotEmpty(t, meta.Name)
	require.NotEmpty(t, meta.Version)
	require.Equal(t, "repo", meta.Name)
}

func TestRepoPlugin_Schema(t *testing.T) {
	t.Parallel()

	p := New()
	schema := p.Schema()

	require.NotNil(t, schema)
	_, ok := schema.(config.RepoStep)
	require.True(t, ok, "schema should be of type RepoStep")
}

func TestRepoPlugin_ApplyClonesRepository(t *testing.T) {
	source := initGitRepo(t)
	dest := filepath.Join(t.TempDir(), "clone")

	p := New()

	step := &config.Step{
		ID:   "clone_repo",
		Type: "repo",
		Repo: &config.RepoStep{
			URL:         source,
			Destination: dest,
		},
	}

	// First evaluate
	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, evalResult.RequiresAction)

	// Then apply
	result, err := p.Apply(context.Background(), evalResult, step)
	require.NoError(t, err)
	require.Equal(t, step.ID, result.StepID)
	require.Equal(t, model.StatusSuccess, result.Status)

	contents, err := os.ReadFile(filepath.Join(dest, "README.md"))
	require.NoError(t, err)
	require.Contains(t, string(contents), "hello repo")
}

func TestRepoPlugin_EvaluateDetectsExistingClone(t *testing.T) {
	source := initGitRepo(t)
	dest := filepath.Join(t.TempDir(), "existing")

	p := New()

	step := &config.Step{
		ID:   "clone_repo",
		Type: "repo",
		Repo: &config.RepoStep{
			URL:         source,
			Destination: dest,
		},
	}

	// Seed the destination using the plugin to ensure a valid clone for the second evaluation.
	firstEval, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, firstEval.RequiresAction)
	_, err = p.Apply(context.Background(), firstEval, step)
	require.NoError(t, err)

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.False(t, evalResult.RequiresAction)
	require.Equal(t, model.StatusSatisfied, evalResult.CurrentState)
}

func TestRepoPlugin_EvaluateDetectsCorruptedRepo(t *testing.T) {
	source := initGitRepo(t)
	dest := filepath.Join(t.TempDir(), "corrupted")
	_, err := git.PlainClone(dest, false, &git.CloneOptions{URL: source})
	require.NoError(t, err)

	// Corrupt the repository so PlainOpen fails.
	require.NoError(t, os.Remove(filepath.Join(dest, ".git", "HEAD")))

	p := New()
	step := &config.Step{
		ID:   "clone_repo",
		Type: "repo",
		Repo: &config.RepoStep{
			URL:         source,
			Destination: dest,
		},
	}

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, evalResult.RequiresAction)
	require.Equal(t, model.StatusDrifted, evalResult.CurrentState)
	require.Contains(t, evalResult.Message, "is not a git repository")
}

func TestRepoPlugin_EvaluateSkipsClone(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "clone")

	p := New()

	step := &config.Step{
		ID:   "clone_repo",
		Type: "repo",
		Repo: &config.RepoStep{
			URL:         "/tmp/example.git",
			Destination: dest,
		},
	}

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, step.ID, evalResult.StepID)
	require.Equal(t, model.StatusMissing, evalResult.CurrentState)
	require.True(t, evalResult.RequiresAction)
	_, err = os.Stat(dest)
	require.Error(t, err, "expected destination to remain untouched during evaluation")
}

func TestRepoPlugin_EvaluateReturnsMissingWhenRepoNotCloned(t *testing.T) {
	t.Parallel()

	dest := filepath.Join(t.TempDir(), "nonexistent")

	p := New()

	step := &config.Step{
		ID:   "clone_repo",
		Type: "repo",
		Repo: &config.RepoStep{
			URL:         "/tmp/example.git",
			Destination: dest,
		},
	}

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusMissing, evalResult.CurrentState)
	require.True(t, evalResult.RequiresAction, "expected Evaluate to return missing when repo not cloned")
}

func TestRepoPlugin_EvaluateReturnsDriftedWhenGitDirMissing(t *testing.T) {
	t.Parallel()

	dest := t.TempDir()
	// Create destination but without .git directory
	require.NoError(t, os.WriteFile(filepath.Join(dest, "file.txt"), []byte("test"), 0o644))

	p := New()

	step := &config.Step{
		ID:   "clone_repo",
		Type: "repo",
		Repo: &config.RepoStep{
			URL:         "/tmp/example.git",
			Destination: dest,
		},
	}

	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, model.StatusDrifted, evalResult.CurrentState)
	require.True(t, evalResult.RequiresAction, "expected Evaluate to return drifted when .git directory missing")
}

func TestRepoPlugin_ApplyWithBranchAndDepth(t *testing.T) {
	source := initGitRepo(t)
	dest := filepath.Join(t.TempDir(), "clone")

	p := New()

	step := &config.Step{
		ID:   "clone_repo",
		Type: "repo",
		Repo: &config.RepoStep{
			URL:         source,
			Destination: dest,
			Branch:      "master",
			Depth:       1,
		},
	}

	// First evaluate
	evalResult, err := p.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, evalResult.RequiresAction)

	// Then apply
	result, err := p.Apply(context.Background(), evalResult, step)
	require.NoError(t, err)
	require.Equal(t, step.ID, result.StepID)
	require.Equal(t, model.StatusSuccess, result.Status)

	contents, err := os.ReadFile(filepath.Join(dest, "README.md"))
	require.NoError(t, err)
	require.Contains(t, string(contents), "hello repo")
}

func initGitRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	require.NoError(t, err)

	wt, err := repo.Worktree()
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello repo"), 0o644))
	_, err = wt.Add("README.md")
	require.NoError(t, err)

	_, err = wt.Commit("initial", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Streamy",
			Email: "streamy@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	return dir
}

func TestRepoPlugin_Evaluate(t *testing.T) {
	t.Run("returns satisfied when repo exists and matches", func(t *testing.T) {
		source := initGitRepo(t)
		dest := filepath.Join(t.TempDir(), "clone")

		p := New()

		step := &config.Step{
			ID:   "clone_repo",
			Type: "repo",
			Repo: &config.RepoStep{
				URL:         source,
				Destination: dest,
			},
		}

		// First clone the repo
		evalResult, err := p.Evaluate(context.Background(), step)
		require.NoError(t, err)
		require.True(t, evalResult.RequiresAction)

		_, err = p.Apply(context.Background(), evalResult, step)
		require.NoError(t, err)

		// Now evaluate again
		evalResult, err = p.Evaluate(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, evalResult.StepID)
		require.Equal(t, model.StatusSatisfied, evalResult.CurrentState)
		require.Contains(t, evalResult.Message, "git repository exists")
	})

	t.Run("returns missing when repository directory does not exist", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "nonexistent")

		p := New()

		step := &config.Step{
			ID:   "clone_repo",
			Type: "repo",
			Repo: &config.RepoStep{
				URL:         "/tmp/example.git",
				Destination: dest,
			},
		}

		evalResult, err := p.Evaluate(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, evalResult.StepID)
		require.Equal(t, model.StatusMissing, evalResult.CurrentState)
		require.Contains(t, evalResult.Message, "does not exist")
	})

	t.Run("returns drifted when directory exists but is not a git repo", func(t *testing.T) {
		dest := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dest, "file.txt"), []byte("test"), 0o644))

		p := New()

		step := &config.Step{
			ID:   "clone_repo",
			Type: "repo",
			Repo: &config.RepoStep{
				URL:         "/tmp/example.git",
				Destination: dest,
			},
		}

		evalResult, err := p.Evaluate(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, evalResult.StepID)
		require.Equal(t, model.StatusDrifted, evalResult.CurrentState)
		require.Contains(t, evalResult.Message, "is not a git repository")
	})
}

func TestRepoPlugin_Evaluate_Errors(t *testing.T) {
	t.Run("returns error when repo config is nil", func(t *testing.T) {
		p := New()

		step := &config.Step{
			ID:   "check_repo",
			Type: "repo",
			Repo: nil,
		}

		_, err := p.Evaluate(context.Background(), step)
		require.Error(t, err)
		require.Contains(t, err.Error(), "repo configuration missing")
	})
}

func TestRepoPlugin_Apply_Errors(t *testing.T) {
	t.Run("returns error when repo config is nil", func(t *testing.T) {
		p := New()

		step := &config.Step{
			ID:   "clone_repo",
			Type: "repo",
			Repo: nil,
		}

		// Need to provide evalResult for Apply
		evalResult := &model.EvaluationResult{
			StepID:         step.ID,
			CurrentState:   model.StatusMissing,
			RequiresAction: true,
			Message:        "Test",
		}

		_, err := p.Apply(context.Background(), evalResult, step)
		require.Error(t, err)
		require.Contains(t, err.Error(), "repo configuration missing")
	})
}
