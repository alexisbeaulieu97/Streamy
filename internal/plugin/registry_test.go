package plugin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPluginRegistry_RegisterAndGet(t *testing.T) {
	registry := NewPluginRegistry(DefaultConfig(), nil)

	plugin := NewMockPlugin("test")

	err := registry.Register(plugin)
	require.NoError(t, err)

	fetched, err := registry.Get("test")
	require.NoError(t, err)
	require.Equal(t, plugin, fetched)

	_, err = registry.Get("missing")
	require.Error(t, err)
	var notFound ErrPluginNotFound
	require.ErrorAs(t, err, &notFound)
	require.Equal(t, "missing", notFound.Name)
}

func TestPluginRegistry_MissingDependencyDetection(t *testing.T) {
	cfg := strictRegistryConfig()
	registry := NewPluginRegistry(cfg, nil)

	dependent := NewMockPlugin("dependent", WithDependencies(Dependency{Name: "missing"}))

	require.NoError(t, registry.Register(dependent))

	err := registry.ValidateDependencies()
	require.Error(t, err)
	var missing *ErrMissingDependency
	require.ErrorAs(t, err, &missing)
	require.Equal(t, "dependent", missing.Plugin)
	require.Equal(t, "missing", missing.Dependency)
}

func TestPluginRegistry_CircularDependencyDetection(t *testing.T) {
	cfg := strictRegistryConfig()
	registry := NewPluginRegistry(cfg, nil)

	pluginA := NewMockPlugin("A", WithDependencies(Dependency{Name: "B"}))
	pluginB := NewMockPlugin("B", WithDependencies(Dependency{Name: "C"}))
	pluginC := NewMockPlugin("C", WithDependencies(Dependency{Name: "A"}))

	require.NoError(t, registry.Register(pluginA))
	require.NoError(t, registry.Register(pluginB))
	require.NoError(t, registry.Register(pluginC))

	err := registry.ValidateDependencies()
	require.Error(t, err)
	var cycle *ErrCircularDependency
	require.ErrorAs(t, err, &cycle)
	require.NotEmpty(t, cycle.Cycle)
}

func TestPluginRegistry_InitializePluginsWithoutInitializer(t *testing.T) {
	registry := NewPluginRegistry(DefaultConfig(), nil)

	plug := NewMockPlugin("simple")

	require.NoError(t, registry.Register(plug))
	require.NoError(t, registry.ValidateDependencies())

	require.NoError(t, registry.InitializePlugins())
	require.NotContains(t, plug.Calls(), "Init")
}

func TestPluginRegistry_InitializationOrder(t *testing.T) {
	registry := NewPluginRegistry(DefaultConfig(), nil)

	order := []string{}
	pluginA := NewInitializingMockPlugin("A", func(*PluginRegistry) error {
		order = append(order, "A")
		return nil
	})

	pluginB := NewInitializingMockPlugin("B", func(*PluginRegistry) error {
		order = append(order, "B")
		return nil
	}, WithDependencies(Dependency{Name: "A"}))

	require.NoError(t, registry.Register(pluginA))
	require.NoError(t, registry.Register(pluginB))

	require.NoError(t, registry.ValidateDependencies())

	err := registry.InitializePlugins()
	require.NoError(t, err)
	require.Equal(t, []string{"A", "B"}, order)
}

func TestPluginRegistry_GetForDependentEnforcesAccessPolicy(t *testing.T) {
	tests := []struct {
		name    string
		config  *RegistryConfig
		wantErr bool
	}{
		{name: "strict", config: &RegistryConfig{DependencyPolicy: PolicyStrict, AccessPolicy: AccessStrict}, wantErr: true},
		{name: "warn", config: &RegistryConfig{DependencyPolicy: PolicyStrict, AccessPolicy: AccessWarn}},
		{name: "off", config: &RegistryConfig{DependencyPolicy: PolicyStrict, AccessPolicy: AccessOff}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			registry := NewPluginRegistry(tc.config, nil)

			dependent := NewMockPlugin("consumer")
			dependency := NewMockPlugin("dep")

			require.NoError(t, registry.Register(dependent))
			require.NoError(t, registry.Register(dependency))

			require.NoError(t, registry.ValidateDependencies())

			plugin, err := registry.GetForDependent("consumer", "dep")
			if tc.wantErr {
				require.Error(t, err)
				var undeclared ErrUndeclaredDependency
				require.ErrorAs(t, err, &undeclared)
				return
			}

			require.NoError(t, err)
			require.Equal(t, dependency, plugin)
		})
	}
}

func TestPluginRegistry_GetForDependentStatefulInstances(t *testing.T) {
	config := &RegistryConfig{DependencyPolicy: PolicyStrict, AccessPolicy: AccessStrict}
	registry := NewPluginRegistry(config, nil)

	stateful := NewMockPlugin("stateful", WithStateful(true))
	dependentA := NewMockPlugin("consumer_a", WithDependencies(Dependency{Name: "stateful"}))
	dependentB := NewMockPlugin("consumer_b", WithDependencies(Dependency{Name: "stateful"}))

	require.NoError(t, registry.Register(stateful))
	require.NoError(t, registry.Register(dependentA))
	require.NoError(t, registry.Register(dependentB))
	require.NoError(t, registry.ValidateDependencies())

	instanceA1, err := registry.GetForDependent("consumer_a", "stateful")
	require.NoError(t, err)
	require.NotNil(t, instanceA1)

	instanceA2, err := registry.GetForDependent("consumer_a", "stateful")
	require.NoError(t, err)
	require.Same(t, instanceA1, instanceA2)

	instanceB, err := registry.GetForDependent("consumer_b", "stateful")
	require.NoError(t, err)
	require.NotSame(t, instanceA1, instanceB)
}

func TestPluginRegistry_ListSortedAndFiltersDisabled(t *testing.T) {
	// Use graceful policy to allow missing dependencies and filter out affected plugins
	registry := NewPluginRegistry(&RegistryConfig{
		DependencyPolicy: PolicyGraceful,
		AccessPolicy:     AccessWarn,
	}, nil)

	alpha := NewMockPlugin("alpha")
	beta := NewMockPlugin("beta", WithDependencies(Dependency{Name: "missing"}))
	gamma := NewMockPlugin("gamma")

	require.NoError(t, registry.Register(alpha))
	require.NoError(t, registry.Register(beta))
	require.NoError(t, registry.Register(gamma))

	require.NoError(t, registry.ValidateDependencies())

	names := registry.List()
	require.Equal(t, []string{"alpha", "gamma"}, names)
}

func strictRegistryConfig() *RegistryConfig {
	return &RegistryConfig{
		DependencyPolicy: PolicyStrict,
		AccessPolicy:     AccessStrict,
	}
}
