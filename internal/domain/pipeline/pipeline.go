package pipeline

import (
	"fmt"
	"maps"
	"slices"
)

// Pipeline represents a complete pipeline configuration.
type Pipeline struct {
	Version     string
	Name        string
	Description string
	Settings    Settings
	Steps       []Step
	Validations []Validation
}

// Validate ensures the pipeline satisfies all invariants.
func (p Pipeline) Validate() error {
	if p.Name == "" {
		return NewMissingFieldError("name")
	}

	if len(p.Steps) == 0 {
		return NewValidationError("pipeline requires at least one step", nil)
	}

	seen := make(map[string]struct{}, len(p.Steps))
	for _, step := range p.Steps {
		if err := step.Validate(); err != nil {
			return err
		}

		if _, ok := seen[step.ID]; ok {
			return NewDuplicateError(step.ID)
		}

		seen[step.ID] = struct{}{}
	}

	if err := p.ValidateDependencies(); err != nil {
		return err
	}

	return nil
}

// ValidateDependencies ensures all dependencies exist and no cycles occur.
func (p Pipeline) ValidateDependencies() error {
	lookup := make(map[string]Step, len(p.Steps))
	for _, step := range p.Steps {
		lookup[step.ID] = step
	}

	for _, step := range p.Steps {
		for _, dep := range step.DependsOn {
			if dep == step.ID {
				return NewDependencyError("step cannot depend on itself", map[string]interface{}{"step_id": step.ID})
			}

			if _, ok := lookup[dep]; !ok {
				return NewDependencyError("dependency not found", map[string]interface{}{"step_id": step.ID, "missing_dependency": dep})
			}
		}
	}

	visited := make(map[string]bool, len(p.Steps))
	stack := make(map[string]bool, len(p.Steps))

	var (
		path   []string
		detect func(string) *DomainError
	)

	detect = func(id string) *DomainError {
		visited[id] = true
		stack[id] = true
		path = append(path, id)

		for _, dep := range lookup[id].DependsOn {
			if !visited[dep] {
				if err := detect(dep); err != nil {
					return err
				}
			} else if stack[dep] {
				cycle := slices.Clone(path)
				cycle = append(cycle, dep)

				return NewCycleError(cycle)
			}
		}

		stack[id] = false
		path = path[:len(path)-1]

		return nil
	}

	for _, step := range p.Steps {
		if !visited[step.ID] {
			if err := detect(step.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetStep retrieves a step by identifier.
func (p Pipeline) GetStep(id string) (*Step, error) {
	for i := range p.Steps {
		if p.Steps[i].ID == id {
			stepCopy := p.Steps[i]
			return &stepCopy, nil
		}
	}

	return nil, NewNotFoundError("step", map[string]interface{}{"step_id": id})
}

// EffectiveSettings exposes the pipeline settings after applying defaults.
func (p Pipeline) EffectiveSettings() Settings {
	return p.Settings.ApplyDefaults()
}

// Clone returns a defensive copy of the pipeline.
func (p Pipeline) Clone() Pipeline {
	steps := make([]Step, len(p.Steps))
	for i, step := range p.Steps {
		cloned := step
		if len(step.DependsOn) > 0 {
			cloned.DependsOn = slices.Clone(step.DependsOn)
		}

		if len(step.Config) > 0 {
			cloned.Config = maps.Clone(step.Config)
		}

		steps[i] = cloned
	}

	validations := make([]Validation, len(p.Validations))
	for i, val := range p.Validations {
		validations[i] = Validation{Type: val.Type, Config: maps.Clone(val.Config)}
	}

	return Pipeline{
		Version:     p.Version,
		Name:        p.Name,
		Description: p.Description,
		Settings:    p.Settings.Clone(),
		Steps:       steps,
		Validations: validations,
	}
}

// MustStep panics if the step does not exist. Intended for internal helpers
// where missing steps indicate programmer error.
func (p Pipeline) MustStep(id string) Step {
	step, err := p.GetStep(id)
	if err != nil {
		panic(fmt.Sprintf("step %s not found", id))
	}

	return *step
}
