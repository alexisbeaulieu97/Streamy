package components

import (
	"github.com/charmbracelet/lipgloss"
)

const paletteShadeCount = 10

// PaletteShades represents a Tailwind-style color scale with 10 shades from lightest to darkest.
// Shades are indexed from 50 (lightest) to 900 (darkest), matching Tailwind's numbering.
type PaletteShades struct {
	colors [paletteShadeCount]lipgloss.Color
}

// NewPaletteShades creates a palette shade scale from the provided colors.
// Colors should be ordered from lightest to darkest. Accepts up to 10 colors.
func NewPaletteShades(colors ...lipgloss.Color) PaletteShades {
	var shades PaletteShades
	for i := 0; i < paletteShadeCount && i < len(colors); i++ {
		shades.colors[i] = colors[i]
	}
	return shades
}

// Color returns the color at the specified shade level.
// Returns an empty string if the shade is out of bounds.
func (ps PaletteShades) Color(shade PaletteShade) lipgloss.Color {
	index := int(shade)
	if index < 0 || index >= paletteShadeCount {
		return ""
	}
	return ps.colors[index]
}

// ColorPalette represents a complete color palette with shades like Tailwind CSS
type ColorPalette struct {
	Slate  PaletteShades
	Blue   PaletteShades
	Green  PaletteShades
	Red    PaletteShades
	Yellow PaletteShades
	Purple PaletteShades
	Cyan   PaletteShades
}

func (cp ColorPalette) Shades(family PaletteFamily) PaletteShades {
	switch family {
	case PaletteSlate:
		return cp.Slate
	case PaletteBlue:
		return cp.Blue
	case PaletteGreen:
		return cp.Green
	case PaletteRed:
		return cp.Red
	case PaletteYellow:
		return cp.Yellow
	case PalettePurple:
		return cp.Purple
	case PaletteCyan:
		return cp.Cyan
	default:
		return cp.Slate
	}
}

// SpacingSize enumerates supported spacing size tokens.
type SpacingSize int

const (
	SpacingSizeNone SpacingSize = iota
	SpacingSizeExtraSmall
	SpacingSizeSmall
	SpacingSizeMedium
	SpacingSizeLarge
	SpacingSizeExtraLarge
	SpacingSizeDoubleExtraLarge
	SpacingSizeTripleExtraLarge
	SpacingSizeQuadExtraLarge
)

// SpacingConfig stores distinct spacing scales for padding and margin.
const spacingSizeCount = int(SpacingSizeQuadExtraLarge) + 1

type spacingTable [spacingSizeCount]int

// SpacingConfig stores distinct spacing scales for padding and margin.
type SpacingConfig struct {
	Margin  spacingTable
	Padding spacingTable
}

// TypographyVariant represents a strongly-typed typography token.
type TypographyVariant int

const (
	TypographyVariantBase TypographyVariant = iota
	TypographyVariantTitle
	TypographyVariantSubtitle
	TypographyVariantBody
	TypographyVariantCode
	TypographyVariantEmphasis

	TypographyVariantTextXs
	TypographyVariantTextSm
	TypographyVariantTextBase
	TypographyVariantTextLg
	TypographyVariantTextXl
	TypographyVariantText2Xl
	TypographyVariantText3Xl

	TypographyVariantFontLight
	TypographyVariantFontNormal
	TypographyVariantFontMedium
	TypographyVariantFontSemibold
	TypographyVariantFontBold
)

type PaletteFamily int

const (
	PaletteSlate PaletteFamily = iota
	PaletteBlue
	PaletteGreen
	PaletteRed
	PaletteYellow
	PalettePurple
	PaletteCyan
)

type PaletteShade int

const (
	PaletteShade50 PaletteShade = iota
	PaletteShade100
	PaletteShade200
	PaletteShade300
	PaletteShade400
	PaletteShade500
	PaletteShade600
	PaletteShade700
	PaletteShade800
	PaletteShade900
)

type BorderVariant int

const (
	BorderVariantNormal BorderVariant = iota
	BorderVariantThick
	BorderVariantRounded
	BorderVariantDouble
)

