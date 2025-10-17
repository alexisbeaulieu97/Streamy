package main

import (
	"context"
	"fmt"
	"os"

	applicationpipeline "github.com/alexisbeaulieu97/streamy/internal/application/pipeline"
	applicationvalidation "github.com/alexisbeaulieu97/streamy/internal/application/validation"
	configinfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/config"
	engineinfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/engine"
	eventsinfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/events"
	logginginfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/logging"
	plugininfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/plugin"
)

func main() {
	appLogger, err := logginginfra.New(logginginfra.Options{
		Level:     "info",
		Component: "cli",
		Layer:     "infrastructure",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create application logger: %v\n", err)
		os.Exit(1)
	}

	correlationID := logginginfra.GenerateCorrelationID()
	ctx := logginginfra.WithCorrelationID(context.Background(), correlationID)

	configLoader := configinfra.NewYAMLLoader(appLogger.With("component", "yaml_loader"))
	dagBuilder := engineinfra.NewDAGBuilder()
	eventPublisher := eventsinfra.NewLoggingPublisher(appLogger.With("component", "event_publisher"))

	portsRegistry := plugininfra.NewRegistry()
	if err := RegisterPortsPlugins(ctx, portsRegistry, appLogger.With("component", "plugin_registry")); err != nil {
		fmt.Fprintf(os.Stderr, "failed to register ports plugins: %v\n", err)
		os.Exit(1)
	}

	executor := engineinfra.NewExecutor(
		portsRegistry,
		engineinfra.WithExecutorLogger(appLogger.With("component", "executor")),
	)
	validationService := applicationvalidation.NewService(appLogger.With("component", "validation_service"))

	prepareUseCase := applicationpipeline.NewPrepareUseCase(
		configLoader,
		dagBuilder,
		appLogger.With("component", "prepare_usecase"),
		eventPublisher,
	)
	applyUseCase := applicationpipeline.NewApplyUseCase(
		prepareUseCase,
		executor,
		validationService,
		appLogger.With("component", "apply_usecase"),
		eventPublisher,
	)
	verifyUseCase := applicationpipeline.NewVerifyUseCase(
		prepareUseCase,
		executor,
		appLogger.With("component", "verify_usecase"),
		eventPublisher,
	)

	app := &AppContext{
		Logger:         appLogger,
		Events:         eventPublisher,
		PrepareUseCase: prepareUseCase,
		ApplyUseCase:   applyUseCase,
		VerifyUseCase:  verifyUseCase,
	}

	rootCmd := newRootCmd(app)
	appLogger.Info(ctx, "starting streamy command", "pid", os.Getpid())

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
