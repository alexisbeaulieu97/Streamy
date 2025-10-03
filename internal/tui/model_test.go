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

func TestNewModelInitialisesState(t *testing.T) {
	cfg := &config.Config{Name: "Test"}
	plan := &engine.ExecutionPlan{}
	m := NewModel(cfg, plan, false)

	require.Equal(t, cfg, m.cfg)
	require.Equal(t, plan, m.plan)
	require.False(t, m.finished)
	require.Zero(t, m.completed)
}

func TestModelInitReturnsTickCommand(t *testing.T) {
	m := NewModel(&config.Config{}, &engine.ExecutionPlan{}, false)
	cmd := m.Init()
	require.NotNil(t, cmd)
}

func TestModelTracksStepResults(t *testing.T) {
	cfg := &config.Config{}
	plan := &engine.ExecutionPlan{Levels: []engine.ExecutionLevel{{StepIDs: []string{"step1"}}}}
	m := NewModel(cfg, plan, false)

	updated, _ := m.Update(StepStartMsg{ID: "step1", Time: time.Now()})
	m = updated.(Model)
	require.Equal(t, model.StatusRunning, m.steps["step1"].Status)

	finished := StepCompleteMsg{Result: model.StepResult{StepID: "step1", Status: model.StatusSuccess}}
	updated, _ = m.Update(finished)
	m = updated.(Model)
	require.Equal(t, model.StatusSuccess, m.steps["step1"].Status)
	require.Equal(t, 1, m.completed)
}

func TestModelHandlesValidationResults(t *testing.T) {
	cfg := &config.Config{}
	plan := &engine.ExecutionPlan{}
	m := NewModel(cfg, plan, false)

	msg := ValidationMsg{Passed: true, Message: "ok"}
	updated, _ := m.Update(msg)
	m = updated.(Model)
	require.Len(t, m.validations, 1)
	require.True(t, m.validations[0].Passed)
}

func TestModelMarksFinished(t *testing.T) {
	cfg := &config.Config{}
	plan := &engine.ExecutionPlan{}
	m := NewModel(cfg, plan, false)

	updated, cmd := m.Update(tea.QuitMsg{})
	require.Nil(t, cmd)
	m = updated.(Model)
	require.True(t, m.finished)
}
