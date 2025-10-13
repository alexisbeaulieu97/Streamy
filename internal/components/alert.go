package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// AlertOptions defines the configuration options for an alert
type AlertOptions struct {
	Variant     AlertVariant
	Title       string
	Dismissible bool
}

// Alert represents a message alert component
type Alert struct {
	message string
	options AlertOptions
}

// NewAlert creates a new alert with the given message and options
func NewAlert(message string, opts AlertOptions) *Alert {
	return &Alert{
		message: message,
		options: opts,
	}
}

// WithVariant sets the alert variant
func (a *Alert) WithVariant(variant AlertVariant) *Alert {
	a.options.Variant = variant
	return a
}

// WithTitle sets the alert title
func (a *Alert) WithTitle(title string) *Alert {
	a.options.Title = title
	return a
}

// WithDismissible sets whether the alert can be dismissed
func (a *Alert) WithDismissible(dismissible bool) *Alert {
	a.options.Dismissible = dismissible
	return a
}

// View renders the alert
func (a *Alert) View() string {
	style := a.buildStyle()
	var content []string

	if a.options.Title != "" {
		titleStyle := Style(lipgloss.NewStyle(), Typography(TypographyVariantEmphasis))
		content = append(content, titleStyle.Render(a.options.Title))
	}

	if a.message != "" {
		content = append(content, a.message)
	}

	if a.options.Dismissible {
		indicator := Style(
			lipgloss.NewStyle(),
			Typography(TypographyVariantEmphasis),
			Foreground(PaletteSurface),
		).Render("[×]")
		content = append(content, indicator)
	}

	innerContent := strings.Join(content, "\n")
	return style.Render(innerContent)
}

// buildStyle calculates the alert style based on current options
func (a *Alert) buildStyle() lipgloss.Style {
	appliers := alertVariantAppliers(a.options.Variant)
	return Style(lipgloss.NewStyle(), appliers...)
}

func alertVariantAppliers(variant AlertVariant) []StyleApplier {
	switch variant {
	case AlertVariantSuccess:
		return AlertSuccessStyle()
	case AlertVariantError:
		return AlertErrorStyle()
	case AlertVariantWarning:
		return AlertWarningStyle()
	case AlertVariantInfo:
		return AlertInfoStyle()
	default:
		return AlertInfoStyle()
	}
}

// SimpleAlert creates an alert with sensible defaults
func SimpleAlert(message string) *Alert {
	return NewAlert(message, AlertOptions{
		Variant:     AlertVariantInfo,
		Title:       "",
		Dismissible: false,
	})
}

// SuccessAlert creates a success alert
func SuccessAlert(message string) *Alert {
	return NewAlert(message, AlertOptions{
		Variant:     AlertVariantSuccess,
		Title:       "Success",
		Dismissible: true,
	})
}

// ErrorAlert creates an error alert
func ErrorAlert(message string) *Alert {
	return NewAlert(message, AlertOptions{
		Variant:     AlertVariantError,
		Title:       "Error",
		Dismissible: true,
	})
}

// WarningAlert creates a warning alert
func WarningAlert(message string) *Alert {
	return NewAlert(message, AlertOptions{
		Variant:     AlertVariantWarning,
		Title:       "Warning",
		Dismissible: true,
	})
}

// InfoAlert creates an info alert
func InfoAlert(message string) *Alert {
	return NewAlert(message, AlertOptions{
		Variant:     AlertVariantInfo,
		Title:       "Info",
		Dismissible: true,
	})
}
