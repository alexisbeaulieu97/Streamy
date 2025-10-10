package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

type listOptions struct {
	jsonOutput bool
}

func newListCmd(rootFlags *rootFlags) *cobra.Command {
	opts := &listOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List registered Streamy pipelines",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.jsonOutput, "json", false, "Output in JSON format")

	return cmd
}

func runList(cmd *cobra.Command, opts *listOptions) error {
	registryPath, err := defaultRegistryPath()
	if err != nil {
		return newCommandError("list", "determining registry path", err, "Ensure your HOME directory is set correctly.")
	}

	statusPath, err := defaultStatusCachePath()
	if err != nil {
		return newCommandError("list", "determining status cache path", err, "Ensure your HOME directory is set correctly.")
	}

	reg, err := registry.NewRegistry(registryPath)
	if err != nil {
		return newCommandError("list", "loading pipeline registry", err, "Check registry file permissions and try again.")
	}

	pipelines := reg.List()
	if len(pipelines) == 0 {
		return renderEmptyList(cmd)
	}

	statusCache, err := registry.NewStatusCache(statusPath)
	if err != nil {
		return newCommandError("list", "loading status cache", err, "Check status cache file permissions and try again.")
	}

	enriched := enrichPipelinesWithStatus(pipelines, statusCache)

	if opts.jsonOutput {
		return renderListJSON(cmd, enriched)
	}

	return renderListTable(cmd, enriched)
}

type pipelineWithStatus struct {
	Pipeline registry.Pipeline
	Status   registry.CachedStatus
}

func enrichPipelinesWithStatus(pipelines []registry.Pipeline, cache *registry.StatusCache) []pipelineWithStatus {
	enriched := make([]pipelineWithStatus, len(pipelines))

	for i, p := range pipelines {
		status, ok := cache.Get(p.ID)
		if !ok {
			status = registry.CachedStatus{Status: registry.StatusUnknown}
		}

		enriched[i] = pipelineWithStatus{
			Pipeline: p,
			Status:   status,
		}
	}

	sort.Slice(enriched, func(i, j int) bool {
		return enriched[i].Pipeline.ID < enriched[j].Pipeline.ID
	})

	return enriched
}

func renderEmptyList(cmd *cobra.Command) error {
	fmt.Fprintln(cmd.OutOrStdout(), "No pipelines registered yet.")
	fmt.Fprintln(cmd.OutOrStdout(), "\nRun 'streamy registry add <config-path>' to add your first pipeline.")
	return nil
}

func renderListTable(cmd *cobra.Command, pipelines []pipelineWithStatus) error {
	writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)

	fmt.Fprintln(writer, "ID\tNAME\tSTATUS\tLAST RUN\tPATH")

	useUnicode := supportsUnicode(cmd.OutOrStdout())

	for _, p := range pipelines {
		statusStr := formatStatus(p.Status.Status, useUnicode)
		lastRun := formatRelativeTime(p.Status.LastRun)

		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n",
			p.Pipeline.ID,
			valueOrFallback(p.Pipeline.Name, "(no name)"),
			statusStr,
			lastRun,
			p.Pipeline.Path,
		)
	}

	return writer.Flush()
}

type listJSONPipeline struct {
	ID           string                  `json:"id"`
	Name         string                  `json:"name"`
	Path         string                  `json:"path"`
	Description  string                  `json:"description"`
	RegisteredAt time.Time               `json:"registered_at"`
	Status       registry.PipelineStatus `json:"status"`
	LastRun      time.Time               `json:"last_run"`
	Summary      string                  `json:"summary"`
	StepCount    int                     `json:"step_count"`
	FailedSteps  []string                `json:"failed_steps,omitempty"`
}

type listJSONPayload struct {
	Version   string             `json:"version"`
	Count     int                `json:"count"`
	Pipelines []listJSONPipeline `json:"pipelines"`
}

func renderListJSON(cmd *cobra.Command, pipelines []pipelineWithStatus) error {
	payload := listJSONPayload{
		Version:   "1.0",
		Count:     len(pipelines),
		Pipelines: make([]listJSONPipeline, len(pipelines)),
	}

	for i, p := range pipelines {
		payload.Pipelines[i] = listJSONPipeline{
			ID:           p.Pipeline.ID,
			Name:         p.Pipeline.Name,
			Path:         p.Pipeline.Path,
			Description:  p.Pipeline.Description,
			RegisteredAt: p.Pipeline.RegisteredAt,
			Status:       p.Status.Status,
			LastRun:      p.Status.LastRun,
			Summary:      p.Status.Summary,
			StepCount:    p.Status.StepCount,
			FailedSteps:  p.Status.FailedSteps,
		}
	}

	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

func supportsUnicode(writer any) bool {
	if file, ok := writer.(*os.File); ok {
		return term.IsTerminal(int(file.Fd()))
	}
	return false
}

func formatStatus(status registry.PipelineStatus, useUnicode bool) string {
	if useUnicode {
		return fmt.Sprintf("%s %s", status.Icon(), status.String())
	}

	return fmt.Sprintf("%s %s", status.IconFallback(), status.String())
}

func formatRelativeTime(ts time.Time) string {
	if ts.IsZero() {
		return "never"
	}

	delta := time.Since(ts)
	if delta < time.Minute {
		return "just now"
	}
	if delta < time.Hour {
		return fmt.Sprintf("%d minutes ago", int(delta.Minutes()))
	}
	if delta < 24*time.Hour {
		return fmt.Sprintf("%d hours ago", int(delta.Hours()))
	}

	return fmt.Sprintf("%d days ago", int(delta.Hours()/24))
}

func valueOrFallback(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}
