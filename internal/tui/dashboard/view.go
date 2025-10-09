package dashboard

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

// View renders the current model state
func (m Model) View() string {
	switch m.viewMode {
	case ViewList:
		return m.renderListView()
	case ViewDetail:
		return m.renderDetailView()
	case ViewHelp:
		return m.renderHelpView()
	case ViewConfirm:
		return m.renderConfirmView()
	default:
		return m.renderListView()
	}
}

// renderListView renders the main pipeline list view
func (m Model) renderListView() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	var content strings.Builder

	// Render header
	content.WriteString(m.renderHeader())
	content.WriteString("\n")

	// Render error banner if present
	if m.showError {
		content.WriteString(m.renderErrorBanner())
		content.WriteString("\n")
	}

	// Render pipeline list
	content.WriteString(m.renderPipelineList())
	content.WriteString("\n")

	// Render footer
	content.WriteString(m.renderFooter())

	return content.String()
}

// renderHeader renders the header with title and status summary
func (m Model) renderHeader() string {
	title := titleStyle.Render("🚀 Streamy Dashboard")
	
	counts := m.CountByStatus()
	summary := fmt.Sprintf(
		"%s %d  %s %d  %s %d  %s %d",
		registry.StatusSatisfied.Icon(), counts[registry.StatusSatisfied],
		registry.StatusDrifted.Icon(), counts[registry.StatusDrifted],
		registry.StatusFailed.Icon(), counts[registry.StatusFailed],
		registry.StatusUnknown.Icon(), counts[registry.StatusUnknown],
	)

	// Add refresh indicator if refreshing
	if m.refreshing {
		summary += fmt.Sprintf("  %s Refreshing %d/%d",
			m.spinner.View(),
			m.refreshProgress,
			m.refreshTotal,
		)
	}

	headerContent := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		summary,
	)

	return headerStyle.Render(headerContent)
}

