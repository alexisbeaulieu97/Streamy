package tests

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/logger"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
)

func TestPluginDependency_Composition(t *testing.T) {
	cfg := &plugin.RegistryConfig{
		DependencyPolicy: plugin.PolicyStrict,
		AccessPolicy:     plugin.AccessStrict,
	}
	registry := plugin.NewPluginRegistry(cfg, nil)

	line := newIntegrationPlugin("line_in_file", withPluginType("line_in_file"))
	line.applyFn = func(ctx context.Context, step *config.Step, reg *plugin.PluginRegistry) (*model.StepResult, error) {
		line.lastApplyStep = cloneStep(step)
		return &model.StepResult{StepID: step.ID, Status: "success"}, nil
	}

	shell := newIntegrationPlugin(
		"shell_profile",
		withPluginType("shell_profile"),
		withDependencies(plugin.Dependency{
			Name:              "line_in_file",
			VersionConstraint: plugin.MustParseVersionConstraint("1.x"),
		}),
	)
	shell.applyFn = func(ctx context.Context, step *config.Step, reg *plugin.PluginRegistry) (*model.StepResult, error) {
		dependentStep := &config.Step{ID: step.ID + "_line", Name: "line_in_file"}
		dep, err := reg.GetForDependent("shell_profile", "line_in_file")
		if err != nil {
			return nil, err
		}
		shell.invokeLog = append(shell.invokeLog, "line_in_file")
		// Evaluate first, then apply
		evalResult, err := dep.Evaluate(ctx, dependentStep)
		if err != nil {
			return nil, err
		}
		return dep.Apply(ctx, evalResult, dependentStep)
	}

	require.NoError(t, registry.Register(line))
	require.NoError(t, registry.Register(shell))

	require.NoError(t, registry.ValidateDependencies())
	require.NoError(t, registry.InitializePlugins())

	pluginFromRegistry, err := registry.Get("shell_profile")
	require.NoError(t, err)
	require.Same(t, shell, pluginFromRegistry)

	ctx := context.Background()
	step := &config.Step{ID: "setup_dev_path", Name: "shell_profile"}
	// Evaluate first, then apply
	evalResult, err := pluginFromRegistry.Evaluate(ctx, step)
	require.NoError(t, err)
	result, err := pluginFromRegistry.Apply(ctx, evalResult, step)
	require.NoError(t, err)
	require.Equal(t, "success", result.Status)
	require.Equal(t, "setup_dev_path_line", line.lastApplyStep.ID)
	require.Equal(t, 1, line.applyCalls)
	require.Equal(t, 1, shell.applyCalls)
	require.Equal(t, []string{"line_in_file"}, shell.invokeLog)
}

func TestPluginDependency_TransitiveChain(t *testing.T) {
	cfg := &plugin.RegistryConfig{DependencyPolicy: plugin.PolicyStrict, AccessPolicy: plugin.AccessStrict}
	registry := plugin.NewPluginRegistry(cfg, nil)

	order := []string{}

	pluginC := newIntegrationPlugin("plugin_c", withInit(func(*plugin.PluginRegistry) error {
		order = append(order, "plugin_c")
		return nil
	}))

	pluginB := newIntegrationPlugin(
		"plugin_b",
		withDependencies(plugin.Dependency{Name: "plugin_c"}),
		withInit(func(*plugin.PluginRegistry) error {
			order = append(order, "plugin_b")
			return nil
		}),
	)
	pluginB.applyFn = func(ctx context.Context, step *config.Step, reg *plugin.PluginRegistry) (*model.StepResult, error) {
		dep, err := reg.GetForDependent("plugin_b", "plugin_c")
		if err != nil {
			return nil, err
		}
		evalResult, err := dep.Evaluate(ctx, step)
		if err != nil {
			return nil, err
		}
		return dep.Apply(ctx, evalResult, step)
	}

	pluginA := newIntegrationPlugin(
		"plugin_a",
		withDependencies(plugin.Dependency{Name: "plugin_b"}),
		withInit(func(*plugin.PluginRegistry) error {
			order = append(order, "plugin_a")
			return nil
		}),
	)
	pluginA.applyFn = func(ctx context.Context, step *config.Step, reg *plugin.PluginRegistry) (*model.StepResult, error) {
		dep, err := reg.GetForDependent("plugin_a", "plugin_b")
		if err != nil {
			return nil, err
		}
		evalResult, err := dep.Evaluate(ctx, step)
		if err != nil {
			return nil, err
		}
		return dep.Apply(ctx, evalResult, step)
	}

	require.NoError(t, registry.Register(pluginC))
	require.NoError(t, registry.Register(pluginB))
	require.NoError(t, registry.Register(pluginA))

	require.NoError(t, registry.ValidateDependencies())
	require.NoError(t, registry.InitializePlugins())
	require.Equal(t, []string{"plugin_c", "plugin_b", "plugin_a"}, order)

	ctx := context.Background()
	step := &config.Step{ID: "composite", Name: "plugin_a"}
	evalResult, err := pluginA.Evaluate(ctx, step)
	require.NoError(t, err)
	_, err = pluginA.Apply(ctx, evalResult, step)
	require.NoError(t, err)

	require.Equal(t, 1, pluginA.applyCalls)
	require.Equal(t, 1, pluginB.applyCalls)
	require.Equal(t, 1, pluginC.applyCalls)
}

