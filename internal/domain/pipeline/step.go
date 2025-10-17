package pipeline

import (
	"fmt"
	"regexp"
	"sort"
)

var stepIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// StepType enumerates supported pipeline step types.
type StepType string

const (
	StepTypePackage    StepType = "package"
	StepTypeRepo       StepType = "repo"
	StepTypeSymlink    StepType = "symlink"
	StepTypeCopy       StepType = "copy"
	StepTypeCommand    StepType = "command"
	StepTypeTemplate   StepType = "template"
	StepTypeLineInFile StepType = "line_in_file"
)

var validStepTypes = []StepType{
	StepTypePackage,
	StepTypeRepo,
	StepTypeSymlink,
	StepTypeCopy,
	StepTypeCommand,
	StepTypeTemplate,
	StepTypeLineInFile,
}

// Step represents a single unit of work in a pipeline.
type Step struct {
	ID            string
	Name          string
	Type          StepType
	DependsOn     []string
	Enabled       bool
	VerifyTimeout int
	Config        map[string]interface{}
}

// Validate ensures the step satisfies all business rules.
func (s Step) Validate() error {
	if s.ID == "" {
		return newMissingFieldError("id")
	}
	if !stepIDPattern.MatchString(s.ID) {
		return newValidationError("step id must match ^[a-zA-Z0-9_-]+$", map[string]interface{}{"step_id": s.ID})
	}

	if s.Type == "" {
		return newMissingFieldError("type")
	}
	if !isValidStepType(s.Type) {
		return newTypeError(fmt.Sprintf("one of %v", validStepTypes), string(s.Type)).WithContext(map[string]interface{}{"step_id": s.ID})
	}

	if s.VerifyTimeout < 0 {
		return newValidationError("verify timeout must be non-negative", map[string]interface{}{"step_id": s.ID})
	}

	if s.Enabled && len(s.Config) == 0 {
		return newValidationError("enabled step requires configuration", map[string]interface{}{"step_id": s.ID})
	}

	return nil
}

// HasDependency returns true if the step depends on the provided identifier.
func (s Step) HasDependency(id string) bool {
	for _, dep := range s.DependsOn {
		if dep == id {
			return true
		}
	}
	return false
}

// SortedDependencies returns a sorted copy of the dependency list.
func (s Step) SortedDependencies() []string {
	deps := append([]string(nil), s.DependsOn...)
	sort.Strings(deps)
	return deps
}

func isValidStepType(st StepType) bool {
	for _, candidate := range validStepTypes {
		if candidate == st {
			return true
		}
	}
	return false
}
