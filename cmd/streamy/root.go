package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

type rootFlags struct {
	verbose bool
	dryRun  bool
	timeout time.Duration
}

var rootDashboardLauncher = runDashboard

func newRootCmd(app *AppContext) *cobra.Command {
	flags := &rootFlags{}

	cmd := &cobra.Command{
		Use:           "streamy",
		Short:         "Streamy automates environment setup from declarative configs",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If no subcommand is provided, launch the dashboard
			if len(args) == 0 {
				ctx, logger := app.CommandContext(cmd, "command.dashboard")

				var cancel context.CancelFunc
				if flags.timeout > 0 {
					ctx, cancel = context.WithTimeout(ctx, flags.timeout)
					defer cancel()
				}

				if logger != nil {
					logger.Info(ctx, "launching dashboard from root command", "command", "dashboard", "source", "root")
				}

				return rootDashboardLauncher(ctx, app, logger)
			}

			return cmd.Help()
		},
	}

	cmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		if app == nil || app.Logger == nil {
			return nil
		}

		level := "warn"
		if flags.verbose {
			level = "info"
		}

		if err := app.Logger.SetLevel(level); err != nil {
			return fmt.Errorf("set log level: %w", err)
		}

		app.Logger.Info(cmd.Context(), "starting streamy command", "pid", os.Getpid())

		return nil
	}

	cmd.PersistentPostRun = func(cmd *cobra.Command, _ []string) {
		if app == nil || app.Logger == nil {
			return
		}

		app.Logger.Info(cmd.Context(), "streamy command completed", "pid", os.Getpid())
	}

	cmd.PersistentFlags().BoolVarP(&flags.verbose, "verbose", "v", false, "Enable verbose logging")
	cmd.PersistentFlags().BoolVar(&flags.dryRun, "dry-run", false, "Preview execution without making changes")
	cmd.PersistentFlags().DurationVar(&flags.timeout, "timeout", 30*time.Minute, "Maximum duration for a command before it is cancelled")

	cmd.AddCommand(newRunCmd(flags, app))
	cmd.AddCommand(newVerifyCmd(flags, app))
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newDashboardCmd(app))
	cmd.AddCommand(newRegistryCmd(flags, app))
	cmd.AddCommand(newRefreshCmd(flags, app))

	return cmd
}
