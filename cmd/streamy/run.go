package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type runOptions struct {
	ConfigPath     string
	PipelineID     string
	Force          bool
	AssumeYes      bool
	NonInteractive bool
	Timeout        time.Duration
}

func newRunCmd(root *rootFlags, app *AppContext) *cobra.Command {
	opts := runOptions{}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a pipeline from a file or the registry",
		Long: strings.TrimSpace(`
Execute a Streamy pipeline from either:

  --file     A local YAML configuration file (single pipeline execution)
  --registry A registered pipeline identifier (<id>@<version>) plus its dependencies

Exactly one of --file or --registry must be supplied.`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return executeRun(cmd, root, app, opts)
		},
	}

	cmd.Flags().StringVar(&opts.ConfigPath, "file", "", "Path to pipeline configuration file")
	cmd.Flags().StringVar(&opts.PipelineID, "registry", "", "Canonical pipeline ID (<id>@<version> or <id>@latest) to run from the registry")
	cmd.Flags().BoolVar(&opts.NonInteractive, "non-interactive", false, "Disable the interactive TUI and print results to stdout")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Execute downstream pipelines despite upstream failures (only valid with --registry)")
	cmd.Flags().BoolVar(&opts.AssumeYes, "yes", false, "Automatically confirm prompts (requires --force)")
	cmd.Flags().DurationVar(&opts.Timeout, "timeout", 0, "Maximum duration for the run command (defaults to root timeout)")

	cmd.MarkFlagsMutuallyExclusive("file", "registry")

	return cmd
}

func executeRun(cmd *cobra.Command, root *rootFlags, app *AppContext, opts runOptions) error {
	if err := validateRunOptions(opts); err != nil {
		return err
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = root.timeout
	}

	nonInteractive := opts.NonInteractive || !term.IsTerminal(int(os.Stdout.Fd()))

	ctx, _ := app.CommandContext(cmd, "command.run")
	execCtx, cancel := deriveExecutionContext(ctx, timeout)
	defer cancel()

	if opts.ConfigPath != "" {
		applyOpts := applyOptions{
			ConfigPath:     opts.ConfigPath,
			DryRun:         root.dryRun,
			NonInteractive: nonInteractive,
			Timeout:        timeout,
		}

		if err := validateApplyOptions(applyOpts); err != nil {
			return err
		}

		return runApply(execCtx, app, applyOpts, app.LoggerFor("command.apply"))
	}

	orchestrateOpts := orchestrateOptions{
		PipelineID:     opts.PipelineID,
		DryRun:         root.dryRun,
		Force:          opts.Force,
		NonInteractive: nonInteractive,
		AssumeYes:      opts.AssumeYes,
		Timeout:        timeout,
	}

	if err := validateOrchestrateOptions(orchestrateOpts); err != nil {
		return err
	}

	return runOrchestrate(execCtx, app, orchestrateOpts, app.LoggerFor("command.orchestrate"), cmd.OutOrStdout())
}

func validateRunOptions(opts runOptions) error {
	fileSet := strings.TrimSpace(opts.ConfigPath) != ""
	registrySet := strings.TrimSpace(opts.PipelineID) != ""

	if !fileSet && !registrySet {
		return fmt.Errorf("either --file or --registry must be specified")
	}

	if fileSet && registrySet {
		return fmt.Errorf("--file and --registry cannot be used together")
	}

	if fileSet {
		if opts.Force {
			return fmt.Errorf("--force can only be used with --registry")
		}

		if opts.AssumeYes {
			return fmt.Errorf("--yes can only be used with --registry")
		}
	}

	if registrySet {
		if !opts.Force && opts.AssumeYes {
			return fmt.Errorf("--yes requires --force when running with --registry")
		}
	}

	return nil
}
