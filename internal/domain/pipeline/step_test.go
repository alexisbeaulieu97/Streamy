package pipeline

import (
	"errors"
	"testing"
)

func TestStepValidate(t *testing.T) {
	tests := []struct {
		name     string
		step     Step
		wantErr  bool
		wantCode ErrorCode
	}{
		{
			name: "valid step",
			step: Step{
				ID:        "install_packages",
				Type:      StepTypePackage,
				DependsOn: []string{"setup"},
				Enabled:   true,
				Config:    map[string]interface{}{"packages": []string{"git"}},
			},
		},
		{
			name:     "missing id",
			step:     Step{Type: StepTypeCommand},
			wantErr:  true,
			wantCode: ErrCodeMissing,
		},
		{
			name:     "enabled without config",
			step:     Step{ID: "enabled", Type: StepTypeCommand, Enabled: true},
			wantErr:  true,
			wantCode: ErrCodeValidation,
		},
		{
			name:     "invalid id pattern",
			step:     Step{ID: "invalid id", Type: StepTypeCommand},
			wantErr:  true,
			wantCode: ErrCodeValidation,
		},
		{
			name:     "invalid type",
			step:     Step{ID: "invalid", Type: StepType("unknown")},
			wantErr:  true,
			wantCode: ErrCodeType,
		},
		{
			name:     "negative timeout",
			step:     Step{ID: "bad_timeout", Type: StepTypeCommand, VerifyTimeout: -1},
			wantErr:  true,
			wantCode: ErrCodeValidation,
		},
	}

	for _, tt := range tests {
		err := tt.step.Validate()
		if (err != nil) != tt.wantErr {
			t.Fatalf("%s: Validate() error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
		if !tt.wantErr {
			continue
		}
		var domainErr *DomainError
		if !errors.As(err, &domainErr) {
			t.Fatalf("%s: expected DomainError, got %T", tt.name, err)
		}
		if domainErr.Code != tt.wantCode {
			t.Fatalf("%s: expected code %s, got %s", tt.name, tt.wantCode, domainErr.Code)
		}
	}
}

func TestStepHasDependency(t *testing.T) {
	step := Step{ID: "a", Type: StepTypeCommand, DependsOn: []string{"b", "c"}}
	if !step.HasDependency("b") {
		t.Fatal("expected dependency b")
	}
	if step.HasDependency("d") {
		t.Fatal("did not expect dependency d")
	}
}

func TestStepSortedDependencies(t *testing.T) {
	step := Step{ID: "a", Type: StepTypeCommand, DependsOn: []string{"c", "a", "b"}}
	deps := step.SortedDependencies()
	expected := []string{"a", "b", "c"}
	for i, want := range expected {
		if deps[i] != want {
			t.Fatalf("unexpected order at %d: got %s want %s", i, deps[i], want)
		}
	}
	if len(step.DependsOn) != 3 {
		t.Fatalf("SortedDependencies should not mutate original slice")
	}
}
