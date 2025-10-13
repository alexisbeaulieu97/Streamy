package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ButtonSize represents different button sizes
type ButtonSize int

const (
	ButtonSizeSmall ButtonSize = iota
	ButtonSizeMedium
	ButtonSizeLarge
)

// ButtonOptions defines the configuration options for a button
type ButtonOptions struct {
	Variant  ButtonVariant
	Size     ButtonSize
	Disabled bool
	Focus    bool
}

// Button represents a clickable button component
type Button struct {
	label   string
	options ButtonOptions
}

// NewButton creates a new button with the given label and options
func NewButton(label string, opts ButtonOptions) *Button {
	return &Button{
		label:   label,
		options: opts,
	}
}

// WithVariant sets the button variant
func (b *Button) WithVariant(variant ButtonVariant) *Button {
	b.options.Variant = variant
	return b
}

// WithSize sets the button size
func (b *Button) WithSize(size ButtonSize) *Button {
	b.options.Size = size
	return b
}

// WithDisabled sets the button disabled state
func (b *Button) WithDisabled(disabled bool) *Button {
	b.options.Disabled = disabled
	return b
}

// WithFocus sets the button focus state
func (b *Button) WithFocus(focus bool) *Button {
	b.options.Focus = focus
	return b
}

// View renders the button
func (b *Button) View() string {
	style := b.buildStyle()
	return style.Render(b.label)
}

// buildStyle calculates the button style based on current options
func (b *Button) buildStyle() lipgloss.Style {
	appliers := append(buttonVariantAppliers(b.options.Variant), buttonSizeAppliers(b.options.Size)...)
	style := Style(lipgloss.NewStyle(), appliers...)

	if b.options.Disabled {
		style = Style(style, Border(BorderVariantNormal), Foreground(PaletteNeutral))
		style = style.Faint(true)
		style = style.BorderForeground(GetTheme().Palette.Neutral.Muted)
	} else if b.options.Focus {
		style = Style(style, Border(BorderVariantThick))
		style = style.BorderForeground(GetTheme().Palette.Primary.Base)
	}

	return style
}

func buttonVariantAppliers(variant ButtonVariant) []StyleApplier {
	switch variant {
	case ButtonVariantPrimary:
		return ButtonPrimaryStyle()
	case ButtonVariantSecondary:
		return ButtonSecondaryStyle()
	case ButtonVariantSuccess:
		return cloneAppliers(AlertSuccessStyle(), Typography(TypographyVariantEmphasis))
	case ButtonVariantError:
		return cloneAppliers(AlertErrorStyle(), Typography(TypographyVariantEmphasis))
	case ButtonVariantWarning:
		return cloneAppliers(AlertWarningStyle(), Typography(TypographyVariantEmphasis))
	case ButtonVariantInfo:
		return cloneAppliers(AlertInfoStyle(), Typography(TypographyVariantEmphasis))
	case ButtonVariantMuted:
		return []StyleApplier{
			Background(PaletteNeutral),
			Border(BorderVariantRounded),
			PaddingX(SpacingSizeMedium),
			PaddingY(SpacingSizeSmall),
			Foreground(PaletteSurface),
			Typography(TypographyVariantEmphasis),
		}
	default:
		return ButtonPrimaryStyle()
	}
}

func buttonSizeAppliers(size ButtonSize) []StyleApplier {
	switch size {
	case ButtonSizeSmall:
		return []StyleApplier{
			PaddingX(SpacingSizeSmall),
			PaddingY(SpacingSizeExtraSmall),
		}
	case ButtonSizeLarge:
		return []StyleApplier{
			PaddingX(SpacingSizeLarge),
			PaddingY(SpacingSizeMedium),
		}
	default:
		return []StyleApplier{
			PaddingX(SpacingSizeMedium),
			PaddingY(SpacingSizeSmall),
		}
	}
}

// SimpleButton creates a button with sensible defaults
func SimpleButton(label string) *Button {
	return NewButton(label, ButtonOptions{
		Variant:  ButtonVariantPrimary,
		Size:     ButtonSizeMedium,
		Disabled: false,
		Focus:    false,
	})
}

// ButtonGroup represents a horizontal group of buttons
type ButtonGroup struct {
	buttons []*Button
	spacing int
}

// NewButtonGroup creates a new button group
func NewButtonGroup(buttons ...*Button) *ButtonGroup {
	return &ButtonGroup{
		buttons: buttons,
		spacing: MarginValue(SpacingSizeSmall),
	}
}

// WithSpacing sets the spacing between buttons
func (bg *ButtonGroup) WithSpacing(spacing int) *ButtonGroup {
	bg.spacing = spacing
	return bg
}

// AddButton adds a button to the group
func (bg *ButtonGroup) AddButton(button *Button) *ButtonGroup {
	bg.buttons = append(bg.buttons, button)
	return bg
}

// View renders the button group
func (bg *ButtonGroup) View() string {
	if len(bg.buttons) == 0 {
		return ""
	}

	var buttonStrings []string
	for _, button := range bg.buttons {
		buttonStrings = append(buttonStrings, button.View())
	}

	spacer := strings.Repeat(" ", bg.spacing)
	return strings.Join(buttonStrings, spacer)
}
