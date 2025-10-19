package tests

import (
	"testing"

	"github.com/stretchr/testify/require"

	applicationpipeline "github.com/alexisbeaulieu97/streamy/internal/application/pipeline"
	applicationvalidation "github.com/alexisbeaulieu97/streamy/internal/application/validation"
	configinfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/config"
	engineinfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/engine"
	eventsinfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/events"
	logginginfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/logging"
	metricsinfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/metrics"
	plugininfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/plugin"
	tracinginfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/tracing"
	commandplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/command"
	copyplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/copy"
	lineinfileplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/lineinfile"
	packageplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/package"
	repoplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/repo"
	symlinkplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/symlink"
	templateplugin "github.com/alexisbeaulieu97/streamy/internal/plugins/template"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

// appHarness wires the application layer components for integration tests.
type appHarness struct {
	Logger         ports.Logger
	Events         ports.EventPublisher
	Metrics        *metricsinfra.Collector
	Tracer         ports.Tracer
	Registry       *plugininfra.Registry
	Executor       ports.PluginExecutor
	ConfigLoader   ports.ConfigLoader
	PrepareUseCase *applicationpipeline.PrepareUseCase
	ApplyUseCase   *applicationpipeline.ApplyUseCase
	VerifyUseCase  *applicationpipeline.VerifyUseCase
}

func newAppHarness(t *testing.T) *appHarness {
	return newAppHarnessWithPlugins(t, builtinPortPlugins()...)
}

func newCustomAppHarness(t *testing.T, plugins ...ports.Plugin) *appHarness {
	return newAppHarnessWithPlugins(t, plugins...)
}

func newAppHarnessWithPlugins(t *testing.T, plugins ...ports.Plugin) *appHarness {
	t.Helper()

	logger := logginginfra.NewNoOpLogger()
	configLoader := configinfra.NewYAMLLoader(logger)
	dagBuilder := engineinfra.NewDAGBuilder()
	events := eventsinfra.NewLoggingPublisher(logger)
	metrics := metricsinfra.NewCollector()
	tracer := tracinginfra.NewNoOpTracer()

	registry := plugininfra.NewRegistry()
	registerPlugins(t, registry, plugins...)

	executor := engineinfra.NewExecutor(
		registry,
		engineinfra.WithExecutorLogger(logger),
		engineinfra.WithExecutorMetrics(metrics),
		engineinfra.WithExecutorTracer(tracer),
		engineinfra.WithExecutorEvents(events),
	)

	validationService := applicationvalidation.NewService(logger)
	prepare := applicationpipeline.NewPrepareUseCase(configLoader, dagBuilder, logger, tracer, events)
	apply := applicationpipeline.NewApplyUseCase(prepare, executor, validationService, logger, metrics, tracer, events)
	verify := applicationpipeline.NewVerifyUseCase(prepare, executor, logger, metrics, tracer, events)

	return &appHarness{
		Logger:         logger,
		Events:         events,
		Metrics:        metrics,
		Tracer:         tracer,
		Registry:       registry,
		Executor:       executor,
		ConfigLoader:   configLoader,
		PrepareUseCase: prepare,
		ApplyUseCase:   apply,
		VerifyUseCase:  verify,
	}
}

// RegisterPlugin registers an additional ports.Plugin with the harness registry.
func (h *appHarness) RegisterPlugin(t *testing.T, plugin ports.Plugin) {
	t.Helper()
	require.NoError(t, h.Registry.Register(plugin))
	require.NoError(t, h.Registry.ValidateDependencies())
	require.NoError(t, h.Registry.InitializePlugins())
}

func registerPlugins(t *testing.T, registry *plugininfra.Registry, plugins ...ports.Plugin) {
	t.Helper()

	for _, plugin := range plugins {
		if plugin == nil {
			continue
		}
		require.NoError(t, registry.Register(plugin))
	}
	require.NoError(t, registry.ValidateDependencies())
	require.NoError(t, registry.InitializePlugins())
}

func builtinPortPlugins() []ports.Plugin {
	return []ports.Plugin{
		commandplugin.New(),
		copyplugin.New(),
		lineinfileplugin.New(),
		packageplugin.New(),
		repoplugin.New(),
		symlinkplugin.New(),
		templateplugin.New(),
	}
}
