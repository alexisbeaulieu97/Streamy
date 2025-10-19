package symlinkplugin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

// Plugin manages symbolic links via the ports.Plugin interface.
type Plugin struct{}

var _ ports.Plugin = (*Plugin)(nil)

// New constructs a ports-native symlink plugin.
func New() ports.Plugin {
	return &Plugin{}
}

// Metadata returns registry metadata for the symlink plugin.
func (Plugin) Metadata() domainplugin.Metadata {
	return domainplugin.Metadata{
		ID:          "symlink",
		Name:        "symlink",
		Version:     "1.0.0",
		Type:        domainplugin.TypeSymlink,
		Description: "Manages symbolic links with target validation.",
	}
}

type evaluationData struct {
	Source        string
	Target        string
	Force         bool
	Exists        bool
	IsSymlink     bool
	CurrentTarget string
}

// Evaluate inspects the filesystem to determine whether a symlink requires reconciliation.
func (Plugin) Evaluate(ctx context.Context, step domainpipeline.Step) (*domainpipeline.EvaluationResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, domainpipeline.NewCancelledError("symlink evaluation cancelled", map[string]interface{}{
			"step_id":     step.ID,
			"plugin_type": string(domainplugin.TypeSymlink),
		})
	}

	cfg, err := decodeConfig(step)
	if err != nil {
		return nil, err
	}

	data := &evaluationData{
		Source: cfg.Source,
		Target: cfg.Target,
		Force:  cfg.Force,
	}

	info, statErr := os.Lstat(cfg.Target)
	switch {
	case errors.Is(statErr, os.ErrNotExist):
		data.Exists = false
		return &domainpipeline.EvaluationResult{
			RequiresAction: true,
			CurrentState:   string(domainpipeline.VerificationFailed),
			DesiredState:   "symlink does not exist",
			Diff:           fmt.Sprintf("would create symlink: %s -> %s", cfg.Target, cfg.Source),
			InternalData:   data,
		}, nil
	case statErr != nil:
		return nil, domainpipeline.NewExecutionError("stat symlink target", statErr, map[string]interface{}{
			"step_id":     step.ID,
			"plugin_type": string(domainplugin.TypeSymlink),
			"target":      cfg.Target,
		})
	default:
		data.Exists = true
	}

	if info.Mode()&os.ModeSymlink == 0 {
		data.IsSymlink = false
		message := "target exists but is not a symlink"
		return &domainpipeline.EvaluationResult{
			RequiresAction: cfg.Force,
			CurrentState:   string(domainpipeline.VerificationFailed),
			DesiredState:   message,
			Diff:           fmt.Sprintf("would replace with symlink: %s -> %s", cfg.Target, cfg.Source),
			InternalData:   data,
		}, nil
	}
	data.IsSymlink = true

	linkTarget, readErr := os.Readlink(cfg.Target)
	if readErr != nil {
		return nil, domainpipeline.NewExecutionError("read symlink target", readErr, map[string]interface{}{
			"step_id":     step.ID,
			"plugin_type": string(domainplugin.TypeSymlink),
			"target":      cfg.Target,
		})
	}
	data.CurrentTarget = linkTarget

	if linkTarget == cfg.Source {
		return &domainpipeline.EvaluationResult{
			RequiresAction: false,
			CurrentState:   string(domainpipeline.VerificationSatisfied),
			DesiredState:   "symlink exists and points to correct target",
			InternalData:   data,
		}, nil
	}

	return &domainpipeline.EvaluationResult{
		RequiresAction: true,
		CurrentState:   string(domainpipeline.VerificationFailed),
		DesiredState:   fmt.Sprintf("symlink points to %s", linkTarget),
		Diff:           fmt.Sprintf("would update symlink: %s -> %s", cfg.Target, cfg.Source),
		InternalData:   data,
	}, nil
}

