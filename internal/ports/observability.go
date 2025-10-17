package ports

import "context"

// MetricsCollector records quantitative observability signals. The interface is
// intentionally generic so adapters can back onto Prometheus, StatsD, or
// vendor-specific SDKs. Standard metric names (see observability.md) include:
//   - Counters:
//     streamy_pipeline_executions_total{status="success|failure|cancelled"}
//     streamy_step_executions_total{step_type="...", status="success|failure|skipped"}
//     streamy_step_changes_total{step_type="..."}
//     streamy_validation_checks_total{check_type="...", status="pass|fail"}
//   - Gauges:
//     streamy_pipeline_active_executions
//     streamy_step_parallel_executions
//   - Histograms:
//     streamy_pipeline_execution_duration_seconds
//     streamy_step_execution_duration_seconds{step_type="..."}
//     streamy_plugin_evaluation_duration_seconds{plugin_type="..."}
//     streamy_validation_check_duration_seconds{check_type="..."}
type MetricsCollector interface {
	IncCounter(ctx context.Context, name string, labels map[string]string)
	SetGauge(ctx context.Context, name string, value float64, labels map[string]string)
	ObserveHistogram(ctx context.Context, name string, value float64, labels map[string]string)
}

// Tracer manages distributed tracing spans. Span names follow the convention
// `<component>.<operation>` (e.g., `pipeline.apply`, `executor.execute`,
// `config.load`, `validation.run`). Adapters should propagate correlation IDs
// and integrate with the chosen tracing backend (e.g., OpenTelemetry).
type Tracer interface {
	StartSpan(ctx context.Context, name string, attributes ...interface{}) (context.Context, Span)
	Inject(ctx context.Context, carrier interface{}) error
	Extract(ctx context.Context, carrier interface{}) (context.Context, error)
}

// Span represents an active tracing span.
type Span interface {
	SetAttribute(key string, value interface{})
	SetStatus(status SpanStatus, message string)
	End()
}

// SpanStatus provides strongly typed span result semantics.
type SpanStatus string

const (
	SpanStatusOK    SpanStatus = "ok"
	SpanStatusError SpanStatus = "error"
)
