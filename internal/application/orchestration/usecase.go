// Package orchestration provides application-level orchestration use cases built on top of the registry and pipeline executor.
package orchestration

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

// PipelineExecutor captures the subset of the apply use case needed for orchestration.
type PipelineExecutor interface {
	Apply(ctx context.Context, configPath string, dryRun bool) (*domainpipeline.Pipeline, []domainpipeline.StepResult, *domainpipeline.VerificationSummary, error)
}

// UseCase coordinates dependency-aware pipeline execution.
type UseCase struct {
	registry      *registry.Registry
	statusCache   *registry.StatusCache
	executor      PipelineExecutor
	maxConcurrent int
}

// NewUseCase constructs an orchestration use case with its dependencies.
// The cache parameter may be nil when callers do not require status persistence.
func NewUseCase(reg *registry.Registry, cache *registry.StatusCache, executor PipelineExecutor) (*UseCase, error) {
	if reg == nil {
		return nil, fmt.Errorf("registry is nil")
	}

	if executor == nil {
		return nil, fmt.Errorf("pipeline executor is nil")
	}

	maxConcurrent := runtime.NumCPU()
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}

	return &UseCase{
		registry:      reg,
		statusCache:   cache,
		executor:      executor,
		maxConcurrent: maxConcurrent,
	}, nil
}

// Plan builds a dependency-aware execution plan without running any pipelines.
func (uc *UseCase) Plan(ctx context.Context, rootPipelineID string) (*domainpipeline.OrchestrationPlan, error) {
	plan, _, err := uc.buildPlan(ctx, rootPipelineID)
	return plan, err
}

func (uc *UseCase) buildPlan(ctx context.Context, rootPipelineID string) (*domainpipeline.OrchestrationPlan, *registry.DependencyGraph, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, fmt.Errorf("context cancelled: %w", err)
	}

	graph, err := registry.NewDependencyGraph(uc.registry, rootPipelineID)
	if err != nil {
		return nil, nil, fmt.Errorf("build dependency graph: %w", err)
	}

	levels, err := graph.TopologicalSort()
	if err != nil {
		return nil, nil, fmt.Errorf("topological sort: %w", err)
	}

	plan := &domainpipeline.OrchestrationPlan{
		RootPipelineID: rootPipelineID,
		Levels:         levels,
		GeneratedAt:    time.Now(),
	}

	return plan, graph, nil
}

// Execute runs the dependency graph rooted at the provided pipeline identifier.
func (uc *UseCase) Execute(ctx context.Context, rootPipelineID string, dryRun, force bool) (*domainpipeline.OrchestrationResult, error) {
	startTime := time.Now()

	plan, graph, err := uc.buildPlan(ctx, rootPipelineID)
	if err != nil {
		return nil, err
	}

	result := &domainpipeline.OrchestrationResult{
		Plan:            plan,
		PipelineResults: make(map[string]*domainpipeline.ExecutionSummary),
		StartedAt:       startTime,
	}

	blocked := make(map[string]string)

	forced := make(map[string]string)

	for _, level := range plan.Levels {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("context cancelled: %w", err)
		}

		runnable := make([]string, 0, len(level))
		for _, pipelineID := range level {
			if blockingID, isBlocked := blocked[pipelineID]; isBlocked {
				if !force {
					result.PipelineResults[pipelineID] = blockedSummary(pipelineID, blockingID)
					result.BlockedCount++

					continue
				}

				forced[pipelineID] = blockingID
			}

			runnable = append(runnable, pipelineID)
		}

		if len(runnable) == 0 {
			continue
		}

		levelResults := uc.executeLevel(ctx, runnable, dryRun)

		for pipelineID, summary := range levelResults {
			if summary == nil {
				continue
			}

			result.PipelineResults[pipelineID] = summary
			result.ExecutedCount++

			switch {
			case summary.Success:
				result.ReadyCount++
			case summary.BlockedBy == pipelineID:
				result.FailedCount++

				dependents := graph.GetTransitiveDependents(pipelineID)
				for _, depID := range dependents {
					blocked[depID] = pipelineID
				}
			default:
				result.BlockedCount++
			}

			if forcedBy, wasForced := forced[pipelineID]; wasForced {
				summary.Forced = true
				summary.ForcedBy = forcedBy

				if summary.Success {
					summary.BlockedBy = ""
				}

				summary.Summary = fmt.Sprintf("%s (forced despite %s failure)", summary.Summary, forcedBy)
			}
		}
	}

	result.FinishedAt = time.Now()
	result.OverallSuccess = result.FailedCount == 0

	if err := uc.persistSummary(result); err != nil {
		return nil, fmt.Errorf("persist orchestration summary: %w", err)
	}

	return result, nil
}

