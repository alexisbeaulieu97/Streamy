package components

import (
	"github.com/alexisbeaulieu97/streamy/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// Container is a generic box that can hold children with border, padding, and styling.
// It's the foundation for more specialized components like Card and Panel.
type Container struct {
	BaseComponent
	children    []ui.Renderable
	layout      *Stack
	border      lipgloss.Border
	borderColor string
	padding     Spacing
	margin      Spacing
}

// NewContainer creates a new container with default settings.
func NewContainer(children ...ui.Renderable) *Container {
	return &Container{
		BaseComponent: NewBaseComponent(),
		children:      children,
		layout:        VStack(children...),
		padding:       Spacing{}, // Zero spacing by default
		margin:        Spacing{},
	}
}

// View renders the container and its children.
func (c *Container) View() string {
	return c.ViewWithContext(DefaultContext())
}

// ViewWithContext renders the container with layout context.
func (c *Container) ViewWithContext(ctx RenderContext) string {
	// Render the layout if children exist, otherwise use empty content so
	// borders/padding/margins still render.
	var content string
	if len(c.children) > 0 {
		content = c.layout.ViewWithContext(ctx)
	}

	// Build the container style
	containerStyle := c.ComputeStyle(ctx.Theme)

	// Apply border if set
	if c.border.Top != "" {
		containerStyle = containerStyle.
			BorderStyle(c.border)

		if c.borderColor != "" {
			containerStyle = containerStyle.BorderForeground(lipgloss.Color(c.borderColor))
		}
	}

	// Apply padding using Spacing value object
	if !c.padding.IsZero() {
		// lipgloss.Padding takes (top, right, bottom, left) or (vertical, horizontal)
		if c.padding.Top == c.padding.Bottom && c.padding.Left == c.padding.Right {
			// Symmetric spacing
			containerStyle = containerStyle.Padding(c.padding.Top, c.padding.Left)
		} else {
			// Full four-value padding
			containerStyle = containerStyle.Padding(c.padding.Top, c.padding.Right, c.padding.Bottom, c.padding.Left)
		}
	}

	// Apply margin if set
	if !c.margin.IsZero() {
		if c.margin.Top == c.margin.Bottom && c.margin.Left == c.margin.Right {
			containerStyle = containerStyle.Margin(c.margin.Top, c.margin.Left)
		} else {
			containerStyle = containerStyle.Margin(c.margin.Top, c.margin.Right, c.margin.Bottom, c.margin.Left)
		}
	}

	return containerStyle.Render(content)
}

// WithBorder sets the border style.
func (c *Container) WithBorder(border lipgloss.Border) *Container {
	c.border = border
	return c
}

// WithBorderColor sets the border color.
func (c *Container) WithBorderColor(color string) *Container {
	c.borderColor = color
	return c
}

// WithPadding sets the padding using a Spacing value object.
func (c *Container) WithPadding(padding Spacing) *Container {
	c.padding = padding
	return c
}

// WithMargin sets the margin using a Spacing value object.
func (c *Container) WithMargin(margin Spacing) *Container {
	c.margin = margin
	return c
}

// WithStyle sets the container style.
func (c *Container) WithStyle(style lipgloss.Style) *Container {
	c.SetStyle(style)
	return c
}

// WithAppliers applies theme-based style modifiers.
func (c *Container) WithAppliers(appliers ...StyleFunc) *Container {
	c.SetAppliers(appliers...)
	return c
}

// WithDirection sets the layout direction.
func (c *Container) WithDirection(dir Direction) *Container {
	c.layout.WithDirection(dir)
	return c
}

// WithGap sets the gap between children.
func (c *Container) WithGap(gap int) *Container {
	c.layout.WithGap(gap)
	return c
}

// WithCrossAlign sets the cross-axis alignment.
func (c *Container) WithCrossAlign(align CrossAxisAlignment) *Container {
	c.layout.WithCrossAlign(align)
	return c
}

// Add appends children to the container.
func (c *Container) Add(children ...ui.Renderable) *Container {
	c.children = append(c.children, children...)
	c.layout.Add(children...)
	return c
}

// Children returns the child renderables.
func (c *Container) Children() []ui.Renderable {
	return c.children
}

// Layout returns the internal stack layout.
func (c *Container) Layout() *Stack {
	return c.layout
}

// SetChildren replaces all children in the container.
func (c *Container) SetChildren(children []ui.Renderable) *Container {
	c.children = children
	c.layout.SetChildren(children)
	return c
}

// SetLayout replaces the container's layout.
func (c *Container) SetLayout(layout *Stack) *Container {
	if layout == nil {
		return c
	}
	c.layout = layout
	return c
}
