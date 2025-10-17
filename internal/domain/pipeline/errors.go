package pipeline

import (
	"errors"
	"fmt"
)

// ErrorCode identifies well-known domain error categories used across the
// pipeline domain layer. These codes mirror the taxonomy defined in
// specs/009-domain-driven-refactor/errors.md.
type ErrorCode string

const (
	ErrCodeValidation ErrorCode = "VALIDATION_ERROR"
	ErrCodeDuplicate  ErrorCode = "DUPLICATE_ID"
	ErrCodeDependency ErrorCode = "DEPENDENCY_ERROR"
	ErrCodeCycle      ErrorCode = "CIRCULAR_DEPENDENCY"
	ErrCodeType       ErrorCode = "INVALID_TYPE"
	ErrCodeNotFound   ErrorCode = "NOT_FOUND"
	ErrCodeMissing    ErrorCode = "MISSING_REQUIRED"
	ErrCodeState      ErrorCode = "INVALID_STATE"
	ErrCodeConflict   ErrorCode = "CONFLICT"
	ErrCodeExecution  ErrorCode = "EXECUTION_ERROR"
	ErrCodePlugin     ErrorCode = "PLUGIN_ERROR"
	ErrCodeTimeout    ErrorCode = "TIMEOUT"
	ErrCodeCancelled  ErrorCode = "CANCELLED"
	ErrCodeInternal   ErrorCode = "INTERNAL_ERROR"
)

// DomainError represents a typed error enriched with contextual data while
// remaining free from infrastructure dependencies.
type DomainError struct {
	Code    ErrorCode
	Message string
	Cause   error
	Context map[string]interface{}
}

// Error implements the error interface.
func (e *DomainError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap exposes the wrapped cause for errors.Is / errors.As usage.
func (e *DomainError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// Is allows errors.Is comparisons against other DomainError values.
func (e *DomainError) Is(target error) bool {
	var domainErr *DomainError
	if !errors.As(target, &domainErr) {
		return false
	}
	return e.Code == domainErr.Code && e.Message == domainErr.Message
}

// WithContext clones the error with additional contextual metadata.
func (e *DomainError) WithContext(ctx map[string]interface{}) *DomainError {
	if e == nil {
		return nil
	}
	merged := make(map[string]interface{}, len(e.Context)+len(ctx))
	for k, v := range e.Context {
		merged[k] = v
	}
	for k, v := range ctx {
		merged[k] = v
	}
	return &DomainError{
		Code:    e.Code,
		Message: e.Message,
		Cause:   e.Cause,
		Context: merged,
	}
}

// newDomainError constructs a DomainError with the supplied code and message.
func newDomainError(code ErrorCode, message string, cause error, context map[string]interface{}) *DomainError {
	return (&DomainError{
		Code:    code,
		Message: message,
		Cause:   cause,
		Context: context,
	})
}

// Helper constructors to simplify error creation throughout the domain.

func newValidationError(message string, context map[string]interface{}) *DomainError {
	return newDomainError(ErrCodeValidation, message, nil, context)
}

func newDuplicateError(identifier string) *DomainError {
	return newDomainError(ErrCodeDuplicate, "duplicate identifier", nil, map[string]interface{}{
		"id": identifier,
	})
}

func newDependencyError(message string, context map[string]interface{}) *DomainError {
	return newDomainError(ErrCodeDependency, message, nil, context)
}

func newCycleError(path []string) *DomainError {
	return newDomainError(ErrCodeCycle, "circular dependency detected", nil, map[string]interface{}{
		"path": path,
	})
}

func newTypeError(expected string, actual string) *DomainError {
	return newDomainError(ErrCodeType, "invalid type", nil, map[string]interface{}{
		"expected": expected,
		"actual":   actual,
	})
}

func newMissingFieldError(field string) *DomainError {
	return newDomainError(ErrCodeMissing, "missing required field", nil, map[string]interface{}{
		"field": field,
	})
}
