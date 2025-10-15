package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Divider renders a visual separator line (renamed from Separator for clarity).
type Divider struct {
	BaseComponent
	char      string
	width     int
	direction Direction
}

// NewDivider creates a divider with the specified character.
func NewDivider() *Divider {
	return &Divider{
		BaseComponent: NewBaseComponent(),
		char:          "─",
		width:         0, // 0 means auto-width
		direction:     DirectionHorizontal,
	}
}

// HorizontalDivider creates a horizontal divider (convenience constructor).
func HorizontalDivider() *Divider {
	return NewDivider()
}

// VerticalDivider creates a vertical divider.
func VerticalDivider() *Divider {
	return NewDivider().WithChar("│").WithDirection(DirectionVertical)
}

// View renders the divider.
func (d *Divider) View() string {
	return d.ViewWithContext(DefaultContext())
}

// ViewWithContext renders the divider with layout context.
func (d *Divider) ViewWithContext(ctx RenderContext) string {
	width := d.width

	// Use constraint width if no explicit width is set
	if width <= 0 && ctx.Constraints.HasWidth() {
		if ctx.Constraints.MaxWidth >= 0 {
			width = ctx.Constraints.MaxWidth
		} else if ctx.Constraints.MinWidth > 0 {
			width = ctx.Constraints.MinWidth
		}
	}

	// Use parent width if available
	if width <= 0 && ctx.ParentWidth > 0 {
		width = ctx.ParentWidth
	}

	// Default width if still not set
	if width <= 0 {
		width = 40
	}

	var content string
	if d.direction == DirectionHorizontal {
		content = strings.Repeat(d.char, width)
	} else {
		// Vertical divider: repeat character vertically
		lines := make([]string, width)
		for i := range lines {
			lines[i] = d.char
		}
		content = lipgloss.JoinVertical(lipgloss.Left, lines...)
	}

	return d.ComputeStyle(ctx.Theme).Render(content)
}

// WithChar sets the character used for the divider.
func (d *Divider) WithChar(char string) *Divider {
	if char != "" {
		d.char = char
	}
	return d
}

// WithWidth sets an explicit width for the divider.
func (d *Divider) WithWidth(width int) *Divider {
	d.width = width
	return d
}

// WithDirection sets the divider direction.
func (d *Divider) WithDirection(dir Direction) *Divider {
	d.direction = dir
	return d
}

// WithStyle sets the divider style.
func (d *Divider) WithStyle(style lipgloss.Style) *Divider {
	d.SetStyle(style)
	return d
}

// WithAppliers applies theme-based style modifiers.
func (d *Divider) WithAppliers(appliers ...StyleFunc) *Divider {
	d.SetAppliers(appliers...)
	return d
}

// Width returns the divider width.
func (d *Divider) Width() int {
	return d.width
}

// Predefined divider styles

// DashedDivider creates a dashed divider.
func DashedDivider() *Divider {
	return NewDivider().WithChar("-")
}

// DottedDivider creates a dotted divider.
func DottedDivider() *Divider {
	return NewDivider().WithChar("·")
}

// DoubleDivider creates a double-line divider.
func DoubleDivider() *Divider {
	return NewDivider().WithChar("═")
}

// ThickDivider creates a thick divider.
func ThickDivider() *Divider {
	return NewDivider().WithChar("━")
}
