package plugin

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ErrPluginNotFound is returned when the requested plugin is not registered.
type ErrPluginNotFound struct {
	Name string
}

func (e ErrPluginNotFound) Error() string {
	return fmt.Sprintf("plugin '%s' not found in registry\nHint: ensure the plugin is registered before usage", e.Name)
}

// ErrCircularDependency is returned when a dependency cycle is detected.
type ErrCircularDependency struct {
	Cycle []string
}

func (e ErrCircularDependency) Error() string {
	if len(e.Cycle) == 0 {
		return "circular dependency detected\nHint: review plugin dependencies to remove cycles"
	}

	sequence := append(append([]string{}, e.Cycle...), e.Cycle[0])
	return fmt.Sprintf(
		"circular dependency detected: %s\nHint: break the cycle by removing or refactoring one of the dependencies",
		strings.Join(sequence, " -> "),
	)
}

// ErrVersionConflict captures version mismatches between dependents and a plugin.
type ErrVersionConflict struct {
	Plugin        string
	RequiredBy    map[string]string // dependent -> version constraint
	ActualVersion string
}

func (e ErrVersionConflict) Error() string {
	conflicts := make([]string, 0, len(e.RequiredBy))
	for dependent, constraint := range e.RequiredBy {
		conflicts = append(conflicts, fmt.Sprintf("%s requires %s", dependent, constraint))
	}
	sort.Strings(conflicts)

	return fmt.Sprintf(
		"version conflict for plugin '%s' (actual %s):\n  %s\nHint: align plugin versions or relax constraints",
		e.Plugin,
		e.ActualVersion,
		strings.Join(conflicts, "\n  "),
	)
}

// ErrUndeclaredDependency is returned when a plugin accesses a dependency it did not declare.
type ErrUndeclaredDependency struct {
	Caller     string
	Dependency string
}

func (e ErrUndeclaredDependency) Error() string {
	return fmt.Sprintf(
		"plugin '%s' attempted to access undeclared dependency '%s'\nHint: add '%s' to PluginMetadata.Dependencies",
		e.Caller,
		e.Dependency,
		e.Dependency,
	)
}

// ErrMissingDependency is returned when a declared dependency has not been registered.
type ErrMissingDependency struct {
	Plugin     string
	Dependency string
}

func (e ErrMissingDependency) Error() string {
	return fmt.Sprintf(
		"plugin '%s' declares dependency '%s' which is not registered\nHint: register the dependency before validating or initializing plugins",
		e.Plugin,
		e.Dependency,
	)
}

// PluginError is the base interface for all plugin errors.
// It provides structured error information that the executor can use
// to make intelligent decisions about how to handle failures.
type PluginError interface {
	error
	StepID() string
	Unwrap() error
}

// ValidationError represents configuration or input validation failures.
// These are typically caused by malformed YAML, missing required fields,
// or invalid field values in the step configuration.
type ValidationError struct {
	ID  string
	Err error
}

// NewValidationError creates a new ValidationError.
func NewValidationError(stepID string, err error) *ValidationError {
	return &ValidationError{
		ID:  stepID,
		Err: err,
	}
}

// Error returns a formatted error message including the step ID.
func (e *ValidationError) Error() string {
	if e.Err == nil {
		return "validation error in step " + e.ID
	}
	return "validation error in step " + e.ID + ": " + e.Err.Error()
}

// StepID returns the identifier of the step where the error occurred.
func (e *ValidationError) StepID() string {
	return e.ID
}

// Unwrap returns the underlying validation error.
func (e *ValidationError) Unwrap() error {
	return e.Err
}

// Is checks if this error matches another ValidationError.
func (e *ValidationError) Is(target error) bool {
	_, ok := target.(*ValidationError)
	return ok
}

// ExecutionError represents external operation failures during state
// assessment or application. These include shell command failures,
// file I/O errors, network operation failures, or external tool errors.
type ExecutionError struct {
	ID  string
	Err error
}

// NewExecutionError creates a new ExecutionError.
func NewExecutionError(stepID string, err error) *ExecutionError {
	return &ExecutionError{
		ID:  stepID,
		Err: err,
	}
}

// Error returns a formatted error message including the step ID.
func (e *ExecutionError) Error() string {
	if e.Err == nil {
		return "execution error in step " + e.ID
	}
	return "execution error in step " + e.ID + ": " + e.Err.Error()
}

// StepID returns the identifier of the step where the error occurred.
func (e *ExecutionError) StepID() string {
	return e.ID
}

// Unwrap returns the underlying execution error.
func (e *ExecutionError) Unwrap() error {
	return e.Err
}

// Is checks if this error matches another ExecutionError.
func (e *ExecutionError) Is(target error) bool {
	_, ok := target.(*ExecutionError)
	return ok
}

// StateError represents inability to determine the current system state.
// These are used when the plugin cannot read or assess the current state,
// such as when files are inaccessible, package managers are unavailable,
// or system state is inconsistent or corrupted.
type StateError struct {
	ID  string
	Err error
}

// NewStateError creates a new StateError.
func NewStateError(stepID string, err error) *StateError {
	return &StateError{
		ID:  stepID,
		Err: err,
	}
}

// Error returns a formatted error message including the step ID.
func (e *StateError) Error() string {
	if e.Err == nil {
		return "state error in step " + e.ID
	}
	return "state error in step " + e.ID + ": " + e.Err.Error()
}

// StepID returns the identifier of the step where the error occurred.
func (e *StateError) StepID() string {
	return e.ID
}

// Unwrap returns the underlying state detection error.
func (e *StateError) Unwrap() error {
	return e.Err
}

// Is checks if this error matches another StateError.
func (e *StateError) Is(target error) bool {
	_, ok := target.(*StateError)
	return ok
}

// AsPluginError attempts to convert any error to a PluginError.
// This helper function can be used by the executor to categorize errors.
func AsPluginError(err error) (PluginError, bool) {
	var pluginErr PluginError
	if errors.As(err, &pluginErr) {
		return pluginErr, true
	}
	return nil, false
}
