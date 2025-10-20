package pipelineconv

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/registry"
	"github.com/alexisbeaulieu97/streamy/internal/tui/components"
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

func TestSummaryToExecutionResultNilSummary(t *testing.T) {
	execResult := SummaryToExecutionResult(nil, "config.yml")

	assert.Equal(t, "verify", execResult.Operation)
	assert.Equal(t, registry.StatusFailed, execResult.Status)
	assert.False(t, execResult.Success)
	assert.Equal(t, "verification unavailable", execResult.Summary)

	if assert.NotNil(t, execResult.Error) {
		assert.Contains(t, execResult.Error.Message, "Verification did not produce a summary")
		assert.Contains(t, execResult.Error.Context, "config.yml")
	}
}

func TestSummaryToExecutionResultDedupesFailures(t *testing.T) {
	summary := &VerificationSummary{
		TotalSteps: 3,
		Results: []*VerificationResult{
			{StepID: "step1", Status: VerificationDrifted},
			{StepID: "step1", Status: VerificationMissing, Error: errors.New("boom")},
			{StepID: "step2", Status: VerificationSatisfied},
		},
	}

	execResult := SummaryToExecutionResult(summary, "config.yml")

	assert.Equal(t, registry.StatusDrifted, execResult.Status)
	assert.False(t, execResult.Success)
	assert.Equal(t, []string{"step1"}, execResult.FailedSteps)
	assert.Len(t, execResult.StepResults, 3)
	assert.Contains(t, execResult.Summary, "steps need changes")
}

func TestMapVerificationStatusLegacyDetails(t *testing.T) {
	res := domainpipeline.VerificationResult{
		Details: map[string]interface{}{"status": "missing"},
	}

	assert.Equal(t, VerificationMissing, mapVerificationStatus(res))
}

func TestMapVerificationStatusFallbacks(t *testing.T) {
	assert.Equal(t, VerificationDrifted, mapVerificationStatus(domainpipeline.VerificationResult{Status: domainpipeline.VerificationFailed}))
	assert.Equal(t, VerificationSatisfied, mapVerificationStatus(domainpipeline.VerificationResult{Status: domainpipeline.VerificationSatisfied}))
	assert.Equal(t, VerificationUnknown, mapVerificationStatus(domainpipeline.VerificationResult{}))
}

func TestFormatVerificationDetails(t *testing.T) {
	formatted := formatVerificationDetails(map[string]interface{}{"alpha": 1, "beta": "two"})

	decoded := make(map[string]interface{})
	assert.NoError(t, json.Unmarshal([]byte(formatted), &decoded))
	assert.Equal(t, float64(1), decoded["alpha"])
	assert.Equal(t, "two", decoded["beta"])
}

func TestFormatVerificationDetailsEmpty(t *testing.T) {
	assert.Equal(t, "", formatVerificationDetails(nil))
}

func TestMapStepStatus(t *testing.T) {
	cases := []struct {
		name   string
		status domainpipeline.ResultStatus
		want   string
	}{
		{name: "success", status: domainpipeline.StatusSuccess, want: stepStatusSuccess},
		{name: "already satisfied", status: domainpipeline.StatusAlreadySatisfied, want: stepStatusSuccess},
		{name: "failure", status: domainpipeline.StatusFailure, want: stepStatusFailed},
		{name: "skipped", status: domainpipeline.StatusSkipped, want: stepStatusSkipped},
		{name: "default", status: "unknown", want: stepStatusFailed},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, mapStepStatus(tc.status))
		})
	}
}

func TestConvertApplyResultsWithErrors(t *testing.T) {
	validationErr := errors.New("validation failed")
	results := []domainpipeline.StepResult{
		{StepID: "step1", Status: domainpipeline.StatusFailure},
		{StepID: "step1", Status: domainpipeline.StatusFailure},
	}

	execResult := ConvertApplyResults(results, "config.yml", false, nil, validationErr)

	assert.False(t, execResult.Success)
	assert.Equal(t, registry.StatusFailed, execResult.Status)
	assert.Contains(t, execResult.FailedSteps, "step1")
	assert.Equal(t, "validation failed", execResult.Summary)

	if assert.NotNil(t, execResult.Error) {
		assert.Equal(t, "VALIDATION_FAILED", execResult.Error.Code)
	}
}

func TestConvertApplyResultsWithExecError(t *testing.T) {
	execErr := errors.New("apply failed")
	results := []domainpipeline.StepResult{
		{StepID: "step1", Status: domainpipeline.StatusSuccess},
	}

	execResult := ConvertApplyResults(results, "config.yml", false, execErr, nil)

	assert.False(t, execResult.Success)
	assert.Equal(t, registry.StatusFailed, execResult.Status)
	assert.Equal(t, "apply failed", execResult.Summary)

	if assert.NotNil(t, execResult.Error) {
		assert.Equal(t, "APPLY_FAILED", execResult.Error.Code)
	}
}

func TestToStepStateDryRun(t *testing.T) {
	res := domainpipeline.StepResult{Status: domainpipeline.StatusSkipped, Changed: true}

	state := ToStepState(res, true)

	assert.Equal(t, components.StepStatusWouldCreate, state.Status)
	assert.True(t, state.Changed)
}

func TestToStepStateLiveExecution(t *testing.T) {
	res := domainpipeline.StepResult{Status: domainpipeline.StatusFailure}

	state := ToStepState(res, false)

	assert.Equal(t, components.StepStatusFailed, state.Status)
}

func TestIsParseError(t *testing.T) {
	parseErr := streamyerrors.NewParseError("config.yml", 12, errors.New("boom"))

	assert.True(t, IsParseError(parseErr))
	assert.False(t, IsParseError(assert.AnError))
}

func TestDedupeStrings(t *testing.T) {
	deduped := dedupeStrings([]string{"b", "a", "b", "a"})
	assert.Equal(t, []string{"a", "b"}, deduped)
	assert.Nil(t, dedupeStrings(nil))
}
