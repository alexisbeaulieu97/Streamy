package metrics

import (
	"context"

	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

// NoOpCollector discards all metrics, useful for tests and lightweight CLIs.
type NoOpCollector struct{}

// NewNoOpCollector constructs a metrics collector that records nothing.
func NewNoOpCollector() *NoOpCollector {
	return &NoOpCollector{}
}

// IncCounter implements ports.MetricsCollector.
func (*NoOpCollector) IncCounter(context.Context, string, map[string]string) {}

// SetGauge implements ports.MetricsCollector.
func (*NoOpCollector) SetGauge(context.Context, string, float64, map[string]string) {}

// ObserveHistogram implements ports.MetricsCollector.
func (*NoOpCollector) ObserveHistogram(context.Context, string, float64, map[string]string) {}

var _ ports.MetricsCollector = (*NoOpCollector)(nil)
