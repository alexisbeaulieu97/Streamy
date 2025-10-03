package engine

import (
	"fmt"
	"strings"
	"time"
)

// ExecutionPlan contains the ordered execution levels for Streamy.
type ExecutionPlan struct {
	Levels []ExecutionLevel
}

// ExecutionLevel represents a set of steps that can run in parallel.
type ExecutionLevel struct {
	StepIDs           []string
	EstimatedDuration time.Duration
}

// GeneratePlan converts a DAG into an execution plan grouped by level.
func GeneratePlan(graph *Graph) (*ExecutionPlan, error) {
	if graph == nil {
		return nil, fmt.Errorf("graph cannot be nil")
	}

	levels := make([]ExecutionLevel, 0, len(graph.Levels))
	for _, ids := range graph.Levels {
		level := ExecutionLevel{
			StepIDs:           append([]string(nil), ids...),
			EstimatedDuration: time.Duration(len(ids))*time.Second + 500*time.Millisecond,
		}
		if level.EstimatedDuration <= 0 {
			level.EstimatedDuration = time.Second
		}
		levels = append(levels, level)
	}

	return &ExecutionPlan{Levels: levels}, nil
}

// String renders a human readable summary of the plan.
func (p *ExecutionPlan) String() string {
	if p == nil {
		return ""
	}

	var b strings.Builder
	for i, level := range p.Levels {
		fmt.Fprintf(&b, "Level %d (%d steps): %s\n", i, len(level.StepIDs), strings.Join(level.StepIDs, ", "))
	}
	return b.String()
}
