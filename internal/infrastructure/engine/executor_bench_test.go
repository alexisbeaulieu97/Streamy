package engine

import (
	"context"
	"fmt"
	"testing"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	domainplugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
	plugininfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/plugin"
)

// benchPlugin is a lightweight ports.Plugin implementation used for executor benchmarks.
type benchPlugin struct{}

func (benchPlugin) Metadata() domainplugin.Metadata {
	return domainplugin.Metadata{
		ID:          "bench",
		Name:        "bench",
		Version:     "1.0.0",
		Type:        domainplugin.TypeCommand,
		Description: "Benchmark stub plugin",
	}
}

func (benchPlugin) Evaluate(ctx context.Context, step domainpipeline.Step) (*domainpipeline.EvaluationResult, error) {
	select {
	case <-ctx.Done():
		return nil, domainpipeline.NewCancelledError("benchmark evaluate cancelled", map[string]interface{}{"step_id": step.ID})
	default:
	}

	return &domainpipeline.EvaluationResult{
		RequiresAction: true,
		CurrentState:   string(domainpipeline.VerificationFailed),
	}, nil
}

func (benchPlugin) Apply(ctx context.Context, _ *domainpipeline.EvaluationResult, step domainpipeline.Step) (*domainpipeline.StepResult, error) {
	select {
	case <-ctx.Done():
		return nil, domainpipeline.NewCancelledError("benchmark apply cancelled", map[string]interface{}{"step_id": step.ID})
	default:
	}

	return &domainpipeline.StepResult{
		StepID:  step.ID,
		Status:  domainpipeline.StatusSuccess,
		Message: "bench plugin noop",
		Changed: true,
	}, nil
}

func BenchmarkExecutorPipeline500Steps(b *testing.B) {
	const stepCount = 500

	pipeline, plan := buildBenchmarkPipeline(stepCount)

	registry := plugininfra.NewRegistry()
	if err := registry.Register(benchPlugin{}); err != nil {
		b.Fatalf("register bench plugin: %v", err)
	}

	exec := NewExecutor(registry, WithExecutorParallelism(32))

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		results, err := exec.Execute(ctx, plan, pipeline)
		if err != nil {
			b.Fatalf("execute: %v", err)
		}

		if len(results) != stepCount {
			b.Fatalf("unexpected result count: got %d want %d", len(results), stepCount)
		}
	}
}

func buildBenchmarkPipeline(stepCount int) (*domainpipeline.Pipeline, *domainpipeline.ExecutionPlan) {
	steps := make([]domainpipeline.Step, stepCount)
	levels := make([]domainpipeline.ExecutionLevel, stepCount)

	for i := 0; i < stepCount; i++ {
		id := fmt.Sprintf("step_%03d", i)

		step := domainpipeline.Step{
			ID:      id,
			Type:    domainpipeline.StepTypeCommand,
			Enabled: true,
			Config: map[string]interface{}{
				"command": "echo",
			},
		}
		if i > 0 {
			step.DependsOn = []string{fmt.Sprintf("step_%03d", i-1)}
		}

		steps[i] = step
		levels[i] = domainpipeline.ExecutionLevel{
			Level:   i,
			StepIDs: []string{id},
		}
	}

	pipeline := &domainpipeline.Pipeline{
		Name: "benchmark",
		Settings: domainpipeline.Settings{
			Parallel: 32,
		},
		Steps: steps,
	}
	plan := &domainpipeline.ExecutionPlan{
		Levels:     levels,
		TotalSteps: stepCount,
	}

	return pipeline, plan
}
