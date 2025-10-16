package config

import (
	"fmt"
	"testing"
)

func BenchmarkValidateConfigLarge(b *testing.B) {
	cfg := generateConfig(2000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := ValidateConfig(cfg); err != nil {
			b.Fatalf("validate config: %v", err)
		}
	}
}

func generateConfig(count int) *Config {
	steps := make([]Step, count)
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("step_%d", i)
		step := Step{ID: id, Type: "command", Enabled: true}
		if err := step.SetConfig(CommandStep{Command: "noop"}); err != nil {
			panic(err)
		}
		if i > 0 {
			step.DependsOn = []string{fmt.Sprintf("step_%d", i-1)}
		}
		steps[i] = step
	}

	return &Config{
		Version: "1.0.0",
		Name:    "benchmark",
		Steps:   steps,
	}
}
