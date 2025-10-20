package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/alexisbeaulieu97/streamy/internal/tui/components"
)

// Update handles Bubbletea messages and updates model state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		return m, nil
	case StepStartMsg:
		m.ensureStep(msg.ID)
		step := m.steps[msg.ID]
		step.Status = components.StepStatusRunning
		m.steps[msg.ID] = step

		return m, nil
	case StepCompleteMsg:
		if msg.StepID == "" {
			return m, nil
		}

		m.ensureStep(msg.StepID)
		existing := m.steps[msg.StepID]
		previouslyCompleted := existing.Status == components.StepStatusSuccess || existing.Status == components.StepStatusSkipped || existing.Status == components.StepStatusFailed || existing.Status == components.StepStatusWouldCreate || existing.Status == components.StepStatusWouldUpdate

		m.steps[msg.StepID] = msg.State
		if !previouslyCompleted {
			m.completed++
			m.markFinishedIfComplete()
		}

		if msg.State.Status == components.StepStatusFailed {
			m.finished = true
		}

		return m, nil
	case ValidationMsg:
		m.validations = append(m.validations, components.ValidationStatus{Passed: msg.Passed, Message: msg.Message})
		return m, nil
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			m.cancelled = true
			m.finished = true

			return m, nil
		}
	case tea.QuitMsg:
		m.finished = true
		return m, nil
	}

	return m, nil
}
