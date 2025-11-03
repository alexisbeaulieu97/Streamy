package pipeline

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// OrchestrationPlan describes the execution order for a dependency-aware pipeline run.
type OrchestrationPlan struct {
	RootPipelineID string
	Levels         [][]string
	GeneratedAt    time.Time
}

// Format returns a human-readable summary of the orchestration levels.
func (p *OrchestrationPlan) Format() string {
	if p == nil {
		return "<nil orchestration plan>"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Orchestration Plan for %s\n", p.RootPipelineID)

	for i, level := range p.Levels {
		fmt.Fprintf(&b, "Level %d: %s\n", i, strings.Join(level, ", "))
	}

	return b.String()
}

// ExecutionSummary captures the outcome of a pipeline within an orchestration run.
type ExecutionSummary struct {
	PipelineID string
	Status     string
	Success    bool
	BlockedBy  string
	Forced     bool
	ForcedBy   string
	Summary    string
	StartedAt  time.Time
	FinishedAt time.Time
	Duration   time.Duration
}

// OrchestrationResult aggregates execution summaries for a dependency-aware run.
type OrchestrationResult struct {
	Plan            *OrchestrationPlan
	PipelineResults map[string]*ExecutionSummary
	StartedAt       time.Time
	FinishedAt      time.Time
	ExecutedCount   int
	ReadyCount      int
	BlockedCount    int
	FailedCount     int
	OverallSuccess  bool
}

// Executed returns the list of pipeline identifiers that ran successfully.
func (r *OrchestrationResult) Executed() []string {
	if r == nil {
		return nil
	}

	capacity := 0
	if r.PipelineResults != nil {
		capacity = len(r.PipelineResults)
	}

	executed := make([]string, 0, capacity)

	for id, summary := range r.PipelineResults {
		if summary != nil && summary.Success {
			executed = append(executed, id)
		}
	}

	slices.Sort(executed)

	return executed
}

// Summary returns a human-readable overview of the orchestration outcome.
func (r *OrchestrationResult) Summary() string {
	if r == nil {
		return "<nil orchestration result>"
	}

	summary := fmt.Sprintf(
		"Orchestration completed: %d executed, %d ready, %d blocked, %d failed",
		r.ExecutedCount,
		r.ReadyCount,
		r.BlockedCount,
		r.FailedCount,
	)

	if !r.StartedAt.IsZero() && !r.FinishedAt.IsZero() {
		duration := r.FinishedAt.Sub(r.StartedAt)
		if duration >= 0 {
			summary = fmt.Sprintf("%s, in %s", summary, duration.Truncate(time.Millisecond))
		} else {
			summary = fmt.Sprintf("%s, duration unavailable", summary)
		}
	} else {
		summary = fmt.Sprintf("%s, duration unavailable", summary)
	}

	return summary
}
