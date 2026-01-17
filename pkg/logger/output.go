package logger

// Output is the interface for log output destinations.
type Output interface {
	// Write writes the formatted log entry.
	Write(p []byte) error
	// Close closes the output.
	Close() error
}