type ButtonVariant int

const (
	ButtonVariantPrimary ButtonVariant = iota
	ButtonVariantSecondary
	ButtonVariantSuccess
	ButtonVariantError
	ButtonVariantWarning
	ButtonVariantInfo
	ButtonVariantMuted
)

type AlertVariant int

const (
	AlertVariantSuccess AlertVariant = iota
	AlertVariantError
	AlertVariantWarning
	AlertVariantInfo
)

type InputState int

const (
	InputStateDefault InputState = iota
	InputStateFocus
)

// Palette describes semantic colour slots used by components.
type Palette struct {
	Primary   ColourSet
	Secondary ColourSet
	Surface   ColourSet
	Success   ColourSet
	Warning   ColourSet
	Danger    ColourSet
	Info      ColourSet
	Neutral   ColourSet
}

// BorderSet groups reusable border definitions.
type BorderSet struct {
	None    lipgloss.Border
	Normal  lipgloss.Border
	Rounded lipgloss.Border
	Thick   lipgloss.Border
	Double  lipgloss.Border
}

// TypographyScale contains semantic typography presets and weights.
type TypographyScale struct {
	Base     lipgloss.Style
	Title    lipgloss.Style
	Subtitle lipgloss.Style
	Body     lipgloss.Style
	Code     lipgloss.Style
	Emphasis lipgloss.Style

	TextXs   lipgloss.Style
	TextSm   lipgloss.Style
	TextBase lipgloss.Style
	TextLg   lipgloss.Style
	TextXl   lipgloss.Style
	Text2Xl  lipgloss.Style
	Text3Xl  lipgloss.Style

	FontLight    lipgloss.Style
	FontNormal   lipgloss.Style
	FontMedium   lipgloss.Style
	FontSemibold lipgloss.Style
	FontBold     lipgloss.Style
}

// InputStyles describes default/focus styles for input controls.
type InputStyles struct {
	Default lipgloss.Style
	Focus   lipgloss.Style
}

// VariantRegistry maps component variants to their styling strategies.
// This allows themes to define variant styling data-driven rather than code-driven.
type VariantRegistry struct {
	strategies map[interface{}]StyleStrategy
}

// NewVariantRegistry creates a new variant registry.
func NewVariantRegistry() *VariantRegistry {
	return &VariantRegistry{
		strategies: make(map[interface{}]StyleStrategy),
	}
}

// Register adds a variant-to-strategy mapping.
func (vr *VariantRegistry) Register(variant interface{}, strategy StyleStrategy) {
	vr.strategies[variant] = strategy
}

// Get retrieves the strategy for a variant, or nil if not found.
func (vr *VariantRegistry) Get(variant interface{}) StyleStrategy {
	return vr.strategies[variant]
}

// Theme represents an immutable styling theme for components.
// Themes should be created once and reused. All modification operations
// return new theme instances rather than mutating the original.
type Theme struct {
	Palette    Palette
	Colors     ColorPalette
	Borders    BorderSet
	Spacing    SpacingConfig
	Typography TypographyScale
	Input      InputStyles
	Variants   *VariantRegistry
}

// Normalize returns a new theme with all fields properly initialized.
// This ensures that partially-specified themes have sensible defaults.
func (t Theme) Normalize() Theme {
	t.Spacing = normalizeSpacingConfig(t.Spacing)
	return t
}

func normalizeSpacingConfig(cfg SpacingConfig) SpacingConfig {
	if spacingTableIsZero(cfg.Padding) {
		cfg.Padding = defaultSpacingTable()
	}
	if spacingTableIsZero(cfg.Margin) {
		cfg.Margin = defaultSpacingTable()
	}
	return cfg
}

func spacingTableIsZero(table spacingTable) bool {
	for _, value := range table {
		if value != 0 {
			return false
		}
	}
	return true
}

func defaultSpacingTable() spacingTable {
	return spacingTable{
		SpacingSizeNone:             0,
		SpacingSizeExtraSmall:       2,
		SpacingSizeSmall:            3,
		SpacingSizeMedium:           4,
		SpacingSizeLarge:            5,
		SpacingSizeExtraLarge:       6,
		SpacingSizeDoubleExtraLarge: 7,
		SpacingSizeTripleExtraLarge: 8,
		SpacingSizeQuadExtraLarge:   9,
	}
}

