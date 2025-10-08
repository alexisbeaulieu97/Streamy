package plugin

import (
	"context"
	"testing"
	"time"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
)

// benchmarkPlugin implements the new Plugin interface for benchmarking
type benchmarkPlugin struct {
	name    string
	version string
}

func (p *benchmarkPlugin) PluginMetadata() PluginMetadata {
	return PluginMetadata{Name: p.name, Version: p.version, Type: p.name}
}

func (p *benchmarkPlugin) Schema() any {
	return nil
}

func (p *benchmarkPlugin) Evaluate(ctx context.Context, step *config.Step) (*model.EvaluationResult, error) {
	// Simulate some evaluation work
	time.Sleep(10 * time.Microsecond) // Small delay to simulate work

	return &model.EvaluationResult{
		StepID:         step.ID,
		CurrentState:   model.StatusUnknown,
		RequiresAction: true,
		Message:        "benchmark evaluation",
		InternalData:   &benchmarkEvaluationData{computed: "value"},
	}, nil
}

func (p *benchmarkPlugin) Apply(ctx context.Context, evalResult *model.EvaluationResult, step *config.Step) (*model.StepResult, error) {
	// Use pre-computed data from Evaluate() if available
	var computedValue string
	if evalResult != nil && evalResult.InternalData != nil {
		data := evalResult.InternalData.(*benchmarkEvaluationData)
		computedValue = data.computed
	} else {
		computedValue = "direct-value"
	}

	// Simulate some apply work
	time.Sleep(50 * time.Microsecond) // Small delay to simulate work

	return &model.StepResult{
		StepID:  step.ID,
		Status:  model.StatusSuccess,
		Message: "benchmark apply completed with " + computedValue,
	}, nil
}

type benchmarkEvaluationData struct {
	computed string
}

// BenchmarkEvaluate measures the performance of the Evaluate method
func BenchmarkEvaluate(b *testing.B) {
	plugin := &benchmarkPlugin{name: "benchmark", version: "1.0.0"}
	step := &config.Step{ID: "test", Type: "benchmark"}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := plugin.Evaluate(ctx, step)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkApply measures the performance of the Apply method
func BenchmarkApply(b *testing.B) {
	plugin := &benchmarkPlugin{name: "benchmark", version: "1.0.0"}
	step := &config.Step{ID: "test", Type: "benchmark"}
	ctx := context.Background()

	// Pre-compute evaluation result
	evalResult, err := plugin.Evaluate(ctx, step)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := plugin.Apply(ctx, evalResult, step)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEvaluateApplySequence measures the complete Evaluate+Apply workflow
func BenchmarkEvaluateApplySequence(b *testing.B) {
	plugin := &benchmarkPlugin{name: "benchmark", version: "1.0.0"}
	step := &config.Step{ID: "test", Type: "benchmark"}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		evalResult, err := plugin.Evaluate(ctx, step)
		if err != nil {
			b.Fatal(err)
		}

		_, err = plugin.Apply(ctx, evalResult, step)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEvaluateWithLargeInternalData tests performance with larger InternalData
func BenchmarkEvaluateWithLargeInternalData(b *testing.B) {
	plugin := &benchmarkPlugin{name: "benchmark", version: "1.0.0"}
	step := &config.Step{ID: "test", Type: "benchmark"}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate large internal data
		largeData := make([]byte, 10*1024) // 10KB
		for j := range largeData {
			largeData[j] = byte(i % 256)
		}

		result, err := plugin.Evaluate(ctx, step)
		if err != nil {
			b.Fatal(err)
		}

		// Replace with large data (simulating expensive computation)
		result.InternalData = largeData
	}
}

// BenchmarkConcurrentEvaluate measures concurrent evaluation performance
func BenchmarkConcurrentEvaluate(b *testing.B) {
	plugin := &benchmarkPlugin{name: "benchmark", version: "1.0.0"}
	step := &config.Step{ID: "test", Type: "benchmark"}
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := plugin.Evaluate(ctx, step)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkConcurrentEvaluateApply measures concurrent Evaluate+Apply performance
func BenchmarkConcurrentEvaluateApply(b *testing.B) {
	plugin := &benchmarkPlugin{name: "benchmark", version: "1.0.0"}
	step := &config.Step{ID: "test", Type: "benchmark"}
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			evalResult, err := plugin.Evaluate(ctx, step)
			if err != nil {
				b.Fatal(err)
			}

			_, err = plugin.Apply(ctx, evalResult, step)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkInterfaceOverhead measures the overhead of the new interface
func BenchmarkInterfaceOverhead(b *testing.B) {
	plugin := &benchmarkPlugin{name: "benchmark", version: "1.0.0"}
	step := &config.Step{ID: "test", Type: "benchmark"}
	ctx := context.Background()

	b.Run("direct_evaluation", func(b *testing.B) {
		// Baseline: direct function call without interface
		for i := 0; i < b.N; i++ {
			_ = &model.EvaluationResult{
				StepID:         step.ID,
				CurrentState:   model.StatusUnknown,
				RequiresAction: true,
				Message:        "baseline",
				InternalData:   &benchmarkEvaluationData{computed: "value"},
			}
		}
	})

	b.Run("interface_evaluation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := plugin.Evaluate(ctx, step)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkInternalDataEfficiency tests the benefit of using InternalData
func BenchmarkInternalDataEfficiency(b *testing.B) {
	plugin := &benchmarkPlugin{name: "benchmark", version: "1.0.0"}
	step := &config.Step{ID: "test", Type: "benchmark"}
	ctx := context.Background()

	b.Run("without_internal_data", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Simulate expensive computation in Apply
			time.Sleep(10 * time.Microsecond)
			_, err := plugin.Apply(ctx, nil, step)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("with_internal_data", func(b *testing.B) {
		// Pre-compute evaluation result
		evalResult, err := plugin.Evaluate(ctx, step)
		if err != nil {
			b.Fatal(err)
		}

		for i := 0; i < b.N; i++ {
			// Use pre-computed data from Evaluate
			_, err := plugin.Apply(ctx, evalResult, step)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
