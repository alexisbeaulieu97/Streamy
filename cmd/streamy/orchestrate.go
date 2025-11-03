package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	apporchestration "github.com/alexisbeaulieu97/streamy/internal/application/orchestration"
	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

type orchestrateOptions struct {
	PipelineID     string
	DryRun         bool
	Force          bool
	NonInteractive bool
	AssumeYes      bool
	Timeout        time.Duration
}

func validateOrchestrateOptions(opts orchestrateOptions) error {
	if opts.PipelineID == "" {
		return fmt.Errorf("pipeline ID is required")
	}

	if opts.Force && opts.NonInteractive && !opts.AssumeYes {
		return fmt.Errorf("--force requires --yes when running non-interactively")
	}

	return nil
}

func runOrchestrate(ctx context.Context, app *AppContext, opts orchestrateOptions, logger ports.Logger, out io.Writer) error {
	execCtx, cancel := deriveExecutionContext(ctx, opts.Timeout)
	defer cancel()

	reg, statusCache, err := loadOrchestrationStores()
	if err != nil {
		return err
	}

	resolvedID, resolveErr := reg.ResolveCanonicalID(opts.PipelineID)
	if resolveErr != nil {
		return newCommandError("run", "resolving pipeline reference", resolveErr, "Ensure the pipeline exists in the registry and has at least one registered version.")
	}

	if !strings.EqualFold(resolvedID, opts.PipelineID) {
		message := fmt.Sprintf("Resolved %s to %s", opts.PipelineID, resolvedID)
		if out != nil {
			fmt.Fprintln(out, message)
		}
		if logger != nil {
			logger.Info(execCtx, "command.orchestrate.resolve", "input", opts.PipelineID, "resolved_pipeline_id", resolvedID)
		}
	}

	opts.PipelineID = resolvedID

	var executor apporchestration.PipelineExecutor
	if app.ApplyUseCase != nil {
		executor = app.ApplyUseCase
	}

	useCase, err := apporchestration.NewUseCase(reg, statusCache, executor)
	if err != nil {
		return newCommandError("orchestrate", "initialising orchestration use case", err, "Ensure the registry is available and the executor is configured before running orchestration.")
	}

	if opts.DryRun {
		return executeDryRun(execCtx, useCase, opts, out)
	}

	if executor == nil {
		return newCommandError("orchestrate", "executing pipelines", fmt.Errorf("apply use case not configured"), "Rebuild the AppContext with ApplyUseCase before running orchestration.")
	}

	if opts.Force {
		if err := ensureForceAcknowledged(execCtx, useCase, statusCache, opts.PipelineID, opts.NonInteractive, opts.AssumeYes, out); err != nil {
			return err
		}
	}

	result, execErr := useCase.Execute(execCtx, opts.PipelineID, false, opts.Force)
	if execErr != nil {
		return newCommandError("orchestrate", "executing orchestration", execErr, "Inspect the error details, resolve issues, and retry.")
	}

	if err := renderOrchestrationResult(result, out); err != nil {
		return err
	}

	if logger != nil {
		logger.Info(execCtx, "command.orchestrate.complete", "pipeline_id", opts.PipelineID, "success", result.OverallSuccess)
	}

	if !result.OverallSuccess {
		return fmt.Errorf("orchestration completed with failures")
	}

	return nil
}

func ensureForceAcknowledged(ctx context.Context, useCase *apporchestration.UseCase, statusCache *registry.StatusCache, pipelineID string, nonInteractive, assumeYes bool, out io.Writer) error {
	plan, planErr := useCase.Plan(ctx, pipelineID)
	if planErr != nil {
		return newCommandError("orchestrate", "building orchestration plan", planErr, "Ensure the pipeline exists and its dependencies are valid.")
	}

	targets := collectForceTargets(plan, statusCache)
	if len(targets) == 0 {
		if !nonInteractive && !assumeYes {
			printForceTargets(out, nil)
		}

		return nil
	}

	if nonInteractive || assumeYes {
		printForceTargets(out, targets)
		return nil
	}

	confirmed, confirmErr := promptForceConfirmation(out, targets)
	if confirmErr != nil {
		return newCommandError("orchestrate", "confirming force execution", confirmErr, "Retry the command or rerun with --yes to skip confirmation.")
	}

	if !confirmed {
		return fmt.Errorf("orchestration cancelled by user")
	}

	return nil
}