// DefaultTheme returns the default theme for components
func DefaultTheme() Theme {
	ac := func(light, dark string) lipgloss.AdaptiveColor {
		return lipgloss.AdaptiveColor{Light: light, Dark: dark}
	}

	palette := Palette{
		Primary: ColourSet{
			Base:     ac("#3b82f6", "#60a5fa"),
			OnBase:   ac("#f8fafc", "#0b1120"),
			Muted:    ac("#2563eb", "#1d4ed8"),
			Contrast: ac("#facc15", "#ca8a04"),
		},
		Secondary: ColourSet{
			Base:     ac("#a855f7", "#c084fc"),
			OnBase:   ac("#f8fafc", "#1f2937"),
			Muted:    ac("#7c3aed", "#6b21a8"),
			Contrast: ac("#f472b6", "#f472b6"),
		},
		Surface: ColourSet{
			Base:     ac("#f9fafb", "#111827"),
			OnBase:   ac("#111827", "#f9fafb"),
			Muted:    ac("#e2e8f0", "#1f2937"),
			Contrast: ac("#3b82f6", "#60a5fa"),
		},
		Success: ColourSet{
			Base:     ac("#22c55e", "#4ade80"),
			OnBase:   ac("#052e16", "#022c22"),
			Muted:    ac("#16a34a", "#15803d"),
			Contrast: ac("#f8fafc", "#f8fafc"),
		},
		Warning: ColourSet{
			Base:     ac("#eab308", "#facc15"),
			OnBase:   ac("#422006", "#422006"),
			Muted:    ac("#ca8a04", "#a16207"),
			Contrast: ac("#111827", "#111827"),
		},
		Danger: ColourSet{
			Base:     ac("#ef4444", "#f87171"),
			OnBase:   ac("#7f1d1d", "#450a0a"),
			Muted:    ac("#dc2626", "#b91c1c"),
			Contrast: ac("#f8fafc", "#f8fafc"),
		},
		Info: ColourSet{
			Base:     ac("#06b6d4", "#22d3ee"),
			OnBase:   ac("#083344", "#04121a"),
			Muted:    ac("#0891b2", "#0e7490"),
			Contrast: ac("#f8fafc", "#f8fafc"),
		},
		Neutral: ColourSet{
			Base:     ac("#64748b", "#94a3b8"),
			OnBase:   ac("#f1f5f9", "#0f172a"),
			Muted:    ac("#475569", "#334155"),
			Contrast: ac("#f8fafc", "#f8fafc"),
		},
	}

	colorFamilies := ColorPalette{
		Slate: NewPaletteShades(
			lipgloss.Color("#f8fafc"),
			lipgloss.Color("#f1f5f9"),
			lipgloss.Color("#e2e8f0"),
			lipgloss.Color("#cbd5e1"),
			lipgloss.Color("#94a3b8"),
			lipgloss.Color("#64748b"),
			lipgloss.Color("#475569"),
			lipgloss.Color("#334155"),
			lipgloss.Color("#1e293b"),
			lipgloss.Color("#0f172a"),
		),
		Blue: NewPaletteShades(
			lipgloss.Color("#eff6ff"),
			lipgloss.Color("#dbeafe"),
			lipgloss.Color("#bfdbfe"),
			lipgloss.Color("#93c5fd"),
			lipgloss.Color("#60a5fa"),
			lipgloss.Color("#3b82f6"),
			lipgloss.Color("#2563eb"),
			lipgloss.Color("#1d4ed8"),
			lipgloss.Color("#1e40af"),
			lipgloss.Color("#1e3a8a"),
		),
		Green: NewPaletteShades(
			lipgloss.Color("#f0fdf4"),
			lipgloss.Color("#dcfce7"),
			lipgloss.Color("#bbf7d0"),
			lipgloss.Color("#86efac"),
			lipgloss.Color("#4ade80"),
			lipgloss.Color("#22c55e"),
			lipgloss.Color("#16a34a"),
			lipgloss.Color("#15803d"),
			lipgloss.Color("#166534"),
			lipgloss.Color("#14532d"),
		),
		Red: NewPaletteShades(
			lipgloss.Color("#fef2f2"),
			lipgloss.Color("#fee2e2"),
			lipgloss.Color("#fecaca"),
			lipgloss.Color("#fca5a5"),
			lipgloss.Color("#f87171"),
			lipgloss.Color("#ef4444"),
			lipgloss.Color("#dc2626"),
			lipgloss.Color("#b91c1c"),
			lipgloss.Color("#991b1b"),
			lipgloss.Color("#7f1d1d"),
		),
		Yellow: NewPaletteShades(
			lipgloss.Color("#fefce8"),
			lipgloss.Color("#fef3c7"),
			lipgloss.Color("#fde68a"),
			lipgloss.Color("#fcd34d"),
			lipgloss.Color("#fbbf24"),
			lipgloss.Color("#eab308"),
			lipgloss.Color("#ca8a04"),
			lipgloss.Color("#a16207"),
			lipgloss.Color("#854d0e"),
			lipgloss.Color("#713f12"),
		),
		Purple: NewPaletteShades(
			lipgloss.Color("#faf5ff"),
			lipgloss.Color("#f3e8ff"),
			lipgloss.Color("#e9d5ff"),
			lipgloss.Color("#d8b4fe"),
			lipgloss.Color("#c084fc"),
			lipgloss.Color("#a855f7"),
			lipgloss.Color("#9333ea"),
			lipgloss.Color("#7c3aed"),
			lipgloss.Color("#6b21a8"),
			lipgloss.Color("#581c87"),
		),
		Cyan: NewPaletteShades(
			lipgloss.Color("#ecfeff"),
			lipgloss.Color("#cffafe"),
			lipgloss.Color("#a5f3fc"),
			lipgloss.Color("#67e8f9"),
			lipgloss.Color("#22d3ee"),
			lipgloss.Color("#06b6d4"),
			lipgloss.Color("#0891b2"),
			lipgloss.Color("#0e7490"),
			lipgloss.Color("#155e75"),
			lipgloss.Color("#164e63"),
		),
	}

	borders := BorderSet{
		None:    lipgloss.Border{},
		Normal:  lipgloss.NormalBorder(),
		Rounded: lipgloss.RoundedBorder(),
		Thick:   lipgloss.ThickBorder(),
		Double:  lipgloss.DoubleBorder(),
	}

	typography := defaultTypography(palette)

	spacing := SpacingConfig{
		Padding: defaultSpacingTable(),
		Margin:  defaultSpacingTable(),
	}

	input := InputStyles{
		Default: lipgloss.NewStyle().
			BorderStyle(borders.Rounded).
			BorderForeground(palette.Neutral.Muted).
			Padding(0, 1).
			Background(palette.Surface.Base).
			Foreground(palette.Surface.OnBase),
		Focus: lipgloss.NewStyle().
			BorderStyle(borders.Thick).
			BorderForeground(palette.Primary.Base).
			Padding(0, 1).
			Background(palette.Surface.Base).
			Foreground(palette.Surface.OnBase),
	}

	// Initialize variant registry with component styling strategies
	variants := NewVariantRegistry()
	registerButtonVariants(variants)
	registerBadgeVariants(variants)
	registerAlertVariants(variants)

	theme := Theme{
		Palette:    palette,
		Colors:     colorFamilies,
		Borders:    borders,
		Spacing:    spacing,
		Typography: typography,
		Input:      input,
		Variants:   variants,
	}

	return theme.Normalize()
}

