package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	applicationpipeline "github.com/alexisbeaulieu97/streamy/internal/application/pipeline"
	applicationvalidation "github.com/alexisbeaulieu97/streamy/internal/application/validation"
	configinfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/config"
	engineinfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/engine"
	eventsinfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/events"
	logginginfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/logging"
	metricsinfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/metrics"
	plugininfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/plugin"
	tracinginfra "github.com/alexisbeaulieu97/streamy/internal/infrastructure/tracing"
	"github.com/spf13/pflag"
)

func main() {
	os.Exit(run())
}

func run() int {
	preflightFlags := pflag.NewFlagSet("streamy-preflight", pflag.ContinueOnError)
	preflightFlags.ParseErrorsWhitelist.UnknownFlags = true
	preflightFlags.SetOutput(io.Discard)

	var verbose bool
	preflightFlags.BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")

	if err := preflightFlags.Parse(os.Args[1:]); err != nil {
		if !errors.Is(err, pflag.ErrHelp) {
			fmt.Fprintf(os.Stderr, "failed to parse flags: %v\n", err)
			return 1
		}
	}

	defaultLevel := "warn"
	if verbose {
		defaultLevel = "info"
	}

	appLogger, err := logginginfra.New(logginginfra.Options{
		Writer:    os.Stderr,
		Level:     defaultLevel,
		Component: "cli",
		Layer:     "infrastructure",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create application logger: %v\n", err)
		return 1
	}

	correlationID := logginginfra.GenerateCorrelationID()
	baseCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	ctx := logginginfra.WithCorrelationID(baseCtx, correlationID)

	cleanup := newCleanupStack(appLogger.With("component", "shutdown"))

	cleanup.Register("release-signal-handler", func(context.Context) error {
		stop()
		return nil
	})

	cleanup.Register("log-shutdown", func(cleanupCtx context.Context) error {
		appLogger.Info(cleanupCtx, "streamy shutdown complete", "pid", os.Getpid())
		return nil
	})
	defer cleanup.Run(ctx)

	configLoader := configinfra.NewYAMLLoader(appLogger.With("component", "yaml_loader"))
	dagBuilder := engineinfra.NewDAGBuilder()
	eventPublisher := eventsinfra.NewLoggingPublisher(appLogger.With("component", "event_publisher"))
	metricsCollector := metricsinfra.NewCollector()
	tracer := tracinginfra.NewTracer(appLogger.With("component", "tracer"))

	portsRegistry := plugininfra.NewRegistry()
	if err := RegisterPortsPlugins(ctx, portsRegistry, appLogger.With("component", "plugin_registry")); err != nil {
		appLogger.Error(ctx, "failed to register ports plugins", "error", err)
		fmt.Fprintf(os.Stderr, "failed to register ports plugins: %v\n", err)

		return 1
	}

	executor := engineinfra.NewExecutor(
		portsRegistry,
		engineinfra.WithExecutorLogger(appLogger.With("component", "executor")),
		engineinfra.WithExecutorMetrics(metricsCollector),
		engineinfra.WithExecutorTracer(tracer),
		engineinfra.WithExecutorEvents(eventPublisher),
	)
	validationService := applicationvalidation.NewService(appLogger.With("component", "validation_service"))

	prepareUseCase := applicationpipeline.NewPrepareUseCase(
		configLoader,
		dagBuilder,
		appLogger.With("component", "prepare_usecase"),
		tracer,
		eventPublisher,
	)
	applyUseCase := applicationpipeline.NewApplyUseCase(
		prepareUseCase,
		executor,
		validationService,
		appLogger.With("component", "apply_usecase"),
		metricsCollector,
		tracer,
		eventPublisher,
	)
	verifyUseCase := applicationpipeline.NewVerifyUseCase(
		prepareUseCase,
		executor,
		appLogger.With("component", "verify_usecase"),
		metricsCollector,
		tracer,
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

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		appLogger.Error(ctx, "streamy command failed", "error", err)
		fmt.Fprintln(os.Stderr, FormatError(err))

		return 1
	}

	return 0
}
