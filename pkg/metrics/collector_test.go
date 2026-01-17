package metrics

import (
	"testing"
)

func TestCollector_Register(t *testing.T) {
	collector := NewCollector()

	counter := NewCounter("test_counter")
	err := collector.Register("test_counter", counter)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Test duplicate registration
	err = collector.Register("test_counter", counter)
	if err == nil {
		t.Error("expected error when registering duplicate metric")
	}

	// Test empty name
	err = collector.Register("", counter)
	if err == nil {
		t.Error("expected error when registering with empty name")
	}

	// Test nil metric
	err = collector.Register("nil_metric", nil)
	if err == nil {
		t.Error("expected error when registering nil metric")
	}
}

func TestCollector_Get(t *testing.T) {
	collector := NewCollector()
	counter := NewCounter("test_counter")
	collector.Register("test_counter", counter)

	metric, err := collector.Get("test_counter")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if metric.Name() != "test_counter" {
		t.Errorf("expected metric name 'test_counter', got '%s'", metric.Name())
	}

	// Test non-existent metric
	_, err = collector.Get("non_existent")
	if err == nil {
		t.Error("expected error when getting non-existent metric")
	}
}

func TestCollector_GetOrCreate(t *testing.T) {
	collector := NewCollector()

	// Create new metric
	metric1 := collector.GetOrCreate("test_metric", func() Metric {
		return NewCounter("test_metric")
	})

	if metric1.Name() != "test_metric" {
		t.Errorf("expected metric name 'test_metric', got '%s'", metric1.Name())
	}

	// Get existing metric
	metric2 := collector.GetOrCreate("test_metric", func() Metric {
		return NewCounter("different_name")
	})

	if metric1 != metric2 {
		t.Error("expected GetOrCreate to return existing metric")
	}
}

func TestCollector_Counter(t *testing.T) {
	collector := NewCollector()

	counter := collector.Counter("test_counter")
	counter.Inc()

	if counter.Get() != 1 {
		t.Errorf("expected counter value 1, got %d", counter.Get())
	}

	// Get same counter again
	counter2 := collector.Counter("test_counter")
	if counter2.Get() != 1 {
		t.Error("expected to get the same counter instance")
	}
}

func TestCollector_Gauge(t *testing.T) {
	collector := NewCollector()

	gauge := collector.Gauge("test_gauge")
	gauge.Set(42)

	if gauge.Get() != 42 {
		t.Errorf("expected gauge value 42, got %d", gauge.Get())
	}

	// Get same gauge again
	gauge2 := collector.Gauge("test_gauge")
	if gauge2.Get() != 42 {
		t.Error("expected to get the same gauge instance")
	}
}

func TestCollector_Histogram(t *testing.T) {
	collector := NewCollector()
	buckets := []float64{1.0, 5.0, 10.0}

	histogram := collector.Histogram("test_histogram", buckets)
	histogram.Observe(3.0)

	if histogram.Count() != 1 {
		t.Errorf("expected histogram count 1, got %d", histogram.Count())
	}
}

func TestCollector_All(t *testing.T) {
	collector := NewCollector()

	collector.Counter("counter1")
	collector.Counter("counter2")
	collector.Gauge("gauge1")

	all := collector.All()
	if len(all) != 3 {
		t.Errorf("expected 3 metrics, got %d", len(all))
	}

	if _, exists := all["counter1"]; !exists {
		t.Error("expected counter1 to exist")
	}
	if _, exists := all["counter2"]; !exists {
		t.Error("expected counter2 to exist")
	}
	if _, exists := all["gauge1"]; !exists {
		t.Error("expected gauge1 to exist")
	}
}

func TestCollector_Reset(t *testing.T) {
	collector := NewCollector()

	counter := collector.Counter("test_counter")
	counter.Add(10)

	gauge := collector.Gauge("test_gauge")
	gauge.Set(20)

	collector.Reset()

	if counter.Get() != 0 {
		t.Errorf("expected counter to be reset to 0, got %d", counter.Get())
	}

	if gauge.Get() != 0 {
		t.Errorf("expected gauge to be reset to 0, got %d", gauge.Get())
	}
}

func TestCollector_Labels(t *testing.T) {
	collector := NewCollector()

	collector.SetLabel("service", "buffalo")
	collector.SetLabel("version", "0.1.0")

	labels := collector.GetLabels()
	if len(labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(labels))
	}

	if labels["service"] != "buffalo" {
		t.Errorf("expected service label 'buffalo', got '%s'", labels["service"])
	}

	if labels["version"] != "0.1.0" {
		t.Errorf("expected version label '0.1.0', got '%s'", labels["version"])
	}
}

func TestCollector_Snapshot(t *testing.T) {
	collector := NewCollector()
	collector.SetLabel("env", "test")

	counter := collector.Counter("test_counter")
	counter.Add(5)

	gauge := collector.Gauge("test_gauge")
	gauge.Set(10)

	snapshot := collector.Snapshot()

	if len(snapshot.Labels) != 1 {
		t.Errorf("expected 1 label in snapshot, got %d", len(snapshot.Labels))
	}

	if len(snapshot.Metrics) != 2 {
		t.Errorf("expected 2 metrics in snapshot, got %d", len(snapshot.Metrics))
	}

	counterSnapshot := snapshot.Metrics["test_counter"]
	if counterSnapshot.Value != int64(5) {
		t.Errorf("expected counter value 5, got %v", counterSnapshot.Value)
	}

	gaugeSnapshot := snapshot.Metrics["test_gauge"]
	if gaugeSnapshot.Value != int64(10) {
		t.Errorf("expected gauge value 10, got %v", gaugeSnapshot.Value)
	}
}

func BenchmarkCollector_Counter(b *testing.B) {
	collector := NewCollector()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		counter := collector.Counter("bench_counter")
		counter.Inc()
	}
}

func BenchmarkCollector_Snapshot(b *testing.B) {
	collector := NewCollector()
	for i := 0; i < 10; i++ {
		collector.Counter("counter_" + string(rune('0'+i))).Add(int64(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = collector.Snapshot()
	}
}
