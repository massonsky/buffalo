package config

import (
	"fmt"
	"path/filepath"
	"strings"

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
	Python     PythonConfig     `mapstructure:"python"`
	Go         GoConfig         `mapstructure:"go"`
	Rust       RustConfig       `mapstructure:"rust"`
	Cpp        CppConfig        `mapstructure:"cpp"`
	Typescript TypescriptConfig `mapstructure:"typescript"`
}

// PythonConfig contains Python-specific settings
type PythonConfig struct {
	Enabled         bool     `mapstructure:"enabled"`
	Package         string   `mapstructure:"package"`
	Generator       string   `mapstructure:"generator"`
	WorkDir         string   `mapstructure:"workdir"`
	ExcludeImports  []string `mapstructure:"exclude_imports"`
	ORM             bool     `mapstructure:"orm"`
	ORMPlugin       string   `mapstructure:"orm_plugin"`
	ModelsOutput    string   `mapstructure:"models_output"`
	Pb2ImportPrefix string   `mapstructure:"pb2_import_prefix"`
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

// TypescriptConfig contains TypeScript-specific settings
type TypescriptConfig struct {
	Enabled      bool                    `mapstructure:"enabled"`
	Generator    string                  `mapstructure:"generator"`
	Output       string                  `mapstructure:"output"`
	Plugins      []string                `mapstructure:"plugins"`
	Options      TypescriptOptionsConfig `mapstructure:"options"`
	ORM          bool                    `mapstructure:"orm"`
	ORMPlugin    string                  `mapstructure:"orm_plugin"`
	ModelsOutput string                  `mapstructure:"models_output"`
}

// TypescriptOptionsConfig contains nested TypeScript options for backward-compatible YAML schema.
type TypescriptOptionsConfig struct {
	Generator        string `mapstructure:"generator"`
	ProtocGenTsPath  string `mapstructure:"protoc_gen_ts_path"`
	TsProtoPath      string `mapstructure:"ts_proto_path"`
	ESModules        *bool  `mapstructure:"es_modules"`
	GenerateGrpc     *bool  `mapstructure:"generate_grpc"`
	GenerateGrpcWeb  *bool  `mapstructure:"generate_grpc_web"`
	GenerateNiceGrpc *bool  `mapstructure:"generate_nice_grpc"`
	OutputIndex      *bool  `mapstructure:"output_index"`
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

	cfg.Normalize()

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

	cfg.Normalize()

	return &cfg, nil
}

// Normalize applies backward-compatible mapping between legacy and current fields.
func (c *Config) Normalize() {
	// Map nested TS options.generator -> languages.typescript.generator.
	if c.Languages.Typescript.Generator == "" && c.Languages.Typescript.Options.Generator != "" {
		c.Languages.Typescript.Generator = c.Languages.Typescript.Options.Generator
	}

	// Map languages.typescript.output to output.directories.typescript.
	if c.Languages.Typescript.Output != "" {
		if c.Output.Directories == nil {
			c.Output.Directories = map[string]string{}
		}
		c.Output.Directories["typescript"] = normalizeLanguageOutputDir(c.Output.BaseDir, c.Languages.Typescript.Output)
	}
}

func normalizeLanguageOutputDir(baseDir string, langOutput string) string {
	base := filepath.Clean(baseDir)
	out := filepath.Clean(langOutput)

	if filepath.IsAbs(out) {
		return out
	}

	rel, err := filepath.Rel(base, out)
	if err == nil {
		// If output is already under baseDir (e.g. ./generated/typescript),
		// store only language subpath (e.g. typescript).
		if rel == "." {
			return rel
		}
		if !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." {
			return rel
		}
	}

	return out
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
		!c.Languages.Cpp.Enabled &&
		!c.Languages.Typescript.Enabled {
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
	case "typescript":
		return c.Languages.Typescript.Enabled
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
	if c.Languages.Typescript.Enabled {
		languages = append(languages, "typescript")
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
	case "typescript":
		if c.Languages.Typescript.ModelsOutput != "" {
			return c.Languages.Typescript.ModelsOutput
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
	case "typescript":
		return c.Languages.Typescript.ORM
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
	case "typescript":
		return c.Languages.Typescript.ORMPlugin
	default:
		return ""
	}
}

// IsGenerateModelsFromProto returns true when plain proto messages
// (without buffalo.models annotations) should also produce model code.
func (c *Config) IsGenerateModelsFromProto() bool {
	return c.Models.GenerateModelsFromProto
}

// GetPb2ImportPrefix returns the pb2 import prefix for Python.
// Only relevant for Python language.
func (c *Config) GetPb2ImportPrefix(language string) string {
	switch language {
	case "python":
		return c.Languages.Python.Pb2ImportPrefix
	default:
		return ""
	}
}