func TestPluginDependency_PolicyModes(t *testing.T) {
	t.Run("strict missing dependency", func(t *testing.T) {
		cfg := &plugin.RegistryConfig{DependencyPolicy: plugin.PolicyStrict, AccessPolicy: plugin.AccessStrict}
		registry := plugin.NewPluginRegistry(cfg, nil)
		missing := newIntegrationPlugin("needs_missing", withDependencies(plugin.Dependency{Name: "absent"}))
		require.NoError(t, registry.Register(missing))
		err := registry.ValidateDependencies()
		require.Error(t, err)
		var missingErr *plugin.ErrMissingDependency
		require.ErrorAs(t, err, &missingErr)
	})

	t.Run("graceful missing dependency", func(t *testing.T) {
		cfg := &plugin.RegistryConfig{DependencyPolicy: plugin.PolicyGraceful, AccessPolicy: plugin.AccessWarn}
		registry := plugin.NewPluginRegistry(cfg, nil)
		alpha := newIntegrationPlugin("alpha")
		beta := newIntegrationPlugin("beta", withDependencies(plugin.Dependency{Name: "absent"}))

		require.NoError(t, registry.Register(alpha))
		require.NoError(t, registry.Register(beta))

		err := registry.ValidateDependencies()
		require.NoError(t, err)

		names := registry.List()
		require.Equal(t, []string{"alpha"}, names)
	})

	t.Run("strict circular dependency", func(t *testing.T) {
		cfg := &plugin.RegistryConfig{DependencyPolicy: plugin.PolicyStrict, AccessPolicy: plugin.AccessStrict}
		registry := plugin.NewPluginRegistry(cfg, nil)
		a := newIntegrationPlugin("cycle_a", withDependencies(plugin.Dependency{Name: "cycle_b"}))
		b := newIntegrationPlugin("cycle_b", withDependencies(plugin.Dependency{Name: "cycle_c"}))
		c := newIntegrationPlugin("cycle_c", withDependencies(plugin.Dependency{Name: "cycle_a"}))

		require.NoError(t, registry.Register(a))
		require.NoError(t, registry.Register(b))
		require.NoError(t, registry.Register(c))

		err := registry.ValidateDependencies()
		require.Error(t, err)
		var cycleErr *plugin.ErrCircularDependency
		require.ErrorAs(t, err, &cycleErr)
		require.NotEmpty(t, cycleErr.Cycle)
	})

	t.Run("graceful undeclared access", func(t *testing.T) {
		cfg := &plugin.RegistryConfig{DependencyPolicy: plugin.PolicyGraceful, AccessPolicy: plugin.AccessWarn}
		registry := plugin.NewPluginRegistry(cfg, nil)
		provider := newIntegrationPlugin("provider")
		consumer := newIntegrationPlugin("consumer")

		require.NoError(t, registry.Register(provider))
		require.NoError(t, registry.Register(consumer))
		require.NoError(t, registry.ValidateDependencies())
		require.NoError(t, registry.InitializePlugins())

		_, err := registry.GetForDependent("consumer", "provider")
		require.NoError(t, err)
	})

	t.Run("strict undeclared access", func(t *testing.T) {
		cfg := &plugin.RegistryConfig{DependencyPolicy: plugin.PolicyStrict, AccessPolicy: plugin.AccessStrict}
		registry := plugin.NewPluginRegistry(cfg, nil)
		provider := newIntegrationPlugin("provider")
		consumer := newIntegrationPlugin("consumer")

		require.NoError(t, registry.Register(provider))
		require.NoError(t, registry.Register(consumer))
		require.NoError(t, registry.ValidateDependencies())
		require.NoError(t, registry.InitializePlugins())

		_, err := registry.GetForDependent("consumer", "provider")
		require.Error(t, err)
		var undeclared plugin.ErrUndeclaredDependency
		require.ErrorAs(t, err, &undeclared)
	})
}

