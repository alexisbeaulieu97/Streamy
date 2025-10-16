package packageplugin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/plugins/internalexec"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

type packagePlugin struct{}

// New creates a new package plugin instance.
func New() plugin.Plugin {
	return &packagePlugin{}
}

var _ plugin.Plugin = (*packagePlugin)(nil)

// PluginMetadata describes the plugin for the dependency registry.
//
// The empty Dependencies slice documents that package does not require other plugins.
// APIVersion pins compatibility with other plugins using the registry-provided interface.
func (p *packagePlugin) PluginMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name:         "package",
		Type:         "package",
		Version:      "1.0.0",
		APIVersion:   "1.x",
		Dependencies: []plugin.Dependency{},
		Stateful:     false,
		Description:  "Manages system packages using apt package manager.",
	}
}

func (p *packagePlugin) Schema() any {
	return config.PackageStep{}
}

// Evaluation data for package operations
type packageEvaluationData struct {
	InstalledPackages []string
	MissingPackages   []string
	PackageStatus     map[string]bool // true = installed, false = missing
}

func (p *packagePlugin) Evaluate(ctx context.Context, step *config.Step) (*model.EvaluationResult, error) {
	pkgCfg, err := loadPackageConfig(step)
	if err != nil {
		return nil, plugin.NewValidationError(step.ID, err)
	}

	if err := ctx.Err(); err != nil {
		return nil, plugin.NewStateError(step.ID, fmt.Errorf("context cancelled: %w", err))
	}

	// Check package status (read-only operation)
	var installedPackages []string
	var missingPackages []string
	packageStatus := make(map[string]bool)

	for _, name := range pkgCfg.Packages {
		if err := runCommand(ctx, "dpkg-query", "-W", name); err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				// Package is not installed
				missingPackages = append(missingPackages, name)
				packageStatus[name] = false
			} else {
				return nil, plugin.NewExecutionError(step.ID, fmt.Errorf("failed to query package %s: %w", name, err))
			}
		} else {
			// Package is installed
			installedPackages = append(installedPackages, name)
			packageStatus[name] = true
		}
	}

	// Store evaluation data to avoid recomputation
	internalData := &packageEvaluationData{
		InstalledPackages: installedPackages,
		MissingPackages:   missingPackages,
		PackageStatus:     packageStatus,
	}

	// Determine current state
	if len(missingPackages) == 0 {
		return &model.EvaluationResult{
			StepID:         step.ID,
			CurrentState:   model.StatusSatisfied,
			RequiresAction: false,
			Message:        fmt.Sprintf("all packages installed: %s", strings.Join(pkgCfg.Packages, ", ")),
			InternalData:   internalData,
		}, nil
	}

	// Some packages are missing
	return &model.EvaluationResult{
		StepID:         step.ID,
		CurrentState:   model.StatusMissing,
		RequiresAction: true,
		Message:        fmt.Sprintf("packages not installed: %s", strings.Join(missingPackages, ", ")),
		Diff:           fmt.Sprintf("Would install: %s", strings.Join(missingPackages, ", ")),
		InternalData:   internalData,
	}, nil
}

func (p *packagePlugin) Apply(ctx context.Context, evalResult *model.EvaluationResult, step *config.Step) (*model.StepResult, error) {
	if _, err := loadPackageConfig(step); err != nil {
		return nil, plugin.NewValidationError(step.ID, err)
	}

	// Use evaluation data to avoid recomputation
	var data *packageEvaluationData
	if evalResult != nil {
		if typed, ok := evalResult.InternalData.(*packageEvaluationData); ok {
			data = typed
		}
	}
	if data == nil {
		// Fallback to re-evaluating
		var err error
		evalResult, err = p.Evaluate(ctx, step)
		if err != nil {
			return nil, convertError(step.ID, err)
		}
		typed, ok := evalResult.InternalData.(*packageEvaluationData)
		if !ok || typed == nil {
			return &model.StepResult{
				StepID:  step.ID,
				Status:  model.StatusFailed,
				Message: "evaluation failed during apply",
				Error:   fmt.Errorf("evaluation result missing package evaluation data"),
			}, plugin.NewExecutionError(step.ID, fmt.Errorf("evaluation failed during apply"))
		}
		data = typed
	}

	// Only apply if changes are needed
	if !evalResult.RequiresAction {
		return &model.StepResult{
			StepID:  step.ID,
			Status:  model.StatusSkipped,
			Message: "no changes needed",
		}, nil
	}

	// Install missing packages
	if len(data.MissingPackages) > 0 {
		args := append([]string{"install", "-y"}, data.MissingPackages...)
		if err := runCommand(ctx, "apt-get", args...); err != nil {
			return &model.StepResult{
				StepID:  step.ID,
				Status:  model.StatusFailed,
				Message: fmt.Sprintf("failed to install packages: %v", err),
				Error:   err,
			}, plugin.NewExecutionError(step.ID, fmt.Errorf("failed to install packages: %w", err))
		}
	}

	return &model.StepResult{
		StepID:  step.ID,
		Status:  model.StatusSuccess,
		Message: fmt.Sprintf("installed packages: %s", strings.Join(data.MissingPackages, ", ")),
	}, nil
}

// Helper functions

func convertError(stepID string, err error) error {
	// Convert legacy errors to new plugin errors
	var valErr *streamyerrors.ValidationError
	if errors.As(err, &valErr) {
		return plugin.NewValidationError(stepID, valErr.Err)
	}

	var execErr *streamyerrors.ExecutionError
	if errors.As(err, &execErr) {
		return plugin.NewExecutionError(stepID, execErr.Err)
	}

	// Fallback to ExecutionError for unknown error types
	return plugin.NewExecutionError(stepID, err)
}

func runCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = os.Environ()

	streamResult, err := internalexec.RunStreaming(cmd)
	if err != nil {
		combinedOutput := internalexec.PrimaryOutput(streamResult)
		if combinedOutput != "" {
			return fmt.Errorf("%w: %s", err, combinedOutput)
		}
		return err
	}

	return nil
}

func loadPackageConfig(step *config.Step) (*config.PackageStep, error) {
	if step == nil {
		return nil, fmt.Errorf("step is nil")
	}

	if len(step.RawConfig()) == 0 {
		return nil, fmt.Errorf("package configuration missing")
	}

	cfg := &config.PackageStep{}
	if err := step.DecodeConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
