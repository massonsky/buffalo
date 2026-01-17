package metrics

import (
	"bytes"
	"strings"
	"testing"
)

func TestExporter_ExportText(t *testing.T) {
	collector := NewCollector()
	collector.SetLabel("service", "buffalo")

	counter := collector.Counter("requests_total")
	counter.Add(100)

	gauge := collector.Gauge("memory_usage")
	gauge.Set(512)

	exporter := NewExporter(collector)
	var buf bytes.Buffer

	err := exporter.Export(FormatText, &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "requests_total") {
		t.Error("expected output to contain 'requests_total'")
	}

	if !strings.Contains(output, "memory_usage") {
		t.Error("expected output to contain 'memory_usage'")
	}

	if !strings.Contains(output, "100") {
		t.Error("expected output to contain counter value '100'")
	}

	if !strings.Contains(output, "512") {
		t.Error("expected output to contain gauge value '512'")
	}
}

func TestExporter_ExportJSON(t *testing.T) {
	collector := NewCollector()
	collector.SetLabel("service", "buffalo")

	counter := collector.Counter("requests_total")
	counter.Add(50)

	exporter := NewExporter(collector)
	var buf bytes.Buffer

	err := exporter.Export(FormatJSON, &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "\"timestamp\"") {
		t.Error("expected JSON to contain 'timestamp'")
	}

	if !strings.Contains(output, "\"labels\"") {
		t.Error("expected JSON to contain 'labels'")
	}

	if !strings.Contains(output, "\"metrics\"") {
		t.Error("expected JSON to contain 'metrics'")
	}

	if !strings.Contains(output, "requests_total") {
		t.Error("expected JSON to contain 'requests_total'")
	}
}

func TestExporter_ExportPrometheus(t *testing.T) {
	collector := NewCollector()
	collector.SetLabel("env", "test")

	counter := collector.Counter("http_requests_total")
	counter.Add(200)

	gauge := collector.Gauge("cpu_usage")
	gauge.Set(75)

	exporter := NewExporter(collector)
	var buf bytes.Buffer

	err := exporter.Export(FormatPrometheus, &buf)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "# TYPE http_requests_total counter") {
		t.Error("expected Prometheus format to contain counter type declaration")
	}

	if !strings.Contains(output, "# TYPE cpu_usage gauge") {
		t.Error("expected Prometheus format to contain gauge type declaration")
	}

	if !strings.Contains(output, "http_requests_total") {
		t.Error("expected output to contain 'http_requests_total'")
	}

	if !strings.Contains(output, "200") {
		t.Error("expected output to contain counter value '200'")
	}
}

func TestExporter_ExportHistogram(t *testing.T) {
	collector := NewCollector()

	histogram := collector.Histogram("request_duration", []float64{0.1, 0.5, 1.0})
	histogram.Observe(0.2)
	histogram.Observe(0.8)
	histogram.Observe(1.5)

	exporter := NewExporter(collector)

	// Test text format
	var textBuf bytes.Buffer
	err := exporter.Export(FormatText, &textBuf)
	if err != nil {
		t.Fatalf("Export text failed: %v", err)
	}

	textOutput := textBuf.String()
	if !strings.Contains(textOutput, "request_duration") {
		t.Error("expected text output to contain 'request_duration'")
	}
	if !strings.Contains(textOutput, "count=3") {
		t.Error("expected text output to contain 'count=3'")
	}

	// Test Prometheus format
	var promBuf bytes.Buffer
	err = exporter.Export(FormatPrometheus, &promBuf)
	if err != nil {
		t.Fatalf("Export prometheus failed: %v", err)
	}

	promOutput := promBuf.String()
	if !strings.Contains(promOutput, "# TYPE request_duration histogram") {
		t.Error("expected Prometheus format to contain histogram type declaration")
	}
	if !strings.Contains(promOutput, "request_duration_count") {
		t.Error("expected Prometheus format to contain histogram count")
	}
	if !strings.Contains(promOutput, "request_duration_sum") {
		t.Error("expected Prometheus format to contain histogram sum")
	}
}

func TestExporter_UnsupportedFormat(t *testing.T) {
	collector := NewCollector()
	exporter := NewExporter(collector)

	var buf bytes.Buffer
	err := exporter.Export(ExportFormat("unsupported"), &buf)
	if err == nil {
		t.Error("expected error for unsupported format")
	}
}

func TestFormatLabels(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		expected string
	}{
		{
			name:     "empty labels",
			labels:   map[string]string{},
			expected: "",
		},
		{
			name:     "single label",
			labels:   map[string]string{"env": "test"},
			expected: "{env=\"test\"}",
		},
		{
			name:     "multiple labels",
			labels:   map[string]string{"env": "test", "service": "buffalo"},
			expected: "{env=\"test\",service=\"buffalo\"}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatLabels(tt.labels)
			// For multiple labels, just check that both are present
			if len(tt.labels) > 1 {
				for key, value := range tt.labels {
					expected := key + "=\"" + value + "\""
					if !strings.Contains(result, expected) {
						t.Errorf("expected result to contain '%s', got '%s'", expected, result)
					}
				}
			} else {
				if result != tt.expected {
					t.Errorf("expected '%s', got '%s'", tt.expected, result)
				}
			}
		})
	}
}

func BenchmarkExporter_Export(b *testing.B) {
	collector := NewCollector()
	for i := 0; i < 10; i++ {
		collector.Counter("counter_" + string(rune('0'+i))).Add(int64(i * 10))
	}

	exporter := NewExporter(collector)
	var buf bytes.Buffer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = exporter.Export(FormatText, &buf)
	}
}
