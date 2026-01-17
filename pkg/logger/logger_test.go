package logger

import (
	"bytes"
	"testing"
)

func TestLogger_BasicLogging(t *testing.T) {
	var buf bytes.Buffer

	log := New(
		WithLevel(DEBUG),
		WithFormatter(NewTextFormatter()),
		WithOutput(NewConsoleOutput(&buf)),
	)

	log.Info("test message")

	output := buf.String()
	if output == "" {
		t.Error("expected log output, got empty string")
	}

	if !contains(output, "INFO") {
		t.Errorf("expected log to contain 'INFO', got: %s", output)
	}

	if !contains(output, "test message") {
		t.Errorf("expected log to contain 'test message', got: %s", output)
	}
}

func TestLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer

	log := New(
		WithLevel(INFO),
		WithFormatter(NewTextFormatter()),
		WithOutput(NewConsoleOutput(&buf)),
	)

	log.Info("test", String("key", "value"))

	output := buf.String()
	if !contains(output, "key=value") {
		t.Errorf("expected log to contain 'key=value', got: %s", output)
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer

	log := New(
		WithLevel(WARN),
		WithFormatter(NewTextFormatter()),
		WithOutput(NewConsoleOutput(&buf)),
	)

	log.Debug("debug message")
	log.Info("info message")

	output := buf.String()
	if output != "" {
		t.Errorf("expected no output for debug/info when level is WARN, got: %s", output)
	}

	log.Warn("warn message")
	output = buf.String()
	if output == "" {
		t.Error("expected warn message to be logged")
	}
}

func TestLogger_MultipleOutputs(t *testing.T) {
	var buf1, buf2 bytes.Buffer

	log := New(
		WithLevel(INFO),
		WithFormatter(NewTextFormatter()),
		WithOutputs(
			NewConsoleOutput(&buf1),
			NewConsoleOutput(&buf2),
		),
	)

	log.Info("test message")

	if buf1.String() == "" {
		t.Error("expected output in first buffer")
	}

	if buf2.String() == "" {
		t.Error("expected output in second buffer")
	}

	if buf1.String() != buf2.String() {
		t.Error("expected same output in both buffers")
	}
}

func TestLogger_ChildLogger(t *testing.T) {
	var buf bytes.Buffer

	parent := New(
		WithLevel(INFO),
		WithFormatter(NewTextFormatter()),
		WithOutput(NewConsoleOutput(&buf)),
		WithFields(Fields{"parent": "value"}),
	)

	child := parent.WithFields(Fields{"child": "value"})
	child.Info("test")

	output := buf.String()
	if !contains(output, "parent=value") {
		t.Errorf("expected parent field in output, got: %s", output)
	}

	if !contains(output, "child=value") {
		t.Errorf("expected child field in output, got: %s", output)
	}
}

func TestLevel_ParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"DEBUG", DEBUG},
		{"debug", DEBUG},
		{"INFO", INFO},
		{"info", INFO},
		{"WARN", WARN},
		{"warn", WARN},
		{"ERROR", ERROR},
		{"error", ERROR},
		{"FATAL", FATAL},
		{"fatal", FATAL},
		{"unknown", INFO}, // default
	}

	for _, tt := range tests {
		result := ParseLevel(tt.input)
		if result != tt.expected {
			t.Errorf("ParseLevel(%s) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{FATAL, "FATAL"},
	}

	for _, tt := range tests {
		result := tt.level.String()
		if result != tt.expected {
			t.Errorf("Level(%d).String() = %s, want %s", tt.level, result, tt.expected)
		}
	}
}

func TestFieldHelpers(t *testing.T) {
	tests := []struct {
		name     string
		field    Field
		expected interface{}
	}{
		{"String", String("key", "value"), "value"},
		{"Int", Int("key", 42), 42},
		{"Bool", Bool("key", true), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.field.Key != "key" {
				t.Errorf("expected key 'key', got '%s'", tt.field.Key)
			}
			if tt.field.Value != tt.expected {
				t.Errorf("expected value %v, got %v", tt.expected, tt.field.Value)
			}
		})
	}
}

func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}

func BenchmarkLogger_Info(b *testing.B) {
	var buf bytes.Buffer
	log := New(
		WithLevel(INFO),
		WithFormatter(NewTextFormatter()),
		WithOutput(NewConsoleOutput(&buf)),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Info("benchmark message")
	}
}

func BenchmarkLogger_InfoWithFields(b *testing.B) {
	var buf bytes.Buffer
	log := New(
		WithLevel(INFO),
		WithFormatter(NewTextFormatter()),
		WithOutput(NewConsoleOutput(&buf)),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Info("benchmark message",
			String("key1", "value1"),
			Int("key2", 42),
			Bool("key3", true),
		)
	}
}
