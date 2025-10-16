package main

import (
	"fmt"

	"github.com/alexisbeaulieu97/streamy/internal/ui/components"
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
			header := components.NewHeader("Streamy")
			divider := components.NewDivider().WithChar("─")
			info := components.NewText("Version: " + version).WithAppliers(components.Typography(components.TypographyVariantTextSm))
			commitText := components.NewText("Commit: " + commit).WithAppliers(components.Typography(components.TypographyVariantTextSm))
			dateText := components.NewText("Built: " + date).WithAppliers(components.Typography(components.TypographyVariantTextSm))

			card := components.NewCard(header, divider, info, commitText, dateText)
			fmt.Println(card.View())

			return nil
		},
	}

	return cmd
}
