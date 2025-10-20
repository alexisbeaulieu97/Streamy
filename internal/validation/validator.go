package validation

import (
	"context"
	"fmt"
	"strings"

	domain "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

// RunValidations executes the provided validations and returns their results.
func RunValidations(ctx context.Context, validations []domain.Validation) ([]Result, error) {
	results := make([]Result, 0, len(validations))

	var failedMessages []string

	for _, val := range validations {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return results, fmt.Errorf("validation cancelled: %w", ctxErr)
		}

		result := Result{Validation: val}

		if err := val.Validate(); err != nil {
			result.Passed = false
			result.Message = err.Error()
			result.Error = err
			failedMessages = append(failedMessages, err.Error())
			results = append(results, result)

			continue
		}

		var execErr error

		switch val.Type {
		case domain.ValidationCommandExists:
			command, err := stringConfigValue(val.Config, "command")
			if err != nil {
				execErr = streamyerrors.NewValidationError("validation.command_exists", err.Error(), nil)
			} else {
				execErr = CheckCommandExists(command)
			}
		case domain.ValidationFileExists:
			path, err := stringConfigValue(val.Config, "path")
			if err != nil {
				execErr = streamyerrors.NewValidationError("validation.file_exists", err.Error(), nil)
			} else {
				execErr = CheckFileExists(path)
			}
		case domain.ValidationPathContains:
			file, fileErr := stringConfigValue(val.Config, "file")

			text, textErr := stringConfigValue(val.Config, "text")
			switch {
			case fileErr != nil:
				execErr = streamyerrors.NewValidationError("validation.path_contains", fileErr.Error(), nil)
			case textErr != nil:
				execErr = streamyerrors.NewValidationError("validation.path_contains", textErr.Error(), nil)
			default:
				execErr = CheckPathContains(file, text)
			}
		default:
			execErr = streamyerrors.NewValidationError("validation.type", fmt.Sprintf("unknown validation type %q", val.Type), nil)
		}

		if execErr != nil {
			result.Passed = false
			result.Message = execErr.Error()
			result.Error = execErr
			failedMessages = append(failedMessages, execErr.Error())
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

func stringConfigValue(cfg map[string]any, key string) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("configuration missing key %q", key)
	}

	raw, ok := cfg[key]
	if !ok {
		return "", fmt.Errorf("configuration missing key %q", key)
	}

	value, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("configuration key %q must be a string", key)
	}

	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("configuration key %q must be non-empty", key)
	}

	return value, nil
}
