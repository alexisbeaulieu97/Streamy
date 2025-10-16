package components

import (
	"github.com/charmbracelet/lipgloss"
)

// Badge is a small status indicator component.
type Badge struct {
	BaseComponent
	text    string
	variant BadgeVariant
}

// BadgeVariant specifies the visual style of a badge.
type BadgeVariant int

const (
	BadgeVariantDefault BadgeVariant = iota
	BadgeVariantPrimary
	BadgeVariantSecondary
	BadgeVariantSuccess
	BadgeVariantWarning
	BadgeVariantError
	BadgeVariantInfo
)

// NewBadge creates a new badge with the given text.
func NewBadge(text string) *Badge {
	return &Badge{
		BaseComponent: NewBaseComponent(),
		text:          text,
		variant:       BadgeVariantDefault,
	}
}

// View renders the badge.
func (b *Badge) View() string {
	return b.ViewWithContext(DefaultContext())
}

// ViewWithContext renders the badge with the given theme context.
func (b *Badge) ViewWithContext(ctx RenderContext) string {
	style := b.computeStyle(ctx.Theme)
	return style.Render(b.text)
}

func (b *Badge) computeStyle(theme Theme) lipgloss.Style {
	baseStyle := b.ComputeStyle(theme)

	// Use variant registry for consistent styling
	if theme.Variants == nil {
		return baseStyle
	}

	if strategy := theme.Variants.Get(b.variant); strategy != nil {
		return strategy.Apply(baseStyle, theme)
	}

	// Fallback to base style if variant not registered
	return baseStyle
}

// WithVariant sets the badge variant.
func (b *Badge) WithVariant(variant BadgeVariant) *Badge {
	b.variant = variant
	return b
}

// WithStyle sets the badge style.
func (b *Badge) WithStyle(style lipgloss.Style) *Badge {
	b.SetStyle(style)
	return b
}

// WithAppliers applies theme-based style modifiers.
func (b *Badge) WithAppliers(appliers ...StyleFunc) *Badge {
	b.AddAppliers(appliers...)
	return b
}

// Text returns the badge text.
func (b *Badge) Text() string {
	return b.text
}

// SetText updates the badge text.
func (b *Badge) SetText(text string) *Badge {
	b.text = text
	return b
}

// Convenience constructors for different badge variants

// PrimaryBadge creates a primary badge.
func PrimaryBadge(text string) *Badge {
	return NewBadge(text).WithVariant(BadgeVariantPrimary)
}

// SecondaryBadge creates a secondary badge.
func SecondaryBadge(text string) *Badge {
	return NewBadge(text).WithVariant(BadgeVariantSecondary)
}

// SuccessBadge creates a success badge.
func SuccessBadge(text string) *Badge {
	return NewBadge(text).WithVariant(BadgeVariantSuccess)
}

// WarningBadge creates a warning badge.
func WarningBadge(text string) *Badge {
	return NewBadge(text).WithVariant(BadgeVariantWarning)
}

// ErrorBadge creates an error badge.
func ErrorBadge(text string) *Badge {
	return NewBadge(text).WithVariant(BadgeVariantError)
}

// InfoBadge creates an info badge.
func InfoBadge(text string) *Badge {
	return NewBadge(text).WithVariant(BadgeVariantInfo)
}