// renderPipelineList renders the list of pipelines
func (m Model) renderPipelineList() string {
	if len(m.pipelines) == 0 {
		return m.renderEmptyState()
	}

	var items []string
	visibleHeight := m.height - 10 // Reserve space for header and footer

	// Calculate scroll window
	start := m.scrollOffset
	end := start + visibleHeight
	if end > len(m.pipelines) {
		end = len(m.pipelines)
	}

	for i := start; i < end; i++ {
		items = append(items, m.renderPipelineItem(i, i == m.cursor))
	}

	// Add scroll indicators if needed
	if start > 0 {
		items = append([]string{lipgloss.NewStyle().Foreground(mutedColor).Render("▲ More above")}, items...)
	}
	if end < len(m.pipelines) {
		items = append(items, lipgloss.NewStyle().Foreground(mutedColor).Render("▼ More below"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, items...)
}

// renderPipelineItem renders a single pipeline item
func (m Model) renderPipelineItem(index int, selected bool) string {
	p := m.pipelines[index]

	// Status icon
	icon := p.Status.Icon()
	if !m.useUnicode {
		icon = p.Status.IconFallback()
	}

	// Add spinner if loading
	if m.IsLoading(p.ID) {
		icon = m.spinner.View()
	}

	// Status with color
	statusStr := GetStatusStyle(p.Status.String()).Render(icon)

	// Pipeline number (1-indexed for display)
	number := fmt.Sprintf("%d.", index+1)

	// Name
	name := p.Name
	if name == "" {
		name = p.ID
	}

	// Description (truncated if too long)
	desc := p.Description
	if len(desc) > 60 {
		desc = desc[:57] + "..."
	}
	if desc == "" {
		desc = lipgloss.NewStyle().Foreground(mutedColor).Render("No description")
	}

	// Last run time
	lastRun := FormatLastRun(p.LastRun)

	// Compose the item
	line1 := fmt.Sprintf("%s %s %s", statusStr, number, lipgloss.NewStyle().Bold(true).Render(name))
	line2 := fmt.Sprintf("   %s", desc)
	line3 := fmt.Sprintf("   %s", lipgloss.NewStyle().Foreground(mutedColor).Render("Last checked: "+lastRun))

	content := lipgloss.JoinVertical(lipgloss.Left, line1, line2, line3)

	// Apply selected style if this item is selected
	if selected {
		return selectedItemStyle.Render(content)
	}
	return itemStyle.Render(content)
}

// renderEmptyState renders the empty state when no pipelines are registered
func (m Model) renderEmptyState() string {
	message := `No pipelines registered yet.

To add a pipeline, use:
  streamy register <name> <config-file>`

	return emptyStateStyle.Render(message)
}

// renderFooter renders the footer with keyboard shortcuts
func (m Model) renderFooter() string {
	hints := []string{
		"↑/↓: navigate",
		"enter: select",
		"r: refresh",
		"?: help",
	}
	
	// Add error dismissal hint if error is showing
	if m.showError {
		hints = append(hints, "x: dismiss error")
	}
	
	hints = append(hints, "q: quit")

	return footerStyle.Render(strings.Join(hints, "  •  "))
}

// renderErrorBanner renders an error message banner
func (m Model) renderErrorBanner() string {
	return errorBannerStyle.Render(m.errorMsg)
}

// FormatLastRun formats a timestamp to a human-readable relative time
func FormatLastRun(t time.Time) string {
	if t.IsZero() {
		return "Never"
	}

	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "Just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("Jan 2, 2006")
	}
}

// Placeholder implementations for other views (to be implemented in later phases)

// renderDetailView renders the detail view for a selected pipeline
func (m Model) renderDetailView() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	// Get the selected pipeline
	var selected *registry.Pipeline
	for i := range m.pipelines {
		if m.pipelines[i].ID == m.selectedID {
			selected = &m.pipelines[i]
			break
		}
	}

	if selected == nil {
		return "Pipeline not found"
	}

	var content strings.Builder

	// Header with pipeline name
	header := titleStyle.Render(fmt.Sprintf("📋 %s", selected.Name))
	content.WriteString(header)
	content.WriteString("\n\n")

	// Render error banner if present
	if m.showError {
		content.WriteString(m.renderErrorBanner())
		content.WriteString("\n\n")
	}

	// Status section
	statusIcon := selected.Status.Icon()
	if !m.useUnicode {
		statusIcon = selected.Status.IconFallback()
	}
	statusLine := fmt.Sprintf("%s Status: %s",
		GetStatusStyle(selected.Status.String()).Render(statusIcon),
		lipgloss.NewStyle().Bold(true).Render(selected.Status.String()))
	content.WriteString(statusLine)
	content.WriteString("\n\n")

	// Metadata section
	metaStyle := lipgloss.NewStyle().Foreground(mutedColor)
	content.WriteString(lipgloss.NewStyle().Bold(true).Render("Metadata"))
	content.WriteString("\n")
	content.WriteString(fmt.Sprintf("  ID: %s\n", selected.ID))
	content.WriteString(fmt.Sprintf("  Path: %s\n", selected.Path))
	content.WriteString(fmt.Sprintf("  Registered: %s\n", selected.RegisteredAt.Format("Jan 2, 2006 15:04")))
	if !selected.LastRun.IsZero() {
		content.WriteString(fmt.Sprintf("  Last Run: %s\n", FormatLastRun(selected.LastRun)))
	}
	content.WriteString("\n")

	// Description section
	if selected.Description != "" {
		content.WriteString(lipgloss.NewStyle().Bold(true).Render("Description"))
		content.WriteString("\n")
		content.WriteString(fmt.Sprintf("  %s\n", selected.Description))
		content.WriteString("\n")
	}

	// Last execution result section
	if selected.LastResult != nil {
		content.WriteString(lipgloss.NewStyle().Bold(true).Render("Last Execution"))
		content.WriteString("\n")
		content.WriteString(fmt.Sprintf("  Operation: %s\n", selected.LastResult.Operation))
		content.WriteString(fmt.Sprintf("  Completed: %s\n", selected.LastResult.CompletedAt.Format("Jan 2, 2006 15:04:05")))
		content.WriteString(fmt.Sprintf("  Duration: %s\n", selected.LastResult.Duration.Round(time.Millisecond)))
		content.WriteString(fmt.Sprintf("  Steps: %d total", len(selected.LastResult.StepResults)))
		
		// Count step statuses
		successCount := 0
		failedCount := 0
		for _, step := range selected.LastResult.StepResults {
			if step.Status == "success" {
				successCount++
			} else if step.Status == "failed" {
				failedCount++
			}
		}
		content.WriteString(fmt.Sprintf(" (%d success, %d failed)\n", successCount, failedCount))
		
		// Show error if present
		if selected.LastResult.Error != nil {
			content.WriteString("\n")
			content.WriteString(lipgloss.NewStyle().Bold(true).Foreground(errorColor).Render("Error"))
			content.WriteString("\n")
			content.WriteString(fmt.Sprintf("  %s\n", selected.LastResult.Error.Message))
			if selected.LastResult.Error.Suggestion != "" {
				content.WriteString(fmt.Sprintf("  Suggestion: %s\n", selected.LastResult.Error.Suggestion))
			}
		}
		content.WriteString("\n")
	}

	// Show loading indicator if operation in progress
	if m.IsLoading(selected.ID) {
		op, ok := m.operations[selected.ID]
		if ok {
			content.WriteString("\n")
			opMsg := fmt.Sprintf("%s %s in progress...", m.spinner.View(), op.Type)
			content.WriteString(lipgloss.NewStyle().Foreground(primaryColor).Render(opMsg))
			content.WriteString("\n")
		}
	}

	// Footer with actions
	hints := []string{
		"v: verify",
		"a: apply",
		"r: refresh",
		"esc: back",
		"?: help",
		"q: quit",
	}
	footer := footerStyle.Render(strings.Join(hints, "  •  "))
	
	// Calculate available height for content
	contentHeight := m.height - 4 // Reserve space for footer
	lines := strings.Split(content.String(), "\n")
	
	// Truncate if too many lines
	if len(lines) > contentHeight {
		lines = lines[:contentHeight]
		content.Reset()
		content.WriteString(strings.Join(lines, "\n"))
		content.WriteString("\n")
		content.WriteString(metaStyle.Render("... (content truncated)"))
		content.WriteString("\n")
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		content.String(),
		"",
		footer,
	)
}

