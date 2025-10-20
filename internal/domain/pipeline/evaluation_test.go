package pipeline

import "testing"

func TestEvaluationResultDefaults(t *testing.T) {
	res := EvaluationResult{RequiresAction: true, CurrentState: "old", DesiredState: "new"}
	if !res.RequiresAction {
		t.Fatal("expected RequiresAction to be true")
	}
}
