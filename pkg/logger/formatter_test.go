package logger

import (
	"encoding/json"
	"testing"
	"time"
)

func TestJSONFormatter_Format(t *testing.T) {
	formatter := NewJSONFormatter()
	entry := &Entry{
		Time:    time.Date(2026, 1, 17, 12, 0, 0, 0, time.UTC),
		Level:   INFO,
		Message: "test message",
		Fields: Fields{
			"key": "value",
		},
	}

	result, err := formatter.Format(entry)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Parse JSON to verify it's valid
	var data map[string]interface{}
	if err := json.Unmarshal(result, &data); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify fields
	if data["level"] != "INFO" {
		t.Errorf("expected level 'INFO', got '%v'", data["level"])
	}

	if data["message"] != "test message" {
		t.Errorf("expected message 'test message', got '%v'", data["message"])
	}

	if data["key"] != "value" {
		t.Errorf("expected key 'value', got '%v'", data["key"])
	}
}

func TestJSONFormatter_PrettyPrint(t *testing.T) {
	formatter := &JSONFormatter{
		TimestampFormat: time.RFC3339,
		PrettyPrint:     true,
	}

	entry := &Entry{
		Time:    time.Now(),
		Level:   INFO,
		Message: "test",
		Fields:  Fields{},
	}

	result, err := formatter.Format(entry)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Pretty printed JSON should contain newlines
	if !contains(string(result), "\n  ") {
		t.Error("expected pretty-printed JSON with indentation")
	}
}

func TestTextFormatter_Format(t *testing.T) {
	formatter := NewTextFormatter()
	entry := &Entry{
		Time:    time.Date(2026, 1, 17, 12, 30, 45, 0, time.UTC),
		Level:   INFO,
		Message: "test message",
		Fields: Fields{
			"key1": "value1",
			"key2": 42,
		},
	}

	result, err := formatter.Format(entry)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// Check for timestamp
	if !contains(output, "12:30:45") {
		t.Errorf("expected timestamp in output, got: %s", output)
	}

	// Check for level
	if !contains(output, "[INFO ]") {
		t.Errorf("expected '[INFO ]' in output, got: %s", output)
	}

	// Check for message
	if !contains(output, "test message") {
		t.Errorf("expected 'test message' in output, got: %s", output)
	}

	// Check for fields
	if !contains(output, "key1=value1") {
		t.Errorf("expected 'key1=value1' in output, got: %s", output)
	}

	if !contains(output, "key2=42") {
		t.Errorf("expected 'key2=42' in output, got: %s", output)
	}
}

func TestTextFormatter_DisableTimestamp(t *testing.T) {
	formatter := &TextFormatter{
		DisableTimestamp: true,
	}

	entry := &Entry{
		Time:    time.Now(),
		Level:   INFO,
		Message: "test",
		Fields:  Fields{},
	}

	result, err := formatter.Format(entry)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// Should not contain timestamp digits
	if contains(output, ":") && contains(output, "202") {
		t.Errorf("expected no timestamp, got: %s", output)
	}
}

func TestColoredFormatter_Format(t *testing.T) {
	formatter := NewColoredFormatter()
	entry := &Entry{
		Time:    time.Now(),
		Level:   INFO,
		Message: "test message",
		Fields:  Fields{"key": "value"},
	}

	result, err := formatter.Format(entry)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// Should contain color codes
	if !contains(output, "\033[") {
		t.Errorf("expected color codes in output, got: %s", output)
	}

	// Should contain reset code
	if !contains(output, colorReset) {
		t.Errorf("expected color reset in output, got: %s", output)
	}
}

func TestColoredFormatter_DisableColors(t *testing.T) {
	formatter := &ColoredFormatter{
		DisableColors: true,
	}

	entry := &Entry{
		Time:    time.Now(),
		Level:   INFO,
		Message: "test",
		Fields:  Fields{},
	}

	result, err := formatter.Format(entry)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(result)

	// Should not contain color codes
	if contains(output, "\033[") {
		t.Errorf("expected no color codes when disabled, got: %s", output)
	}
}

func TestColoredFormatter_LevelColors(t *testing.T) {
	formatter := NewColoredFormatter()

	tests := []struct {
		level         Level
		expectedColor string
	}{
		{DEBUG, colorBlue},
		{INFO, colorGreen},
		{WARN, colorYellow},
		{ERROR, colorRed},
		{FATAL, colorPurple},
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			entry := &Entry{
				Time:    time.Now(),
				Level:   tt.level,
				Message: "test",
				Fields:  Fields{},
			}

			result, err := formatter.Format(entry)
			if err != nil {
				t.Fatalf("Format() error = %v", err)
			}

			output := string(result)
			if !contains(output, tt.expectedColor) {
				t.Errorf("expected color %s in output, got: %s", tt.expectedColor, output)
			}
		})
	}
}
