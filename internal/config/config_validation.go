package config

import (
	"fmt"
	"strings"

	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

// ValidateConfig performs structural and cross-field validation on an entire configuration.
func ValidateConfig(cfg *Config) error {
	if cfg == nil {
		return streamyerrors.NewValidationError("config", "configuration is nil", nil)
	}

	v := validatorInstance()
	if err := v.Struct(cfg); err != nil {
		return convertValidationError(err)
	}

	stepIndex := make(map[string]int, len(cfg.Steps))

	for i, step := range cfg.Steps {
		if _, exists := stepIndex[step.ID]; exists {
			return streamyerrors.NewValidationError(fieldForStep(i, "id"), fmt.Sprintf("duplicate step id %q", step.ID), nil)
		}

		if err := ValidateStep(step); err != nil {
			return err
		}

		stepIndex[step.ID] = i
	}

	for i, step := range cfg.Steps {
		for _, dep := range step.DependsOn {
			index, ok := stepIndex[dep]
			if !ok {
				return streamyerrors.NewValidationError(fieldForStep(i, "depends_on"), fmt.Sprintf("references unknown step %q", dep), nil)
			}

			if index >= i {
				// Allow forward references; cycle detection handles invalid ordering.
				continue
			}
		}
	}

	if cycle := detectCycle(cfg.Steps); len(cycle) > 0 {
		return streamyerrors.NewValidationError("steps", fmt.Sprintf("dependency cycle detected: %s", strings.Join(cycle, " -> ")), nil)
	}

	for i, validation := range cfg.Validations {
		if err := validateValidation(validation, i); err != nil {
			return err
		}
	}

	return nil
}
