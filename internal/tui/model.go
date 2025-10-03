package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/engine"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/tui/components"
)

// StepStartMsg indicates a step has started executing.
type StepStartMsg struct {
	ID   string
	Time time.Time
}

// StepCompleteMsg reports that a step has finished execution.
type StepCompleteMsg struct {
	Result model.StepResult
}

// ValidationMsg carries the outcome of a validation.
type ValidationMsg struct {
	Passed  bool
	Message string
}

type tickMsg struct{}

// Model contains the Bubbletea state for Streamy's execution TUI.
type Model struct {
	cfg            *config.Config
	plan           *engine.ExecutionPlan
	steps          map[string]model.StepResult
	order          []string
	validations    []components.ValidationStatus
	total          int
	completed      int
	finished       bool
	cancelled      bool
	nonInteractive bool
}

// NewModel constructs a new TUI model for the given configuration and plan.
func NewModel(cfg *config.Config, plan *engine.ExecutionPlan, nonInteractive bool) Model {
	m := Model{
		cfg:            cfg,
		plan:           plan,
		steps:          make(map[string]model.StepResult),
		order:          make([]string, 0),
		validations:    make([]components.ValidationStatus, 0),
		nonInteractive: nonInteractive,
	}

	if plan != nil {
		for _, level := range plan.Levels {
			for _, id := range level.StepIDs {
				if _, exists := m.steps[id]; !exists {
					m.steps[id] = model.StepResult{StepID: id, Status: model.StatusPending}
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
		m.steps[id] = model.StepResult{StepID: id, Status: model.StatusPending}
		m.order = append(m.order, id)
		m.total++
	}
}

func (m *Model) markFinishedIfComplete() {
	if m.total > 0 && m.completed >= m.total {
		m.finished = true
	}
}
