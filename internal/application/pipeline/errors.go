package pipeline

import (
	"fmt"
	"strconv"
	"strings"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
)

const (
	prepareHint     = "Ensure the configuration file exists, is readable, and passes validation."
	dagBuildHint    = "Check for missing or circular step dependencies in the configuration."
	dagValidateHint = "Verify that every execution level references the correct steps and dependencies."
	executionHint   = "Run with --verbose or inspect plugin logs to diagnose failing steps."
	validationHint  = "Review validation definitions or re-run with --verbose for detailed output."
	verifyHint      = "Run with --verbose to inspect verification failures and ensure plugins support verification."
)

// AggregateError represents multiple errors that occurred during a single pipeline stage.
type AggregateError struct {
	Message string
	Errors  []error
}

// Error implements the error interface.
func (e *AggregateError) Error() string {
	if e == nil {
		return ""
	}

	msg := e.Message
	if msg == "" {
		msg = "multiple errors occurred"
	}

	var b strings.Builder
	b.WriteString(msg)
	for idx, err := range e.Errors {
		if err == nil {
			continue
		}
		b.WriteString("\n  ")
		b.WriteString(strconv.Itoa(idx + 1))
		b.WriteString(". ")
		b.WriteString(err.Error())
	}
	return b.String()
}

// Unwrap exposes the underlying errors for errors.Is / errors.As.
func (e *AggregateError) Unwrap() []error {
	if e == nil {
		return nil
	}
	result := make([]error, 0, len(e.Errors))
	for _, err := range e.Errors {
		if err != nil {
			result = append(result, err)
		}
	}
	return result
}

func wrapPipelineError(stage, hint string, err error) error {
	if err == nil {
		return nil
	}
	if hint == "" {
		return fmt.Errorf("%s: %w", stage, err)
	}
	return fmt.Errorf("%s: %w\nHint: %s", stage, err, hint)
}

func aggregateErrors(message string, errs []error) error {
	filtered := make([]error, 0, len(errs))
	for _, err := range errs {
		if err == nil {
			continue
		}
		filtered = append(filtered, err)
	}
	switch len(filtered) {
	case 0:
		return nil
	case 1:
		return filtered[0]
	default:
		return &AggregateError{
			Message: message,
			Errors:  filtered,
		}
	}
}

func wrapExecutionError(results []domainpipeline.StepResult, err error) error {
	if err == nil {
		return nil
	}
	stepErrors := make([]error, 0, len(results)+1)
	for _, res := range results {
		if res.Error != nil {
			stepErrors = append(stepErrors, res.Error)
		}
	}
	stepErrors = append(stepErrors, err)

	aggregated := aggregateErrors("pipeline execution failed", stepErrors)
	if aggregated == nil {
		aggregated = err
	}
	return wrapPipelineError("execute pipeline", executionHint, aggregated)
}

func wrapValidationError(err error) error {
	return wrapPipelineError("run post-execution validations", validationHint, err)
}

func wrapVerificationError(err error) error {
	return wrapPipelineError("verify pipeline", verifyHint, err)
}

func wrapDagBuildError(err error) error {
	return wrapPipelineError("build execution plan", dagBuildHint, err)
}

func wrapDagValidationError(err error) error {
	return wrapPipelineError("validate execution plan", dagValidateHint, err)
}

func wrapPrepareError(err error) error {
	return wrapPipelineError("prepare pipeline", prepareHint, err)
}
