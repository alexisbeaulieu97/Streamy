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
	pluginpkg "github.com/alexisbeaulieu97/streamy/internal/plugin"
)

func TestRepoPlugin_Metadata(t *testing.T) {
	t.Parallel()

	p := New()
	meta := p.Metadata()

	require.NotEmpty(t, meta.Name)
	require.NotEmpty(t, meta.Version)
	require.Equal(t, "repo", meta.Type)
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
	require.Implements(t, (*pluginpkg.Plugin)(nil), p)

	step := &config.Step{
		ID:   "clone_repo",
		Type: "repo",
		Repo: &config.RepoStep{
			URL:         source,
			Destination: dest,
		},
	}

	result, err := p.Apply(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, step.ID, result.StepID)
	require.Equal(t, "success", result.Status)

	contents, err := os.ReadFile(filepath.Join(dest, "README.md"))
	require.NoError(t, err)
	require.Contains(t, string(contents), "hello repo")
}

func TestRepoPlugin_CheckDetectsExistingClone(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "existing")
	require.NoError(t, os.MkdirAll(filepath.Join(dest, ".git"), 0o755))

	p := New()

	step := &config.Step{
		ID:   "clone_repo",
		Type: "repo",
		Repo: &config.RepoStep{
			URL:         "/tmp/example.git",
			Destination: dest,
		},
	}

	ok, err := p.Check(context.Background(), step)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestRepoPlugin_DryRunSkipsClone(t *testing.T) {
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

	res, err := p.DryRun(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, step.ID, res.StepID)
	require.Equal(t, "skipped", res.Status)
	_, err = os.Stat(dest)
	require.Error(t, err, "expected destination to remain untouched during dry-run")
}

func TestRepoPlugin_CheckReturnsFalseWhenRepoNotCloned(t *testing.T) {
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

	ok, err := p.Check(context.Background(), step)
	require.NoError(t, err)
	require.False(t, ok, "expected Check to return false when repo not cloned")
}

func TestRepoPlugin_CheckReturnsFalseWhenGitDirMissing(t *testing.T) {
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

	ok, err := p.Check(context.Background(), step)
	require.NoError(t, err)
	require.False(t, ok, "expected Check to return false when .git directory missing")
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

	result, err := p.Apply(context.Background(), step)
	require.NoError(t, err)
	require.Equal(t, step.ID, result.StepID)
	require.Equal(t, "success", result.Status)

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

func TestRepoPlugin_Verify(t *testing.T) {
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
		_, err := p.Apply(context.Background(), step)
		require.NoError(t, err)

		// Now verify
		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "satisfied", string(result.Status))
		require.Contains(t, result.Message, "git repository exists")
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

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "missing", string(result.Status))
		require.Contains(t, result.Message, "does not exist")
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

		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "drifted", string(result.Status))
		require.Contains(t, result.Message, "is not a git repository")
	})

	t.Run("returns drifted when remote URL does not match", func(t *testing.T) {
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
		_, err := p.Apply(context.Background(), step)
		require.NoError(t, err)

		// Now verify with a different URL
		step.Repo.URL = "/tmp/different.git"
		result, err := p.Verify(context.Background(), step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "drifted", string(result.Status))
		require.Contains(t, result.Message, "remote URL is")
	})

	t.Run("returns blocked when context is cancelled", func(t *testing.T) {
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

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		result, err := p.Verify(ctx, step)
		require.NoError(t, err)
		require.Equal(t, step.ID, result.StepID)
		require.Equal(t, "blocked", string(result.Status))
		require.Contains(t, result.Message, "cancelled")
		require.NotNil(t, result.Error)
	})

	t.Run("returns error when repo config is nil", func(t *testing.T) {
		p := New()

		step := &config.Step{
			ID:   "clone_repo",
			Type: "repo",
			Repo: nil,
		}

		_, err := p.Verify(context.Background(), step)
		require.Error(t, err)
		require.Contains(t, err.Error(), "repo configuration missing")
	})
}

func TestRepoPlugin_Check_Errors(t *testing.T) {
	t.Run("returns error when repo config is nil", func(t *testing.T) {
		p := New()

		step := &config.Step{
			ID:   "check_repo",
			Type: "repo",
			Repo: nil,
		}

		_, err := p.Check(context.Background(), step)
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

		_, err := p.Apply(context.Background(), step)
		require.Error(t, err)
		require.Contains(t, err.Error(), "repo configuration missing")
	})

	t.Run("returns error when clone fails", func(t *testing.T) {
		p := New()

		step := &config.Step{
			ID:   "clone_repo",
			Type: "repo",
			Repo: &config.RepoStep{
				URL:         "https://github.com/nonexistent/nonexistent.git",
				Destination: filepath.Join(t.TempDir(), "clone"),
			},
		}

		result, err := p.Apply(context.Background(), step)
		require.Error(t, err)
		require.NotNil(t, result)
		require.Equal(t, "failed", result.Status)
	})
}
