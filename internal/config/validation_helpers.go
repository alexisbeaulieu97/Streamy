package config

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"

	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

// convertValidationError normalizes validator errors into Streamy validation errors.
func convertValidationError(err error) error {
	if err == nil {
		return nil
	}

	if ves, ok := err.(validator.ValidationErrors); ok {
		ve := ves[0]
		field := yamlishFieldName(ve)
		msg := fmt.Sprintf("%s failed validation for tag '%s'", field, ve.Tag())
		return streamyerrors.NewValidationError(field, msg, err)
	}

	return streamyerrors.NewValidationError("config", err.Error(), err)
}

func yamlishFieldName(fe validator.FieldError) string {
	ns := fe.StructNamespace()
	parts := strings.Split(ns, ".")
	var lowered []string
	for _, part := range parts {
		lowered = append(lowered, strings.ToLower(part))
	}
	return strings.Join(lowered, ".")
}

func fieldForStep(index int, field string) string {
	return fmt.Sprintf("steps[%d].%s", index, field)
}

func fieldForValidation(index int, field string) string {
	if field == "type" {
		return fmt.Sprintf("validations[%d].type", index)
	}
	return fmt.Sprintf("validations[%d].%s", index, field)
}
