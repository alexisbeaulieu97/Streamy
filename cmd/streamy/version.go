package main

import (
	"fmt"

	"github.com/alexisbeaulieu97/streamy/internal/components"
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
			cardData := components.CardData{
				Title:       "Streamy",
				Description: "A powerful configuration management and deployment tool",
				Icon:        "🚀",
				Metadata: map[string]string{
					"Version": version,
					"Commit":  commit,
					"Built":   date,
				},
			}

			card := components.StatusCard(cardData, "info")
			fmt.Fprintln(cmd.OutOrStdout(), card.View())
			return nil
		},
	}

	return cmd
}
