package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/spf13/cobra"

	"github.com/alexisbeaulieu97/streamy/internal/pipelineconv"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
	"github.com/alexisbeaulieu97/streamy/internal/registry"
	streamyerrors "github.com/alexisbeaulieu97/streamy/pkg/errors"
)

type refreshOptions struct {
	concurrency    int
	pipelineID     string
	dryRun         bool
	timeout        time.Duration
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

			ctx, logger := app.CommandContext(cmd, "command.registry.refresh")
			if logger != nil {
				logger.Info(ctx, "refresh start", "target_pipeline", opts.pipelineID, "dry_run", opts.dryRun, "concurrency", opts.concurrency)
			}

			err := runRefresh(ctx, logger, cmd, opts, app)
			if err != nil && logger != nil {
				logger.Error(ctx, "refresh command failed", "error", err)
			}

			return err
		},
	}

	cmd.Flags().IntVarP(&opts.concurrency, "concurrency", "c", 5, "Number of pipelines to verify concurrently")
	cmd.Flags().DurationVar(&opts.timeout, "timeout", time.Minute, "Timeout per pipeline verification (e.g. 45s, 2m)")
	cmd.Flags().DurationVar(&opts.perStepTimeout, "per-step-timeout", 30*time.Second, "Default timeout per step; accepts Go duration strings (e.g. 60s)")

	return cmd
}

func runRefresh(ctx context.Context, logger ports.Logger, cmd *cobra.Command, opts *refreshOptions, app *AppContext) error {
	registryPath, statusPath, err := resolveRefreshPaths()
	if err != nil {
		return err
	}

	reg, err := registry.NewRegistry(registryPath)
	if err != nil {
		return newCommandError("refresh", "loading registry", err, "Check registry file permissions and try again.")
	}

	pipelines := reg.List()
	if len(pipelines) == 0 {
		logNoPipelines(ctx, logger, opts.pipelineID)
		notifyNoPipelines(cmd.OutOrStdout())

		return nil
	}

	pipelines, err = filterPipelinesByID(ctx, logger, pipelines, opts.pipelineID)
	if err != nil {
		return err
	}

	if opts.dryRun {
		runRefreshDryRun(ctx, logger, cmd.OutOrStdout(), pipelines)
		return nil
	}

	statusCache, err := registry.NewStatusCache(statusPath)
	if err != nil {
		return newCommandError("refresh", "loading status cache", err, "Check status cache file permissions and try again.")
	}

	logRefreshVerification(ctx, logger, len(pipelines), opts.timeout, opts.perStepTimeout)

	results := verifyPipelines(ctx, logger, cmd, app, pipelines, opts.concurrency, opts.timeout, opts.perStepTimeout)

	updateStatusCache(statusCache, results)

	if err := statusCache.Save(); err != nil {
		return newCommandError("refresh", "saving status cache", err, "Check disk space and file permissions, then retry.")
	}

	summary := summarizeResults(results)
	printRefreshSummary(cmd.OutOrStdout(), summary)
	logRefreshCompleted(ctx, logger, len(pipelines), summary)

	return nil
}

func resolveRefreshPaths() (string, string, error) {
	registryPath, err := defaultRegistryPath()
	if err != nil {
		return "", "", newCommandError("refresh", "determining registry path", err, "Ensure your HOME directory is set correctly.")
	}

	statusPath, err := defaultStatusCachePath()
	if err != nil {
		return "", "", newCommandError("refresh", "determining status cache path", err, "Ensure your HOME directory is set correctly.")
	}

	return registryPath, statusPath, nil
}

func logNoPipelines(ctx context.Context, logger ports.Logger, pipelineID string) {
	if logger == nil {
		return
	}

	logger.Info(ctx, "no pipelines registered for refresh", "pipeline_count", 0, "target_pipeline", pipelineID)
}

func notifyNoPipelines(out io.Writer) {
	_, _ = fmt.Fprintln(out, "No pipelines registered. Run 'streamy registry add <config-path>' first.")
}

