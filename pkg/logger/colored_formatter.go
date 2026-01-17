package logger

import (
	"bytes"
	"fmt"
	"sort"
	"time"
)

// Color codes for terminal output.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[37m"
	colorWhite  = "\033[97m"
)

// ColoredFormatter formats log entries with colors for terminal output.
type ColoredFormatter struct {
	// TimestampFormat is the format for timestamps.
	TimestampFormat string
	// DisableTimestamp disables timestamp output.
	DisableTimestamp bool
	// DisableColors disables color output.
	DisableColors bool
	// FullTimestamp enables full timestamp instead of just time.
	FullTimestamp bool
}

// NewColoredFormatter creates a new colored formatter.
func NewColoredFormatter() *ColoredFormatter {
	return &ColoredFormatter{
		TimestampFormat:  "15:04:05",
		DisableTimestamp: false,
		DisableColors:    false,
		FullTimestamp:    false,
	}
}

// Format formats the entry with colors.
func (f *ColoredFormatter) Format(entry *Entry) ([]byte, error) {
	var buf bytes.Buffer

	levelColor := f.getLevelColor(entry.Level)

	// Timestamp
	if !f.DisableTimestamp {
		var timestamp string
		if f.FullTimestamp {
			timestamp = entry.Time.Format(time.RFC3339)
		} else {
			timestamp = entry.Time.Format(f.TimestampFormat)
		}

		if !f.DisableColors {
			buf.WriteString(colorGray)
		}
		buf.WriteString(timestamp)
		if !f.DisableColors {
			buf.WriteString(colorReset)
		}
		buf.WriteByte(' ')
	}

	// Level with color
	if !f.DisableColors {
		buf.WriteString(levelColor)
	}
	buf.WriteString(fmt.Sprintf("[%-5s]", entry.Level.String()))
	if !f.DisableColors {
		buf.WriteString(colorReset)
	}
	buf.WriteByte(' ')

	// Message
	buf.WriteString(entry.Message)

	// Fields
	if len(entry.Fields) > 0 {
		buf.WriteByte(' ')
		f.writeFields(&buf, entry.Fields)
	}

	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

func (f *ColoredFormatter) writeFields(buf *bytes.Buffer, fields Fields) {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	first := true
	for _, k := range keys {
		if !first {
			buf.WriteByte(' ')
		}
		first = false

		if !f.DisableColors {
			buf.WriteString(colorCyan)
		}
		buf.WriteString(k)
		if !f.DisableColors {
			buf.WriteString(colorReset)
		}
		buf.WriteByte('=')
		fmt.Fprintf(buf, "%v", fields[k])
	}
}

func (f *ColoredFormatter) getLevelColor(level Level) string {
	if f.DisableColors {
		return ""
	}

	switch level {
	case DEBUG:
		return colorBlue
	case INFO:
		return colorGreen
	case WARN:
		return colorYellow
	case ERROR:
		return colorRed
	case FATAL:
		return colorPurple
	default:
		return colorWhite
	}
}
