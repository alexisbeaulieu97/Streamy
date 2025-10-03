package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	sectionStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).MarginTop(1)

	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	runningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
	failureStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	skippedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	pendingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	summaryStyle = lipgloss.NewStyle().MarginTop(1)
)
