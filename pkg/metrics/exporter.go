package metrics

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/massonsky/buffalo/pkg/logger"
)

// Exporter exports metrics in various formats.
type Exporter struct {
	collector *Collector
	logger    *logger.Logger
}

// NewExporter creates a new metrics exporter.
func NewExporter(collector *Collector) *Exporter {
	return &Exporter{
		collector: collector,
	}
}

// WithLogger sets the logger for the exporter.
func (e *Exporter) WithLogger(log *logger.Logger) *Exporter {
	e.logger = log
	return e
}

// ExportFormat represents the format for exporting metrics.
type ExportFormat string

const (
	FormatText       ExportFormat = "text"
	FormatJSON       ExportFormat = "json"
	FormatPrometheus ExportFormat = "prometheus"
)

// Export exports metrics in the specified format.
func (e *Exporter) Export(format ExportFormat, writer io.Writer) error {
	snapshot := e.collector.Snapshot()

	switch format {
	case FormatText:
		return e.exportText(snapshot, writer)
	case FormatJSON:
		return e.exportJSON(snapshot, writer)
	case FormatPrometheus:
		return e.exportPrometheus(snapshot, writer)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// exportText exports metrics in plain text format.
func (e *Exporter) exportText(snapshot *Snapshot, writer io.Writer) error {
	lines := make([]string, 0, len(snapshot.Metrics)+8)

	// Add timestamp
	lines = append(lines, fmt.Sprintf("# Metrics Snapshot - %s", snapshot.Timestamp.Format(time.RFC3339)))
	lines = append(lines, "")

	// Add labels
	if len(snapshot.Labels) > 0 {
		lines = append(lines, "# Labels:")
		for key, value := range snapshot.Labels {
			lines = append(lines, fmt.Sprintf("#   %s = %s", key, value))
		}
		lines = append(lines, "")
	}

	// Add metrics
	lines = append(lines, "# Metrics:")

	// Sort metric names for consistent output
	names := make([]string, 0, len(snapshot.Metrics))
	for name := range snapshot.Metrics {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		metric := snapshot.Metrics[name]
		lines = append(lines, formatMetricText(metric))
	}

	_, err := writer.Write([]byte(strings.Join(lines, "\n") + "\n"))
	return err
}

// formatMetricText formats a single metric as text.
func formatMetricText(metric MetricSnapshot) string {
	switch metric.Type {
	case "counter":
		return fmt.Sprintf("%s{type=\"counter\"} %v", metric.Name, metric.Value)
	case "gauge":
		return fmt.Sprintf("%s{type=\"gauge\"} %v", metric.Name, metric.Value)
	case "histogram":
		data := metric.Value.(map[string]interface{})
		return fmt.Sprintf("%s{type=\"histogram\"} count=%v sum=%v avg=%.2f",
			metric.Name,
			data["count"],
			data["sum"],
			float64(data["sum"].(int64))/float64(maxInt64(data["count"].(int64), 1)))
	default:
		return fmt.Sprintf("%s{type=\"%s\"} %v", metric.Name, metric.Type, metric.Value)
	}
}

// exportJSON exports metrics in JSON format.
func (e *Exporter) exportJSON(snapshot *Snapshot, writer io.Writer) error {
	var lines []string
	lines = append(lines, "{")
	lines = append(lines, fmt.Sprintf("  \"timestamp\": \"%s\",", snapshot.Timestamp.Format(time.RFC3339)))

	// Labels
	if len(snapshot.Labels) > 0 {
		lines = append(lines, "  \"labels\": {")
		labelLines := make([]string, 0, len(snapshot.Labels))
		for key, value := range snapshot.Labels {
			labelLines = append(labelLines, fmt.Sprintf("    \"%s\": \"%s\"", key, value))
		}
		lines = append(lines, strings.Join(labelLines, ",\n"))
		lines = append(lines, "  },")
	}

	// Metrics
	lines = append(lines, "  \"metrics\": {")
	names := make([]string, 0, len(snapshot.Metrics))
	for name := range snapshot.Metrics {
		names = append(names, name)
	}
	sort.Strings(names)

	metricLines := make([]string, 0, len(names))
	for _, name := range names {
		metric := snapshot.Metrics[name]
		metricLines = append(metricLines, formatMetricJSON(metric))
	}
	lines = append(lines, strings.Join(metricLines, ",\n"))
	lines = append(lines, "  }")
	lines = append(lines, "}")

	_, err := writer.Write([]byte(strings.Join(lines, "\n") + "\n"))
	return err
}

// formatMetricJSON formats a single metric as JSON.
func formatMetricJSON(metric MetricSnapshot) string {
	switch metric.Type {
	case "histogram":
		data := metric.Value.(map[string]interface{})
		return fmt.Sprintf("    \"%s\": {\"type\": \"%s\", \"count\": %v, \"sum\": %v}",
			metric.Name, metric.Type, data["count"], data["sum"])
	default:
		return fmt.Sprintf("    \"%s\": {\"type\": \"%s\", \"value\": %v}",
			metric.Name, metric.Type, metric.Value)
	}
}

// exportPrometheus exports metrics in Prometheus format.
func (e *Exporter) exportPrometheus(snapshot *Snapshot, writer io.Writer) error {
	lines := make([]string, 0, len(snapshot.Metrics))

	// Sort metric names
	names := make([]string, 0, len(snapshot.Metrics))
	for name := range snapshot.Metrics {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		metric := snapshot.Metrics[name]
		lines = append(lines, formatMetricPrometheus(metric, snapshot.Labels))
	}

	_, err := writer.Write([]byte(strings.Join(lines, "\n") + "\n"))
	return err
}

// formatMetricPrometheus formats a single metric in Prometheus format.
func formatMetricPrometheus(metric MetricSnapshot, labels map[string]string) string {
	labelStr := formatLabels(labels)

	switch metric.Type {
	case "counter":
		return fmt.Sprintf("# TYPE %s counter\n%s%s %v", metric.Name, metric.Name, labelStr, metric.Value)
	case "gauge":
		return fmt.Sprintf("# TYPE %s gauge\n%s%s %v", metric.Name, metric.Name, labelStr, metric.Value)
	case "histogram":
		data := metric.Value.(map[string]interface{})
		lines := []string{
			fmt.Sprintf("# TYPE %s histogram", metric.Name),
			fmt.Sprintf("%s_count%s %v", metric.Name, labelStr, data["count"]),
			fmt.Sprintf("%s_sum%s %v", metric.Name, labelStr, data["sum"]),
		}
		return strings.Join(lines, "\n")
	default:
		return fmt.Sprintf("%s%s %v", metric.Name, labelStr, metric.Value)
	}
}

// formatLabels formats labels for Prometheus format.
func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}

	parts := make([]string, 0, len(labels))
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=\"%s\"", key, labels[key]))
	}

	return "{" + strings.Join(parts, ",") + "}"
}

// LogMetrics logs all metrics using the configured logger.
func (e *Exporter) LogMetrics() {
	if e.logger == nil {
		return
	}

	snapshot := e.collector.Snapshot()
	for name, metric := range snapshot.Metrics {
		fields := logger.Fields{
			"metric_name": name,
			"metric_type": metric.Type,
		}

		switch metric.Type {
		case "histogram":
			data := metric.Value.(map[string]interface{})
			fields["count"] = data["count"]
			fields["sum"] = data["sum"]
		default:
			fields["value"] = metric.Value
		}

		e.logger.WithFields(fields).Info("metric")
	}
}

// maxInt64 returns the maximum of two int64 values.
func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
