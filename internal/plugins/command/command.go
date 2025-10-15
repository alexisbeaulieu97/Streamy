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
	"github.com/alexisbeaulieu97/streamy/internal/plugins/internalexec"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

type commandPlugin struct{}

// New creates a new command plugin instance.
func New() plugin.Plugin {
	return &commandPlugin{}
}

var _ plugin.Plugin = (*commandPlugin)(nil)

// PluginMetadata describes the plugin for the dependency registry.
//
// The empty Dependencies slice documents that command does not require other plugins.
// APIVersion pins compatibility with other plugins using the registry-provided interface.
func (p *commandPlugin) PluginMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name:         "command",
		Type:         "command",
		Version:      "1.0.0",
		APIVersion:   "1.x",
		Dependencies: []plugin.Dependency{},
		Stateful:     false,
		Description:  "Executes shell commands with environment and working directory control.",
	}
}

func (p *commandPlugin) Schema() any {
	return config.CommandStep{}
}

// Evaluation data for command operations
type commandEvaluationData struct {
	Shell           string
	ShellArgs       []string
	CheckCommand    string
	CheckEnv        []string
	CheckWorkDir    string
	Command         string
	CommandEnv      []string
	CommandWorkDir  string
	CheckExitCode   int
	CheckOutput     string
	CheckError      error
	ShellDetermined bool
}

func (p *commandPlugin) Evaluate(ctx context.Context, step *config.Step) (*model.EvaluationResult, error) {
	cfg, err := loadCommandConfig(step)
	if err != nil {
		stepID := ""
		if step != nil {
			stepID = step.ID
		}
		return nil, plugin.NewValidationError(stepID, fmt.Errorf("command configuration decode failed: %w", err))
	}

	if err := ctx.Err(); err != nil {
		return nil, plugin.NewStateError(step.ID, fmt.Errorf("context cancelled: %w", err))
	}

	// Determine shell (read-only operation)
	shell, shellArgs, err := determineShell(cfg.Shell)
	if err != nil {
		return nil, plugin.NewExecutionError(step.ID, fmt.Errorf("cannot determine shell: %w", err))
	}

	// Store evaluation data to avoid recomputation
	internalData := &commandEvaluationData{
		Shell:           shell,
		ShellArgs:       shellArgs,
		CheckCommand:    cfg.Check,
		CheckEnv:        buildEnv(cfg.Env),
		CheckWorkDir:    cfg.WorkDir,
		Command:         cfg.Command,
		CommandEnv:      buildEnv(cfg.Env),
		CommandWorkDir:  cfg.WorkDir,
		ShellDetermined: true,
	}

	// If no Check command is specified, we cannot evaluate the state
	if strings.TrimSpace(cfg.Check) == "" {
		return &model.EvaluationResult{
			StepID:         step.ID,
			CurrentState:   model.StatusUnknown,
			RequiresAction: true, // Assume we need to run the command
			Message:        "no verification command specified - will execute command",
			Diff:           fmt.Sprintf("Would execute: %s", cfg.Command),
			InternalData:   internalData,
		}, nil
	}

	// Execute check command (read-only operation)
	args := append(shellArgs, cfg.Check)
	cmd := exec.CommandContext(ctx, shell, args...)
	cmd.Env = internalData.CheckEnv
	if cfg.WorkDir != "" {
		cmd.Dir = cfg.WorkDir
	}

	output, err := cmd.CombinedOutput()
	internalData.CheckOutput = string(output)
	internalData.CheckError = err

	var currentState model.VerificationStatus
	var requiresAction bool
	var message string

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// Non-zero exit code = missing/drifted
			currentState = model.StatusMissing
			requiresAction = true
			message = fmt.Sprintf("check command failed (exit code %d): %s", exitErr.ExitCode(), string(output))
			internalData.CheckExitCode = exitErr.ExitCode()
		} else {
			// Other error (command not found, timeout, etc.) = blocked
			currentState = model.StatusBlocked
			requiresAction = false
			message = fmt.Sprintf("check command error: %v", err)
		}
	} else {
		// Exit code 0 = satisfied
		currentState = model.StatusSatisfied
		requiresAction = false
		message = "check command succeeded (exit code 0)"
		internalData.CheckExitCode = 0
	}

	// Generate diff for commands that need action
	var diff string
	if requiresAction && cfg.Command != "" {
		diff = fmt.Sprintf("Would execute: %s", cfg.Command)
	}

	return &model.EvaluationResult{
		StepID:         step.ID,
		CurrentState:   currentState,
		RequiresAction: requiresAction,
		Message:        message,
		Diff:           diff,
		InternalData:   internalData,
	}, nil
}

func (p *commandPlugin) Apply(ctx context.Context, evalResult *model.EvaluationResult, step *config.Step) (*model.StepResult, error) {
	cfg, err := loadCommandConfig(step)
	if err != nil {
		stepID := ""
		if step != nil {
			stepID = step.ID
		}
		return nil, plugin.NewValidationError(stepID, fmt.Errorf("command configuration decode failed: %w", err))
	}

	// Use evaluation data to avoid recomputation
	var data *commandEvaluationData
	if evalResult != nil {
		if typed, ok := evalResult.InternalData.(*commandEvaluationData); ok {
			data = typed
		}
	}
	if data == nil {
		// Fallback to re-evaluating
		var evalErr error
		evalResult, evalErr = p.Evaluate(ctx, step)
		if evalErr != nil {
			return nil, convertError(step.ID, evalErr)
		}
		typed, ok := evalResult.InternalData.(*commandEvaluationData)
		if !ok || typed == nil {
			return &model.StepResult{
				StepID:  step.ID,
				Status:  model.StatusFailed,
				Message: "evaluation failed during apply",
				Error:   fmt.Errorf("evaluation result missing command evaluation data"),
			}, plugin.NewExecutionError(step.ID, fmt.Errorf("evaluation failed during apply"))
		}
		data = typed
	}

	// Only apply if changes are needed (or if no check command exists)
	if !evalResult.RequiresAction {
		return &model.StepResult{
			StepID:  step.ID,
			Status:  model.StatusSkipped,
			Message: "no changes needed",
		}, nil
	}

	// Execute the command
	args := append(data.ShellArgs, cfg.Command)
	cmd := exec.CommandContext(ctx, data.Shell, args...)
	cmd.Env = data.CommandEnv
	if cfg.WorkDir != "" {
		cmd.Dir = cfg.WorkDir
	}

	streamResult, err := internalexec.RunStreaming(cmd)
	if err != nil {
		combinedOutput := internalexec.PrimaryOutput(streamResult)
		if combinedOutput != "" {
			err = fmt.Errorf("%w: %s", err, combinedOutput)
		}

		return &model.StepResult{
			StepID:  step.ID,
			Status:  model.StatusFailed,
			Message: fmt.Sprintf("command failed: %v", err),
			Error:   err,
		}, plugin.NewExecutionError(step.ID, fmt.Errorf("command failed: %w", err))
	}

	return &model.StepResult{
		StepID:  step.ID,
		Status:  model.StatusSuccess,
		Message: fmt.Sprintf("executed: %s", cfg.Command),
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

func loadCommandConfig(step *config.Step) (*config.CommandStep, error) {
	if step == nil {
		return nil, fmt.Errorf("step is nil")
	}

	raw := step.RawConfig()
	if len(raw) == 0 {
		return nil, fmt.Errorf("command configuration missing")
	}

	cfg := &config.CommandStep{}
	if err := step.DecodeConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func buildEnv(custom map[string]string) []string {
	env := os.Environ()
	for k, v := range custom {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	return env
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
