package tui

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/engine"
	"github.com/alexisbeaulieu97/streamy/internal/model"
)

func TestViewRendersBasicLayout(t *testing.T) {
	plan := &engine.ExecutionPlan{Levels: []engine.ExecutionLevel{{StepIDs: []string{"step1", "step2"}}}}
	m := NewModel(&config.Config{Name: "Test Config"}, plan, false)
	m.steps["step1"] = model.StepResult{StepID: "step1", Status: model.StatusSuccess, Message: "done"}
	m.steps["step2"] = model.StepResult{StepID: "step2", Status: model.StatusRunning}
	m.completed = 1

	view := m.View()
	require.Contains(t, view, "Test Config")
	require.Contains(t, view, "step1")
	require.Contains(t, view, "step2")
	require.Contains(t, view, "done")
}

func TestViewShowsSummaryWhenFinished(t *testing.T) {
	m := NewModel(&config.Config{Name: "Finished"}, &engine.ExecutionPlan{}, false)
	m.finished = true
	m.completed = 3
	m.total = 4

	view := m.View()
	require.Contains(t, view, "Finished")
	require.Contains(t, view, "3/4")
}

func TestStatusIcon(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		status   string
		expected string
	}{
		{"success shows checkmark", model.StatusSuccess, "✓"},
		{"running shows hourglass", model.StatusRunning, "⏳"},
		{"failed shows cross", model.StatusFailed, "✗"},
		{"skipped shows circle-slash", model.StatusSkipped, "⊘"},
		{"pending shows ellipsis", model.StatusPending, "…"},
		{"unknown shows ellipsis", "unknown", "…"},
		{"empty shows ellipsis", "", "…"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			icon := StatusIcon(tt.status)
			require.Contains(t, icon, tt.expected)
		})
	}
}
