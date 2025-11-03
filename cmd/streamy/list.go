package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/alexisbeaulieu97/streamy/internal/ports"
	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

type listOptions struct {
	jsonOutput bool
	tree       bool
}

func newListCmd(_ *rootFlags, app *AppContext) *cobra.Command {
	opts := &listOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List registered Streamy pipelines",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, logger := app.CommandContext(cmd, "command.registry.list")
			if logger != nil {
				logger.Info(ctx, "listing pipelines", "json", opts.jsonOutput, "tree", opts.tree)
			}

			err := runList(ctx, logger, cmd, opts)
			if err != nil && logger != nil {
				logger.Error(ctx, "list command failed", "error", err)
			}

			return err
		},
	}

	cmd.Flags().BoolVar(&opts.jsonOutput, "json", false, "Output in JSON format")
	cmd.Flags().BoolVar(&opts.tree, "tree", false, "Render dependency tree view")

	return cmd
}

func runList(ctx context.Context, logger ports.Logger, cmd *cobra.Command, opts *listOptions) error {
	if opts.jsonOutput && opts.tree {
		return newCommandError("list", "parsing flags", fmt.Errorf("--json and --tree cannot be combined"), "Choose either --json or --tree output, not both.")
	}

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
		if logger != nil {
			logger.Info(ctx, "no pipelines registered", "pipeline_count", 0)
		}

		return renderEmptyList(cmd)
	}

	statusCache, err := registry.NewStatusCache(statusPath)
	if err != nil {
		return newCommandError("list", "loading status cache", err, "Check status cache file permissions and try again.")
	}

	if opts.tree {
		return renderListTree(cmd, reg, statusCache)
	}

	enriched := enrichPipelinesWithStatus(pipelines, statusCache)

	if opts.jsonOutput {
		if logger != nil {
			logger.Info(ctx, "rendering pipeline list", "format", "json", "pipeline_count", len(enriched))
		}

		return renderListJSON(cmd, enriched)
	}

	if logger != nil {
		logger.Info(ctx, "rendering pipeline list", "format", "table", "pipeline_count", len(enriched))
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
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No pipelines registered yet.")
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nRun 'streamy registry add <config-path>' to add your first pipeline.")

	return nil
}

func renderListTable(cmd *cobra.Command, pipelines []pipelineWithStatus) error {
	writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)

	_, _ = fmt.Fprintln(writer, "ID\tNAME\tSTATUS\tLAST RUN\tPATH")

	useUnicode := supportsUnicode(cmd.OutOrStdout())

	for _, p := range pipelines {
		statusStr := formatStatus(p.Status.Status, useUnicode)
		lastRun := formatRelativeTime(p.Status.LastRun)

		_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n",
			p.Pipeline.ID,
			valueOrFallback(p.Pipeline.Name, "(no name)"),
			statusStr,
			lastRun,
			p.Pipeline.Path,
		)
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush table output: %w", err)
	}

	return nil
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

	if err := encoder.Encode(payload); err != nil {
		return fmt.Errorf("encode pipeline list JSON: %w", err)
	}

	return nil
}

func renderListTree(cmd *cobra.Command, reg *registry.Registry, _ *registry.StatusCache) error {
	out := cmd.OutOrStdout()
	useUnicode := supportsUnicode(out)

	pipelines := reg.PipelinesByID()
	if len(pipelines) == 0 {
		return renderEmptyList(cmd)
	}

	dependents := reg.DependentsIndex()

	roots := make([]string, 0, len(pipelines))
	for id := range pipelines {
		if len(dependents[id]) == 0 {
			roots = append(roots, id)
		}
	}

	if len(roots) == 0 {
		for id := range pipelines {
			roots = append(roots, id)
		}
	}

	sort.Strings(roots)

	ctx := &treeContext{
		writer:      out,
		registry:    reg,
		pipelines:   pipelines,
		useUnicode:  useUnicode,
		visitedPath: make(map[string]bool),
	}

	for i, root := range roots {
		ctx.visitedPath = make(map[string]bool)
		ctx.printNode(root, "", i == len(roots)-1)

		if i != len(roots)-1 {
			_, _ = fmt.Fprintln(out)
		}
	}

	return nil
}

type treeContext struct {
	writer      io.Writer
	registry    *registry.Registry
	pipelines   map[string]registry.Pipeline
	useUnicode  bool
	visitedPath map[string]bool
}

func (t *treeContext) printNode(pipelineID, prefix string, isLast bool) {
	midBranch := "├─"
	lastBranch := "└─"
	continuationPrefix := "│  "
	spacePrefix := "   "

	if !t.useUnicode {
		midBranch = "+-"
		lastBranch = "`-"
		continuationPrefix = "|  "
		spacePrefix = "   "
	}

	branch := midBranch
	nextPrefix := prefix + continuationPrefix

	if isLast {
		branch = lastBranch
		nextPrefix = prefix + spacePrefix
	}

	info := t.nodeInfo(pipelineID)
	line := t.formatLine(pipelineID, info)
	_, _ = fmt.Fprintf(t.writer, "%s%s %s\n", prefix, branch, line)

	if info.missing || len(info.dependencies) == 0 {
		return
	}

	if t.visitedPath[pipelineID] {
		return
	}

	t.visitedPath[pipelineID] = true
	defer delete(t.visitedPath, pipelineID)

	for idx, depID := range info.dependencies {
		t.printNode(depID, nextPrefix, idx == len(info.dependencies)-1)
	}
}

type treeNodeInfo struct {
	status       registry.PipelineStatus
	reason       string
	missing      bool
	dependencies []string
}

func (t *treeContext) nodeInfo(pipelineID string) treeNodeInfo {
	pipeline, exists := t.pipelines[pipelineID]
	if !exists {
		return treeNodeInfo{
			status:  registry.StatusBlocked,
			reason:  "not registered",
			missing: true,
		}
	}

	missing := t.registry.FindUnregisteredDependencies(pipeline.Dependencies)
	if len(missing) > 0 {
		reason := fmt.Sprintf("missing dependencies: %s", strings.Join(missing, ", "))

		deps := append([]string(nil), pipeline.Dependencies...)
		sort.Strings(deps)

		return treeNodeInfo{
			status:       registry.StatusBlocked,
			reason:       reason,
			dependencies: deps,
		}
	}

	deps := append([]string(nil), pipeline.Dependencies...)
	sort.Strings(deps)

	return treeNodeInfo{
		status:       registry.StatusReady,
		dependencies: deps,
	}
}

func (t *treeContext) formatLine(pipelineID string, info treeNodeInfo) string {
	icon := info.status.Icon()
	if !t.useUnicode {
		icon = info.status.IconFallback()
	}

	label := "Ready"
	if info.status != registry.StatusReady {
		label = "Blocked"
	}

	if info.status == registry.StatusBlocked && info.reason != "" {
		return fmt.Sprintf("%s %s (%s: %s)", icon, pipelineID, label, info.reason)
	}

	return fmt.Sprintf("%s %s (%s)", icon, pipelineID, label)
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
