package validation

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/massonsky/buffalo/internal/plugin"
	"github.com/massonsky/buffalo/pkg/logger"
)

// ══════════════════════════════════════════════════════════════════
//  Plugin interface compliance
// ══════════════════════════════════════════════════════════════════

func TestValidatePlugin_ImplementsInterface(t *testing.T) {
	var _ plugin.Plugin = NewValidatePlugin()
}

func TestValidatePlugin_Metadata(t *testing.T) {
	p := NewValidatePlugin()

	if p.Name() != "buffalo-validate" {
		t.Errorf("expected name 'buffalo-validate', got '%s'", p.Name())
	}
	if p.Version() != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", p.Version())
	}
	if p.Type() != plugin.PluginTypeGenerator {
		t.Errorf("expected type 'generator', got '%s'", p.Type())
	}
	if p.Description() == "" {
		t.Error("expected non-empty description")
	}
}

func TestValidatePlugin_Init(t *testing.T) {
	p := NewValidatePlugin()

	cfg := DefaultValidateConfig()
	err := p.Init(cfg)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
}

func TestValidatePlugin_InitWithOptions(t *testing.T) {
	p := NewValidatePlugin()

	cfg := plugin.Config{
		Name:    "buffalo-validate",
		Enabled: true,
		Options: map[string]interface{}{
			"strict":    true,
			"languages": []interface{}{"go", "python"},
		},
	}

	err := p.Init(cfg)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	vp := p.(*ValidatePlugin)
	if !vp.strict {
		t.Error("expected strict=true after init")
	}
	if len(vp.languages) != 2 {
		t.Errorf("expected 2 languages, got %d", len(vp.languages))
	}
}

