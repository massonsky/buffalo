package builder

import (
	"bufio"
	"context"
	"os"
	"strings"
)

// ProtoFile represents a parsed proto file
type ProtoFile struct {
	// Path is the file path
	Path string

	// Package is the proto package name
	Package string

	// Syntax is the proto syntax version (proto2 or proto3)
	Syntax string

	// Imports are the imported proto files
	Imports []string

	// Services are the defined services
	Services []Service

	// Messages are the defined messages
	Messages []Message

	// Enums are the defined enums
	Enums []Enum

	// Options are file-level options
	Options map[string]string
}

// Service represents a gRPC service
type Service struct {
	Name    string
	Methods []Method
}

// Method represents a service method
type Method struct {
	Name       string
	InputType  string
	OutputType string
	Options    map[string]string
}

// Message represents a proto message
type Message struct {
	Name   string
	Fields []Field
}

// Field represents a message field
type Field struct {
	Name     string
	Type     string
	Number   int
	Repeated bool
	Optional bool
}

// Enum represents a proto enum
type Enum struct {
	Name   string
	Values []EnumValue
}

// EnumValue represents an enum value
type EnumValue struct {
	Name   string
	Number int
}

// ProtoParser parses proto files
type ProtoParser interface {
	// ParseFiles parses multiple proto files
	ParseFiles(ctx context.Context, files []string, importPaths []string) ([]*ProtoFile, error)

	// ParseFile parses a single proto file
	ParseFile(ctx context.Context, path string, importPaths []string) (*ProtoFile, error)
}

// protoParser implements ProtoParser
type protoParser struct {
	log Logger
}

// NewProtoParser creates a new ProtoParser
func NewProtoParser(log Logger) ProtoParser {
	return &protoParser{log: log}
}

// ParseFiles parses multiple proto files
func (p *protoParser) ParseFiles(ctx context.Context, files []string, importPaths []string) ([]*ProtoFile, error) {
	p.log.Debug("Parsing proto files", "count", len(files))

	var result []*ProtoFile
	for _, file := range files {
		protoFile, err := p.ParseFile(ctx, file, importPaths)
		if err != nil {
			return nil, err
		}
		result = append(result, protoFile)
	}

	return result, nil
}

// ParseFile parses a single proto file
func (p *protoParser) ParseFile(ctx context.Context, path string, importPaths []string) (*ProtoFile, error) {
	p.log.Debug("Parsing proto file", "path", path)

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	protoFile := &ProtoFile{
		Path:    path,
		Options: make(map[string]string),
	}

	scanner := bufio.NewScanner(f)
	inMessage := false
	inService := false
	inEnum := false
	var currentMessage *Message
	var currentService *Service
	var currentEnum *Enum
	braceDepth := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// Remove inline comments
		if idx := strings.Index(line, "//"); idx > 0 {
			line = strings.TrimSpace(line[:idx])
		}

		// Track brace depth for nested blocks
		openBraces := strings.Count(line, "{")
		closeBraces := strings.Count(line, "}")

		if inMessage && braceDepth == 1 {
			if closeBraces > 0 && braceDepth+openBraces-closeBraces < 1 {
				protoFile.Messages = append(protoFile.Messages, *currentMessage)
				currentMessage = nil
				inMessage = false
				braceDepth += openBraces - closeBraces
				continue
			}
			// Parse field
			if field, ok := parseField(line); ok {
				currentMessage.Fields = append(currentMessage.Fields, field)
			}
			braceDepth += openBraces - closeBraces
			continue
		}

		if inService && braceDepth == 1 {
			if closeBraces > 0 && braceDepth+openBraces-closeBraces < 1 {
				protoFile.Services = append(protoFile.Services, *currentService)
				currentService = nil
				inService = false
				braceDepth += openBraces - closeBraces
				continue
			}
			// Parse rpc method
			if method, ok := parseMethod(line); ok {
				currentService.Methods = append(currentService.Methods, method)
			}
			braceDepth += openBraces - closeBraces
			continue
		}

		if inEnum && braceDepth == 1 {
			if closeBraces > 0 && braceDepth+openBraces-closeBraces < 1 {
				protoFile.Enums = append(protoFile.Enums, *currentEnum)
				currentEnum = nil
				inEnum = false
				braceDepth += openBraces - closeBraces
				continue
			}
			// Parse enum value
			if ev, ok := parseEnumValue(line); ok {
				currentEnum.Values = append(currentEnum.Values, ev)
			}
			braceDepth += openBraces - closeBraces
			continue
		}

		if braceDepth > 1 {
			braceDepth += openBraces - closeBraces
			continue
		}

		// Parse top-level declarations
		if strings.HasPrefix(line, "syntax") {
			if val := extractQuotedValue(line); val != "" {
				protoFile.Syntax = val
			}
		} else if strings.HasPrefix(line, "package ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				protoFile.Package = strings.TrimSuffix(parts[1], ";")
			}
		} else if strings.HasPrefix(line, "import ") {
			if val := extractQuotedValue(line); val != "" {
				protoFile.Imports = append(protoFile.Imports, val)
			}
		} else if strings.HasPrefix(line, "message ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentMessage = &Message{Name: parts[1]}
				inMessage = true
				braceDepth += openBraces - closeBraces
			}
		} else if strings.HasPrefix(line, "service ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentService = &Service{Name: parts[1]}
				inService = true
				braceDepth += openBraces - closeBraces
			}
		} else if strings.HasPrefix(line, "enum ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentEnum = &Enum{Name: parts[1]}
				inEnum = true
				braceDepth += openBraces - closeBraces
			}
		} else {
			braceDepth += openBraces - closeBraces
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return protoFile, nil
}

