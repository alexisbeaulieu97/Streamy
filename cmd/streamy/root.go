package main

import (
	"github.com/spf13/cobra"
)

type rootFlags struct {
	verbose bool
	dryRun  bool
}

func newRootCmd() *cobra.Command {
	flags := &rootFlags{}

	cmd := &cobra.Command{
		Use:           "streamy",
		Short:         "Streamy automates environment setup from declarative configs",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().BoolVarP(&flags.verbose, "verbose", "v", false, "Enable verbose logging")
	cmd.PersistentFlags().BoolVar(&flags.dryRun, "dry-run", false, "Preview execution without making changes")

	cmd.AddCommand(newApplyCmd(flags))
	cmd.AddCommand(newVersionCmd())

	return cmd
}
