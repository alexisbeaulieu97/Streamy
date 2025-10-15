package components

import (
	"github.com/alexisbeaulieu97/streamy/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// Card is a specialized container with default styling for card-like UI elements.
// It's a semantic component built on top of Container.
type Card struct {
	*Container
}

// NewCard creates a new card with default card styling.
func NewCard(children ...ui.Renderable) *Card {
	container := NewContainer(children...).
		WithBorder(lipgloss.RoundedBorder()).
		WithPadding(UniformSpacing(1))

	// Apply default card theme
	container.WithAppliers(CardBaseStyle()...)

	return &Card{
		Container: container,
	}
}

// WithTitle adds a title to the card (convenience method).
func (c *Card) WithTitle(title string) *Card {
	// Prepend a header to the children
	header := NewHeader(title).WithAppliers(
		Typography(TypographyVariantTitle),
	)

	// Create a new layout with header first
	allChildren := make([]ui.Renderable, 0, len(c.Children())+1)
	allChildren = append(allChildren, header)
	allChildren = append(allChildren, c.Children()...)

	// Update children and layout
	c.SetChildren(allChildren)
	c.SetLayout(VStack(allChildren...))

	return c
}

// WithFooter adds a footer to the card (convenience method).
func (c *Card) WithFooter(footer ui.Renderable) *Card {
	c.Add(HorizontalDivider(), footer)
	return c
}

// AsContainer returns the underlying container for advanced customization.
func (c *Card) AsContainer() *Container {
	return c.Container
}
