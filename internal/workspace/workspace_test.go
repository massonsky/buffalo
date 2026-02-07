package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "buffalo-workspace-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test config
	configContent := `
workspace:
  name: test-platform
  version: "1.0.0"

projects:
  - name: user-service
    path: ./services/user-service
    tags: [backend, core]
    
  - name: order-service
    path: ./services/order-service
    depends_on: [user-service]
    tags: [backend]

  - name: common-protos
    path: ./shared/common-protos
    tags: [shared]

policies:
  consistent_versions: true
  shared_dependencies: true
  no_circular_deps: true
`
	configPath := filepath.Join(tmpDir, "buffalo-workspace.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Load config
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Workspace.Name != "test-platform" {
		t.Errorf("Expected workspace name 'test-platform', got '%s'", cfg.Workspace.Name)
	}

	if len(cfg.Projects) != 3 {
		t.Errorf("Expected 3 projects, got %d", len(cfg.Projects))
	}

	if !cfg.Policies.NoCircularDeps {
		t.Error("Expected NoCircularDeps to be true")
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		isValid  bool
		errCount int
	}{
		{
			name: "valid config",
			config: Config{
				Workspace: WorkspaceSettings{Name: "test"},
				Projects: []Project{
					{Name: "p1", Path: "./p1"},
					{Name: "p2", Path: "./p2", DependsOn: []string{"p1"}},
				},
			},
			isValid:  true,
			errCount: 0,
		},
		{
			name: "missing workspace name",
			config: Config{
				Projects: []Project{
					{Name: "p1", Path: "./p1"},
				},
			},
			isValid:  false,
			errCount: 1,
		},
		{
			name: "duplicate project names",
			config: Config{
				Workspace: WorkspaceSettings{Name: "test"},
				Projects: []Project{
					{Name: "p1", Path: "./p1"},
					{Name: "p1", Path: "./p1-dup"},
				},
			},
			isValid:  false,
			errCount: 1,
		},
		{
			name: "missing project path",
			config: Config{
				Workspace: WorkspaceSettings{Name: "test"},
				Projects: []Project{
					{Name: "p1"},
				},
			},
			isValid:  false,
			errCount: 1,
		},
		{
			name: "unknown dependency",
			config: Config{
				Workspace: WorkspaceSettings{Name: "test"},
				Projects: []Project{
					{Name: "p1", Path: "./p1", DependsOn: []string{"unknown"}},
				},
			},
			isValid:  false,
			errCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.Validate()

			if result.Valid != tt.isValid {
				t.Errorf("Expected Valid=%v, got %v", tt.isValid, result.Valid)
			}

			if len(result.Errors) != tt.errCount {
				t.Errorf("Expected %d errors, got %d: %+v", tt.errCount, len(result.Errors), result.Errors)
			}
		})
	}
}

func TestConfig_DetectCycles(t *testing.T) {
	tests := []struct {
		name       string
		projects   []Project
		hasCycles  bool
		cycleCount int
	}{
		{
			name: "no cycles",
			projects: []Project{
				{Name: "a", Path: "./a", DependsOn: []string{"b"}},
				{Name: "b", Path: "./b", DependsOn: []string{"c"}},
				{Name: "c", Path: "./c"},
			},
			hasCycles:  false,
			cycleCount: 0,
		},
		{
			name: "simple cycle",
			projects: []Project{
				{Name: "a", Path: "./a", DependsOn: []string{"b"}},
				{Name: "b", Path: "./b", DependsOn: []string{"a"}},
			},
			hasCycles:  true,
			cycleCount: 1,
		},
		{
			name: "self cycle",
			projects: []Project{
				{Name: "a", Path: "./a", DependsOn: []string{"a"}},
			},
			hasCycles:  true,
			cycleCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Workspace: WorkspaceSettings{Name: "test"},
				Projects:  tt.projects,
			}

			cycles := cfg.DetectCycles()

			if tt.hasCycles && len(cycles) == 0 {
				t.Error("Expected cycles but found none")
			}

			if !tt.hasCycles && len(cycles) > 0 {
				t.Errorf("Expected no cycles but found: %v", cycles)
			}
		})
	}
}

