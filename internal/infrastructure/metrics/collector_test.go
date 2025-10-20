package metrics

import (
	"context"
	"testing"
)

func TestCollector_IncCounterAndGauge(t *testing.T) {
	t.Parallel()

	collector := NewCollector()
	ctx := context.Background()

	collector.IncCounter(ctx, "streamy_pipeline_executions_total", map[string]string{"status": "success"})
	collector.IncCounter(ctx, "streamy_pipeline_executions_total", map[string]string{"status": "success"})
	collector.IncCounter(ctx, "streamy_pipeline_executions_total", map[string]string{"status": "failure"})
	collector.SetGauge(ctx, "streamy_pipeline_active_executions", 3, map[string]string{"cluster": "local"})

	snap := collector.Snapshot()

	if got := snap.Counters["streamy_pipeline_executions_total"]["status=failure"]; got != 1 {
		t.Fatalf("expected failure counter at 1, got %v", got)
	}

	if got := snap.Counters["streamy_pipeline_executions_total"]["status=success"]; got != 2 {
		t.Fatalf("expected success counter at 2, got %v", got)
	}

	if got := snap.Gauges["streamy_pipeline_active_executions"]["cluster=local"]; got != 3 {
		t.Fatalf("expected gauge to be 3, got %v", got)
	}
}

func TestCollector_ObserveHistogram(t *testing.T) {
	t.Parallel()

	collector := NewCollector()
	ctx := context.Background()
	labels := map[string]string{"step_type": "command"}

	collector.ObserveHistogram(ctx, "streamy_step_execution_duration_seconds", 1.5, labels)
	collector.ObserveHistogram(ctx, "streamy_step_execution_duration_seconds", 0.5, labels)
	collector.ObserveHistogram(ctx, "streamy_step_execution_duration_seconds", 2.0, labels)

	snap := collector.Snapshot()
	hist := snap.Histograms["streamy_step_execution_duration_seconds"]["step_type=command"]

	if hist.Count != 3 {
		t.Fatalf("expected count 3, got %d", hist.Count)
	}

	if hist.Min != 0.5 {
		t.Fatalf("expected min 0.5, got %v", hist.Min)
	}

	if hist.Max != 2.0 {
		t.Fatalf("expected max 2.0, got %v", hist.Max)
	}

	if hist.Sum != 4.0 {
		t.Fatalf("expected sum 4.0, got %v", hist.Sum)
	}

	if hist.Last != 2.0 {
		t.Fatalf("expected last observation 2.0, got %v", hist.Last)
	}
}

func TestCollector_LabelCanonicalisation(t *testing.T) {
	t.Parallel()

	collector := NewCollector()
	ctx := context.Background()

	labelsA := map[string]string{"status": "success", "pipeline": "demo"}
	labelsB := map[string]string{"pipeline": "demo", "status": "success"}

	collector.IncCounter(ctx, "streamy_pipeline_executions_total", labelsA)
	collector.IncCounter(ctx, "streamy_pipeline_executions_total", labelsB)

	snap := collector.Snapshot()
	counterSeries := snap.Counters["streamy_pipeline_executions_total"]

	if len(counterSeries) != 1 {
		t.Fatalf("expected single series for identical label sets, got %d", len(counterSeries))
	}

	for key, value := range counterSeries {
		if key != "pipeline=demo,status=success" {
			t.Fatalf("unexpected label key %q", key)
		}

		if value != 2 {
			t.Fatalf("expected combined counter value 2, got %v", value)
		}
	}
}
