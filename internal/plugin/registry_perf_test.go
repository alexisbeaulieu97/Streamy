package plugin

import (
	"context"
	"fmt"
	"testing"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
)

type benchPlugin struct {
	metadata PluginMetadata
}

func newBenchPlugin(name string, deps ...Dependency) *benchPlugin {
	meta := PluginMetadata{
		Name:         name,
		Version:      "1.0.0",
		APIVersion:   "1.x",
		Dependencies: deps,
	}
	return &benchPlugin{metadata: meta}
}

func (p *benchPlugin) Metadata() Metadata {
	return Metadata{Name: p.metadata.Name, Version: p.metadata.Version, Type: p.metadata.Name}
}

func (p *benchPlugin) Schema() interface{} { return nil }

func (p *benchPlugin) Check(context.Context, *config.Step) (bool, error) { return false, nil }

func (p *benchPlugin) Apply(context.Context, *config.Step) (*model.StepResult, error) {
	return &model.StepResult{Status: "success"}, nil
}

func (p *benchPlugin) DryRun(context.Context, *config.Step) (*model.StepResult, error) {
	return &model.StepResult{Status: "skipped"}, nil
}

func (p *benchPlugin) Verify(context.Context, *config.Step) (*model.VerificationResult, error) {
	return &model.VerificationResult{Status: model.StatusSatisfied}, nil
}

func (p *benchPlugin) PluginMetadata() PluginMetadata { return p.metadata }

func BenchmarkPluginRegistryValidateDependencies(b *testing.B) {
	for _, tc := range []struct {
		name        string
		pluginCount int
	}{
		{name: "10_plugins", pluginCount: 10},
		{name: "50_plugins", pluginCount: 50},
		{name: "100_plugins", pluginCount: 100},
	} {
		b.Run(tc.name, func(b *testing.B) {
			registry := NewPluginRegistry(DefaultConfig(), nil)
			for i := 0; i < tc.pluginCount; i++ {
				name := benchmarkPluginName(i)
				if err := registry.Register(newBenchPlugin(name)); err != nil {
					b.Fatalf("register %s: %v", name, err)
				}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := registry.ValidateDependencies(); err != nil {
					b.Fatalf("validate: %v", err)
				}
			}
		})
	}
}

func BenchmarkPluginRegistryGet(b *testing.B) {
	for _, tc := range []struct {
		name        string
		pluginCount int
	}{
		{name: "10_plugins", pluginCount: 10},
		{name: "50_plugins", pluginCount: 50},
		{name: "100_plugins", pluginCount: 100},
	} {
		b.Run(tc.name, func(b *testing.B) {
			registry := NewPluginRegistry(DefaultConfig(), nil)
			names := make([]string, tc.pluginCount)
			for i := 0; i < tc.pluginCount; i++ {
				name := benchmarkPluginName(i)
				names[i] = name
				if err := registry.Register(newBenchPlugin(name)); err != nil {
					b.Fatalf("register %s: %v", name, err)
				}
			}

			if err := registry.ValidateDependencies(); err != nil {
				b.Fatalf("initial validate: %v", err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				name := names[i%len(names)]
				if _, err := registry.Get(name); err != nil {
					b.Fatalf("get %s: %v", name, err)
				}
			}
		})
	}
}

func benchmarkPluginName(idx int) string {
	return fmt.Sprintf("plugin_%03d", idx)
}