// registerButtonVariants populates button variant strategies
func registerButtonVariants(registry *VariantRegistry) {
	registry.Register(ButtonVariantPrimary, NewCompositeStrategy(
		Background(PalettePrimary),
		PaddingX(SpacingSizeMedium),
		PaddingY(SpacingSizeExtraSmall),
	))
	registry.Register(ButtonVariantSecondary, NewCompositeStrategy(
		Background(PaletteSecondary),
		PaddingX(SpacingSizeMedium),
		PaddingY(SpacingSizeExtraSmall),
	))
	registry.Register(ButtonVariantSuccess, NewCompositeStrategy(
		Background(PaletteSuccess),
		PaddingX(SpacingSizeMedium),
		PaddingY(SpacingSizeExtraSmall),
	))
	registry.Register(ButtonVariantError, NewCompositeStrategy(
		Background(PaletteDanger),
		PaddingX(SpacingSizeMedium),
		PaddingY(SpacingSizeExtraSmall),
	))
	registry.Register(ButtonVariantWarning, NewCompositeStrategy(
		Background(PaletteWarning),
		PaddingX(SpacingSizeMedium),
		PaddingY(SpacingSizeExtraSmall),
	))
	registry.Register(ButtonVariantInfo, NewCompositeStrategy(
		Background(PaletteInfo),
		PaddingX(SpacingSizeMedium),
		PaddingY(SpacingSizeExtraSmall),
	))
	registry.Register(ButtonVariantMuted, NewCompositeStrategy(
		Background(PaletteNeutral),
		PaddingX(SpacingSizeMedium),
		PaddingY(SpacingSizeExtraSmall),
	))
}

