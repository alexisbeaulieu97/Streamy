package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/engine"
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

var applyCmdRunner = runApply

func newApplyCmd(root *rootFlags) *cobra.Command {
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

			return applyCmdRunner(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.ConfigPath, "config", "c", "", "Path to configuration file")
	cmd.MarkFlagRequired("config") //nolint:errcheck

	return cmd
}

func runApply(opts applyOptions) error {
	cfg, err := config.ParseConfig(opts.ConfigPath)
	if err != nil {
		return err
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	graph, err := engine.BuildDAG(cfg.Steps)
	if err != nil {
		return err
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	plan, err := engine.GeneratePlan(graph)
	if err != nil {
		return err
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	effectiveDryRun := opts.DryRun || cfg.Settings.DryRun
	effectiveVerbose := opts.Verbose || cfg.Settings.Verbose

	level := "info"
	if effectiveVerbose {
		level = "debug"
	}

	log, err := logger.New(logger.Options{Level: level, HumanReadable: true})
	if err != nil {
		return err
	}

	parallel := cfg.Settings.Parallel
	if parallel <= 0 {
		parallel = 4
	}

	execCtx := &engine.ExecutionContext{
		Config:          cfg,
		DryRun:          effectiveDryRun,
		Verbose:         effectiveVerbose,
		ContinueOnError: cfg.Settings.ContinueOnError,
		WorkerPool:      make(chan struct{}, parallel),
		Results:         make(map[string]*model.StepResult),
		Logger:          log,
		Context:         ctx,
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

	results, execErr := engine.Execute(execCtx, plan)
	for _, res := range results {
		dispatchTuiMessage(interactive, program, &modelState, tui.StepCompleteMsg{Result: res})
	}

	var valErr error
	if len(cfg.Validations) > 0 {
		validationResults, err := validationpkg.RunValidations(ctx, cfg.Validations)
		valErr = err
		for _, vr := range validationResults {
			dispatchTuiMessage(interactive, program, &modelState, tui.ValidationMsg{Passed: vr.Passed, Message: vr.Message})
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
		fmt.Fprintln(os.Stdout, modelState.View())
	}

	if execErr != nil {
		return execErr
	}
	if valErr != nil {
		return valErr
	}

	return nil
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
