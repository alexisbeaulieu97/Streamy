package main

import "testing"

func TestValidateRunOptions(t *testing.T) {
	cases := []struct {
		name    string
		opts    runOptions
		expects bool
	}{
		{name: "missing operands", opts: runOptions{}, expects: false},
		{name: "file only", opts: runOptions{ConfigPath: "pipeline.yaml"}, expects: true},
		{name: "registry only", opts: runOptions{PipelineID: "pipeline@1.0"}, expects: true},
		{name: "both provided", opts: runOptions{ConfigPath: "pipeline.yaml", PipelineID: "pipeline@1.0"}, expects: false},
		{name: "file with force", opts: runOptions{ConfigPath: "pipeline.yaml", Force: true}, expects: false},
		{name: "file with yes", opts: runOptions{ConfigPath: "pipeline.yaml", AssumeYes: true}, expects: false},
		{name: "registry yes without force", opts: runOptions{PipelineID: "pipeline@1.0", AssumeYes: true}, expects: false},
		{name: "registry with force/yes", opts: runOptions{PipelineID: "pipeline@1.0", Force: true, AssumeYes: true}, expects: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRunOptions(tc.opts)
			if tc.expects && err != nil {
				t.Fatalf("expected success, got error: %v", err)
			}

			if !tc.expects && err == nil {
				t.Fatalf("expected error, got success")
			}
		})
	}
}
