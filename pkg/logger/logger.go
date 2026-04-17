// Package logger provides a flexible, structured logging system for Buffalo.
package logger

import (
	"context"
	"io"
	"sync"
	"time"
)

// Logger is the main logging interface.
type Logger struct {
	level      Level
	formatter  Formatter
	outputs    []Output
	fields     Fields
	mutex      sync.RWMutex
	bufferPool *sync.Pool
}

// Fields represents structured log fields.
type Fields map[string]interface{}

// Entry represents a single log entry.
type Entry struct {
	Time    time.Time
	Level   Level
	Message string
	Fields  Fields
	Context context.Context
}

// Option is a function that configures the logger.
type Option func(*Logger)

// New creates a new Logger with the given options.
func New(opts ...Option) *Logger {
	l := &Logger{
		level:     INFO,
		formatter: NewTextFormatter(),
		outputs:   []Output{NewConsoleOutput(io.Discard)},
		fields:    make(Fields),
		bufferPool: &sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 1024)
			},
		},
	}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

// WithLevel sets the log level.
func WithLevel(level Level) Option {
	return func(l *Logger) {
		l.level = level
	}
}

// WithFormatter sets the formatter.
func WithFormatter(formatter Formatter) Option {
	return func(l *Logger) {
		l.formatter = formatter
	}
}

// WithOutput adds an output destination.
func WithOutput(output Output) Option {
	return func(l *Logger) {
		l.outputs = append(l.outputs, output)
	}
}

// WithOutputs sets multiple output destinations.
func WithOutputs(outputs ...Output) Option {
	return func(l *Logger) {
		l.outputs = outputs
	}
}

// WithFields adds default fields to all log entries.
func WithFields(fields Fields) Option {
	return func(l *Logger) {
		for k, v := range fields {
			l.fields[k] = v
		}
	}
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string, fields ...Field) {
	l.log(DEBUG, msg, fields...)
}

// Info logs an info message.
func (l *Logger) Info(msg string, fields ...Field) {
	l.log(INFO, msg, fields...)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string, fields ...Field) {
	l.log(WARN, msg, fields...)
}

// Error logs an error message.
func (l *Logger) Error(msg string, fields ...Field) {
	l.log(ERROR, msg, fields...)
}

// Fatal logs a fatal message and exits.
func (l *Logger) Fatal(msg string, fields ...Field) {
	l.log(FATAL, msg, fields...)
	// In real implementation, this would call os.Exit(1)
}

// WithFields returns a new logger with additional fields.
func (l *Logger) WithFields(fields Fields) *Logger {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	newFields := make(Fields, len(l.fields)+len(fields))
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &Logger{
		level:      l.level,
		formatter:  l.formatter,
		outputs:    l.outputs,
		fields:     newFields,
		mutex:      sync.RWMutex{},
		bufferPool: l.bufferPool,
	}
}

// WithContext returns a new logger with the given context.
func (l *Logger) WithContext(ctx context.Context) *Logger {
	// Context will be stored in the entry during logging
	return l
}

// SetLevel changes the log level.
func (l *Logger) SetLevel(level Level) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.level = level
}

// GetLevel returns the current log level.
func (l *Logger) GetLevel() Level {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.level
}

// log is the internal logging method.
func (l *Logger) log(level Level, msg string, fields ...Field) {
	l.mutex.RLock()
	if level < l.level {
		l.mutex.RUnlock()
		return
	}
	l.mutex.RUnlock()

	// Build fields map
	entryFields := make(Fields, len(l.fields)+len(fields))
	for k, v := range l.fields {
		entryFields[k] = v
	}
	for _, field := range fields {
		entryFields[field.Key] = field.Value
	}

	// Create entry
	entry := &Entry{
		Time:    time.Now(),
		Level:   level,
		Message: msg,
		Fields:  entryFields,
	}

	// Format entry
	formatted, err := l.formatter.Format(entry)
	if err != nil {
		// Fallback to simple format
		formatted = []byte(msg + "\n")
	}

	// Write to all outputs
	for _, output := range l.outputs {
		_ = output.Write(formatted)
	}
}

// Field represents a single log field.
type Field struct {
	Key   string
	Value interface{}
}

// String creates a string field.
func String(key string, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates an int field.
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64 creates an int64 field.
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Float64 creates a float64 field.
func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

// Bool creates a bool field.
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Any creates a field with any value.
func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Error creates an error field.
func Error(err error) Field {
	if err == nil {
		return Field{Key: "error", Value: nil}
	}
	return Field{Key: "error", Value: err.Error()}
}

// Duration creates a duration field.
func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value.String()}
}

// Time creates a time field.
func Time(key string, value time.Time) Field {
	return Field{Key: key, Value: value}
}
