package logger

// Formatter is the interface for formatting log entries.
type Formatter interface {
	// Format formats a log entry into bytes.
	Format(entry *Entry) ([]byte, error)
}
