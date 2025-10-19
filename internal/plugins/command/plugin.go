package commandplugin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/plugins/internalexec"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

// Plugin implements the command step using the ports.Plugin interface.
type Plugin struct{}

// New constructs a ports.Plugin implementation.
func New() ports.Plugin {
	return &Plugin{}
}

// Metadata returns the domain metadata for the command plugin.
func (Plugin) Metadata() domainplugin.Metadata {
	return domainplugin.Metadata{
		ID:          "command",
		Name:        "command",
		Version:     "1.0.0",
		Type:        domainplugin.TypeCommand,
		Description: "Executes shell commands with environment and working directory control.",
	}
}

type evaluationData struct {
	Shell          string
	ShellArgs      []string
	CheckCommand   string
	CheckEnv       []string
	CheckWorkDir   string
	Command        string
	CommandEnv     []string
	CommandWorkDir string
}

type stepConfig struct {
	Command string
	Check   string
	Shell   string
	WorkDir string
	Env     map[string]string
}

// Evaluate inspects the system state using an optional check command.
func (p *Plugin) Evaluate(ctx context.Context, step domainpipeline.Step) (*domainpipeline.EvaluationResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, domainpipeline.NewDomainError(
			domainpipeline.ErrCodeCancelled,
			"command evaluation cancelled",
			err,
			map[string]interface{}{"step_id": step.ID, "plugin_type": string(domainplugin.TypeCommand)},
		)
	}

	cfg, err := decodeConfig(step)
	if err != nil {
		return nil, err
	}

	shell, shellArgs, err := determineShell(cfg.Shell)
	if err != nil {
		return nil, domainpipeline.NewDomainError(
			domainpipeline.ErrCodeExecution,
			"determine shell",
			err,
			map[string]interface{}{"step_id": step.ID, "plugin_type": string(domainplugin.TypeCommand)},
		)
	}

	data := &evaluationData{
		Shell:          shell,
		ShellArgs:      shellArgs,
		CheckCommand:   cfg.Check,
		CheckEnv:       buildEnv(cfg.Env),
		CheckWorkDir:   cfg.WorkDir,
		Command:        cfg.Command,
		CommandEnv:     buildEnv(cfg.Env),
		CommandWorkDir: cfg.WorkDir,
	}

	if strings.TrimSpace(cfg.Check) == "" {
		return &domainpipeline.EvaluationResult{
			RequiresAction: true,
			CurrentState:   string(domainpipeline.VerificationUnknown),
			DesiredState:   "no check command provided",
			Diff:           fmt.Sprintf("Would execute: %s", cfg.Command),
			InternalData:   data,
		}, nil
	}

	args := append(shellArgs, cfg.Check)
	cmd := exec.CommandContext(ctx, shell, args...) //nolint:gosec // shell determined above
	cmd.Env = data.CheckEnv
	if cfg.WorkDir != "" {
		cmd.Dir = cfg.WorkDir
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return nil, domainpipeline.NewDomainError(
				domainpipeline.ErrCodeCancelled,
				"check command cancelled",
				err,
				map[string]interface{}{"step_id": step.ID, "plugin_type": string(domainplugin.TypeCommand)},
			)
		default:
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				return &domainpipeline.EvaluationResult{
					RequiresAction: true,
					CurrentState:   string(domainpipeline.VerificationFailed),
					DesiredState:   fmt.Sprintf("check command exit code %d", exitErr.ExitCode()),
					Diff:           fmt.Sprintf("Would execute: %s", cfg.Command),
					InternalData:   data,
				}, nil
			}

			return nil, domainpipeline.NewDomainError(
				domainpipeline.ErrCodeExecution,
				"check command failed",
				err,
				map[string]interface{}{"step_id": step.ID, "plugin_type": string(domainplugin.TypeCommand), "output": string(output)},
			)
		}
	}

	return &domainpipeline.EvaluationResult{
		RequiresAction: false,
		CurrentState:   string(domainpipeline.VerificationSatisfied),
		DesiredState:   "check command succeeded",
		InternalData:   data,
	}, nil
}