func loadOrchestrationStores() (*registry.Registry, *registry.StatusCache, error) {
	registryPath, err := defaultRegistryPath()
	if err != nil {
		return nil, nil, newCommandError("orchestrate", "determining registry path", err, "Ensure your HOME directory is set correctly.")
	}

	statusCachePath, err := defaultStatusCachePath()
	if err != nil {
		return nil, nil, newCommandError("orchestrate", "determining status cache path", err, "Ensure your HOME directory is set correctly.")
	}

	reg, err := registry.NewRegistry(registryPath)
	if err != nil {
		return nil, nil, newCommandError("orchestrate", "loading pipeline registry", err, "Check registry file permissions and try again.")
	}

	statusCache, err := registry.NewStatusCache(statusCachePath)
	if err != nil {
		return nil, nil, newCommandError("orchestrate", "loading status cache", err, "Check status cache file permissions and try again.")
	}

	return reg, statusCache, nil
}

func executeDryRun(ctx context.Context, useCase *apporchestration.UseCase, opts orchestrateOptions, out io.Writer) error {
	plan, planErr := useCase.Plan(ctx, opts.PipelineID)
	if planErr != nil {
		return newCommandError("orchestrate", "building orchestration plan", planErr, "Ensure the pipeline exists and its dependencies are valid.")
	}

	if out == nil {
		out = os.Stdout
	}

	_, _ = fmt.Fprintln(out, plan.Format())

	return nil
}

func renderOrchestrationResult(result *domainpipeline.OrchestrationResult, out io.Writer) error {
	if result == nil {
		return fmt.Errorf("no orchestration result to render")
	}

	if out == nil {
		out = os.Stdout
	}

	_, _ = fmt.Fprintln(out, result.Summary())

	for _, pipelineID := range flattenLevels(result.Plan.Levels) {
		summary := result.PipelineResults[pipelineID]
		if summary == nil {
			continue
		}

		status := registry.PipelineStatus(summary.Status)
		icon := iconForStatus(status)

		line := fmt.Sprintf("%s %s - %s", icon, pipelineID, summary.Summary)
		if summary.Forced {
			if summary.ForcedBy != "" {
				line = fmt.Sprintf("%s [FORCED: %s]", line, summary.ForcedBy)
			} else {
				line = fmt.Sprintf("%s [FORCED]", line)
			}
		} else if summary.BlockedBy != "" && summary.BlockedBy != pipelineID {
			line = fmt.Sprintf("%s (blocked by %s)", line, summary.BlockedBy)
		}

		_, _ = fmt.Fprintln(out, line)
	}

	return nil
}

func flattenLevels(levels [][]string) []string {
	var order []string
	for _, level := range levels {
		order = append(order, level...)
	}

	return order
}

func iconForStatus(status registry.PipelineStatus) string {
	switch status {
	case registry.StatusReady:
		return "🟢"
	case registry.StatusFailed:
		return "🔴"
	default:
		return "🟠"
	}
}

type forceTarget struct {
	ID        string
	BlockedBy string
	Summary   string
}

func collectForceTargets(plan *domainpipeline.OrchestrationPlan, cache *registry.StatusCache) []forceTarget {
	if plan == nil || cache == nil {
		return nil
	}

	seen := make(map[string]struct{})

	targets := make([]forceTarget, 0)

	for _, pipelineID := range flattenLevels(plan.Levels) {
		if _, ok := seen[pipelineID]; ok {
			continue
		}

		seen[pipelineID] = struct{}{}

		status, ok := cache.Get(pipelineID)
		if !ok {
			continue
		}

		if status.Status != registry.StatusBlocked {
			continue
		}

		targets = append(targets, forceTarget{
			ID:        pipelineID,
			BlockedBy: status.BlockedBy,
			Summary:   status.Summary,
		})
	}

	return targets
}

func printForceTargets(out io.Writer, targets []forceTarget) {
	if out == nil {
		out = os.Stdout
	}

	if len(targets) == 0 {
		_, _ = fmt.Fprintln(out, "No blocked pipelines detected; proceeding with --force execution.")
		return
	}

	_, _ = fmt.Fprintln(out, "WARNING: Executing the following blocked pipelines due to --force:")
	for _, target := range targets {
		line := fmt.Sprintf(" - %s", target.ID)
		if target.BlockedBy != "" {
			line = fmt.Sprintf("%s (blocked by %s)", line, target.BlockedBy)
		} else if target.Summary != "" {
			line = fmt.Sprintf("%s (%s)", line, target.Summary)
		}

		_, _ = fmt.Fprintln(out, line)
	}
}

func promptForceConfirmation(out io.Writer, targets []forceTarget) (bool, error) {
	printForceTargets(out, targets)

	if out == nil {
		out = os.Stdout
	}

	_, _ = fmt.Fprint(out, "Proceed with forced execution? [y/N]: ")

	response, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("read confirmation input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))

	return response == "y" || response == "yes", nil
}
