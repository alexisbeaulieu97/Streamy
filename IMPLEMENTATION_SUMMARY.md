# Streamy Fluent Theme Refactor – Summary

## Overview

The components theme subsystem now centres on a lean, type-safe architecture:

1. A single `Theme` struct carrying semantic palette slots, spacing tables, typography presets, borders, and input styles.
2. Fluent modifiers that translate those theme values into reusable `StyleApplier`s.
3. Component helpers (`CardBaseStyle()`, `ButtonPrimaryStyle()`, etc.) implemented as functions so they always honour the active theme.

Legacy string-based helpers and the token translator were removed to keep the Go surface area explicit and predictable.

## Key Changes

- **Palette**: Semantic colour slots are stored in `Palette`, each backed by a `ColourSet` (base, on-base, muted, contrast). `DefaultTheme()` and `DarkTheme()` populate these slots directly.
- **Spacing**: `SpacingSize` enums feed into `PaddingValue()` / `MarginValue()` and the fluent modifiers (`Padding`, `PaddingX`, `Margin`, …).
- **Typography**: Presets live in `TypographyScale`; modifiers accept `TypographyVariant` values instead of function pointers.
- **Borders**: `Border(BorderVariant…)` switches on the current theme’s border set.
- **Style Bundles**: Card, button, and alert presets are now functions returning fresh slices, preventing shared-slice mutation bugs.
- **Components**: Buttons and alerts compute their styles lazily in `View()`, support disabled/focus variations, and rely entirely on the fluent modifiers.
- **Tests**: Updated to exercise the new palette, spacing helpers, component states, and to drop translator-specific cases.

## Usage Snapshot

```go
badge := components.Style(
	lipgloss.NewStyle(),
	components.Background(components.PaletteSuccess),
	components.Border(components.BorderVariantRounded),
	components.PaddingX(components.SpacingSizeSmall),
	components.Typography(components.TypographyVariantEmphasis),
)

button := components.SimpleButton("Deploy").WithFocus(true).View()
alert  := components.ErrorAlert("Build failed").View()
```

## Testing

`go test ./internal/components` now covers:

- palette slot access and spacing helpers
- typography modifiers and input styles
- button/alert rendering across state changes
- concurrent theme reads
- card text wrapping expectations

All tests pass.

