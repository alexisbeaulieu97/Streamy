package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/tui/components"
)

func TestNewModelInitialisesState(t *testing.T) {
	pip := &pipeline.Pipeline{Name: "Test"}
	plan := &pipeline.ExecutionPlan{}
	m := NewModel(pip, plan, false)

	require.Equal(t, pip, m.pipeline)
	require.Equal(t, plan, m.plan)
	require.False(t, m.finished)
	require.Zero(t, m.completed)
}

func TestModelInitReturnsTickCommand(t *testing.T) {
	m := NewModel(&pipeline.Pipeline{}, &pipeline.ExecutionPlan{}, false)
	cmd := m.Init()
	require.NotNil(t, cmd)

	// Execute the tick command to test it actually works
	msg := cmd()
	require.NotNil(t, msg)
}

func TestModelTracksStepResults(t *testing.T) {
	pip := &pipeline.Pipeline{}
	plan := &pipeline.ExecutionPlan{Levels: []pipeline.ExecutionLevel{{StepIDs: []string{"step1"}}}}
	m := NewModel(pip, plan, false)

	updated, _ := m.Update(StepStartMsg{ID: "step1", Time: time.Now()})
	m = updated.(Model)
	require.Equal(t, components.StepStatusRunning, m.steps["step1"].Status)

	finished := StepCompleteMsg{StepID: "step1", State: components.StepState{Status: components.StepStatusSuccess}}
	updated, _ = m.Update(finished)
	m = updated.(Model)
	require.Equal(t, components.StepStatusSuccess, m.steps["step1"].Status)
	require.Equal(t, 1, m.completed)
}

func TestModelHandlesValidationResults(t *testing.T) {
	pip := &pipeline.Pipeline{}
	plan := &pipeline.ExecutionPlan{}
	m := NewModel(pip, plan, false)

	msg := ValidationMsg{Passed: true, Message: "ok"}
	updated, _ := m.Update(msg)
	m = updated.(Model)
	require.Len(t, m.validations, 1)
	require.True(t, m.validations[0].Passed)
}

func TestModelMarksFinished(t *testing.T) {
	pip := &pipeline.Pipeline{}
	plan := &pipeline.ExecutionPlan{}
	m := NewModel(pip, plan, false)

	updated, cmd := m.Update(tea.QuitMsg{})
	require.Nil(t, cmd)
	m = updated.(Model)
	require.True(t, m.finished)
}

func TestModelTotalSteps(t *testing.T) {
	t.Parallel()

	t.Run("returns zero for empty model", func(t *testing.T) {
		t.Parallel()
		m := NewModel(&pipeline.Pipeline{}, &pipeline.ExecutionPlan{}, false)
		require.Equal(t, 0, m.TotalSteps())
	})

	t.Run("returns total after processing steps", func(t *testing.T) {
		t.Parallel()
		pip := &pipeline.Pipeline{}
		plan := &pipeline.ExecutionPlan{Levels: []pipeline.ExecutionLevel{{StepIDs: []string{"step1", "step2"}}}}
		m := NewModel(pip, plan, false)

		updated, _ := m.Update(StepStartMsg{ID: "step1", Time: time.Now()})
		m = updated.(Model)
		updated, _ = m.Update(StepStartMsg{ID: "step2", Time: time.Now()})
		m = updated.(Model)

		require.Equal(t, 2, m.TotalSteps())
	})
}

func TestModelCompletedSteps(t *testing.T) {
	t.Parallel()

	t.Run("returns zero initially", func(t *testing.T) {
		t.Parallel()
		m := NewModel(&pipeline.Pipeline{}, &pipeline.ExecutionPlan{}, false)
		require.Equal(t, 0, m.CompletedSteps())
	})

	t.Run("increments after completing steps", func(t *testing.T) {
		t.Parallel()
		pip := &pipeline.Pipeline{}
		plan := &pipeline.ExecutionPlan{Levels: []pipeline.ExecutionLevel{{StepIDs: []string{"step1", "step2"}}}}
		m := NewModel(pip, plan, false)

		updated, _ := m.Update(StepStartMsg{ID: "step1", Time: time.Now()})
		m = updated.(Model)
		require.Equal(t, 0, m.CompletedSteps())

		finished := StepCompleteMsg{StepID: "step1", State: components.StepState{Status: components.StepStatusSuccess}}
		updated, _ = m.Update(finished)
		m = updated.(Model)
		require.Equal(t, 1, m.CompletedSteps())

		updated, _ = m.Update(StepStartMsg{ID: "step2", Time: time.Now()})
		m = updated.(Model)
		finished = StepCompleteMsg{StepID: "step2", State: components.StepState{Status: components.StepStatusSuccess}}
		updated, _ = m.Update(finished)
		m = updated.(Model)
		require.Equal(t, 2, m.CompletedSteps())
	})
}

func TestModelIsFinished(t *testing.T) {
	t.Parallel()

	t.Run("returns false initially", func(t *testing.T) {
		t.Parallel()
		m := NewModel(&pipeline.Pipeline{}, &pipeline.ExecutionPlan{}, false)
		require.False(t, m.IsFinished())
	})

	t.Run("returns true after quit", func(t *testing.T) {
		t.Parallel()
		m := NewModel(&pipeline.Pipeline{}, &pipeline.ExecutionPlan{}, false)
		updated, _ := m.Update(tea.QuitMsg{})
		m = updated.(Model)
		require.True(t, m.IsFinished())
	})
}

func TestModelEnsureStep(t *testing.T) {
	t.Parallel()

	t.Run("adds new step", func(t *testing.T) {
		t.Parallel()
		m := NewModel(&pipeline.Pipeline{}, &pipeline.ExecutionPlan{}, false)
		m.ensureStep("new_step")

		require.Contains(t, m.steps, "new_step")
		require.Equal(t, components.StepStatusPending, m.steps["new_step"].Status)
		require.Equal(t, 1, m.total)
		require.Contains(t, m.order, "new_step")
	})

	t.Run("does not add duplicate step", func(t *testing.T) {
		t.Parallel()
		m := NewModel(&pipeline.Pipeline{}, &pipeline.ExecutionPlan{}, false)
		m.ensureStep("step1")
		m.ensureStep("step1")

		require.Len(t, m.steps, 1)
		require.Equal(t, 1, m.total)
		require.Len(t, m.order, 1)
	})

	t.Run("ignores empty step ID", func(t *testing.T) {
		t.Parallel()
		m := NewModel(&pipeline.Pipeline{}, &pipeline.ExecutionPlan{}, false)
		m.ensureStep("")

		require.Empty(t, m.steps)
		require.Equal(t, 0, m.total)
		require.Empty(t, m.order)
	})

	t.Run("maintains order of multiple steps", func(t *testing.T) {
		t.Parallel()
		m := NewModel(&pipeline.Pipeline{}, &pipeline.ExecutionPlan{}, false)
		m.ensureStep("step1")
		m.ensureStep("step2")
		m.ensureStep("step3")

		require.Equal(t, []string{"step1", "step2", "step3"}, m.order)
		require.Equal(t, 3, m.total)
	})
}
