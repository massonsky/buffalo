package main

import (
	"context"
	"testing"

	"github.com/massonsky/buffalo/internal/plugin"
)

func TestNamingValidatorName(t *testing.T) {
	v := &NamingValidator{}
	if v.Name() != "naming-validator" {
		t.Errorf("Expected name 'naming-validator', got '%s'", v.Name())
	}
}

func TestNamingValidatorType(t *testing.T) {
	v := &NamingValidator{}
	if v.Type() != plugin.PluginTypeValidator {
		t.Errorf("Expected type PluginTypeValidator, got %s", v.Type())
	}
}

func TestNamingValidatorInit(t *testing.T) {
	v := &NamingValidator{}
	config := plugin.Config{
		Options: map[string]interface{}{
			"strict_mode": true,
		},
	}

	err := v.Init(config)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if !v.strictMode {
		t.Error("Expected strict_mode to be true")
	}
}

func TestNamingValidatorExecuteValidFiles(t *testing.T) {
	v := &NamingValidator{}
	v.Init(plugin.Config{
		Options: map[string]interface{}{
			"strict_mode": false,
		},
	})

	ctx := context.Background()
	input := &plugin.Input{
		ProtoFiles: []string{
			"user_service.proto",
			"api_v1.proto",
			"message_types.proto",
		},
	}

	output, err := v.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !output.Success {
		t.Error("Expected validation to succeed for valid files")
	}

	if len(output.Errors) > 0 {
		t.Errorf("Expected no errors, got %d: %v", len(output.Errors), output.Errors)
	}
}

func TestNamingValidatorExecuteInvalidExtension(t *testing.T) {
	v := &NamingValidator{}
	v.Init(plugin.Config{})

	ctx := context.Background()
	input := &plugin.Input{
		ProtoFiles: []string{
			"user_service.txt", // Wrong extension
		},
	}

	output, err := v.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if output.Success {
		t.Error("Expected validation to fail for invalid extension")
	}

	if len(output.Errors) == 0 {
		t.Error("Expected error for invalid extension")
	}
}

func TestNamingValidatorExecuteWithSpaces(t *testing.T) {
	v := &NamingValidator{}
	v.Init(plugin.Config{})

	ctx := context.Background()
	input := &plugin.Input{
		ProtoFiles: []string{
			"user service.proto", // Contains spaces
		},
	}

	output, err := v.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if output.Success {
		t.Error("Expected validation to fail for filename with spaces")
	}

	if len(output.Errors) == 0 {
		t.Error("Expected error for filename with spaces")
	}
}

func TestNamingValidatorExecuteNotSnakeCase(t *testing.T) {
	v := &NamingValidator{}
	v.Init(plugin.Config{
		Options: map[string]interface{}{
			"strict_mode": false,
		},
	})

	ctx := context.Background()
	input := &plugin.Input{
		ProtoFiles: []string{
			"UserService.proto", // PascalCase, not snake_case
		},
	}

	output, err := v.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// In non-strict mode, should warn but not fail
	if !output.Success {
		t.Error("Expected validation to succeed in non-strict mode")
	}

	if len(output.Warnings) == 0 {
		t.Error("Expected warning for non-snake_case filename")
	}
}

func TestNamingValidatorExecuteStrictMode(t *testing.T) {
	v := &NamingValidator{}
	v.Init(plugin.Config{
		Options: map[string]interface{}{
			"strict_mode": true,
		},
	})

	ctx := context.Background()
	input := &plugin.Input{
		ProtoFiles: []string{
			"UserService.proto", // Contains uppercase
		},
	}

	output, err := v.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// In strict mode, should fail
	if output.Success {
		t.Error("Expected validation to fail in strict mode for uppercase")
	}

	if len(output.Errors) == 0 {
		t.Error("Expected error in strict mode for uppercase")
	}
}

func TestIsSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"user_service", true},
		{"api_v1", true},
		{"message_types", true},
		{"simple", true},
		{"with_numbers_123", true},
		{"UserService", false},   // PascalCase
		{"user-service", false},  // kebab-case
		{"user__service", false}, // double underscore
		{"_user_service", false}, // leading underscore
		{"user_service_", false}, // trailing underscore
		{"", false},              // empty
		{"123_start", false},     // starts with number
		{"user service", false},  // space
		{"user.service", false},  // dot
		{"User_Service", false},  // mixed case
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("isSnakeCase(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNamingValidatorShutdown(t *testing.T) {
	v := &NamingValidator{}
	err := v.Shutdown()
	if err != nil {
		t.Errorf("Shutdown should not return error, got: %v", err)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Name != "naming-validator" {
		t.Errorf("Expected name 'naming-validator', got '%s'", config.Name)
	}

	if !config.Enabled {
		t.Error("Expected plugin to be enabled by default")
	}

	if len(config.HookPoints) == 0 {
		t.Error("Expected at least one hook point")
	}

	if config.Priority != 100 {
		t.Errorf("Expected priority 100, got %d", config.Priority)
	}
}

func TestValidationRules(t *testing.T) {
	v := &NamingValidator{}
	rules := v.ValidationRules()

	if len(rules) == 0 {
		t.Error("Expected at least one validation rule")
	}

	// Check that rules contain expected keywords
	hasProtoExtension := false
	hasSnakeCase := false

	for _, rule := range rules {
		if containsString(rule, "proto_extension") || containsString(rule, ".proto") {
			hasProtoExtension = true
		}
		if containsString(rule, "snake_case") {
			hasSnakeCase = true
		}
	}

	if !hasProtoExtension {
		t.Error("Expected rule about .proto extension")
	}

	if !hasSnakeCase {
		t.Error("Expected rule about snake_case")
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
