# TailwindCSS-inspired Theme System for Streamy

The Streamy theme system has been enhanced to provide TailwindCSS-like styling utilities for terminal interfaces, built on top of LipGloss. This system includes adaptive colors, extended palettes, spacing utilities, and component presets.

## Features

### 1. **Adaptive Color System**
Colors automatically adapt to light/dark terminal backgrounds using `lipgloss.CompleteAdaptiveColor`.

### 2. **Extended Color Palette**
Full TailwindCSS-inspired color scale with 50-900 shades for multiple color families:
- Slate (gray scale)
- Blue, Green, Red, Yellow
- Purple, Cyan

### 3. **Utility-First Styling**
Functions similar to TailwindCSS utilities:
- `Background()` / `Bg()` - Background colors
- `Text()` / `T()` - Text colors
- `Padding()` / `P()` - All-around padding
- `PaddingX()` / `Px()` - Horizontal padding
- `PaddingY()` / `Py()` - Vertical padding
- `Margin()` / `M()` - All-around margin
- `MarginX()` / `Mx()` - Horizontal margin
- `MarginY()` / `My()` - Vertical margin
- `Rounded()` - Border styles
- `FontSize()` - Text sizes
- `FontWeight()` - Font weights

### 4. **Component Presets**
Pre-built styles for common UI components:
- Button styles: primary, secondary, success, error, warning, info, muted
- Alert styles: success, error, warning, info
- Input styles: default, focus
- Card styles: default, border

### 5. **Spacing Scale**
0-8 scale following TailwindCSS conventions where each unit roughly equals 4px.

## Usage Examples

### Basic Color Utilities
```go
// Semantic colors (adaptive)
primaryText := components.Style(
	lipgloss.NewStyle(),
	components.Foreground(components.PalettePrimary),
).Render("Primary text")
successBg := components.Style(
	lipgloss.NewStyle(),
	components.Background(components.PaletteSuccess),
).Render("Success")

// Palette colors (specific shades)
blue500 := components.TextPalette(components.PaletteBlue, components.PaletteShade500).Render("Blue text")
slate100 := components.BackgroundPalette(components.PaletteSlate, components.PaletteShade100).Render("Light bg")
```

### Spacing Utilities
```go
// All-around padding
padded := components.Padding(components.SpacingSizeMedium).Render("Padded text")

// Directional padding
horizontalPad := components.PaddingX(components.SpacingSizeLarge).Render("Horizontally padded")
verticalPad := components.PaddingY(components.SpacingSizeSmall).Render("Vertically\npadded")

// Margins
withMargin := components.Margin(components.SpacingSizeMedium).Render("With margin")
```

### Combined Styling
```go
styled := components.Style(
	lipgloss.NewStyle(),
	components.Background(components.PalettePrimary),
	components.Foreground(components.PaletteSurface),
	components.Typography(components.TypographyVariantEmphasis),
	components.Padding(components.SpacingSizeMedium),
	components.Border(components.BorderVariantRounded),
).Render("Styled text")
```

### Component Presets
```go
// Buttons
button := components.SimpleButton("Click me").View()

// Alerts
alert := components.SuccessAlert("✅ Success!").View()

// Inputs
input := components.InputStyle(components.InputStateFocus).Render("Input field")
```

## Color Reference

### Semantic Palette Slots
- `Primary` / `Secondary` – brand accents
- `Surface` – backgrounds and default foregrounds
- `Success`, `Warning`, `Danger`, `Info` – status colours
- `Neutral` – greys used for chrome and disabled states

### Palette Colors
Available in 50-900 shades:
- `slate-50` to `slate-900`
- `blue-50` to `blue-900`
- `green-50` to `green-900`
- `red-50` to `red-900`
- `yellow-50` to `yellow-900`
- `purple-50` to `purple-900`
- `cyan-50` to `cyan-900`

## Spacing Scale

Spacing helpers accept the `SpacingSize` enum. The default theme maps them as follows:

```text
SpacingSizeNone             -> 0
SpacingSizeExtraSmall       -> 2
SpacingSizeSmall            -> 3
SpacingSizeMedium           -> 4
SpacingSizeLarge            -> 5
SpacingSizeExtraLarge       -> 6
SpacingSizeDoubleExtraLarge -> 7
SpacingSizeTripleExtraLarge -> 8
SpacingSizeQuadExtraLarge   -> 9
```

## Typography
```go
// Text sizes
"text-xs", "text-sm", "text-base", "text-lg", "text-xl", "text-2xl", "text-3xl"

// Font weights
"font-light", "font-normal", "font-medium", "font-semibold", "font-bold"
```

## Border Styles
```go
"none", "normal", "thick", "rounded", "double"
```

## Migration from Old Theme System

The typed helpers keep the theme API self-describing:

```go
// Semantic helpers
components.SemanticColorValue(components.SemanticPrimary)
components.Spacing(components.SpacingScaleMargin, components.SpacingSizeSmall)
components.TypographyStyle(components.TypographyVariantTitle)

// Typed utility wrappers
components.TextSemantic(components.SemanticPrimary)
components.Margin(components.SpacingSizeSmall)
components.TypographyStyle(components.TypographyVariantTextLg)
```

## Advanced Usage

### Custom Colors
```go
// Direct color usage
customStyle := lipgloss.NewStyle().
    Background(components.PaletteColor(components.PaletteBlue, components.PaletteShade600)).
    Foreground(components.SemanticColorValue(components.SemanticForeground))
```

### Theme Access
```go
// Access current theme
theme := components.GetTheme()
blue := theme.Colors.Blue.Color(components.PaletteShade500)

// Use palette colors directly
style := lipgloss.NewStyle().Background(blue)
```

### Custom Themes
```go
// Create custom theme
customTheme := components.Theme{
    Primary: lipgloss.CompleteAdaptiveColor{
        Light: lipgloss.CompleteColor{TrueColor: "#custom-color", ANSI256: "123", ANSI: "5"},
        Dark:  lipgloss.CompleteColor{TrueColor: "#custom-dark", ANSI256: "124", ANSI: "13"},
    },
    // ... other theme properties
}
components.SetTheme(customTheme)
```

This enhanced theme system provides a powerful, TailwindCSS-inspired approach to styling terminal interfaces while maintaining the elegance and simplicity of LipGloss.
