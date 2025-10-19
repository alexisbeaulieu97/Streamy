package pipeline

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
		return NewValidationError("execution plan must contain at least one level", nil)
	}

	seen := make(map[string]struct{})

	for _, level := range p.Levels {
		if len(level.StepIDs) == 0 {
			return NewValidationError("execution level must contain steps", map[string]interface{}{"level": level.Level})
		}
		for _, id := range level.StepIDs {
			if _, ok := seen[id]; ok {
				return NewValidationError("step appears in multiple execution levels", map[string]interface{}{"step_id": id})
			}
			seen[id] = struct{}{}
		}
	}

	stepByID := make(map[string]Step, len(pipeline.Steps))
	for _, step := range pipeline.Steps {
		stepByID[step.ID] = step
	}

	for _, step := range pipeline.Steps {
		if !step.Enabled {
			continue
		}
		if _, ok := seen[step.ID]; !ok {
			return NewDependencyError("plan missing step", map[string]interface{}{"step_id": step.ID})
		}
	}

	levelIndex := make(map[string]int)
	for _, level := range p.Levels {
		for _, id := range level.StepIDs {
			levelIndex[id] = level.Level
		}
	}

	for _, step := range pipeline.Steps {
		if !step.Enabled {
			continue
		}
		stepLevel, ok := levelIndex[step.ID]
		if !ok {
			// This should have been caught earlier, but guard defensively.
			return NewDependencyError("plan missing step", map[string]interface{}{"step_id": step.ID})
		}
		for _, dep := range step.DependsOn {
			if depStep, ok := stepByID[dep]; ok && !depStep.Enabled {
				continue
			}
			depLevel, ok := levelIndex[dep]
			if !ok {
				return NewDependencyError("plan missing dependency", map[string]interface{}{
					"step_id":       step.ID,
					"dependency_id": dep,
				})
			}
			if depLevel > stepLevel {
				return NewDependencyError("dependency scheduled after dependent", map[string]interface{}{
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
	return 0, NewDependencyError("step not present in execution plan", map[string]interface{}{"step_id": stepID})
}
