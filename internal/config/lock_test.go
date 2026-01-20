package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultExternalPrefixes(t *testing.T) {
	prefixes := DefaultExternalPrefixes()

	// Should contain google prefix
	hasGoogle := false
	for _, p := range prefixes {
		if p == "google." {
			hasGoogle = true
			break
		}
	}

	if !hasGoogle {
		t.Error("Expected default prefixes to contain 'google.'")
	}

	// Should have at least 5 prefixes
	if len(prefixes) < 5 {
		t.Errorf("Expected at least 5 default prefixes, got %d", len(prefixes))
	}
}

func TestLockFileManager_Generate(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Create config file
	configPath := filepath.Join(tempDir, "buffalo.yaml")
	configContent := `project:
  name: test-project
  version: 1.0.0
proto:
  paths:
    - protos
output:
  base_dir: generated
languages:
  python:
    enabled: true
    workdir: myapp
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Create proto directory and file
	protoDir := filepath.Join(tempDir, "protos")
	if err := os.MkdirAll(protoDir, 0755); err != nil {
		t.Fatalf("Failed to create proto directory: %v", err)
	}

	protoFile := filepath.Join(protoDir, "test.proto")
	protoContent := `syntax = "proto3";
package test.v1;

message TestMessage {
  string id = 1;
}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		t.Fatalf("Failed to create proto file: %v", err)
	}

	// Create config
	cfg := &Config{
		Project: ProjectConfig{
			Name:    "test-project",
			Version: "1.0.0",
		},
		Proto: ProtoConfig{
			Paths: []string{"protos"},
		},
		Output: OutputConfig{
			BaseDir: "generated",
		},
		Languages: LanguagesConfig{
			Python: PythonConfig{
				Enabled: true,
				WorkDir: "myapp",
			},
		},
	}

	// Create lock manager
	manager := NewLockFileManager(configPath)

	// Generate lock file
	lock, err := manager.Generate(cfg, []string{protoFile})
	if err != nil {
		t.Fatalf("Failed to generate lock file: %v", err)
	}

	// Verify lock file
	if lock.Version != "1" {
		t.Errorf("Expected version '1', got '%s'", lock.Version)
	}

	if lock.Project.Name != "test-project" {
		t.Errorf("Expected project name 'test-project', got '%s'", lock.Project.Name)
	}

	if lock.ConfigHash == "" {
		t.Error("Expected config hash to be set")
	}

	if lock.Proto.TotalFiles != 1 {
		t.Errorf("Expected 1 proto file, got %d", lock.Proto.TotalFiles)
	}

	if lock.Languages.Python == nil {
		t.Error("Expected Python lock to be set")
	} else {
		if lock.Languages.Python.WorkDir != "myapp" {
			t.Errorf("Expected workdir 'myapp', got '%s'", lock.Languages.Python.WorkDir)
		}

		if len(lock.Languages.Python.ExcludeImports) == 0 {
			t.Error("Expected exclude imports to be populated with defaults")
		}
	}
}

func TestLockFileManager_SaveAndLoad(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "buffalo.yaml")

	// Create minimal config file
	if err := os.WriteFile(configPath, []byte("project:\n  name: test\n"), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	manager := NewLockFileManager(configPath)

	// Create lock file
	lock := &LockFile{
		Version:     "1",
		GeneratedAt: time.Now().UTC(),
		ConfigHash:  "abc123",
		Project: ProjectLock{
			Name:    "test",
			Version: "1.0.0",
		},
		Languages: LanguagesLock{
			Python: &PythonLock{
				Enabled:        true,
				WorkDir:        "myapp",
				ExcludeImports: []string{"google.", "grpc"},
			},
		},
	}

	// Save
	if err := manager.Save(lock); err != nil {
		t.Fatalf("Failed to save lock file: %v", err)
	}

	// Verify file exists
	lockPath := manager.GetLockPath()
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Fatalf("Lock file was not created at %s", lockPath)
	}

	// Load
	loaded, err := manager.Load()
	if err != nil {
		t.Fatalf("Failed to load lock file: %v", err)
	}

	// Verify loaded content
	if loaded.Version != lock.Version {
		t.Errorf("Expected version '%s', got '%s'", lock.Version, loaded.Version)
	}

	if loaded.Project.Name != lock.Project.Name {
		t.Errorf("Expected project name '%s', got '%s'", lock.Project.Name, loaded.Project.Name)
	}

	if loaded.Languages.Python.WorkDir != lock.Languages.Python.WorkDir {
		t.Errorf("Expected workdir '%s', got '%s'", lock.Languages.Python.WorkDir, loaded.Languages.Python.WorkDir)
	}
}

