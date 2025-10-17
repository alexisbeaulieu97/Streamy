package main

import (
	"context"

	"github.com/spf13/cobra"

	applicationpipeline "github.com/alexisbeaulieu97/streamy/internal/application/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

// AppContext bundles long-lived services created at startup.
type AppContext struct {
	Logger         ports.Logger
	Events         ports.EventPublisher
	PrepareUseCase *applicationpipeline.PrepareUseCase
	ApplyUseCase   *applicationpipeline.ApplyUseCase
	VerifyUseCase  *applicationpipeline.VerifyUseCase
}

// CommandContext returns the command context (falling back to Background)
// together with a component-scoped logger.
func (a *AppContext) CommandContext(cmd *cobra.Command, component string) (context.Context, ports.Logger) {
	ctx := context.Background()
	if cmd != nil && cmd.Context() != nil {
		ctx = cmd.Context()
	}
	return ctx, a.LoggerFor(component)
}

// LoggerFor derives a child logger with the supplied component name.
func (a *AppContext) LoggerFor(component string) ports.Logger {
	if a == nil || a.Logger == nil {
		return nil
	}
	return a.Logger.With("component", component)
}

// EventPublisher returns the configured event publisher (may be nil during tests).
func (a *AppContext) EventPublisher() ports.EventPublisher {
	if a == nil {
		return nil
	}
	return a.Events
}
