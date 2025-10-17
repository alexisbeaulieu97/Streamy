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
		if logger != nil {
			logger.Info(ctx, "no pipelines registered for refresh")
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No pipelines registered. Run 'streamy registry add <config-path>' first.")
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
			if logger != nil {
				logger.Warn(ctx, "target pipeline not found", "pipeline_id", opts.pipelineID)
			}
			return newCommandError("refresh", fmt.Sprintf("looking up pipeline %q", opts.pipelineID), errors.New("pipeline not found"), "Run 'streamy list' to view registered pipelines.")
		}
		pipelines = filtered
	}

	sort.Slice(pipelines, func(i, j int) bool {
		return pipelines[i].ID < pipelines[j].ID
	})

	if opts.dryRun {
		if logger != nil {
			logger.Info(ctx, "refresh dry-run", "pipeline_count", len(pipelines))
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Dry-run: Would refresh the following pipelines:")
		for _, p := range pipelines {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - %s (%s)\n", p.ID, valueOrFallback(p.Name, "(no name)"))
		}
		return nil
	}

	statusCache, err := registry.NewStatusCache(statusPath)
	if err != nil {
		return newCommandError("refresh", "loading status cache", err, "Check status cache file permissions and try again.")
	}

	if logger != nil {
		logger.Info(ctx, "refresh verifying pipelines", "pipeline_count", len(pipelines), "timeout", opts.timeout.String(), "per_step_timeout", opts.perStepTimeout.String())
	}

	results := verifyPipelines(ctx, logger, cmd, app, pipelines, opts.concurrency, opts.timeout, opts.perStepTimeout)

	updateStatusCache(statusCache, results)

	if err := statusCache.Save(); err != nil {
		return newCommandError("refresh", "saving status cache", err, "Check disk space and file permissions, then retry.")
	}

	summary := summarizeResults(results)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nSummary:\n  ✓ Satisfied: %d\n  ✗ Failed:    %d\n  ⚠ Drifted:   %d\n", summary.satisfied, summary.failed, summary.drifted)

	if logger != nil {
		logger.Info(ctx, "refresh completed", "pipelines", len(pipelines), "satisfied", summary.satisfied, "failed", summary.failed, "drifted", summary.drifted)
	}

	return nil
}

func verifyPipelines(ctx context.Context, logger ports.Logger, cmd *cobra.Command, app *AppContext, pipelines []registry.Pipeline, concurrency int, timeout time.Duration, perStepTimeout time.Duration) []refreshResult {
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
			_, _ = fmt.Fprintf(out, "[%d/%d] %s... ", i+1, len(pipelines), pipeline.ID)

			var pipelineLogger ports.Logger
			if logger != nil {
				pipelineLogger = logger.With("pipeline_id", pipeline.ID)
				pipelineLogger.Info(ctx, "pipeline verification started")
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

func refreshPipeline(ctx context.Context, logger ports.Logger, app *AppContext, p registry.Pipeline, timeout time.Duration, perStepTimeout time.Duration) refreshResult {
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	if logger != nil {
		logger.Debug(ctx, "preparing pipeline", "config_path", p.Path)
	}

	preparedPipeline, _, err := app.PrepareUseCase.Prepare(ctx, p.Path)
	if err != nil {
		if logger != nil {
			logger.Error(ctx, "pipeline preparation failed", "error", err)
		}
		status := registry.StatusFailed
		var validationErr *streamyerrors.ValidationError
		if errors.As(err, &validationErr) {
			return refreshResult{Status: status, Summary: "Configuration validation failed", Err: err}
		}
		if pipelineconv.IsConfigError(err) {
			return refreshResult{Status: status, Summary: "Configuration error", Err: err}
		}
		return refreshResult{Status: status, Summary: "Configuration load failed", Err: err}
	}

	if perStepTimeout > 0 {
		totalTimeout := perStepTimeout * time.Duration(len(preparedPipeline.Steps))
		if len(preparedPipeline.Steps) == 0 {
			totalTimeout = perStepTimeout
		}
		if totalTimeout > 0 {
			ctx, cancel = context.WithTimeout(ctx, totalTimeout)
			defer cancel()
		}
	}

	result := refreshResult{
		Status:  registry.StatusFailed,
		Summary: "Verification failed",
		Err:     nil,
		Outcome: nil,
	}

	if logger != nil {
		logger.Debug(ctx, "verifying pipeline state")
	}

	pipelineDomain, verificationResults, err := app.VerifyUseCase.Verify(ctx, p.Path)
	if err != nil {
		if logger != nil {
			logger.Error(ctx, "pipeline verification errored", "error", err)
		}
		var validationErr *streamyerrors.ValidationError
		if errors.As(err, &validationErr) {
			result.Summary = "Configuration validation failed"
		} else if pipelineconv.IsConfigError(err) {
			result.Summary = "Configuration error"
		} else if ctx.Err() != nil {
			result.Summary = "Verification cancelled"
		} else {
			result.Summary = "Verification failed"
		}
		result.Err = err
		return result
	}

	summary := pipelineconv.BuildVerificationSummary(pipelineDomain, verificationResults)
	execResult := pipelineconv.SummaryToExecutionResult(summary, p.Path)
	result.Outcome = execResult
	result.Status = execResult.Status
	result.Summary = execResult.Summary
	result.StepCount = execResult.StepCount
	result.Err = nil

	if logger != nil {
		logger.Info(ctx, "pipeline verification succeeded", "status", result.Status, "step_count", result.StepCount)
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
