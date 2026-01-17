package logger

import (
	"encoding/json"
	"time"
)

// JSONFormatter formats log entries as JSON.
type JSONFormatter struct {
	// TimestampFormat is the format for timestamps.
	TimestampFormat string
	// PrettyPrint enables pretty-printed JSON.
	PrettyPrint bool
}

// NewJSONFormatter creates a new JSON formatter.
func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{
		TimestampFormat: time.RFC3339,
		PrettyPrint:     false,
	}
}

// Format formats the entry as JSON.
func (f *JSONFormatter) Format(entry *Entry) ([]byte, error) {
	data := make(map[string]interface{})

	// Add standard fields
	data["time"] = entry.Time.Format(f.TimestampFormat)
	data["level"] = entry.Level.String()
	data["message"] = entry.Message

	// Add custom fields
	for k, v := range entry.Fields {
		data[k] = v
	}

	var result []byte
	var err error

	if f.PrettyPrint {
		result, err = json.MarshalIndent(data, "", "  ")
	} else {
		result, err = json.Marshal(data)
	}

	if err != nil {
		return nil, err
	}

	// Add newline
	result = append(result, '\n')
	return result, nil
}
