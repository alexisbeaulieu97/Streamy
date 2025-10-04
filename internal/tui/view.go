package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/tui/components"
)

// View renders the current state of the model.
func (m Model) View() string {
	var sections []string

	title := titleStyle.Render(fmt.Sprintf("Streamy • %s", m.title()))
	sections = append(sections, title)

	progress := components.NewProgress(m.total).View(m.completed)
	sections = append(sections, sectionStyle.Render("Progress"), progress)

	listComp := components.NewStepList(m.order, m.steps)
	entries := listComp.Entries()
	if len(entries) > 0 {
		sections = append(sections, sectionStyle.Render("Steps"))
		sections = append(sections, renderStepEntries(entries))
	}

	summary := components.NewSummary(components.SummaryData{
		Total:       m.total,
		Completed:   m.completed,
		Finished:    m.finished,
		Cancelled:   m.cancelled,
		Validations: m.validations,
	}).View()
	if strings.TrimSpace(summary) != "" {
		sections = append(sections, sectionStyle.Render("Summary"), summaryStyle.Render(summary))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func renderStepEntries(entries []components.StepEntry) string {
	var lines []string
	for _, entry := range entries {
		res := entry.Result
		icon := StatusIcon(res.Status)
		line := fmt.Sprintf(" %s %s", icon, entry.ID)
		if strings.TrimSpace(res.Message) != "" {
			line = fmt.Sprintf("%s — %s", line, res.Message)
		}
		if res.Duration > 0 {
			line = fmt.Sprintf("%s (%s)", line, res.Duration.Truncate(10*time.Millisecond))
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (m Model) title() string {
	if m.cfg != nil && strings.TrimSpace(m.cfg.Name) != "" {
		return m.cfg.Name
	}
	return "Execution"
}

// StatusIcon returns the glyph representing a step status.
func StatusIcon(status string) string {
	switch status {
	case model.StatusSuccess:
		return successStyle.Render("✓")
	case model.StatusRunning:
		return runningStyle.Render("⏳")
	case model.StatusFailed:
		return failureStyle.Render("✗")
	case model.StatusSkipped:
		return skippedStyle.Render("⊘")
	case model.StatusWouldCreate:
		return pendingStyle.Render("✱")
	case model.StatusWouldUpdate:
		return pendingStyle.Render("↻")
	default:
		return pendingStyle.Render("…")
	}
}
