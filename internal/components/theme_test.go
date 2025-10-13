package components

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestDefaultTheme(t *testing.T) {
	theme := DefaultTheme()

	assert.Equal(t, "#3b82f6", theme.Palette.Primary.Base.Light)
	assert.Equal(t, "#111827", theme.Palette.Surface.OnBase.Light)

	assert.Equal(t, lipgloss.RoundedBorder(), theme.Borders.Rounded)
	assert.Equal(t, lipgloss.ThickBorder(), theme.Borders.Thick)

	assert.Equal(t, 4, theme.Spacing.Padding[SpacingSizeMedium])
	assert.Equal(t, 3, theme.Spacing.Margin[SpacingSizeSmall])

	assert.True(t, theme.Typography.Title.GetBold(), "title typography should be bold")
	assert.NotEqual(t, lipgloss.Style{}, theme.Input.Default, "input default style should be set")
}

func TestDarkTheme(t *testing.T) {
	light := DefaultTheme()
	dark := DarkTheme()

	assert.NotEqual(t, light.Palette.Surface.Base.Light, dark.Palette.Surface.Base.Light, "dark theme should invert surface base")
	assert.NotEqual(t, light.Typography.Base.GetForeground(), dark.Typography.Base.GetForeground(), "dark theme should adjust typography foreground")
}

func TestSetGetTheme(t *testing.T) {
	original := GetTheme()

	custom := DefaultTheme()
	custom.Palette.Primary.Base = lipgloss.AdaptiveColor{Light: "#0000ff", Dark: "#1e3a8a"}
	SetTheme(custom)

	active := GetTheme()
	assert.Equal(t, "#0000ff", active.Palette.Primary.Base.Light)

	SetTheme(original)
}

func TestPaletteColor(t *testing.T) {
	color, ok := PaletteColor(PaletteBlue, PaletteShade500)
	assert.True(t, ok)
	assert.Equal(t, lipgloss.Color("#3b82f6"), color)

	_, ok = PaletteColor(PaletteBlue, PaletteShade(99))
	assert.False(t, ok, "out-of-range shades should report missing")
}

func TestBorderStyle(t *testing.T) {
	assert.Equal(t, lipgloss.NormalBorder(), BorderStyle(BorderVariantNormal))
	assert.Equal(t, lipgloss.DoubleBorder(), BorderStyle(BorderVariantDouble))
}

func TestSpacingHelpers(t *testing.T) {
	SetTheme(DefaultTheme())
	assert.Equal(t, 4, PaddingValue(SpacingSizeMedium))
	assert.Equal(t, 3, MarginValue(SpacingSizeSmall))
}

func TestTypographyStyle(t *testing.T) {
	emphasis := TypographyStyle(TypographyVariantEmphasis)
	assert.True(t, emphasis.GetBold(), "emphasis typography should be bold")
}

func TestInputStyle(t *testing.T) {
	focus := InputStyle(InputStateFocus).Render("input")
	normal := InputStyle(InputStateDefault).Render("input")
	assert.NotEqual(t, normal, focus, "focus rendering should differ from default")
}
