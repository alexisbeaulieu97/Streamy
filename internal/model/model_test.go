package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type TestError struct {
	msg string
}

func (e *TestError) Error() string {
	return e.msg
}

func TestStepResultCreation(t *testing.T) {
	t.Parallel()

	t.Run("creates step result with all fields", func(t *testing.T) {
		t.Parallel()
		now := time.Now()
		result := StepResult{
			StepID:    "test_step",
			Status:    StatusSuccess,
			Message:   "completed",
			Duration:  time.Second,
			Timestamp: now,
		}

		require.Equal(t, "test_step", result.StepID)
		require.Equal(t, StatusSuccess, result.Status)
		require.Equal(t, "completed", result.Message)
		require.Equal(t, time.Second, result.Duration)
		require.Equal(t, now, result.Timestamp)
	})

	t.Run("creates step result with error", func(t *testing.T) {
		t.Parallel()
		err := &TestError{msg: "test error"}
		result := StepResult{
			StepID: "failed_step",
			Status: StatusFailed,
			Error:  err,
		}

		require.Equal(t, "failed_step", result.StepID)
		require.Equal(t, StatusFailed, result.Status)
		require.Equal(t, err, result.Error)
	})
}

func TestStatusConstants(t *testing.T) {
	t.Parallel()

	// Verify status constants are set correctly
	require.Equal(t, "pending", StatusPending)
	require.Equal(t, "running", StatusRunning)
	require.Equal(t, "success", StatusSuccess)
	require.Equal(t, "skipped", StatusSkipped)
	require.Equal(t, "failed", StatusFailed)
}
