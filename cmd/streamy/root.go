package main

import (
	"github.com/spf13/cobra"
)

type rootFlags struct {
	verbose bool
	dryRun  bool
}

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
				if logger != nil {
					logger.Info(ctx, "launching dashboard from root command")
				}
				return runDashboard(ctx, app, logger)
			}
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().BoolVarP(&flags.verbose, "verbose", "v", false, "Enable verbose logging")
	cmd.PersistentFlags().BoolVar(&flags.dryRun, "dry-run", false, "Preview execution without making changes")

	cmd.AddCommand(newApplyCmd(flags, app))
	cmd.AddCommand(newVerifyCmd(flags, app))
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newDashboardCmd(app))
	cmd.AddCommand(newRegistryCmd(flags, app))
	cmd.AddCommand(newRefreshCmd(flags, app))

	return cmd
}
