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
		if step.Package == nil {
			return streamyerrors.NewValidationError(step.ID, "package configuration is required", nil)
		}
		if err := v.Struct(step.Package); err != nil {
			return convertValidationError(err)
		}
	case "repo":
		if step.Repo == nil {
			return streamyerrors.NewValidationError(step.ID, "repo configuration is required", nil)
		}
		if err := v.Struct(step.Repo); err != nil {
			return convertValidationError(err)
		}
	case "symlink":
		if step.Symlink == nil {
			return streamyerrors.NewValidationError(step.ID, "symlink configuration is required", nil)
		}
		if err := v.Struct(step.Symlink); err != nil {
			return convertValidationError(err)
		}
	case "copy":
		if step.Copy == nil {
			return streamyerrors.NewValidationError(step.ID, "copy configuration is required", nil)
		}
		if err := v.Struct(step.Copy); err != nil {
			return convertValidationError(err)
		}
	case "command":
		if step.Command == nil {
			return streamyerrors.NewValidationError(step.ID, "command configuration is required", nil)
		}
		if err := v.Struct(step.Command); err != nil {
			return convertValidationError(err)
		}
	case "template":
		if step.Template == nil {
			return streamyerrors.NewValidationError(step.ID, "template configuration is required", nil)
		}
		if err := v.Struct(step.Template); err != nil {
			return convertValidationError(err)
		}
		if err := validateTemplateConfiguration(step); err != nil {
			return err
		}
	case "line_in_file":
		if step.LineInFile == nil {
			return streamyerrors.NewValidationError(step.ID, "line_in_file configuration is required", nil)
		}
		if err := v.Struct(step.LineInFile); err != nil {
			return convertValidationError(err)
		}
	default:
		return streamyerrors.NewValidationError(step.ID, fmt.Sprintf("unknown step type %q", step.Type), nil)
	}

	return nil
}
