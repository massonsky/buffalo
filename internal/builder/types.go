package builder

import (
	"github.com/massonsky/buffalo/pkg/logger"
)

// Logger is a minimal interface for logging
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// loggerAdapter adapts pkg/logger.Logger to builder.Logger interface
type loggerAdapter struct {
	log *logger.Logger
}

// NewLoggerAdapter creates a new logger adapter
func NewLoggerAdapter(log *logger.Logger) Logger {
	return &loggerAdapter{log: log}
}

func (l *loggerAdapter) Debug(msg string, args ...interface{}) {
	fields := convertArgsToFields(args)
	l.log.Debug(msg, fields...)
}

func (l *loggerAdapter) Info(msg string, args ...interface{}) {
	fields := convertArgsToFields(args)
	l.log.Info(msg, fields...)
}

func (l *loggerAdapter) Warn(msg string, args ...interface{}) {
	fields := convertArgsToFields(args)
	l.log.Warn(msg, fields...)
}

func (l *loggerAdapter) Error(msg string, args ...interface{}) {
	fields := convertArgsToFields(args)
	l.log.Error(msg, fields...)
}

// convertArgsToFields converts key-value pairs to logger.Field
func convertArgsToFields(args []interface{}) []logger.Field {
	fields := make([]logger.Field, 0, len(args)/2)
	for i := 0; i < len(args)-1; i += 2 {
		key, ok := args[i].(string)
		if !ok {
			continue
		}
		value := args[i+1]
		fields = append(fields, logger.Any(key, value))
	}
	return fields
}
