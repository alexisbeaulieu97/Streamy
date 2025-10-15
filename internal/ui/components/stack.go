package components

import (
	"strings"

	"github.com/alexisbeaulieu97/streamy/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// Direction specifies the layout direction for a Stack.
type Direction int

const (
	DirectionVertical Direction = iota
	DirectionHorizontal
)

// Stack is a layout component that arranges children in a single direction.
// It replaces the old Box component with better semantics and functionality.
type Stack struct {
	BaseComponent
	children    []ui.Renderable
	direction   Direction
	gap         int
	mainAlign   MainAxisAlignment
	crossAlign  CrossAxisAlignment
	constraints Constraints
}

// NewStack creates a new stack with default vertical layout.
func NewStack(children ...ui.Renderable) *Stack {
	return &Stack{
		BaseComponent: NewBaseComponent(),
		children:      children,
		direction:     DirectionVertical,
		gap:           0,
		mainAlign:     MainStart,
		crossAlign:    CrossStart,
		constraints:   Unconstrained(),
	}
}

// VStack creates a vertical stack (convenience constructor).
func VStack(children ...ui.Renderable) *Stack {
	return NewStack(children...).WithDirection(DirectionVertical)
}

// HStack creates a horizontal stack (convenience constructor).
func HStack(children ...ui.Renderable) *Stack {
	return NewStack(children...).WithDirection(DirectionHorizontal)
}

// View renders the stack and its children.
func (s *Stack) View() string {
	return s.ViewWithContext(DefaultContext())
}

// ViewWithContext renders the stack with layout context.
func (s *Stack) ViewWithContext(ctx RenderContext) string {
	if len(s.children) == 0 {
		return s.ComputeStyle(ctx.Theme).Render("")
	}

	// Merge stack constraints with parent context constraints
	effectiveConstraints := s.mergeConstraints(ctx.Constraints)
	
	// Derive child constraints based on layout direction
	childCtx := ctx.WithConstraints(s.deriveChildConstraints(effectiveConstraints))

	// Render all children with propagated constraints
	childViews := make([]string, 0, len(s.children))
	for _, child := range s.children {
		if child == nil {
			continue
		}

		var view string
		if contextual, ok := child.(ContextualRenderable); ok {
			view = contextual.ViewWithContext(childCtx)
		} else {
			view = child.View()
		}

		if view != "" {
			childViews = append(childViews, view)
		}
	}

	if len(childViews) == 0 {
		return s.ComputeStyle(ctx.Theme).Render("")
	}

	// Apply gap spacing
	var content string
	if s.direction == DirectionHorizontal {
		content = s.joinHorizontal(childViews)
	} else {
		content = s.joinVertical(childViews)
	}

	// Apply constraints to final content if specified
	finalStyle := s.ComputeStyle(ctx.Theme)
	if effectiveConstraints.MaxWidth > 0 {
		finalStyle = finalStyle.MaxWidth(effectiveConstraints.MaxWidth)
	}
	if effectiveConstraints.MaxHeight > 0 {
		finalStyle = finalStyle.MaxHeight(effectiveConstraints.MaxHeight)
	}

	return finalStyle.Render(content)
}

// mergeConstraints combines stack-level constraints with parent context constraints.
func (s *Stack) mergeConstraints(parentConstraints Constraints) Constraints {
	result := parentConstraints
	
	// Stack constraints override parent constraints if more restrictive
	if s.constraints.MaxWidth > 0 && (result.MaxWidth <= 0 || s.constraints.MaxWidth < result.MaxWidth) {
		result.MaxWidth = s.constraints.MaxWidth
	}
	if s.constraints.MaxHeight > 0 && (result.MaxHeight <= 0 || s.constraints.MaxHeight < result.MaxHeight) {
		result.MaxHeight = s.constraints.MaxHeight
	}
	if s.constraints.MinWidth > result.MinWidth {
		result.MinWidth = s.constraints.MinWidth
	}
	if s.constraints.MinHeight > result.MinHeight {
		result.MinHeight = s.constraints.MinHeight
	}
	
	return result
}

// deriveChildConstraints computes constraints for child components based on layout direction.
func (s *Stack) deriveChildConstraints(parentConstraints Constraints) Constraints {
	childConstraints := parentConstraints
	
	// For horizontal stacks, divide width among children
	if s.direction == DirectionHorizontal && parentConstraints.MaxWidth > 0 && len(s.children) > 0 {
		// Account for gaps
		totalGap := s.gap * (len(s.children) - 1)
		availableWidth := parentConstraints.MaxWidth - totalGap
		if availableWidth > 0 {
			childConstraints.MaxWidth = availableWidth / len(s.children)
		}
	}
	
	// For vertical stacks, width propagates unchanged
	// Height division would require measuring rendered content first
	
	return childConstraints
}

func (s *Stack) joinVertical(views []string) string {
	if s.gap == 0 {
		return lipgloss.JoinVertical(s.crossAlign.toLipglossPosition(), views...)
	}

	// Insert gap rows between children
	spacer := strings.Repeat("\n", s.gap)
	result := make([]string, 0, len(views)*2-1)
	for i, view := range views {
		if i > 0 {
			result = append(result, spacer)
		}
		result = append(result, view)
	}

	return lipgloss.JoinVertical(s.crossAlign.toLipglossPosition(), result...)
}

func (s *Stack) joinHorizontal(views []string) string {
	if s.gap == 0 {
		return lipgloss.JoinHorizontal(s.crossAlign.toLipglossPosition(), views...)
	}

	// Insert gap columns between children
	spacer := strings.Repeat(" ", s.gap)
	result := make([]string, 0, len(views)*2-1)
	for i, view := range views {
		if i > 0 {
			result = append(result, spacer)
		}
		result = append(result, view)
	}

	return lipgloss.JoinHorizontal(s.crossAlign.toLipglossPosition(), result...)
}

// WithDirection sets the layout direction.
func (s *Stack) WithDirection(dir Direction) *Stack {
	s.direction = dir
	return s
}

// WithGap sets the spacing between children.
func (s *Stack) WithGap(gap int) *Stack {
	s.gap = gap
	return s
}

// WithMainAlign sets the main axis alignment.
func (s *Stack) WithMainAlign(align MainAxisAlignment) *Stack {
	s.mainAlign = align
	return s
}

// WithCrossAlign sets the cross axis alignment.
func (s *Stack) WithCrossAlign(align CrossAxisAlignment) *Stack {
	s.crossAlign = align
	return s
}

// WithStyle sets the container style.
func (s *Stack) WithStyle(style lipgloss.Style) *Stack {
	s.SetStyle(style)
	return s
}

// WithAppliers applies theme-based style modifiers.
func (s *Stack) WithAppliers(appliers ...StyleFunc) *Stack {
	s.SetAppliers(appliers...)
	return s
}

// WithConstraints sets sizing constraints.
func (s *Stack) WithConstraints(constraints Constraints) *Stack {
	s.constraints = constraints
	return s
}

// Add appends children to the stack.
func (s *Stack) Add(children ...ui.Renderable) *Stack {
	s.children = append(s.children, children...)
	return s
}

// Children returns the child renderables.
func (s *Stack) Children() []ui.Renderable {
	return s.children
}

// SetChildren replaces all children in the stack.
func (s *Stack) SetChildren(children []ui.Renderable) *Stack {
	s.children = children
	return s
}

// Helper to convert CrossAxisAlignment to lipgloss.Position
func (c CrossAxisAlignment) toLipglossPosition() lipgloss.Position {
	switch c {
	case CrossStart:
		return lipgloss.Left
	case CrossCenter:
		return lipgloss.Center
	case CrossEnd:
		return lipgloss.Right
	default:
		return lipgloss.Left
	}
}
