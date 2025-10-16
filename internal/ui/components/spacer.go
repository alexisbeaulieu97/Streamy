package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Spacer is a primitive component that renders empty space.
// Useful for adding fixed spacing between components.
type Spacer struct {
	BaseComponent
	width  int
	height int
}

// NewSpacer creates a spacer with the given dimensions.
func NewSpacer(width, height int) *Spacer {
	return &Spacer{
		BaseComponent: NewBaseComponent(),
		width:         width,
		height:        height,
	}
}

// HorizontalSpacer creates a horizontal spacer (width only).
func HorizontalSpacer(width int) *Spacer {
	return NewSpacer(width, 1)
}

// VerticalSpacer creates a vertical spacer (height only).
func VerticalSpacer(height int) *Spacer {
	return NewSpacer(0, height)
}

// View renders the spacer as empty space.
func (s *Spacer) View() string {
	w := s.width
	if w < 0 {
		w = 0
	}
	h := s.height
	if h < 0 {
		h = 0
	}

	if w == 0 && h == 0 {
		return ""
	}

	// Create horizontal space
	line := strings.Repeat(" ", w)

	// Create vertical space
	if h > 1 {
		lines := make([]string, h)
		for i := range lines {
			lines[i] = line
		}
		return lipgloss.JoinVertical(lipgloss.Left, lines...)
	}

	return line
}

// Width returns the spacer width.
func (s *Spacer) Width() int {
	return s.width
}

// Height returns the spacer height.
func (s *Spacer) Height() int {
	return s.height
}

// WithWidth sets the spacer width.
func (s *Spacer) WithWidth(width int) *Spacer {
	s.width = width
	return s
}

// WithHeight sets the spacer height.
func (s *Spacer) WithHeight(height int) *Spacer {
	s.height = height
	return s
}
