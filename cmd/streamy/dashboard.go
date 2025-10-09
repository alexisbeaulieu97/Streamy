package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/alexisbeaulieu97/streamy/internal/registry"
	"github.com/alexisbeaulieu97/streamy/internal/tui/dashboard"
)

func newDashboardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Launch the interactive dashboard",
		Long:  `Launch the interactive TUI dashboard to view and manage all registered pipelines.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDashboard()
		},
	}

	return cmd
}

func runDashboard() error {
	// Determine registry and cache paths
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	streamyDir := filepath.Join(homeDir, ".streamy")
	registryPath := filepath.Join(streamyDir, "registry.json")
	cachePath := filepath.Join(streamyDir, "status-cache.json")

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

	// Get plugin registry
	pluginReg := getAppRegistry()

	// Create dashboard model
	m := dashboard.NewModel(pipelines, reg, cache, pluginReg)

	// Create and run Bubble Tea program
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run dashboard: %w", err)
	}

	return nil
}
