package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Project: ProjectConfig{
					Name:    "test-project",
					Version: "0.1.0",
				},
				Proto: ProtoConfig{
					Paths: []string{"./protos"},
				},
				Output: OutputConfig{
					BaseDir: "./generated",
				},
				Languages: LanguagesConfig{
					Python: PythonConfig{Enabled: true},
				},
			},
			wantErr: false,
		},
		{
			name: "missing project name",
			config: Config{
				Proto: ProtoConfig{
					Paths: []string{"./protos"},
				},
				Output: OutputConfig{
					BaseDir: "./generated",
				},
				Languages: LanguagesConfig{
					Python: PythonConfig{Enabled: true},
				},
			},
			wantErr: true,
		},
		{
			name: "missing proto paths",
			config: Config{
				Project: ProjectConfig{
					Name: "test-project",
				},
				Output: OutputConfig{
					BaseDir: "./generated",
				},
				Languages: LanguagesConfig{
					Python: PythonConfig{Enabled: true},
				},
			},
			wantErr: true,
		},
		{
			name: "missing output base dir",
			config: Config{
				Project: ProjectConfig{
					Name: "test-project",
				},
				Proto: ProtoConfig{
					Paths: []string{"./protos"},
				},
				Languages: LanguagesConfig{
					Python: PythonConfig{Enabled: true},
				},
			},
			wantErr: true,
		},
		{
			name: "no languages enabled",
			config: Config{
				Project: ProjectConfig{
					Name: "test-project",
				},
				Proto: ProtoConfig{
					Paths: []string{"./protos"},
				},
				Output: OutputConfig{
					BaseDir: "./generated",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid logging level",
			config: Config{
				Project: ProjectConfig{
					Name: "test-project",
				},
				Proto: ProtoConfig{
					Paths: []string{"./protos"},
				},
				Output: OutputConfig{
					BaseDir: "./generated",
				},
				Languages: LanguagesConfig{
					Python: PythonConfig{Enabled: true},
				},
				Logging: LoggingConfig{
					Level: "invalid",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_GetOutputDir(t *testing.T) {
	cfg := &Config{
		Output: OutputConfig{
			BaseDir: "./generated",
			Directories: map[string]string{
				"python": "py",
				"go":     "golang",
			},
		},
	}

	tests := []struct {
		name     string
		language string
		want     string
	}{
		{
			name:     "existing mapping",
			language: "python",
			want:     filepath.Join("generated", "py"),
		},
		{
			name:     "existing mapping go",
			language: "go",
			want:     filepath.Join("generated", "golang"),
		},
		{
			name:     "default mapping",
			language: "rust",
			want:     filepath.Join("generated", "rust"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.GetOutputDir(tt.language)
			if got != tt.want {
				t.Errorf("Config.GetOutputDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_IsLanguageEnabled(t *testing.T) {
	cfg := &Config{
		Languages: LanguagesConfig{
			Python: PythonConfig{Enabled: true},
			Go:     GoConfig{Enabled: false},
			Rust:   RustConfig{Enabled: true},
			Cpp:    CppConfig{Enabled: false},
		},
	}

	tests := []struct {
		name     string
		language string
		want     bool
	}{
		{"python enabled", "python", true},
		{"go disabled", "go", false},
		{"rust enabled", "rust", true},
		{"cpp disabled", "cpp", false},
		{"unknown language", "java", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cfg.IsLanguageEnabled(tt.language); got != tt.want {
				t.Errorf("Config.IsLanguageEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_GetEnabledLanguages(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		want   []string
	}{
		{
			name: "all enabled",
			config: Config{
				Languages: LanguagesConfig{
					Python: PythonConfig{Enabled: true},
					Go:     GoConfig{Enabled: true},
					Rust:   RustConfig{Enabled: true},
					Cpp:    CppConfig{Enabled: true},
				},
			},
			want: []string{"python", "go", "rust", "cpp"},
		},
		{
			name: "python and go only",
			config: Config{
				Languages: LanguagesConfig{
					Python: PythonConfig{Enabled: true},
					Go:     GoConfig{Enabled: true},
				},
			},
			want: []string{"python", "go"},
		},
		{
			name: "none enabled",
			config: Config{
				Languages: LanguagesConfig{},
			},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetEnabledLanguages()
			if len(got) != len(tt.want) {
				t.Errorf("Config.GetEnabledLanguages() len = %v, want %v", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("Config.GetEnabledLanguages()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create valid config file
	validConfig := `
project:
  name: test-project
  version: 0.1.0

proto:
  paths:
    - ./protos

output:
  base_dir: ./generated

languages:
  python:
    enabled: true
`
	validPath := filepath.Join(tmpDir, "valid.yaml")
	if err := os.WriteFile(validPath, []byte(validConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Create invalid config file
	invalidConfig := `
project:
  name: test-project

proto:
  paths: []

output:
  base_dir: ./generated

languages:
  python:
    enabled: false
`
	invalidPath := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(invalidPath, []byte(invalidConfig), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid config",
			path:    validPath,
			wantErr: false,
		},
		{
			name:    "invalid config",
			path:    invalidPath,
			wantErr: true,
		},
		{
			name:    "non-existent file",
			path:    filepath.Join(tmpDir, "notfound.yaml"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := LoadFromFile(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFromFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && cfg == nil {
				t.Error("LoadFromFile() returned nil config")
			}
		})
	}
}

func TestLoad(t *testing.T) {
	// Setup viper with test config
	viper.Reset()
	viper.Set("project.name", "test-project")
	viper.Set("project.version", "0.1.0")
	viper.Set("proto.paths", []string{"./protos"})
	viper.Set("output.base_dir", "./generated")
	viper.Set("languages.python.enabled", true)

	cfg, err := Load()
	if err != nil {
		t.Errorf("Load() error = %v", err)
		return
	}
	if cfg == nil {
		t.Error("Load() returned nil config")
		return
	}
	if cfg.Project.Name != "test-project" {
		t.Errorf("Load() project.name = %v, want test-project", cfg.Project.Name)
	}
}

func TestConfig_NormalizeTypescript(t *testing.T) {
	cfg := &Config{
		Output: OutputConfig{
			BaseDir: "./generated",
		},
		Languages: LanguagesConfig{
			Typescript: TypescriptConfig{
				Enabled: true,
				Output:  "./generated/typescript",
				Options: TypescriptOptionsConfig{
					Generator: "ts-proto",
				},
			},
		},
	}

	cfg.Normalize()

	if cfg.Languages.Typescript.Generator != "ts-proto" {
		t.Fatalf("expected generator ts-proto, got %q", cfg.Languages.Typescript.Generator)
	}

	dir, ok := cfg.Output.Directories["typescript"]
	if !ok {
		t.Fatal("expected output.directories.typescript to be set")
	}

	if dir != "typescript" {
		t.Fatalf("expected normalized directory 'typescript', got %q", dir)
	}
}
