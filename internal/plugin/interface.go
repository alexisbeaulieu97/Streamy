package plugin

import (
	"context"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
)

// PluginInitializer allows a plugin to receive a reference to the registry
// during startup. Plugins that do not need initialization can ignore this
// interface; the registry detects it via type assertion and only calls Init
// when implemented.
type PluginInitializer interface {
	Init(registry *PluginRegistry) error
}

// Plugin defines the unified contract all Streamy plugins must satisfy.
//
// This interface replaces the legacy 4-method approach (Check/Apply/DryRun/Verify)
// with a cleaner 2-method Evaluate/Apply pattern.
//
// Implementations should:
//   - Return rich metadata via PluginMetadata()
//   - Provide configuration schemas via Schema()
//   - Implement read-only state assessment via Evaluate()
//   - Implement state mutation via Apply()
//   - Optionally implement PluginInitializer for registry access
type Plugin interface {
	// PluginMetadata returns the plugin's identity, capabilities, and dependencies.
	// This provides rich metadata for the dependency registry and plugin discovery.
	PluginMetadata() PluginMetadata

	// Schema returns a struct that defines the YAML configuration schema
	// for this plugin's steps. The returned struct should have JSON tags
	// for schema generation and validation.
	Schema() any

	// Evaluate performs a STRICTLY READ-ONLY assessment of the system's
	// current state against the desired state defined in the step configuration.
	//
	// CRITICAL CONTRACT: This method MUST NOT mutate any system state.
	// It should only read current state and compute what changes (if any)
	// would be needed to reach the desired state.
	//
	// Returns:
	//   - EvaluationResult: Rich state assessment including current status,
	//     whether action is required, human-readable message, optional diff,
	//     and optional internal data to pass to Apply()
	//   - error: PluginError (ValidationError, ExecutionError, StateError)
	//     if evaluation cannot be completed
	Evaluate(ctx context.Context, step *config.Step) (*model.EvaluationResult, error)

	// Apply mutates the system to match the desired state defined in the
	// step configuration. This method is ONLY called by the engine if
	// Evaluate() reported RequiresAction = true.
	//
	// The evalResult parameter contains the result from the previous
	// Evaluate() call, including InternalData that can be used to avoid
	// redundant computation.
	//
	// This method MUST be idempotent: calling it multiple times with the
	// same inputs should produce the same final state.
	//
	// Returns:
	//   - StepResult: Outcome of the apply operation (success, failure, etc.)
	//   - error: PluginError if the operation fails
	Apply(ctx context.Context, evalResult *model.EvaluationResult, step *config.Step) (*model.StepResult, error)
}
