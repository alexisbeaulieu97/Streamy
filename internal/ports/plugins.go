package ports

import (
	"context"

	pipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	plugin "github.com/alexisbeaulieu97/streamy/internal/domain/plugin"
)

// Plugin encapsulates the lifecycle for a pipeline step implementation. The
// contract mirrors the historical Streamy plugin API:
//   - Metadata() documents identity, version, and dependencies.
//   - Evaluate() inspects system state without mutating it and determines
//     whether changes are required.
//   - Apply() performs the desired mutations and returns a StepResult with
//     structured diagnostics.
//
// Implementations must honour context cancellation and return domain errors
// enriched with fields such as step_id and plugin_type.
type Plugin interface {
	Metadata() plugin.Metadata
	Evaluate(ctx context.Context, step pipeline.Step) (*pipeline.EvaluationResult, error)
	Apply(ctx context.Context, evaluation *pipeline.EvaluationResult, step pipeline.Step) (*pipeline.StepResult, error)
}

// PluginRegistry manages plugin discovery and registration. Infrastructure
// adapters populate the registry at startup (see cmd/streamy/plugins_import.go)
// while application services resolve plugins by step type. Registries must be
// safe for concurrent use because executors may resolve plugins in parallel.
type PluginRegistry interface {
	Register(p Plugin) error
	Get(stepType plugin.Type) (Plugin, error)
	List() []Plugin
}