func TestValidatePlugin_Shutdown(t *testing.T) {
	p := NewValidatePlugin()
	if err := p.Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

// ══════════════════════════════════════════════════════════════════
//  Plugin execution
// ══════════════════════════════════════════════════════════════════

func TestValidatePlugin_Execute_NoAnnotations(t *testing.T) {
	p := NewValidatePlugin()
	p.Init(DefaultValidateConfig())

	// Write a proto file with no annotations
	tempDir := t.TempDir()
	protoFile := filepath.Join(tempDir, "simple.proto")
	writeTestFile(t, protoFile, `syntax = "proto3";
package test;
message Foo {
  string name = 1;
}
`)

	output, err := p.Execute(context.Background(), &plugin.Input{
		ProtoFiles: []string{protoFile},
		OutputDir:  tempDir,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !output.Success {
		t.Error("expected success=true")
	}
	if len(output.GeneratedFiles) != 0 {
		t.Errorf("expected 0 generated files, got %d", len(output.GeneratedFiles))
	}

	// Check for informational message about no annotations
	foundMsg := false
	for _, msg := range output.Messages {
		if strings.Contains(msg, "No buffalo.validate") {
			foundMsg = true
			break
		}
	}
	if !foundMsg {
		t.Error("expected a message about no annotations found")
	}
}

func TestValidatePlugin_Execute_WithAnnotations(t *testing.T) {
	p := NewValidatePlugin()
	cfg := plugin.Config{
		Name:    "buffalo-validate",
		Enabled: true,
		Options: map[string]interface{}{
			"languages": []interface{}{"go"},
		},
	}
	p.Init(cfg)

	tempDir := t.TempDir()
	protoFile := filepath.Join(tempDir, "location.proto")
	writeTestFile(t, protoFile, `syntax = "proto3";
package geo;
message Location {
  double lat = 1 [(buffalo.validate.rules).double = {gte: -90, lte: 90}];
  double lng = 2 [(buffalo.validate.rules).double = {gte: -180, lte: 180}];
}
`)

	output, err := p.Execute(context.Background(), &plugin.Input{
		ProtoFiles: []string{protoFile},
		OutputDir:  tempDir,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !output.Success {
		t.Errorf("expected success=true, errors: %v", output.Errors)
	}
	if len(output.GeneratedFiles) == 0 {
		t.Error("expected generated files, got 0")
	}

	// Verify generated file exists on disk
	for _, f := range output.GeneratedFiles {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			t.Errorf("generated file does not exist: %s", f)
		}
	}
}

func TestValidatePlugin_Execute_AllLanguages(t *testing.T) {
	p := NewValidatePlugin()
	p.Init(DefaultValidateConfig())

	tempDir := t.TempDir()
	protoFile := filepath.Join(tempDir, "user.proto")
	writeTestFile(t, protoFile, `syntax = "proto3";
package api;
message User {
  string email = 1 [(buffalo.validate.rules).string = {email: true, min_len: 5}];
  int32 age = 2 [(buffalo.validate.rules).int32 = {gt: 0, lte: 150}];
}
`)

	output, err := p.Execute(context.Background(), &plugin.Input{
		ProtoFiles: []string{protoFile},
		OutputDir:  tempDir,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !output.Success {
		t.Errorf("expected success, errors: %v", output.Errors)
	}

	// Should generate files for all 4 languages
	// go: 1 file, python: 1 file, cpp: 2 files (h+cc), rust: 1 file = 5 total
	if len(output.GeneratedFiles) < 4 {
		t.Errorf("expected at least 4 generated files, got %d: %v",
			len(output.GeneratedFiles), output.GeneratedFiles)
	}

	// Verify language directories were created
	for _, lang := range []string{"go", "python", "cpp", "rust"} {
		langDir := filepath.Join(tempDir, lang)
		if _, err := os.Stat(langDir); os.IsNotExist(err) {
			t.Errorf("language directory not created: %s", langDir)
		}
	}
}

func TestValidatePlugin_Execute_StrictModeError(t *testing.T) {
	p := NewValidatePlugin()
	cfg := plugin.Config{
		Name:    "buffalo-validate",
		Enabled: true,
		Options: map[string]interface{}{
			"strict":    true,
			"languages": []interface{}{"go"},
		},
	}
	p.Init(cfg)

	tempDir := t.TempDir()
	protoFile := filepath.Join(tempDir, "bad.proto")
	writeTestFile(t, protoFile, `syntax = "proto3";
package test;
message Bad {
  double lat = 1 [(buffalo.validate.rules).double = {gte: not_a_number}];
}
`)

	output, err := p.Execute(context.Background(), &plugin.Input{
		ProtoFiles: []string{protoFile},
		OutputDir:  tempDir,
	})
	if err != nil {
		t.Fatalf("Execute should not return error, got: %v", err)
	}
	if output.Success {
		t.Error("expected success=false in strict mode with invalid annotation")
	}
	if len(output.Errors) == 0 {
		t.Error("expected at least one error")
	}
}

func TestValidatePlugin_Execute_NonStrictModeWarning(t *testing.T) {
	p := NewValidatePlugin()
	cfg := plugin.Config{
		Name:    "buffalo-validate",
		Enabled: true,
		Options: map[string]interface{}{
			"strict":    false,
			"languages": []interface{}{"go"},
		},
	}
	p.Init(cfg)

	tempDir := t.TempDir()
	protoFile := filepath.Join(tempDir, "bad.proto")
	writeTestFile(t, protoFile, `syntax = "proto3";
package test;
message Bad {
  double lat = 1 [(buffalo.validate.rules).double = {gte: not_a_number}];
}
`)

	output, err := p.Execute(context.Background(), &plugin.Input{
		ProtoFiles: []string{protoFile},
		OutputDir:  tempDir,
	})
	if err != nil {
		t.Fatalf("Execute should not return error, got: %v", err)
	}
	if !output.Success {
		t.Error("expected success=true in non-strict mode")
	}
	if len(output.Warnings) == 0 {
		t.Error("expected at least one warning in non-strict mode")
	}
}

func TestValidatePlugin_Execute_MissingFile(t *testing.T) {
	p := NewValidatePlugin()
	p.Init(DefaultValidateConfig())

	tempDir := t.TempDir()
	output, err := p.Execute(context.Background(), &plugin.Input{
		ProtoFiles: []string{filepath.Join(tempDir, "nonexistent.proto")},
		OutputDir:  tempDir,
	})
	if err != nil {
		t.Fatalf("Execute should not return error, got: %v", err)
	}
	if !output.Success {
		t.Error("expected success=true (missing files are warnings)")
	}
	if len(output.Warnings) == 0 {
		t.Error("expected a warning about missing file")
	}
}

func TestValidatePlugin_Execute_MultipleProtoFiles(t *testing.T) {
	p := NewValidatePlugin()
	cfg := plugin.Config{
		Name:    "buffalo-validate",
		Enabled: true,
		Options: map[string]interface{}{
			"languages": []interface{}{"go"},
		},
	}
	p.Init(cfg)

	tempDir := t.TempDir()

	// Proto with annotations
	protoFile1 := filepath.Join(tempDir, "user.proto")
	writeTestFile(t, protoFile1, `syntax = "proto3";
package api;
message User {
  string email = 1 [(buffalo.validate.rules).string = {email: true}];
}
`)

	// Proto without annotations
	protoFile2 := filepath.Join(tempDir, "empty.proto")
	writeTestFile(t, protoFile2, `syntax = "proto3";
package api;
message Empty {
  string name = 1;
}
`)

	// Another proto with annotations
	protoFile3 := filepath.Join(tempDir, "location.proto")
	writeTestFile(t, protoFile3, `syntax = "proto3";
package geo;
message Location {
  double lat = 1 [(buffalo.validate.rules).double = {gte: -90, lte: 90}];
}
`)

	output, err := p.Execute(context.Background(), &plugin.Input{
		ProtoFiles: []string{protoFile1, protoFile2, protoFile3},
		OutputDir:  tempDir,
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !output.Success {
		t.Errorf("expected success, errors: %v", output.Errors)
	}
	// Should have generated files for User and Location
	if len(output.GeneratedFiles) < 2 {
		t.Errorf("expected at least 2 generated files, got %d", len(output.GeneratedFiles))
	}
}

func TestValidatePlugin_ValidationRules(t *testing.T) {
	p := NewValidatePlugin().(*ValidatePlugin)
	rules := p.ValidationRules()
	if len(rules) == 0 {
		t.Error("expected non-empty validation rules list")
	}

	// Check that core rules are listed
	ruleSet := make(map[string]bool)
	for _, r := range rules {
		ruleSet[r] = true
	}

	expected := []string{"required", "gte", "lte", "gt", "lt", "email", "uuid", "min_len", "max_len"}
	for _, exp := range expected {
		if !ruleSet[exp] {
			t.Errorf("expected rule '%s' in list", exp)
		}
	}
}

// ══════════════════════════════════════════════════════════════════
//  Plugin registration with Registry
// ══════════════════════════════════════════════════════════════════

func TestValidatePlugin_RegistryIntegration(t *testing.T) {
	// This test verifies the plugin works with the real Registry
	log := newTestLogger()
	registry := plugin.NewRegistry(log)

	p := NewValidatePlugin()
	cfg := DefaultValidateConfig()

	err := registry.Register(p, cfg)
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	err = registry.InitAll()
	if err != nil {
		t.Fatalf("failed to init: %v", err)
	}

	registered, err := registry.Get("buffalo-validate")
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}

	if registered.Plugin.Name() != "buffalo-validate" {
		t.Errorf("expected name 'buffalo-validate', got '%s'", registered.Plugin.Name())
	}

	err = registry.ShutdownAll()
	if err != nil {
		t.Fatalf("failed to shutdown: %v", err)
	}
}

func TestValidatePlugin_ExecuteHookIntegration(t *testing.T) {
	log := newTestLogger()
	registry := plugin.NewRegistry(log)

	p := NewValidatePlugin()
	cfg := plugin.Config{
		Name:    "buffalo-validate",
		Enabled: true,
		HookPoints: []plugin.HookPoint{
			plugin.HookPointPostParse,
		},
		Priority: 90,
		Options: map[string]interface{}{
			"languages": []interface{}{"go"},
		},
	}

	registry.Register(p, cfg)
	registry.InitAll()
	defer registry.ShutdownAll()

	tempDir := t.TempDir()
	protoFile := filepath.Join(tempDir, "test.proto")
	writeTestFile(t, protoFile, `syntax = "proto3";
package test;
message Test {
  double val = 1 [(buffalo.validate.rules).double = {gte: 0, lte: 100}];
}
`)

	input := &plugin.Input{
		ProtoFiles: []string{protoFile},
		OutputDir:  tempDir,
	}

	err := registry.ExecuteHook(context.Background(), plugin.HookPointPostParse, input)
	if err != nil {
		t.Fatalf("ExecuteHook failed: %v", err)
	}

	// Generated files should have been added to input
	if len(input.GeneratedFiles) == 0 {
		t.Error("expected generated files in input after hook execution")
	}
}

// ══════════════════════════════════════════════════════════════════
//  Helpers
// ══════════════════════════════════════════════════════════════════

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}

// newTestLogger creates a logger for tests using the real logger package.
func newTestLogger() *logger.Logger {
	return logger.New()
}
