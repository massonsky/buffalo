package config

import (
	"fmt"
	"path/filepath"

	"github.com/massonsky/buffalo/internal/dependency"
	"github.com/massonsky/buffalo/pkg/errors"
	"github.com/spf13/viper"
)

// Config represents the Buffalo configuration
type Config struct {
	Project      ProjectConfig           `mapstructure:"project"`
	Proto        ProtoConfig             `mapstructure:"proto"`
	Output       OutputConfig            `mapstructure:"output"`
	Languages    LanguagesConfig         `mapstructure:"languages"`
	Build        BuildConfig             `mapstructure:"build"`
	Versioning   VersioningConfig        `mapstructure:"versioning"`
	Logging      LoggingConfig           `mapstructure:"logging"`
	Dependencies []dependency.Dependency `mapstructure:"dependencies"`
	Plugins      []PluginConfig          `mapstructure:"plugins"`
	Templates    []TemplateConfig        `mapstructure:"templates"`
	Models       ModelsConfig            `mapstructure:"models"`
}

// ModelsConfig contains buffalo.models generation settings.
type ModelsConfig struct {
	Enabled                 bool              `mapstructure:"enabled"`
	GenerateModelsFromProto bool              `mapstructure:"generate_models_from_proto"`
	BaseModelFields         []BaseFieldConfig `mapstructure:"base_model_fields"`
}

// BaseFieldConfig describes a field injected into the generated BaseModel.
type BaseFieldConfig struct {
	Name         string `mapstructure:"name"`
	Type         string `mapstructure:"type"`
	PrimaryKey   bool   `mapstructure:"primary_key"`
	AutoGenerate bool   `mapstructure:"auto_generate"`
	AutoNow      bool   `mapstructure:"auto_now"`
	AutoNowAdd   bool   `mapstructure:"auto_now_add"`
	Nullable     bool   `mapstructure:"nullable"`
	Comment      string `mapstructure:"comment"`
}

// ProjectConfig contains project-level settings
type ProjectConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
}

// ProtoConfig contains proto file settings
type ProtoConfig struct {
	Paths       []string `mapstructure:"paths"`
	Exclude     []string `mapstructure:"exclude"`
	ImportPaths []string `mapstructure:"import_paths"`
}

// OutputConfig contains output settings
type OutputConfig struct {
	BaseDir                string            `mapstructure:"base_dir"`
	Directories            map[string]string `mapstructure:"directories"`
	PreserveProtoStructure bool              `mapstructure:"preserve_proto_structure"`
}

// LanguagesConfig contains language-specific settings
type LanguagesConfig struct {
	Python PythonConfig `mapstructure:"python"`
	Go     GoConfig     `mapstructure:"go"`
	Rust   RustConfig   `mapstructure:"rust"`
	Cpp    CppConfig    `mapstructure:"cpp"`
}

// PythonConfig contains Python-specific settings
type PythonConfig struct {
	Enabled        bool     `mapstructure:"enabled"`
	Package        string   `mapstructure:"package"`
	Generator      string   `mapstructure:"generator"`
	WorkDir        string   `mapstructure:"workdir"`
	ExcludeImports []string `mapstructure:"exclude_imports"`
	ORM            bool     `mapstructure:"orm"`
	ORMPlugin      string   `mapstructure:"orm_plugin"`
	ModelsOutput   string   `mapstructure:"models_output"`
}

// GoConfig contains Go-specific settings
type GoConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	Module       string `mapstructure:"module"`
	Generator    string `mapstructure:"generator"`
	ORM          bool   `mapstructure:"orm"`
	ORMPlugin    string `mapstructure:"orm_plugin"`
	ModelsOutput string `mapstructure:"models_output"`
}

// RustConfig contains Rust-specific settings
type RustConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	Generator    string `mapstructure:"generator"`
	ORM          bool   `mapstructure:"orm"`
	ORMPlugin    string `mapstructure:"orm_plugin"`
	ModelsOutput string `mapstructure:"models_output"`
}

// CppConfig contains C++-specific settings
type CppConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	Namespace    string `mapstructure:"namespace"`
	ORM          bool   `mapstructure:"orm"`
	ORMPlugin    string `mapstructure:"orm_plugin"`
	ModelsOutput string `mapstructure:"models_output"`
}

// BuildConfig contains build settings
type BuildConfig struct {
	Workers     int         `mapstructure:"workers"`
	Incremental bool        `mapstructure:"incremental"`
	Cache       CacheConfig `mapstructure:"cache"`
}

// CacheConfig contains cache settings
type CacheConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Directory string `mapstructure:"directory"`
}

// VersioningConfig contains versioning settings
type VersioningConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	Strategy     string `mapstructure:"strategy"`      // hash, timestamp, semantic, git
	OutputFormat string `mapstructure:"output_format"` // directory, suffix
	KeepVersions int    `mapstructure:"keep_versions"` // 0 = keep all, >0 = keep N latest
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
	File   string `mapstructure:"file"`
}