// extractQuotedValue extracts a quoted string value from a line like: syntax = "proto3";
func extractQuotedValue(line string) string {
	start := strings.Index(line, "\"")
	if start < 0 {
		return ""
	}
	end := strings.Index(line[start+1:], "\"")
	if end < 0 {
		return ""
	}
	return line[start+1 : start+1+end]
}

// parseField parses a proto field line like: string name = 1;
func parseField(line string) (Field, bool) {
	line = strings.TrimSuffix(strings.TrimSpace(line), ";")
	if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "option") || strings.HasPrefix(line, "reserved") {
		return Field{}, false
	}

	parts := strings.Fields(line)
	if len(parts) < 4 {
		return Field{}, false
	}

	field := Field{}
	idx := 0

	if parts[0] == "repeated" {
		field.Repeated = true
		idx++
	} else if parts[0] == "optional" {
		field.Optional = true
		idx++
	}

	if idx+3 > len(parts) {
		return Field{}, false
	}

	field.Type = parts[idx]
	field.Name = parts[idx+1]
	// parts[idx+2] should be "="
	if idx+3 < len(parts) {
		num := strings.TrimSuffix(parts[idx+3], ";")
		var n int
		for _, ch := range num {
			if ch >= '0' && ch <= '9' {
				n = n*10 + int(ch-'0')
			}
		}
		field.Number = n
	}

	return field, true
}

// parseMethod parses an rpc method line like: rpc GetTest(TestMessage) returns (TestMessage);
func parseMethod(line string) (Method, bool) {
	if !strings.HasPrefix(line, "rpc ") {
		return Method{}, false
	}

	// rpc MethodName(InputType) returns (OutputType);
	line = strings.TrimPrefix(line, "rpc ")
	line = strings.TrimSuffix(line, ";")
	line = strings.TrimSuffix(line, "{")
	line = strings.TrimSpace(line)

	parenStart := strings.Index(line, "(")
	if parenStart < 0 {
		return Method{}, false
	}
	name := strings.TrimSpace(line[:parenStart])

	parenEnd := strings.Index(line, ")")
	if parenEnd < 0 {
		return Method{}, false
	}
	inputType := strings.TrimSpace(line[parenStart+1 : parenEnd])

	rest := line[parenEnd+1:]
	returnsIdx := strings.Index(rest, "returns")
	if returnsIdx < 0 {
		return Method{}, false
	}
	rest = rest[returnsIdx+7:]

	retParenStart := strings.Index(rest, "(")
	retParenEnd := strings.Index(rest, ")")
	if retParenStart < 0 || retParenEnd < 0 {
		return Method{}, false
	}
	outputType := strings.TrimSpace(rest[retParenStart+1 : retParenEnd])

	return Method{
		Name:       name,
		InputType:  inputType,
		OutputType: outputType,
	}, true
}

// parseEnumValue parses an enum value line like: UNKNOWN = 0;
func parseEnumValue(line string) (EnumValue, bool) {
	line = strings.TrimSuffix(strings.TrimSpace(line), ";")
	if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "option") || strings.HasPrefix(line, "reserved") {
		return EnumValue{}, false
	}

	parts := strings.Fields(line)
	if len(parts) < 3 || parts[1] != "=" {
		return EnumValue{}, false
	}

	name := parts[0]
	numStr := strings.TrimSuffix(parts[2], ";")
	var num int
	for _, ch := range numStr {
		if ch >= '0' && ch <= '9' {
			num = num*10 + int(ch-'0')
		}
	}

	return EnumValue{Name: name, Number: num}, true
}
