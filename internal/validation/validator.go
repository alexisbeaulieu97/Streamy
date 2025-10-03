package validation

import (
	"context"
	"fmt"
	"strings"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

// RunValidations executes the provided validations and returns their results.
func RunValidations(ctx context.Context, validations []config.Validation) ([]ValidationResult, error) {
	results := make([]ValidationResult, 0, len(validations))
	var failedMessages []string

	for _, val := range validations {
		result := ValidationResult{Validation: val}

		var err error
		switch val.Type {
		case "command_exists":
			if val.CommandExists == nil {
				err = streamyerrors.NewValidationError("validation.command_exists", "configuration missing", nil)
			} else {
				err = CheckCommandExists(val.CommandExists.Command)
			}
		case "file_exists":
			if val.FileExists == nil {
				err = streamyerrors.NewValidationError("validation.file_exists", "configuration missing", nil)
			} else {
				err = CheckFileExists(val.FileExists.Path)
			}
		case "path_contains":
			if val.PathContains == nil {
				err = streamyerrors.NewValidationError("validation.path_contains", "configuration missing", nil)
			} else {
				err = CheckPathContains(val.PathContains.File, val.PathContains.Text)
			}
		default:
			err = streamyerrors.NewValidationError("validation.type", fmt.Sprintf("unknown validation type %q", val.Type), nil)
		}

		if err != nil {
			result.Passed = false
			result.Message = err.Error()
			result.Error = err
			failedMessages = append(failedMessages, err.Error())
		} else {
			result.Passed = true
			result.Message = "passed"
		}

		results = append(results, result)
	}

	if len(failedMessages) > 0 {
		combined := strings.Join(failedMessages, "; ")
		return results, fmt.Errorf("validations failed: %s", combined)
	}

	return results, nil
}
