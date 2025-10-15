package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/alexisbeaulieu97/streamy/internal/app/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/logger"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/tui"
	validationpkg "github.com/alexisbeaulieu97/streamy/internal/validation"
)

type applyOptions struct {
	ConfigPath     string
	DryRun         bool
	Verbose        bool
	NonInteractive bool
}

func newApplyCmd(root *rootFlags, app *AppContext) *cobra.Command {
	opts := applyOptions{}

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply a Streamy configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.DryRun = root.dryRun
			opts.Verbose = root.verbose
			opts.NonInteractive = !term.IsTerminal(int(os.Stdout.Fd()))

			if err := validateApplyOptions(opts); err != nil {
				return err
			}

			return runApply(app, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.ConfigPath, "config", "c", "", "Path to configuration file")
	cmd.MarkFlagRequired("config") //nolint:errcheck

	return cmd
}

func runApply(app *AppContext, opts applyOptions) error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	service := app.Pipeline

	prepared, err := service.Prepare(opts.ConfigPath)
	if err != nil {
		return err
	}

	cfg := prepared.Config
	plan := prepared.Plan

	effectiveVerbose := opts.Verbose || cfg.Settings.Verbose

	level := "info"
	if effectiveVerbose {
		level = "debug"
	}

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

	_, execErr := service.Apply(ctx, pipeline.ApplyRequest{
		Prepared:        prepared,
		ConfigPath:      opts.ConfigPath,
		LoggerOptions:   logger.Options{Level: level, HumanReadable: true},
		DryRunOverride:  opts.DryRun,
		VerboseOverride: opts.Verbose,
		OnStepResult: func(res model.StepResult) {
			dispatchTuiMessage(interactive, program, &modelState, tui.StepCompleteMsg{Result: res})
		},
		OnValidation: func(result validationpkg.ValidationResult) {
			dispatchTuiMessage(interactive, program, &modelState, tui.ValidationMsg{Passed: result.Passed, Message: result.Message})
		},
	})

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
