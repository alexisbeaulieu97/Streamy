package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/alexisbeaulieu97/streamy/internal/model"
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
		step.Status = model.StatusRunning
		m.steps[msg.ID] = step
		return m, nil
	case StepCompleteMsg:
		id := msg.Result.StepID
		if id == "" {
			return m, nil
		}
		m.ensureStep(id)
		existing := m.steps[id]
		previouslyCompleted := existing.Status == model.StatusSuccess || existing.Status == model.StatusSkipped || existing.Status == model.StatusFailed
		m.steps[id] = msg.Result
		if !previouslyCompleted {
			m.completed++
			m.markFinishedIfComplete()
		}
		if msg.Result.Status == model.StatusFailed {
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
