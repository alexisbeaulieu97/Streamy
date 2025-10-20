package engine

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync/atomic"
	"testing"

	"github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
)

func TestDAGBuilderBuildSuccess(t *testing.T) {
	builder := NewDAGBuilder()
	ctx := context.Background()

	steps := []pipeline.Step{
		{ID: "a", Type: pipeline.StepTypeCommand, Enabled: true},
		{ID: "b", Type: pipeline.StepTypeCommand, Enabled: true, DependsOn: []string{"a"}},
		{ID: "c", Type: pipeline.StepTypeCommand, Enabled: true, DependsOn: []string{"a"}},
		{ID: "d", Type: pipeline.StepTypeCommand, Enabled: true, DependsOn: []string{"b", "c"}},
	}

	plan, err := builder.Build(ctx, steps)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	if plan == nil {
		t.Fatalf("expected plan, got nil")
	}

	if len(plan.Levels) != 3 {
		t.Fatalf("expected 3 levels, got %d", len(plan.Levels))
	}

	if plan.TotalSteps != 4 {
		t.Fatalf("expected total steps 4, got %d", plan.TotalSteps)
	}

	assertSameElements(t, plan.Levels[0].StepIDs, []string{"a"})
	assertSameElements(t, plan.Levels[1].StepIDs, []string{"b", "c"})
	assertSameElements(t, plan.Levels[2].StepIDs, []string{"d"})
}

func TestDAGBuilderBuildMissingDependency(t *testing.T) {
	builder := NewDAGBuilder()
	ctx := context.Background()

	steps := []pipeline.Step{
		{ID: "a", Type: pipeline.StepTypeCommand, Enabled: true, DependsOn: []string{"missing"}},
	}

	_, err := builder.Build(ctx, steps)
	if err == nil {
		t.Fatalf("expected error")
	}

	assertDomainErrorCode(t, err, pipeline.ErrCodeDependency)
}

func TestDAGBuilderBuildCycle(t *testing.T) {
	builder := NewDAGBuilder()
	ctx := context.Background()

	steps := []pipeline.Step{
		{ID: "a", Type: pipeline.StepTypeCommand, Enabled: true, DependsOn: []string{"c"}},
		{ID: "b", Type: pipeline.StepTypeCommand, Enabled: true, DependsOn: []string{"a"}},
		{ID: "c", Type: pipeline.StepTypeCommand, Enabled: true, DependsOn: []string{"b"}},
	}

	_, err := builder.Build(ctx, steps)
	if err == nil {
		t.Fatalf("expected cycle error")
	}

	assertDomainErrorCode(t, err, pipeline.ErrCodeCycle)
}

func TestDAGBuilderBuildIgnoresDisabledSteps(t *testing.T) {
	builder := NewDAGBuilder()
	ctx := context.Background()

	steps := []pipeline.Step{
		{ID: "a", Type: pipeline.StepTypeCommand, Enabled: false},
		{ID: "b", Type: pipeline.StepTypeCommand, Enabled: true},
	}

	plan, err := builder.Build(ctx, steps)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	if plan.TotalSteps != 1 {
		t.Fatalf("expected total steps 1, got %d", plan.TotalSteps)
	}

	if len(plan.Levels) != 1 {
		t.Fatalf("expected 1 level, got %d", len(plan.Levels))
	}

	assertSameElements(t, plan.Levels[0].StepIDs, []string{"b"})
}

func TestDAGBuilderBuildComplexBranches(t *testing.T) {
	builder := NewDAGBuilder()
	ctx := context.Background()

	steps := []pipeline.Step{
		{ID: "a", Type: pipeline.StepTypeCommand, Enabled: true},
		{ID: "b", Type: pipeline.StepTypeCommand, Enabled: true, DependsOn: []string{"a"}},
		{ID: "c", Type: pipeline.StepTypeCommand, Enabled: true, DependsOn: []string{"a"}},
		{ID: "d", Type: pipeline.StepTypeCommand, Enabled: true, DependsOn: []string{"b"}},
		{ID: "e", Type: pipeline.StepTypeCommand, Enabled: true, DependsOn: []string{"b", "c"}},
	}

	plan, err := builder.Build(ctx, steps)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	if plan.TotalSteps != 5 {
		t.Fatalf("expected total steps 5, got %d", plan.TotalSteps)
	}

	if len(plan.Levels) != 3 {
		t.Fatalf("expected 3 levels, got %d", len(plan.Levels))
	}

	assertSameElements(t, plan.Levels[0].StepIDs, []string{"a"})
	assertSameElements(t, plan.Levels[1].StepIDs, []string{"b", "c"})
	assertSameElements(t, plan.Levels[2].StepIDs, []string{"d", "e"})
}

func TestDAGBuilderBuildCancelled(t *testing.T) {
	builder := NewDAGBuilder()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	steps := []pipeline.Step{{ID: "a", Type: pipeline.StepTypeCommand, Enabled: true}}

	_, err := builder.Build(ctx, steps)
	if err == nil {
		t.Fatalf("expected cancellation error")
	}

	assertDomainErrorCode(t, err, pipeline.ErrCodeCancelled)
}

func TestDAGBuilderCancellationDuringProcessing(t *testing.T) {
	builder := NewDAGBuilder()

	ctx := &countingContext{
		Context: context.Background(),
		limit:   1000,
	}

	steps := make([]pipeline.Step, 0, 5000)

	for i := 0; i < 5000; i++ {
		id := fmt.Sprintf("step-%d", i)

		step := pipeline.Step{ID: id, Type: pipeline.StepTypeCommand, Enabled: true}
		if i > 0 {
			step.DependsOn = []string{fmt.Sprintf("step-%d", i-1)}
		}

		steps = append(steps, step)
	}

	_, err := builder.Build(ctx, steps)
	if err == nil {
		t.Fatalf("expected cancellation error")
	}

	assertDomainErrorCode(t, err, pipeline.ErrCodeCancelled)
}

type countingContext struct {
	context.Context
	limit int32
	count int32
}

func (c *countingContext) Err() error {
	if err := c.Context.Err(); err != nil {
		return fmt.Errorf("context cancelled: %w", err)
	}

	if atomic.AddInt32(&c.count, 1) >= c.limit {
		return fmt.Errorf("count limit reached: %w", context.Canceled)
	}

	return nil
}

func assertDomainErrorCode(t *testing.T, err error, code pipeline.ErrorCode) {
	t.Helper()

	var domainErr *pipeline.DomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected DomainError, got %T", err)
	}

	if domainErr.Code != code {
		t.Fatalf("expected code %s, got %s", code, domainErr.Code)
	}
}

func assertSameElements(t *testing.T, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}

	sort.Strings(got)
	sort.Strings(want)

	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}
