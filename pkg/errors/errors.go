// Package errors provides enhanced error handling with codes, stack traces, and context.
package errors

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// Error represents an enhanced error with additional context.
type Error struct {
	// Code is the error code for categorization.
	Code ErrorCode
	// Message is the error message.
	Message string
	// Cause is the underlying error that caused this error.
	Cause error
	// Stack is the call stack where the error was created.
	Stack []uintptr
	// Context contains additional contextual information.
	Context map[string]interface{}
}

// New creates a new Error with the given code and message.
func New(code ErrorCode, message string, args ...interface{}) *Error {
	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}

	return &Error{
		Code:    code,
		Message: message,
		Stack:   captureStack(2),
		Context: make(map[string]interface{}),
	}
}

// Wrap wraps an existing error with additional context.
func Wrap(err error, code ErrorCode, message string, args ...interface{}) *Error {
	if err == nil {
		return nil
	}

	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}

	return &Error{
		Code:    code,
		Message: message,
		Cause:   err,
		Stack:   captureStack(2),
		Context: make(map[string]interface{}),
	}
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the cause of the error for errors.Unwrap.
func (e *Error) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches the target error.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// WithContext adds context to the error.
func (e *Error) WithContext(key string, value interface{}) *Error {
	e.Context[key] = value
	return e
}

// WithContextMap adds multiple context values.
func (e *Error) WithContextMap(context map[string]interface{}) *Error {
	for k, v := range context {
		e.Context[k] = v
	}
	return e
}

// GetContext retrieves a context value.
func (e *Error) GetContext(key string) (interface{}, bool) {
	v, ok := e.Context[key]
	return v, ok
}

// StackTrace returns the formatted stack trace.
func (e *Error) StackTrace() string {
	if len(e.Stack) == 0 {
		return ""
	}

	var buf strings.Builder
	frames := runtime.CallersFrames(e.Stack)

	for {
		frame, more := frames.Next()
		fmt.Fprintf(&buf, "%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line)
		if !more {
			break
		}
	}

	return buf.String()
}

// DetailedError returns a detailed error message with stack trace and context.
func (e *Error) DetailedError() string {
	var buf strings.Builder

	fmt.Fprintf(&buf, "Error: %s\n", e.Error())

	if len(e.Context) > 0 {
		buf.WriteString("Context:\n")
		for k, v := range e.Context {
			fmt.Fprintf(&buf, "  %s: %v\n", k, v)
		}
	}

	if len(e.Stack) > 0 {
		buf.WriteString("Stack Trace:\n")
		buf.WriteString(e.StackTrace())
	}

	return buf.String()
}

// captureStack captures the call stack.
func captureStack(skip int) []uintptr {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(skip+1, pcs[:])
	return pcs[:n]
}

// Standard error wrapping functions

// Is checks if err matches target using errors.Is.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's chain that matches target.
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// Unwrap returns the result of calling the Unwrap method on err.
func Unwrap(err error) error {
	return errors.Unwrap(err)
}
