package errors_test

import (
	"fmt"

	"github.com/massonsky/buffalo/pkg/errors"
)

func ExampleNew() {
	err := errors.New(errors.ErrNotFound, "proto file not found")
	fmt.Println(err)
	// Output: [NOT_FOUND] proto file not found
}

func ExampleNew_withFormatting() {
	err := errors.New(errors.ErrNotFound, "file %s not found in directory %s", "test.proto", "/protos")
	fmt.Println(err)
	// Output: [NOT_FOUND] file test.proto not found in directory /protos
}

func ExampleWrap() {
	originalErr := fmt.Errorf("connection refused")
	err := errors.Wrap(originalErr, errors.ErrIO, "failed to connect to server")
	fmt.Println(err)
	// Output: [IO_ERROR] failed to connect to server: connection refused
}

func ExampleError_WithContext() {
	err := errors.New(errors.ErrCompilation, "protoc compilation failed").
		WithContext("file", "user.proto").
		WithContext("line", 42).
		WithContext("column", 10)

	if file, ok := err.GetContext("file"); ok {
		fmt.Printf("Error in file: %v\n", file)
	}
	if line, ok := err.GetContext("line"); ok {
		fmt.Printf("At line: %v\n", line)
	}
	// Output:
	// Error in file: user.proto
	// At line: 42
}

func ExampleError_WithContextMap() {
	context := map[string]interface{}{
		"proto_file": "service.proto",
		"language":   "python",
		"output_dir": "/gen/python",
	}

	err := errors.New(errors.ErrCompilation, "compilation failed").
		WithContextMap(context)

	if protoFile, ok := err.GetContext("proto_file"); ok {
		fmt.Printf("Proto file: %v\n", protoFile)
	}
	// Output: Proto file: service.proto
}

func ExampleIs() {
	err1 := errors.New(errors.ErrNotFound, "file not found")
	err2 := errors.New(errors.ErrNotFound, "different message")

	if errors.Is(err1, err2) {
		fmt.Println("Both errors have the same error code")
	}
	// Output: Both errors have the same error code
}

func ExampleError_StackTrace() {
	err := errors.New(errors.ErrInternal, "internal error")
	stack := err.StackTrace()

	// Stack trace will contain function names and line numbers
	fmt.Printf("Stack trace captured: %d bytes\n", len(stack))
	// Output example will vary, but demonstrates stack capture
}

func ExampleError_DetailedError() {
	err := errors.New(errors.ErrConfig, "invalid configuration").
		WithContext("config_file", "/etc/buffalo/config.yaml").
		WithContext("field", "compilers.python.path")

	// DetailedError provides comprehensive information
	// including message, code, context, and stack trace
	detailed := err.DetailedError()
	fmt.Printf("Detailed error length: %d bytes\n", len(detailed))
	// Output will include all error details
}

func ExampleUnwrap() {
	originalErr := fmt.Errorf("disk full")
	wrappedErr := errors.Wrap(originalErr, errors.ErrIO, "failed to write file")

	unwrapped := errors.Unwrap(wrappedErr)
	fmt.Println(unwrapped)
	// Output: disk full
}

// Example of error handling in practice
func ExampleError_practicalUsage() {
	// Simulating a function that might fail
	err := readProtoFile("/path/to/file.proto")
	if err != nil {
		// Check for specific error code
		if buffErr, ok := err.(*errors.Error); ok {
			switch buffErr.Code {
			case errors.ErrNotFound:
				fmt.Println("Proto file not found - please check the path")
			case errors.ErrIO:
				fmt.Println("Failed to read proto file - check permissions")
			default:
				fmt.Println("Unexpected error occurred")
			}
		}
	}
	// Output: Proto file not found - please check the path
}

func readProtoFile(path string) error {
	// Simulated error
	return errors.New(errors.ErrNotFound, "proto file %s not found", path)
}
