package tools

import (
	"testing"
)

func TestGetPlatform(t *testing.T) {
	platform := GetPlatform()
	validPlatforms := []string{"linux", "darwin", "windows"}

	found := false
	for _, p := range validPlatforms {
		if platform == p {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("GetPlatform() returned unexpected value: %s", platform)
	}
}

func TestGetToolsForLanguage(t *testing.T) {
	tests := []struct {
		name     string
		language string
		minCount int
	}{
		{"Go tools", "go", 4},
		{"Python tools", "python", 4},
		{"Rust tools", "rust", 2},
		{"C++ tools", "cpp", 3},
		{"Core tools", "all", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tools := GetToolsForLanguage(tt.language)
			if len(tools) < tt.minCount {
				t.Errorf("GetToolsForLanguage(%s) returned %d tools, expected at least %d",
					tt.language, len(tools), tt.minCount)
			}
		})
	}
}

func TestGetCriticalTools(t *testing.T) {
	tests := []struct {
		name     string
		language string
		minCount int
	}{
		{"Go critical tools", "go", 3},
		{"Python critical tools", "python", 3},
		{"Core critical tools", "all", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tools := GetCriticalTools(tt.language)
			if len(tools) < tt.minCount {
				t.Errorf("GetCriticalTools(%s) returned %d tools, expected at least %d",
					tt.language, len(tools), tt.minCount)
			}

			// Verify all returned tools are critical
			for _, tool := range tools {
				if !tool.Critical {
					t.Errorf("GetCriticalTools(%s) returned non-critical tool: %s",
						tt.language, tool.Name)
				}
			}
		})
	}
}

func TestGetAllCriticalTools(t *testing.T) {
	tools := GetAllCriticalTools()
	if len(tools) < 5 {
		t.Errorf("GetAllCriticalTools() returned %d tools, expected at least 5", len(tools))
	}

	// Verify all returned tools are critical
	for _, tool := range tools {
		if !tool.Critical {
			t.Errorf("GetAllCriticalTools() returned non-critical tool: %s", tool.Name)
		}
	}
}

func TestToolRegistryIntegrity(t *testing.T) {
	languages := map[string]bool{
		"all":    true,
		"go":     true,
		"python": true,
		"rust":   true,
		"cpp":    true,
	}

	for _, tool := range ToolRegistry {
		// Check language is valid
		if !languages[tool.Language] {
			t.Errorf("Tool %s has invalid language: %s", tool.Name, tool.Language)
		}

		// Check name is not empty
		if tool.Name == "" {
			t.Error("Found tool with empty name")
		}

		// Check description is not empty
		if tool.Description == "" {
			t.Errorf("Tool %s has empty description", tool.Name)
		}

		// Check install methods are present
		if len(tool.InstallMethods) == 0 {
			t.Errorf("Tool %s has no install methods", tool.Name)
		}

		// Check install methods have at least Linux
		if _, ok := tool.InstallMethods["linux"]; !ok {
			t.Errorf("Tool %s has no Linux install method", tool.Name)
		}
	}
}

func TestToolHasCheckMethod(t *testing.T) {
	for _, tool := range ToolRegistry {
		// Tools should have either CheckCommand or CheckFunc (except some pip packages)
		if tool.CheckCommand == "" && tool.CheckFunc == nil {
			// Python packages without commands are OK (they use pip check)
			if tool.Language != "python" {
				t.Errorf("Tool %s has no check method", tool.Name)
			}
		}
	}
}
