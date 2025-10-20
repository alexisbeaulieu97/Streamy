// Package main renders a console showcase of TUI components.
package main

import (
	"bufio"
	"os"

	"github.com/alexisbeaulieu97/streamy/internal/ui/components"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	writer := bufio.NewWriter(os.Stdout)
	printLine(writer, "=== Component Library Showcase ===")
	printLine(writer, "")

	// 1. Basic Text Components
	printLine(writer, "--- Text Components ---")

	text := components.BoldText("Hello, Components!")
	printLine(writer, text.View())

	faintText := components.NewText("This is faint text").WithAppliers(components.Typography(components.TypographyVariantTextSm))
	printLine(writer, faintText.View())

	styledText := components.NewText("Themed Text").
		WithAppliers(
			components.Foreground(components.PalettePrimary),
			components.Typography(components.TypographyVariantEmphasis),
		)
	printLine(writer, styledText.View())
	printLine(writer, "")

	// 2. Headers
	printLine(writer, "--- Headers ---")

	h1 := components.NewHeader("Main Title").
		WithAppliers(components.Typography(components.TypographyVariantTitle))
	printLine(writer, h1.View())

	h2 := components.NewHeader("Subtitle").
		WithSubtitle("With additional context")
	printLine(writer, h2.View())
	printLine(writer, "")

	// 3. Dividers
	printLine(writer, "--- Dividers ---")

	divider := components.HorizontalDivider().WithWidth(50)
	printLine(writer, divider.View())

	dashedDivider := components.DashedDivider().WithWidth(50)
	printLine(writer, dashedDivider.View())

	thickDivider := components.ThickDivider().WithWidth(50)
	printLine(writer, thickDivider.View())
	printLine(writer, "")

	// 4. Buttons
	printLine(writer, "--- Buttons ---")

	primaryBtn := components.PrimaryButton("Primary")
	printLine(writer, primaryBtn.View())

	secondaryBtn := components.SecondaryButton("Secondary")
	printLine(writer, secondaryBtn.View())

	successBtn := components.SuccessButton("Success")
	printLine(writer, successBtn.View())

	errorBtn := components.ErrorButton("Error")
	printLine(writer, errorBtn.View())

	warningBtn := components.WarningButton("Warning")
	printLine(writer, warningBtn.View())

	infoBtn := components.InfoButton("Info")
	printLine(writer, infoBtn.View())

	disabledBtn := components.PrimaryButton("Disabled").WithDisabled(true)
	printLine(writer, disabledBtn.View())
	printLine(writer, "")

	// 5. Badges
	printLine(writer, "--- Badges ---")

	primaryBadge := components.PrimaryBadge("v1.0.0")
	printLine(writer, primaryBadge.View())

	successBadge := components.SuccessBadge("Active")
	printLine(writer, successBadge.View())

	warningBadge := components.WarningBadge("Beta")
	printLine(writer, warningBadge.View())

	errorBadge := components.ErrorBadge("Deprecated")
	printLine(writer, errorBadge.View())
	printLine(writer, "")

	// 6. Stack Layout (Horizontal)
	printLine(writer, "--- Horizontal Stack ---")

	hstack := components.HStack(
		components.PrimaryButton("Left"),
		components.SecondaryButton("Middle"),
		components.SuccessButton("Right"),
	).WithGap(2)
	printLine(writer, hstack.View())
	printLine(writer, "")

	// 7. Stack Layout (Vertical)
	printLine(writer, "--- Vertical Stack ---")

	vstack := components.VStack(
		components.BoldText("First item"),
		components.NewText("Second item"),
		components.NewText("Third item").WithAppliers(components.Typography(components.TypographyVariantTextSm)),
	).WithGap(1)
	printLine(writer, vstack.View())
	printLine(writer, "")

	// 8. Alerts
	printLine(writer, "--- Alerts ---")

	successAlert := components.SuccessAlert("Operation completed successfully!")
	printLine(writer, successAlert.View())
	printLine(writer, "")

	warningAlert := components.WarningAlert("Warning: This action cannot be undone")
	printLine(writer, warningAlert.View())
	printLine(writer, "")

	errorAlert := components.ErrorAlert("Error: Failed to connect to server")
	printLine(writer, errorAlert.View())
	printLine(writer, "")

	infoAlert := components.InfoAlert("Tip: You can use keyboard shortcuts")
	printLine(writer, infoAlert.View())
	printLine(writer, "")

	// 9. Cards
	printLine(writer, "--- Cards ---")

	simpleCard := components.NewCard(
		components.NewHeader("Simple Card"),
		components.HorizontalDivider(),
		components.NewText("This is a card with some content"),
	)
	printLine(writer, simpleCard.View())
	printLine(writer, "")

	cardWithTitle := components.NewCard(
		components.NewText("Card content goes here"),
		components.NewText("More content below"),
	).WithTitle("Card with Title")
	printLine(writer, cardWithTitle.View())
	printLine(writer, "")

	// 10. Panels
	printLine(writer, "--- Panels ---")

	panel := components.NewPanel(
		components.NewText("Panel content"),
		components.NewText("Panels are lighter than cards"),
	).WithTitle("Information Panel")
	printLine(writer, panel.View())
	printLine(writer, "")

	// 11. Container
	printLine(writer, "--- Custom Container ---")

	container := components.NewContainer(
		components.BoldText("Custom styled container"),
		components.HorizontalDivider(),
		components.NewText("With padding and border"),
	).
		WithBorder(lipgloss.DoubleBorder()).
		WithBorderColor("#3b82f6").
		WithPadding(components.UniformSpacing(2)).
		WithAppliers(
			components.Background(components.PaletteSurface),
		)
	printLine(writer, container.View())
	printLine(writer, "")

	// 12. Complex Composition
	printLine(writer, "--- Complex Example ---")

	complexCard := components.NewCard(
		components.VStack(
			components.HStack(
				components.NewHeader("Dashboard"),
				components.SuccessBadge("Live"),
			).WithGap(2),
			components.HorizontalDivider(),
			components.NewText("System Status: All services operational"),
			components.VStack(
				components.HStack(
					components.EmphasisText("CPU:"),
					components.SuccessBadge("45%"),
				).WithGap(2),
				components.HStack(
					components.EmphasisText("Memory:"),
					components.WarningBadge("78%"),
				).WithGap(2),
				components.HStack(
					components.EmphasisText("Disk:"),
					components.SuccessBadge("32%"),
				).WithGap(2),
			).WithGap(1),
			components.HorizontalDivider(),
			components.HStack(
				components.PrimaryButton("Refresh"),
				components.SecondaryButton("Details"),
			).WithGap(2),
		).WithGap(1),
	)
	printLine(writer, complexCard.View())
	printLine(writer, "")

	// 13. Spacers
	printLine(writer, "--- Spacers ---")

	stackWithSpacers := components.HStack(
		components.NewText("Left"),
		components.HorizontalSpacer(10),
		components.NewText("Right"),
	)
	printLine(writer, stackWithSpacers.View())
	printLine(writer, "")

	// 14. Theme Switching
	printLine(writer, "--- Theme Switching ---")
	printLine(writer, "Default Theme:")

	themedCard := components.NewCard(
		components.NewText("Themed content"),
	).WithTitle("Themed Card")
	printLine(writer, themedCard.View())

	printLine(writer, "\nDark Theme:")
	// Create a render context with dark theme
	darkCtx := components.DefaultContext().WithTheme(components.DarkTheme())
	themedCardDark := components.NewCard(
		components.NewText("Themed content"),
	).WithTitle("Themed Card")
	printLine(writer, themedCardDark.ViewWithContext(darkCtx))

	// Default theme is used automatically in View()
	_ = writer.Flush()
}

func printLine(writer *bufio.Writer, line string) {
	_, _ = writer.WriteString(line + "\n")
}
