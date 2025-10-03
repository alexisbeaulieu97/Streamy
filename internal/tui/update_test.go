package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/engine"
	"github.com/alexisbeaulieu97/streamy/internal/model"
)

func TestUpdateHandlesStepStart(t *testing.T) {
	m := NewModel(&config.Config{}, &engine.ExecutionPlan{Levels: []engine.ExecutionLevel{{StepIDs: []string{"step"}}}}, false)
	updated, _ := m.Update(StepStartMsg{ID: "step", Time: time.Now()})
	m = updated.(Model)
	require.Equal(t, model.StatusRunning, m.steps["step"].Status)
}

func TestUpdateHandlesStepCompletion(t *testing.T) {
	m := NewModel(&config.Config{}, &engine.ExecutionPlan{Levels: []engine.ExecutionLevel{{StepIDs: []string{"step"}}}}, false)
	res := model.StepResult{StepID: "step", Status: model.StatusSuccess}
	updated, _ := m.Update(StepCompleteMsg{Result: res})
	m = updated.(Model)
	require.Equal(t, res.Status, m.steps["step"].Status)
	require.Equal(t, 1, m.completed)
}

func TestUpdateHandlesValidationMessages(t *testing.T) {
	m := NewModel(&config.Config{}, &engine.ExecutionPlan{}, false)
	msg := ValidationMsg{Passed: false, Message: "missing path"}
	updated, _ := m.Update(msg)
	m = updated.(Model)
	require.Len(t, m.validations, 1)
	require.False(t, m.validations[0].Passed)
}

func TestUpdateHandlesTeaMessages(t *testing.T) {
	m := NewModel(&config.Config{}, &engine.ExecutionPlan{}, false)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	require.Nil(t, cmd)
	m = updated.(Model)
	require.True(t, m.cancelled)
}
