// Package pipeline defines domain error types and validation utilities.
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
	// ErrCodeValidation indicates input validation problems discovered when constructing or executing a pipeline.
	ErrCodeValidation ErrorCode = "VALIDATION_ERROR"
	// ErrCodeDuplicate is returned when attempting to register a pipeline that already exists.
	ErrCodeDuplicate ErrorCode = "DUPLICATE_ID"
	// ErrCodeDependency signals dependency resolution failures between pipeline steps.
	ErrCodeDependency ErrorCode = "DEPENDENCY_ERROR"
	// ErrCodeCycle describes cycles detected in the execution graph.
	ErrCodeCycle ErrorCode = "CIRCULAR_DEPENDENCY"
	// ErrCodeType marks an unexpected type encountered during processing.
	ErrCodeType ErrorCode = "INVALID_TYPE"
	// ErrCodeNotFound is returned when a referenced pipeline resource cannot be located.
	ErrCodeNotFound ErrorCode = "NOT_FOUND"
	// ErrCodeMissing identifies required configuration that is absent.
	ErrCodeMissing ErrorCode = "MISSING_REQUIRED"
	// ErrCodeState highlights invalid lifecycle transitions for a pipeline or step.
	ErrCodeState ErrorCode = "INVALID_STATE"
	// ErrCodeConflict communicates conflicting pipeline configuration.
	ErrCodeConflict ErrorCode = "CONFLICT"
	// ErrCodeExecution captures runtime execution failures.
	ErrCodeExecution ErrorCode = "EXECUTION_ERROR"
	// ErrCodePlugin wraps errors surfaced by external plugins.
	ErrCodePlugin ErrorCode = "PLUGIN_ERROR"
	// ErrCodeTimeout indicates that an operation exceeded the allowed time.
	ErrCodeTimeout ErrorCode = "TIMEOUT"
	// ErrCodeCancelled signals that an operation was deliberately cancelled.
	ErrCodeCancelled ErrorCode = "CANCELLED"
	// ErrCodeInternal surfaces unexpected internal errors.
	ErrCodeInternal ErrorCode = "INTERNAL_ERROR"
	// ErrCodeConfig denotes configuration parsing or validation problems.
	ErrCodeConfig ErrorCode = "CONFIG_ERROR"
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

// NewValidationError returns a validation error with the supplied context.
func NewValidationError(message string, context map[string]interface{}) *DomainError {
	return NewDomainError(ErrCodeValidation, message, nil, context)
}

// NewDuplicateError returns a duplicate identifier error for the provided ID.
func NewDuplicateError(identifier string) *DomainError {
	return NewDomainError(ErrCodeDuplicate, "duplicate identifier", nil, map[string]interface{}{
		"id": identifier,
	})
}

// NewDependencyError reports a dependency resolution failure.
func NewDependencyError(message string, context map[string]interface{}) *DomainError {
	return NewDomainError(ErrCodeDependency, message, nil, context)
}

// NewCycleError returns an error describing a detected circular dependency path.
func NewCycleError(path []string) *DomainError {
	return NewDomainError(ErrCodeCycle, "circular dependency detected", nil, map[string]interface{}{
		"path": path,
	})
}

// NewTypeError reports that an unexpected type was encountered.
func NewTypeError(expected, actual string) *DomainError {
	return NewDomainError(ErrCodeType, "invalid type", nil, map[string]interface{}{
		"expected": expected,
		"actual":   actual,
	})
}

// NewMissingFieldError reports a missing required configuration field.
func NewMissingFieldError(field string) *DomainError {
	return NewDomainError(ErrCodeMissing, "missing required field", nil, map[string]interface{}{
		"field": field,
	})
}

// NewNotFoundError constructs an error describing a missing entity.
func NewNotFoundError(entity string, context map[string]interface{}) *DomainError {
	ctx := map[string]interface{}{"entity": entity}
	for k, v := range context {
		ctx[k] = v
	}

	return NewDomainError(ErrCodeNotFound, "resource not found", nil, ctx)
}

// NewStateError returns an error for invalid state transitions.
func NewStateError(message string, context map[string]interface{}) *DomainError {
	return NewDomainError(ErrCodeState, message, nil, context)
}

// NewConflictError reports mutually exclusive configuration values.
func NewConflictError(message string, context map[string]interface{}) *DomainError {
	return NewDomainError(ErrCodeConflict, message, nil, context)
}

// NewExecutionError wraps execution-time failures with optional context.
func NewExecutionError(message string, cause error, context map[string]interface{}) *DomainError {
	return NewDomainError(ErrCodeExecution, message, cause, context)
}

// NewPluginError wraps plugin-originating errors.
func NewPluginError(message string, cause error, context map[string]interface{}) *DomainError {
	return NewDomainError(ErrCodePlugin, message, cause, context)
}

// NewTimeoutError returns an error for operations that exceeded their deadline.
func NewTimeoutError(message string, cause error, context map[string]interface{}) *DomainError {
	return NewDomainError(ErrCodeTimeout, message, cause, context)
}

// NewCancelledError reports an intentional cancellation.
func NewCancelledError(message string, context map[string]interface{}) *DomainError {
	return NewDomainError(ErrCodeCancelled, message, nil, context)
}

// NewInternalError wraps unexpected internal failures.
func NewInternalError(message string, cause error, context map[string]interface{}) *DomainError {
	return NewDomainError(ErrCodeInternal, message, cause, context)
}

// NewConfigError reports configuration parsing or validation issues.
func NewConfigError(message string, cause error, context map[string]interface{}) *DomainError {
	return NewDomainError(ErrCodeConfig, message, cause, context)
}
