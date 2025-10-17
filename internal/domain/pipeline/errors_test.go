package pipeline

import (
	"errors"
	"fmt"
	"testing"
)

func TestDomainError_Error(t *testing.T) {
	err := &DomainError{Code: ErrCodeValidation, Message: "invalid"}
	want := "VALIDATION_ERROR: invalid"
	if err.Error() != want {
		t.Fatalf("expected %q, got %q", want, err.Error())
	}

	wrapped := &DomainError{Code: ErrCodeExecution, Message: "failure", Cause: err}
	wantWrapped := "EXECUTION_ERROR: failure: VALIDATION_ERROR: invalid"
	if wrapped.Error() != wantWrapped {
		t.Fatalf("expected %q, got %q", wantWrapped, wrapped.Error())
	}
}

func TestDomainError_IsAndUnwrap(t *testing.T) {
	inner := &DomainError{Code: ErrCodeTimeout, Message: "timed out"}
	outer := &DomainError{Code: ErrCodeExecution, Message: "exec", Cause: inner}

	if !errors.Is(outer, inner) {
		t.Fatal("expected errors.Is to match wrapped domain error")
	}

	if errors.Is(inner, outer) {
		t.Fatal("expected errors.Is to be directional")
	}

	if errors.Is(outer, fmt.Errorf("other")) {
		t.Fatal("expected non-domain errors to return false")
	}

	mismatch := &DomainError{Code: ErrCodeTimeout, Message: "other timeout"}
	if errors.Is(outer, mismatch) {
		t.Fatal("expected mismatched domain errors to be unequal")
	}
}

func TestDomainError_WithContext(t *testing.T) {
	err := &DomainError{Code: ErrCodeDependency, Message: "missing", Context: map[string]interface{}{"step_id": "build"}}
	updated := err.WithContext(map[string]interface{}{"dependency": "setup"})

	if updated.Context["step_id"] != "build" || updated.Context["dependency"] != "setup" {
		t.Fatalf("context merge failed: %+v", updated.Context)
	}

	if updated == err {
		t.Fatal("WithContext should return a new instance")
	}
}

func TestDomainError_ErrorNilReceiver(t *testing.T) {
	var err *DomainError
	if got := err.Error(); got != "<nil>" {
		t.Fatalf("expected <nil> string, got %q", got)
	}
}

func TestDomainError_UnwrapNil(t *testing.T) {
	var err *DomainError
	if err.Unwrap() != nil {
		t.Fatal("expected nil unwrap for nil receiver")
	}
}

func TestDomainError_WithContextNil(t *testing.T) {
	var err *DomainError
	if err.WithContext(map[string]interface{}{"key": "value"}) != nil {
		t.Fatal("expected nil WithContext result for nil receiver")
	}
}
