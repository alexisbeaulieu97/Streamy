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
		RunE: func(cmd *cobra.Command, _ []string) error {
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

	if err := cmd.MarkFlagRequired("config"); err != nil {
		panic(fmt.Sprintf("failed to mark config flag required: %v", err))
	}

	return cmd
}

func runApply(ctx context.Context, app *AppContext, opts applyOptions, logger ports.Logger) error {
	if ctx == nil {
		return fmt.Errorf("context is required")
	}

	execCtx, cancel := deriveExecutionContext(ctx, opts.Timeout)
	defer cancel()

	defer logApplyLifecycle(execCtx, logger, opts)()

	pipelineDomain, planDomain, err := app.PrepareUseCase.Prepare(execCtx, opts.ConfigPath)
	if err != nil {
		return newCommandError(
			"apply",
			"preparing pipeline configuration",
			err,
			"Fix the configuration errors shown above and try again.",
		)
	}

	modelState := tui.NewModel(pipelineDomain, planDomain, opts.NonInteractive)
	session := newApplySession(execCtx, !opts.NonInteractive, &modelState)

	_, domainResults, summary, execErr := app.ApplyUseCase.Apply(execCtx, opts.ConfigPath, opts.DryRun)

	for _, res := range domainResults {
		state := pipelineconv.ToStepState(res, opts.DryRun)
		session.Dispatch(tui.StepCompleteMsg{StepID: res.StepID, State: state})
	}

	if summary != nil {
		for _, validation := range summary.Results {
			session.Dispatch(tui.ValidationMsg{
				Passed:  validation.IsSatisfied(),
				Message: validation.FormatMessage(),
			})
		}
	}

	if err := session.Close(); err != nil {
		return err
	}

	if execErr != nil {
		return newCommandError(
			"apply",
			"executing pipeline",
			execErr,
			"Inspect the reported step failures, fix them, then re-run.",
		)
	}

	return nil
}

func deriveExecutionContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout > 0 {
		return context.WithTimeout(ctx, timeout)
	}

	return context.WithCancel(ctx)
}

func logApplyLifecycle(ctx context.Context, logger ports.Logger, opts applyOptions) func() {
	if logger == nil {
		return func() {}
	}

	logger.Info(ctx, "command.apply.start", "config_path", opts.ConfigPath, "dry_run", opts.DryRun, "non_interactive", opts.NonInteractive)

	return func() {
		logger.Info(ctx, "command.apply.finish", "config_path", opts.ConfigPath)
	}
}

type applySession struct {
	interactive bool
	program     *tea.Program
	programErr  error
	programDone chan struct{}
	state       *tui.Model
}

func newApplySession(ctx context.Context, interactive bool, state *tui.Model) *applySession {
	session := &applySession{
		interactive: interactive,
		state:       state,
	}

	if !interactive {
		return session
	}

	session.program = tea.NewProgram(*state)
	session.programDone = make(chan struct{})

	go func() {
		_, session.programErr = session.program.Run()
		close(session.programDone)
	}()

	go func() {
		select {
		case <-ctx.Done():
			session.program.Send(tea.QuitMsg{})
		case <-session.programDone:
		}
	}()

	return session
}

func (s *applySession) Dispatch(msg tea.Msg) {
	if s.interactive {
		if s.program != nil {
			s.program.Send(msg)
		}

		return
	}

	updated, _ := s.state.Update(msg)
	if m, ok := updated.(tui.Model); ok {
		*s.state = m
	}
}

func (s *applySession) Close() error {
	if s.interactive {
		if s.program != nil {
			s.program.Send(tea.QuitMsg{})
		}

		if s.programDone != nil {
			select {
			case <-s.programDone:
			case <-time.After(250 * time.Millisecond):
			}
		}

		if s.programErr != nil {
			return newCommandError(
				"apply",
				"running interactive UI",
				s.programErr,
				"Re-run with --non-interactive if the TUI cannot start.",
			)
		}

		return nil
	}

	_, _ = fmt.Fprintln(os.Stdout, s.state.View())

	return nil
}