func TestConfig_BuildDependencyGraph(t *testing.T) {
	cfg := &Config{
		Workspace: WorkspaceSettings{Name: "test"},
		Projects: []Project{
			{Name: "a", Path: "./a", DependsOn: []string{"b", "c"}},
			{Name: "b", Path: "./b", DependsOn: []string{"c"}},
			{Name: "c", Path: "./c"},
		},
	}

	graph := cfg.BuildDependencyGraph()

	// Check forward dependencies
	if len(graph.Projects["a"]) != 2 {
		t.Errorf("Expected 'a' to have 2 dependencies, got %d", len(graph.Projects["a"]))
	}

	// Check reverse dependencies
	if len(graph.Reverse["c"]) != 2 {
		t.Errorf("Expected 'c' to have 2 dependents, got %d", len(graph.Reverse["c"]))
	}
}

func TestConfig_GetProjectsByTag(t *testing.T) {
	cfg := &Config{
		Workspace: WorkspaceSettings{Name: "test"},
		Projects: []Project{
			{Name: "a", Path: "./a", Tags: []string{"backend", "core"}},
			{Name: "b", Path: "./b", Tags: []string{"backend"}},
			{Name: "c", Path: "./c", Tags: []string{"frontend"}},
		},
	}

	backend := cfg.GetProjectsByTag("backend")
	if len(backend) != 2 {
		t.Errorf("Expected 2 backend projects, got %d", len(backend))
	}

	frontend := cfg.GetProjectsByTag("frontend")
	if len(frontend) != 1 {
		t.Errorf("Expected 1 frontend project, got %d", len(frontend))
	}

	nonexistent := cfg.GetProjectsByTag("nonexistent")
	if len(nonexistent) != 0 {
		t.Errorf("Expected 0 projects for nonexistent tag, got %d", len(nonexistent))
	}
}

func TestProject_IsEnabled(t *testing.T) {
	// Default (nil) should be enabled
	p1 := Project{Name: "p1", Path: "./p1"}
	if !p1.IsEnabled() {
		t.Error("Project with nil Enabled should be enabled")
	}

	// Explicit true
	enabled := true
	p2 := Project{Name: "p2", Path: "./p2", Enabled: &enabled}
	if !p2.IsEnabled() {
		t.Error("Project with Enabled=true should be enabled")
	}

	// Explicit false
	disabled := false
	p3 := Project{Name: "p3", Path: "./p3", Enabled: &disabled}
	if p3.IsEnabled() {
		t.Error("Project with Enabled=false should be disabled")
	}
}

func TestInitConfig(t *testing.T) {
	cfg := InitConfig("my-workspace")

	if cfg.Workspace.Name != "my-workspace" {
		t.Errorf("Expected workspace name 'my-workspace', got '%s'", cfg.Workspace.Name)
	}

	if !cfg.Policies.NoCircularDeps {
		t.Error("Expected NoCircularDeps to be true by default")
	}

	if len(cfg.Projects) != 0 {
		t.Error("Expected empty projects list")
	}
}

func TestFindConfig(t *testing.T) {
	// Create temp directory structure
	tmpDir, err := os.MkdirTemp("", "buffalo-find-config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "sub", "dir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create config in root
	configPath := filepath.Join(tmpDir, "buffalo-workspace.yaml")
	if err := os.WriteFile(configPath, []byte("workspace:\n  name: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Find from subdirectory should find parent config
	found, err := FindConfig(subDir)
	if err != nil {
		t.Fatalf("FindConfig failed: %v", err)
	}

	if found != configPath {
		t.Errorf("Expected to find '%s', got '%s'", configPath, found)
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "buffalo-save-config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &Config{
		Workspace: WorkspaceSettings{Name: "test", Version: "1.0.0"},
		Projects: []Project{
			{Name: "p1", Path: "./p1", Tags: []string{"backend"}},
		},
		Policies: Policies{NoCircularDeps: true},
	}

	configPath := filepath.Join(tmpDir, "buffalo-workspace.yaml")
	if err := SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Reload and verify
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if loaded.Workspace.Name != cfg.Workspace.Name {
		t.Errorf("Name mismatch: got '%s', want '%s'", loaded.Workspace.Name, cfg.Workspace.Name)
	}

	if len(loaded.Projects) != len(cfg.Projects) {
		t.Errorf("Projects count mismatch: got %d, want %d", len(loaded.Projects), len(cfg.Projects))
	}
}
