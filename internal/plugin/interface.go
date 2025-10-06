package plugin

import (
	"context"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
)

// Metadata describes legacy plugin capabilities.
//
// # Migration Note
// Plugins participating in the dependency registry should implement the
// MetadataProvider interface and return a PluginMetadata value that includes
// dependency declarations, version constraints, and stateful behaviour.
// The Metadata struct remains for backward compatibility with existing
// plugins that have not yet migrated.
type Metadata struct {
	Name    string
	Version string
	Type    string
}

// MetadataProvider exposes the enriched PluginMetadata used by the registry.
// Implementing this interface is optional but required for dependency-aware
// plugins to declare dependencies, version constraints, and policy hints.
type MetadataProvider interface {
	PluginMetadata() PluginMetadata
}

// PluginInitializer allows a plugin to receive a reference to the registry
// during startup. Plugins that do not need initialization can ignore this
// interface; the registry detects it via type assertion and only calls Init
// when implemented.
type PluginInitializer interface {
	Init(registry *PluginRegistry) error
}

// Plugin defines the contract all Streamy plugins must satisfy.
//
// Implementations should:
//   - Return stable identity data via Metadata(), noting that dependency-aware
//     plugins typically also implement MetadataProvider for richer metadata.
//   - Provide JSON/YAML schemas from Schema() to support validation.
//   - Use Check/Apply/DryRun/Verify to participate in Streamy's reconciliation loop.
//   - Optionally implement PluginInitializer to acquire dependencies from the
//     registry during initialization.
type Plugin interface {
	Metadata() Metadata
	Schema() interface{}
	Check(ctx context.Context, step *config.Step) (bool, error)
	Apply(ctx context.Context, step *config.Step) (*model.StepResult, error)
	DryRun(ctx context.Context, step *config.Step) (*model.StepResult, error)
	Verify(ctx context.Context, step *config.Step) (*model.VerificationResult, error)
}
