package main

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/alexisbeaulieu97/streamy/internal/ports"
	"github.com/alexisbeaulieu97/streamy/internal/registry"
	"github.com/alexisbeaulieu97/streamy/internal/tui/dashboard"
)

func newDashboardCmd(app *AppContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Launch the interactive dashboard",
		Long:  `Launch the interactive TUI dashboard to view and manage all registered pipelines.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, logger := app.CommandContext(cmd, "command.dashboard")
			if logger != nil {
				logger.Info(ctx, "launching dashboard")
			}
			err := runDashboard(ctx, app, logger)
			if err != nil && logger != nil {
				logger.Error(ctx, "dashboard command failed", "error", err)
			}
			return err
		},
	}

	return cmd
}

func runDashboard(ctx context.Context, app *AppContext, logger ports.Logger) error {
	registryPath, err := defaultRegistryPath()
	if err != nil {
		if logger != nil {
			logger.Error(ctx, "registry path resolution failed", "error", err)
		}
		return fmt.Errorf("failed to determine registry path: %w", err)
	}

	cachePath, err := defaultStatusCachePath()
	if err != nil {
		if logger != nil {
			logger.Error(ctx, "status cache path resolution failed", "error", err)
		}
		return fmt.Errorf("failed to determine status cache path: %w", err)
	}

	// Load registry
	reg, err := registry.NewRegistry(registryPath)
	if err != nil {
		if logger != nil {
			logger.Error(ctx, "failed to load registry", "error", err)
		}
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Load status cache
	cache, err := registry.NewStatusCache(cachePath)
	if err != nil {
		if logger != nil {
			logger.Error(ctx, "failed to load status cache", "error", err)
		}
		return fmt.Errorf("failed to load status cache: %w", err)
	}

	// Get pipelines
	pipelines := reg.List()
	if logger != nil {
		logger.Info(ctx, "dashboard loaded", "pipeline_count", len(pipelines))
	}

	service := newDashboardPipelineAdapter(app)

	// Create dashboard model
	m := dashboard.NewModel(pipelines, reg, cache, service)

	// Create and run Bubble Tea program
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		if logger != nil {
			logger.Error(ctx, "dashboard execution failed", "error", err)
		}
		return fmt.Errorf("failed to run dashboard: %w", err)
	}

	if logger != nil {
		logger.Info(ctx, "dashboard closed")
	}

	return nil
}