func TestPluginDependency_BackwardCompatibility(t *testing.T) {
	var logBuffer bytes.Buffer
	log, err := logger.New(logger.Options{Writer: &logBuffer, Level: "debug", HumanReadable: true})
	require.NoError(t, err)

	cfg := &plugin.RegistryConfig{DependencyPolicy: plugin.PolicyGraceful, AccessPolicy: plugin.AccessStrict}
	registry := plugin.NewPluginRegistry(cfg, log)

	legacy := &legacyPlugin{name: "legacy_tool"}
	modern := newIntegrationPlugin(
		"modern_tool",
		withDependencies(plugin.Dependency{
			Name:              "legacy_tool",
			VersionConstraint: plugin.MustParseVersionConstraint("1.x"),
		}),
	)

	require.NoError(t, registry.Register(legacy))
	require.NoError(t, registry.Register(modern))

	require.NoError(t, registry.ValidateDependencies())
	require.NoError(t, registry.InitializePlugins())

	// Legacy plugin still retrievable
	legacyFromRegistry, err := registry.Get("legacy_tool")
	require.NoError(t, err)
	evalResult, err := legacyFromRegistry.Evaluate(context.Background(), &config.Step{ID: "legacy", Name: "legacy_tool"})
	require.NoError(t, err)
	_, err = legacyFromRegistry.Apply(context.Background(), evalResult, &config.Step{ID: "legacy", Name: "legacy_tool"})
	require.NoError(t, err)

	// Modern plugin can resolve dependency on legacy plugin
	evalResult, err = modern.Evaluate(context.Background(), &config.Step{ID: "modern", Name: "modern_tool"})
	require.NoError(t, err)
	_, err = modern.Apply(context.Background(), evalResult, &config.Step{ID: "modern", Name: "modern_tool"})
	require.NoError(t, err)

	names := registry.List()
	require.Contains(t, names, "modern_tool")

	// Legacy plugin cannot access undeclared dependencies under strict policy
	_, err = registry.GetForDependent("legacy_tool", "modern_tool")
	require.Error(t, err)
	var undeclared plugin.ErrUndeclaredDependency
	require.ErrorAs(t, err, &undeclared)

	require.Contains(t, logBuffer.String(), "legacy")
}

// --- Test plugin helpers ----------------------------------------------------

type integrationPluginOption func(*integrationTestPlugin)

type integrationTestPlugin struct {
	metadata      plugin.PluginMetadata
	pluginType    string
	registry      *plugin.PluginRegistry
	initFn        func(*plugin.PluginRegistry) error
	evaluateFn    func(context.Context, *config.Step) (*model.EvaluationResult, error)
	applyFn       func(context.Context, *config.Step, *plugin.PluginRegistry) (*model.StepResult, error)
	applyCalls    int
	lastApplyStep *config.Step
	invokeLog     []string
}

func newIntegrationPlugin(name string, opts ...integrationPluginOption) *integrationTestPlugin {
	p := &integrationTestPlugin{
		metadata: plugin.PluginMetadata{
			Name:         name,
			Version:      "1.0.0",
			APIVersion:   "1.x",
			Dependencies: []plugin.Dependency{},
		},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func withDependencies(deps ...plugin.Dependency) integrationPluginOption {
	copied := make([]plugin.Dependency, len(deps))
	copy(copied, deps)
	return func(p *integrationTestPlugin) {
		p.metadata.Dependencies = copied
	}
}

func withPluginType(t string) integrationPluginOption {
	return func(p *integrationTestPlugin) {
		p.pluginType = t
	}
}

func withInit(fn func(*plugin.PluginRegistry) error) integrationPluginOption {
	return func(p *integrationTestPlugin) {
		p.initFn = fn
	}
}

func (p *integrationTestPlugin) PluginMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{Name: p.metadata.Name, Version: p.metadata.Version, Type: p.pluginType}
}

func (p *integrationTestPlugin) Schema() any { return nil }

func (p *integrationTestPlugin) Evaluate(ctx context.Context, step *config.Step) (*model.EvaluationResult, error) {
	if p.evaluateFn != nil {
		return p.evaluateFn(ctx, step)
	}
	// Default evaluation - always requires action
	return &model.EvaluationResult{
		StepID:         step.ID,
		CurrentState:   model.StatusUnknown,
		RequiresAction: true,
		Message:        "integration test plugin evaluation",
	}, nil
}

func (p *integrationTestPlugin) Apply(ctx context.Context, evalResult *model.EvaluationResult, step *config.Step) (*model.StepResult, error) {
	p.applyCalls++
	p.lastApplyStep = cloneStep(step)
	if p.applyFn != nil {
		return p.applyFn(ctx, step, p.registry)
	}
	return &model.StepResult{StepID: step.ID, Status: "success"}, nil
}

func (p *integrationTestPlugin) Init(registry *plugin.PluginRegistry) error {
	p.registry = registry
	if p.initFn != nil {
		return p.initFn(registry)
	}
	return nil
}

// --- Legacy plugin ---------------------------------------------------------

type legacyPlugin struct {
	name string
}

func (p *legacyPlugin) PluginMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{Name: p.name, Version: "1.0.0", Type: "legacy"}
}

func (p *legacyPlugin) Schema() any { return nil }

func (p *legacyPlugin) Evaluate(ctx context.Context, step *config.Step) (*model.EvaluationResult, error) {
	return &model.EvaluationResult{
		StepID:         step.ID,
		CurrentState:   model.StatusUnknown,
		RequiresAction: true,
		Message:        "legacy plugin evaluation",
	}, nil
}

func (p *legacyPlugin) Apply(ctx context.Context, evalResult *model.EvaluationResult, step *config.Step) (*model.StepResult, error) {
	return &model.StepResult{StepID: step.ID, Status: "success"}, nil
}

// --- Utility helpers -------------------------------------------------------

func cloneStep(step *config.Step) *config.Step {
	if step == nil {
		return nil
	}
	clone := *step
	if step.DependsOn != nil {
		clone.DependsOn = append([]string(nil), step.DependsOn...)
	}
	return &clone
}