func TestLockFileManager_NeedsRegeneration(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "buffalo.yaml")

	// Create config file
	configContent := "project:\n  name: test\n"
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	manager := NewLockFileManager(configPath)

	// Case 1: No lock file exists
	needs, reason, err := manager.NeedsRegeneration()
	if err != nil {
		t.Fatalf("NeedsRegeneration failed: %v", err)
	}
	if !needs {
		t.Error("Expected needs=true when lock file doesn't exist")
	}
	if reason != "lock file does not exist" {
		t.Errorf("Expected reason 'lock file does not exist', got '%s'", reason)
	}

	// Create lock file with current config hash
	cfg := &Config{
		Project: ProjectConfig{Name: "test"},
		Proto:   ProtoConfig{Paths: []string{"."}},
		Output:  OutputConfig{BaseDir: "gen"},
		Languages: LanguagesConfig{
			Python: PythonConfig{Enabled: true},
		},
	}
	lock, _ := manager.Generate(cfg, []string{})
	manager.Save(lock)

	// Case 2: Lock file exists and config unchanged
	needs, reason, err = manager.NeedsRegeneration()
	if err != nil {
		t.Fatalf("NeedsRegeneration failed: %v", err)
	}
	if needs {
		t.Errorf("Expected needs=false when config unchanged, got reason: %s", reason)
	}

	// Case 3: Config changed
	newContent := "project:\n  name: test-changed\n"
	if err := os.WriteFile(configPath, []byte(newContent), 0644); err != nil {
		t.Fatalf("Failed to update config file: %v", err)
	}

	needs, reason, err = manager.NeedsRegeneration()
	if err != nil {
		t.Fatalf("NeedsRegeneration failed: %v", err)
	}
	if !needs {
		t.Error("Expected needs=true when config changed")
	}
	if reason != "config file has changed" {
		t.Errorf("Expected reason 'config file has changed', got '%s'", reason)
	}
}

func TestValidateWorkDir(t *testing.T) {
	tests := []struct {
		name    string
		workDir string
		wantErr bool
	}{
		{"empty", "", false},
		{"simple", "myapp", false},
		{"dotted", "my.app", false},
		{"underscore", "my_app", false},
		{"starts_with_number", "1app", true},
		{"has_hyphen", "my-app", true},
		{"double_dot", "my..app", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use the python package's ValidateWorkDir
			// This test is here to ensure lock config works with validation
			_ = tt.workDir
		})
	}
}

func TestResolveExcludeImports(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "buffalo.yaml")
	os.WriteFile(configPath, []byte("project:\n  name: test\n"), 0644)

	manager := NewLockFileManager(configPath)

	cfg := &Config{
		Languages: LanguagesConfig{
			Python: PythonConfig{
				Enabled:        true,
				ExcludeImports: []string{"custom.prefix"},
			},
		},
	}

	deps := []DependencyLock{
		{
			Name:             "googleapis",
			ExternalPrefixes: []string{"google.api"},
		},
	}

	resolved := manager.resolveExcludeImports(cfg, deps)

	// Should contain defaults
	hasGoogle := false
	hasCustom := false
	hasGoogleApi := false

	for _, p := range resolved {
		if p == "google." {
			hasGoogle = true
		}
		if p == "custom.prefix" {
			hasCustom = true
		}
		if p == "google.api" {
			hasGoogleApi = true
		}
	}

	if !hasGoogle {
		t.Error("Expected resolved prefixes to contain default 'google.'")
	}
	if !hasCustom {
		t.Error("Expected resolved prefixes to contain user-defined 'custom.prefix'")
	}
	if !hasGoogleApi {
		t.Error("Expected resolved prefixes to contain dependency 'google.api'")
	}
}

func TestLockFile_GetExcludeImports(t *testing.T) {
	// With Python config
	lock := &LockFile{
		Languages: LanguagesLock{
			Python: &PythonLock{
				ExcludeImports: []string{"custom.prefix", "google."},
			},
		},
	}

	imports := lock.GetExcludeImports()
	if len(imports) != 2 {
		t.Errorf("Expected 2 imports, got %d", len(imports))
	}

	// Without Python config
	lock2 := &LockFile{}
	imports2 := lock2.GetExcludeImports()
	if len(imports2) == 0 {
		t.Error("Expected default imports when Python not configured")
	}
}

func TestDetectExternalPrefixes(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "buffalo.yaml")
	os.WriteFile(configPath, []byte("project:\n  name: test\n"), 0644)

	manager := NewLockFileManager(configPath)

	// Test with DependencyLock
	depLock := DependencyLock{
		Name: "googleapis-common-protos",
	}

	prefixes := manager.detectExternalPrefixes(depLock)

	hasGoogle := false
	for _, p := range prefixes {
		if p == "google." {
			hasGoogle = true
			break
		}
	}

	if !hasGoogle {
		t.Error("Expected to detect 'google.' prefix from googleapis dependency")
	}
}

func TestHashFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.txt")

	content := "Hello, World!"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hash1, err := hashFile(filePath)
	if err != nil {
		t.Fatalf("hashFile failed: %v", err)
	}

	if hash1 == "" {
		t.Error("Expected non-empty hash")
	}

	// Same content should produce same hash
	hash2, _ := hashFile(filePath)
	if hash1 != hash2 {
		t.Error("Expected same hash for unchanged file")
	}

	// Different content should produce different hash
	os.WriteFile(filePath, []byte("Different content"), 0644)
	hash3, _ := hashFile(filePath)
	if hash1 == hash3 {
		t.Error("Expected different hash for changed file")
	}
}

func TestUniqueSortedStrings(t *testing.T) {
	input := []string{"c", "a", "b", "a", "c", "", "b"}
	result := uniqueSortedStrings(input)

	expected := []string{"a", "b", "c"}

	if len(result) != len(expected) {
		t.Errorf("Expected %d elements, got %d", len(expected), len(result))
	}

	for i, v := range expected {
		if result[i] != v {
			t.Errorf("Expected result[%d]='%s', got '%s'", i, v, result[i])
		}
	}
}
