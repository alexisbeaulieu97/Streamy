package packageplugin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/plugins/internalexec"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

// errPackageMissing is returned when the system reports a package is not installed.
var errPackageMissing = errors.New("package not installed")

type commandRunner func(ctx context.Context, name string, args ...string) (internalexec.Result, error)

var runCommand commandRunner = func(ctx context.Context, name string, args ...string) (internalexec.Result, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = os.Environ()

	res, err := internalexec.RunStreaming(cmd)
	if err == nil {
		return res, nil
	}

	if errors.Is(err, context.Canceled) {
		return res, err
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return res, err
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && name == "dpkg-query" {
		// Non-zero exit from dpkg-query means package is missing.
		return res, fmt.Errorf("%w: %w", errPackageMissing, err)
	}

	output := internalexec.PrimaryOutput(res)
	if output != "" {
		return res, fmt.Errorf("%w: %s", err, output)
	}

	return res, err
}

// Plugin implements package management using the ports.Plugin interface.
type Plugin struct{}

// Ensure Plugin satisfies ports.Plugin.
var _ ports.Plugin = (*Plugin)(nil)

// New constructs a ports-native package plugin.
func New() ports.Plugin {
	return &Plugin{}
}

// Metadata describes the package plugin for registry wiring.
func (Plugin) Metadata() domainplugin.Metadata {
	return domainplugin.Metadata{
		ID:          "package",
		Name:        "package",
		Version:     "1.0.0",
		Type:        domainplugin.TypePackage,
		Description: "Manages system packages using apt package manager.",
	}
}

type evaluationData struct {
	Packages          []string
	InstalledPackages []string
	MissingPackages   []string
	Update            bool
}

// Evaluate inspects package state using dpkg-query.
func (Plugin) Evaluate(ctx context.Context, step domainpipeline.Step) (*domainpipeline.EvaluationResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, domainpipeline.NewCancelledError("package evaluation cancelled", map[string]interface{}{"step_id": step.ID})
	}

	cfg, err := decodeConfig(step)
	if err != nil {
		return nil, err
	}

	data := &evaluationData{
		Packages: cfg.Packages,
		Update:   cfg.Update,
	}

	// Empty package list means nothing to do; legacy behaviour treated this as satisfied.
	if len(cfg.Packages) == 0 {
		return &domainpipeline.EvaluationResult{
			RequiresAction: false,
			CurrentState:   string(domainpipeline.VerificationSatisfied),
			DesiredState:   "no packages specified",
			InternalData:   data,
		}, nil
	}

	for _, name := range cfg.Packages {
		_, cmdErr := runCommand(ctx, "dpkg-query", "-W", name)
		if cmdErr != nil {
			switch {
			case errors.Is(cmdErr, context.Canceled):
				return nil, domainpipeline.NewCancelledError("dpkg-query cancelled", map[string]interface{}{"step_id": step.ID, "package": name})
			case errors.Is(cmdErr, context.DeadlineExceeded):
				return nil, domainpipeline.NewTimeoutError("dpkg-query timed out", cmdErr, map[string]interface{}{"step_id": step.ID, "package": name})
			case errors.Is(cmdErr, errPackageMissing):
				data.MissingPackages = append(data.MissingPackages, name)
			default:
				return nil, domainpipeline.NewExecutionError("query package state", cmdErr, map[string]interface{}{"step_id": step.ID, "package": name})
			}
			continue
		}
		data.InstalledPackages = append(data.InstalledPackages, name)
	}

	if len(data.MissingPackages) == 0 {
		return &domainpipeline.EvaluationResult{
			RequiresAction: false,
			CurrentState:   string(domainpipeline.VerificationSatisfied),
			DesiredState:   fmt.Sprintf("all packages installed: %s", strings.Join(cfg.Packages, ", ")),
			InternalData:   data,
		}, nil
	}

	diff := fmt.Sprintf("would install: %s", strings.Join(data.MissingPackages, ", "))
	return &domainpipeline.EvaluationResult{
		RequiresAction: true,
		CurrentState:   string(domainpipeline.VerificationFailed),
		DesiredState:   fmt.Sprintf("packages not installed: %s", strings.Join(data.MissingPackages, ", ")),
		Diff:           diff,
		InternalData:   data,
	}, nil
}

