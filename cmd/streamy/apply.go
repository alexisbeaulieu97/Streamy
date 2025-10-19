package main

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/alexisbeaulieu97/streamy/internal/pipelineconv"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
	"github.com/alexisbeaulieu97/streamy/internal/tui"
)

type applyOptions struct {
	ConfigPath          string
	DryRun              bool
	Verbose             bool
	NonInteractive      bool
	ForceNonInteractive bool
	Timeout             time.Duration
}

func newApplyCmd(root *rootFlags, app *AppContext) *cobra.Command {
	opts := applyOptions{}

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply a Streamy configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.DryRun = root.dryRun
			opts.Verbose = root.verbose
			opts.NonInteractive = opts.ForceNonInteractive || !term.IsTerminal(int(os.Stdout.Fd()))
			if opts.Timeout <= 0 {
				opts.Timeout = root.timeout
			}

			if err := validateApplyOptions(opts); err != nil {
				return err
			}

			cmdCtx, logger := app.CommandContext(cmd, "command.apply")

			return runApply(cmdCtx, app, opts, logger)
		},
	}

	cmd.Flags().StringVarP(&opts.ConfigPath, "config", "c", "", "Path to configuration file")
	cmd.Flags().BoolVar(&opts.ForceNonInteractive, "non-interactive", false, "Disable the interactive TUI and print results to stdout")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 0, "Maximum duration for the apply command (defaults to root timeout)")
	cmd.MarkFlagRequired("config") //nolint:errcheck

	return cmd
}

func runApply(ctx context.Context, app *AppContext, opts applyOptions, logger ports.Logger) error {
	if ctx == nil {
		return fmt.Errorf("context is required")
	}

	var (
		execCtx context.Context
		cancel  context.CancelFunc
	)
	if opts.Timeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, opts.Timeout)
	} else {
		execCtx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	if logger != nil {
		logger.Info(execCtx, "command.apply.start", "config_path", opts.ConfigPath, "dry_run", opts.DryRun, "non_interactive", opts.NonInteractive)
		defer logger.Info(execCtx, "command.apply.finish", "config_path", opts.ConfigPath)
	}

	pipelineDomain, planDomain, err := app.PrepareUseCase.Prepare(execCtx, opts.ConfigPath)
	if err != nil {
		return err
	}

	modelState := tui.NewModel(pipelineDomain, planDomain, opts.NonInteractive)
	interactive := !opts.NonInteractive

	var (
		program     *tea.Program
		programErr  error
		programDone chan struct{}
	)

	if interactive {
		program = tea.NewProgram(modelState)
		programDone = make(chan struct{})
		go func() {
			_, programErr = program.Run()
			close(programDone)
		}()
		go func() {
			select {
			case <-execCtx.Done():
				program.Send(tea.QuitMsg{})
			case <-programDone:
			}
		}()
	}

	_, domainResults, summary, execErr := app.ApplyUseCase.Apply(execCtx, opts.ConfigPath, opts.DryRun)

	for _, res := range domainResults {
		state := pipelineconv.ToStepState(res, opts.DryRun)
		dispatchTuiMessage(interactive, program, &modelState, tui.StepCompleteMsg{StepID: res.StepID, State: state})
	}

	if summary != nil {
		for _, validation := range summary.Results {
			dispatchTuiMessage(interactive, program, &modelState, tui.ValidationMsg{
				Passed:  validation.IsSatisfied(),
				Message: validation.FormatMessage(),
			})
		}
	}

	if interactive {
		if program != nil {
			program.Send(tea.QuitMsg{})
		}
		if programDone != nil {
			select {
			case <-programDone:
			case <-time.After(250 * time.Millisecond):
			}
		}
		if programErr != nil {
			return programErr
		}
	} else {
		_, _ = fmt.Fprintln(os.Stdout, modelState.View())
	}

	return execErr
}

func dispatchTuiMessage(interactive bool, program *tea.Program, state *tui.Model, msg tea.Msg) {
	if interactive {
		if program != nil {
			program.Send(msg)
		}
		return
	}

	updated, _ := state.Update(msg)
	if m, ok := updated.(tui.Model); ok {
		*state = m
	}
}
