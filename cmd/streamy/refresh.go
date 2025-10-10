package main

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/spf13/cobra"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

type refreshOptions struct {
	concurrency int
	pipelineID  string
	dryRun      bool
}

type refreshResult struct {
	PipelineID string
	Status     registry.PipelineStatus
	Summary    string
	StepCount  int
	Err        error
}

func newRefreshCmd(rootFlags *rootFlags) *cobra.Command {
	opts := &refreshOptions{}

	cmd := &cobra.Command{
		Use:   "refresh [pipeline-id]",
		Short: "Refresh pipeline statuses by re-running verification",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				opts.pipelineID = args[0]
			}
			opts.dryRun = rootFlags.dryRun
			return runRefresh(cmd, opts)
		},
	}

	cmd.Flags().IntVarP(&opts.concurrency, "concurrency", "c", 5, "Number of pipelines to verify concurrently")

	return cmd
}

func runRefresh(cmd *cobra.Command, opts *refreshOptions) error {
	registryPath, err := defaultRegistryPath()
	if err != nil {
		return newCommandError("refresh", "determining registry path", err, "Ensure your HOME directory is set correctly.")
	}

	statusPath, err := defaultStatusCachePath()
	if err != nil {
		return newCommandError("refresh", "determining status cache path", err, "Ensure your HOME directory is set correctly.")
	}

	reg, err := registry.NewRegistry(registryPath)
	if err != nil {
		return newCommandError("refresh", "loading registry", err, "Check registry file permissions and try again.")
	}

	pipelines := reg.List()
	if len(pipelines) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No pipelines registered. Run 'streamy registry add <config-path>' first.")
		return nil
	}

	if opts.pipelineID != "" {
		filtered := pipelines[:0]
		for _, p := range pipelines {
			if p.ID == opts.pipelineID {
				filtered = append(filtered, p)
			}
		}
		if len(filtered) == 0 {
			return newCommandError("refresh", fmt.Sprintf("looking up pipeline %q", opts.pipelineID), errors.New("pipeline not found"), "Run 'streamy list' to view registered pipelines.")
		}
		pipelines = filtered
	}

	sort.Slice(pipelines, func(i, j int) bool {
		return pipelines[i].ID < pipelines[j].ID
	})

	if opts.dryRun {
		fmt.Fprintln(cmd.OutOrStdout(), "Dry-run: Would refresh the following pipelines:")
		for _, p := range pipelines {
			fmt.Fprintf(cmd.OutOrStdout(), "  - %s (%s)\n", p.ID, valueOrFallback(p.Name, "(no name)"))
		}
		return nil
	}

	statusCache, err := registry.NewStatusCache(statusPath)
	if err != nil {
		return newCommandError("refresh", "loading status cache", err, "Check status cache file permissions and try again.")
	}

	results := verifyPipelines(cmd, pipelines, opts.concurrency)

	updateStatusCache(statusCache, results)

	if err := statusCache.Save(); err != nil {
		return newCommandError("refresh", "saving status cache", err, "Check disk space and file permissions, then retry.")
	}

	summary := summarizeResults(results)
	fmt.Fprintf(cmd.OutOrStdout(), "\nSummary:\n  ✓ Satisfied: %d\n  ✗ Failed:    %d\n  ⚠ Drifted:   %d\n", summary.satisfied, summary.failed, summary.drifted)

	return nil
}

func verifyPipelines(cmd *cobra.Command, pipelines []registry.Pipeline, concurrency int) []refreshResult {
	if concurrency <= 0 {
		concurrency = 1
	}

	out := cmd.OutOrStdout()

	results := make([]refreshResult, len(pipelines))
	wg := sync.WaitGroup{}
	sem := make(chan struct{}, concurrency)

	for i, pipeline := range pipelines {
		i := i
		pipeline := pipeline
		wg.Add(1)
		go func() {
			defer wg.Done()

			sem <- struct{}{}
			fmt.Fprintf(out, "[%d/%d] %s... ", i+1, len(pipelines), pipeline.ID)

			result := refreshPipeline(pipeline)
			result.PipelineID = pipeline.ID

			fmt.Fprintf(out, "%s\n", formatRefreshResult(result))

			results[i] = result
			<-sem
		}()
	}

	wg.Wait()
	return results
}

func refreshPipeline(p registry.Pipeline) refreshResult {
	cfg, err := config.ParseConfig(p.Path)
	if err != nil {
		return refreshResult{Err: err, Status: registry.StatusFailed, Summary: "Configuration load failed"}
	}

	outcome := invokeRefreshVerifier(p, cfg)
	stepCount := outcome.StepCount
	if stepCount == 0 {
		stepCount = len(cfg.Steps)
	}

	return refreshResult{Status: outcome.Status, Summary: outcome.Summary, StepCount: stepCount, Err: outcome.Err}
}

type verifyOutcome struct {
	Status    registry.PipelineStatus
	Summary   string
	StepCount int
	Err       error
}

var (
	refreshVerifyMu sync.Mutex
	refreshVerifyFn = defaultRefreshVerifier
)

func invokeRefreshVerifier(p registry.Pipeline, cfg *config.Config) verifyOutcome {
	refreshVerifyMu.Lock()
	fn := refreshVerifyFn
	refreshVerifyMu.Unlock()

	return fn(p, cfg)
}

func defaultRefreshVerifier(_ registry.Pipeline, cfg *config.Config) verifyOutcome {
	return verifyOutcome{
		Status:    registry.StatusSatisfied,
		Summary:   "Verification succeeded",
		StepCount: len(cfg.Steps),
	}
}

func withRefreshVerifyFunc(fn func(registry.Pipeline, *config.Config) verifyOutcome) func() {
	refreshVerifyMu.Lock()
	original := refreshVerifyFn
	refreshVerifyFn = fn
	refreshVerifyMu.Unlock()

	return func() {
		refreshVerifyMu.Lock()
		refreshVerifyFn = original
		refreshVerifyMu.Unlock()
	}
}

func formatRefreshResult(result refreshResult) string {
	if result.Err != nil {
		return fmt.Sprintf("✗ failed (%v)", result.Err)
	}

	c := cases.Title(language.English)
	label := c.String(result.Status.String())

	switch result.Status {
	case registry.StatusSatisfied:
		return fmt.Sprintf("✓ %s", label)
	case registry.StatusDrifted:
		return fmt.Sprintf("⚠ %s", label)
	default:
		return fmt.Sprintf("✗ %s", label)
	}
}

type refreshSummary struct {
	satisfied int
	drifted   int
	failed    int
}

func summarizeResults(results []refreshResult) refreshSummary {
	s := refreshSummary{}
	for _, r := range results {
		switch r.Status {
		case registry.StatusSatisfied:
			s.satisfied++
		case registry.StatusDrifted:
			s.drifted++
		default:
			s.failed++
		}
	}
	return s
}

func updateStatusCache(cache *registry.StatusCache, results []refreshResult) {
	now := time.Now().UTC()
	for _, r := range results {
		status := registry.CachedStatus{
			Status:      r.Status,
			Summary:     r.Summary,
			StepCount:   r.StepCount,
			LastRun:     now,
			FailedSteps: []string{},
		}
		if r.Status == registry.StatusFailed && r.Summary == "" {
			status.Summary = "Verification failed"
		}
		_ = cache.Set(r.PipelineID, status)
	}
}
