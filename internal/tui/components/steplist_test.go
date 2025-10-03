package components

import (
	"testing"

	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/stretchr/testify/require"
)

func TestNewStepList(t *testing.T) {
	t.Parallel()

	t.Run("creates empty step list", func(t *testing.T) {
		t.Parallel()
		sl := NewStepList([]string{}, map[string]model.StepResult{})
		require.Empty(t, sl.entries)
	})

	t.Run("creates step list with single step", func(t *testing.T) {
		t.Parallel()
		order := []string{"step1"}
		steps := map[string]model.StepResult{
			"step1": {Status: model.StatusPending},
		}

		sl := NewStepList(order, steps)
		require.Len(t, sl.entries, 1)
		require.Equal(t, "step1", sl.entries[0].ID)
		require.Equal(t, model.StatusPending, sl.entries[0].Result.Status)
	})

	t.Run("creates step list with multiple steps in order", func(t *testing.T) {
		t.Parallel()
		order := []string{"step1", "step2", "step3"}
		steps := map[string]model.StepResult{
			"step1": {Status: model.StatusSuccess},
			"step2": {Status: model.StatusRunning},
			"step3": {Status: model.StatusPending},
		}

		sl := NewStepList(order, steps)
		require.Len(t, sl.entries, 3)
		require.Equal(t, "step1", sl.entries[0].ID)
		require.Equal(t, model.StatusSuccess, sl.entries[0].Result.Status)
		require.Equal(t, "step2", sl.entries[1].ID)
		require.Equal(t, model.StatusRunning, sl.entries[1].Result.Status)
		require.Equal(t, "step3", sl.entries[2].ID)
		require.Equal(t, model.StatusPending, sl.entries[2].Result.Status)
	})

	t.Run("respects provided order", func(t *testing.T) {
		t.Parallel()
		order := []string{"step3", "step1", "step2"}
		steps := map[string]model.StepResult{
			"step1": {Status: model.StatusSuccess},
			"step2": {Status: model.StatusRunning},
			"step3": {Status: model.StatusPending},
		}

		sl := NewStepList(order, steps)
		require.Len(t, sl.entries, 3)
		require.Equal(t, "step3", sl.entries[0].ID)
		require.Equal(t, "step1", sl.entries[1].ID)
		require.Equal(t, "step2", sl.entries[2].ID)
	})

	t.Run("handles steps with various statuses", func(t *testing.T) {
		t.Parallel()
		order := []string{"pending", "running", "success", "failed", "skipped"}
		steps := map[string]model.StepResult{
			"pending": {Status: model.StatusPending},
			"running": {Status: model.StatusRunning},
			"success": {Status: model.StatusSuccess},
			"failed":  {Status: model.StatusFailed},
			"skipped": {Status: model.StatusSkipped},
		}

		sl := NewStepList(order, steps)
		require.Len(t, sl.entries, 5)
	})
}

func TestStepListEntries(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice for empty list", func(t *testing.T) {
		t.Parallel()
		sl := NewStepList([]string{}, map[string]model.StepResult{})
		entries := sl.Entries()
		require.Empty(t, entries)
	})

	t.Run("returns copy of entries", func(t *testing.T) {
		t.Parallel()
		order := []string{"step1", "step2"}
		steps := map[string]model.StepResult{
			"step1": {Status: model.StatusSuccess},
			"step2": {Status: model.StatusRunning},
		}

		sl := NewStepList(order, steps)
		entries := sl.Entries()
		require.Len(t, entries, 2)
		require.Equal(t, "step1", entries[0].ID)
		require.Equal(t, "step2", entries[1].ID)
	})

	t.Run("returns independent copy", func(t *testing.T) {
		t.Parallel()
		order := []string{"step1"}
		steps := map[string]model.StepResult{
			"step1": {Status: model.StatusSuccess},
		}

		sl := NewStepList(order, steps)
		entries1 := sl.Entries()
		entries2 := sl.Entries()

		// Modifying one should not affect the other
		entries1[0].ID = "modified"
		require.Equal(t, "step1", entries2[0].ID)
	})

	t.Run("preserves entry details", func(t *testing.T) {
		t.Parallel()
		order := []string{"step1"}
		steps := map[string]model.StepResult{
			"step1": {
				Status:  model.StatusSuccess,
				Message: "all done",
			},
		}

		sl := NewStepList(order, steps)
		entries := sl.Entries()
		require.Len(t, entries, 1)
		require.Equal(t, "step1", entries[0].ID)
		require.Equal(t, model.StatusSuccess, entries[0].Result.Status)
		require.Equal(t, "all done", entries[0].Result.Message)
	})
}