// Apply reconciles the symlink on disk when evaluation indicates drift or missing state.
func (Plugin) Apply(ctx context.Context, evaluation *domainpipeline.EvaluationResult, step domainpipeline.Step) (*domainpipeline.StepResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, domainpipeline.NewCancelledError("symlink apply cancelled", map[string]interface{}{
			"step_id":     step.ID,
			"plugin_type": string(domainplugin.TypeSymlink),
		})
	}

	cfg, err := decodeConfig(step)
	if err != nil {
		return nil, err
	}

	if _, err := ensureEvaluationData(ctx, evaluation, step); err != nil {
		return nil, err
	}

	if evaluation != nil && !evaluation.RequiresAction {
		return &domainpipeline.StepResult{
			StepID:  step.ID,
			Status:  domainpipeline.StatusAlreadySatisfied,
			Message: "no changes required",
		}, nil
	}

	if err := os.MkdirAll(filepath.Dir(cfg.Target), 0o755); err != nil {
		return nil, domainpipeline.NewExecutionError("create symlink directory", err, map[string]interface{}{
			"step_id":     step.ID,
			"plugin_type": string(domainplugin.TypeSymlink),
			"directory":   filepath.Dir(cfg.Target),
		})
	}

	if _, err := os.Lstat(cfg.Target); err == nil {
		if !cfg.Force {
			return nil, domainpipeline.NewExecutionError("target exists and force disabled", fmt.Errorf("target %s already exists", cfg.Target), map[string]interface{}{
				"step_id":     step.ID,
				"plugin_type": string(domainplugin.TypeSymlink),
				"target":      cfg.Target,
			})
		}
		if removeErr := os.Remove(cfg.Target); removeErr != nil {
			return nil, domainpipeline.NewExecutionError("remove existing target", removeErr, map[string]interface{}{
				"step_id":     step.ID,
				"plugin_type": string(domainplugin.TypeSymlink),
				"target":      cfg.Target,
			})
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, domainpipeline.NewExecutionError("stat target", err, map[string]interface{}{
			"step_id":     step.ID,
			"plugin_type": string(domainplugin.TypeSymlink),
			"target":      cfg.Target,
		})
	}

	if err := os.Symlink(cfg.Source, cfg.Target); err != nil {
		return nil, domainpipeline.NewExecutionError("create symlink", err, map[string]interface{}{
			"step_id":     step.ID,
			"plugin_type": string(domainplugin.TypeSymlink),
			"source":      cfg.Source,
			"target":      cfg.Target,
		})
	}

	return &domainpipeline.StepResult{
		StepID:  step.ID,
		Status:  domainpipeline.StatusSuccess,
		Message: fmt.Sprintf("created symlink %s -> %s", cfg.Target, cfg.Source),
		Changed: true,
	}, nil
}

func ensureEvaluationData(ctx context.Context, evaluation *domainpipeline.EvaluationResult, step domainpipeline.Step) (*evaluationData, error) {
	if evaluation != nil {
		if data, ok := evaluation.InternalData.(*evaluationData); ok && data != nil {
			return data, nil
		}
	}

	eval, err := (&Plugin{}).Evaluate(ctx, step)
	if err != nil {
		return nil, err
	}
	data, ok := eval.InternalData.(*evaluationData)
	if !ok || data == nil {
		return nil, domainpipeline.NewInternalError("symlink evaluation missing internal data", nil, map[string]interface{}{
			"step_id":     step.ID,
			"plugin_type": string(domainplugin.TypeSymlink),
		})
	}
	return data, nil
}

type stepConfig struct {
	Source string
	Target string
	Force  bool
}

func decodeConfig(step domainpipeline.Step) (stepConfig, error) {
	if strings.TrimSpace(step.ID) == "" {
		return stepConfig{}, domainpipeline.NewValidationError("symlink step missing id", map[string]interface{}{
			"plugin_type": string(domainplugin.TypeSymlink),
		})
	}
	if step.Config == nil {
		return stepConfig{}, domainpipeline.NewValidationError("symlink configuration missing", map[string]interface{}{
			"step_id":     step.ID,
			"plugin_type": string(domainplugin.TypeSymlink),
		})
	}

	source, ok := getString(step.Config, "source")
	if !ok || strings.TrimSpace(source) == "" {
		return stepConfig{}, domainpipeline.NewValidationError("symlink source is required", map[string]interface{}{
			"step_id":     step.ID,
			"plugin_type": string(domainplugin.TypeSymlink),
		})
	}

	target, ok := getString(step.Config, "target")
	if !ok || strings.TrimSpace(target) == "" {
		return stepConfig{}, domainpipeline.NewValidationError("symlink target is required", map[string]interface{}{
			"step_id":     step.ID,
			"plugin_type": string(domainplugin.TypeSymlink),
		})
	}

	force := false
	if raw, exists := step.Config["force"]; exists {
		parsed, err := parseBool(raw)
		if err != nil {
			return stepConfig{}, domainpipeline.NewValidationError("symlink force must be boolean", map[string]interface{}{
				"step_id":     step.ID,
				"plugin_type": string(domainplugin.TypeSymlink),
				"value":       raw,
			})
		}
		force = parsed
	}

	return stepConfig{
		Source: strings.TrimSpace(source),
		Target: strings.TrimSpace(target),
		Force:  force,
	}, nil
}

func getString(values map[string]interface{}, key string) (string, bool) {
	raw, ok := values[key]
	if !ok || raw == nil {
		return "", false
	}
	switch v := raw.(type) {
	case string:
		return v, true
	default:
		return fmt.Sprintf("%v", v), true
	}
}

func parseBool(value interface{}) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true", "yes", "1":
			return true, nil
		case "false", "no", "0", "":
			return false, nil
		default:
			return false, fmt.Errorf("invalid boolean string %q", v)
		}
	default:
		return false, fmt.Errorf("invalid boolean type %T", value)
	}
}
