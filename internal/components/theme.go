package components

import (
	"sync"

	"github.com/charmbracelet/lipgloss"
)

const paletteShadeCount = 10

type PaletteShades struct {
	colors [paletteShadeCount]lipgloss.Color
}

func NewPaletteShades(colors ...lipgloss.Color) PaletteShades {
	var shades PaletteShades
	for i := 0; i < paletteShadeCount && i < len(colors); i++ {
		shades.colors[i] = colors[i]
	}
	return shades
}

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

// Theme represents the global styling theme for components
type Theme struct {
	Palette    Palette
	Colors     ColorPalette
	Borders    BorderSet
	Spacing    SpacingConfig
	Typography TypographyScale
	Input      InputStyles
}

// ThemeManager coordinates access to a Theme instance.
type ThemeManager struct {
	mu    sync.RWMutex
	theme Theme
}

// NewThemeManager allocates a ThemeManager with the provided theme.
func NewThemeManager(theme Theme) *ThemeManager {
	return &ThemeManager{theme: cloneTheme(normalizeTheme(theme))}
}

// SetTheme replaces the managed theme.
func (m *ThemeManager) SetTheme(theme Theme) {
	m.mu.Lock()
	m.theme = cloneTheme(normalizeTheme(theme))
	m.mu.Unlock()
}

// Theme returns a copy of the managed theme.
func (m *ThemeManager) Theme() Theme {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return cloneTheme(m.theme)
}

func normalizeTheme(theme Theme) Theme {
	theme.Spacing = normalizeSpacingConfig(theme.Spacing)
	return theme
}

