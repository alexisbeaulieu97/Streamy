package config

import (
	"fmt"

	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

// ValidateStep inspects a single step for structural correctness independent of other steps.
func ValidateStep(step Step) error {
	v := validatorInstance()
	if err := v.Struct(step); err != nil {
		return convertValidationError(err)
	}

	switch step.Type {
	case "package":
		var cfg PackageStep
		if err := decodeStepConfig(step, "package", &cfg); err != nil {
			return err
		}
		if err := v.Struct(cfg); err != nil {
			return convertValidationError(err)
		}
	case "repo":
		var cfg RepoStep
		if err := decodeStepConfig(step, "repo", &cfg); err != nil {
			return err
		}
		if err := v.Struct(cfg); err != nil {
			return convertValidationError(err)
		}
	case "symlink":
		var cfg SymlinkStep
		if err := decodeStepConfig(step, "symlink", &cfg); err != nil {
			return err
		}
		if err := v.Struct(cfg); err != nil {
			return convertValidationError(err)
		}
	case "copy":
		var cfg CopyStep
		if err := decodeStepConfig(step, "copy", &cfg); err != nil {
			return err
		}
		if err := v.Struct(cfg); err != nil {
			return convertValidationError(err)
		}
	case "command":
		var cfg CommandStep
		if err := decodeStepConfig(step, "command", &cfg); err != nil {
			return err
		}
		if err := v.Struct(cfg); err != nil {
			return convertValidationError(err)
		}
	case "template":
		var cfg TemplateStep
		if err := decodeStepConfig(step, "template", &cfg); err != nil {
			return err
		}
		if err := v.Struct(cfg); err != nil {
			return convertValidationError(err)
		}
		if err := validateTemplateConfiguration(step.ID, cfg); err != nil {
			return err
		}
	case "line_in_file":
		var cfg LineInFileStep
		if err := decodeStepConfig(step, "line_in_file", &cfg); err != nil {
			return err
		}
		if err := v.Struct(cfg); err != nil {
			return convertValidationError(err)
		}
	default:
		return streamyerrors.NewValidationError(step.ID, fmt.Sprintf("unknown step type %q", step.Type), nil)
	}

	return nil
}

func decodeStepConfig(step Step, typeName string, dst any) error {
	if len(step.RawConfig()) == 0 {
		return streamyerrors.NewValidationError(step.ID, fmt.Sprintf("%s configuration is required", typeName), nil)
	}
	if err := step.DecodeConfig(dst); err != nil {
		return streamyerrors.NewValidationError(step.ID, fmt.Sprintf("%s configuration is invalid", typeName), err)
	}
	return nil
}
