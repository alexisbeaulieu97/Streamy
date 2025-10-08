package symlinkplugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
)

type symlinkPlugin struct{}

// New creates a new symlink plugin.
func New() plugin.Plugin {
	return &symlinkPlugin{}
}

var _ plugin.Plugin = (*symlinkPlugin)(nil)
func (p *symlinkPlugin) PluginMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name:         "symlink",
		Version:      "1.0.0",
		APIVersion:   "1.x",
		Dependencies: []plugin.Dependency{},
		Stateful:     false,
		Description:  "Manages symbolic links with target validation.",
	}
}

func (p *symlinkPlugin) Schema() any {
	return config.SymlinkStep{}
}

func (p *symlinkPlugin) Evaluate(ctx context.Context, step *config.Step) (*model.EvaluationResult, error) {
	cfg := step.Symlink
	if cfg == nil {
		return nil, plugin.NewValidationError(step.ID, fmt.Errorf("symlink configuration missing"))
	}

	// Check if symlink exists and points to correct target (read-only)
	info, err := os.Lstat(cfg.Target)
	if err != nil {
		if os.IsNotExist(err) {
			return &model.EvaluationResult{
				StepID:         step.ID,
				CurrentState:   model.StatusMissing,
				RequiresAction: true,
				Message:        fmt.Sprintf("symlink %s does not exist", cfg.Target),
				Diff:           fmt.Sprintf("Would create symlink: %s -> %s", cfg.Target, cfg.Source),
			}, nil
		}
		return nil, plugin.NewStateError(step.ID, fmt.Errorf("cannot stat symlink target: %w", err))
	}

	if info.Mode()&os.ModeSymlink == 0 {
		return &model.EvaluationResult{
			StepID:         step.ID,
			CurrentState:   model.StatusDrifted,
			RequiresAction: true,
			Message:        fmt.Sprintf("target %s exists but is not a symlink", cfg.Target),
			Diff:           fmt.Sprintf("Would replace with symlink: %s -> %s", cfg.Target, cfg.Source),
		}, nil
	}

	target, err := os.Readlink(cfg.Target)
	if err != nil {
		return nil, plugin.NewStateError(step.ID, fmt.Errorf("cannot read symlink target: %w", err))
	}

	if target == cfg.Source {
		return &model.EvaluationResult{
			StepID:         step.ID,
			CurrentState:   model.StatusSatisfied,
			RequiresAction: false,
			Message:        fmt.Sprintf("symlink %s -> %s is correct", cfg.Target, cfg.Source),
		}, nil
	}

	return &model.EvaluationResult{
		StepID:         step.ID,
		CurrentState:   model.StatusDrifted,
		RequiresAction: true,
		Message:        fmt.Sprintf("symlink points to wrong target: %s -> %s (expected %s)", cfg.Target, target, cfg.Source),
		Diff:           fmt.Sprintf("Would update symlink: %s: %s -> %s", cfg.Target, target, cfg.Source),
	}, nil
}

func (p *symlinkPlugin) Apply(ctx context.Context, evalResult *model.EvaluationResult, step *config.Step) (*model.StepResult, error) {
	cfg := step.Symlink
	if cfg == nil {
		return nil, plugin.NewValidationError(step.ID, fmt.Errorf("symlink configuration missing"))
	}

	if err := os.MkdirAll(filepath.Dir(cfg.Target), 0o755); err != nil {
		return nil, plugin.NewExecutionError(step.ID, fmt.Errorf("failed to create directory: %w", err))
	}

	// Handle existing file/symlink
	if _, err := os.Lstat(cfg.Target); err == nil {
		if !cfg.Force {
			return &model.StepResult{
				StepID:  step.ID,
				Status:  model.StatusFailed,
				Message: fmt.Sprintf("target %s already exists and force is false", cfg.Target),
				Error:   fmt.Errorf("target exists"),
			}, plugin.NewExecutionError(step.ID, fmt.Errorf("target exists"))
		}
		if err := os.Remove(cfg.Target); err != nil {
			return &model.StepResult{
				StepID:  step.ID,
				Status:  model.StatusFailed,
				Message: fmt.Sprintf("failed to remove existing target %s: %v", cfg.Target, err),
				Error:   err,
			}, plugin.NewExecutionError(step.ID, fmt.Errorf("failed to remove existing target: %w", err))
		}
	}

	if err := os.Symlink(cfg.Source, cfg.Target); err != nil {
		return &model.StepResult{
			StepID:  step.ID,
			Status:  model.StatusFailed,
			Message: fmt.Sprintf("failed to create symlink %s -> %s: %v", cfg.Target, cfg.Source, err),
			Error:   err,
		}, plugin.NewExecutionError(step.ID, fmt.Errorf("failed to create symlink: %w", err))
	}

	return &model.StepResult{
		StepID:  step.ID,
		Status:  model.StatusSuccess,
		Message: fmt.Sprintf("created symlink %s -> %s", cfg.Target, cfg.Source),
	}, nil
}
