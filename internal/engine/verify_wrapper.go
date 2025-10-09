package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/alexisbeaulieu97/streamy/internal/config"
	"github.com/alexisbeaulieu97/streamy/internal/logger"
	"github.com/alexisbeaulieu97/streamy/internal/plugin"
	"github.com/alexisbeaulieu97/streamy/internal/registry"
)

// VerifyPipelineResult contains the results of a pipeline verification
type VerifyPipelineResult struct {
	Status      registry.PipelineStatus
	Summary     string
	StepCount   int
	FailedSteps []string
	Duration    time.Duration
	StepResults []registry.StepResult
	Error       *registry.ErrorDetail
}

// VerifyPipeline runs verification for a single pipeline configuration
func VerifyPipeline(ctx context.Context, configPath string, pluginRegistry *plugin.PluginRegistry) (*VerifyPipelineResult, error) {
	startTime := time.Now()

	// Parse configuration
	cfg, err := config.ParseConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Create logger with minimal output for background operations
	log, err := logger.New(logger.Options{Level: "error", HumanReadable: false})
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	// Create executor
	executor := NewExecutor(log)

	// Create execution context
	execCtx := &ExecutionContext{
		Config:   cfg,
		DryRun:   true,
		Verbose:  false,
		Logger:   log,
		Context:  ctx,
		Registry: pluginRegistry,
	}

	// Run verification
	perStepTimeout := 30 * time.Second
	summary, err := executor.VerifySteps(execCtx, cfg.Steps, perStepTimeout)
	if err != nil {
		return &VerifyPipelineResult{
			Status:   registry.StatusFailed,
			Summary:  fmt.Sprintf("Verification failed: %s", err.Error()),
			Duration: time.Since(startTime),
			Error: &registry.ErrorDetail{
				Code:       "VERIFY_FAILED",
				Message:    err.Error(),
				Context:    fmt.Sprintf("Config: %s", configPath),
				Suggestion: "Check config file syntax and step definitions",
			},
		}, nil
	}

	// Convert summary to result
	result := &VerifyPipelineResult{
		StepCount:   summary.TotalSteps,
		Duration:    summary.Duration,
		StepResults: make([]registry.StepResult, 0),
	}

	// Determine status
	if summary.Missing > 0 || summary.Drifted > 0 {
		result.Status = registry.StatusDrifted
		result.Summary = fmt.Sprintf("%d steps need changes", summary.Missing+summary.Drifted)
	} else if summary.Blocked > 0 || summary.Unknown > 0 {
		result.Status = registry.StatusFailed
		result.Summary = fmt.Sprintf("%d steps failed or unknown", summary.Blocked+summary.Unknown)
	} else {
		result.Status = registry.StatusSatisfied
		result.Summary = fmt.Sprintf("All %d steps passed", summary.Satisfied)
	}

	return result, nil
}
