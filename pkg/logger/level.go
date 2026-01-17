package logger

import "strings"

// Level represents the severity level of a log entry.
type Level int

const (
	// DEBUG level for detailed debugging information.
	DEBUG Level = iota
	// INFO level for general informational messages.
	INFO
	// WARN level for warning messages.
	WARN
	// ERROR level for error messages.
	ERROR
	// FATAL level for fatal error messages.
	FATAL
)

// String returns the string representation of the level.
func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// ShortString returns a short string representation of the level.
func (l Level) ShortString() string {
	switch l {
	case DEBUG:
		return "DBG"
	case INFO:
		return "INF"
	case WARN:
		return "WRN"
	case ERROR:
		return "ERR"
	case FATAL:
		return "FTL"
	default:
		return "UNK"
	}
}

// ParseLevel parses a string into a Level.
func ParseLevel(s string) Level {
	switch strings.ToUpper(s) {
	case "DEBUG", "DBG":
		return DEBUG
	case "INFO", "INF":
		return INFO
	case "WARN", "WARNING", "WRN":
		return WARN
	case "ERROR", "ERR":
		return ERROR
	case "FATAL", "FTL":
		return FATAL
	default:
		return INFO
	}
}

// MarshalText implements encoding.TextMarshaler.
func (l Level) MarshalText() ([]byte, error) {
	return []byte(l.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (l *Level) UnmarshalText(text []byte) error {
	*l = ParseLevel(string(text))
	return nil
}
