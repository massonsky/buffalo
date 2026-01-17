package logger

import (
	"io"
	"os"
)

// ConsoleOutput writes logs to a writer (typically os.Stdout or os.Stderr).
type ConsoleOutput struct {
	writer io.Writer
}

// NewConsoleOutput creates a new console output.
func NewConsoleOutput(writer io.Writer) *ConsoleOutput {
	if writer == nil {
		writer = os.Stdout
	}
	return &ConsoleOutput{
		writer: writer,
	}
}

// Write writes the log entry to the console.
func (o *ConsoleOutput) Write(p []byte) error {
	_, err := o.writer.Write(p)
	return err
}

// Close closes the output (no-op for console).
func (o *ConsoleOutput) Close() error {
	return nil
}

// NewStdoutOutput creates a console output that writes to stdout.
func NewStdoutOutput() *ConsoleOutput {
	return NewConsoleOutput(os.Stdout)
}

// NewStderrOutput creates a console output that writes to stderr.
func NewStderrOutput() *ConsoleOutput {
	return NewConsoleOutput(os.Stderr)
}
