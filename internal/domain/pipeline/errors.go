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
	ErrCodeConfig     ErrorCode = "CONFIG_ERROR"
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
	return e.Code == domainErr.Code && (domainErr.Message == "" || e.Message == domainErr.Message)
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

// NewDomainError constructs a DomainError with the supplied code, message, cause, and context.
func NewDomainError(code ErrorCode, message string, cause error, context map[string]interface{}) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
		Cause:   cause,
		Context: context,
	}
}

// Helper constructors to simplify error creation throughout the domain.

func NewValidationError(message string, context map[string]interface{}) *DomainError {
	return NewDomainError(ErrCodeValidation, message, nil, context)
}

func NewDuplicateError(identifier string) *DomainError {
	return NewDomainError(ErrCodeDuplicate, "duplicate identifier", nil, map[string]interface{}{
		"id": identifier,
	})
}

func NewDependencyError(message string, context map[string]interface{}) *DomainError {
	return NewDomainError(ErrCodeDependency, message, nil, context)
}

func NewCycleError(path []string) *DomainError {
	return NewDomainError(ErrCodeCycle, "circular dependency detected", nil, map[string]interface{}{
		"path": path,
	})
}

func NewTypeError(expected string, actual string) *DomainError {
	return NewDomainError(ErrCodeType, "invalid type", nil, map[string]interface{}{
		"expected": expected,
		"actual":   actual,
	})
}

func NewMissingFieldError(field string) *DomainError {
	return NewDomainError(ErrCodeMissing, "missing required field", nil, map[string]interface{}{
		"field": field,
	})
}

func NewNotFoundError(entity string, context map[string]interface{}) *DomainError {
	ctx := map[string]interface{}{"entity": entity}
	for k, v := range context {
		ctx[k] = v
	}
	return NewDomainError(ErrCodeNotFound, "resource not found", nil, ctx)
}

func NewStateError(message string, context map[string]interface{}) *DomainError {
	return NewDomainError(ErrCodeState, message, nil, context)
}

func NewConflictError(message string, context map[string]interface{}) *DomainError {
	return NewDomainError(ErrCodeConflict, message, nil, context)
}

func NewExecutionError(message string, cause error, context map[string]interface{}) *DomainError {
	return NewDomainError(ErrCodeExecution, message, cause, context)
}

func NewPluginError(message string, cause error, context map[string]interface{}) *DomainError {
	return NewDomainError(ErrCodePlugin, message, cause, context)
}

func NewTimeoutError(message string, cause error, context map[string]interface{}) *DomainError {
	return NewDomainError(ErrCodeTimeout, message, cause, context)
}

func NewCancelledError(message string, context map[string]interface{}) *DomainError {
	return NewDomainError(ErrCodeCancelled, message, nil, context)
}

func NewInternalError(message string, cause error, context map[string]interface{}) *DomainError {
	return NewDomainError(ErrCodeInternal, message, cause, context)
}

func NewConfigError(message string, cause error, context map[string]interface{}) *DomainError {
	return NewDomainError(ErrCodeConfig, message, cause, context)
}
