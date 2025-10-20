// Package validation coordinates domain validation execution for applications.
package validation

import (
	"context"
	"errors"
	"fmt"

	domain "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
	legacyvalidation "github.com/alexisbeaulieu97/streamy/internal/validation"
)

// Service executes domain validations using infrastructure checks from the
// legacy validation package while emitting structured logs.
type Service struct {
	logger ports.Logger
}

// NewService constructs a validation service.
func NewService(logger ports.Logger) *Service {
	return &Service{logger: logger}
}

// RunValidations executes validations and aggregates the results into a domain
// VerificationSummary. Failures return a summary alongside an error annotated
// with domain context.
func (s *Service) RunValidations(ctx context.Context, validations []domain.Validation) (domain.VerificationSummary, error) {
	summary := domain.VerificationSummary{}
	failureErrors := make([]error, 0)

	for _, val := range validations {
		if err := ctx.Err(); err != nil {
			return summary, domain.NewDomainError(domain.ErrCodeCancelled, "validation cancelled", err, nil)
		}

		outcome := s.executeValidation(ctx, val)
		if outcome.fatal {
			return summary, outcome.err
		}

		summary.Add(outcome.result)

		if outcome.err != nil {
			failureErrors = append(failureErrors, outcome.err)
		}
	}

	if summary.FailedChecks > 0 {
		aggregateContext := map[string]interface{}{"failed_checks": summary.FailedChecks}
		if len(summary.FailureDetails) > 0 {
			aggregateContext["failures"] = summary.FailureDetails
		}

		joined := errors.Join(failureErrors...)

		return summary, domain.NewDomainError(domain.ErrCodeValidation, "one or more validations failed", joined, aggregateContext)
	}

	return summary, nil
}

type validationOutcome struct {
	result domain.VerificationResult
	err    error
	fatal  bool
}

func (s *Service) executeValidation(ctx context.Context, val domain.Validation) validationOutcome {
	result := domain.VerificationResult{Type: string(val.Type)}

	if validateErr := val.Validate(); validateErr != nil {
		err := domain.NewDomainError(domain.ErrCodeValidation, "validation descriptor invalid", validateErr, map[string]interface{}{"validation_type": val.Type})
		return validationOutcome{result: result, err: err, fatal: true}
	}

	runner, ok := validationRunners[val.Type]
	if !ok {
		err := &domain.DomainError{
			Code:    domain.ErrCodeValidation,
			Message: "unsupported validation type",
			Context: map[string]interface{}{"validation_type": val.Type},
		}

		return s.failValidation(ctx, result, err)
	}

	err := runner(val)
	if err != nil {
		return s.failValidation(ctx, result, err)
	}

	return s.passValidation(ctx, result)
}

func (s *Service) failValidation(ctx context.Context, result domain.VerificationResult, err error) validationOutcome {
	var domainErr *domain.DomainError
	if errors.As(err, &domainErr) {
		err = domainErr
	} else {
		err = domain.NewDomainError(domain.ErrCodeValidation, err.Error(), err, map[string]interface{}{"validation_type": result.Type})
	}

	result.Status = domain.VerificationFailed
	result.Message = err.Error()
	result.Details = map[string]interface{}{
		"validation_type": result.Type,
	}

	if s.logger != nil {
		s.logger.Warn(ctx, "validation failed", "validation_type", result.Type, "error", err)
	}

	return validationOutcome{result: result, err: err, fatal: false}
}

func (s *Service) passValidation(ctx context.Context, result domain.VerificationResult) validationOutcome {
	result.Status = domain.VerificationSatisfied
	result.Message = "passed"

	if s.logger != nil {
		s.logger.Info(ctx, "validation passed", "validation_type", result.Type)
	}

	return validationOutcome{result: result}
}

type validationRunner func(domain.Validation) error

var validationRunners = map[domain.ValidationType]validationRunner{
	domain.ValidationCommandExists: runCommandExists,
	domain.ValidationFileExists:    runFileExists,
	domain.ValidationPathContains:  runPathContains,
}

func runCommandExists(val domain.Validation) error {
	command, err := stringConfig(val.Config, "command")
	if err != nil {
		return err
	}

	if execErr := legacyvalidation.CheckCommandExists(command); execErr != nil {
		return fmt.Errorf("check command exists: %w", execErr)
	}

	return nil
}

func runFileExists(val domain.Validation) error {
	path, err := stringConfig(val.Config, "path")
	if err != nil {
		return err
	}

	if execErr := legacyvalidation.CheckFileExists(path); execErr != nil {
		return fmt.Errorf("check file exists: %w", execErr)
	}

	return nil
}

func runPathContains(val domain.Validation) error {
	file, fileErr := stringConfig(val.Config, "file")
	if fileErr != nil {
		return fileErr
	}

	text, textErr := stringConfig(val.Config, "text")
	if textErr != nil {
		return textErr
	}

	if execErr := legacyvalidation.CheckPathContains(file, text); execErr != nil {
		return fmt.Errorf("check path contains: %w", execErr)
	}

	return nil
}

func stringConfig(cfg map[string]interface{}, key string) (string, error) {
	if cfg == nil {
		return "", &domain.DomainError{Code: domain.ErrCodeValidation, Message: "validation config is required", Context: map[string]interface{}{"required_key": key}}
	}

	raw, ok := cfg[key]
	if !ok {
		return "", &domain.DomainError{Code: domain.ErrCodeMissing, Message: "validation config key missing", Context: map[string]interface{}{"missing_key": key}}
	}

	value, ok := raw.(string)
	if !ok || value == "" {
		return "", &domain.DomainError{Code: domain.ErrCodeValidation, Message: "validation config key must be a non-empty string", Context: map[string]interface{}{"invalid_key": key}}
	}

	return value, nil
}

var _ ports.ValidationService = (*Service)(nil)
