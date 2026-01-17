package plugin

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// SimpleNamingValidator is a built-in validator for testing
// It validates proto file naming without requiring external plugins
type SimpleNamingValidator struct {
	config     Config
	strictMode bool
}

// NewSimpleNamingValidator creates a new built-in naming validator
func NewSimpleNamingValidator() Plugin {
	return &SimpleNamingValidator{}
}

func (v *SimpleNamingValidator) Name() string {
	return "naming-validator"
}

func (v *SimpleNamingValidator) Version() string {
	return "1.0.0"
}

func (v *SimpleNamingValidator) Type() PluginType {
	return PluginTypeValidator
}

func (v *SimpleNamingValidator) Description() string {
	return "Validates proto file naming conventions (snake_case, no spaces, etc)"
}

func (v *SimpleNamingValidator) Init(config Config) error {
	v.config = config

	if strictMode, ok := config.Options["strict_mode"].(bool); ok {
		v.strictMode = strictMode
	}

	return nil
}

func (v *SimpleNamingValidator) Execute(ctx context.Context, input *Input) (*Output, error) {
	output := &Output{
		Success:  true,
		Messages: []string{},
		Warnings: []string{},
		Errors:   []string{},
	}

	output.Messages = append(output.Messages, fmt.Sprintf("Validating %d proto files", len(input.ProtoFiles)))

	for _, protoFile := range input.ProtoFiles {
		fileName := filepath.Base(protoFile)

		// Rule 1: Must end with .proto
		if !strings.HasSuffix(fileName, ".proto") {
			output.Errors = append(output.Errors,
				fmt.Sprintf("File %s does not have .proto extension", fileName))
			output.Success = false
			continue
		}

		baseName := strings.TrimSuffix(fileName, ".proto")

		// Rule 2: Must be snake_case (lowercase with underscores)
		if !isSnakeCase(baseName) {
			msg := fmt.Sprintf("File %s is not in snake_case format", fileName)
			if v.strictMode {
				output.Errors = append(output.Errors, msg)
				output.Success = false
			} else {
				output.Warnings = append(output.Warnings, msg)
			}
		}

		// Rule 3: No spaces
		if strings.Contains(fileName, " ") {
			output.Errors = append(output.Errors,
				fmt.Sprintf("File %s contains spaces", fileName))
			output.Success = false
		}

		// Rule 4: No uppercase in strict mode
		if v.strictMode && baseName != strings.ToLower(baseName) {
			output.Errors = append(output.Errors,
				fmt.Sprintf("File %s contains uppercase letters (strict mode)", fileName))
			output.Success = false
		}
	}

	if output.Success {
		output.Messages = append(output.Messages, "✅ All proto files pass naming validation")
	} else {
		output.Messages = append(output.Messages, "❌ Some proto files have naming violations")
	}

	return output, nil
}

func (v *SimpleNamingValidator) Shutdown() error {
	return nil
}

// isSnakeCase checks if a string is in snake_case format
func isSnakeCase(s string) bool {
	if len(s) == 0 || !isLowerLetter(rune(s[0])) {
		return false
	}

	for _, c := range s {
		if !isLowerLetter(c) && !isDigit(c) && c != '_' {
			return false
		}
	}

	if strings.Contains(s, "__") {
		return false
	}

	if strings.HasPrefix(s, "_") || strings.HasSuffix(s, "_") {
		return false
	}

	return true
}

func isLowerLetter(c rune) bool {
	return c >= 'a' && c <= 'z'
}

func isDigit(c rune) bool {
	return c >= '0' && c <= '9'
}
