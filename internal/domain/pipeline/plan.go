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
	if err := p.ensureLevelsPresent(); err != nil {
		return err
	}

	seen, err := p.collectStepIDs()
	if err != nil {
		return err
	}

	stepByID := buildStepMap(pipeline.Steps)

	if err := ensureStepsAreKnown(seen, stepByID); err != nil {
		return err
	}

	if err := ensureEnabledStepsCovered(pipeline.Steps, seen); err != nil {
		return err
	}

	levelIndex := buildLevelIndex(p.Levels)

	return validateStepDependencies(pipeline.Steps, levelIndex, stepByID)
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

func (p ExecutionPlan) ensureLevelsPresent() error {
	if len(p.Levels) == 0 {
		return NewValidationError("execution plan must contain at least one level", nil)
	}

	return nil
}

func (p ExecutionPlan) collectStepIDs() (map[string]struct{}, error) {
	seen := make(map[string]struct{})

	for _, level := range p.Levels {
		if len(level.StepIDs) == 0 {
			return nil, NewValidationError("execution level must contain steps", map[string]interface{}{"level": level.Level})
		}

		for _, id := range level.StepIDs {
			if _, ok := seen[id]; ok {
				return nil, NewValidationError("step appears in multiple execution levels", map[string]interface{}{"step_id": id})
			}

			seen[id] = struct{}{}
		}
	}

	return seen, nil
}

func buildStepMap(steps []Step) map[string]Step {
	stepByID := make(map[string]Step, len(steps))
	for _, step := range steps {
		stepByID[step.ID] = step
	}

	return stepByID
}

func ensureStepsAreKnown(seen map[string]struct{}, stepByID map[string]Step) error {
	for id := range seen {
		step, ok := stepByID[id]
		if !ok {
			return NewDependencyError("execution plan references unknown step", map[string]interface{}{"step_id": id, "reason": "unknown"})
		}

		if !step.Enabled {
			return NewDependencyError("execution plan references disabled step", map[string]interface{}{"step_id": id, "reason": "disabled"})
		}
	}

	return nil
}

func ensureEnabledStepsCovered(steps []Step, seen map[string]struct{}) error {
	for _, step := range steps {
		if !step.Enabled {
			continue
		}

		if _, ok := seen[step.ID]; !ok {
			return NewDependencyError("plan missing step", map[string]interface{}{"step_id": step.ID})
		}
	}

	return nil
}

func buildLevelIndex(levels []ExecutionLevel) map[string]int {
	levelIndex := make(map[string]int)

	for _, level := range levels {
		for _, id := range level.StepIDs {
			levelIndex[id] = level.Level
		}
	}

	return levelIndex
}

func validateStepDependencies(steps []Step, levelIndex map[string]int, stepByID map[string]Step) error {
	for _, step := range steps {
		if !step.Enabled {
			continue
		}

		stepLevel, ok := levelIndex[step.ID]
		if !ok {
			return NewDependencyError("plan missing step", map[string]interface{}{"step_id": step.ID})
		}

		for _, dep := range step.DependsOn {
			dStep, depExists := stepByID[dep]
			if depExists && !dStep.Enabled {
				continue
			}

			depLevel, ok := levelIndex[dep]
			if !ok {
				return NewDependencyError("plan missing dependency", map[string]interface{}{
					"step_id":       step.ID,
					"dependency_id": dep,
				})
			}

			if depLevel >= stepLevel {
				return NewDependencyError("dependency scheduled after dependent", map[string]interface{}{
					"step_id":          step.ID,
					"dependency_id":    dep,
					"step_level":       stepLevel,
					"dependency_level": depLevel,
				})
			}
		}
	}

	return nil
}
