package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/logger"
	"github.com/alexisbeaulieu97/streamy/internal/model"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

// ApplyPipelineResult contains the results of a pipeline apply operation
type ApplyPipelineResult struct {
	Status      registry.PipelineStatus
	Summary     string
	StepCount   int
	FailedSteps []string
	Duration    time.Duration
	StepResults []registry.StepResult
	Error       *registry.ErrorDetail
}

// ApplyPipeline runs apply operation for a single pipeline configuration
func ApplyPipeline(ctx context.Context, configPath string, pluginRegistry *plugin.PluginRegistry) (*ApplyPipelineResult, error) {
	startTime := time.Now()

	// Parse configuration
	cfg, err := config.ParseConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Build DAG
	graph, err := BuildDAG(cfg.Steps)
	if err != nil {
		return nil, fmt.Errorf("failed to build DAG: %w", err)
	}

	// Generate plan
	plan, err := GeneratePlan(graph)
	if err != nil {
		return nil, fmt.Errorf("failed to generate plan: %w", err)
	}

	// Create logger with minimal output for background operations
	log, err := logger.New(logger.Options{Level: "error", HumanReadable: false})
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	// Determine parallelism
	parallel := cfg.Settings.Parallel
	if parallel <= 0 {
		parallel = 4
	}

	// Create execution context
	execCtx := &ExecutionContext{
		Config:          cfg,
		DryRun:          false,
		Verbose:         false,
		ContinueOnError: cfg.Settings.ContinueOnError,
		WorkerPool:      make(chan struct{}, parallel),
		Results:         make(map[string]*model.StepResult),
		Logger:          log,
		Context:         ctx,
		Registry:        pluginRegistry,
	}

	// Execute the plan
	results, err := Execute(execCtx, plan)

	// Convert results
	result := &ApplyPipelineResult{
		StepCount:   len(cfg.Steps),
		Duration:    time.Since(startTime),
		StepResults: make([]registry.StepResult, 0),
		FailedSteps: make([]string, 0),
	}

	// Check for errors
	if err != nil {
		result.Status = registry.StatusFailed
		result.Summary = fmt.Sprintf("Apply failed: %s", err.Error())
		result.Error = &registry.ErrorDetail{
			Code:       "APPLY_FAILED",
			Message:    err.Error(),
			Context:    fmt.Sprintf("Config: %s", configPath),
			Suggestion: "Review error and consider manual intervention",
		}
		return result, nil
	}

	// Check step results
	hasFailures := false
	for _, stepResult := range results {
		if stepResult.Status == model.StatusFailed {
			hasFailures = true
			result.FailedSteps = append(result.FailedSteps, stepResult.StepID)
		}
	}

	if hasFailures {
		result.Status = registry.StatusFailed
		result.Summary = fmt.Sprintf("%d steps failed", len(result.FailedSteps))
		result.Error = &registry.ErrorDetail{
			Code:       "APPLY_FAILED",
			Message:    fmt.Sprintf("Steps failed: %v", result.FailedSteps),
			Context:    fmt.Sprintf("Config: %s", configPath),
			Suggestion: "Check logs for step failure details",
		}
	} else {
		result.Status = registry.StatusSatisfied
		result.Summary = fmt.Sprintf("All %d steps applied successfully", len(results))
	}

	return result, nil
}
