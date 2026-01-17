package errors

import (
	"fmt"
	"testing"
)

func TestNew(t *testing.T) {
	err := New(ErrNotFound, "resource not found")

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err.Code != ErrNotFound {
		t.Errorf("expected code %s, got %s", ErrNotFound, err.Code)
	}

	if err.Message != "resource not found" {
		t.Errorf("expected message 'resource not found', got '%s'", err.Message)
	}

	if len(err.Stack) == 0 {
		t.Error("expected stack trace to be captured")
	}
}

func TestNew_WithFormatting(t *testing.T) {
	err := New(ErrNotFound, "file %s not found in %s", "test.proto", "/path/to/proto")

	expected := "file test.proto not found in /path/to/proto"
	if err.Message != expected {
		t.Errorf("expected message '%s', got '%s'", expected, err.Message)
	}
}

func TestWrap(t *testing.T) {
	cause := fmt.Errorf("original error")
	err := Wrap(cause, ErrIO, "failed to read file")

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err.Code != ErrIO {
		t.Errorf("expected code %s, got %s", ErrIO, err.Code)
	}

	if err.Cause != cause {
		t.Error("expected cause to be set")
	}

	if err.Unwrap() != cause {
		t.Error("Unwrap() should return the cause")
	}
}

func TestWrap_NilError(t *testing.T) {
	err := Wrap(nil, ErrIO, "failed to read file")

	if err != nil {
		t.Error("expected nil when wrapping nil error")
	}
}

func TestError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected string
	}{
		{
			name:     "error without cause",
			err:      New(ErrNotFound, "resource not found"),
			expected: "[NOT_FOUND] resource not found",
		},
		{
			name:     "error with cause",
			err:      Wrap(fmt.Errorf("original"), ErrIO, "failed to read"),
			expected: "[IO_ERROR] failed to read: original",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestError_Is(t *testing.T) {
	err1 := New(ErrNotFound, "test1")
	err2 := New(ErrNotFound, "test2")
	err3 := New(ErrIO, "test3")

	if !Is(err1, err2) {
		t.Error("expected errors with same code to match")
	}

	if Is(err1, err3) {
		t.Error("expected errors with different codes to not match")
	}
}

func TestError_WithContext(t *testing.T) {
	err := New(ErrNotFound, "file not found").
		WithContext("file", "test.proto").
		WithContext("path", "/path/to/proto")

	file, ok := err.GetContext("file")
	if !ok {
		t.Error("expected 'file' context to be set")
	}
	if file != "test.proto" {
		t.Errorf("expected file 'test.proto', got '%v'", file)
	}

	path, ok := err.GetContext("path")
	if !ok {
		t.Error("expected 'path' context to be set")
	}
	if path != "/path/to/proto" {
		t.Errorf("expected path '/path/to/proto', got '%v'", path)
	}
}

func TestError_WithContextMap(t *testing.T) {
	context := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}

	err := New(ErrInternal, "test").WithContextMap(context)

	val1, ok := err.GetContext("key1")
	if !ok || val1 != "value1" {
		t.Error("expected key1 to be set")
	}

	val2, ok := err.GetContext("key2")
	if !ok || val2 != 42 {
		t.Error("expected key2 to be set")
	}
}

func TestError_StackTrace(t *testing.T) {
	err := New(ErrInternal, "test error")

	stack := err.StackTrace()
	if stack == "" {
		t.Error("expected stack trace to not be empty")
	}

	// Should contain function name
	if !contains(stack, "TestError_StackTrace") {
		t.Errorf("expected stack to contain function name, got:\n%s", stack)
	}

	// Should contain file name
	if !contains(stack, "errors_test.go") {
		t.Errorf("expected stack to contain file name, got:\n%s", stack)
	}
}

func TestError_DetailedError(t *testing.T) {
	err := New(ErrNotFound, "resource not found").
		WithContext("id", "123").
		WithContext("type", "user")

	detailed := err.DetailedError()

	// Should contain error message
	if !contains(detailed, "resource not found") {
		t.Error("expected detailed error to contain error message")
	}

	// Should contain context
	if !contains(detailed, "Context:") {
		t.Error("expected detailed error to contain context section")
	}

	if !contains(detailed, "id: 123") {
		t.Error("expected detailed error to contain context values")
	}

	// Should contain stack trace
	if !contains(detailed, "Stack Trace:") {
		t.Error("expected detailed error to contain stack trace")
	}
}

func TestErrorCode_String(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		expected string
	}{
		{ErrNotFound, "NOT_FOUND"},
		{ErrIO, "IO_ERROR"},
		{ErrConfig, "CONFIG_ERROR"},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			result := tt.code.String()
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestError_UnwrapChain(t *testing.T) {
	err1 := fmt.Errorf("level 1")
	err2 := Wrap(err1, ErrIO, "level 2")
	err3 := Wrap(err2, ErrInternal, "level 3")

	// Test Unwrap
	unwrapped := Unwrap(err3)
	if unwrapped != err2 {
		t.Error("expected to unwrap to err2")
	}

	// Test Is with chain
	if !Is(err3, err2) {
		t.Error("expected Is to work with error chain")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func BenchmarkNew(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = New(ErrNotFound, "test error")
	}
}

func BenchmarkWrap(b *testing.B) {
	cause := fmt.Errorf("cause")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Wrap(cause, ErrIO, "wrapped error")
	}
}

func BenchmarkError_WithContext(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = New(ErrNotFound, "test").WithContext("key", "value")
	}
}
