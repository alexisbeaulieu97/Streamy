package pipeline

import (
	"strings"
	"testing"
	"time"

	assert "github.com/stretchr/testify/assert"
)

func TestOrchestrationPlanFormat(t *testing.T) {
	t.Parallel()

	plan := &OrchestrationPlan{
		RootPipelineID: "frontend@1.0",
		Levels: [][]string{
			{"base@1.0", "network@1.0"},
			{"db@1.0"},
			{"frontend@1.0"},
		},
		GeneratedAt: time.Now(),
	}

	out := plan.Format()
	assert.Contains(t, out, "Orchestration Plan for frontend@1.0")
	assert.Contains(t, out, "Level 0: base@1.0, network@1.0")
	assert.Contains(t, out, "Level 2: frontend@1.0")
}

func TestOrchestrationResultExecuted(t *testing.T) {
	t.Parallel()

	result := &OrchestrationResult{
		PipelineResults: map[string]*ExecutionSummary{
			"frontend@1.0": {PipelineID: "frontend@1.0", Success: true},
			"backend@1.0":  {PipelineID: "backend@1.0", Success: false},
			"db@1.0":       nil,
		},
	}

	executed := result.Executed()
	assert.ElementsMatch(t, []string{"frontend@1.0"}, executed)
}

func TestOrchestrationPlanFormatNil(t *testing.T) {
	t.Parallel()

	var plan *OrchestrationPlan
	assert.Equal(t, "<nil orchestration plan>", strings.TrimSpace(plan.Format()))
}

func TestOrchestrationResultSummary(t *testing.T) {
	t.Parallel()

	now := time.Now()

	result := &OrchestrationResult{
		StartedAt:     now,
		FinishedAt:    now.Add(2 * time.Second),
		ExecutedCount: 3,
		ReadyCount:    2,
		BlockedCount:  1,
		FailedCount:   0,
	}

	summary := result.Summary()
	assert.Contains(t, summary, "Executed 3 pipelines")
	assert.Contains(t, summary, "2 ready")
	assert.Contains(t, summary, "1 blocked")
}
