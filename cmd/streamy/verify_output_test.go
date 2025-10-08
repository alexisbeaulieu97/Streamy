package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/model"
)

func TestPrintTableOutput(t *testing.T) {
	summary := &model.VerificationSummary{
		TotalSteps: 2,
		Satisfied:  1,
		Missing:    1,
		Results: []*model.VerificationResult{
			{
				StepID:    "ok-step",
				Status:    model.StatusSatisfied,
				Message:   "everything is fine",
				Duration:  1500 * time.Millisecond,
				Timestamp: time.Now(),
			},
			{
				StepID:    "missing-step",
				Status:    model.StatusMissing,
				Message:   "needs attention",
				Duration:  2 * time.Second,
				Timestamp: time.Now(),
			},
		},
		Duration: 3 * time.Second,
	}

	output := captureStdout(t, func() {
		printTableOutput(summary)
	})

	require.Contains(t, output, "Verification Results")
	require.Contains(t, output, "ok-step")
	require.Contains(t, output, "missing-step")
	require.Contains(t, output, "Summary:")
	require.Contains(t, output, "Total:")
	require.Contains(t, output, "✔ Satisfied")
	require.Contains(t, output, "✖ Missing")
}

func TestPrintVerboseOutputIncludesDetails(t *testing.T) {
	driftDetails := "--- diff ---"
	blockErr := errors.New("network failure")

	summary := &model.VerificationSummary{
		TotalSteps: 2,
		Drifted:    1,
		Blocked:    1,
		Results: []*model.VerificationResult{
			{
				StepID:    "drifted-step",
				Status:    model.StatusDrifted,
				Message:   "drift detected",
				Details:   driftDetails,
				Duration:  500 * time.Millisecond,
				Timestamp: time.Now(),
			},
			{
				StepID:    "blocked-step",
				Status:    model.StatusBlocked,
				Message:   "blocked execution",
				Error:     blockErr,
				Duration:  time.Second,
				Timestamp: time.Now(),
			},
		},
		Duration: 1500 * time.Millisecond,
	}

	output := captureStdout(t, func() {
		printVerboseOutput(summary)
	})

	require.Contains(t, output, "drifted-step")
	require.Contains(t, output, driftDetails)
	require.Contains(t, output, "blocked-step")
	require.Contains(t, output, blockErr.Error())
	require.Contains(t, output, "Detailed Diff Output")
}

func TestPrintJSONOutput(t *testing.T) {
	summary := &model.VerificationSummary{
		TotalSteps: 1,
		Satisfied:  1,
		Results: []*model.VerificationResult{
			{
				StepID:    "ok",
				Status:    model.StatusSatisfied,
				Message:   "done",
				Duration:  time.Second,
				Timestamp: time.Unix(1700000000, 0).UTC(),
			},
		},
		Duration: time.Second,
	}

	output := captureStdout(t, func() {
		require.NoError(t, printJSONOutput(summary, "streamy.yml"))
	})

	require.Contains(t, output, `"config_file": "streamy.yml"`)
	require.Contains(t, output, `"status": "satisfied"`)
	require.Contains(t, output, `"total_steps": 1`)
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = w
	fn()
	require.NoError(t, w.Close())
	os.Stdout = original

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)
	require.NoError(t, r.Close())

	return buf.String()
}
