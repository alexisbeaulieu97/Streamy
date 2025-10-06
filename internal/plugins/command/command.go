package commandplugin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

type commandPlugin struct{}

// New creates a new command plugin instance.
func New() plugin.Plugin {
	return &commandPlugin{}
}

func init() {
	if err := plugin.RegisterPlugin("command", New()); err != nil {
		panic(err)
	}
}

func (p *commandPlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		Name:    "shell-command",
		Version: "1.0.0",
		Type:    "command",
	}
}

func (p *commandPlugin) Schema() interface{} {
	return config.CommandStep{}
}

func (p *commandPlugin) Check(ctx context.Context, step *config.Step) (bool, error) {
	cfg := step.Command
	if cfg == nil {
		return false, streamyerrors.NewValidationError(step.ID, "command configuration missing", nil)
	}

	if strings.TrimSpace(cfg.Check) == "" {
		return false, nil
	}

	shell, shellArgs, err := determineShell(cfg.Shell)
	if err != nil {
		return false, streamyerrors.NewExecutionError(step.ID, err)
	}

	args := append(shellArgs, cfg.Check)
	cmd := exec.CommandContext(ctx, shell, args...)
	cmd.Env = buildEnv(cfg.Env)
	if cfg.WorkDir != "" {
		cmd.Dir = cfg.WorkDir
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return false, nil
		}
		if len(output) > 0 {
			return false, streamyerrors.NewExecutionError(step.ID, fmt.Errorf("%w: %s", err, string(output)))
		}
		return false, streamyerrors.NewExecutionError(step.ID, err)
	}

	return true, nil
}

func (p *commandPlugin) Apply(ctx context.Context, step *config.Step) (*model.StepResult, error) {
	cfg := step.Command
	if cfg == nil {
		return nil, streamyerrors.NewValidationError(step.ID, "command configuration missing", nil)
	}

	shell, shellArgs, err := determineShell(cfg.Shell)
	if err != nil {
		return nil, streamyerrors.NewExecutionError(step.ID, err)
	}

	args := append(shellArgs, cfg.Command)
	cmd := exec.CommandContext(ctx, shell, args...)
	cmd.Env = buildEnv(cfg.Env)
	if cfg.WorkDir != "" {
		cmd.Dir = cfg.WorkDir
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := err.Error()
		if len(output) > 0 {
			errMsg = fmt.Sprintf("%s: %s", err.Error(), string(output))
		}
		result := &model.StepResult{StepID: step.ID, Status: model.StatusFailed, Message: errMsg, Error: err}
		return result, streamyerrors.NewExecutionError(step.ID, err)
	}

	return &model.StepResult{
		StepID:  step.ID,
		Status:  model.StatusSuccess,
		Message: "command executed",
	}, nil
}

func (p *commandPlugin) DryRun(ctx context.Context, step *config.Step) (*model.StepResult, error) {
	return &model.StepResult{
		StepID:  step.ID,
		Status:  model.StatusSkipped,
		Message: "dry-run: command not executed",
	}, nil
}

func determineShell(explicit string) (string, []string, error) {
	if explicit != "" {
		return explicit, []string{"-c"}, nil
	}

	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C"}, nil
	}

	if path, err := exec.LookPath("bash"); err == nil {
		return path, []string{"-c"}, nil
	}

	if path, err := exec.LookPath("sh"); err == nil {
		return path, []string{"-c"}, nil
	}

	return "", nil, fmt.Errorf("no suitable shell found")
}

func buildEnv(custom map[string]string) []string {
	env := os.Environ()
	for k, v := range custom {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	return env
}
