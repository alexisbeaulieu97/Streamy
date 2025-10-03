package components

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewSummary(t *testing.T) {
	t.Parallel()

	t.Run("creates summary with data", func(t *testing.T) {
		t.Parallel()
		data := SummaryData{
			Total:     10,
			Completed: 5,
			Finished:  false,
		}
		summary := NewSummary(data)
		require.Equal(t, data, summary.data)
	})
}

func TestSummaryView(t *testing.T) {
	t.Parallel()

	t.Run("renders empty summary", func(t *testing.T) {
		t.Parallel()
		data := SummaryData{
			Total:     0,
			Completed: 0,
			Finished:  false,
		}
		summary := NewSummary(data)
		view := summary.View()
		require.Equal(t, "", view)
	})

	t.Run("renders steps progress", func(t *testing.T) {
		t.Parallel()
		data := SummaryData{
			Total:     10,
			Completed: 5,
			Finished:  false,
		}
		summary := NewSummary(data)
		view := summary.View()
		require.Contains(t, view, "Steps: 5/10 completed")
	})

	t.Run("renders successful completion", func(t *testing.T) {
		t.Parallel()
		data := SummaryData{
			Total:     10,
			Completed: 10,
			Finished:  true,
		}
		summary := NewSummary(data)
		view := summary.View()
		require.Contains(t, view, "Steps: 10/10 completed")
		require.Contains(t, view, "Execution finished successfully")
	})

	t.Run("renders partial completion when finished", func(t *testing.T) {
		t.Parallel()
		data := SummaryData{
			Total:     10,
			Completed: 7,
			Finished:  true,
		}
		summary := NewSummary(data)
		view := summary.View()
		require.Contains(t, view, "Steps: 7/10 completed")
		require.Contains(t, view, "Execution finished with pending steps")
	})

	t.Run("renders cancelled execution", func(t *testing.T) {
		t.Parallel()
		data := SummaryData{
			Total:     10,
			Completed: 3,
			Finished:  false,
			Cancelled: true,
		}
		summary := NewSummary(data)
		view := summary.View()
		require.Contains(t, view, "Execution cancelled")
	})

	t.Run("renders passing validations", func(t *testing.T) {
		t.Parallel()
		data := SummaryData{
			Total:     5,
			Completed: 5,
			Finished:  true,
			Validations: []ValidationStatus{
				{Passed: true, Message: "git is installed"},
				{Passed: true, Message: "config file exists"},
			},
		}
		summary := NewSummary(data)
		view := summary.View()
		require.Contains(t, view, "Validations:")
		require.Contains(t, view, "✓ git is installed")
		require.Contains(t, view, "✓ config file exists")
	})

	t.Run("renders failing validations", func(t *testing.T) {
		t.Parallel()
		data := SummaryData{
			Total:     5,
			Completed: 5,
			Finished:  true,
			Validations: []ValidationStatus{
				{Passed: true, Message: "git is installed"},
				{Passed: false, Message: "docker not found"},
			},
		}
		summary := NewSummary(data)
		view := summary.View()
		require.Contains(t, view, "Validations:")
		require.Contains(t, view, "✓ git is installed")
		require.Contains(t, view, "✗ docker not found")
	})

	t.Run("renders mixed validations", func(t *testing.T) {
		t.Parallel()
		data := SummaryData{
			Total:     5,
			Completed: 5,
			Finished:  true,
			Validations: []ValidationStatus{
				{Passed: true, Message: "check 1"},
				{Passed: false, Message: "check 2"},
				{Passed: true, Message: "check 3"},
			},
		}
		summary := NewSummary(data)
		view := summary.View()
		lines := strings.Split(view, "\n")
		require.True(t, len(lines) >= 5) // Header + 3 validations + footer
	})

	t.Run("renders validations without steps", func(t *testing.T) {
		t.Parallel()
		data := SummaryData{
			Total:     0,
			Completed: 0,
			Finished:  false,
			Validations: []ValidationStatus{
				{Passed: true, Message: "pre-check passed"},
			},
		}
		summary := NewSummary(data)
		view := summary.View()
		require.Contains(t, view, "Validations:")
		require.Contains(t, view, "✓ pre-check passed")
	})

	t.Run("renders empty validations list", func(t *testing.T) {
		t.Parallel()
		data := SummaryData{
			Total:       5,
			Completed:   5,
			Finished:    true,
			Validations: []ValidationStatus{},
		}
		summary := NewSummary(data)
		view := summary.View()
		require.NotContains(t, view, "Validations:")
	})

	t.Run("multiline output format", func(t *testing.T) {
		t.Parallel()
		data := SummaryData{
			Total:     10,
			Completed: 10,
			Finished:  true,
			Validations: []ValidationStatus{
				{Passed: true, Message: "validation 1"},
			},
		}
		summary := NewSummary(data)
		view := summary.View()
		lines := strings.Split(view, "\n")
		require.True(t, len(lines) >= 3) // Steps + Finished + Validations header + validation
	})
}

func TestSummaryViewEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("cancelled execution shows before finished message", func(t *testing.T) {
		t.Parallel()
		data := SummaryData{
			Total:     10,
			Completed: 5,
			Finished:  true,
			Cancelled: true,
		}
		summary := NewSummary(data)
		view := summary.View()
		require.Contains(t, view, "Execution cancelled")
		// Should show cancelled, not finished message
		require.NotContains(t, view, "finished successfully")
		require.NotContains(t, view, "finished with pending steps")
	})

	t.Run("zero completed with finished flag", func(t *testing.T) {
		t.Parallel()
		data := SummaryData{
			Total:     5,
			Completed: 0,
			Finished:  true,
		}
		summary := NewSummary(data)
		view := summary.View()
		require.Contains(t, view, "Steps: 0/5 completed")
		require.Contains(t, view, "Execution finished with pending steps")
	})
}
