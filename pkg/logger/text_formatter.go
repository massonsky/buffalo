package logger

import (
	"bytes"
	"fmt"
	"sort"
	"time"
)

// TextFormatter formats log entries as plain text.
type TextFormatter struct {
	// TimestampFormat is the format for timestamps.
	TimestampFormat string
	// DisableTimestamp disables timestamp output.
	DisableTimestamp bool
	// FullTimestamp enables full timestamp instead of just time.
	FullTimestamp bool
	// FieldsOrder defines the order of fields in output.
	FieldsOrder []string
}

// NewTextFormatter creates a new text formatter.
func NewTextFormatter() *TextFormatter {
	return &TextFormatter{
		TimestampFormat:  "15:04:05",
		DisableTimestamp: false,
		FullTimestamp:    false,
	}
}

// Format formats the entry as plain text.
func (f *TextFormatter) Format(entry *Entry) ([]byte, error) {
	var buf bytes.Buffer

	// Timestamp
	if !f.DisableTimestamp {
		var timestamp string
		if f.FullTimestamp {
			timestamp = entry.Time.Format(time.RFC3339)
		} else {
			timestamp = entry.Time.Format(f.TimestampFormat)
		}
		buf.WriteString(timestamp)
		buf.WriteByte(' ')
	}

	// Level
	buf.WriteString(fmt.Sprintf("[%-5s]", entry.Level.String()))
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

func (f *TextFormatter) writeFields(buf *bytes.Buffer, fields Fields) {
	// Get sorted keys
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}

	// Apply custom order if specified
	if len(f.FieldsOrder) > 0 {
		sort.Slice(keys, func(i, j int) bool {
			iOrder := f.getFieldOrder(keys[i])
			jOrder := f.getFieldOrder(keys[j])
			if iOrder != jOrder {
				return iOrder < jOrder
			}
			return keys[i] < keys[j]
		})
	} else {
		sort.Strings(keys)
	}

	first := true
	for _, k := range keys {
		if !first {
			buf.WriteByte(' ')
		}
		first = false

		fmt.Fprintf(buf, "%s=%v", k, fields[k])
	}
}

func (f *TextFormatter) getFieldOrder(key string) int {
	for i, k := range f.FieldsOrder {
		if k == key {
			return i
		}
	}
	return len(f.FieldsOrder)
}
