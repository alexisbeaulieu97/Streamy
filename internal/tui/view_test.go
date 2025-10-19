package tui

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/tui/components"
)

func TestViewRendersBasicLayout(t *testing.T) {
	plan := &pipeline.ExecutionPlan{Levels: []pipeline.ExecutionLevel{{StepIDs: []string{"step1", "step2"}}}}
	m := NewModel(&pipeline.Pipeline{Name: "Test Config"}, plan, false)
	m.steps["step1"] = components.StepState{Status: components.StepStatusSuccess, Message: "done"}
	m.steps["step2"] = components.StepState{Status: components.StepStatusRunning}
	m.completed = 1

	view := m.View()
	require.Contains(t, view, "Test Config")
	require.Contains(t, view, "step1")
	require.Contains(t, view, "step2")
	require.Contains(t, view, "done")
}

func TestViewShowsSummaryWhenFinished(t *testing.T) {
	m := NewModel(&pipeline.Pipeline{Name: "Finished"}, &pipeline.ExecutionPlan{}, false)
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
		status   components.StepStatus
		expected string
	}{
		{"success shows checkmark", components.StepStatusSuccess, "✓"},
		{"running shows hourglass", components.StepStatusRunning, "⏳"},
		{"failed shows cross", components.StepStatusFailed, "✗"},
		{"skipped shows circle-slash", components.StepStatusSkipped, "⊘"},
		{"would create shows star", components.StepStatusWouldCreate, "✱"},
		{"would update shows refresh", components.StepStatusWouldUpdate, "↻"},
		{"unknown shows ellipsis", components.StepStatus("unknown"), "…"},
		{"empty shows ellipsis", components.StepStatus(""), "…"},
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