// Apply installs missing packages using apt-get.
func (Plugin) Apply(ctx context.Context, evaluation *domainpipeline.EvaluationResult, step domainpipeline.Step) (*domainpipeline.StepResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, domainpipeline.NewCancelledError("package apply cancelled", map[string]interface{}{"step_id": step.ID})
	}

	cfg, err := decodeConfig(step)
	if err != nil {
		return nil, err
	}

	data, err := ensureEvaluationData(ctx, evaluation, step)
	if err != nil {
		return nil, err
	}

	if evaluation != nil && !evaluation.RequiresAction {
		return &domainpipeline.StepResult{
			StepID:  step.ID,
			Status:  domainpipeline.StatusAlreadySatisfied,
			Message: "no changes required",
		}, nil
	}

	if len(data.MissingPackages) == 0 {
		return &domainpipeline.StepResult{
			StepID:  step.ID,
			Status:  domainpipeline.StatusAlreadySatisfied,
			Message: "all packages already installed",
		}, nil
	}

	args := append([]string{"install", "-y"}, data.MissingPackages...)
	if cfg.Update {
		updateRes, updateErr := runCommand(ctx, "apt-get", "update")
		if updateErr != nil {
			switch {
			case errors.Is(updateErr, context.Canceled):
				return nil, domainpipeline.NewCancelledError("apt-get update cancelled", map[string]interface{}{"step_id": step.ID})
			case errors.Is(updateErr, context.DeadlineExceeded):
				return nil, domainpipeline.NewTimeoutError("apt-get update timed out", updateErr, map[string]interface{}{"step_id": step.ID})
			default:
				return nil, domainpipeline.NewExecutionError("update package index", updateErr, map[string]interface{}{
					"step_id": step.ID,
					"output":  internalexec.PrimaryOutput(updateRes),
				})
			}
		}
	}

	res, cmdErr := runCommand(ctx, "apt-get", args...)
	if cmdErr != nil {
		switch {
		case errors.Is(cmdErr, context.Canceled):
			return nil, domainpipeline.NewCancelledError("apt-get cancelled", map[string]interface{}{"step_id": step.ID})
		case errors.Is(cmdErr, context.DeadlineExceeded):
			return nil, domainpipeline.NewTimeoutError("apt-get timed out", cmdErr, map[string]interface{}{"step_id": step.ID})
		default:
			return nil, domainpipeline.NewExecutionError("install packages", cmdErr, map[string]interface{}{
				"step_id":  step.ID,
				"packages": data.MissingPackages,
				"output":   internalexec.PrimaryOutput(res),
			})
		}
	}

	return &domainpipeline.StepResult{
		StepID:  step.ID,
		Status:  domainpipeline.StatusSuccess,
		Message: fmt.Sprintf("installed packages: %s", strings.Join(data.MissingPackages, ", ")),
		Changed: true,
		Diff:    fmt.Sprintf("installed: %s", strings.Join(data.MissingPackages, ", ")),
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
		return nil, domainpipeline.NewInternalError("package evaluation missing internal data", nil, map[string]interface{}{"step_id": step.ID})
	}
	return data, nil
}

type stepConfig struct {
	Packages []string
	Update   bool
}

func decodeConfig(step domainpipeline.Step) (stepConfig, error) {
	if strings.TrimSpace(step.ID) == "" {
		return stepConfig{}, domainpipeline.NewValidationError("package step missing id", map[string]interface{}{"step": step})
	}

	if step.Config == nil {
		return stepConfig{}, domainpipeline.NewValidationError("package configuration missing", map[string]interface{}{"step_id": step.ID})
	}

	rawPackages, ok := step.Config["packages"]
	if !ok {
		return stepConfig{}, domainpipeline.NewValidationError("packages list required", map[string]interface{}{"step_id": step.ID})
	}

	packages, err := toStringSlice(rawPackages)
	if err != nil {
		return stepConfig{}, domainpipeline.NewValidationError("packages must be an array of strings", map[string]interface{}{"step_id": step.ID})
	}

	update := getBool(step.Config, "update")

	return stepConfig{
		Packages: packages,
		Update:   update,
	}, nil
}

func toStringSlice(value interface{}) ([]string, error) {
	switch v := value.(type) {
	case []string:
		return append([]string(nil), v...), nil
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			switch typed := item.(type) {
			case string:
				out = append(out, typed)
			case fmt.Stringer:
				out = append(out, typed.String())
			default:
				return nil, fmt.Errorf("expected string, got %T", item)
			}
		}
		return out, nil
	default:
		return nil, fmt.Errorf("invalid packages type %T", value)
	}
}

func getBool(values map[string]interface{}, key string) bool {
	raw, ok := values[key]
	if !ok {
		return false
	}
	switch v := raw.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(v, "true") || strings.EqualFold(v, "yes") || v == "1"
	default:
		return false
	}
}
