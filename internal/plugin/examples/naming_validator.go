// Package examples provides example plugins for Buffalo
// This demonstrates how to create a simple validator plugin
package examples

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/massonsky/buffalo/internal/plugin"
)

// NamingValidator validates proto file naming conventions
type NamingValidator struct {
	config     plugin.Config
	strictMode bool
}

// New creates a new instance of the naming validator plugin
// This function is required and will be called by the plugin loader
func New() plugin.Plugin {
	return &NamingValidator{}
}

// DefaultConfig returns the default configuration for this plugin
func DefaultConfig() plugin.Config {
	return plugin.Config{
		Name:    "naming-validator",
		Enabled: true,
		HookPoints: []plugin.HookPoint{
			plugin.HookPointPreBuild,
		},
		Priority: 100,
		Options: map[string]interface{}{
			"strict_mode": false,
		},
	}
}

// Name returns the plugin name
func (v *NamingValidator) Name() string {
	return "naming-validator"
}

// Version returns the plugin version
func (v *NamingValidator) Version() string {
	return "1.0.0"
}

// Type returns the plugin type
func (v *NamingValidator) Type() plugin.PluginType {
	return plugin.PluginTypeValidator
}

// Description returns a description of what this plugin does
func (v *NamingValidator) Description() string {
	return "Validates proto file naming conventions (snake_case, no spaces, etc)"
}

// Init initializes the plugin with configuration
func (v *NamingValidator) Init(config plugin.Config) error {
	v.config = config

	// Extract strict mode from options
	if strictMode, ok := config.Options["strict_mode"].(bool); ok {
		v.strictMode = strictMode
	}

	return nil
}

// Execute runs the validation logic
func (v *NamingValidator) Execute(ctx context.Context, input *plugin.Input) (*plugin.Output, error) {
	output := &plugin.Output{
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

// Shutdown performs cleanup
func (v *NamingValidator) Shutdown() error {
	// Nothing to clean up for this simple plugin
	return nil
}

// ValidationRules returns the rules this validator checks
func (v *NamingValidator) ValidationRules() []string {
	return []string{
		"proto_extension: Files must have .proto extension",
		"snake_case: Files should use snake_case naming",
		"no_spaces: Files must not contain spaces",
		"lowercase: Files must be lowercase (strict mode only)",
	}
}

// isSnakeCase checks if a string is in snake_case format
func isSnakeCase(s string) bool {
	// Valid snake_case: lowercase letters, numbers, and underscores
	// Must start with a letter
	if len(s) == 0 || !isLowerLetter(rune(s[0])) {
		return false
	}

	for _, c := range s {
		if !isLowerLetter(c) && !isDigit(c) && c != '_' {
			return false
		}
	}

	// No consecutive underscores
	if strings.Contains(s, "__") {
		return false
	}

	// No leading or trailing underscores
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

// Compile-time check that NamingValidator implements required interfaces
var _ plugin.Plugin = (*NamingValidator)(nil)
var _ plugin.ValidatorPlugin = (*NamingValidator)(nil)
