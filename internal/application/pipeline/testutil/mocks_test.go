package testutil

import (
	"context"
	"testing"

	require "github.com/stretchr/testify/require"

	domainpipeline "github.com/alexisbeaulieu97/streamy/internal/domain/pipeline"
	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

type dummyEvent struct {
	typ     string
	payload interface{}
}

func (e dummyEvent) EventType() string    { return e.typ }
func (e dummyEvent) Payload() interface{} { return e.payload }

func TestMockConfigLoader(t *testing.T) {
	loader := &MockConfigLoader{
		LoadFunc: func(_ context.Context, path string) (*domainpipeline.Pipeline, error) {
			return &domainpipeline.Pipeline{Steps: []domainpipeline.Step{{ID: path}}}, nil
		},
		ValidateFunc: func(_ context.Context, path string) error {
			require.Equal(t, "config.yml", path)
			return nil
		},
	}

	pipeline, err := loader.Load(context.Background(), "config.yml")
	require.NoError(t, err)
	require.Len(t, pipeline.Steps, 1)
	require.Equal(t, "config.yml", pipeline.Steps[0].ID)

	require.NoError(t, loader.Validate(context.Background(), "config.yml"))

	nilLoader := (*MockConfigLoader)(nil)
	p, err := nilLoader.Load(context.Background(), "ignored")
	require.NoError(t, err)
	require.Nil(t, p)
	require.NoError(t, nilLoader.Validate(context.Background(), "ignored"))
}

func TestMockDAGBuilderRecordsCalls(t *testing.T) {
	steps := []domainpipeline.Step{{ID: "build", Enabled: true}}

	builder := &MockDAGBuilder{
		BuildFunc: func(_ context.Context, s []domainpipeline.Step) (*domainpipeline.ExecutionPlan, error) {
			require.ElementsMatch(t, steps, s)
			return &domainpipeline.ExecutionPlan{Levels: []domainpipeline.ExecutionLevel{{Level: 0, StepIDs: []string{"build"}}}}, nil
		},
	}

	plan, err := builder.Build(context.Background(), steps)
	require.NoError(t, err)
	require.NotNil(t, plan)
	require.Len(t, builder.Calls, 1)
	require.Equal(t, steps, builder.Calls[0].Steps)
}

func TestMockPluginExecutorRecords(t *testing.T) {
	plan := &domainpipeline.ExecutionPlan{Levels: []domainpipeline.ExecutionLevel{{Level: 0, StepIDs: []string{"s"}}}}
	pipeline := &domainpipeline.Pipeline{Steps: []domainpipeline.Step{{ID: "s", Enabled: true}}}

	executor := &MockPluginExecutor{
		ExecuteFunc: func(_ context.Context, p *domainpipeline.ExecutionPlan, pl *domainpipeline.Pipeline) ([]domainpipeline.StepResult, error) {
			require.Equal(t, plan, p)
			require.Equal(t, pipeline, pl)

			return []domainpipeline.StepResult{{StepID: "s", Status: domainpipeline.StatusSuccess}}, nil
		},
		VerifyFunc: func(_ context.Context, pl *domainpipeline.Pipeline) ([]domainpipeline.VerificationResult, error) {
			require.Equal(t, pipeline, pl)

			return []domainpipeline.VerificationResult{{StepID: "s", Status: domainpipeline.VerificationSatisfied}}, nil
		},
	}

	results, err := executor.Execute(context.Background(), plan, pipeline)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "s", results[0].StepID)

	verifications, err := executor.Verify(context.Background(), pipeline)
	require.NoError(t, err)
	require.Len(t, verifications, 1)
	require.Equal(t, "s", verifications[0].StepID)

	require.Len(t, executor.ExecuteCalls, 1)
	require.Len(t, executor.VerifyCalls, 1)
}

func TestMockEventPublisher(t *testing.T) {
	publisher := &MockEventPublisher{}
	event := dummyEvent{typ: "pipeline.started", payload: map[string]string{"id": "p"}}

	require.NoError(t, publisher.Publish(context.Background(), event))
	require.Len(t, publisher.Published, 1)
	require.Equal(t, event.EventType(), publisher.Published[0].Type)
	require.Equal(t, event.Payload(), publisher.Published[0].Payload)

	sub, err := publisher.Subscribe(event.EventType(), nil)
	require.NoError(t, err)
	require.NotNil(t, sub)
	sub.Unsubscribe()
}

func TestMockLoggerWithEntries(t *testing.T) {
	logger := NewMockLogger()
	child := logger.With("pipeline", "p-1").(*MockLogger)

	child.Info(context.Background(), "ready", "result", "ok")

	entries := child.Entries()
	require.Len(t, entries, 1)
	require.Equal(t, "info", entries[0].Level)
	require.Equal(t, "ready", entries[0].Msg)
	require.Equal(t, "p-1", entries[0].Fields["pipeline"])
	require.Equal(t, "ok", entries[0].Fields["result"])

	// Entries should be a copy; modifying the returned slice must not affect internal state.
	entriesCopy := append([]LogEntry{}, entries...)
	entriesCopy = append(entriesCopy, LogEntry{Msg: "extra"})

	require.Len(t, entriesCopy, 2)
	require.Len(t, child.Entries(), 1)

	// Parent logger should remain unaffected by child writes.
	require.Empty(t, logger.Entries())
}

func TestMockMetricsCollectorClonesLabels(t *testing.T) {
	collector := NewMockMetricsCollector()
	labels := map[string]string{"step": "apply"}

	collector.IncCounter(context.Background(), "runs", labels)
	collector.SetGauge(context.Background(), "duration", 42.0, labels)
	collector.ObserveHistogram(context.Background(), "latency", 1.5, labels)

	labels["step"] = "mutated"

	require.Len(t, collector.CounterCalls, 1)
	require.Equal(t, "apply", collector.CounterCalls[0].Labels["step"])

	require.Len(t, collector.GaugeCalls, 1)
	require.Equal(t, 42.0, collector.GaugeCalls[0].Value)

	require.Len(t, collector.HistogramCalls, 1)
	require.Equal(t, 1.5, collector.HistogramCalls[0].Value)
}

func TestMockTracerAndSpan(t *testing.T) {
	tracer := NewMockTracer()
	ctx, span := tracer.StartSpan(context.Background(), "apply", "pipeline", "p-1")
	require.NotNil(t, ctx)

	mockSpan, ok := span.(*MockSpan)
	require.True(t, ok)

	mockSpan.SetAttribute("key", "value")
	mockSpan.SetAttribute("", "ignored")
	mockSpan.SetStatus(ports.SpanStatusOK, "done")
	mockSpan.End()

	require.Len(t, tracer.StartCalls, 1)
	call := tracer.StartCalls[0]
	require.Equal(t, "apply", call.Name)
	require.Equal(t, []interface{}{"pipeline", "p-1"}, call.Attributes)

	require.Equal(t, "value", mockSpan.Attributes["key"])
	require.Nil(t, mockSpan.Attributes[""], "blank key should be ignored")
	require.Equal(t, ports.SpanStatusOK, mockSpan.Status)
	require.Equal(t, "done", mockSpan.StatusMessage)
	require.Equal(t, 1, mockSpan.EndCount)
}

func TestCloneLabels(t *testing.T) {
	original := map[string]string{"a": "1"}
	cloned := cloneLabels(original)
	require.Equal(t, original, cloned)

	original["a"] = "2"

	require.Equal(t, "1", cloned["a"])

	require.Nil(t, cloneLabels(nil))
	require.Nil(t, cloneLabels(map[string]string{}))
}
