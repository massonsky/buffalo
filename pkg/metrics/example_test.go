package metrics_test

import (
	"fmt"
	"os"
	"time"

	"github.com/massonsky/buffalo/pkg/metrics"
)

func ExampleCollector() {
	// Create a new metrics collector
	collector := metrics.NewCollector()
	collector.SetLabel("service", "buffalo")
	collector.SetLabel("version", "0.1.0")

	// Create and use a counter
	counter := collector.Counter("requests_total")
	counter.Add(100)

	// Create and use a gauge
	gauge := collector.Gauge("active_connections")
	gauge.Set(42)

	fmt.Printf("Requests: %d\n", counter.Get())
	fmt.Printf("Connections: %d\n", gauge.Get())
	// Output:
	// Requests: 100
	// Connections: 42
}

func ExampleCounter() {
	counter := metrics.NewCounter("http_requests")

	// Increment by 1
	counter.Inc()

	// Add a specific value
	counter.Add(5)

	fmt.Printf("Total requests: %d\n", counter.Get())
	// Output: Total requests: 6
}

func ExampleGauge() {
	gauge := metrics.NewGauge("memory_usage_mb")

	// Set to a specific value
	gauge.Set(512)

	// Increment
	gauge.Inc()

	// Decrement
	gauge.Dec()

	// Add/subtract values
	gauge.Add(100)
	gauge.Sub(50)

	fmt.Printf("Memory usage: %d MB\n", gauge.Get())
	// Output: Memory usage: 562 MB
}

func ExampleHistogram() {
	// Define buckets for request duration in seconds
	buckets := []float64{0.1, 0.5, 1.0, 2.0, 5.0}
	histogram := metrics.NewHistogram("request_duration_seconds", buckets)

	// Record observations
	histogram.Observe(0.2)
	histogram.Observe(0.8)
	histogram.Observe(1.5)
	histogram.Observe(3.0)

	fmt.Printf("Count: %d\n", histogram.Count())
	fmt.Printf("Average: %.2f seconds\n", histogram.Average())
	// Output:
	// Count: 4
	// Average: 1.38 seconds
}

func ExampleExporter_text() {
	collector := metrics.NewCollector()
	collector.SetLabel("service", "buffalo")

	counter := collector.Counter("operations_total")
	counter.Add(150)

	exporter := metrics.NewExporter(collector)
	exporter.Export(metrics.FormatText, os.Stdout)
}

func ExampleExporter_prometheus() {
	collector := metrics.NewCollector()
	collector.SetLabel("env", "production")

	counter := collector.Counter("http_requests_total")
	counter.Add(1000)

	gauge := collector.Gauge("cpu_usage_percent")
	gauge.Set(75)

	exporter := metrics.NewExporter(collector)
	exporter.Export(metrics.FormatPrometheus, os.Stdout)
}

func Example_realWorldUsage() {
	// Initialize metrics collector
	collector := metrics.NewCollector()
	collector.SetLabel("service", "api-gateway")
	collector.SetLabel("version", "1.0.0")

	// Create metrics
	requestCounter := collector.Counter("http_requests_total")
	errorCounter := collector.Counter("http_errors_total")
	activeConnections := collector.Gauge("active_connections")
	requestDuration := collector.Histogram("request_duration_seconds",
		[]float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0})

	// Simulate some activity
	for i := 0; i < 10; i++ {
		// Track request
		requestCounter.Inc()
		activeConnections.Inc()

		// Simulate request processing
		start := time.Now()
		time.Sleep(time.Millisecond * 10)
		duration := time.Since(start).Seconds()
		requestDuration.Observe(duration)

		// Simulate occasional errors
		if i%3 == 0 {
			errorCounter.Inc()
		}

		activeConnections.Dec()
	}

	// Export metrics
	fmt.Println("=== Metrics Summary ===")
	fmt.Printf("Total Requests: %d\n", requestCounter.Get())
	fmt.Printf("Total Errors: %d\n", errorCounter.Get())
	fmt.Printf("Active Connections: %d\n", activeConnections.Get())
	fmt.Printf("Average Duration: %.3f seconds\n", requestDuration.Average())
}
