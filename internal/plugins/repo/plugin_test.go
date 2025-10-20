package repoplugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	assert "github.com/stretchr/testify/assert"
	require "github.com/stretchr/testify/require"

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

func TestEvaluateDetectsFileAsDestination(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "clone")
	require.NoError(t, os.WriteFile(dest, []byte("oops"), 0o644))

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
	require.Contains(t, result.DesiredState, "destination exists but is not a directory")
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

func TestEvaluateDetectsMismatchedURL(t *testing.T) {
	source := initGitRepo(t)
	dest := filepath.Join(t.TempDir(), "clone")

	// Clone the repo first
	plugin := New()
	step := domainpipeline.Step{
		ID:   "clone_repo",
		Type: domainpipeline.StepTypeRepo,
		Config: map[string]interface{}{
			"url":         source,
			"destination": dest,
		},
	}
	_, err := plugin.Apply(context.Background(), nil, step)
	require.NoError(t, err)

	// Now, evaluate with a different URL
	step.Config["url"] = "/another/repo.git"
	result, err := plugin.Evaluate(context.Background(), step)
	require.NoError(t, err)
	require.True(t, result.RequiresAction)
	require.Contains(t, result.DesiredState, "remote URL is")
}

func TestApplyWithCancelledContext(t *testing.T) {
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
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := plugin.Apply(ctx, nil, step)
	require.Error(t, err)

	var domainErr *domainpipeline.DomainError
	require.ErrorAs(t, err, &domainErr)
	require.Equal(t, domainpipeline.ErrCodeCancelled, domainErr.Code)
}

func TestDecodeConfigWithInvalidDepth(t *testing.T) {
	_, err := decodeConfig(domainpipeline.Step{
		ID: "repo",
		Config: map[string]interface{}{
			"url":         "https://example.com/repo.git",
			"destination": "/tmp/repo",
			"depth":       -1,
		},
	})
	require.Error(t, err)
}

func TestDecodeConfigWithEnvOverrides(t *testing.T) {
	t.Setenv("STREAMY_REPO_PATH", "/override/path")
	t.Setenv("STREAMY_REPO_BRANCH", "override_branch")
	t.Setenv("STREAMY_REPO_DEPTH", "42")
	t.Setenv("STREAMY_REPO_URL", "https://override.com/repo.git")

	cfg, err := decodeConfig(domainpipeline.Step{
		ID: "repo",
		Config: map[string]interface{}{
			"url":         "https://example.com/repo.git",
			"destination": "/tmp/repo",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "/override/path", cfg.Destination)
	require.Equal(t, "override_branch", cfg.Branch)
	require.Equal(t, 42, cfg.Depth)
	require.Equal(t, "https://override.com/repo.git", cfg.URL)
}

func TestParseDepth(t *testing.T) {
	// Test with different valid types
	validInputs := []interface{}{10, int32(20), int64(30), "50", 0}
	expected := []int{10, 20, 30, 50, 0}

	for i, input := range validInputs {
		val, err := parseDepth(input)
		require.NoError(t, err)
		require.Equal(t, expected[i], val)
	}

	// Test with invalid types
	invalidInputs := []interface{}{-1, -1.0, "-1", "abc", 1.5, true, nil, 40.5}
	for _, input := range invalidInputs {
		_, err := parseDepth(input)
		require.Error(t, err)
	}
}

func TestUrlsEqual(t *testing.T) {
	assert.True(t, urlsEqual("https://a.com/b.git", "https://a.com/b"))
	assert.True(t, urlsEqual("https://a.com/b", "https://a.com/b.git"))
	assert.False(t, urlsEqual("https://a.com/b.git", "https://a.com/c.git"))
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
