package components

import (
	"github.com/alexisbeaulieu97/streamy/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// Alert is a composite component for displaying notifications and messages.
type Alert struct {
	BaseComponent
	message string
	icon    string
	variant AlertVariant
	title   string
}

// NewAlert creates a new alert with the given message.
func NewAlert(message string) *Alert {
	return &Alert{
		BaseComponent: NewBaseComponent(),
		message:       message,
		variant:       AlertVariantInfo,
		icon:          "ℹ",
	}
}

// View renders the alert.
func (a *Alert) View() string {
	return a.ViewWithContext(DefaultContext())
}

// ViewWithContext renders the alert with the provided render context.
func (a *Alert) ViewWithContext(ctx RenderContext) string {
	// Build content
	var children []ui.Renderable

	// Create the main message line with icon
	messageLine := a.icon + " " + a.message
	if a.title != "" {
		titleText := EmphasisText(a.title)
		messageText := NewText(messageLine)
		children = []ui.Renderable{titleText, messageText}
	} else {
		messageText := NewText(messageLine)
		children = []ui.Renderable{messageText}
	}

	// Create container with appropriate styling
	container := NewContainer(children...).
		WithPadding(UniformSpacing(1)).
		WithBorder(lipgloss.NormalBorder())

	theme := ctx.Theme
	if theme.Variants == nil {
		theme = DefaultTheme()
		ctx = ctx.WithTheme(theme)
	}

	var styleFuncs []StyleFunc

	if theme.Variants != nil {
		if strategy := theme.Variants.Get(a.variant); strategy != nil {
			styleFuncs = append(styleFuncs, func(base lipgloss.Style, theme Theme) lipgloss.Style {
				return strategy.Apply(base, theme)
			})
		}
	}

	if borderFunc := alertBorderColorFunc(a.variant); borderFunc != nil {
		styleFuncs = append(styleFuncs, borderFunc)
	}

	if len(styleFuncs) > 0 {
		container.WithAppliers(styleFuncs...)
	}

	return container.ViewWithContext(ctx)
}

// WithVariant sets the alert variant.
func (a *Alert) WithVariant(variant AlertVariant) *Alert {
	a.variant = variant

	// Update icon based on variant
	switch variant {
	case AlertVariantSuccess:
		a.icon = "✓"
	case AlertVariantWarning:
		a.icon = "⚠"
	case AlertVariantError:
		a.icon = "✗"
	case AlertVariantInfo:
		a.icon = "ℹ"
	}

	return a
}

// WithIcon sets a custom icon.
func (a *Alert) WithIcon(icon string) *Alert {
	a.icon = icon
	return a
}

// WithTitle adds a title to the alert.
func (a *Alert) WithTitle(title string) *Alert {
	a.title = title
	return a
}

// WithStyle sets the alert style (applied to container).
func (a *Alert) WithStyle(style lipgloss.Style) *Alert {
	a.SetStyle(style)
	return a
}

// WithAppliers applies theme-based style modifiers.
func (a *Alert) WithAppliers(appliers ...StyleFunc) *Alert {
	a.AddAppliers(appliers...)
	return a
}

// Message returns the alert message.
func (a *Alert) Message() string {
	return a.message
}

// SetMessage updates the alert message.
func (a *Alert) SetMessage(message string) *Alert {
	a.message = message
	return a
}

// Convenience constructors for different alert variants

// SuccessAlert creates a success alert.
func SuccessAlert(message string) *Alert {
	return NewAlert(message).WithVariant(AlertVariantSuccess)
}

// WarningAlert creates a warning alert.
func WarningAlert(message string) *Alert {
	return NewAlert(message).WithVariant(AlertVariantWarning)
}

// ErrorAlert creates an error alert.
func ErrorAlert(message string) *Alert {
	return NewAlert(message).WithVariant(AlertVariantError)
}

// InfoAlert creates an info alert.
func InfoAlert(message string) *Alert {
	return NewAlert(message).WithVariant(AlertVariantInfo)
}

func alertBorderColorFunc(variant AlertVariant) StyleFunc {
	return func(base lipgloss.Style, theme Theme) lipgloss.Style {
		var colour lipgloss.AdaptiveColor

		switch variant {
		case AlertVariantSuccess:
			colour = theme.Palette.Success.Base
		case AlertVariantWarning:
			colour = theme.Palette.Warning.Base
		case AlertVariantError:
			colour = theme.Palette.Danger.Base
		case AlertVariantInfo:
			colour = theme.Palette.Info.Base
		default:
			return base
		}

		if colour.Light == "" && colour.Dark == "" {
			return base
		}

		return base.BorderForeground(colour)
	}
}
