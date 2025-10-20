// Package tui contains the shared TUI message model used across screens.
package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/tui/components"
)

// StepStartMsg indicates a step has started executing.
type StepStartMsg struct {
	ID   string
	Time time.Time
}

// StepCompleteMsg reports that a step has finished execution.
type StepCompleteMsg struct {
	StepID string
	State  components.StepState
}

// ValidationMsg carries the outcome of a validation.
type ValidationMsg struct {
	Passed  bool
	Message string
}

type tickMsg struct{}

// Model contains the Bubbletea state for Streamy's execution TUI.
type Model struct {
	pipeline       *pipeline.Pipeline
	plan           *pipeline.ExecutionPlan
	steps          map[string]components.StepState
	order          []string
	validations    []components.ValidationStatus
	total          int
	completed      int
	finished       bool
	cancelled      bool
	nonInteractive bool
}

// NewModel constructs a new TUI model for the given pipeline and plan.
func NewModel(pip *pipeline.Pipeline, plan *pipeline.ExecutionPlan, nonInteractive bool) Model {
	m := Model{
		pipeline:       pip,
		plan:           plan,
		steps:          make(map[string]components.StepState),
		order:          make([]string, 0),
		validations:    make([]components.ValidationStatus, 0),
		nonInteractive: nonInteractive,
	}

	if plan != nil {
		for _, level := range plan.Levels {
			for _, id := range level.StepIDs {
				if _, exists := m.steps[id]; !exists {
					m.steps[id] = components.StepState{Status: components.StepStatusPending}
					m.order = append(m.order, id)
					m.total++
				}
			}
		}
	}

	return m
}

// Init starts the Bubbletea program.
func (m Model) Init() tea.Cmd {
	return tea.Tick(time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
}

// TotalSteps returns the total number of steps tracked by the model.
func (m Model) TotalSteps() int {
	return m.total
}

// CompletedSteps returns the number of completed steps.
func (m Model) CompletedSteps() int {
	return m.completed
}

// IsFinished reports whether execution has completed.
func (m Model) IsFinished() bool {
	return m.finished
}

func (m *Model) ensureStep(id string) {
	if id == "" {
		return
	}

	if _, exists := m.steps[id]; !exists {
		m.steps[id] = components.StepState{Status: components.StepStatusPending}
		m.order = append(m.order, id)
		m.total++
	}
}

func (m *Model) markFinishedIfComplete() {
	if m.total > 0 && m.completed >= m.total {
		m.finished = true
	}
}
