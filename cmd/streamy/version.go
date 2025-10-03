package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Display build information",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "Streamy %s\ncommit: %s\nbuilt: %s\n", version, commit, date)
			return nil
		},
	}

	return cmd
}
