package packageplugin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

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

func (p *packagePlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		Name:    "apt-packages",
		Version: "1.0.0",
		Type:    "package",
	}
}

func (p *packagePlugin) Schema() interface{} {
	return config.PackageStep{}
}

func (p *packagePlugin) Check(ctx context.Context, step *config.Step) (bool, error) {
	pkgCfg := step.Package
	if pkgCfg == nil {
		return false, streamyerrors.NewValidationError(step.ID, "package configuration missing", nil)
	}

	for _, name := range pkgCfg.Packages {
		if err := runCommand(ctx, "dpkg-query", "-W", name); err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				return false, nil
			}
			return false, streamyerrors.NewExecutionError(step.ID, err)
		}
	}

	return true, nil
}

func (p *packagePlugin) Apply(ctx context.Context, step *config.Step) (*model.StepResult, error) {
	pkgCfg := step.Package
	if pkgCfg == nil {
		return nil, streamyerrors.NewValidationError(step.ID, "package configuration missing", nil)
	}

	args := append([]string{"install", "-y"}, pkgCfg.Packages...)
	if err := runCommand(ctx, "apt-get", args...); err != nil {
		result := &model.StepResult{
			StepID:  step.ID,
			Status:  model.StatusFailed,
			Message: err.Error(),
			Error:   err,
		}
		return result, streamyerrors.NewExecutionError(step.ID, err)
	}

	return &model.StepResult{
		StepID:  step.ID,
		Status:  model.StatusSuccess,
		Message: fmt.Sprintf("installed packages: %s", strings.Join(pkgCfg.Packages, ", ")),
	}, nil
}

func (p *packagePlugin) DryRun(ctx context.Context, step *config.Step) (*model.StepResult, error) {
	return &model.StepResult{
		StepID:  step.ID,
		Status:  model.StatusSkipped,
		Message: "dry-run: packages not installed",
	}, nil
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

func (p *packagePlugin) Verify(ctx context.Context, step *config.Step) (*model.VerificationResult, error) {
	start := time.Now()
	pkgCfg := step.Package
	if pkgCfg == nil {
		return nil, streamyerrors.NewValidationError(step.ID, "package configuration missing", nil)
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return &model.VerificationResult{
			StepID:    step.ID,
			Status:    model.StatusBlocked,
			Message:   "verification cancelled",
			Error:     ctx.Err(),
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}, nil
	default:
	}

	var missingPackages []string
	for _, name := range pkgCfg.Packages {
		if err := runCommand(ctx, "dpkg-query", "-W", name); err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				missingPackages = append(missingPackages, name)
			} else {
				return &model.VerificationResult{
					StepID:    step.ID,
					Status:    model.StatusBlocked,
					Message:   fmt.Sprintf("cannot query package %s: %v", name, err),
					Error:     err,
					Duration:  time.Since(start),
					Timestamp: time.Now(),
				}, nil
			}
		}
	}

	if len(missingPackages) > 0 {
		return &model.VerificationResult{
			StepID:    step.ID,
			Status:    model.StatusMissing,
			Message:   fmt.Sprintf("packages not installed: %s", strings.Join(missingPackages, ", ")),
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}, nil
	}

	return &model.VerificationResult{
		StepID:    step.ID,
		Status:    model.StatusSatisfied,
		Message:   fmt.Sprintf("all packages installed: %s", strings.Join(pkgCfg.Packages, ", ")),
		Duration:  time.Since(start),
		Timestamp: time.Now(),
	}, nil
}
