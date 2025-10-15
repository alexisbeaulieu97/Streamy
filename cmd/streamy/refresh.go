package main

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/spf13/cobra"

	"github.com/alexisbeaulieu97/streamy/internal/app/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/logger"
	"github.com/alexisbeaulieu97/streamy/internal/registry"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

type refreshOptions struct {
	concurrency    int
	pipelineID     string
	dryRun         bool
	timeout        time.Duration
	configPath     string
	perStepTimeout time.Duration
}

type refreshResult struct {
	PipelineID string
	Status     registry.PipelineStatus
	Summary    string
	StepCount  int
	Err        error
	Outcome    *registry.ExecutionResult
}

func newRefreshCmd(rootFlags *rootFlags, app *AppContext) *cobra.Command {
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
			return runRefresh(cmd, opts, app)
		},
	}

	cmd.Flags().IntVarP(&opts.concurrency, "concurrency", "c", 5, "Number of pipelines to verify concurrently")
	cmd.Flags().DurationVar(&opts.timeout, "timeout", time.Minute, "Timeout per pipeline verification (e.g. 45s, 2m)")
	cmd.Flags().StringVar(&opts.configPath, "config-path", "", "Path to configuration file")
	cmd.Flags().DurationVar(&opts.perStepTimeout, "per-step-timeout", 30*time.Second, "Default timeout per step; accepts Go duration strings (e.g. 60s)")

	return cmd
}

func runRefresh(cmd *cobra.Command, opts *refreshOptions, app *AppContext) error {
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

	service := app.Pipeline

	results := verifyPipelines(cmd, service, pipelines, opts.concurrency, opts.timeout, opts.configPath, opts.perStepTimeout)

	updateStatusCache(statusCache, results)

	if err := statusCache.Save(); err != nil {
		return newCommandError("refresh", "saving status cache", err, "Check disk space and file permissions, then retry.")
	}

	summary := summarizeResults(results)
	fmt.Fprintf(cmd.OutOrStdout(), "\nSummary:\n  ✓ Satisfied: %d\n  ✗ Failed:    %d\n  ⚠ Drifted:   %d\n", summary.satisfied, summary.failed, summary.drifted)

	return nil
}

func verifyPipelines(cmd *cobra.Command, service *pipeline.Service, pipelines []registry.Pipeline, concurrency int, timeout time.Duration, configPath string, perStepTimeout time.Duration) []refreshResult {
	if concurrency <= 0 {
		concurrency = 1
	}

	ctx := cmd.Context()
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

			result := refreshPipeline(ctx, service, pipeline, timeout, configPath, perStepTimeout)
			result.PipelineID = pipeline.ID

			fmt.Fprintf(out, "%s\n", formatRefreshResult(result))

			results[i] = result
			<-sem
		}()
	}

	wg.Wait()
	return results
}

func refreshPipeline(ctx context.Context, service *pipeline.Service, p registry.Pipeline, timeout time.Duration, configPath string, perStepTimeout time.Duration) refreshResult {
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	prepared, err := service.Prepare(p.Path)
	if err != nil {
		status := registry.StatusFailed
		var validationErr *streamyerrors.ValidationError
		if errors.As(err, &validationErr) {
			return refreshResult{Status: status, Summary: "Configuration validation failed", Err: err}
		}
		return refreshResult{Status: status, Summary: "Configuration load failed", Err: err}
	}

	outcome, verifyErr := service.Verify(ctx, pipeline.VerifyRequest{
		Prepared:       prepared,
		LoggerOptions:  logger.Options{Level: "error", HumanReadable: false},
		DefaultTimeout: perStepTimeout,
		ConfigPath:     configPath,
		PerStepTimeout: perStepTimeout,
	})

	result := refreshResult{
		Status:  registry.StatusFailed,
		Summary: "Verification failed",
		Err:     verifyErr,
		Outcome: nil,
	}

	if outcome != nil && outcome.ExecutionResult != nil {
		result.Outcome = outcome.ExecutionResult
		result.Status = outcome.ExecutionResult.Status
		result.Summary = outcome.ExecutionResult.Summary
		result.StepCount = outcome.ExecutionResult.StepCount
		if result.StepCount == 0 {
			result.StepCount = len(prepared.Config.Steps)
		}
		switch result.Status {
		case registry.StatusSatisfied, registry.StatusDrifted:
			result.Err = nil
		}
	}

	if result.StepCount == 0 {
		result.StepCount = len(prepared.Config.Steps)
	}

	if verifyErr == nil && result.Outcome == nil {
		result.Status = registry.StatusFailed
		result.Summary = "Verification produced no result"
	}

	return result
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
			FailedSteps: nil,
		}
		if r.Outcome != nil {
			status.FailedSteps = append([]string(nil), r.Outcome.FailedSteps...)
		}
		if r.Status == registry.StatusFailed && r.Summary == "" {
			status.Summary = "Verification failed"
		}
		_ = cache.Set(r.PipelineID, status)
	}
}