// registerBadgeVariants populates badge variant strategies
func registerBadgeVariants(registry *VariantRegistry) {
	registry.Register(BadgeVariantPrimary, NewCompositeStrategy(
		Background(PalettePrimary),
		PaddingX(SpacingSizeSmall),
		Border(BorderVariantRounded),
	))
	registry.Register(BadgeVariantSecondary, NewCompositeStrategy(
		Background(PaletteSecondary),
		PaddingX(SpacingSizeSmall),
		Border(BorderVariantRounded),
	))
	registry.Register(BadgeVariantSuccess, NewCompositeStrategy(
		Background(PaletteSuccess),
		PaddingX(SpacingSizeSmall),
		Border(BorderVariantRounded),
	))
	registry.Register(BadgeVariantWarning, NewCompositeStrategy(
		Background(PaletteWarning),
		PaddingX(SpacingSizeSmall),
		Border(BorderVariantRounded),
	))
	registry.Register(BadgeVariantError, NewCompositeStrategy(
		Background(PaletteDanger),
		PaddingX(SpacingSizeSmall),
		Border(BorderVariantRounded),
	))
	registry.Register(BadgeVariantInfo, NewCompositeStrategy(
		Background(PaletteInfo),
		PaddingX(SpacingSizeSmall),
		Border(BorderVariantRounded),
	))
	registry.Register(BadgeVariantDefault, NewCompositeStrategy(
		Background(PaletteNeutral),
		PaddingX(SpacingSizeSmall),
		Border(BorderVariantRounded),
	))
}

// registerAlertVariants populates alert variant strategies
func registerAlertVariants(registry *VariantRegistry) {
	registry.Register(AlertVariantSuccess, NewCompositeStrategy(
		Background(PaletteSuccess),
	))
	registry.Register(AlertVariantWarning, NewCompositeStrategy(
		Background(PaletteWarning),
	))
	registry.Register(AlertVariantError, NewCompositeStrategy(
		Background(PaletteDanger),
	))
	registry.Register(AlertVariantInfo, NewCompositeStrategy(
		Background(PaletteInfo),
	))
}

func defaultTypography(p Palette) TypographyScale {
	base := lipgloss.NewStyle().Foreground(p.Surface.OnBase)

	title := base.
		Bold(true).
		Foreground(p.Primary.Base)

	subtitle := base.
		Foreground(p.Secondary.Muted).
		Faint(true)

	body := base

	code := base.
		Foreground(p.Secondary.Base).
		Background(p.Surface.Muted).
		Padding(0, 1)

	emphasis := base.
		Bold(true)

	return TypographyScale{
		Base:       body,
		Title:      title,
		Subtitle:   subtitle,
		Body:       body,
		Code:       code,
		Emphasis:   emphasis,
		TextXs:     body.Faint(true),
		TextSm:     body,
		TextBase:   body,
		TextLg:     body.Bold(true),
		TextXl:     body.Bold(true).Underline(true),
		Text2Xl:    body.Bold(true).Underline(true).MarginTop(1),
		Text3Xl:    body.Bold(true).Underline(true).MarginTop(1).MarginBottom(1),
		FontLight:  body.Faint(true),
		FontNormal: body,
		FontMedium: body.Bold(true),
		FontSemibold: body.
			Bold(true).
			Underline(true),
		FontBold: body.
			Bold(true).
			Italic(true),
	}
}

