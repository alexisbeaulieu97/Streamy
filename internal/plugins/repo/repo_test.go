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
