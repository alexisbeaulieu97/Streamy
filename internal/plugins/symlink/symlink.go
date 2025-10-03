package symlinkplugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

type symlinkPlugin struct{}

// New creates a new symlink plugin.
func New() plugin.Plugin {
	return &symlinkPlugin{}
}

func init() {
	if err := plugin.RegisterPlugin("symlink", New()); err != nil {
		panic(err)
	}
}

func (p *symlinkPlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		Name:    "symlink",
		Version: "1.0.0",
		Type:    "symlink",
	}
}

func (p *symlinkPlugin) Schema() interface{} {
	return config.SymlinkStep{}
}

func (p *symlinkPlugin) Check(ctx context.Context, step *config.Step) (bool, error) {
	cfg := step.Symlink
	if cfg == nil {
		return false, streamyerrors.NewValidationError(step.ID, "symlink configuration missing", nil)
	}

	info, err := os.Lstat(cfg.Target)
	if err != nil {
		return false, nil
	}

	if info.Mode()&os.ModeSymlink == 0 {
		return false, nil
	}

	target, err := os.Readlink(cfg.Target)
	if err != nil {
		return false, streamyerrors.NewExecutionError(step.ID, err)
	}

	return target == cfg.Source, nil
}

func (p *symlinkPlugin) Apply(ctx context.Context, step *config.Step) (*model.StepResult, error) {
	cfg := step.Symlink
	if cfg == nil {
		return nil, streamyerrors.NewValidationError(step.ID, "symlink configuration missing", nil)
	}

	if err := os.MkdirAll(filepath.Dir(cfg.Target), 0o755); err != nil {
		return nil, streamyerrors.NewExecutionError(step.ID, err)
	}

	if _, err := os.Lstat(cfg.Target); err == nil {
		if !cfg.Force {
			return &model.StepResult{
				StepID:  step.ID,
				Status:  model.StatusFailed,
				Message: fmt.Sprintf("target %s already exists", cfg.Target),
			}, streamyerrors.NewExecutionError(step.ID, fmt.Errorf("target exists"))
		}
		if err := os.Remove(cfg.Target); err != nil {
			return &model.StepResult{
				StepID:  step.ID,
				Status:  model.StatusFailed,
				Message: err.Error(),
				Error:   err,
			}, streamyerrors.NewExecutionError(step.ID, err)
		}
	}

	if err := os.Symlink(cfg.Source, cfg.Target); err != nil {
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
		Message: fmt.Sprintf("linked %s -> %s", cfg.Target, cfg.Source),
	}, nil
}

func (p *symlinkPlugin) DryRun(ctx context.Context, step *config.Step) (*model.StepResult, error) {
	return &model.StepResult{
		StepID:  step.ID,
		Status:  model.StatusSkipped,
		Message: "dry-run: symlink not created",
	}, nil
}
