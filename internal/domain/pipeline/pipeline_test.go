package pipeline

import (
	"errors"
	"testing"
)

func TestPipelineValidate(t *testing.T) {
	p := Pipeline{
		Name: "test",
		Steps: []Step{
			{ID: "setup", Type: StepTypeCommand},
			{ID: "install", Type: StepTypePackage, DependsOn: []string{"setup"}},
		},
	}

	if err := p.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPipelineValidateDuplicateStep(t *testing.T) {
	p := Pipeline{
		Name: "invalid",
		Steps: []Step{
			{ID: "dup", Type: StepTypeCommand},
			{ID: "dup", Type: StepTypePackage},
		},
	}

	err := p.Validate()
	if err == nil {
		t.Fatal("expected duplicate error")
	}
	var domainErr *DomainError
	if !errors.As(err, &domainErr) || domainErr.Code != ErrCodeDuplicate {
		t.Fatalf("expected duplicate domain error, got %v", err)
	}
}

func TestPipelineValidateDependencies(t *testing.T) {
	p := Pipeline{
		Name: "invalid",
		Steps: []Step{
			{ID: "a", Type: StepTypeCommand, DependsOn: []string{"missing"}},
		},
	}

	err := p.Validate()
	if err == nil {
		t.Fatal("expected missing dependency error")
	}
	var domainErr *DomainError
	if !errors.As(err, &domainErr) || domainErr.Code != ErrCodeDependency {
		t.Fatalf("expected dependency domain error, got %v", err)
	}
}

func TestPipelineValidateDependencyCycle(t *testing.T) {
	p := Pipeline{
		Name: "cycle",
		Steps: []Step{
			{ID: "a", Type: StepTypeCommand, DependsOn: []string{"b"}},
			{ID: "b", Type: StepTypeCommand, DependsOn: []string{"a"}},
		},
	}

	err := p.Validate()
	if err == nil {
		t.Fatal("expected cycle error")
	}
	var domainErr *DomainError
	if !errors.As(err, &domainErr) || domainErr.Code != ErrCodeCycle {
		t.Fatalf("expected cycle error code, got %v", err)
	}
}

func TestPipelineGetStep(t *testing.T) {
	p := Pipeline{
		Name:  "steps",
		Steps: []Step{{ID: "a", Type: StepTypeCommand}},
	}

	step, err := p.GetStep("a")
	if err != nil || step == nil || step.ID != "a" {
		t.Fatalf("expected step a, got %v, %v", step, err)
	}

	if _, err = p.GetStep("missing"); err == nil {
		t.Fatal("expected not found error")
	}
	var domainErr *DomainError
	if !errors.As(err, &domainErr) || domainErr.Code != ErrCodeNotFound {
		t.Fatalf("expected not found domain error, got %v", err)
	}
}

func TestPipelineMustStep(t *testing.T) {
	p := Pipeline{Steps: []Step{{ID: "x", Type: StepTypeCommand}}}
	if step := p.MustStep("x"); step.ID != "x" {
		t.Fatalf("unexpected step %v", step)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for missing step")
		}
	}()
	_ = p.MustStep("missing")
}

func TestPipelineClone(t *testing.T) {
	p := Pipeline{
		Name:     "original",
		Settings: Settings{Parallel: 2},
		Steps:    []Step{{ID: "a", Type: StepTypeCommand}},
		Validations: []Validation{{
			Type:   ValidationCommandExists,
			Config: map[string]interface{}{"command": "git"},
		}},
	}

	clone := p.Clone()
	clone.Steps[0].ID = "b"
	clone.Validations[0].Config["command"] = "hg"

	if p.Steps[0].ID != "a" {
		t.Fatal("expected original steps unchanged")
	}
	if p.Validations[0].Config["command"] != "git" {
		t.Fatal("expected original validation config unchanged")
	}
}

func TestPipelineEffectiveSettings(t *testing.T) {
	p := Pipeline{Settings: Settings{Parallel: 0, Timeout: 0}}
	eff := p.EffectiveSettings()
	if eff.Parallel != 4 || eff.Timeout != 300 {
		t.Fatalf("expected defaults applied, got %+v", eff)
	}
}
