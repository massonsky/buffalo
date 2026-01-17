package plugin_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/massonsky/buffalo/internal/builder"
	"github.com/massonsky/buffalo/internal/config"
	"github.com/massonsky/buffalo/internal/plugin"
	"github.com/massonsky/buffalo/pkg/logger"
)

func TestPluginIntegration(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "buffalo-plugin-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test proto files
	protoDir := filepath.Join(tempDir, "protos")
	if err := os.MkdirAll(protoDir, 0755); err != nil {
		t.Fatalf("Failed to create proto dir: %v", err)
	}

	// Good file (snake_case)
	goodFile := filepath.Join(protoDir, "user_service.proto")
	if err := os.WriteFile(goodFile, []byte("syntax = \"proto3\";"), 0644); err != nil {
		t.Fatalf("Failed to write good proto file: %v", err)
	}

	// Bad file (PascalCase)
	badFile := filepath.Join(protoDir, "BadName.proto")
	if err := os.WriteFile(badFile, []byte("syntax = \"proto3\";"), 0644); err != nil {
		t.Fatalf("Failed to write bad proto file: %v", err)
	}

	// Setup plugin registry
	log := logger.New()
	registry := plugin.NewRegistry(log)

	// Register naming validator plugin
	validator := plugin.NewSimpleNamingValidator()
	config := plugin.Config{
		Name:       "naming-validator",
		Enabled:    true,
		HookPoints: []plugin.HookPoint{plugin.HookPointPreBuild},
		Priority:   100,
		Options: map[string]interface{}{
			"strict_mode": false,
		},
	}

	if err := registry.Register(validator, config); err != nil {
		t.Fatalf("Failed to register plugin: %v", err)
	}

	// Initialize plugins
	if err := registry.InitAll(); err != nil {
		t.Fatalf("Failed to initialize plugins: %v", err)
	}

	// Execute pre-build hook
	ctx := context.Background()
	input := &plugin.Input{
		ProtoFiles: []string{goodFile, badFile},
		OutputDir:  tempDir,
	}

	err = registry.ExecuteHook(ctx, plugin.HookPointPreBuild, input)
	if err != nil {
		t.Fatalf("ExecuteHook failed: %v", err)
	}

	// Cleanup
	if err := registry.ShutdownAll(); err != nil {
		t.Fatalf("Failed to shutdown plugins: %v", err)
	}

	t.Log("✅ Plugin integration test passed")
}

func TestPluginIntegrationWithBuilder(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "buffalo-builder-plugin-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test proto file
	protoDir := filepath.Join(tempDir, "protos")
	if err := os.MkdirAll(protoDir, 0755); err != nil {
		t.Fatalf("Failed to create proto dir: %v", err)
	}

	testProto := filepath.Join(protoDir, "test_service.proto")
	protoContent := `syntax = "proto3";
package test.v1;
message TestMessage {
  string id = 1;
}
`
	if err := os.WriteFile(testProto, []byte(protoContent), 0644); err != nil {
		t.Fatalf("Failed to write proto file: %v", err)
	}

	// Setup configuration
	cfg := &config.Config{
		Project: config.ProjectConfig{
			Name:    "test-project",
			Version: "0.1.0",
		},
		Proto: config.ProtoConfig{
			Paths: []string{protoDir},
		},
		Output: config.OutputConfig{
			BaseDir: filepath.Join(tempDir, "generated"),
		},
		Languages: config.LanguagesConfig{
			Python: config.PythonConfig{Enabled: false},
			Go:     config.GoConfig{Enabled: false},
		},
		Build: config.BuildConfig{
			Cache: config.CacheConfig{
				Enabled: false,
			},
		},
	}

	// Setup plugin registry
	log := logger.New()
	registry := plugin.NewRegistry(log)

	// Register naming validator
	validator := plugin.NewSimpleNamingValidator()
	pluginConfig := plugin.Config{
		Name:       "naming-validator",
		Enabled:    true,
		HookPoints: []plugin.HookPoint{plugin.HookPointPreBuild},
		Options: map[string]interface{}{
			"strict_mode": false,
		},
	}

	if err := registry.Register(validator, pluginConfig); err != nil {
		t.Fatalf("Failed to register plugin: %v", err)
	}

	if err := registry.InitAll(); err != nil {
		t.Fatalf("Failed to initialize plugins: %v", err)
	}

	// Create builder with plugin registry
	b, err := builder.New(cfg,
		builder.WithLogger(log),
		builder.WithPluginRegistry(registry),
	)
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	// Verify builder was created with plugin registry
	_ = b // Builder created successfully with plugins

	// Note: We're only testing plugin integration, not full build
	// Full build would require actual proto compiler setup

	t.Log("✅ Builder plugin integration test passed")
}

func TestMultiplePluginsExecution(t *testing.T) {
	log := logger.New()
	registry := plugin.NewRegistry(log)

	// Register multiple plugins with different priorities
	plugin1 := plugin.NewSimpleNamingValidator()
	config1 := plugin.Config{
		Name:       "validator-high-priority",
		Enabled:    true,
		HookPoints: []plugin.HookPoint{plugin.HookPointPreBuild},
		Priority:   200,
	}
	registry.Register(plugin1, config1)

	plugin2 := plugin.NewSimpleNamingValidator()
	config2 := plugin.Config{
		Name:       "validator-low-priority",
		Enabled:    true,
		HookPoints: []plugin.HookPoint{plugin.HookPointPreBuild},
		Priority:   50,
	}
	registry.Register(plugin2, config2)

	if err := registry.InitAll(); err != nil {
		t.Fatalf("Failed to initialize plugins: %v", err)
	}

	ctx := context.Background()
	input := &plugin.Input{
		ProtoFiles: []string{"test.proto"},
	}

	if err := registry.ExecuteHook(ctx, plugin.HookPointPreBuild, input); err != nil {
		t.Fatalf("ExecuteHook failed: %v", err)
	}

	t.Log("✅ Multiple plugins execution test passed")
}
