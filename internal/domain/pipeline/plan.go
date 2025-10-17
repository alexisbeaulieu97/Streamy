package pipeline

import "fmt"

// ExecutionLevel groups steps that can execute in parallel.
type ExecutionLevel struct {
	Level   int
	StepIDs []string
}

// ExecutionPlan represents the ordered execution strategy for a pipeline.
type ExecutionPlan struct {
	Levels            []ExecutionLevel
	TotalSteps        int
	EstimatedDuration int
}

// Validate ensures the plan is coherent with the pipeline definition.
func (p ExecutionPlan) Validate(pipeline Pipeline) error {
	if len(p.Levels) == 0 {
		return newValidationError("execution plan must contain at least one level", nil)
	}

	seen := make(map[string]struct{})

	for _, level := range p.Levels {
		if len(level.StepIDs) == 0 {
			return newValidationError("execution level must contain steps", map[string]interface{}{"level": level.Level})
		}
		for _, id := range level.StepIDs {
			if _, ok := seen[id]; ok {
				return newDependencyError("step appears in multiple execution levels", map[string]interface{}{"step_id": id})
			}
			seen[id] = struct{}{}
		}
	}

	for _, step := range pipeline.Steps {
		if _, ok := seen[step.ID]; !ok {
			return newDependencyError("plan missing step", map[string]interface{}{"step_id": step.ID})
		}
	}

	levelIndex := make(map[string]int)
	for _, level := range p.Levels {
		for _, id := range level.StepIDs {
			levelIndex[id] = level.Level
		}
	}

	for _, step := range pipeline.Steps {
		for _, dep := range step.DependsOn {
			if levelIndex[dep] > levelIndex[step.ID] {
				return newDependencyError("dependency scheduled after dependent", map[string]interface{}{
					"step_id":       step.ID,
					"dependency_id": dep,
				})
			}
		}
	}

	return nil
}

// LevelForStep returns the level index for the provided step.
func (p ExecutionPlan) LevelForStep(stepID string) (int, error) {
	for _, level := range p.Levels {
		for _, id := range level.StepIDs {
			if id == stepID {
				return level.Level, nil
			}
		}
	}
	return 0, fmt.Errorf("step %s not present in execution plan", stepID)
}
