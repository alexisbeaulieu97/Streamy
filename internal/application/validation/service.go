package validation

import (
	"context"

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
	for _, val := range validations {
		if err := ctx.Err(); err != nil {
			return summary, &domain.DomainError{Code: domain.ErrCodeCancelled, Message: "validation cancelled", Cause: err}
		}
		result := domain.VerificationResult{
			Type: string(val.Type),
		}

		var err error
		switch val.Type {
		case domain.ValidationCommandExists:
			command, getErr := stringConfig(val.Config, "command")
			if getErr != nil {
				err = getErr
			} else {
				err = legacyvalidation.CheckCommandExists(command)
			}
		case domain.ValidationFileExists:
			path, getErr := stringConfig(val.Config, "path")
			if getErr != nil {
				err = getErr
			} else {
				err = legacyvalidation.CheckFileExists(path)
			}
		case domain.ValidationPathContains:
			file, fileErr := stringConfig(val.Config, "file")
			text, textErr := stringConfig(val.Config, "text")
			if fileErr != nil {
				err = fileErr
			} else if textErr != nil {
				err = textErr
			} else {
				err = legacyvalidation.CheckPathContains(file, text)
			}
		default:
			err = &domain.DomainError{
				Code:    domain.ErrCodeValidation,
				Message: "unsupported validation type",
				Context: map[string]interface{}{"validation_type": val.Type},
			}
		}

		if err != nil {
			result.Status = domain.VerificationFailed
			result.Message = err.Error()
			result.Details = map[string]interface{}{
				"validation_type": val.Type,
			}
			if s.logger != nil {
				s.logger.Warn(ctx, "validation failed", "validation_type", val.Type, "error", err)
			}
			summary.Add(result)
			continue
		}

		result.Status = domain.VerificationSatisfied
		result.Message = "passed"
		if s.logger != nil {
			s.logger.Info(ctx, "validation passed", "validation_type", val.Type)
		}
		summary.Add(result)
	}

	if summary.FailedChecks > 0 {
		return summary, &domain.DomainError{
			Code:    domain.ErrCodeValidation,
			Message: "one or more validations failed",
			Context: map[string]interface{}{"failed_checks": summary.FailedChecks},
		}
	}

	return summary, nil
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
