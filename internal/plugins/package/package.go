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
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

type packagePlugin struct{}

// New creates a new package plugin instance.
func New() plugin.Plugin {
	return &packagePlugin{}
}

func init() {
	if err := plugin.RegisterPlugin("package", New()); err != nil {
		panic(err)
	}
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
	return cmd.Run()
}