func cloneTheme(theme Theme) Theme {
	theme.Spacing = cloneSpacingConfig(theme.Spacing)
	return theme
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

func cloneSpacingConfig(cfg SpacingConfig) SpacingConfig {
	return SpacingConfig{
		Margin:  cfg.Margin,
		Padding: cfg.Padding,
	}
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

	theme := Theme{
		Palette:    palette,
		Colors:     colorFamilies,
		Borders:    borders,
		Spacing:    spacing,
		Typography: typography,
		Input:      input,
	}

	return normalizeTheme(theme)
}

func defaultTypography(p Palette) TypographyScale {
	base := lipgloss.NewStyle().Foreground(p.Surface.OnBase)

	title := base.Copy().
		Bold(true).
		Foreground(p.Primary.Base)

	subtitle := base.Copy().
		Foreground(p.Secondary.Muted).
		Faint(true)

	body := base.Copy()

	code := base.Copy().
		Foreground(p.Secondary.Base).
		Background(p.Surface.Muted).
		Padding(0, 1)

	emphasis := base.Copy().
		Bold(true)

	return TypographyScale{
		Base:       body,
		Title:      title,
		Subtitle:   subtitle,
		Body:       body,
		Code:       code,
		Emphasis:   emphasis,
		TextXs:     body.Copy().Faint(true),
		TextSm:     body.Copy(),
		TextBase:   body.Copy(),
		TextLg:     body.Copy().Bold(true),
		TextXl:     body.Copy().Bold(true).Underline(true),
		Text2Xl:    body.Copy().Bold(true).Underline(true).MarginTop(1),
		Text3Xl:    body.Copy().Bold(true).Underline(true).MarginTop(1).MarginBottom(1),
		FontLight:  body.Copy().Faint(true),
		FontNormal: body.Copy(),
		FontMedium: body.Copy().Bold(true),
		FontSemibold: body.Copy().
			Bold(true).
			Underline(true),
		FontBold: body.Copy().
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
	return normalizeTheme(theme)
}

// LightTheme returns a light theme variant
func LightTheme() Theme {
	return DefaultTheme()
}

// Theme variables for easy access
var defaultThemeManager = NewThemeManager(DefaultTheme())

// SetTheme sets the global theme
func SetTheme(theme Theme) {
	defaultThemeManager.SetTheme(theme)
}

// GetTheme returns the current global theme
func GetTheme() Theme {
	return defaultThemeManager.Theme()
}

// Helper functions to access theme properties using typed variants

func PaletteColor(family PaletteFamily, shade PaletteShade) (lipgloss.Color, bool) {
	shades := GetTheme().Colors.Shades(family)
	color := shades.Color(shade)
	if color == "" {
		return "", false
	}
	return color, true
}

func BorderStyle(variant BorderVariant) lipgloss.Border {
	theme := GetTheme()
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

func PaddingValue(size SpacingSize) int {
	return spacingLookup(GetTheme().Spacing.Padding, size)
}

func MarginValue(size SpacingSize) int {
	return spacingLookup(GetTheme().Spacing.Margin, size)
}

func spacingLookup(table spacingTable, size SpacingSize) int {
	index := int(size)
	if index < 0 || index >= len(table) {
		index = int(SpacingSizeMedium)
	}
	return table[index]
}

// TypographyStyle returns the specified typography style from the current theme.
func TypographyStyle(variant TypographyVariant) lipgloss.Style {
	typo := GetTheme().Typography
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

func InputStyle(state InputState) lipgloss.Style {
	input := GetTheme().Input
	if state == InputStateFocus {
		return input.Focus
	}
	return input.Default
}

// StyleApplier represents a function that can apply styling to a lipgloss.Style
type StyleApplier interface {
	Apply(base lipgloss.Style, theme Theme) lipgloss.Style
}

// StyleFunc implements StyleApplier for a function type
type StyleFunc func(lipgloss.Style, Theme) lipgloss.Style

func (fn StyleFunc) Apply(base lipgloss.Style, theme Theme) lipgloss.Style {
	return fn(base, theme)
}

// Style applies a series of modifiers to create a final style
func Style(base lipgloss.Style, appliers ...StyleApplier) lipgloss.Style {
	theme := GetTheme()
	for _, applier := range appliers {
		base = applier.Apply(base, theme)
	}
	return base
}

func cloneAppliers(base []StyleApplier, extras ...StyleApplier) []StyleApplier {
	cloned := make([]StyleApplier, len(base)+len(extras))
	copy(cloned, base)
	copy(cloned[len(base):], extras)
	return cloned
}

// ColourSet represents a semantic color set with base, on-base, muted, and contrast colors.
type ColourSet struct {
	Base     lipgloss.AdaptiveColor
	OnBase   lipgloss.AdaptiveColor
	Muted    lipgloss.AdaptiveColor
	Contrast lipgloss.AdaptiveColor
}

// PaletteSlot provides access to a semantic colour slot.
type PaletteSlot func(Palette) ColourSet

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

// Background applies a semantic background colour and matching foreground.
func Background(slot PaletteSlot) StyleFunc {
	return func(base lipgloss.Style, theme Theme) lipgloss.Style {
		cs := slot(theme.Palette)
		return base.Background(cs.Base).Foreground(cs.OnBase)
	}
}

// Foreground applies a semantic foreground colour.
func Foreground(slot PaletteSlot) StyleFunc {
	return func(base lipgloss.Style, theme Theme) lipgloss.Style {
		cs := slot(theme.Palette)
		return base.Foreground(cs.Base)
	}
}

func Border(variant BorderVariant) StyleFunc {
	return func(base lipgloss.Style, theme Theme) lipgloss.Style {
		return base.Border(borderForVariant(theme, variant))
	}
}

func borderForVariant(theme Theme, variant BorderVariant) lipgloss.Border {
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
		return base.Inherit(TypographyStyle(variant))
	}
}

// Predefined style bundles for common component patterns

func CardBaseStyle() []StyleApplier {
	return []StyleApplier{
		Background(PaletteSurface),
		Border(BorderVariantRounded),
		Margin(SpacingSizeSmall),
		Padding(SpacingSizeMedium),
	}
}

func ButtonPrimaryStyle() []StyleApplier {
	return []StyleApplier{
		Background(PalettePrimary),
		Border(BorderVariantRounded),
		PaddingX(SpacingSizeMedium),
		PaddingY(SpacingSizeSmall),
		Typography(TypographyVariantEmphasis),
	}
}

func ButtonSecondaryStyle() []StyleApplier {
	return []StyleApplier{
		Background(PaletteSecondary),
		Border(BorderVariantRounded),
		PaddingX(SpacingSizeMedium),
		PaddingY(SpacingSizeSmall),
		Typography(TypographyVariantEmphasis),
	}
}

func AlertSuccessStyle() []StyleApplier {
	return []StyleApplier{
		Background(PaletteSuccess),
		Border(BorderVariantNormal),
		Padding(SpacingSizeSmall),
	}
}

func AlertErrorStyle() []StyleApplier {
	return []StyleApplier{
		Background(PaletteDanger),
		Border(BorderVariantNormal),
		Padding(SpacingSizeSmall),
	}
}

func AlertWarningStyle() []StyleApplier {
	return []StyleApplier{
		Background(PaletteWarning),
		Border(BorderVariantNormal),
		Padding(SpacingSizeSmall),
	}
}

func AlertInfoStyle() []StyleApplier {
	return []StyleApplier{
		Background(PaletteInfo),
		Border(BorderVariantNormal),
		Padding(SpacingSizeSmall),
	}
}

// Utility functions for typed styling

func BackgroundPalette(family PaletteFamily, shade PaletteShade) lipgloss.Style {
	if color, ok := PaletteColor(family, shade); ok {
		return lipgloss.NewStyle().Background(color)
	}
	return lipgloss.NewStyle()
}

func TextPalette(family PaletteFamily, shade PaletteShade) lipgloss.Style {
	if color, ok := PaletteColor(family, shade); ok {
		return lipgloss.NewStyle().Foreground(color)
	}
	return lipgloss.NewStyle()
}
