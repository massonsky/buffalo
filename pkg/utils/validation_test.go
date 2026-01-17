package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateProtoFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create valid proto file
	validProto := filepath.Join(tempDir, "valid.proto")
	if err := os.WriteFile(validProto, []byte("syntax = \"proto3\";"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create empty proto file
	emptyProto := filepath.Join(tempDir, "empty.proto")
	if err := os.WriteFile(emptyProto, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	// Create invalid named file
	invalidName := filepath.Join(tempDir, "123invalid.proto")
	if err := os.WriteFile(invalidName, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		path           string
		expectValid    bool
		expectErrors   bool
		expectWarnings bool
	}{
		{
			name:           "valid proto file",
			path:           validProto,
			expectValid:    true,
			expectErrors:   false,
			expectWarnings: false,
		},
		{
			name:           "empty proto file",
			path:           emptyProto,
			expectValid:    true,
			expectErrors:   false,
			expectWarnings: true,
		},
		{
			name:         "non-existent file",
			path:         filepath.Join(tempDir, "notexist.proto"),
			expectValid:  false,
			expectErrors: true,
		},
		{
			name:         "directory instead of file",
			path:         tempDir,
			expectValid:  false,
			expectErrors: true,
		},
		{
			name:         "invalid file name",
			path:         invalidName,
			expectValid:  false,
			expectErrors: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateProtoFile(tt.path)
			if err != nil {
				t.Fatalf("ValidateProtoFile returned error: %v", err)
			}

			if result.Valid != tt.expectValid {
				t.Errorf("expected valid=%v, got %v", tt.expectValid, result.Valid)
			}

			if tt.expectErrors && !result.HasErrors() {
				t.Error("expected errors but got none")
			}

			if !tt.expectErrors && result.HasErrors() {
				t.Errorf("unexpected errors: %v", result.Errors)
			}

			if tt.expectWarnings && !result.HasWarnings() {
				t.Error("expected warnings but got none")
			}
		})
	}
}

func TestValidateProtoPackageName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid simple", "mypackage", true},
		{"valid nested", "my.package.name", true},
		{"valid with numbers", "package123", true},
		{"invalid uppercase", "MyPackage", false},
		{"invalid start with number", "123package", false},
		{"invalid dash", "my-package", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateProtoPackageName(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestValidateOutputDir(t *testing.T) {
	tempDir := t.TempDir()

	// Create existing directory
	existingDir := filepath.Join(tempDir, "existing")
	if err := os.MkdirAll(existingDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a file (not directory)
	file := filepath.Join(tempDir, "file.txt")
	if err := os.WriteFile(file, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name         string
		path         string
		expectValid  bool
		expectErrors bool
	}{
		{
			name:         "existing writable directory",
			path:         existingDir,
			expectValid:  true,
			expectErrors: false,
		},
		{
			name:         "non-existent directory",
			path:         filepath.Join(tempDir, "newdir"),
			expectValid:  true, // Should be valid with warning
			expectErrors: false,
		},
		{
			name:         "file instead of directory",
			path:         file,
			expectValid:  false,
			expectErrors: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateOutputDir(tt.path)
			if err != nil {
				t.Fatalf("ValidateOutputDir returned error: %v", err)
			}

			if result.Valid != tt.expectValid {
				t.Errorf("expected valid=%v, got %v", tt.expectValid, result.Valid)
			}

			if tt.expectErrors && !result.HasErrors() {
				t.Error("expected errors but got none")
			}
		})
	}
}

func TestValidatePath(t *testing.T) {
	tempDir := t.TempDir()

	// Create test file
	file := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(file, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		path        string
		expectValid bool
	}{
		{"existing file", file, true},
		{"existing directory", tempDir, true},
		{"non-existent path", filepath.Join(tempDir, "notexist"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidatePath(tt.path)
			if err != nil {
				t.Fatalf("ValidatePath returned error: %v", err)
			}

			if result.Valid != tt.expectValid {
				t.Errorf("expected valid=%v, got %v", tt.expectValid, result.Valid)
			}
		})
	}
}

func TestValidateFilePattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		wantErr bool
	}{
		{"valid pattern", "*.proto", false},
		{"valid complex", "test_*.proto", false},
		{"invalid pattern", "[", true},
		{"empty pattern", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilePattern(tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("expected error: %v, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestIsValidProtoFileName(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{"valid", "service.proto", true},
		{"valid with underscore", "my_service.proto", true},
		{"valid uppercase", "Service.proto", true},
		{"valid mixed case", "MyService.proto", true},
		{"invalid no extension", "service", false},
		{"invalid wrong extension", "service.txt", false},
		{"invalid start with number", "123service.proto", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidProtoFileName(tt.filename)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestValidationResult(t *testing.T) {
	result := &ValidationResult{Valid: true}

	// Test AddError
	result.AddError("error message")
	if result.Valid {
		t.Error("expected Valid to be false after AddError")
	}
	if !result.HasErrors() {
		t.Error("expected HasErrors to be true")
	}
	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors))
	}

	// Test AddWarning
	result.AddWarning("warning message")
	if !result.HasWarnings() {
		t.Error("expected HasWarnings to be true")
	}
	if len(result.Warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(result.Warnings))
	}
}
