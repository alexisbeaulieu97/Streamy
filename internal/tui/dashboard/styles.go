package dashboard

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	primaryColor    = lipgloss.Color("99")  // Purple
	successColor    = lipgloss.Color("42")  // Green
	warningColor    = lipgloss.Color("226") // Yellow
	errorColor      = lipgloss.Color("196") // Red
	mutedColor      = lipgloss.Color("245") // Gray
	accentColor     = lipgloss.Color("212") // Pink
	backgroundColor = lipgloss.Color("235") // Dark gray

	// Title style
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			PaddingLeft(2).
			PaddingRight(2).
			MarginBottom(1)

	// Header style
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(mutedColor).
			PaddingBottom(1).
			MarginBottom(1)

	// Pipeline item styles
	itemStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			PaddingRight(2).
			MarginBottom(0)

	selectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				PaddingRight(2).
				MarginBottom(0).
				Foreground(accentColor).
				Bold(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderLeft(true).
				BorderForeground(primaryColor)

	// Status indicator styles
	statusSatisfiedStyle = lipgloss.NewStyle().
				Foreground(successColor).
				Bold(true)

	statusDriftedStyle = lipgloss.NewStyle().
				Foreground(warningColor).
				Bold(true)

	statusFailedStyle = lipgloss.NewStyle().
				Foreground(errorColor).
				Bold(true)

	statusUnknownStyle = lipgloss.NewStyle().
				Foreground(mutedColor)

	// Footer style
	footerStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(mutedColor).
			PaddingTop(1).
			MarginTop(1)

	// Error banner style
	errorBannerStyle = lipgloss.NewStyle().
				Foreground(errorColor).
				Background(lipgloss.Color("52")). // Dark red background
				Bold(true).
				Padding(1, 2).
				MarginBottom(1).
				BorderStyle(lipgloss.ThickBorder()).
				BorderForeground(errorColor)

	// Info banner style
	infoBannerStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Background(lipgloss.Color("237")). // Dark background
			Padding(1, 2).
			MarginBottom(1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(primaryColor)

	// Help overlay styles
	helpTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Align(lipgloss.Center).
			MarginBottom(1)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true).
			Width(12)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	helpBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(2, 4).
			Background(backgroundColor)

	// Confirm dialog styles
	confirmBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(warningColor).
			Padding(2, 4).
			Background(backgroundColor).
			Align(lipgloss.Center)

	confirmTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(warningColor).
				Align(lipgloss.Center).
				MarginBottom(1)

	confirmButtonStyle = lipgloss.NewStyle().
				Bold(true).
				Padding(0, 2).
				MarginLeft(1).
				MarginRight(1)

	confirmButtonYesStyle = confirmButtonStyle.
				Foreground(successColor).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(successColor)

	confirmButtonNoStyle = confirmButtonStyle.
				Foreground(errorColor).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(errorColor)

	// Detail view styles
	detailLabelStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Bold(true).
				Width(16)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	detailSectionStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(mutedColor).
				Padding(1, 2).
				MarginTop(1).
				MarginBottom(1)

	// Empty state style
	emptyStateStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true).
			Align(lipgloss.Center).
			PaddingTop(4).
			PaddingBottom(4)

	// Spinner style
	spinnerStyle = lipgloss.NewStyle().
			Foreground(primaryColor)

	// Progress indicator style
	progressStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)
)

// GetStatusStyle returns the appropriate style for a pipeline status
func GetStatusStyle(status string) lipgloss.Style {
	switch status {
	case "satisfied":
		return statusSatisfiedStyle
	case "drifted":
		return statusDriftedStyle
	case "failed":
		return statusFailedStyle
	default:
		return statusUnknownStyle
	}
}

// ApplyMaxWidth applies a maximum width to all relevant styles
func ApplyMaxWidth(width int) {
	itemStyle = itemStyle.MaxWidth(width - 4)
	selectedItemStyle = selectedItemStyle.MaxWidth(width - 4)
	headerStyle = headerStyle.Width(width - 2)
	footerStyle = footerStyle.Width(width - 2)
}
