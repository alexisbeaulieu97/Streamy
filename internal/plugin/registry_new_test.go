package plugin

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/model"
)

func TestPluginRegistryRegisterGetAndList(t *testing.T) {
	registry := NewPluginRegistry(&RegistryConfig{
		DependencyPolicy: PolicyStrict,
		AccessPolicy:     AccessStrict,
	}, nil)

	core := newRegistryTestPlugin("core", false, nil)
	require.NoError(t, registry.Register(core))

	require.NoError(t, registry.ValidateDependencies())
	require.NoError(t, registry.InitializePlugins())

	// Metadata is stored and defaults applied.
	meta, ok := registry.metadata["core"]
	require.True(t, ok)
	require.Equal(t, "core", meta.Name)
	require.Equal(t, "1.0.0", meta.Version)
	require.Equal(t, "1.x", meta.APIVersion)
	require.Empty(t, meta.Dependencies)

	// Get returns registered plugin.
	got, err := registry.Get("core")
	require.NoError(t, err)
	require.Same(t, core, got)

	// List excludes disabled entries and is sorted.
	names := registry.List()
	require.Equal(t, []string{"core"}, names)
}

func TestPluginRegistryStatefulInstancesPerDependent(t *testing.T) {
	cfg := &RegistryConfig{
		DependencyPolicy: PolicyStrict,
		AccessPolicy:     AccessStrict,
	}
	registry := NewPluginRegistry(cfg, nil)

	stateful := newRegistryTestPlugin("stateful", true, nil)
	consumer := newRegistryTestPlugin("consumer", false, []Dependency{{Name: "stateful"}})
	other := newRegistryTestPlugin("other", false, []Dependency{{Name: "stateful"}})

	require.NoError(t, registry.Register(stateful))
	require.NoError(t, registry.Register(consumer))
	require.NoError(t, registry.Register(other))

	require.NoError(t, registry.ValidateDependencies())
	require.NoError(t, registry.InitializePlugins())

	// Each dependent gets its own instance.
	consumerInstance, err := registry.GetForDependent("consumer", "stateful")
	require.NoError(t, err)
	require.NotNil(t, consumerInstance)
	require.NotSame(t, stateful, consumerInstance)

	// Same dependent should receive cached instance.
	again, err := registry.GetForDependent("consumer", "stateful")
	require.NoError(t, err)
	require.Same(t, consumerInstance, again)

	// Different dependent receives a distinct instance.
	otherInstance, err := registry.GetForDependent("other", "stateful")
	require.NoError(t, err)
	require.NotSame(t, consumerInstance, otherInstance)
	require.NotSame(t, stateful, otherInstance)
}

func TestPluginRegistryUndeclaredDependencyStrict(t *testing.T) {
	cfg := &RegistryConfig{
		DependencyPolicy: PolicyStrict,
		AccessPolicy:     AccessStrict,
	}
	registry := NewPluginRegistry(cfg, nil)

	provider := newRegistryTestPlugin("provider", false, nil)
	consumer := newRegistryTestPlugin("consumer", false, nil)

	require.NoError(t, registry.Register(provider))
	require.NoError(t, registry.Register(consumer))
	require.NoError(t, registry.ValidateDependencies())
	require.NoError(t, registry.InitializePlugins())

	_, err := registry.GetForDependent("consumer", "provider")
	require.Error(t, err)
	var undeclared ErrUndeclaredDependency
	require.ErrorAs(t, err, &undeclared)
	require.Equal(t, "consumer", undeclared.Caller)
	require.Equal(t, "provider", undeclared.Dependency)
}

func TestPluginRegistryDisablesPluginsGracefully(t *testing.T) {
	cfg := &RegistryConfig{
		DependencyPolicy: PolicyGraceful,
		AccessPolicy:     AccessStrict,
	}
	registry := NewPluginRegistry(cfg, nil)

	dependent := newRegistryTestPlugin("needs-missing", false, []Dependency{{Name: "absent"}})
	require.NoError(t, registry.Register(dependent))

	err := registry.ValidateDependencies()
	require.NoError(t, err)

	// Plugin should be disabled and excluded from listings.
	require.Empty(t, registry.List())

	_, err = registry.Get("needs-missing")
	require.Error(t, err)
	var notFound ErrPluginNotFound
	require.ErrorAs(t, err, &notFound)
}

func newRegistryTestPlugin(name string, stateful bool, deps []Dependency) *registryTestPlugin {
	return &registryTestPlugin{
		meta: PluginMetadata{
			Name:         name,
			Version:      "1.0.0",
			APIVersion:   "",
			Type:         name,
			Stateful:     stateful,
			Dependencies: deps,
		},
	}
}

type registryTestPlugin struct {
	meta PluginMetadata
}

func (p *registryTestPlugin) PluginMetadata() PluginMetadata { return p.meta }

func (p *registryTestPlugin) Schema() any { return struct{}{} }

func (p *registryTestPlugin) Evaluate(_ context.Context, step *config.Step) (*model.EvaluationResult, error) {
	if step == nil {
		return nil, fmt.Errorf("step is nil")
	}
	return &model.EvaluationResult{
		StepID:         step.ID,
		CurrentState:   model.StatusSatisfied,
		RequiresAction: false,
		Message:        "noop",
	}, nil
}

func (p *registryTestPlugin) Apply(_ context.Context, _ *model.EvaluationResult, step *config.Step) (*model.StepResult, error) {
	return &model.StepResult{
		StepID: step.ID,
		Status: model.StatusSuccess,
	}, nil
}

func (p *registryTestPlugin) Init(*PluginRegistry) error {
	return nil
}
