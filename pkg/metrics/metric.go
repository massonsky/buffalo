package metrics

import (
	"sync/atomic"
)

// Metric is the interface for all metrics.
type Metric interface {
	// Name returns the metric name.
	Name() string

	// Type returns the metric type.
	Type() string

	// Value returns the current value of the metric.
	Value() interface{}

	// Reset resets the metric to its initial state.
	Reset()
}

// Counter is a metric that can only increase.
type Counter struct {
	name  string
	value int64
}

// NewCounter creates a new counter metric.
func NewCounter(name string) *Counter {
	return &Counter{
		name: name,
	}
}

// Name returns the counter name.
func (c *Counter) Name() string {
	return c.name
}

// Type returns "counter".
func (c *Counter) Type() string {
	return "counter"
}

// Value returns the current counter value.
func (c *Counter) Value() interface{} {
	return atomic.LoadInt64(&c.value)
}

// Inc increments the counter by 1.
func (c *Counter) Inc() {
	atomic.AddInt64(&c.value, 1)
}

// Add adds the given value to the counter.
func (c *Counter) Add(delta int64) {
	if delta < 0 {
		return // Counters can only increase
	}
	atomic.AddInt64(&c.value, delta)
}

// Get returns the current value.
func (c *Counter) Get() int64 {
	return atomic.LoadInt64(&c.value)
}

// Reset resets the counter to zero.
func (c *Counter) Reset() {
	atomic.StoreInt64(&c.value, 0)
}

// Gauge is a metric that can increase or decrease.
type Gauge struct {
	name  string
	value int64
}

// NewGauge creates a new gauge metric.
func NewGauge(name string) *Gauge {
	return &Gauge{
		name: name,
	}
}

// Name returns the gauge name.
func (g *Gauge) Name() string {
	return g.name
}

// Type returns "gauge".
func (g *Gauge) Type() string {
	return "gauge"
}

// Value returns the current gauge value.
func (g *Gauge) Value() interface{} {
	return atomic.LoadInt64(&g.value)
}

// Set sets the gauge to the given value.
func (g *Gauge) Set(value int64) {
	atomic.StoreInt64(&g.value, value)
}

// Inc increments the gauge by 1.
func (g *Gauge) Inc() {
	atomic.AddInt64(&g.value, 1)
}

// Dec decrements the gauge by 1.
func (g *Gauge) Dec() {
	atomic.AddInt64(&g.value, -1)
}

// Add adds the given value to the gauge.
func (g *Gauge) Add(delta int64) {
	atomic.AddInt64(&g.value, delta)
}

// Sub subtracts the given value from the gauge.
func (g *Gauge) Sub(delta int64) {
	atomic.AddInt64(&g.value, -delta)
}

// Get returns the current value.
func (g *Gauge) Get() int64 {
	return atomic.LoadInt64(&g.value)
}

// Reset resets the gauge to zero.
func (g *Gauge) Reset() {
	atomic.StoreInt64(&g.value, 0)
}

// Histogram tracks the distribution of values.
type Histogram struct {
	name    string
	buckets []float64
	counts  []int64
	sum     int64
	count   int64
}

// NewHistogram creates a new histogram metric.
func NewHistogram(name string, buckets []float64) *Histogram {
	if buckets == nil {
		buckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
	}

	return &Histogram{
		name:    name,
		buckets: buckets,
		counts:  make([]int64, len(buckets)+1),
	}
}

// Name returns the histogram name.
func (h *Histogram) Name() string {
	return h.name
}

// Type returns "histogram".
func (h *Histogram) Type() string {
	return "histogram"
}

// Value returns the histogram data.
func (h *Histogram) Value() interface{} {
	counts := make([]int64, len(h.counts))
	for i := range h.counts {
		counts[i] = atomic.LoadInt64(&h.counts[i])
	}

	return map[string]interface{}{
		"buckets": h.buckets,
		"counts":  counts,
		"sum":     atomic.LoadInt64(&h.sum),
		"count":   atomic.LoadInt64(&h.count),
	}
}

// Observe records a value in the histogram.
func (h *Histogram) Observe(value float64) {
	// Find the bucket
	bucketIndex := len(h.buckets)
	for i, bucket := range h.buckets {
		if value <= bucket {
			bucketIndex = i
			break
		}
	}

	// Increment the bucket count
	atomic.AddInt64(&h.counts[bucketIndex], 1)

	// Update sum and count
	atomic.AddInt64(&h.sum, int64(value*1000)) // Store as milliseconds
	atomic.AddInt64(&h.count, 1)
}

// Count returns the total number of observations.
func (h *Histogram) Count() int64 {
	return atomic.LoadInt64(&h.count)
}

// Sum returns the sum of all observations.
func (h *Histogram) Sum() float64 {
	return float64(atomic.LoadInt64(&h.sum)) / 1000.0
}

// Average returns the average of all observations.
func (h *Histogram) Average() float64 {
	count := h.Count()
	if count == 0 {
		return 0
	}
	return h.Sum() / float64(count)
}

// Reset resets the histogram.
func (h *Histogram) Reset() {
	for i := range h.counts {
		atomic.StoreInt64(&h.counts[i], 0)
	}
	atomic.StoreInt64(&h.sum, 0)
	atomic.StoreInt64(&h.count, 0)
}