// PluginConfig contains plugin configuration
type PluginConfig struct {
	Name       string                 `mapstructure:"name"`
	Enabled    bool                   `mapstructure:"enabled"`
	HookPoints []string               `mapstructure:"hooks"`
	Priority   int                    `mapstructure:"priority"`
	Options    map[string]interface{} `mapstructure:"config"`
}

// TemplateConfig contains template configuration
type TemplateConfig struct {
	Name     string            `mapstructure:"name"`
	Language string            `mapstructure:"language"`
	Path     string            `mapstructure:"path"`
	Patterns []string          `mapstructure:"patterns"`
	Vars     map[string]string `mapstructure:"vars"`
}

// Load loads configuration from viper
func Load() (*Config, error) {
	var cfg Config

	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, errors.Wrap(err, errors.ErrConfig, "failed to unmarshal config")
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadFromFile loads configuration from a specific file
func LoadFromFile(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)

	if err := v.ReadInConfig(); err != nil {
		return nil, errors.Wrap(err, errors.ErrConfig, "failed to read config file")
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, errors.Wrap(err, errors.ErrConfig, "failed to unmarshal config")
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate project
	if c.Project.Name == "" {
		return errors.New(errors.ErrConfig, "project.name is required")
	}

	// Validate proto paths
	if len(c.Proto.Paths) == 0 {
		return errors.New(errors.ErrConfig, "proto.paths must contain at least one path")
	}

	// Validate output
	if c.Output.BaseDir == "" {
		return errors.New(errors.ErrConfig, "output.base_dir is required")
	}

	// Validate at least one language is enabled
	if !c.Languages.Python.Enabled &&
		!c.Languages.Go.Enabled &&
		!c.Languages.Rust.Enabled &&
		!c.Languages.Cpp.Enabled {
		return errors.New(errors.ErrConfig, "at least one language must be enabled")
	}

	// Validate logging level
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if c.Logging.Level != "" && !validLevels[c.Logging.Level] {
		return errors.New(errors.ErrConfig, fmt.Sprintf("invalid logging level: %s", c.Logging.Level))
	}

	return nil
}

// GetOutputDir returns the output directory for a specific language
func (c *Config) GetOutputDir(language string) string {
	if dir, ok := c.Output.Directories[language]; ok {
		return filepath.Join(c.Output.BaseDir, dir)
	}
	return filepath.Join(c.Output.BaseDir, language)
}

// IsLanguageEnabled returns whether a language is enabled
func (c *Config) IsLanguageEnabled(language string) bool {
	switch language {
	case "python":
		return c.Languages.Python.Enabled
	case "go":
		return c.Languages.Go.Enabled
	case "rust":
		return c.Languages.Rust.Enabled
	case "cpp":
		return c.Languages.Cpp.Enabled
	default:
		return false
	}
}

// GetEnabledLanguages returns a list of enabled languages
func (c *Config) GetEnabledLanguages() []string {
	var languages []string
	if c.Languages.Python.Enabled {
		languages = append(languages, "python")
	}
	if c.Languages.Go.Enabled {
		languages = append(languages, "go")
	}
	if c.Languages.Rust.Enabled {
		languages = append(languages, "rust")
	}
	if c.Languages.Cpp.Enabled {
		languages = append(languages, "cpp")
	}
	return languages
}

// GetModelsOutputDir returns the models output directory for a specific language.
// Priority: language.models_output > output.directories["models_<lang>"] > output.base_dir/<lang>/models
func (c *Config) GetModelsOutputDir(language string) string {
	switch language {
	case "python":
		if c.Languages.Python.ModelsOutput != "" {
			return c.Languages.Python.ModelsOutput
		}
	case "go":
		if c.Languages.Go.ModelsOutput != "" {
			return c.Languages.Go.ModelsOutput
		}
	case "rust":
		if c.Languages.Rust.ModelsOutput != "" {
			return c.Languages.Rust.ModelsOutput
		}
	case "cpp":
		if c.Languages.Cpp.ModelsOutput != "" {
			return c.Languages.Cpp.ModelsOutput
		}
	}

	// Fallback: base_dir/<lang>/models
	return filepath.Join(c.GetOutputDir(language), "models")
}

// IsORMEnabled returns whether ORM model generation is enabled for a language.
func (c *Config) IsORMEnabled(language string) bool {
	if !c.Models.Enabled {
		return false
	}
	switch language {
	case "python":
		return c.Languages.Python.ORM
	case "go":
		return c.Languages.Go.ORM
	case "rust":
		return c.Languages.Rust.ORM
	case "cpp":
		return c.Languages.Cpp.ORM
	default:
		return false
	}
}

// GetORMPlugin returns the ORM plugin string for a language.
func (c *Config) GetORMPlugin(language string) string {
	switch language {
	case "python":
		return c.Languages.Python.ORMPlugin
	case "go":
		return c.Languages.Go.ORMPlugin
	case "rust":
		return c.Languages.Rust.ORMPlugin
	case "cpp":
		return c.Languages.Cpp.ORMPlugin
	default:
		return ""
	}
}

// IsGenerateModelsFromProto returns true when plain proto messages
// (without buffalo.models annotations) should also produce model code.
func (c *Config) IsGenerateModelsFromProto() bool {
	return c.Models.GenerateModelsFromProto
}
