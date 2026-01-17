package metrics

import (
	"sync"
	"time"

	"github.com/massonsky/buffalo/pkg/errors"
)

// Collector collects and manages metrics.
type Collector struct {
	mu      sync.RWMutex
	metrics map[string]Metric
	labels  map[string]string
}

// NewCollector creates a new metrics collector.
func NewCollector() *Collector {
	return &Collector{
		metrics: make(map[string]Metric),
		labels:  make(map[string]string),
	}
}

// Register registers a new metric.
func (c *Collector) Register(name string, metric Metric) error {
	if name == "" {
		return errors.New(errors.ErrInvalidArgument, "metric name cannot be empty")
	}
	if metric == nil {
		return errors.New(errors.ErrInvalidArgument, "metric cannot be nil")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.metrics[name]; exists {
		return errors.New(errors.ErrAlreadyExists, "metric already registered: %s", name)
	}

	c.metrics[name] = metric
	return nil
}

// Get retrieves a metric by name.
func (c *Collector) Get(name string) (Metric, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	metric, exists := c.metrics[name]
	if !exists {
		return nil, errors.New(errors.ErrNotFound, "metric not found: %s", name)
	}

	return metric, nil
}

// GetOrCreate retrieves a metric or creates it if it doesn't exist.
func (c *Collector) GetOrCreate(name string, factory func() Metric) Metric {
	c.mu.Lock()
	defer c.mu.Unlock()

	if metric, exists := c.metrics[name]; exists {
		return metric
	}

	metric := factory()
	c.metrics[name] = metric
	return metric
}

// Counter returns a counter metric, creating it if necessary.
func (c *Collector) Counter(name string) *Counter {
	metric := c.GetOrCreate(name, func() Metric {
		return NewCounter(name)
	})
	return metric.(*Counter)
}

// Gauge returns a gauge metric, creating it if necessary.
func (c *Collector) Gauge(name string) *Gauge {
	metric := c.GetOrCreate(name, func() Metric {
		return NewGauge(name)
	})
	return metric.(*Gauge)
}

// Histogram returns a histogram metric, creating it if necessary.
func (c *Collector) Histogram(name string, buckets []float64) *Histogram {
	metric := c.GetOrCreate(name, func() Metric {
		return NewHistogram(name, buckets)
	})
	return metric.(*Histogram)
}

// All returns all registered metrics.
func (c *Collector) All() map[string]Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]Metric, len(c.metrics))
	for name, metric := range c.metrics {
		result[name] = metric
	}

	return result
}

// Reset resets all metrics.
func (c *Collector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, metric := range c.metrics {
		metric.Reset()
	}
}

// SetLabel sets a global label for all metrics.
func (c *Collector) SetLabel(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.labels[key] = value
}

// GetLabels returns all global labels.
func (c *Collector) GetLabels() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]string, len(c.labels))
	for k, v := range c.labels {
		result[k] = v
	}

	return result
}

// Snapshot creates a snapshot of all metrics.
type Snapshot struct {
	Timestamp time.Time
	Labels    map[string]string
	Metrics   map[string]MetricSnapshot
}

// MetricSnapshot represents a snapshot of a metric.
type MetricSnapshot struct {
	Name  string
	Type  string
	Value interface{}
}

// Snapshot creates a snapshot of all metrics.
func (c *Collector) Snapshot() *Snapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()

	snapshot := &Snapshot{
		Timestamp: time.Now(),
		Labels:    c.GetLabels(),
		Metrics:   make(map[string]MetricSnapshot, len(c.metrics)),
	}

	for name, metric := range c.metrics {
		snapshot.Metrics[name] = MetricSnapshot{
			Name:  name,
			Type:  metric.Type(),
			Value: metric.Value(),
		}
	}

	return snapshot
}
