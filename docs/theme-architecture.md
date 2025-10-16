% Streamy Components Theming – Architecture

# Motivation

- Keep the day-to-day authoring experience “ridiculously simple” while respecting Go’s strengths.
- Promote cohesive, theme-driven styling by routing everything through semantic palette slots.
- Reduce hidden global state and favour explicit, type-safe APIs.
- Make it easy to extend the system (new components, new palette slots) without touching existing call sites.

# Architecture Overview

The theming system now has two cooperating layers:

1. **Core Theme** – A strongly typed description of palette colours, spacing, borders, typography, and component presets.
2. **Fluent Modifiers** – Small composable helpers that transform a `lipgloss.Style` using data from the active theme.

Typed data flows downward from the theme into modifiers and finally into components.

```
┌─────────────┐
│   Theme(s)  │       Palette / Spacing / Typography values
└─────┬───────┘
      │
┌─────▼───────┐
│  Modifiers  │  Background(), Border(), Padding(), Typography(), …
└─────┬───────┘
      │
┌─────▼───────┐
│ Components  │  Buttons, Alerts, Cards, custom widgets
└─────────────┘
```

# Core Theme

```go
type ColourSet struct {
	Base     lipgloss.AdaptiveColor // background or primary tone
	OnBase   lipgloss.AdaptiveColor // text/icon colour for Base
	Muted    lipgloss.AdaptiveColor // desaturated variant
	Contrast lipgloss.AdaptiveColor // accent colour that pops on Base
}

type Palette struct {
	Primary, Secondary, Surface ColourSet
	Success, Warning, Danger    ColourSet
	Info, Neutral               ColourSet
}

type BorderSet struct {
	None    lipgloss.Border
	Normal  lipgloss.Border
	Rounded lipgloss.Border
	Thick   lipgloss.Border
	Double  lipgloss.Border
}

type TypographyScale struct {
	Base, Title, Subtitle, Body lipgloss.Style
	Code, Emphasis              lipgloss.Style
	TextXs, TextSm, TextBase    lipgloss.Style
	TextLg, TextXl, Text2Xl     lipgloss.Style
	Text3Xl                     lipgloss.Style
	FontLight, FontNormal       lipgloss.Style
	FontMedium, FontSemibold    lipgloss.Style
	FontBold                    lipgloss.Style
}

type SpacingConfig struct {
	Padding [SpacingSizeQuadExtraLarge + 1]int
	Margin  [SpacingSizeQuadExtraLarge + 1]int
}

type Theme struct {
	Palette    Palette
	Colors     ColorPalette // tailwind-style colour families for direct shade lookups
	Borders    BorderSet
	Spacing    SpacingConfig
	Typography TypographyScale
	Input      InputStyles
}
```

A `ThemeManager` keeps an atomic copy of the active theme:

```go
var defaultThemeManager = NewThemeManager(DefaultTheme())

func SetTheme(theme Theme) { defaultThemeManager.SetTheme(theme) }
func GetTheme() Theme      { return defaultThemeManager.Theme() }
```

# Fluent Modifiers

Modifiers remain `StyleApplier`s. Each helper reads the current theme and mutates a base style.

```go
func Background(slot PaletteSlot) StyleFunc {
	return func(base lipgloss.Style, theme Theme) lipgloss.Style {
		cs := slot(theme.Palette)
		return base.Background(cs.Base).Foreground(cs.OnBase)
	}
}

func Border(variant BorderVariant) StyleFunc {
	return func(base lipgloss.Style, theme Theme) lipgloss.Style {
		return base.Border(borderForVariant(theme, variant))
	}
}

func Padding(size SpacingSize) StyleFunc {
	return func(base lipgloss.Style, theme Theme) lipgloss.Style {
		value := spacingLookup(theme.Spacing.Padding, size)
		return base.Padding(value)
	}
}

func Typography(variant TypographyVariant) StyleFunc {
	return func(base lipgloss.Style, theme Theme) lipgloss.Style {
		return base.Inherit(TypographyStyle(variant))
	}
}
```

They are chained through `Style`:

```go
badge := Style(
	lipgloss.NewStyle(),
	Background(PaletteSuccess),
	Border(BorderVariantRounded),
	PaddingX(SpacingSizeSmall),
	Typography(TypographyVariantEmphasis),
)
```

# Predefined Bundles

Common component presets are exposed as helper functions that always consult the active theme:

```go
func CardBaseStyle() []StyleApplier {
	return []StyleApplier{
		Background(PaletteSurface),
		Border(BorderVariantRounded),
		Margin(SpacingSizeSmall),
		Padding(SpacingSizeMedium),
	}
}
```

Buttons and alerts simply call those helpers when constructing their styles, ensuring that theme swaps are reflected everywhere with no cached global slices.

# Theme Variants

- `DefaultTheme()` seeds the standard palette (cool blues, warm warn/danger, neutral greys).
- `DarkTheme()` rebalances surface/neutral slots for dark-first UI.
- Additional variants can be added by cloning `DefaultTheme()` and tweaking palette slots.

# Removal of Token Translator

The original Tailwind-like string translator has been removed. All styling is expressed through typed modifiers, which keeps call sites explicit and makes refactoring safe. If configuration files need string tokens in the future, they can live as an optional adapter outside the core package.

# Extending the System

- Add new palette slots by extending `Palette` and supplying matching modifiers (e.g. `PaletteAccent` and `AccentBannerStyle()`).
- Extend `TypographyVariant` and `TypographyScale` together so helpers stay in sync.
- Use `SpacingSize` constants to keep spacing consistent—if you need custom values, update the theme’s `SpacingConfig`.
- Prefer composing existing modifiers in new helpers instead of constructing raw `lipgloss.Style` structs inside components.

This architecture keeps the API small, predictable, and entirely driven by the active theme while remaining easy to grow alongside Streamy.

