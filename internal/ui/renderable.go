// Package ui defines common rendering abstractions shared across the TUI components.
package ui

// Renderable is a minimal interface implemented by components that can render themselves.
type Renderable interface {
	View() string
}
