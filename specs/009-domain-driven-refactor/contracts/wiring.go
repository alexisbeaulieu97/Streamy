//go:build ignore
// +build ignore

// Package wiring documents dependency injection patterns for Streamy's
// domain-driven architecture. The content is intentionally expressed as
// comments so the file remains valid Go source while providing richly
// formatted examples for specification readers.
//
// PATTERN 1: Use Case with Port Dependencies
//
//	applyUseCase := applicationpipeline.NewApplyUseCase(
//	    configLoader,  // ports.ConfigLoader
//	    dagBuilder,    // ports.DAGBuilder
//	    planner,       // ports.ExecutionPlanner
//	    executor,      // ports.PluginExecutor
//	    validator,     // ports.ValidationService
//	    logger,        // ports.Logger
//	    metrics,       // ports.MetricsCollector
//	)
//
//	if err := applyUseCase.Apply(ctx, configPath, dryRun); err != nil {
//	    logger.Error(ctx, "apply failed", "error", err)
//	}
//
// PATTERN 2: Adapter with Configuration
//
//	loaderCfg := configinfra.YAMLLoaderConfig{
//	    ValidateSchema: true,
//	    MaxFileSize:    10 * 1024 * 1024,
//	}
//	loader := configinfra.NewYAMLLoader(loaderCfg, logger)
//
// PATTERN 3: Factory Functions for Complex Object Graphs
//
//	executorFactory := engineinfra.NewExecutorFactory(
//	    registry, logger, metrics, tracer, events,
//	)
//	executor := executorFactory.Create(engineinfra.WithParallelism(8))
//
// PATTERN 4: Composition Root in cmd/streamy/main.go
//
//	registry := plugininfra.NewRegistry()
//	_ = RegisterPortsPlugins(ctx, registry, logger)
//
//	prepare := applicationpipeline.NewPrepareUseCase(
//	    loader, dagBuilder, logger, tracer, events,
//	)
//	applyUseCase := applicationpipeline.NewApplyUseCase(
//	    prepare, executor, validationService, logger, metrics, tracer, events,
//	)
//	verifyUseCase := applicationpipeline.NewVerifyUseCase(
//	    prepare, executor, logger, metrics, tracer, events,
//	)
//
// PATTERN 5: Testing with Mocks
//
//	harness := tests.NewAppHarness(t)
//	harness.RegisterPlugin(t, fakeplugin.New())
//	err := harness.ApplyUseCase.Apply(ctx, configPath, false)
//
// PATTERN 6: Functional Options for Optional Dependencies
//
//	executor := engineinfra.NewExecutor(
//	    registry,
//	    engineinfra.WithExecutorLogger(logger),
//	    engineinfra.WithExecutorMetrics(metrics),
//	    engineinfra.WithExecutorTracer(tracer),
//	    engineinfra.WithExecutorEvents(events),
//	)
package wiring