func (uc *UseCase) executeLevel(ctx context.Context, pipelineIDs []string, dryRun bool) map[string]*domainpipeline.ExecutionSummary {
	results := make(map[string]*domainpipeline.ExecutionSummary, len(pipelineIDs))
	resultsMu := sync.Mutex{}

	var wg sync.WaitGroup
	sem := make(chan struct{}, uc.maxConcurrency())

	for _, pipelineID := range pipelineIDs {
		wg.Add(1)
		sem <- struct{}{}

		go func(id string) {
			defer wg.Done()
			defer func() { <-sem }()

			start := time.Now()
			summary := &domainpipeline.ExecutionSummary{
				PipelineID: id,
				Status:     registry.StatusReady.String(),
				Success:    true,
				StartedAt:  start,
			}

			if err := ctx.Err(); err != nil {
				summary.Status = registry.StatusFailed.String()
				summary.Success = false
				summary.BlockedBy = id
				summary.Summary = fmt.Sprintf("execution cancelled: %v", err)
				summary.FinishedAt = time.Now()
				summary.Duration = summary.FinishedAt.Sub(summary.StartedAt)
				resultsMu.Lock()
				results[id] = summary
				resultsMu.Unlock()

				return
			}

			pipelineEntry, err := uc.registry.Get(id)
			if err != nil {
				summary.Status = registry.StatusFailed.String()
				summary.Success = false
				summary.BlockedBy = id
				summary.Summary = fmt.Sprintf("failed to load pipeline: %v", err)
			} else if uc.executor == nil {
				summary.Status = registry.StatusFailed.String()
				summary.Success = false
				summary.BlockedBy = id
				summary.Summary = "pipeline executor not configured"
			} else {
				_, _, _, execErr := uc.executor.Apply(ctx, pipelineEntry.Path, dryRun)
				if execErr != nil {
					summary.Status = registry.StatusFailed.String()
					summary.Success = false
					summary.BlockedBy = id
					summary.Summary = execErr.Error()
				} else {
					summary.Status = registry.StatusReady.String()
					summary.Success = true
					summary.Summary = "Pipeline completed successfully"
				}
			}

			summary.FinishedAt = time.Now()
			summary.Duration = summary.FinishedAt.Sub(summary.StartedAt)
			resultsMu.Lock()
			results[id] = summary
			resultsMu.Unlock()
		}(pipelineID)
	}

	wg.Wait()

	return results
}

func (uc *UseCase) persistSummary(result *domainpipeline.OrchestrationResult) error {
	if uc.statusCache == nil || result == nil || result.Plan == nil {
		return nil
	}

	for pipelineID, execSummary := range result.PipelineResults {
		if execSummary == nil {
			continue
		}

		lastRun := execSummary.FinishedAt
		if lastRun.IsZero() {
			lastRun = result.FinishedAt
		}

		status := registry.PipelineStatus(execSummary.Status)

		cached := registry.CachedStatus{
			Status:    status,
			LastRun:   lastRun,
			Summary:   execSummary.Summary,
			BlockedBy: execSummary.BlockedBy,
		}

		if err := uc.statusCache.Set(pipelineID, cached); err != nil {
			return fmt.Errorf("cache status for %s: %w", pipelineID, err)
		}
	}

	order := flattenLevels(result.Plan.Levels)

	summary := registry.OrchestrationSummary{
		RootPipelineID: result.Plan.RootPipelineID,
		StartTime:      result.StartedAt,
		EndTime:        result.FinishedAt,
		TotalPipelines: len(order),
		Executed:       result.ExecutedCount,
		Blocked:        result.BlockedCount,
		Failed:         result.FailedCount,
		PipelineOrder:  order,
	}

	if err := uc.statusCache.AddOrchestration(summary); err != nil {
		return fmt.Errorf("record orchestration summary: %w", err)
	}

	return nil
}

func (uc *UseCase) maxConcurrency() int {
	if uc == nil || uc.maxConcurrent <= 0 {
		n := runtime.NumCPU()
		if n < 1 {
			n = 1
		}

		return n
	}

	return uc.maxConcurrent
}

func flattenLevels(levels [][]string) []string {
	var order []string
	for _, level := range levels {
		order = append(order, level...)
	}

	return order
}

func blockedSummary(pipelineID, blockingID string) *domainpipeline.ExecutionSummary {
	now := time.Now()

	return &domainpipeline.ExecutionSummary{
		PipelineID: pipelineID,
		Status:     registry.StatusBlocked.String(),
		Success:    false,
		BlockedBy:  blockingID,
		Summary:    fmt.Sprintf("Blocked by %s", blockingID),
		StartedAt:  now,
		FinishedAt: now,
		Duration:   0,
	}
}
