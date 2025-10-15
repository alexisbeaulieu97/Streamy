// Package components provides a declarative, theme-aware UI component library for terminal applications.
//
// # Overview
//
// This package offers a React-inspired component model with Tailwind-style utilities, built on top of
// lipgloss for terminal UI rendering. All components are themeable, composable, and type-safe.
//
// # Architecture
//
// The component system has three layers:
//
//  1. Theme Layer - Immutable theme definitions (colors, spacing, typography)
//  2. Modifier Layer - StyleFunc transformations that apply theme data to styles
//  3. Component Layer - Composable UI elements that render to strings
//
// # Theme System
//
// Themes are immutable and passed explicitly through RenderContext, eliminating global state:
//
//	theme := components.DefaultTheme()
//	ctx := components.DefaultContext().WithTheme(theme)
//	output := component.ViewWithContext(ctx)
//
// For simple cases, View() uses the default theme automatically:
//
//	output := component.View()
//
// # Core Components
//
// Primitive components:
//   - Text: Styled text content
//   - Spacer: Empty space for layout
//   - Divider: Visual separators
//
// Layout components:
//   - Stack: Vertical/horizontal arrangement with gaps and alignment
//   - Container: Generic box with borders and padding
//
// Semantic components:
//   - Card: Styled container for grouped content
//   - Panel: Lighter container for sections
//   - Button: Interactive button (visual-only for now)
//   - Badge: Status indicators
//   - Alert: Notification messages
//   - Header: Titles and headings
//
// # Style Modifiers
//
// Components accept theme-aware style functions through WithAppliers:
//
//	card := NewCard().WithAppliers(
//		Background(PalettePrimary),
//		Padding(SpacingSizeLarge),
//		Border(BorderVariantRounded),
//	)
//
// Available modifiers:
//   - Background(slot): Semantic background color with matching foreground
//   - Foreground(slot): Semantic text color
//   - Border(variant): Border style from theme
//   - Padding/PaddingX/PaddingY(size): Spacing from theme scale
//   - Margin/MarginX/MarginY(size): Margin from theme scale
//   - Typography(variant): Typography preset from theme
//
// # Composition
//
// Components compose naturally through the Renderable interface:
//
//	content := VStack(
//		NewHeader("Dashboard"),
//		HorizontalDivider(),
//		NewCard(
//			NewText("Status: Active").Bold(),
//			SuccessBadge("Running"),
//		).WithTitle("System Status"),
//	).WithGap(1)
//
// # Context-Based Rendering
//
// All components implement both View() and ViewWithContext() methods:
//
//	// Simple rendering with default theme
//	output := component.View()
//
//	// Explicit theme and layout constraints
//	ctx := RenderContext{
//		Theme: customTheme,
//		Constraints: WithMaxWidth(80),
//		ParentWidth: 100,
//	}
//	output := component.ViewWithContext(ctx)
//
// # Type Safety
//
// The package uses typed enums instead of magic strings:
//
//	SpacingSize:        SpacingSizeSmall, SpacingSizeMedium, etc.
//	ButtonVariant:      ButtonVariantPrimary, ButtonVariantSecondary, etc.
//	PaletteSlot:        PalettePrimary, PaletteSuccess, etc.
//	BorderVariant:      BorderVariantRounded, BorderVariantThick, etc.
//	TypographyVariant: TypographyVariantTitle, TypographyVariantBody, etc.
//
// This provides compile-time safety and excellent IDE autocomplete support.
//
// # Custom Themes
//
// Create custom themes by modifying the default:
//
//	customTheme := components.DefaultTheme()
//	customTheme.Palette.Primary = components.ColourSet{
//		Base:   lipgloss.AdaptiveColor{Light: "#ff0000", Dark: "#ff5555"},
//		OnBase: lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#000000"},
//		// ... other color definitions
//	}
//	customTheme = customTheme.Normalize() // Ensure all fields are initialized
//
// # Performance
//
// Themes are immutable value types, avoiding expensive cloning. Rendering is stateless and
// deterministic - the same component with the same context always produces the same output.
//
// # Examples
//
// See the examples/components directory for comprehensive usage examples.
package components
