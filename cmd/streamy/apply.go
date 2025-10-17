package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/alexisbeaulieu97/streamy/internal/pipelineconv"
	"github.com/alexisbeaulieu97/streamy/internal/tui"
)

type applyOptions struct {
	ConfigPath          string
	DryRun              bool
	Verbose             bool
	NonInteractive      bool
	ForceNonInteractive bool
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

			if err := validateApplyOptions(opts); err != nil {
				return err
			}

			return runApply(cmd.Context(), app, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.ConfigPath, "config", "c", "", "Path to configuration file")
	cmd.Flags().BoolVar(&opts.ForceNonInteractive, "non-interactive", false, "Disable the interactive TUI and print results to stdout")
	cmd.MarkFlagRequired("config") //nolint:errcheck

	return cmd
}

func runApply(ctx context.Context, app *AppContext, opts applyOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	pipelineDomain, planDomain, err := app.PrepareUseCase.Prepare(ctx, opts.ConfigPath)
	if err != nil {
		return err
	}

	cfg := pipelineconv.ConvertPipelineToConfig(pipelineDomain)
	plan := pipelineconv.ConvertPlanToEngine(planDomain)

	modelState := tui.NewModel(cfg, plan, opts.NonInteractive)
	interactive := !opts.NonInteractive

	var program *tea.Program
	var programErr error
	done := make(chan struct{})

	if interactive {
		program = tea.NewProgram(modelState)
		go func() {
			_, programErr = program.Run()
			close(done)
		}()
	}

	_, domainResults, summary, execErr := app.ApplyUseCase.Apply(ctx, opts.ConfigPath, opts.DryRun)

	for _, res := range domainResults {
		modelRes := pipelineconv.ConvertStepResult(res, opts.DryRun)
		dispatchTuiMessage(interactive, program, &modelState, tui.StepCompleteMsg{Result: modelRes})
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
		<-done
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
