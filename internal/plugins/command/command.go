package commandplugin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/plugins/internalexec"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

type commandPlugin struct{}

// New creates a new command plugin instance.
func New() plugin.Plugin {
	return &commandPlugin{}
}

var _ plugin.Plugin = (*commandPlugin)(nil)
var _ plugin.MetadataProvider = (*commandPlugin)(nil)

func (p *commandPlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{
		Name:    "shell-command",
		Version: "1.0.0",
		Type:    "command",
	}
}

// PluginMetadata describes the plugin for the dependency registry.
//
// The empty Dependencies slice documents that command does not require other plugins.
// APIVersion pins compatibility with other plugins using the registry-provided interface.
func (p *commandPlugin) PluginMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name:         "command",
		Version:      "1.0.0",
		APIVersion:   "1.x",
		Dependencies: []plugin.Dependency{},
		Stateful:     false,
		Description:  "Executes shell commands with environment and working directory control.",
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

	streamResult, err := internalexec.RunStreaming(cmd)
	if err != nil {
		combinedOutput := internalexec.PrimaryOutput(streamResult)
		if combinedOutput != "" {
			err = fmt.Errorf("%w: %s", err, combinedOutput)
		}

		result := &model.StepResult{StepID: step.ID, Status: model.StatusFailed, Message: err.Error(), Error: err}
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

func (p *commandPlugin) Verify(ctx context.Context, step *config.Step) (*model.VerificationResult, error) {
	start := time.Now()
	cfg := step.Command
	if cfg == nil {
		return nil, streamyerrors.NewValidationError(step.ID, "command configuration missing", nil)
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

	// If no Check command is specified, default to checking if the command exists
	if strings.TrimSpace(cfg.Check) == "" {
		// Default verification: check if the command exists using command -v or which
		checkCmd := "command"
		if runtime.GOOS == "windows" {
			checkCmd = "where"
		}
		cfg.Check = fmt.Sprintf("%s %s", checkCmd, strings.Fields(cfg.Command)[0])
	}

	// Execute the check command
	shell, shellArgs, err := determineShell(cfg.Shell)
	if err != nil {
		return &model.VerificationResult{
			StepID:    step.ID,
			Status:    model.StatusBlocked,
			Message:   fmt.Sprintf("cannot determine shell: %v", err),
			Error:     err,
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}, nil
	}

	args := append(shellArgs, cfg.Check)
	cmd := exec.CommandContext(ctx, shell, args...)
	cmd.Env = buildEnv(cfg.Env)
	if cfg.WorkDir != "" {
		cmd.Dir = cfg.WorkDir
	}

	err = cmd.Run()
	if err == nil {
		// Exit code 0 = satisfied
		return &model.VerificationResult{
			StepID:    step.ID,
			Status:    model.StatusSatisfied,
			Message:   "check command succeeded (exit code 0)",
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}, nil
	}

	// Check for exit error vs other errors
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		// Non-zero exit code = missing
		return &model.VerificationResult{
			StepID:    step.ID,
			Status:    model.StatusMissing,
			Message:   fmt.Sprintf("check command failed (exit code %d)", exitErr.ExitCode()),
			Duration:  time.Since(start),
			Timestamp: time.Now(),
		}, nil
	}

	// Other errors (command not found, timeout, etc.) = blocked
	return &model.VerificationResult{
		StepID:    step.ID,
		Status:    model.StatusBlocked,
		Message:   fmt.Sprintf("check command error: %v", err),
		Error:     err,
		Duration:  time.Since(start),
		Timestamp: time.Now(),
	}, nil
}
