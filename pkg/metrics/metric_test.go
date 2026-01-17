package metrics

import (
	"testing"
)

func TestCounter(t *testing.T) {
	counter := NewCounter("test_counter")

	if counter.Name() != "test_counter" {
		t.Errorf("expected name 'test_counter', got '%s'", counter.Name())
	}

	if counter.Type() != "counter" {
		t.Errorf("expected type 'counter', got '%s'", counter.Type())
	}

	if counter.Get() != 0 {
		t.Errorf("expected initial value 0, got %d", counter.Get())
	}

	counter.Inc()
	if counter.Get() != 1 {
		t.Errorf("expected value 1 after Inc(), got %d", counter.Get())
	}

	counter.Add(5)
	if counter.Get() != 6 {
		t.Errorf("expected value 6 after Add(5), got %d", counter.Get())
	}

	// Test that negative values are ignored
	counter.Add(-3)
	if counter.Get() != 6 {
		t.Errorf("expected value 6 after Add(-3), got %d", counter.Get())
	}

	counter.Reset()
	if counter.Get() != 0 {
		t.Errorf("expected value 0 after Reset(), got %d", counter.Get())
	}
}

func TestGauge(t *testing.T) {
	gauge := NewGauge("test_gauge")

	if gauge.Name() != "test_gauge" {
		t.Errorf("expected name 'test_gauge', got '%s'", gauge.Name())
	}

	if gauge.Type() != "gauge" {
		t.Errorf("expected type 'gauge', got '%s'", gauge.Type())
	}

	gauge.Set(10)
	if gauge.Get() != 10 {
		t.Errorf("expected value 10 after Set(10), got %d", gauge.Get())
	}

	gauge.Inc()
	if gauge.Get() != 11 {
		t.Errorf("expected value 11 after Inc(), got %d", gauge.Get())
	}

	gauge.Dec()
	if gauge.Get() != 10 {
		t.Errorf("expected value 10 after Dec(), got %d", gauge.Get())
	}

	gauge.Add(5)
	if gauge.Get() != 15 {
		t.Errorf("expected value 15 after Add(5), got %d", gauge.Get())
	}

	gauge.Sub(3)
	if gauge.Get() != 12 {
		t.Errorf("expected value 12 after Sub(3), got %d", gauge.Get())
	}

	gauge.Reset()
	if gauge.Get() != 0 {
		t.Errorf("expected value 0 after Reset(), got %d", gauge.Get())
	}
}

func TestHistogram(t *testing.T) {
	buckets := []float64{1.0, 2.0, 5.0, 10.0}
	histogram := NewHistogram("test_histogram", buckets)

	if histogram.Name() != "test_histogram" {
		t.Errorf("expected name 'test_histogram', got '%s'", histogram.Name())
	}

	if histogram.Type() != "histogram" {
		t.Errorf("expected type 'histogram', got '%s'", histogram.Type())
	}

	histogram.Observe(0.5)
	histogram.Observe(1.5)
	histogram.Observe(3.0)
	histogram.Observe(7.0)
	histogram.Observe(15.0)

	if histogram.Count() != 5 {
		t.Errorf("expected count 5, got %d", histogram.Count())
	}

	expectedSum := 0.5 + 1.5 + 3.0 + 7.0 + 15.0
	if histogram.Sum() != expectedSum {
		t.Errorf("expected sum %.2f, got %.2f", expectedSum, histogram.Sum())
	}

	expectedAvg := expectedSum / 5.0
	avg := histogram.Average()
	if avg < expectedAvg-0.01 || avg > expectedAvg+0.01 {
		t.Errorf("expected average %.2f, got %.2f", expectedAvg, avg)
	}

	histogram.Reset()
	if histogram.Count() != 0 {
		t.Errorf("expected count 0 after Reset(), got %d", histogram.Count())
	}
}

func TestHistogram_DefaultBuckets(t *testing.T) {
	histogram := NewHistogram("test", nil)

	if len(histogram.buckets) == 0 {
		t.Error("expected default buckets to be set")
	}
}

func BenchmarkCounter_Inc(b *testing.B) {
	counter := NewCounter("bench_counter")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		counter.Inc()
	}
}

func BenchmarkGauge_Set(b *testing.B) {
	gauge := NewGauge("bench_gauge")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		gauge.Set(int64(i))
	}
}

func BenchmarkHistogram_Observe(b *testing.B) {
	histogram := NewHistogram("bench_histogram", nil)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		histogram.Observe(float64(i % 100))
	}
}
