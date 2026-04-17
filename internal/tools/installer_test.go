package tools

import (
	"testing"

	"github.com/massonsky/buffalo/pkg/logger"
)

func TestNewInstaller(t *testing.T) {
	log := logger.New()
	installer := NewInstaller(log)

	if installer == nil {
		t.Fatal("NewInstaller() returned nil")
	}

	if installer.platform == "" {
		t.Error("Installer platform is empty")
	}
}

func TestInstallerListTools(t *testing.T) {
	log := logger.New()
	installer := NewInstaller(log)

	tests := []struct {
		name       string
		languages  []string
		includeAll bool
		minCount   int
	}{
		{"Go only", []string{"go"}, false, 3},
		{"Python only", []string{"python"}, false, 3},
		{"Go and Python", []string{"go", "python"}, false, 5},
		{"Go with all", []string{"go"}, true, 5},
		{"All languages", []string{"go", "python", "rust", "cpp"}, false, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tools := installer.ListTools(tt.languages, tt.includeAll)
			if len(tools) < tt.minCount {
				t.Errorf("ListTools(%v, %v) returned %d tools, expected at least %d",
					tt.languages, tt.includeAll, len(tools), tt.minCount)
			}
		})
	}
}

func TestInstallerCheck(t *testing.T) {
	log := logger.New()
	installer := NewInstaller(log)

	// Test checking protoc (should be installed in dev environment)
	protocTool := Tool{
		Name:         "protoc",
		CheckCommand: "protoc",
		CheckArgs:    []string{"--version"},
	}

	installed, version, err := installer.Check(protocTool)

	// protoc might or might not be installed, so we just check that check runs without panic
	t.Logf("protoc check: installed=%v, version=%q, err=%v", installed, version, err)
}

func TestInstallerInstallDryRun(t *testing.T) {
	log := logger.New()
	installer := NewInstaller(log)

	tool := Tool{
		Name:         "test-tool",
		Description:  "Test tool",
		CheckCommand: "nonexistent-tool",
		CheckArgs:    []string{"--version"},
		InstallMethods: map[string]string{
			"linux":   "echo test",
			"darwin":  "echo test",
			"windows": "echo test",
		},
	}

	opts := InstallOptions{
		DryRun: true,
	}

	result := installer.Install(tool, opts)

	if !result.Skipped {
		t.Error("Expected result to be skipped in dry run mode")
	}

	if result.Error != nil {
		t.Errorf("Unexpected error in dry run: %v", result.Error)
	}
}

func TestInstallerInstallAlreadyInstalled(t *testing.T) {
	log := logger.New()
	installer := NewInstaller(log)

	// Test with a tool that should exist (echo on Linux/macOS)
	tool := Tool{
		Name:         "echo",
		Description:  "Echo command",
		CheckCommand: "echo",
		CheckArgs:    []string{"test"},
		InstallMethods: map[string]string{
			"linux":   "echo already",
			"darwin":  "echo already",
			"windows": "echo already",
		},
	}

	opts := InstallOptions{
		Force: false,
	}

	result := installer.Install(tool, opts)

	if !result.AlreadyOK {
		// echo should be available
		t.Logf("Result: %+v", result)
	}
}

func TestInstallerCheckAll(t *testing.T) {
	log := logger.New()
	installer := NewInstaller(log)

	languages := []string{"go"}
	results := installer.CheckAll(languages)

	// Should have core and go sections
	if _, ok := results["core"]; !ok {
		t.Error("CheckAll results missing 'core' section")
	}

	if _, ok := results["go"]; !ok {
		t.Error("CheckAll results missing 'go' section")
	}

	// Go section should have multiple tools
	if len(results["go"]) < 2 {
		t.Errorf("Expected at least 2 Go tools, got %d", len(results["go"]))
	}
}

func TestInstallOptionsDefaults(t *testing.T) {
	opts := InstallOptions{}

	if opts.Force {
		t.Error("Default Force should be false")
	}

	if opts.DryRun {
		t.Error("Default DryRun should be false")
	}

	if opts.Interactive {
		t.Error("Default Interactive should be false")
	}

	if opts.IncludeAll {
		t.Error("Default IncludeAll should be false")
	}
}
