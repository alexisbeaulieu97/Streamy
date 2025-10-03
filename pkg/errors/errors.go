package errors

import (
	"fmt"
)

// ParseError represents a YAML parsing failure with optional line metadata.
type ParseError struct {
	Path    string
	Line    int
	Message string
	Err     error
}

// NewParseError constructs a ParseError.
func NewParseError(path string, line int, err error) error {
	message := ""
	if err != nil {
		message = err.Error()
	}
	return &ParseError{Path: path, Line: line, Message: message, Err: err}
}

func (e *ParseError) Error() string {
	if e == nil {
		return ""
	}

	if e.Line > 0 {
		return fmt.Sprintf("parse error: %s:%d: %s", e.Path, e.Line, e.Message)
	}
	return fmt.Sprintf("parse error: %s: %s", e.Path, e.Message)
}

// Unwrap exposes the underlying error.
func (e *ParseError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// ValidationError captures configuration validation issues.
type ValidationError struct {
	Field   string
	Message string
	Err     error
}

// NewValidationError constructs a ValidationError.
func NewValidationError(field, message string, err error) error {
	return &ValidationError{Field: field, Message: message, Err: err}
}

func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	if e.Field != "" {
		return fmt.Sprintf("validation error: %s: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

// Unwrap exposes the underlying error.
func (e *ValidationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// ExecutionError represents a runtime failure while executing a step.
type ExecutionError struct {
	StepID string
	Err    error
}

// NewExecutionError constructs an ExecutionError.
func NewExecutionError(stepID string, err error) error {
	return &ExecutionError{StepID: stepID, Err: err}
}

func (e *ExecutionError) Error() string {
	if e == nil {
		return ""
	}
	if e.StepID != "" {
		return fmt.Sprintf("execution error on step %s: %v", e.StepID, e.Err)
	}
	return fmt.Sprintf("execution error: %v", e.Err)
}

// Unwrap exposes the root error.
func (e *ExecutionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// PluginError indicates issues within plugin registration or execution.
type PluginError struct {
	Plugin  string
	Message string
	Err     error
}

// NewPluginError constructs a PluginError for the given plugin type.
func NewPluginError(plugin string, err error) error {
	message := ""
	if err != nil {
		message = err.Error()
	}
	return &PluginError{Plugin: plugin, Message: message, Err: err}
}

func (e *PluginError) Error() string {
	if e == nil {
		return ""
	}
	if e.Plugin != "" {
		return fmt.Sprintf("plugin error [%s]: %s", e.Plugin, e.Message)
	}
	return fmt.Sprintf("plugin error: %s", e.Message)
}

// Unwrap exposes the underlying error.
func (e *PluginError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
