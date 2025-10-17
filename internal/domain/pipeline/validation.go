package pipeline

import (
	"fmt"
	"strings"
)

// ValidationType enumerates supported validation kinds.
type ValidationType string

const (
	ValidationCommandExists ValidationType = "command_exists"
	ValidationFileExists    ValidationType = "file_exists"
	ValidationPathContains  ValidationType = "path_contains"
)

var supportedValidationTypes = []ValidationType{
	ValidationCommandExists,
	ValidationFileExists,
	ValidationPathContains,
}

// Validation represents a post-execution validation definition.
type Validation struct {
	Type   ValidationType
	Config map[string]interface{}
}

// Validate ensures the validation definition is well formed.
func (v Validation) Validate() error {
	if v.Type == "" {
		return newMissingFieldError("validation.type")
	}

	if !isSupportedValidationType(v.Type) {
		return newTypeError(fmt.Sprintf("one of %v", supportedValidationTypes), string(v.Type)).
			WithContext(map[string]interface{}{"validation_type": v.Type})
	}

	switch v.Type {
	case ValidationCommandExists:
		command, err := v.stringConfig("command")
		if err != nil {
			return err
		}
		if strings.TrimSpace(command) == "" {
			return newValidationError("validation command must be non-empty", map[string]interface{}{
				"validation_type": v.Type,
				"field":           "command",
			})
		}
	case ValidationFileExists:
		if _, err := v.stringConfig("path"); err != nil {
			return err
		}
	case ValidationPathContains:
		if _, err := v.stringConfig("file"); err != nil {
			return err
		}
		if _, err := v.stringConfig("text"); err != nil {
			return err
		}
	default:
		return newTypeError(fmt.Sprintf("one of %v", supportedValidationTypes), string(v.Type)).
			WithContext(map[string]interface{}{"validation_type": v.Type})
	}

	return nil
}

func (v Validation) stringConfig(key string) (string, error) {
	if v.Config == nil {
		return "", newValidationError("validation config is required", map[string]interface{}{
			"validation_type": v.Type,
		})
	}

	raw, ok := v.Config[key]
	if !ok {
		return "", newMissingFieldError(fmt.Sprintf("validation.config.%s", key)).
			WithContext(map[string]interface{}{
				"validation_type": v.Type,
			})
	}

	value, ok := raw.(string)
	if !ok {
		return "", newValidationError("validation config field must be a string", map[string]interface{}{
			"validation_type": v.Type,
			"field":           key,
			"actual_type":     fmt.Sprintf("%T", raw),
		})
	}

	return value, nil
}

func isSupportedValidationType(t ValidationType) bool {
	for _, candidate := range supportedValidationTypes {
		if candidate == t {
			return true
		}
	}
	return false
}
