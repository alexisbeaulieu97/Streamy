package components

import (
	"github.com/charmbracelet/lipgloss"
)

// Header represents a heading or title component.
type Header struct {
	BaseComponent
	title    string
	subtitle string
	level    int
}

// NewHeader creates a new header with the given title.
func NewHeader(title string) *Header {
	return &Header{
		BaseComponent: NewBaseComponent(),
		title:         title,
		level:         1,
	}
}

// View renders the header.
func (h *Header) View() string {
	return h.ViewWithContext(DefaultContext())
}

// ViewWithContext renders the header with the given theme context.
func (h *Header) ViewWithContext(ctx RenderContext) string {
	style := h.ComputeStyle(ctx.Theme)

	if h.subtitle == "" {
		return style.Render(h.title)
	}

	// Render title and subtitle together
	titleStyle := style
	subtitleStyle := style.Faint(true)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render(h.title),
		subtitleStyle.Render(h.subtitle),
	)

	return content
}

// WithStyle sets the header style.
func (h *Header) WithStyle(style lipgloss.Style) *Header {
	h.SetStyle(style)
	return h
}

// WithAppliers applies theme-based style modifiers.
func (h *Header) WithAppliers(appliers ...StyleFunc) *Header {
	h.SetAppliers(appliers...)
	return h
}

// WithSubtitle adds a subtitle to the header.
func (h *Header) WithSubtitle(subtitle string) *Header {
	h.subtitle = subtitle
	return h
}

// WithLevel sets the header level (1-6, like HTML h1-h6).
func (h *Header) WithLevel(level int) *Header {
	if level < 1 {
		level = 1
	}
	if level > 6 {
		level = 6
	}
	h.level = level
	return h
}

// Title returns the header title.
func (h *Header) Title() string {
	return h.title
}

// Subtitle returns the header subtitle.
func (h *Header) Subtitle() string {
	return h.subtitle
}

// Level returns the header level.
func (h *Header) Level() int {
	return h.level
}