// DarkTheme returns a dark theme variant
func DarkTheme() Theme {
	theme := DefaultTheme()

	theme.Palette.Surface = ColourSet{
		Base:     lipgloss.AdaptiveColor{Light: "#111827", Dark: "#0b1120"},
		OnBase:   lipgloss.AdaptiveColor{Light: "#f9fafb", Dark: "#e5e7eb"},
		Muted:    lipgloss.AdaptiveColor{Light: "#1f2937", Dark: "#111827"},
		Contrast: lipgloss.AdaptiveColor{Light: "#3b82f6", Dark: "#60a5fa"},
	}

	theme.Palette.Neutral = ColourSet{
		Base:     lipgloss.AdaptiveColor{Light: "#475569", Dark: "#334155"},
		OnBase:   lipgloss.AdaptiveColor{Light: "#e5e7eb", Dark: "#cbd5f5"},
		Muted:    lipgloss.AdaptiveColor{Light: "#374151", Dark: "#1f2937"},
		Contrast: lipgloss.AdaptiveColor{Light: "#f8fafc", Dark: "#f8fafc"},
	}

	theme.Typography = defaultTypography(theme.Palette)

	// Re-register variants with updated palette
	theme.Variants = NewVariantRegistry()
	registerButtonVariants(theme.Variants)
	registerBadgeVariants(theme.Variants)
	registerAlertVariants(theme.Variants)

	return theme.Normalize()
}

// LightTheme returns a light theme variant
func LightTheme() Theme {
	return DefaultTheme()
}

// Helper functions to access theme properties using typed variants

// PaletteColor returns the color for a given palette family and shade.
// Returns an empty string and false if the shade is invalid.
func PaletteColor(theme Theme, family PaletteFamily, shade PaletteShade) (lipgloss.Color, bool) {
	shades := theme.Colors.Shades(family)
	color := shades.Color(shade)
	if color == "" {
		return "", false
	}
	return color, true
}

// BorderForVariant returns the border style for the given variant.
func BorderForVariant(theme Theme, variant BorderVariant) lipgloss.Border {
	switch variant {
	case BorderVariantNormal:
		return theme.Borders.Normal
	case BorderVariantThick:
		return theme.Borders.Thick
	case BorderVariantDouble:
		return theme.Borders.Double
	case BorderVariantRounded:
		return theme.Borders.Rounded
	default:
		return theme.Borders.None
	}
}

// PaddingValue returns the padding value for the given size.
func PaddingValue(theme Theme, size SpacingSize) int {
	return spacingLookup(theme.Spacing.Padding, size)
}

// MarginValue returns the margin value for the given size.
func MarginValue(theme Theme, size SpacingSize) int {
	return spacingLookup(theme.Spacing.Margin, size)
}

func spacingLookup(table spacingTable, size SpacingSize) int {
	index := int(size)
	if index < 0 || index >= len(table) {
		index = int(SpacingSizeMedium)
	}
	return table[index]
}

