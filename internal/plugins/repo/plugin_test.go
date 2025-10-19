package repoplugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/require"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
)

func TestMetadata(t *testing.T) {
	meta := New().Metadata()
	require.Equal(t, "repo", meta.Name)
	require.Equal(t, domainplugin.TypeRepo, meta.Type)
}

func TestEvaluateMissingRepository(t *testing.T) {
	t.Parallel()

	dest := filepath.Join(t.TempDir(), "clone")
	step := domainpipeline.Step{
		ID:   "clone_repo",
		Type: domainpipeline.StepTypeRepo,
		Config: map[string]interface{}{
			"url":         "/tmp/example.git",
			"destination": dest,
		},
	}

	result, err := New().Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, result.RequiresAction)
	require.Equal(t, string(domainpipeline.VerificationFailed), result.CurrentState)
}

func TestApplyClonesRepository(t *testing.T) {
	source := initGitRepo(t)
	dest := filepath.Join(t.TempDir(), "clone")

	step := domainpipeline.Step{
		ID:   "clone_repo",
		Type: domainpipeline.StepTypeRepo,
		Config: map[string]interface{}{
			"url":         source,
			"destination": dest,
		},
	}

	plugin := New()

	eval, err := plugin.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, eval.RequiresAction)

	res, err := plugin.Apply(context.Background(), eval, step)
	require.NoError(t, err)
	require.Equal(t, domainpipeline.StatusSuccess, res.Status)
	require.True(t, res.Changed)

	data, err := os.ReadFile(filepath.Join(dest, "README.md"))
	require.NoError(t, err)
	require.Contains(t, string(data), "hello repo")
}

func TestEvaluateSatisfiedAfterClone(t *testing.T) {
	source := initGitRepo(t)
	dest := filepath.Join(t.TempDir(), "clone")

	step := domainpipeline.Step{
		ID:   "clone_repo",
		Type: domainpipeline.StepTypeRepo,
		Config: map[string]interface{}{
			"url":         source,
			"destination": dest,
		},
	}

	plugin := New()
	eval, err := plugin.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, eval.RequiresAction)
	_, err = plugin.Apply(context.Background(), eval, step)
	require.NoError(t, err)

	result, err := plugin.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.False(t, result.RequiresAction)
	require.Equal(t, string(domainpipeline.VerificationSatisfied), result.CurrentState)
}

func TestEvaluateDetectsNonGitDirectory(t *testing.T) {
	dest := t.TempDir()
	require.NoError(t, os.MkdirAll(dest, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dest, "file.txt"), []byte("oops"), 0o644))

	step := domainpipeline.Step{
		ID:   "clone_repo",
		Type: domainpipeline.StepTypeRepo,
		Config: map[string]interface{}{
			"url":         "/tmp/example.git",
			"destination": dest,
		},
	}

	result, err := New().Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, result.RequiresAction)
	require.Equal(t, string(domainpipeline.VerificationFailed), result.CurrentState)
	require.Contains(t, result.DesiredState, "not a git repository")
}

func TestApplyRemovesMismatchedRepository(t *testing.T) {
	source := initGitRepo(t)
	dest := filepath.Join(t.TempDir(), "clone")

	// Seed destination with a different repo
	require.NoError(t, os.MkdirAll(dest, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dest, "file.txt"), []byte("not git"), 0o644))

	step := domainpipeline.Step{
		ID:   "clone_repo",
		Type: domainpipeline.StepTypeRepo,
		Config: map[string]interface{}{
			"url":         source,
			"destination": dest,
		},
	}

	plugin := New()
	eval, err := plugin.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, eval.RequiresAction)

	res, err := plugin.Apply(context.Background(), eval, step)
	require.NoError(t, err)
	require.Equal(t, domainpipeline.StatusSuccess, res.Status)
	require.True(t, res.Changed)

	_, err = os.Stat(filepath.Join(dest, "README.md"))
	require.NoError(t, err)
}

func TestDecodeConfigValidatesInputs(t *testing.T) {
	_, err := decodeConfig(domainpipeline.Step{ID: "missing"})
	require.Error(t, err)

	_, err = decodeConfig(domainpipeline.Step{
		ID:     "repo",
		Config: map[string]interface{}{},
	})
	require.Error(t, err)

	cfg, err := decodeConfig(domainpipeline.Step{
		ID: "repo",
		Config: map[string]interface{}{
			"url":         "https://example.com/repo.git",
			"destination": "/tmp/repo",
			"depth":       "5",
			"branch":      "main",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "https://example.com/repo.git", cfg.URL)
	require.Equal(t, "/tmp/repo", cfg.Destination)
	require.Equal(t, "main", cfg.Branch)
	require.Equal(t, 5, cfg.Depth)
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
