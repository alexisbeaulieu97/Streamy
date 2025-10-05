package engine

import (
	"context"
	"fmt"
	"testing"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
)

func BenchmarkBuildDAGLarge(b *testing.B) {
	steps := generateLinearSteps(2000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := BuildDAG(steps); err != nil {
			b.Fatalf("build dag: %v", err)
		}
	}
}

func BenchmarkExecuteDryRunLarge(b *testing.B) {
	plugin.ResetRegistry()
	if err := plugin.RegisterPlugin("command", &benchmarkPlugin{}); err != nil {
		b.Fatalf("register plugin: %v", err)
	}

	cfg := &config.Config{
		Version: "1.0",
		Name:    "bench",
		Settings: config.Settings{
			Parallel: 16,
		},
		Steps: generateLinearSteps(1000),
	}

	graph, err := BuildDAG(cfg.Steps)
	if err != nil {
		b.Fatalf("build dag: %v", err)
	}
	plan, err := GeneratePlan(graph)
	if err != nil {
		b.Fatalf("generate plan: %v", err)
	}

	execCtx := &ExecutionContext{
		Config:     cfg,
		DryRun:     true,
		WorkerPool: make(chan struct{}, cfg.Settings.Parallel),
		Results:    make(map[string]*model.StepResult),
		Context:    context.Background(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		execCtx.Results = make(map[string]*model.StepResult)
		if _, err := Execute(execCtx, plan); err != nil {
			b.Fatalf("execute: %v", err)
		}
	}
}

func generateLinearSteps(count int) []config.Step {
	steps := make([]config.Step, count)
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("step_%d", i)
		step := config.Step{
			ID:      id,
			Type:    "command",
			Enabled: true,
			Command: &config.CommandStep{Command: "noop"},
		}
		if i > 0 {
			step.DependsOn = []string{fmt.Sprintf("step_%d", i-1)}
		}
		steps[i] = step
	}
	return steps
}

type benchmarkPlugin struct{}

func (benchmarkPlugin) Metadata() plugin.Metadata {
	return plugin.Metadata{Name: "bench", Type: "command", Version: "1.0.0"}
}
func (benchmarkPlugin) Schema() interface{} { return nil }
func (benchmarkPlugin) Check(context.Context, *config.Step) (bool, error) {
	return false, nil
}
func (benchmarkPlugin) Verify(ctx context.Context, step *config.Step) (*model.VerificationResult, error) {
	return &model.VerificationResult{
		StepID:  step.ID,
		Status:  model.StatusSatisfied,
		Message: "benchmark verification satisfied",
	}, nil
}
func (benchmarkPlugin) Apply(ctx context.Context, step *config.Step) (*model.StepResult, error) {
	return &model.StepResult{StepID: step.ID, Status: model.StatusSuccess}, nil
}
func (benchmarkPlugin) DryRun(ctx context.Context, step *config.Step) (*model.StepResult, error) {
	return &model.StepResult{StepID: step.ID, Status: model.StatusSkipped}, nil
}
