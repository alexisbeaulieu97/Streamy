package metrics

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/alexisbeaulieu97/streamy/internal/ports"
)

// Collector is an in-memory MetricsCollector implementation. It tracks
// counters, gauges, and basic histogram statistics so callers can expose the
// data to Prometheus exporters or other sinks in the future. All operations are
// safe for concurrent use.
type Collector struct {
	mu         sync.RWMutex
	counters   map[string]map[string]*counterSeries
	gauges     map[string]map[string]*gaugeSeries
	histograms map[string]map[string]*histogramSeries
}

type counterSeries struct {
	value  float64
	labels map[string]string
}

type gaugeSeries struct {
	value  float64
	labels map[string]string
}

type histogramSeries struct {
	count  int64
	sum    float64
	min    float64
	max    float64
	last   float64
	labels map[string]string
}

// Snapshot represents a point-in-time view of the collector's state.
type Snapshot struct {
	Counters   map[string]map[string]float64
	Gauges     map[string]map[string]float64
	Histograms map[string]map[string]Histogram
}

// Histogram aggregates simple statistics for a metric series.
type Histogram struct {
	Count int64
	Sum   float64
	Min   float64
	Max   float64
	Last  float64
}

// NewCollector constructs an empty MetricsCollector implementation.
func NewCollector() *Collector {
	return &Collector{
		counters:   make(map[string]map[string]*counterSeries),
		gauges:     make(map[string]map[string]*gaugeSeries),
		histograms: make(map[string]map[string]*histogramSeries),
	}
}

// IncCounter increments a named counter by one, creating the series if needed.
func (c *Collector) IncCounter(_ context.Context, name string, labels map[string]string) {
	if name == "" {
		return
	}
	key := labelsKey(labels)

	c.mu.Lock()
	defer c.mu.Unlock()

	series, ok := c.counters[name]
	if !ok {
		series = make(map[string]*counterSeries)
		c.counters[name] = series
	}

	entry, ok := series[key]
	if !ok {
		entry = &counterSeries{
			labels: copyLabels(labels),
		}
		series[key] = entry
	}
	entry.value++
}

// SetGauge assigns the provided value to a gauge.
func (c *Collector) SetGauge(_ context.Context, name string, value float64, labels map[string]string) {
	if name == "" {
		return
	}
	key := labelsKey(labels)

	c.mu.Lock()
	defer c.mu.Unlock()

	series, ok := c.gauges[name]
	if !ok {
		series = make(map[string]*gaugeSeries)
		c.gauges[name] = series
	}

	entry, ok := series[key]
	if !ok {
		entry = &gaugeSeries{
			labels: copyLabels(labels),
		}
		series[key] = entry
	}
	entry.value = value
}

// ObserveHistogram records a single observation for the named histogram.
func (c *Collector) ObserveHistogram(_ context.Context, name string, value float64, labels map[string]string) {
	if name == "" {
		return
	}
	key := labelsKey(labels)

	c.mu.Lock()
	defer c.mu.Unlock()

	series, ok := c.histograms[name]
	if !ok {
		series = make(map[string]*histogramSeries)
		c.histograms[name] = series
	}

	entry, ok := series[key]
	if !ok {
		entry = &histogramSeries{
			min:    value,
			max:    value,
			labels: copyLabels(labels),
		}
		series[key] = entry
	}

	entry.count++
	entry.sum += value
	entry.last = value
	if value < entry.min {
		entry.min = value
	}
	if value > entry.max {
		entry.max = value
	}
}

// Snapshot returns a deep copy of the collector's metrics for inspection.
func (c *Collector) Snapshot() Snapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()

	counterCopy := make(map[string]map[string]float64, len(c.counters))
	for name, series := range c.counters {
		copySeries := make(map[string]float64, len(series))
		for key, entry := range series {
			copySeries[key] = entry.value
		}
		counterCopy[name] = copySeries
	}

	gaugeCopy := make(map[string]map[string]float64, len(c.gauges))
	for name, series := range c.gauges {
		copySeries := make(map[string]float64, len(series))
		for key, entry := range series {
			copySeries[key] = entry.value
		}
		gaugeCopy[name] = copySeries
	}

	histCopy := make(map[string]map[string]Histogram, len(c.histograms))
	for name, series := range c.histograms {
		copySeries := make(map[string]Histogram, len(series))
		for key, entry := range series {
			copySeries[key] = Histogram{
				Count: entry.count,
				Sum:   entry.sum,
				Min:   entry.min,
				Max:   entry.max,
				Last:  entry.last,
			}
		}
		histCopy[name] = copySeries
	}

	return Snapshot{
		Counters:   counterCopy,
		Gauges:     gaugeCopy,
		Histograms: histCopy,
	}
}

func labelsKey(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var b strings.Builder
	for idx, key := range keys {
		if idx > 0 {
			b.WriteByte(',')
		}
		b.WriteString(key)
		b.WriteByte('=')
		b.WriteString(labels[key])
	}
	return b.String()
}

func copyLabels(labels map[string]string) map[string]string {
	if len(labels) == 0 {
		return nil
	}
	result := make(map[string]string, len(labels))
	for k, v := range labels {
		result[k] = v
	}
	return result
}

var _ ports.MetricsCollector = (*Collector)(nil)
