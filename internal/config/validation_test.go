package config

import (
	"fmt"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"

	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

func TestValidateValidation(t *testing.T) {
	t.Parallel()

	t.Run("validates command_exists type", func(t *testing.T) {
		t.Parallel()
		val := Validation{
			Type: "command_exists",
			CommandExists: &CommandExistsValidation{
				Command: "git",
			},
		}
		err := validateValidation(val, 0)
		require.NoError(t, err)
	})

	t.Run("rejects command_exists without command field", func(t *testing.T) {
		t.Parallel()
		val := Validation{
			Type:          "command_exists",
			CommandExists: nil,
		}
		err := validateValidation(val, 0)
		require.Error(t, err)

		var validationErr *streamyerrors.ValidationError
		require.ErrorAs(t, err, &validationErr)
		require.Contains(t, validationErr.Error(), "command")
	})

	t.Run("rejects command_exists with empty command", func(t *testing.T) {
		t.Parallel()
		val := Validation{
			Type: "command_exists",
			CommandExists: &CommandExistsValidation{
				Command: "",
			},
		}
		err := validateValidation(val, 0)
		require.Error(t, err)
	})

	t.Run("validates file_exists type", func(t *testing.T) {
		t.Parallel()
		val := Validation{
			Type: "file_exists",
			FileExists: &FileExistsValidation{
				Path: "/tmp/test.txt",
			},
		}
		err := validateValidation(val, 0)
		require.NoError(t, err)
	})

	t.Run("rejects file_exists without path field", func(t *testing.T) {
		t.Parallel()
		val := Validation{
			Type:       "file_exists",
			FileExists: nil,
		}
		err := validateValidation(val, 0)
		require.Error(t, err)

		var validationErr *streamyerrors.ValidationError
		require.ErrorAs(t, err, &validationErr)
		require.Contains(t, validationErr.Error(), "path")
	})

	t.Run("rejects file_exists with empty path", func(t *testing.T) {
		t.Parallel()
		val := Validation{
			Type: "file_exists",
			FileExists: &FileExistsValidation{
				Path: "",
			},
		}
		err := validateValidation(val, 0)
		require.Error(t, err)
	})

	t.Run("validates path_contains type", func(t *testing.T) {
		t.Parallel()
		val := Validation{
			Type: "path_contains",
			PathContains: &PathContainsValidation{
				File: "/tmp/config.txt",
				Text: "some text",
			},
		}
		err := validateValidation(val, 0)
		require.NoError(t, err)
	})

	t.Run("rejects path_contains without file and text fields", func(t *testing.T) {
		t.Parallel()
		val := Validation{
			Type:         "path_contains",
			PathContains: nil,
		}
		err := validateValidation(val, 0)
		require.Error(t, err)

		var validationErr *streamyerrors.ValidationError
		require.ErrorAs(t, err, &validationErr)
		require.Contains(t, validationErr.Error(), "file")
	})

	t.Run("rejects path_contains with empty file", func(t *testing.T) {
		t.Parallel()
		val := Validation{
			Type: "path_contains",
			PathContains: &PathContainsValidation{
				File: "",
				Text: "some text",
			},
		}
		err := validateValidation(val, 0)
		require.Error(t, err)
	})

	t.Run("rejects path_contains with empty text", func(t *testing.T) {
		t.Parallel()
		val := Validation{
			Type: "path_contains",
			PathContains: &PathContainsValidation{
				File: "/tmp/file",
				Text: "",
			},
		}
		err := validateValidation(val, 0)
		require.Error(t, err)
	})

	t.Run("rejects unknown validation type", func(t *testing.T) {
		t.Parallel()
		val := Validation{
			Type: "unknown_type",
		}
		err := validateValidation(val, 0)
		require.Error(t, err)
	})
}

func TestConvertValidationError(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for nil error", func(t *testing.T) {
		t.Parallel()
		result := convertValidationError(nil)
		require.Nil(t, result)
	})

	t.Run("converts validator.ValidationErrors", func(t *testing.T) {
		t.Parallel()
		// Create a validator error by validating an invalid struct
		type testStruct struct {
			Field string `validate:"required"`
		}

		v := validator.New()
		err := v.Struct(&testStruct{Field: ""})
		require.Error(t, err)

		result := convertValidationError(err)
		require.Error(t, result)

		var validationErr *streamyerrors.ValidationError
		require.ErrorAs(t, result, &validationErr)
	})

	t.Run("wraps non-validator errors", func(t *testing.T) {
		t.Parallel()
		originalErr := fmt.Errorf("some other error")
		result := convertValidationError(originalErr)
		require.Error(t, result)

		var validationErr *streamyerrors.ValidationError
		require.ErrorAs(t, result, &validationErr)
		require.Equal(t, "config", validationErr.Field)
	})
}

func TestFieldForValidation(t *testing.T) {
	t.Parallel()

	t.Run("formats type field", func(t *testing.T) {
		t.Parallel()
		result := fieldForValidation(0, "type")
		require.Equal(t, "validations[0].type", result)
	})

	t.Run("formats command field", func(t *testing.T) {
		t.Parallel()
		result := fieldForValidation(0, "command")
		require.Equal(t, "validations[0].command", result)
	})

	t.Run("formats path field", func(t *testing.T) {
		t.Parallel()
		result := fieldForValidation(1, "path")
		require.Equal(t, "validations[1].path", result)
	})

	t.Run("formats file field", func(t *testing.T) {
		t.Parallel()
		result := fieldForValidation(2, "file")
		require.Equal(t, "validations[2].file", result)
	})

	t.Run("uses correct index", func(t *testing.T) {
		t.Parallel()
		result := fieldForValidation(5, "command")
		require.Equal(t, "validations[5].command", result)
	})
}

func TestFieldForStep(t *testing.T) {
	t.Parallel()

	t.Run("formats step field with index", func(t *testing.T) {
		t.Parallel()
		result := fieldForStep(0, "id")
		require.Equal(t, "steps[0].id", result)
	})

	t.Run("formats various fields", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, "steps[1].type", fieldForStep(1, "type"))
		require.Equal(t, "steps[2].depends_on", fieldForStep(2, "depends_on"))
		require.Equal(t, "steps[3].command", fieldForStep(3, "command"))
	})
}

func TestYamlishFieldName(t *testing.T) {
	t.Parallel()

	t.Run("converts field error to lowercase path", func(t *testing.T) {
		t.Parallel()
		// Create a validator error
		type testStruct struct {
			FieldName string `validate:"required"`
		}

		v := validator.New()
		err := v.Struct(&testStruct{FieldName: ""})
		require.Error(t, err)

		if ves, ok := err.(validator.ValidationErrors); ok {
			require.Len(t, ves, 1)
			result := yamlishFieldName(ves[0])
			require.Contains(t, result, "fieldname")
		}
	})

	t.Run("handles nested struct paths", func(t *testing.T) {
		t.Parallel()
		type Inner struct {
			Value string `validate:"required"`
		}
		type Outer struct {
			Inner Inner `validate:"required"`
		}

		v := validator.New()
		err := v.Struct(&Outer{Inner: Inner{Value: ""}})
		require.Error(t, err)

		if ves, ok := err.(validator.ValidationErrors); ok {
			require.NotEmpty(t, ves)
			result := yamlishFieldName(ves[0])
			// Should be lowercase
			require.Equal(t, result, toLower(result))
		}
	})
}

func toLower(s string) string {
	result := ""
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			result += string(r + 32)
		} else {
			result += string(r)
		}
	}
	return result
}

