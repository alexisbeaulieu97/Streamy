package config

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"

	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

var (
	validatorOnce sync.Once
	validateInst  *validator.Validate

	semverPattern   = regexp.MustCompile(`^\d+\.\d+(?:\.\d+)?(?:-[0-9A-Za-z-.]+)?(?:\+[0-9A-Za-z-.]+)?$`)
	stepIDPattern   = regexp.MustCompile(`^[a-z0-9_]+$`)
	stepTypes       = map[string]struct{}{"package": {}, "repo": {}, "symlink": {}, "copy": {}, "command": {}}
	validationTypes = map[string]struct{}{"command_exists": {}, "file_exists": {}, "path_contains": {}}
)

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

// ValidateConfig performs schema and cross-field validation on the configuration.
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

// ValidateStep validates a single step independent of other configuration properties.
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
	default:
		return streamyerrors.NewValidationError(step.ID, fmt.Sprintf("unknown step type %q", step.Type), nil)
	}

	return nil
}

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

func detectCycle(steps []Step) []string {
	enabled := make(map[string]bool, len(steps))
	for _, step := range steps {
		if step.Enabled {
			enabled[step.ID] = true
		}
	}

	graph := make(map[string][]string, len(enabled))
	for _, step := range steps {
		if !enabled[step.ID] {
			continue
		}
		deps := make([]string, 0, len(step.DependsOn))
		for _, dep := range step.DependsOn {
			if enabled[dep] {
				deps = append(deps, dep)
			}
		}
		graph[step.ID] = deps
	}

	visiting := make(map[string]bool, len(steps))
	visited := make(map[string]bool, len(steps))
	var stack []string

	var cycle []string
	var dfs func(string) bool
	dfs = func(node string) bool {
		visiting[node] = true
		stack = append(stack, node)

		for _, dep := range graph[node] {
			if !visited[dep] {
				if visiting[dep] {
					idx := indexOf(stack, dep)
					if idx >= 0 {
						cycle = append([]string{}, stack[idx:]...)
						cycle = append(cycle, dep)
					}
					return true
				}
				if dfs(dep) {
					return true
				}
			}
		}

		visiting[node] = false
		visited[node] = true
		stack = stack[:len(stack)-1]
		return false
	}

	// Ensure deterministic order by sorting IDs.
	ids := make([]string, 0, len(enabled))
	for id := range enabled {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		if visited[id] {
			continue
		}
		if dfs(id) {
			break
		}
	}

	return cycle
}

func indexOf(slice []string, target string) int {
	for i, v := range slice {
		if v == target {
			return i
		}
	}
	return -1
}
