package components

import (
	"github.com/alexisbeaulieu97/streamy/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// Panel is a container for grouping related content into sections.
// Unlike Card, it has less prominent styling and is meant for layout organization.
type Panel struct {
	*Container
	header ui.Renderable
	footer ui.Renderable
}

// NewPanel creates a new panel with default styling.
func NewPanel(children ...ui.Renderable) *Panel {
	container := NewContainer(children...).
		WithPadding(UniformSpacing(1))

	// Apply subtle panel theme
	container = container.WithAppliers(
		Background(PaletteSurface),
	)

	return &Panel{
		Container: container,
	}
}

// WithHeader adds a header to the panel.
func (p *Panel) WithHeader(header ui.Renderable) *Panel {
	// Store the prior header before overwriting it
	priorHeader := p.header
	p.header = header

	// Get current children and conditionally strip existing header/divider pattern
	currentChildren := p.Children()
	remainingChildren := currentChildren

	// Only strip header+divider when the existing first child equals the panel's prior header
	if len(currentChildren) >= 2 && priorHeader != nil {
		if currentChildren[0] == priorHeader {
			// Look for the pattern: our header followed by a divider
			if _, ok := currentChildren[1].(*Divider); ok {
				// Strip the first two elements (our header + divider)
				remainingChildren = currentChildren[2:]
			}
		}
	}

	// Rebuild children with new header
	allChildren := []ui.Renderable{header, HorizontalDivider()}
	allChildren = append(allChildren, remainingChildren...)

	// Use Container's public API to update children and layout atomically
	p.SetChildren(allChildren)
	p.SetLayout(VStack(allChildren...))
	return p
}

// WithFooter adds a footer to the panel.
func (p *Panel) WithFooter(footer ui.Renderable) *Panel {
	p.footer = footer

	currentChildren := p.Children()
	remainingChildren := currentChildren

	// Strip existing footer pattern (divider + footer) if present at the end.
	// Only strip when penultimate is a Divider AND last child is not a Divider.
	if len(currentChildren) >= 2 {
		penultimate := currentChildren[len(currentChildren)-2]
		last := currentChildren[len(currentChildren)-1]
		if _, ok := penultimate.(*Divider); ok {
			if _, lastIsDivider := last.(*Divider); !lastIsDivider {
				remainingChildren = currentChildren[:len(currentChildren)-2]
			}
		}
	}

	allChildren := append([]ui.Renderable{}, remainingChildren...)
	allChildren = append(allChildren, HorizontalDivider(), footer)

	p.SetChildren(allChildren)
	p.SetLayout(VStack(allChildren...))
	return p
}

// WithTitle is a convenience method to add a text header.
func (p *Panel) WithTitle(title string) *Panel {
	header := NewHeader(title).WithAppliers(
		Typography(TypographyVariantTitle),
	)
	return p.WithHeader(header)
}

// WithBorder adds a border to the panel.
func (p *Panel) WithBorder(border lipgloss.Border) *Panel {
	p.Container.WithBorder(border)
	return p
}

// AsContainer returns the underlying container for advanced customization.
func (p *Panel) AsContainer() *Container {
	return p.Container
}
