package main

import "github.com/spf13/cobra"

func newRegistryCmd(rootFlags *rootFlags, app *AppContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registry",
		Short: "Manage Streamy pipeline registry",
		Long:  "Manage the Streamy pipeline registry, including adding, listing, removing, refreshing, and showing pipeline details.",
		Aliases: []string{
			"reg",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newAddCmd(rootFlags))
	cmd.AddCommand(newListCmd(rootFlags))
	cmd.AddCommand(newRemoveCmd(rootFlags))
	cmd.AddCommand(newRefreshCmd(rootFlags, app))
	cmd.AddCommand(newShowCmd(rootFlags))

	return cmd
}
