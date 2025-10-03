package repoplugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

type repoPlugin struct{}

// New creates a new repository plugin.
func New() plugin.Plugin {
	return &repoPlugin{}
}

func init() {
	if err := plugin.RegisterPlugin("repo", New()); err != nil {
		panic(err)
	}
}

func (p *repoPlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		Name:    "git-repo",
		Version: "1.0.0",
		Type:    "repo",
	}
}

func (p *repoPlugin) Schema() interface{} {
	return config.RepoStep{}
}

func (p *repoPlugin) Check(ctx context.Context, step *config.Step) (bool, error) {
	repoCfg := step.Repo
	if repoCfg == nil {
		return false, streamyerrors.NewValidationError(step.ID, "repo configuration missing", nil)
	}

	if _, err := os.Stat(repoCfg.Destination); err != nil {
		return false, nil
	}

	if _, err := os.Stat(filepath.Join(repoCfg.Destination, ".git")); err != nil {
		return false, nil
	}

	return true, nil
}

func (p *repoPlugin) Apply(ctx context.Context, step *config.Step) (*model.StepResult, error) {
	repoCfg := step.Repo
	if repoCfg == nil {
		return nil, streamyerrors.NewValidationError(step.ID, "repo configuration missing", nil)
	}

	if err := os.MkdirAll(filepath.Dir(repoCfg.Destination), 0o755); err != nil {
		return nil, streamyerrors.NewExecutionError(step.ID, err)
	}

	opts := &git.CloneOptions{
		URL: repoCfg.URL,
	}

	if repoCfg.Depth > 0 {
		opts.Depth = repoCfg.Depth
	}

	if repoCfg.Branch != "" {
		opts.ReferenceName = plumbing.NewBranchReferenceName(repoCfg.Branch)
		opts.SingleBranch = true
	}

	if _, err := git.PlainCloneContext(ctx, repoCfg.Destination, false, opts); err != nil {
		return &model.StepResult{
			StepID:  step.ID,
			Status:  model.StatusFailed,
			Message: err.Error(),
			Error:   err,
		}, streamyerrors.NewExecutionError(step.ID, err)
	}

	return &model.StepResult{
		StepID:  step.ID,
		Status:  model.StatusSuccess,
		Message: fmt.Sprintf("cloned %s", repoCfg.URL),
	}, nil
}

func (p *repoPlugin) DryRun(ctx context.Context, step *config.Step) (*model.StepResult, error) {
	return &model.StepResult{
		StepID:  step.ID,
		Status:  model.StatusSkipped,
		Message: "dry-run: repository not cloned",
	}, nil
}
