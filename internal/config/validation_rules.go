package config

import (
	"fmt"

	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

// validateValidation checks a single post-execution validation entry.
func validateValidation(val Validation, index int) error {
	v := validatorInstance()
	if err := v.Struct(val); err != nil {
		return convertValidationError(err)
	}

	switch val.Type {
	case "command_exists":
		if val.CommandExists == nil {
			return streamyerrors.NewValidationError(fieldForValidation(index, "command"), "command is required", nil)
		}
		if err := v.Struct(val.CommandExists); err != nil {
			return convertValidationError(err)
		}
	case "file_exists":
		if val.FileExists == nil {
			return streamyerrors.NewValidationError(fieldForValidation(index, "path"), "path is required", nil)
		}
		if err := v.Struct(val.FileExists); err != nil {
			return convertValidationError(err)
		}
	case "path_contains":
		if val.PathContains == nil {
			return streamyerrors.NewValidationError(fieldForValidation(index, "file"), "file and text are required", nil)
		}
		if err := v.Struct(val.PathContains); err != nil {
			return convertValidationError(err)
		}
	default:
		if _, ok := validationTypes[val.Type]; !ok {
			return streamyerrors.NewValidationError(fieldForValidation(index, "type"), fmt.Sprintf("unknown validation type %q", val.Type), nil)
		}
	}

	return nil
}