// TypographyStyle returns the specified typography style from the given theme.
func TypographyStyle(theme Theme, variant TypographyVariant) lipgloss.Style {
	typo := theme.Typography
	switch variant {
	case TypographyVariantTitle:
		return typo.Title
	case TypographyVariantSubtitle:
		return typo.Subtitle
	case TypographyVariantBody:
		return typo.Body
	case TypographyVariantCode:
		return typo.Code
	case TypographyVariantEmphasis:
		return typo.Emphasis
	case TypographyVariantTextXs:
		return typo.TextXs
	case TypographyVariantTextSm:
		return typo.TextSm
	case TypographyVariantTextBase:
		return typo.TextBase
	case TypographyVariantTextLg:
		return typo.TextLg
	case TypographyVariantTextXl:
		return typo.TextXl
	case TypographyVariantText2Xl:
		return typo.Text2Xl
	case TypographyVariantText3Xl:
		return typo.Text3Xl
	case TypographyVariantFontLight:
		return typo.FontLight
	case TypographyVariantFontNormal:
		return typo.FontNormal
	case TypographyVariantFontMedium:
		return typo.FontMedium
	case TypographyVariantFontSemibold:
		return typo.FontSemibold
	case TypographyVariantFontBold:
		return typo.FontBold
	default:
		return typo.Base
	}
}

// InputStyle returns the input style for the given state.
func InputStyle(theme Theme, state InputState) lipgloss.Style {
	input := theme.Input
	if state == InputStateFocus {
		return input.Focus
	}
	return input.Default
}

// ColourSet represents a semantic color set with base, on-base, muted, and contrast colors.
// This provides complete color combinations that work well together:
//
//   - Base: The primary background or brand color
//   - OnBase: Text/content color that contrasts well with Base
//   - Muted: A desaturated variant of Base for subtle accents
//   - Contrast: An accent color that "pops" against Base
//
// All colors are adaptive, providing both light and dark mode variants.
type ColourSet struct {
	Base     lipgloss.AdaptiveColor
	OnBase   lipgloss.AdaptiveColor
	Muted    lipgloss.AdaptiveColor
	Contrast lipgloss.AdaptiveColor
}

// PaletteSlot provides access to a semantic colour slot from a Palette.
// Use the predefined slots (PalettePrimary, PaletteSuccess, etc.) for type-safe access.
type PaletteSlot func(Palette) ColourSet

// Predefined semantic palette slots for type-safe theme access.
// Use these with style modifiers: Background(PalettePrimary), Foreground(PaletteSuccess), etc.
var (
	PalettePrimary   PaletteSlot = func(p Palette) ColourSet { return p.Primary }
	PaletteSecondary PaletteSlot = func(p Palette) ColourSet { return p.Secondary }
	PaletteSurface   PaletteSlot = func(p Palette) ColourSet { return p.Surface }
	PaletteSuccess   PaletteSlot = func(p Palette) ColourSet { return p.Success }
	PaletteWarning   PaletteSlot = func(p Palette) ColourSet { return p.Warning }
	PaletteDanger    PaletteSlot = func(p Palette) ColourSet { return p.Danger }
	PaletteInfo      PaletteSlot = func(p Palette) ColourSet { return p.Info }
	PaletteNeutral   PaletteSlot = func(p Palette) ColourSet { return p.Neutral }
)

// Fluent modifier functions

// Background applies a semantic background colour and matching foreground for optimal contrast.
// This is the recommended way to apply colors, as it automatically handles text legibility.
//
// Example:
//
//	card := NewCard().WithAppliers(Background(PalettePrimary))
func Background(slot PaletteSlot) StyleFunc {
	return func(base lipgloss.Style, theme Theme) lipgloss.Style {
		cs := slot(theme.Palette)
		return base.Background(cs.Base).Foreground(cs.OnBase)
	}
}

// Foreground applies a semantic foreground colour without changing the background.
// Use this for text color changes when the background should remain unchanged.
//
// Example:
//
//	text := NewText("Error").WithAppliers(Foreground(PaletteDanger))
func Foreground(slot PaletteSlot) StyleFunc {
	return func(base lipgloss.Style, theme Theme) lipgloss.Style {
		cs := slot(theme.Palette)
		return base.Foreground(cs.Base)
	}
}

// Border applies a border style from the theme.
func Border(variant BorderVariant) StyleFunc {
	return func(base lipgloss.Style, theme Theme) lipgloss.Style {
		return base.Border(BorderForVariant(theme, variant))
	}
}

func Padding(size SpacingSize) StyleFunc {
	return func(base lipgloss.Style, theme Theme) lipgloss.Style {
		value := spacingLookup(theme.Spacing.Padding, size)
		return base.Padding(value)
	}
}