func filterPipelinesByID(ctx context.Context, logger ports.Logger, pipelines []registry.Pipeline, targetID string) ([]registry.Pipeline, error) {
	if targetID == "" {
		sortPipelinesByID(pipelines)
		return pipelines, nil
	}

	filtered := make([]registry.Pipeline, 0, 1)

	for _, p := range pipelines {
		if p.ID == targetID {
			filtered = append(filtered, p)
			break
		}
	}

	if len(filtered) == 0 {
		if logger != nil {
			logger.Warn(ctx, "target pipeline not found", "pipeline_id", targetID)
		}

		return nil, newCommandError("refresh", fmt.Sprintf("looking up pipeline %q", targetID), errors.New("pipeline not found"), "Run 'streamy list' to view registered pipelines.")
	}

	sortPipelinesByID(filtered)

	return filtered, nil
}

func sortPipelinesByID(pipelines []registry.Pipeline) {
	sort.Slice(pipelines, func(i, j int) bool {
		return pipelines[i].ID < pipelines[j].ID
	})
}

func runRefreshDryRun(ctx context.Context, logger ports.Logger, out io.Writer, pipelines []registry.Pipeline) {
	if logger != nil {
		logger.Info(ctx, "refresh dry-run", "pipeline_count", len(pipelines))
	}

	_, _ = fmt.Fprintln(out, "Dry-run: Would refresh the following pipelines:")
	for _, p := range pipelines {
		_, _ = fmt.Fprintf(out, "  - %s (%s)\n", p.ID, valueOrFallback(p.Name, "(no name)"))
	}
}

func withOptionalTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, func() {}
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)

	return ctxWithTimeout, cancel
}

func perStepTimeoutDuration(perStepTimeout time.Duration, stepCount int) time.Duration {
	if perStepTimeout <= 0 {
		return 0
	}

	if stepCount <= 0 {
		return perStepTimeout
	}

	return perStepTimeout * time.Duration(stepCount)
}

func preparePipelineForRefresh(ctx context.Context, logger ports.Logger, app *AppContext, p registry.Pipeline) (int, *refreshResult) {
	if logger != nil {
		logger.Debug(ctx, "preparing pipeline", "config_path", p.Path)
	}

	preparedPipeline, _, err := app.PrepareUseCase.Prepare(ctx, p.Path)
	if err != nil {
		if logger != nil {
			logger.Error(ctx, "pipeline preparation failed", "error", err)
		}

		summary := preparationFailureSummary(err)

		return 0, &refreshResult{
			Status:  registry.StatusFailed,
			Summary: summary,
			Err:     err,
		}
	}

	return len(preparedPipeline.Steps), nil
}

func preparationFailureSummary(err error) string {
	var validationErr *streamyerrors.ValidationError
	if errors.As(err, &validationErr) {
		return "Configuration validation failed"
	}

	if pipelineconv.IsConfigError(err) {
		return "Configuration error"
	}

	return "Configuration load failed"
}

func verifyPreparedPipeline(ctx context.Context, logger ports.Logger, app *AppContext, p registry.Pipeline) refreshResult {
	if logger != nil {
		logger.Debug(ctx, "verifying pipeline state", "pipeline_id", p.ID)
	}

	pipelineDomain, verificationResults, err := app.VerifyUseCase.Verify(ctx, p.Path)
	if err != nil {
		if logger != nil {
			logger.Error(ctx, "pipeline verification errored", "error", err)
		}

		return refreshResult{
			Status:  registry.StatusFailed,
			Summary: verificationFailureSummary(ctx, err),
			Err:     err,
		}
	}

	summary := pipelineconv.BuildVerificationSummary(pipelineDomain, verificationResults)
	execResult := pipelineconv.SummaryToExecutionResult(summary, p.Path)

	if logger != nil {
		logger.Info(ctx, "pipeline verification succeeded", "status", execResult.Status, "step_count", execResult.StepCount)
	}

	return refreshResult{
		Status:    execResult.Status,
		Summary:   execResult.Summary,
		StepCount: execResult.StepCount,
		Outcome:   execResult,
	}
}

