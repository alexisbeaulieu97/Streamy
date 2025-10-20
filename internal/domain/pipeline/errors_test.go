package pipeline

import (
	"context"
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

func TestErrorHelperConstructors(t *testing.T) {
	t.Parallel()

	var (
		baseCause    = errors.New("root cause")
		timeoutCause = context.DeadlineExceeded
	)

	testCases := []struct {
		name    string
		err     *DomainError
		want    ErrorCode
		message string
	}{
		{"validation", NewValidationError("invalid", nil), ErrCodeValidation, "invalid"},
		{"duplicate", NewDuplicateError("step-1"), ErrCodeDuplicate, "duplicate identifier"},
		{"dependency", NewDependencyError("missing dep", map[string]interface{}{"dep": "step-2"}), ErrCodeDependency, "missing dep"},
		{"cycle", NewCycleError([]string{"a", "b"}), ErrCodeCycle, "circular dependency detected"},
		{"type", NewTypeError("command", "template"), ErrCodeType, "invalid type"},
		{"missing", NewMissingFieldError("field"), ErrCodeMissing, "missing required field"},
		{"not found", NewNotFoundError("step", nil), ErrCodeNotFound, "resource not found"},
		{"state", NewStateError("bad state", nil), ErrCodeState, "bad state"},
		{"conflict", NewConflictError("conflict", nil), ErrCodeConflict, "conflict"},
		{"execution", NewExecutionError("exec failed", baseCause, nil), ErrCodeExecution, "exec failed"},
		{"plugin", NewPluginError("plugin failed", baseCause, nil), ErrCodePlugin, "plugin failed"},
		{"timeout", NewTimeoutError("timed out", timeoutCause, nil), ErrCodeTimeout, "timed out"},
		{"cancelled", NewCancelledError("cancelled", nil), ErrCodeCancelled, "cancelled"},
		{"internal", NewInternalError("internal", baseCause, nil), ErrCodeInternal, "internal"},
		{"config", NewConfigError("config error", baseCause, map[string]interface{}{"path": "config.yaml"}), ErrCodeConfig, "config error"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.err == nil {
				t.Fatalf("expected error instance for %s", tc.name)
			}

			if tc.err.Code != tc.want {
				t.Fatalf("expected code %s, got %s", tc.want, tc.err.Code)
			}

			if tc.err.Message != tc.message {
				t.Fatalf("expected message %q, got %q", tc.message, tc.err.Message)
			}

			switch tc.want {
			case ErrCodeExecution, ErrCodePlugin, ErrCodeInternal:
				if !errors.Is(tc.err, baseCause) {
					t.Fatalf("expected to wrap base cause for %s", tc.name)
				}
			case ErrCodeTimeout:
				if !errors.Is(tc.err, timeoutCause) {
					t.Fatalf("expected timeout to wrap context deadline for %s", tc.name)
				}
			}
		})
	}

	// Ensure NewDomainError behaves as passthrough.
	custom := NewDomainError(ErrCodeInternal, "custom", baseCause, map[string]interface{}{"x": 1})
	if custom.Code != ErrCodeInternal || custom.Message != "custom" || !errors.Is(custom, baseCause) {
		t.Fatal("NewDomainError did not populate fields correctly")
	}

	if custom.Context["x"] != 1 {
		t.Fatal("expected context propagated")
	}
}
