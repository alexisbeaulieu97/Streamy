package pipeline

import (
	"errors"
	"testing"
)

func TestExecutionPlanValidate(t *testing.T) {
	pl := ExecutionPlan{
		Levels: []ExecutionLevel{
			{Level: 0, StepIDs: []string{"setup"}},
			{Level: 1, StepIDs: []string{"install"}},
		},
		TotalSteps: 2,
	}

	pipe := Pipeline{
		Name: "plan",
		Steps: []Step{
			{ID: "setup", Type: StepTypeCommand, Enabled: true},
			{ID: "install", Type: StepTypePackage, DependsOn: []string{"setup"}, Enabled: true},
		},
	}

	if err := pl.Validate(pipe); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecutionPlanLevelForStep(t *testing.T) {
	pl := ExecutionPlan{
		Levels: []ExecutionLevel{
			{Level: 0, StepIDs: []string{"setup", "prepare"}},
			{Level: 1, StepIDs: []string{"install"}},
		},
	}

	level, err := pl.LevelForStep("install")
	if err != nil {
		t.Fatalf("expected level lookup success, got %v", err)
	}

	if level != 1 {
		t.Fatalf("expected level 1, got %d", level)
	}

	if _, err := pl.LevelForStep("missing"); err == nil {
		t.Fatal("expected error for missing step")
	} else {
		var domainErr *DomainError
		if !errors.As(err, &domainErr) || domainErr.Code != ErrCodeDependency {
			t.Fatalf("expected dependency error, got %v", err)
		}
	}
}

func TestExecutionPlanValidateMissingStep(t *testing.T) {
	pl := ExecutionPlan{
		Levels: []ExecutionLevel{{Level: 0, StepIDs: []string{"setup"}}},
	}
	pipe := Pipeline{Name: "plan", Steps: []Step{{ID: "setup", Enabled: true}, {ID: "install", Enabled: true}}}

	err := pl.Validate(pipe)
	if err == nil {
		t.Fatal("expected missing step error")
	}

	var domainErr *DomainError
	if !errors.As(err, &domainErr) || domainErr.Code != ErrCodeDependency {
		t.Fatalf("expected dependency domain error, got %v", err)
	}
}

func TestExecutionPlanValidateDependencyOrder(t *testing.T) {
	pl := ExecutionPlan{
		Levels: []ExecutionLevel{
			{Level: 0, StepIDs: []string{"install"}},
			{Level: 1, StepIDs: []string{"setup"}},
		},
	}
	pipe := Pipeline{
		Name: "plan",
		Steps: []Step{
			{ID: "setup", Type: StepTypeCommand, Enabled: true},
			{ID: "install", Type: StepTypePackage, DependsOn: []string{"setup"}, Enabled: true},
		},
	}

	err := pl.Validate(pipe)
	if err == nil {
		t.Fatal("expected dependency order error")
	}

	var domainErr *DomainError
	if !errors.As(err, &domainErr) || domainErr.Code != ErrCodeDependency {
		t.Fatalf("expected dependency domain error, got %v", err)
	}
}

func TestExecutionPlanValidateDependencySameLevel(t *testing.T) {
	pl := ExecutionPlan{
		Levels: []ExecutionLevel{
			{Level: 0, StepIDs: []string{"setup", "install"}},
		},
	}
	pipe := Pipeline{
		Name: "plan",
		Steps: []Step{
			{ID: "setup", Type: StepTypeCommand, Enabled: true},
			{ID: "install", Type: StepTypePackage, DependsOn: []string{"setup"}, Enabled: true},
		},
	}

	err := pl.Validate(pipe)
	if err == nil {
		t.Fatal("expected dependency order error for same-level dependency")
	}

	var domainErr *DomainError
	if !errors.As(err, &domainErr) || domainErr.Code != ErrCodeDependency {
		t.Fatalf("expected dependency domain error, got %v", err)
	}

	if domainErr.Context["step_level"] != 0 || domainErr.Context["dependency_level"] != 0 {
		t.Fatalf("expected level context, got %v", domainErr.Context)
	}
}

func TestExecutionPlanValidateEnsuresLevelsPresent(t *testing.T) {
	pl := ExecutionPlan{}
	pipe := Pipeline{Name: "plan", Steps: []Step{{ID: "setup", Enabled: true}}}

	err := pl.Validate(pipe)
	if err == nil {
		t.Fatal("expected validation error when no levels provided")
	}

	var domainErr *DomainError
	if !errors.As(err, &domainErr) || domainErr.Code != ErrCodeValidation {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestExecutionPlanValidateSkipsDisabledSteps(t *testing.T) {
	pl := ExecutionPlan{
		Levels: []ExecutionLevel{
			{Level: 0, StepIDs: []string{"install"}},
		},
	}
	pipe := Pipeline{
		Name: "plan",
		Steps: []Step{
			{ID: "setup", Type: StepTypeCommand, Enabled: false},
			{ID: "install", Type: StepTypePackage, Enabled: true, DependsOn: []string{"setup"}},
		},
	}

	if err := pl.Validate(pipe); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
