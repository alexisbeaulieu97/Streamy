package components

import (
	"github.com/charmbracelet/lipgloss"
)

// Button represents an interactive button component (visual only for now).
type Button struct {
	BaseComponent
	label    string
	variant  ButtonVariant
	disabled bool
	active   bool
}

// NewButton creates a new button with the given label.
func NewButton(label string) *Button {
	return &Button{
		BaseComponent: NewBaseComponent(),
		label:         label,
		variant:       ButtonVariantPrimary,
		disabled:      false,
		active:        false,
	}
}

// View renders the button.
func (b *Button) View() string {
	return b.ViewWithContext(DefaultContext())
}

// ViewWithContext renders the button with the given theme context.
func (b *Button) ViewWithContext(ctx RenderContext) string {
	style := b.computeStyle(ctx.Theme)
	return style.Render(b.label)
}

func (b *Button) computeStyle(theme Theme) lipgloss.Style {
	// Get base style
	baseStyle := b.ComputeStyle(theme)

	// Use variant registry for consistent styling
	var style lipgloss.Style
	if strategy := theme.Variants.Get(b.variant); strategy != nil {
		style = strategy.Apply(baseStyle, theme)
	} else {
		style = baseStyle
	}

	// Apply state-specific styling
	if b.disabled {
		style = style.Faint(true)
	}

	if b.active {
		style = style.Bold(true).Underline(true)
	}

	return style
}

// WithVariant sets the button variant.
func (b *Button) WithVariant(variant ButtonVariant) *Button {
	b.variant = variant
	return b
}

// WithDisabled sets the disabled state.
func (b *Button) WithDisabled(disabled bool) *Button {
	b.disabled = disabled
	return b
}

// WithActive sets the active/selected state.
func (b *Button) WithActive(active bool) *Button {
	b.active = active
	return b
}

// WithStyle sets the button style.
func (b *Button) WithStyle(style lipgloss.Style) *Button {
	b.SetStyle(style)
	return b
}

// WithAppliers applies theme-based style modifiers.
func (b *Button) WithAppliers(appliers ...StyleFunc) *Button {
	b.AddAppliers(appliers...)
	return b
}

// Label returns the button label.
func (b *Button) Label() string {
	return b.label
}

// SetLabel updates the button label.
func (b *Button) SetLabel(label string) *Button {
	b.label = label
	return b
}

// IsDisabled returns true if the button is disabled.
func (b *Button) IsDisabled() bool {
	return b.disabled
}

// IsActive returns true if the button is active.
func (b *Button) IsActive() bool {
	return b.active
}

// Convenience constructors for different button variants

// PrimaryButton creates a primary button.
func PrimaryButton(label string) *Button {
	return NewButton(label).WithVariant(ButtonVariantPrimary)
}

// SecondaryButton creates a secondary button.
func SecondaryButton(label string) *Button {
	return NewButton(label).WithVariant(ButtonVariantSecondary)
}

// SuccessButton creates a success button.
func SuccessButton(label string) *Button {
	return NewButton(label).WithVariant(ButtonVariantSuccess)
}

// ErrorButton creates an error/danger button.
func ErrorButton(label string) *Button {
	return NewButton(label).WithVariant(ButtonVariantError)
}

// WarningButton creates a warning button.
func WarningButton(label string) *Button {
	return NewButton(label).WithVariant(ButtonVariantWarning)
}

// InfoButton creates an info button.
func InfoButton(label string) *Button {
	return NewButton(label).WithVariant(ButtonVariantInfo)
}

// MutedButton creates a muted/neutral button.
func MutedButton(label string) *Button {
	return NewButton(label).WithVariant(ButtonVariantMuted)
}
