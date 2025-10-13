package components

import (
	"sync"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestStyleApplier(t *testing.T) {
	style := Style(
		lipgloss.NewStyle(),
		Background(PalettePrimary),
		Padding(SpacingSizeMedium),
		Border(BorderVariantRounded),
	)

	assert.NotEmpty(t, style.GetBackground(), "expected background to be set")
	assert.True(t, style.GetPaddingLeft() > 0, "expected padding to be applied")
}

func TestPaletteSlots(t *testing.T) {
	palette := GetTheme().Palette
	assert.NotEmpty(t, palette.Primary.Base.Light, "primary light tone should be set")
	assert.NotEmpty(t, palette.Secondary.Base.Dark, "secondary dark tone should be set")
}

func TestPredefinedBundles(t *testing.T) {
	cardStyle := Style(lipgloss.NewStyle(), CardBaseStyle()...)
	assert.NotEmpty(t, cardStyle.GetBackground(), "card style should set background")

	buttonStyle := Style(lipgloss.NewStyle(), ButtonPrimaryStyle()...)
	assert.NotEmpty(t, buttonStyle.GetBackground(), "button style should set background")

	alertStyle := Style(lipgloss.NewStyle(), AlertSuccessStyle()...)
	assert.NotEmpty(t, alertStyle.GetBackground(), "alert style should set background")
}

func TestSpacingValues(t *testing.T) {
	assert.Equal(t, 4, PaddingValue(SpacingSizeMedium), "padding value should match spacing table")
	assert.Equal(t, 3, MarginValue(SpacingSizeSmall), "margin value should match spacing table")
}

func TestTypographyModifier(t *testing.T) {
	title := TypographyStyle(TypographyVariantTitle)
	assert.True(t, title.GetBold(), "title typography should be bold")
}

func TestButtonStates(t *testing.T) {
	button := SimpleButton("Save")
	normal := button.View()

	disabled := button.WithDisabled(true).View()
	assert.NotEqual(t, normal, disabled, "disabled state should render differently")

	focused := button.WithDisabled(false).WithFocus(true).View()
	assert.NotEqual(t, normal, focused, "focused state should render differently")
}

func TestAlertVariants(t *testing.T) {
	success := SuccessAlert("operation completed").View()
	errorView := ErrorAlert("operation failed").View()
	assert.NotEqual(t, success, errorView, "different alert helpers should render unique output")
}

func TestThemeSwitch(t *testing.T) {
	original := GetTheme()

	SetTheme(DarkTheme())
	dark := GetTheme()
	assert.NotEqual(t, original.Palette.Surface.Base.Light, dark.Palette.Surface.Base.Light, "dark theme should change surface colour")

	SetTheme(DefaultTheme())
	restored := GetTheme()
	assert.Equal(t, original.Palette.Surface.Base.Light, restored.Palette.Surface.Base.Light, "theme should restore original surface colour")
}

func TestConcurrentThemeAccess(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			palette := GetTheme().Palette
			assert.NotEmpty(t, palette.Primary.Base.Light)
		}()
	}
	wg.Wait()
}

func TestFluentModifierChain(t *testing.T) {
	style := Style(
		lipgloss.NewStyle(),
		Background(PaletteSuccess),
		Border(BorderVariantRounded),
		PaddingX(SpacingSizeLarge),
		PaddingY(SpacingSizeSmall),
		Typography(TypographyVariantEmphasis),
	)

	assert.NotEmpty(t, style.GetBackground(), "chained modifiers should set background")
	assert.True(t, style.GetPaddingLeft() > 0, "chained modifiers should set padding")
}
