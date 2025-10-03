package config

import (
	"regexp"
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	validatorOnce sync.Once
	validateInst  *validator.Validate

	semverPattern   = regexp.MustCompile(`^\d+\.\d+(?:\.\d+)?(?:-[0-9A-Za-z-.]+)?(?:\+[0-9A-Za-z-.]+)?$`)
	stepIDPattern   = regexp.MustCompile(`^[a-z0-9_]+$`)
	stepTypes       = map[string]struct{}{"package": {}, "repo": {}, "symlink": {}, "copy": {}, "command": {}}
	validationTypes = map[string]struct{}{"command_exists": {}, "file_exists": {}, "path_contains": {}}
)

// validatorInstance configures and returns the shared validator instance used across the config package.
func validatorInstance() *validator.Validate {
	validatorOnce.Do(func() {
		v := validator.New()

		_ = v.RegisterValidation("semver", func(fl validator.FieldLevel) bool {
			return semverPattern.MatchString(fl.Field().String())
		})

		_ = v.RegisterValidation("step_id", func(fl validator.FieldLevel) bool {
			return stepIDPattern.MatchString(fl.Field().String())
		})

		validateInst = v
	})

	return validateInst
}
