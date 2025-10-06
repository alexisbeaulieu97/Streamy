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

func TestVerificationStatus_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status VerificationStatus
		want   bool
	}{
		{"satisfied is valid", StatusSatisfied, true},
		{"missing is valid", StatusMissing, true},
		{"drifted is valid", StatusDrifted, true},
		{"blocked is valid", StatusBlocked, true},
		{"unknown is valid", StatusUnknown, true},
		{"invalid status", VerificationStatus("invalid"), false},
		{"empty status", VerificationStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.status.IsValid()
			require.Equal(t, tt.want, got)
		})
	}
}

func TestVerificationSummary_AllSatisfied(t *testing.T) {
	t.Parallel()

	t.Run("returns true when all steps satisfied", func(t *testing.T) {
		t.Parallel()
		summary := &VerificationSummary{
			TotalSteps: 5,
			Satisfied:  5,
			Missing:    0,
			Drifted:    0,
			Blocked:    0,
			Unknown:    0,
		}
		require.True(t, summary.AllSatisfied())
	})

	t.Run("returns false when some steps missing", func(t *testing.T) {
		t.Parallel()
		summary := &VerificationSummary{
			TotalSteps: 5,
			Satisfied:  4,
			Missing:    1,
		}
		require.False(t, summary.AllSatisfied())
	})

	t.Run("returns false when some steps drifted", func(t *testing.T) {
		t.Parallel()
		summary := &VerificationSummary{
			TotalSteps: 5,
			Satisfied:  4,
			Drifted:    1,
		}
		require.False(t, summary.AllSatisfied())
	})

	t.Run("returns true for zero steps", func(t *testing.T) {
		t.Parallel()
		summary := &VerificationSummary{
			TotalSteps: 0,
			Satisfied:  0,
		}
		require.True(t, summary.AllSatisfied())
	})
}

func TestVerificationSummary_NeedsApply(t *testing.T) {
	t.Parallel()

	t.Run("returns false when all satisfied", func(t *testing.T) {
		t.Parallel()
		summary := &VerificationSummary{
			TotalSteps: 5,
			Satisfied:  5,
		}
		require.False(t, summary.NeedsApply())
	})

	t.Run("returns true when steps missing", func(t *testing.T) {
		t.Parallel()
		summary := &VerificationSummary{
			TotalSteps: 5,
			Satisfied:  4,
			Missing:    1,
		}
		require.True(t, summary.NeedsApply())
	})

	t.Run("returns true when steps drifted", func(t *testing.T) {
		t.Parallel()
		summary := &VerificationSummary{
			TotalSteps: 5,
			Satisfied:  4,
			Drifted:    1,
		}
		require.True(t, summary.NeedsApply())
	})

	t.Run("returns true when steps blocked", func(t *testing.T) {
		t.Parallel()
		summary := &VerificationSummary{
			TotalSteps: 5,
			Satisfied:  4,
			Blocked:    1,
		}
		require.True(t, summary.NeedsApply())
	})

	t.Run("returns true when steps unknown", func(t *testing.T) {
		t.Parallel()
		summary := &VerificationSummary{
			TotalSteps: 5,
			Satisfied:  4,
			Unknown:    1,
		}
		require.True(t, summary.NeedsApply())
	})
}

func TestVerificationSummary_ExitCode(t *testing.T) {
	t.Parallel()

	t.Run("returns 0 when all satisfied", func(t *testing.T) {
		t.Parallel()
		summary := &VerificationSummary{
			TotalSteps: 5,
			Satisfied:  5,
		}
		require.Equal(t, 0, summary.ExitCode())
	})

	t.Run("returns 1 when steps missing", func(t *testing.T) {
		t.Parallel()
		summary := &VerificationSummary{
			TotalSteps: 5,
			Satisfied:  4,
			Missing:    1,
		}
		require.Equal(t, 1, summary.ExitCode())
	})

	t.Run("returns 1 when steps drifted", func(t *testing.T) {
		t.Parallel()
		summary := &VerificationSummary{
			TotalSteps: 5,
			Satisfied:  4,
			Drifted:    1,
		}
		require.Equal(t, 1, summary.ExitCode())
	})
}

func TestVerificationResult_Creation(t *testing.T) {
	t.Parallel()

	t.Run("creates verification result with all fields", func(t *testing.T) {
		t.Parallel()
		now := time.Now()
		result := VerificationResult{
			StepID:    "test_step",
			Status:    StatusSatisfied,
			Message:   "verified",
			Details:   "no changes",
			Duration:  time.Millisecond * 100,
			Timestamp: now,
		}

		require.Equal(t, "test_step", result.StepID)
		require.Equal(t, StatusSatisfied, result.Status)
		require.Equal(t, "verified", result.Message)
		require.Equal(t, "no changes", result.Details)
		require.Equal(t, time.Millisecond*100, result.Duration)
		require.Equal(t, now, result.Timestamp)
	})

	t.Run("creates verification result with error", func(t *testing.T) {
		t.Parallel()
		err := &TestError{msg: "verification error"}
		result := VerificationResult{
			StepID: "blocked_step",
			Status: StatusBlocked,
			Error:  err,
		}

		require.Equal(t, "blocked_step", result.StepID)
		require.Equal(t, StatusBlocked, result.Status)
		require.Equal(t, err, result.Error)
	})
}
