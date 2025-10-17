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
			{ID: "setup", Type: StepTypeCommand},
			{ID: "install", Type: StepTypePackage, DependsOn: []string{"setup"}},
		},
	}

	if err := pl.Validate(pipe); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecutionPlanValidateMissingStep(t *testing.T) {
	pl := ExecutionPlan{
		Levels: []ExecutionLevel{{Level: 0, StepIDs: []string{"setup"}}},
	}
	pipe := Pipeline{Name: "plan", Steps: []Step{{ID: "setup"}, {ID: "install"}}}

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
	pipe := Pipeline{Name: "plan", Steps: []Step{{ID: "setup", Type: StepTypeCommand}, {ID: "install", Type: StepTypePackage, DependsOn: []string{"setup"}}}}

	err := pl.Validate(pipe)
	if err == nil {
		t.Fatal("expected dependency order error")
	}
	var domainErr *DomainError
	if !errors.As(err, &domainErr) || domainErr.Code != ErrCodeDependency {
		t.Fatalf("expected dependency domain error, got %v", err)
	}
}
