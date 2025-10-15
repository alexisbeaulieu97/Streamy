package components

import (
	"github.com/alexisbeaulieu97/streamy/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// BaseComponent provides common functionality for all components.
// Embed this in your component structs to get standard behavior.
type BaseComponent struct {
	style    lipgloss.Style
	strategy StyleStrategy
}

// StyleStrategy defines how styling should be applied to a component.
// This abstraction allows for composable, testable styling logic.
type StyleStrategy interface {
	Apply(base lipgloss.Style, theme Theme) lipgloss.Style
}

// StyleFunc is a function that applies styling transformations to a lipgloss.Style
// using data from a Theme. This is the core abstraction for theme-aware styling.
type StyleFunc func(lipgloss.Style, Theme) lipgloss.Style

// CompositeStrategy applies multiple StyleFunc in sequence.
type CompositeStrategy struct {
	funcs []StyleFunc
}

// Apply applies all style functions in order.
func (c CompositeStrategy) Apply(base lipgloss.Style, theme Theme) lipgloss.Style {
	for _, fn := range c.funcs {
		base = fn(base, theme)
	}
	return base
}

// NewCompositeStrategy creates a strategy from multiple style functions.
func NewCompositeStrategy(funcs ...StyleFunc) StyleStrategy {
	return CompositeStrategy{funcs: funcs}
}

// NewBaseComponent creates a new base component with default styling.
func NewBaseComponent() BaseComponent {
	return BaseComponent{
		style:    lipgloss.NewStyle(),
		strategy: CompositeStrategy{}, // Empty strategy
	}
}

// ComputeStyle returns the computed style for this component using the provided theme.
// This replaces the old Style() method which relied on global theme state.
func (b *BaseComponent) ComputeStyle(theme Theme) lipgloss.Style {
	if b.strategy == nil {
		return b.style
	}
	return b.strategy.Apply(b.style, theme)
}

// SetStyle replaces the raw lipgloss style.
func (b *BaseComponent) SetStyle(style lipgloss.Style) {
	b.style = style
}

// SetStrategy replaces the style strategy.
func (b *BaseComponent) SetStrategy(strategy StyleStrategy) {
	b.strategy = strategy
}

// SetAppliers sets the style strategy from style functions.
func (b *BaseComponent) SetAppliers(appliers ...StyleFunc) {
	b.strategy = NewCompositeStrategy(appliers...)
}

// AddAppliers appends additional style appliers to the existing strategy.
// If the current strategy is not a CompositeStrategy, it wraps the existing
// strategy and appends the new appliers, preserving any custom strategy logic.
func (b *BaseComponent) AddAppliers(appliers ...StyleFunc) {
	if existing, ok := b.strategy.(CompositeStrategy); ok {
		// Make a defensive copy of the funcs slice to avoid mutating shared arrays
		newFuncs := make([]StyleFunc, len(existing.funcs), len(existing.funcs)+len(appliers))
		copy(newFuncs, existing.funcs)
		newFuncs = append(newFuncs, appliers...)
		b.strategy = CompositeStrategy{funcs: newFuncs}
	} else {
		// Wrap existing strategy with new appliers to preserve custom strategy logic
		currentStrategy := b.strategy
		wrapper := func(base lipgloss.Style, theme Theme) lipgloss.Style {
			// Apply existing strategy first if present
			if currentStrategy != nil {
				base = currentStrategy.Apply(base, theme)
			}
			// Then apply new appliers
			for _, applier := range appliers {
				base = applier(base, theme)
			}
			return base
		}
		b.strategy = NewCompositeStrategy(wrapper)
	}
}

// Spacing represents spacing (padding or margin) around a component.
// Uses CSS box model ordering: Top, Right, Bottom, Left (clockwise from top).
type Spacing struct {
	Top    int
	Right  int
	Bottom int
	Left   int
}

// UniformSpacing creates spacing with the same value on all sides.
func UniformSpacing(size int) Spacing {
	return Spacing{Top: size, Right: size, Bottom: size, Left: size}
}

// HorizontalSpacing creates spacing on left and right sides only.
func HorizontalSpacing(size int) Spacing {
	return Spacing{Top: 0, Right: size, Bottom: 0, Left: size}
}

// VerticalSpacing creates spacing on top and bottom sides only.
func VerticalSpacing(size int) Spacing {
	return Spacing{Top: size, Right: 0, Bottom: size, Left: 0}
}

// SymmetricSpacing creates spacing with different horizontal and vertical values.
func SymmetricSpacing(vertical, horizontal int) Spacing {
	return Spacing{Top: vertical, Right: horizontal, Bottom: vertical, Left: horizontal}
}

// CustomSpacing creates spacing with explicit values for each side (top, right, bottom, left).
func CustomSpacing(top, right, bottom, left int) Spacing {
	return Spacing{Top: top, Right: right, Bottom: bottom, Left: left}
}

// IsZero returns true if all spacing values are zero.
func (s Spacing) IsZero() bool {
	return s.Top == 0 && s.Right == 0 && s.Bottom == 0 && s.Left == 0
}

// Horizontal returns the total horizontal spacing (left + right).
func (s Spacing) Horizontal() int {
	return s.Left + s.Right
}

// Vertical returns the total vertical spacing (top + bottom).
func (s Spacing) Vertical() int {
	return s.Top + s.Bottom
}

// Constraints defines sizing constraints for layout calculations.
type Constraints struct {
	MinWidth  int
	MaxWidth  int
	MinHeight int
	MaxHeight int
}

// Unconstrained returns constraints with no limits.
func Unconstrained() Constraints {
	return Constraints{
		MinWidth:  0,
		MaxWidth:  -1, // -1 means unlimited
		MinHeight: 0,
		MaxHeight: -1,
	}
}

// WithWidth creates constraints with a fixed width.
func WithWidth(width int) Constraints {
	return Constraints{
		MinWidth:  width,
		MaxWidth:  width,
		MinHeight: 0,
		MaxHeight: -1,
	}
}

// WithMaxWidth creates constraints with a maximum width.
func WithMaxWidth(maxWidth int) Constraints {
	return Constraints{
		MinWidth:  0,
		MaxWidth:  maxWidth,
		MinHeight: 0,
		MaxHeight: -1,
	}
}

// Constrain applies the constraints to a given size.
func (c Constraints) Constrain(width, height int) (int, int) {
	w := width
	h := height

	if c.MinWidth > 0 && w < c.MinWidth {
		w = c.MinWidth
	}
	if c.MaxWidth != -1 && w > c.MaxWidth {
		w = c.MaxWidth
	}
	if c.MinHeight > 0 && h < c.MinHeight {
		h = c.MinHeight
	}
	if c.MaxHeight != -1 && h > c.MaxHeight {
		h = c.MaxHeight
	}

	return w, h
}

// HasWidth returns true if there's a width constraint.
func (c Constraints) HasWidth() bool {
	return c.MinWidth > 0 || c.MaxWidth >= 0
}

// HasHeight returns true if there's a height constraint.
func (c Constraints) HasHeight() bool {
	return c.MinHeight > 0 || c.MaxHeight >= 0
}

// RenderContext provides layout information and theme to components during rendering.
// This context-based approach eliminates global state and enables:
// - Parallel testing without theme conflicts
// - Multiple themes in the same application
// - Explicit data flow for better reasoning
type RenderContext struct {
	Theme       Theme
	Constraints Constraints
	ParentWidth int
}

// DefaultContext returns a render context with the default theme and no constraints.
func DefaultContext() RenderContext {
	return RenderContext{
		Theme:       DefaultTheme(),
		Constraints: Unconstrained(),
	}
}

// WithTheme returns a new context with the specified theme.
func (r RenderContext) WithTheme(theme Theme) RenderContext {
	r.Theme = theme
	return r
}

// WithConstraints returns a new context with the given constraints.
func (r RenderContext) WithConstraints(c Constraints) RenderContext {
	r.Constraints = c
	return r
}

// ContextualRenderable is a component that can receive layout context.
// This is an advanced interface; most components only need Renderable.
type ContextualRenderable interface {
	ui.Renderable
	ViewWithContext(ctx RenderContext) string
}

// Alignment specifies how content should be aligned.
type Alignment int

const (
	AlignStart Alignment = iota
	AlignCenter
	AlignEnd
	AlignStretch
)

// ToLipglossPosition converts Alignment to lipgloss.Position.
func (a Alignment) ToLipglossPosition() lipgloss.Position {
	switch a {
	case AlignStart:
		return lipgloss.Left
	case AlignCenter:
		return lipgloss.Center
	case AlignEnd:
		return lipgloss.Right
	default:
		return lipgloss.Left
	}
}

// MainAxisAlignment specifies how children are aligned along the main axis.
type MainAxisAlignment int

const (
	MainStart MainAxisAlignment = iota
	MainCenter
	MainEnd
	MainSpaceBetween
	MainSpaceAround
	MainSpaceEvenly
)

// CrossAxisAlignment specifies how children are aligned along the cross axis.
type CrossAxisAlignment int

const (
	CrossStart CrossAxisAlignment = iota
	CrossCenter
	CrossEnd
	CrossStretch
)