// Apply executes the command when evaluation determined work is required.
func (p *Plugin) Apply(ctx context.Context, evaluation *domainpipeline.EvaluationResult, step domainpipeline.Step) (*domainpipeline.StepResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, domainpipeline.NewDomainError(
			domainpipeline.ErrCodeCancelled,
			"command apply cancelled",
			err,
			map[string]interface{}{"step_id": step.ID, "plugin_type": string(domainplugin.TypeCommand)},
		)
	}

	cfg, err := decodeConfig(step)
	if err != nil {
		return nil, err
	}

	data, err := p.ensureEvaluationData(ctx, evaluation, step)
	if err != nil {
		return nil, err
	}

	if evaluation != nil && !evaluation.RequiresAction {
		return &domainpipeline.StepResult{
			StepID:  step.ID,
			Status:  domainpipeline.StatusAlreadySatisfied,
			Message: "no changes needed",
		}, nil
	}

	args := append(data.ShellArgs, cfg.Command)
	cmd := exec.CommandContext(ctx, data.Shell, args...) //nolint:gosec // shell determined earlier
	cmd.Env = data.CommandEnv
	if cfg.WorkDir != "" {
		cmd.Dir = cfg.WorkDir
	}

	streamResult, err := internalexec.RunStreaming(cmd)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, domainpipeline.NewDomainError(
				domainpipeline.ErrCodeCancelled,
				"command cancelled",
				err,
				map[string]interface{}{"step_id": step.ID, "plugin_type": string(domainplugin.TypeCommand)},
			)
		}

		message := internalexec.PrimaryOutput(streamResult)
		if message == "" {
			message = err.Error()
		}
		return nil, domainpipeline.NewDomainError(
			domainpipeline.ErrCodeExecution,
			"command failed",
			err,
			map[string]interface{}{"step_id": step.ID, "plugin_type": string(domainplugin.TypeCommand), "output": message},
		)
	}

	return &domainpipeline.StepResult{
		StepID:  step.ID,
		Status:  domainpipeline.StatusSuccess,
		Message: fmt.Sprintf("executed: %s", cfg.Command),
		Changed: true,
	}, nil
}

func (p *Plugin) ensureEvaluationData(ctx context.Context, evaluation *domainpipeline.EvaluationResult, step domainpipeline.Step) (*evaluationData, error) {
	if evaluation != nil {
		if typed, ok := evaluation.InternalData.(*evaluationData); ok && typed != nil {
			return typed, nil
		}
	}

	evalResult, err := p.Evaluate(ctx, step)
	if err != nil {
		return nil, err
	}
	typed, ok := evalResult.InternalData.(*evaluationData)
	if !ok || typed == nil {
		return nil, domainpipeline.NewDomainError(
			domainpipeline.ErrCodeExecution,
			"evaluation missing internal data",
			nil,
			map[string]interface{}{"step_id": step.ID, "plugin_type": string(domainplugin.TypeCommand)},
		)
	}
	return typed, nil
}

func decodeConfig(step domainpipeline.Step) (stepConfig, error) {
	if strings.TrimSpace(step.ID) == "" {
		return stepConfig{}, domainpipeline.NewDomainError(
			domainpipeline.ErrCodeValidation,
			"command step missing id",
			nil,
			map[string]interface{}{"step": step},
		)
	}
	if len(step.Config) == 0 {
		return stepConfig{}, domainpipeline.NewDomainError(
			domainpipeline.ErrCodeValidation,
			"command configuration missing",
			nil,
			map[string]interface{}{"step_id": step.ID},
		)
	}

	cfg := stepConfig{Env: map[string]string{}}

	if cmd, ok := getString(step.Config, "command"); ok && strings.TrimSpace(cmd) != "" {
		cfg.Command = cmd
	} else {
		return stepConfig{}, domainpipeline.NewDomainError(
			domainpipeline.ErrCodeValidation,
			"command is required",
			nil,
			map[string]interface{}{"step_id": step.ID},
		)
	}

	if check, ok := getString(step.Config, "check"); ok {
		cfg.Check = check
	}
	if shell, ok := getString(step.Config, "shell"); ok {
		cfg.Shell = shell
	}
	if workdir, ok := getString(step.Config, "workdir"); ok {
		cfg.WorkDir = workdir
	}

	if rawEnv, ok := step.Config["env"].(map[string]interface{}); ok {
		for k, v := range rawEnv {
			if v == nil {
				continue
			}
			cfg.Env[k] = fmt.Sprint(v)
		}
	} else if env, ok := step.Config["env"].(map[string]string); ok {
		for k, v := range env {
			cfg.Env[k] = v
		}
	}

	return cfg, nil
}

func getString(values map[string]interface{}, key string) (string, bool) {
	if values == nil {
		return "", false
	}
	val, ok := values[key]
	if !ok {
		return "", false
	}
	switch typed := val.(type) {
	case string:
		return typed, true
	case fmt.Stringer:
		return typed.String(), true
	default:
		return fmt.Sprintf("%v", typed), true
	}
}

func buildEnv(custom map[string]string) []string {
	env := os.Environ()
	for k, v := range custom {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	return env
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
