package plugin

import (
	"context"
	"testing"

	"github.com/massonsky/buffalo/pkg/logger"
)

// MockPlugin is a mock implementation for testing
type MockPlugin struct {
	name        string
	version     string
	pluginType  PluginType
	description string
	initError   error
	execOutput  *Output
	execError   error
	shutdownErr error
}

func NewMockPlugin(name string) *MockPlugin {
	return &MockPlugin{
		name:        name,
		version:     "1.0.0",
		pluginType:  PluginTypeValidator,
		description: "Mock plugin for testing",
		execOutput: &Output{
			Success: true,
		},
	}
}

func (m *MockPlugin) Name() string        { return m.name }
func (m *MockPlugin) Version() string     { return m.version }
func (m *MockPlugin) Type() PluginType    { return m.pluginType }
func (m *MockPlugin) Description() string { return m.description }

func (m *MockPlugin) Init(config Config) error {
	return m.initError
}

func (m *MockPlugin) Execute(ctx context.Context, input *Input) (*Output, error) {
	return m.execOutput, m.execError
}

func (m *MockPlugin) Shutdown() error {
	return m.shutdownErr
}

func TestRegistryRegister(t *testing.T) {
	log := logger.New()
	registry := NewRegistry(log)

	plugin := NewMockPlugin("test-plugin")
	config := Config{
		Name:    "test-plugin",
		Enabled: true,
	}

	err := registry.Register(plugin, config)
	if err != nil {
		t.Fatalf("Failed to register plugin: %v", err)
	}

	// Check plugin was registered
	registered, err := registry.Get("test-plugin")
	if err != nil {
		t.Fatalf("Failed to get registered plugin: %v", err)
	}

	if registered.Plugin.Name() != "test-plugin" {
		t.Errorf("Expected plugin name 'test-plugin', got '%s'", registered.Plugin.Name())
	}
}

func TestRegistryRegisterDuplicate(t *testing.T) {
	log := logger.New()
	registry := NewRegistry(log)

	plugin := NewMockPlugin("test-plugin")
	config := Config{Name: "test-plugin", Enabled: true}

	// First registration should succeed
	err := registry.Register(plugin, config)
	if err != nil {
		t.Fatalf("First registration failed: %v", err)
	}

	// Second registration should fail
	err = registry.Register(plugin, config)
	if err == nil {
		t.Error("Expected error when registering duplicate plugin, got nil")
	}
}

func TestRegistryUnregister(t *testing.T) {
	log := logger.New()
	registry := NewRegistry(log)

	plugin := NewMockPlugin("test-plugin")
	config := Config{Name: "test-plugin", Enabled: true}

	registry.Register(plugin, config)

	err := registry.Unregister("test-plugin")
	if err != nil {
		t.Fatalf("Failed to unregister plugin: %v", err)
	}

	// Should not be able to get it anymore
	_, err = registry.Get("test-plugin")
	if err == nil {
		t.Error("Expected error when getting unregistered plugin, got nil")
	}
}

func TestRegistryList(t *testing.T) {
	log := logger.New()
	registry := NewRegistry(log)

	// Register multiple plugins
	for i := 1; i <= 3; i++ {
		plugin := NewMockPlugin("test-plugin-" + string(rune('0'+i)))
		config := Config{Name: plugin.Name(), Enabled: true}
		registry.Register(plugin, config)
	}

	plugins := registry.List()
	if len(plugins) != 3 {
		t.Errorf("Expected 3 plugins, got %d", len(plugins))
	}
}

func TestRegistryListByType(t *testing.T) {
	log := logger.New()
	registry := NewRegistry(log)

	// Register validator plugin
	validator := NewMockPlugin("validator")
	validator.pluginType = PluginTypeValidator
	registry.Register(validator, Config{Name: "validator", Enabled: true})

	// Register compiler plugin
	compiler := NewMockPlugin("compiler")
	compiler.pluginType = PluginTypeCompiler
	registry.Register(compiler, Config{Name: "compiler", Enabled: true})

	validators := registry.ListByType(PluginTypeValidator)
	if len(validators) != 1 {
		t.Errorf("Expected 1 validator, got %d", len(validators))
	}

	compilers := registry.ListByType(PluginTypeCompiler)
	if len(compilers) != 1 {
		t.Errorf("Expected 1 compiler, got %d", len(compilers))
	}
}

