package pipeline

import (
	"errors"
	"strings"
	"testing"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
)

func TestAggregateError_ErrorAndUnwrap(t *testing.T) {
	base := errors.New("root cause")
	derr := domainpipeline.NewExecutionError("step failed", base, map[string]interface{}{"step_id": "build"})

	agg := &AggregateError{
		Message: "pipeline execution failed",
		Errors:  []error{derr, base},
	}

	msg := agg.Error()
	if !strings.Contains(msg, "pipeline execution failed") {
		t.Fatalf("expected message to contain summary, got %q", msg)
	}
	if !strings.Contains(msg, "1. ") || !strings.Contains(msg, "2. ") {
		t.Fatalf("expected enumeration of errors, got %q", msg)
	}

	if !errors.Is(agg, base) {
		t.Fatal("expected aggregated error to unwrap to base cause")
	}
	if !errors.Is(agg, derr) {
		t.Fatal("expected aggregated error to unwrap to domain error")
	}

	children := agg.Unwrap()
	if len(children) != 2 {
		t.Fatalf("expected two child errors, got %d", len(children))
	}
}

func TestWrapPipelineErrorAddsHint(t *testing.T) {
	base := errors.New("boom")
	wrapped := wrapPipelineError("execute pipeline", executionHint, base)

	if !errors.Is(wrapped, base) {
		t.Fatal("expected wrapped error to contain original error")
	}
	if !strings.Contains(wrapped.Error(), "Hint:") {
		t.Fatalf("expected hint in error message, got %q", wrapped.Error())
	}
}

func TestWrapExecutionErrorAggregatesStepErrors(t *testing.T) {
	stepErr := domainpipeline.NewExecutionError("plugin failed", errors.New("root"), map[string]interface{}{"step_id": "build"})
	err := wrapExecutionError([]domainpipeline.StepResult{{StepID: "build", Error: stepErr}}, errors.New("executor failed"))

	if !strings.Contains(err.Error(), "pipeline execution failed") {
		t.Fatalf("expected aggregated message, got %q", err.Error())
	}
	if !errors.Is(err, stepErr) {
		t.Fatal("expected aggregated error to include step error")
	}
}