// renderHelpView renders the help overlay
func (m Model) renderHelpView() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	title := titleStyle.Render("❓ Streamy Dashboard Help")
	
	helpContent := `
List View:
  ↑/↓, j/k      Navigate up/down
  1-9           Jump to pipeline by number
  Enter         View pipeline details
  r             Refresh all pipelines
  ?             Toggle this help
  q, Ctrl+C     Quit application

Detail View:
  v             Run verification
  a             Apply configuration (with confirmation)
  r             Refresh this pipeline
  Esc           Back to list
  ?             Toggle this help
  q, Ctrl+C     Quit application

Status Indicators:
  🟢 Satisfied   All steps are in desired state
  🟡 Drifted     Some steps need changes
  🔴 Failed      Verification failed or errors occurred
  ⚪ Unknown     Status not yet checked

Tips:
  • Pipeline status is cached between sessions
  • Failed/drifted pipelines are sorted to the top
  • Use Ctrl+C at any time to safely exit
  • Refresh updates status from actual system state
`

	helpText := lipgloss.NewStyle().
		Padding(1, 2).
		Render(helpContent)

	footer := footerStyle.Render("Press ? or Esc to close")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		helpText,
		footer,
	)
}

// renderConfirmView renders a confirmation dialog
func (m Model) renderConfirmView() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	// Render the background (dimmed list view) - could be used for overlay effect
	// For now, we just show the dialog without background
	
	// Build confirmation message
	var message string
	switch m.confirmAction {
	case "cancel_verify":
		message = "⚠️  Cancel verification?"
	case "cancel_apply":
		message = "⚠️  Cancel apply operation?"
	case "apply":
		message = "⚠️  Apply configuration changes?\n\nThis will modify your system."
	default:
		message = "Confirm action?"
	}

	// Style the dialog
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(warningColor).
		Padding(1, 2).
		Width(50).
		Align(lipgloss.Center)

	dialog := dialogStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Center,
			message,
			"",
			lipgloss.NewStyle().Foreground(mutedColor).Render("y = Yes    n = No    Esc = Cancel"),
		),
	)

	// Center the dialog
	centerStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center)

	return centerStyle.Render(dialog)
}
