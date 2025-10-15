package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/alexisbeaulieu97/streamy/internal/registry"
	"github.com/alexisbeaulieu97/streamy/internal/tui/dashboard"
)

func newDashboardCmd(app *AppContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Launch the interactive dashboard",
		Long:  `Launch the interactive TUI dashboard to view and manage all registered pipelines.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDashboard(app)
		},
	}

	return cmd
}

func runDashboard(app *AppContext) error {
	registryPath, err := defaultRegistryPath()
	if err != nil {
		return fmt.Errorf("failed to determine registry path: %w", err)
	}

	cachePath, err := defaultStatusCachePath()
	if err != nil {
		return fmt.Errorf("failed to determine status cache path: %w", err)
	}

	// Load registry
	reg, err := registry.NewRegistry(registryPath)
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Load status cache
	cache, err := registry.NewStatusCache(cachePath)
	if err != nil {
		return fmt.Errorf("failed to load status cache: %w", err)
	}

	// Get pipelines
	pipelines := reg.List()

	pipelineSvc := app.Pipeline

	// Create dashboard model
	m := dashboard.NewModel(pipelines, reg, cache, pipelineSvc)

	// Create and run Bubble Tea program
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run dashboard: %w", err)
	}

	return nil
}
