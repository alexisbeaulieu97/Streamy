package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

type showOptions struct {
	jsonOutput bool
}

func newShowCmd(rootFlags *rootFlags) *cobra.Command {
	opts := &showOptions{}

	cmd := &cobra.Command{
		Use:   "show <pipeline-id>",
		Short: "Show detailed information about a pipeline",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShow(cmd, args[0], opts)
		},
	}

	cmd.Flags().BoolVar(&opts.jsonOutput, "json", false, "Output pipeline details as JSON")

	return cmd
}

func runShow(cmd *cobra.Command, pipelineID string, opts *showOptions) error {
	if strings.TrimSpace(pipelineID) == "" {
		return newCommandError("show", "validating pipeline ID", errors.New("pipeline ID cannot be empty"), "Provide the pipeline ID you wish to inspect.")
	}

	registryPath, err := defaultRegistryPath()
	if err != nil {
		return newCommandError("show", "determining registry path", err, "Ensure your HOME directory is set correctly.")
	}

	statusPath, err := defaultStatusCachePath()
	if err != nil {
		return newCommandError("show", "determining status cache path", err, "Ensure your HOME directory is set correctly.")
	}

	reg, err := registry.NewRegistry(registryPath)
	if err != nil {
		return newCommandError("show", "loading registry", err, "Check registry file permissions and try again.")
	}

	pipeline, err := reg.Get(pipelineID)
	if err != nil {
		return newCommandError("show", fmt.Sprintf("looking up pipeline %q", pipelineID), err, "Run 'streamy list' to view registered pipelines.")
	}

	statusCache, err := registry.NewStatusCache(statusPath)
	if err != nil {
		return newCommandError("show", "loading status cache", err, "Check status cache file permissions and try again.")
	}

	status, ok := statusCache.Get(pipelineID)
	if !ok {
		status = registry.CachedStatus{Status: registry.StatusUnknown}
	}

	if opts.jsonOutput {
		return renderShowJSON(cmd, pipeline, status)
	}

	return renderShowTable(cmd, pipeline, status)
}

func renderShowTable(cmd *cobra.Command, pipeline registry.Pipeline, status registry.CachedStatus) error {
	fmt.Fprintf(cmd.OutOrStdout(), "Pipeline: %s\n", pipeline.ID)
	fmt.Fprintf(cmd.OutOrStdout(), "Name:     %s\n", valueOrFallback(pipeline.Name, "(no name)"))
	fmt.Fprintf(cmd.OutOrStdout(), "Path:     %s\n", pipeline.Path)
	fmt.Fprintf(cmd.OutOrStdout(), "\nDescription:\n  %s\n\n", valueOrFallback(pipeline.Description, "(none)"))

	fmt.Fprintf(cmd.OutOrStdout(), "Status:   %s\n", formatStatus(status.Status, supportsUnicode(cmd.OutOrStdout())))
	fmt.Fprintf(cmd.OutOrStdout(), "Last Run: %s\n", formatLastRun(status.LastRun))
	fmt.Fprintf(cmd.OutOrStdout(), "Summary:  %s\n", valueOrFallback(status.Summary, "(none)"))
	fmt.Fprintf(cmd.OutOrStdout(), "Steps:    %d\n", status.StepCount)

	if len(status.FailedSteps) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Failed Steps:\n")
		for _, step := range status.FailedSteps {
			fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", step)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\nRegistered: %s\n", pipeline.RegisteredAt.Format(time.RFC3339))
	return nil
}

type showJSONPayload struct {
	ID           string                  `json:"id"`
	Name         string                  `json:"name"`
	Path         string                  `json:"path"`
	Description  string                  `json:"description"`
	RegisteredAt time.Time               `json:"registered_at"`
	Status       registry.PipelineStatus `json:"status"`
	LastRun      *time.Time              `json:"last_run,omitempty"`
	Summary      string                  `json:"summary"`
	StepCount    int                     `json:"step_count"`
	FailedSteps  []string                `json:"failed_steps,omitempty"`
}

func renderShowJSON(cmd *cobra.Command, pipeline registry.Pipeline, status registry.CachedStatus) error {
	payload := showJSONPayload{
		ID:           pipeline.ID,
		Name:         pipeline.Name,
		Path:         pipeline.Path,
		Description:  pipeline.Description,
		RegisteredAt: pipeline.RegisteredAt,
		Status:       status.Status,
		Summary:      status.Summary,
		StepCount:    status.StepCount,
		FailedSteps:  status.FailedSteps,
	}

	if !status.LastRun.IsZero() {
		payload.LastRun = &status.LastRun
	}

	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

func formatLastRun(ts time.Time) string {
	if ts.IsZero() {
		return "never"
	}
	return fmt.Sprintf("%s (%s)", ts.Format(time.RFC3339), formatRelativeTime(ts))
}
