package main

import (
	"fmt"

	"github.com/alexisbeaulieu97/streamy/internal/ui/components"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	fmt.Println("=== Component Library Showcase ===")
	fmt.Println()

	// 1. Basic Text Components
	fmt.Println("--- Text Components ---")
	text := components.BoldText("Hello, Components!")
	fmt.Println(text.View())

	faintText := components.NewText("This is faint text").WithAppliers(components.Typography(components.TypographyVariantTextSm))
	fmt.Println(faintText.View())

	styledText := components.NewText("Themed Text").
		WithAppliers(
			components.Foreground(components.PalettePrimary),
			components.Typography(components.TypographyVariantEmphasis),
		)
	fmt.Println(styledText.View())
	fmt.Println()

	// 2. Headers
	fmt.Println("--- Headers ---")
	h1 := components.NewHeader("Main Title").
		WithAppliers(components.Typography(components.TypographyVariantTitle))
	fmt.Println(h1.View())

	h2 := components.NewHeader("Subtitle").
		WithSubtitle("With additional context")
	fmt.Println(h2.View())
	fmt.Println()

	// 3. Dividers
	fmt.Println("--- Dividers ---")
	divider := components.HorizontalDivider().WithWidth(50)
	fmt.Println(divider.View())

	dashedDivider := components.DashedDivider().WithWidth(50)
	fmt.Println(dashedDivider.View())

	thickDivider := components.ThickDivider().WithWidth(50)
	fmt.Println(thickDivider.View())
	fmt.Println()

	// 4. Buttons
	fmt.Println("--- Buttons ---")
	primaryBtn := components.PrimaryButton("Primary")
	fmt.Println(primaryBtn.View())

	secondaryBtn := components.SecondaryButton("Secondary")
	fmt.Println(secondaryBtn.View())

	successBtn := components.SuccessButton("Success")
	fmt.Println(successBtn.View())

	errorBtn := components.ErrorButton("Error")
	fmt.Println(errorBtn.View())

	warningBtn := components.WarningButton("Warning")
	fmt.Println(warningBtn.View())

	infoBtn := components.InfoButton("Info")
	fmt.Println(infoBtn.View())

	disabledBtn := components.PrimaryButton("Disabled").WithDisabled(true)
	fmt.Println(disabledBtn.View())
	fmt.Println()

	// 5. Badges
	fmt.Println("--- Badges ---")
	primaryBadge := components.PrimaryBadge("v1.0.0")
	fmt.Println(primaryBadge.View())

	successBadge := components.SuccessBadge("Active")
	fmt.Println(successBadge.View())

	warningBadge := components.WarningBadge("Beta")
	fmt.Println(warningBadge.View())

	errorBadge := components.ErrorBadge("Deprecated")
	fmt.Println(errorBadge.View())
	fmt.Println()

	// 6. Stack Layout (Horizontal)
	fmt.Println("--- Horizontal Stack ---")
	hstack := components.HStack(
		components.PrimaryButton("Left"),
		components.SecondaryButton("Middle"),
		components.SuccessButton("Right"),
	).WithGap(2)
	fmt.Println(hstack.View())
	fmt.Println()

	// 7. Stack Layout (Vertical)
	fmt.Println("--- Vertical Stack ---")
	vstack := components.VStack(
		components.BoldText("First item"),
		components.NewText("Second item"),
		components.NewText("Third item").WithAppliers(components.Typography(components.TypographyVariantTextSm)),
	).WithGap(1)
	fmt.Println(vstack.View())
	fmt.Println()

	// 8. Alerts
	fmt.Println("--- Alerts ---")
	successAlert := components.SuccessAlert("Operation completed successfully!")
	fmt.Println(successAlert.View())
	fmt.Println()

	warningAlert := components.WarningAlert("Warning: This action cannot be undone")
	fmt.Println(warningAlert.View())
	fmt.Println()

	errorAlert := components.ErrorAlert("Error: Failed to connect to server")
	fmt.Println(errorAlert.View())
	fmt.Println()

	infoAlert := components.InfoAlert("Tip: You can use keyboard shortcuts")
	fmt.Println(infoAlert.View())
	fmt.Println()

	// 9. Cards
	fmt.Println("--- Cards ---")
	simpleCard := components.NewCard(
		components.NewHeader("Simple Card"),
		components.HorizontalDivider(),
		components.NewText("This is a card with some content"),
	)
	fmt.Println(simpleCard.View())
	fmt.Println()

	cardWithTitle := components.NewCard(
		components.NewText("Card content goes here"),
		components.NewText("More content below"),
	).WithTitle("Card with Title")
	fmt.Println(cardWithTitle.View())
	fmt.Println()

	// 10. Panels
	fmt.Println("--- Panels ---")
	panel := components.NewPanel(
		components.NewText("Panel content"),
		components.NewText("Panels are lighter than cards"),
	).WithTitle("Information Panel")
	fmt.Println(panel.View())
	fmt.Println()

	// 11. Container
	fmt.Println("--- Custom Container ---")
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
	fmt.Println(container.View())
	fmt.Println()

	// 12. Complex Composition
	fmt.Println("--- Complex Example ---")
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
	fmt.Println(complexCard.View())
	fmt.Println()

	// 13. Spacers
	fmt.Println("--- Spacers ---")
	stackWithSpacers := components.HStack(
		components.NewText("Left"),
		components.HorizontalSpacer(10),
		components.NewText("Right"),
	)
	fmt.Println(stackWithSpacers.View())
	fmt.Println()

	// 14. Theme Switching
	fmt.Println("--- Theme Switching ---")
	fmt.Println("Default Theme:")
	themedCard := components.NewCard(
		components.NewText("Themed content"),
	).WithTitle("Themed Card")
	fmt.Println(themedCard.View())

	fmt.Println("\nDark Theme:")
	// Create a render context with dark theme
	darkCtx := components.DefaultContext().WithTheme(components.DarkTheme())
	themedCardDark := components.NewCard(
		components.NewText("Themed content"),
	).WithTitle("Themed Card")
	fmt.Println(themedCardDark.ViewWithContext(darkCtx))

	// Default theme is used automatically in View()
}