func PaddingX(size SpacingSize) StyleFunc {
	return func(base lipgloss.Style, theme Theme) lipgloss.Style {
		value := spacingLookup(theme.Spacing.Padding, size)
		return base.PaddingLeft(value).PaddingRight(value)
	}
}

func PaddingY(size SpacingSize) StyleFunc {
	return func(base lipgloss.Style, theme Theme) lipgloss.Style {
		value := spacingLookup(theme.Spacing.Padding, size)
		return base.PaddingTop(value).PaddingBottom(value)
	}
}

func Margin(size SpacingSize) StyleFunc {
	return func(base lipgloss.Style, theme Theme) lipgloss.Style {
		value := spacingLookup(theme.Spacing.Margin, size)
		return base.Margin(value)
	}
}

func MarginX(size SpacingSize) StyleFunc {
	return func(base lipgloss.Style, theme Theme) lipgloss.Style {
		value := spacingLookup(theme.Spacing.Margin, size)
		return base.MarginLeft(value).MarginRight(value)
	}
}

func MarginY(size SpacingSize) StyleFunc {
	return func(base lipgloss.Style, theme Theme) lipgloss.Style {
		value := spacingLookup(theme.Spacing.Margin, size)
		return base.MarginTop(value).MarginBottom(value)
	}
}

// Typography applies typography styling
func Typography(variant TypographyVariant) StyleFunc {
	return func(base lipgloss.Style, theme Theme) lipgloss.Style {
		return base.Inherit(TypographyStyle(theme, variant))
	}
}

// Predefined style bundles for common component patterns

func CardBaseStyle() []StyleFunc {
	return []StyleFunc{
		Background(PaletteSurface),
		Border(BorderVariantRounded),
		Margin(SpacingSizeSmall),
		Padding(SpacingSizeMedium),
	}
}

func ButtonPrimaryStyle() []StyleFunc {
	return []StyleFunc{
		Background(PalettePrimary),
		Border(BorderVariantRounded),
		PaddingX(SpacingSizeMedium),
		PaddingY(SpacingSizeSmall),
		Typography(TypographyVariantEmphasis),
	}
}

func ButtonSecondaryStyle() []StyleFunc {
	return []StyleFunc{
		Background(PaletteSecondary),
		Border(BorderVariantRounded),
		PaddingX(SpacingSizeMedium),
		PaddingY(SpacingSizeSmall),
		Typography(TypographyVariantEmphasis),
	}
}

func AlertSuccessStyle() []StyleFunc {
	return []StyleFunc{
		Background(PaletteSuccess),
		Border(BorderVariantNormal),
		Padding(SpacingSizeSmall),
	}
}

func AlertErrorStyle() []StyleFunc {
	return []StyleFunc{
		Background(PaletteDanger),
		Border(BorderVariantNormal),
		Padding(SpacingSizeSmall),
	}
}

func AlertWarningStyle() []StyleFunc {
	return []StyleFunc{
		Background(PaletteWarning),
		Border(BorderVariantNormal),
		Padding(SpacingSizeSmall),
	}
}

func AlertInfoStyle() []StyleFunc {
	return []StyleFunc{
		Background(PaletteInfo),
		Border(BorderVariantNormal),
		Padding(SpacingSizeSmall),
	}
}

// Utility functions for typed styling

// BackgroundPalette creates a style with a background color from a specific palette shade.
func BackgroundPalette(theme Theme, family PaletteFamily, shade PaletteShade) lipgloss.Style {
	if color, ok := PaletteColor(theme, family, shade); ok {
		return lipgloss.NewStyle().Background(color)
	}
	return lipgloss.NewStyle()
}

// TextPalette creates a style with a foreground color from a specific palette shade.
func TextPalette(theme Theme, family PaletteFamily, shade PaletteShade) lipgloss.Style {
	if color, ok := PaletteColor(theme, family, shade); ok {
		return lipgloss.NewStyle().Foreground(color)
	}
	return lipgloss.NewStyle()
}
