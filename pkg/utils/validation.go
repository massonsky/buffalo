package utils

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/massonsky/buffalo/pkg/errors"
)

var (
	// protoFileNameRegex matches valid proto file names.
	// Proto files must have .proto extension and valid base name.
	// Allow both uppercase and lowercase in file names
	protoFileNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*\.proto$`)

	// packageNameRegex matches valid proto package names.
	packageNameRegex = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)*$`)
)

// ValidationResult contains the result of a validation.
type ValidationResult struct {
	Valid    bool     // Whether the validation passed
	Errors   []string // List of validation errors
	Warnings []string // List of validation warnings
}

// AddError adds an error to the validation result.
func (vr *ValidationResult) AddError(format string, args ...interface{}) {
	vr.Valid = false
	vr.Errors = append(vr.Errors, strings.TrimSpace(formatMessage(format, args...)))
}

// AddWarning adds a warning to the validation result.
func (vr *ValidationResult) AddWarning(format string, args ...interface{}) {
	vr.Warnings = append(vr.Warnings, strings.TrimSpace(formatMessage(format, args...)))
}

// HasErrors returns true if there are any errors.
func (vr *ValidationResult) HasErrors() bool {
	return len(vr.Errors) > 0
}

// HasWarnings returns true if there are any warnings.
func (vr *ValidationResult) HasWarnings() bool {
	return len(vr.Warnings) > 0
}

// formatMessage formats a message with arguments.
func formatMessage(format string, args ...interface{}) string {
	if len(args) == 0 {
		return format
	}
	return strings.TrimSpace(strings.ReplaceAll(format, "%s", "%v"))
}

// ValidateProtoFile validates a proto file.
func ValidateProtoFile(path string) (*ValidationResult, error) {
	result := &ValidationResult{Valid: true}

	// Check if path is empty
	if path == "" {
		return nil, errors.New(errors.ErrInvalidArgument, "path cannot be empty")
	}

	// Check if file exists
	if !FileExists(path) {
		result.AddError("proto file does not exist: %s", path)
		return result, nil
	}

	// Check if it's a file (not a directory)
	if IsDir(path) {
		result.AddError("path is a directory, not a file: %s", path)
		return result, nil
	}

	// Check file extension
	if !HasExtension(path, ".proto") {
		result.AddError("file does not have .proto extension: %s", path)
		return result, nil
	}

	// Validate file name
	baseName := filepath.Base(path)
	if !protoFileNameRegex.MatchString(baseName) {
		result.AddError("invalid proto file name: %s (must match pattern: [a-zA-Z_][a-zA-Z0-9_]*.proto)", baseName)
	}

	// Check if file is readable
	file, err := os.Open(path)
	if err != nil {
		result.AddError("cannot read proto file: %s", err.Error())
		return result, nil
	}
	file.Close()

	// Check file size (warn if empty or very large)
	size, err := GetFileSize(path)
	if err != nil {
		result.AddError("cannot get file size: %s", err.Error())
		return result, nil
	}

	if size == 0 {
		result.AddWarning("proto file is empty: %s", path)
	} else if size > 1024*1024 { // > 1MB
		result.AddWarning("proto file is very large (%d bytes): %s", size, path)
	}

	return result, nil
}

// ValidateProtoPackageName validates a proto package name.
func ValidateProtoPackageName(packageName string) bool {
	if packageName == "" {
		return false
	}
	return packageNameRegex.MatchString(packageName)
}

// ValidateOutputDir validates an output directory.
func ValidateOutputDir(path string) (*ValidationResult, error) {
	result := &ValidationResult{Valid: true}

	// Check if path is empty
	if path == "" {
		return nil, errors.New(errors.ErrInvalidArgument, "path cannot be empty")
	}

	// Check if directory exists
	if !FileExists(path) {
		result.AddWarning("output directory does not exist (will be created): %s", path)
		return result, nil
	}

	// Check if it's a directory
	if !IsDir(path) {
		result.AddError("path is not a directory: %s", path)
		return result, nil
	}

	// Check if directory is writable by trying to create a temp file
	tempFile := filepath.Join(path, ".buffalo_test_write")
	if err := os.WriteFile(tempFile, []byte("test"), 0644); err != nil {
		result.AddError("output directory is not writable: %s", path)
		return result, nil
	}
	os.Remove(tempFile)

	return result, nil
}

// ValidatePath validates a file or directory path.
func ValidatePath(path string) (*ValidationResult, error) {
	result := &ValidationResult{Valid: true}

	// Check if path is empty
	if path == "" {
		return nil, errors.New(errors.ErrInvalidArgument, "path cannot be empty")
	}

	// Check if path exists
	if !FileExists(path) {
		result.AddError("path does not exist: %s", path)
		return result, nil
	}

	// Check if path is accessible
	if _, err := os.Stat(path); err != nil {
		result.AddError("cannot access path: %s", err.Error())
		return result, nil
	}

	return result, nil
}

// ValidateFilePattern validates a file pattern for searching.
func ValidateFilePattern(pattern string) error {
	if pattern == "" {
		return errors.New(errors.ErrInvalidArgument, "pattern cannot be empty")
	}

	// Try to match the pattern with a test string
	_, err := filepath.Match(pattern, "test.proto")
	if err != nil {
		return errors.Wrap(err, errors.ErrInvalidArgument, "invalid file pattern: %s", pattern)
	}

	return nil
}

// IsValidProtoFileName checks if a file name is a valid proto file name.
func IsValidProtoFileName(name string) bool {
	return protoFileNameRegex.MatchString(name)
}

// IsValidPackageName checks if a package name is valid.
func IsValidPackageName(name string) bool {
	return ValidateProtoPackageName(name)
}