func verificationFailureSummary(ctx context.Context, err error) string {
	var validationErr *streamyerrors.ValidationError

	switch {
	case errors.As(err, &validationErr):
		return "Configuration validation failed"
	case pipelineconv.IsConfigError(err):
		return "Configuration error"
	case ctx.Err() != nil:
		return "Verification cancelled"
	default:
		return "Verification failed"
	}
}

func logRefreshVerification(ctx context.Context, logger ports.Logger, pipelineCount int, timeout, perStepTimeout time.Duration) {
	if logger == nil {
		return
	}

	logger.Info(ctx, "refresh verifying pipelines", "pipeline_count", pipelineCount, "timeout", timeout.String(), "per_step_timeout", perStepTimeout.String())
}

func logRefreshCompleted(ctx context.Context, logger ports.Logger, pipelineCount int, summary refreshSummary) {
	if logger == nil {
		return
	}

	logger.Info(ctx, "refresh completed", "pipelines", pipelineCount, "satisfied", summary.satisfied, "failed", summary.failed, "drifted", summary.drifted)
}

func printRefreshSummary(out io.Writer, summary refreshSummary) {
	_, _ = fmt.Fprintf(out, "\nSummary:\n  ✓ Satisfied: %d\n  ✗ Failed:    %d\n  ⚠ Drifted:   %d\n", summary.satisfied, summary.failed, summary.drifted)
}

func verifyPipelines(ctx context.Context, logger ports.Logger, cmd *cobra.Command, app *AppContext, pipelines []registry.Pipeline, concurrency int, timeout, perStepTimeout time.Duration) []refreshResult {
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

			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				err := ctx.Err()
				result := refreshResult{
					PipelineID: pipeline.ID,
					Status:     registry.StatusFailed,
					Summary:    verificationFailureSummary(ctx, err),
					Err:        err,
				}
				results[i] = result

				_, _ = fmt.Fprintf(out, "[%d/%d] %s... %s\n", i+1, len(pipelines), pipeline.ID, formatRefreshResult(result))
				if logger != nil {
					logger.With("pipeline_id", pipeline.ID).Error(ctx, "pipeline verification cancelled", "error", err)
				}

				return
			}

			_, _ = fmt.Fprintf(out, "[%d/%d] %s... ", i+1, len(pipelines), pipeline.ID)

			var pipelineLogger ports.Logger
			if logger != nil {
				pipelineLogger = logger.With("pipeline_id", pipeline.ID)
				pipelineLogger.Info(ctx, "pipeline verification started", "pipeline_id", pipeline.ID)
			}

			result := refreshPipeline(ctx, pipelineLogger, app, pipeline, timeout, perStepTimeout)
			result.PipelineID = pipeline.ID

			_, _ = fmt.Fprintf(out, "%s\n", formatRefreshResult(result))

			results[i] = result

			if pipelineLogger != nil {
				if result.Err != nil {
					pipelineLogger.Error(ctx, "pipeline verification finished", "status", result.Status, "error", result.Err)
				} else {
					pipelineLogger.Info(ctx, "pipeline verification finished", "status", result.Status)
				}
			}

			<-sem
		}()
	}

	wg.Wait()

	return results
}

func refreshPipeline(ctx context.Context, logger ports.Logger, app *AppContext, p registry.Pipeline, timeout, perStepTimeout time.Duration) refreshResult {
	ctx, cancelPipeline := withOptionalTimeout(ctx, timeout)
	defer cancelPipeline()

	stepCount, failure := preparePipelineForRefresh(ctx, logger, app, p)
	if failure != nil {
		return *failure
	}

	perStepTotal := perStepTimeoutDuration(perStepTimeout, stepCount)

	ctx, cancelSteps := withOptionalTimeout(ctx, perStepTotal)
	defer cancelSteps()

	return verifyPreparedPipeline(ctx, logger, app, p)
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
