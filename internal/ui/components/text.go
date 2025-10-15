package components

import "github.com/charmbracelet/lipgloss"

// Text is a primitive component for rendering styled text content.
type Text struct {
	BaseComponent
	content string
}

// NewText creates a new text component with the given content.
func NewText(content string) *Text {
	return &Text{
		BaseComponent: NewBaseComponent(),
		content:       content,
	}
}

// View renders the text with its styling.
func (t *Text) View() string {
	return t.ViewWithContext(DefaultContext())
}

// ViewWithContext renders the text with the given theme context.
func (t *Text) ViewWithContext(ctx RenderContext) string {
	return t.ComputeStyle(ctx.Theme).Render(t.content)
}

// Content returns the text content.
func (t *Text) Content() string {
	return t.content
}

// SetContent updates the text content.
func (t *Text) SetContent(content string) *Text {
	t.content = content
	return t
}

// WithStyle sets the lipgloss style directly.
func (t *Text) WithStyle(style lipgloss.Style) *Text {
	t.SetStyle(style)
	return t
}

// WithAppliers applies theme-based style modifiers.
func (t *Text) WithAppliers(appliers ...StyleFunc) *Text {
	t.SetAppliers(appliers...)
	return t
}

// WithStrategy sets a custom styling strategy.
func (t *Text) WithStrategy(strategy StyleStrategy) *Text {
	t.SetStrategy(strategy)
	return t
}

// Theme-aware text constructor helpers

// BoldText creates bold text using theme typography.
func BoldText(content string) *Text {
	return NewText(content).WithAppliers(Typography(TypographyVariantFontBold))
}

// EmphasisText creates emphasized text using theme typography.
func EmphasisText(content string) *Text {
	return NewText(content).WithAppliers(Typography(TypographyVariantEmphasis))
}

// CodeText creates code-styled text using theme typography.
func CodeText(content string) *Text {
	return NewText(content).WithAppliers(Typography(TypographyVariantCode))
}

// TitleText creates title text using theme typography.
func TitleText(content string) *Text {
	return NewText(content).WithAppliers(Typography(TypographyVariantTitle))
}

// SubtitleText creates subtitle text using theme typography.
func SubtitleText(content string) *Text {
	return NewText(content).WithAppliers(Typography(TypographyVariantSubtitle))
}
