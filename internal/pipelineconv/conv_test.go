package pipelineconv

import (
	"testing"
	"time"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
	assert "github.com/stretchr/testify/assert"
)

func TestBuildVerificationSummary(t *testing.T) {
	pipeline := &domainpipeline.Pipeline{
		Steps: []domainpipeline.Step{
			{ID: "step1"},
			{ID: "step2"},
			{ID: "step3"},
			{ID: "step4"},
			{ID: "step5"},
		},
	}
	results := []domainpipeline.VerificationResult{
		{StepID: "step1", Status: domainpipeline.VerificationSatisfied, Message: "Step 1 is satisfied"},
		{StepID: "step2", Status: domainpipeline.VerificationFailed, Message: "Step 2 has drifted"},
		{StepID: "step3", Status: domainpipeline.VerificationUnknown, Message: "Step 3 is in an unknown state"},
		{StepID: "step4", Status: domainpipeline.VerificationSatisfied, Details: map[string]interface{}{"current_state": "blocked"}},
		{StepID: "step5", Status: domainpipeline.VerificationSatisfied, Details: map[string]interface{}{"current_state": "missing"}},
	}

	summary := BuildVerificationSummary(pipeline, results)

	assert.Equal(t, 5, summary.TotalSteps)
	assert.Equal(t, 1, summary.Satisfied)
	assert.Equal(t, 1, summary.Drifted)
	assert.Equal(t, 1, summary.Unknown)
	assert.Equal(t, 1, summary.Blocked)
	assert.Equal(t, 1, summary.Missing)
	assert.Len(t, summary.Results, 5)

	assert.False(t, summary.AllSatisfied())
	assert.Equal(t, 1, summary.ExitCode())
}

func TestSummaryToExecutionResult(t *testing.T) {
	summary := &VerificationSummary{
		TotalSteps: 2,
		Satisfied:  1,
		Drifted:    1,
		Results: []*VerificationResult{
			{StepID: "step1", Status: VerificationSatisfied},
			{StepID: "step2", Status: VerificationDrifted, Error: assert.AnError},
		},
	}

	execResult := SummaryToExecutionResult(summary, "/path/to/config")

	assert.Equal(t, "verify", execResult.Operation)
	assert.Equal(t, "drifted", string(execResult.Status))
	assert.False(t, execResult.Success)
	assert.Equal(t, 2, execResult.StepCount)
	assert.Len(t, execResult.StepResults, 2)
	assert.Contains(t, execResult.FailedSteps, "step2")
	assert.NotNil(t, execResult.Error)
}

func TestConvertStepResult(t *testing.T) {
	res := domainpipeline.StepResult{
		StepID:   "step1",
		Status:   domainpipeline.StatusSuccess,
		Changed:  true,
		Duration: time.Second,
	}

	// Test with dryRun = true
	stepResultDryRun := ConvertStepResult(res, true)
	assert.Equal(t, "step1", stepResultDryRun.StepID)
	assert.Equal(t, "would_update", stepResultDryRun.Status)

	// Test with dryRun = false
	stepResult := ConvertStepResult(res, false)
	assert.Equal(t, "success", stepResult.Status)
}

func TestConvertApplyResults(t *testing.T) {
	results := []domainpipeline.StepResult{
		{StepID: "step1", Status: domainpipeline.StatusSuccess},
		{StepID: "step2", Status: domainpipeline.StatusFailure, Error: &domainpipeline.DomainError{}},
	}

	execResult := ConvertApplyResults(results, "/path/to/config", false, nil, nil)

	assert.Equal(t, "apply", execResult.Operation)
	assert.Equal(t, "failed", string(execResult.Status))
	assert.False(t, execResult.Success)
	assert.Len(t, execResult.FailedSteps, 1)
	assert.Contains(t, execResult.FailedSteps, "step2")
}

func TestIsConfigError(t *testing.T) {
	assert.True(t, IsConfigError(&streamyerrors.ParseError{}))
	assert.True(t, IsConfigError(&streamyerrors.ValidationError{}))
	assert.True(t, IsConfigError(&domainpipeline.DomainError{Code: domainpipeline.ErrCodeValidation}))
	assert.False(t, IsConfigError(assert.AnError))
}