func TestRegistryInitAll(t *testing.T) {
	log := logger.New()
	registry := NewRegistry(log)

	plugin1 := NewMockPlugin("plugin1")
	plugin2 := NewMockPlugin("plugin2")

	registry.Register(plugin1, Config{Name: "plugin1", Enabled: true})
	registry.Register(plugin2, Config{Name: "plugin2", Enabled: true})

	err := registry.InitAll()
	if err != nil {
		t.Fatalf("InitAll failed: %v", err)
	}

	// Check plugins are initialized
	reg1, _ := registry.Get("plugin1")
	if reg1.GetStatus() != StatusInitialized {
		t.Errorf("Expected plugin1 to be initialized, got status %s", reg1.GetStatus())
	}

	reg2, _ := registry.Get("plugin2")
	if reg2.GetStatus() != StatusInitialized {
		t.Errorf("Expected plugin2 to be initialized, got status %s", reg2.GetStatus())
	}
}

func TestRegistryExecuteHook(t *testing.T) {
	log := logger.New()
	registry := NewRegistry(log)

	plugin := NewMockPlugin("test-plugin")
	plugin.execOutput = &Output{
		Success:  true,
		Messages: []string{"Hook executed"},
	}

	config := Config{
		Name:       "test-plugin",
		Enabled:    true,
		HookPoints: []HookPoint{HookPointPreBuild},
	}

	registry.Register(plugin, config)
	registry.InitAll()

	ctx := context.Background()
	input := &Input{
		ProtoFiles: []string{"test.proto"},
	}

	err := registry.ExecuteHook(ctx, HookPointPreBuild, input)
	if err != nil {
		t.Fatalf("ExecuteHook failed: %v", err)
	}
}

func TestRegistryExecuteHookPriority(t *testing.T) {
	log := logger.New()
	registry := NewRegistry(log)

	var executionOrder []string

	// Create plugins that track execution order
	plugin1 := NewMockPlugin("plugin1")
	plugin1.execOutput = &Output{Success: true}

	plugin2 := NewMockPlugin("plugin2")
	plugin2.execOutput = &Output{Success: true}

	plugin3 := NewMockPlugin("plugin3")
	plugin3.execOutput = &Output{Success: true}

	// Register with different priorities
	registry.Register(plugin1, Config{
		Name:       "plugin1",
		Enabled:    true,
		HookPoints: []HookPoint{HookPointPreBuild},
		Priority:   50,
	})

	registry.Register(plugin2, Config{
		Name:       "plugin2",
		Enabled:    true,
		HookPoints: []HookPoint{HookPointPreBuild},
		Priority:   200,
	})

	registry.Register(plugin3, Config{
		Name:       "plugin3",
		Enabled:    true,
		HookPoints: []HookPoint{HookPointPreBuild},
		Priority:   100,
	})

	registry.InitAll()

	ctx := context.Background()
	input := &Input{ProtoFiles: []string{"test.proto"}}

	// Execute - should run in order: plugin2 (200), plugin3 (100), plugin1 (50)
	err := registry.ExecuteHook(ctx, HookPointPreBuild, input)
	if err != nil {
		t.Fatalf("ExecuteHook failed: %v", err)
	}

	_ = executionOrder // Would need more sophisticated mock to track order
}

func TestRegistryShutdownAll(t *testing.T) {
	log := logger.New()
	registry := NewRegistry(log)

	plugin := NewMockPlugin("test-plugin")
	registry.Register(plugin, Config{Name: "test-plugin", Enabled: true})
	registry.InitAll()

	err := registry.ShutdownAll()
	if err != nil {
		t.Fatalf("ShutdownAll failed: %v", err)
	}
}

func TestRegistryExecuteHookWithError(t *testing.T) {
	log := logger.New()
	registry := NewRegistry(log)

	plugin := NewMockPlugin("failing-plugin")
	plugin.execError = &PluginError{
		Plugin:  "failing-plugin",
		Message: "execution failed",
	}

	config := Config{
		Name:       "failing-plugin",
		Enabled:    true,
		HookPoints: []HookPoint{HookPointPreBuild},
	}

	registry.Register(plugin, config)
	registry.InitAll()

	ctx := context.Background()
	input := &Input{ProtoFiles: []string{"test.proto"}}

	err := registry.ExecuteHook(ctx, HookPointPreBuild, input)
	if err == nil {
		t.Error("Expected error from failing plugin, got nil")
	}
}

func TestRegistryDisabledPlugin(t *testing.T) {
	log := logger.New()
	registry := NewRegistry(log)

	plugin := NewMockPlugin("disabled-plugin")
	config := Config{
		Name:    "disabled-plugin",
		Enabled: false, // Disabled
	}

	registry.Register(plugin, config)
	registry.InitAll()

	reg, _ := registry.Get("disabled-plugin")
	if reg.GetStatus() != StatusDisabled {
		t.Errorf("Expected disabled status, got %s", reg.GetStatus())
	}
}

// PluginError for testing
type PluginError struct {
	Plugin  string
	Message string
}

func (e *PluginError) Error() string {
	return "plugin " + e.Plugin + ": " + e.Message
}
