package pipeline

import "testing"

func TestStepResultHelpers(t *testing.T) {
	res := StepResult{StepID: "a", Status: StatusSuccess, Message: "ok"}
	if !res.IsSuccess() {
		t.Fatal("expected success")
	}
	if res.IsFailure() {
		t.Fatal("did not expect failure")
	}

	failure := StepResult{StepID: "b", Status: StatusFailure, Error: &DomainError{Code: ErrCodeExecution, Message: "boom"}}
	if failure.FormatOutput() != "EXECUTION_ERROR: boom" {
		t.Fatalf("unexpected output: %s", failure.FormatOutput())
	}
}

func TestVerificationResultHelpers(t *testing.T) {
	res := VerificationResult{Status: VerificationSatisfied}
	if !res.IsSatisfied() {
		t.Fatal("expected satisfied")
	}
	if res.FormatMessage() != string(VerificationSatisfied) {
		t.Fatalf("unexpected message: %s", res.FormatMessage())
	}
}

func TestVerificationSummary(t *testing.T) {
	var summary VerificationSummary
	summary.Add(VerificationResult{Status: VerificationSatisfied})
	summary.Add(VerificationResult{Status: VerificationFailed})

	if summary.TotalChecks != 2 || summary.PassedChecks != 1 || summary.FailedChecks != 1 {
		t.Fatalf("unexpected summary counters: %+v", summary)
	}

	other := VerificationSummary{TotalChecks: 1, PassedChecks: 1, Results: []VerificationResult{{Status: VerificationSatisfied}}}
	summary.Merge(other)

	if summary.TotalChecks != 3 || summary.PassedChecks != 2 {
		t.Fatalf("unexpected merged summary: %+v", summary)
	}
}
