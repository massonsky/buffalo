package models

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/massonsky/buffalo/internal/plugin"
	"github.com/massonsky/buffalo/pkg/logger"
)

// ══════════════════════════════════════════════════════════════════
//  ModelsPlugin — buffalo.models generator plugin
// ══════════════════════════════════════════════════════════════════

// ModelsPlugin implements plugin.Plugin and generates typed code models
// from buffalo.models annotations in proto files.
type ModelsPlugin struct {
	config plugin.Config
	log    *logger.Logger
}

// NewModelsPlugin creates a new ModelsPlugin instance.
func NewModelsPlugin(log *logger.Logger) plugin.Plugin {
	return &ModelsPlugin{log: log}
}

func (p *ModelsPlugin) Name() string            { return "buffalo-models" }
func (p *ModelsPlugin) Version() string         { return "1.0.0" }
func (p *ModelsPlugin) Type() plugin.PluginType { return plugin.PluginTypeGenerator }
func (p *ModelsPlugin) Description() string {
	return "Generates typed code models from buffalo.models proto annotations"
}

func (p *ModelsPlugin) Init(cfg plugin.Config) error {
	p.config = cfg
	return nil
}

func (p *ModelsPlugin) Shutdown() error { return nil }

// Execute parses proto files for buffalo.models annotations and generates
// model source code for each enabled language.
func (p *ModelsPlugin) Execute(ctx context.Context, input *plugin.Input) (*plugin.Output, error) {
	output := &plugin.Output{
		Success:  true,
		Messages: []string{},
		Warnings: []string{},
		Errors:   []string{},
		Metadata: map[string]interface{}{},
	}

	// Load ORM settings from plugin options
	var cfg *pluginModelsConfig
	cfg = p.buildConfig()

	// ── Parse all proto files for model definitions ──
	var allModels []ModelDef
	for _, protoFile := range input.ProtoFiles {
		content, err := os.ReadFile(protoFile)
		if err != nil {
			output.Warnings = append(output.Warnings,
				fmt.Sprintf("buffalo-models: error reading %s: %v", protoFile, err))
			continue
		}

		var models []ModelDef
		if cfg.generateModelsFromProto {
			// Generate models from ALL proto messages (not just annotated ones)
			models, err = ExtractAllMessages(string(content), protoFile)
		} else {
			// Only annotated messages
			models, err = ExtractModels(string(content), protoFile)
		}
		if err != nil {
			output.Warnings = append(output.Warnings,
				fmt.Sprintf("buffalo-models: error parsing %s: %v", protoFile, err))
			continue
		}

		allModels = append(allModels, models...)
	}

	if len(allModels) == 0 {
		output.Messages = append(output.Messages, "buffalo-models: no model annotations found")
		return output, nil
	}

	output.Messages = append(output.Messages,
		fmt.Sprintf("buffalo-models: found %d model(s) in %d proto file(s)",
			len(allModels), len(input.ProtoFiles)))

	// ── Generate models for each enabled language ──
	languages := []string{"python", "go", "rust", "cpp", "typescript"}
	for _, lang := range languages {
		if !cfg.isEnabled(lang) {
			continue
		}

		orm := cfg.getORM(lang)
		outputDir := cfg.getOutputDir(lang, input.OutputDir)

		pb2Prefix := cfg.getPb2ImportPrefix(lang)
		generated, err := p.generateForLanguage(lang, orm, outputDir, pb2Prefix, allModels, input)
		if err != nil {
			output.Errors = append(output.Errors,
				fmt.Sprintf("buffalo-models [%s]: %v", lang, err))
			output.Success = false
			continue
		}

		output.GeneratedFiles = append(output.GeneratedFiles, generated...)
		output.Messages = append(output.Messages,
			fmt.Sprintf("buffalo-models [%s/%s]: generated %d file(s) → %s",
				lang, orm.Name, len(generated), outputDir))
	}

	output.Metadata["models_count"] = len(allModels)
	return output, nil
}

// ──────────────────────────────────────────────────────────────────
//  Internal helpers — pluginModelsConfig
// ──────────────────────────────────────────────────────────────────

// pluginModelsConfig extracts ORM settings from plugin.Config.Options.
type pluginModelsConfig struct {
	langs                   map[string]langORMCfg
	generateModelsFromProto bool
}

type langORMCfg struct {
	enabled         bool
	ormPlugin       string
	outputDir       string
	pb2ImportPrefix string
}

func (p *ModelsPlugin) buildConfig() *pluginModelsConfig {
	c := &pluginModelsConfig{
		langs: map[string]langORMCfg{},
	}
	if p.config.Options == nil {
		return c
	}
	// Check generate_models_from_proto flag
	if v, ok := p.config.Options["generate_models_from_proto"]; ok {
		if genFromProto, isBool := v.(bool); isBool {
			c.generateModelsFromProto = genFromProto
		}
	}
	for _, lang := range []string{"python", "go", "rust", "cpp", "typescript"} {
		if v, ok := p.config.Options[lang+"_orm"]; ok {
			if enabled, isBool := v.(bool); isBool && enabled {
				cfg := langORMCfg{enabled: true}
				if plugin, ok := p.config.Options[lang+"_orm_plugin"].(string); ok {
					cfg.ormPlugin = plugin
				}
				if dir, ok := p.config.Options[lang+"_models_output"].(string); ok {
					cfg.outputDir = dir
				}
				if prefix, ok := p.config.Options[lang+"_pb2_import_prefix"].(string); ok {
					cfg.pb2ImportPrefix = prefix
				}
				c.langs[lang] = cfg
			}
		}
	}
	return c
}

func (c *pluginModelsConfig) isEnabled(lang string) bool {
	if l, ok := c.langs[lang]; ok {
		return l.enabled
	}
	return false
}

func (c *pluginModelsConfig) getORM(lang string) ORMPlugin {
	if l, ok := c.langs[lang]; ok && l.ormPlugin != "" {
		return ParseORMPlugin(l.ormPlugin)
	}
	return ORMPlugin{Name: "None"}
}

func (c *pluginModelsConfig) getOutputDir(lang, fallback string) string {
	if l, ok := c.langs[lang]; ok && l.outputDir != "" {
		return l.outputDir
	}
	return filepath.Join(fallback, "models")
}

func (c *pluginModelsConfig) getPb2ImportPrefix(lang string) string {
	if l, ok := c.langs[lang]; ok {
		return l.pb2ImportPrefix
	}
	return ""
}

func (p *ModelsPlugin) generateForLanguage(
	lang string,
	orm ORMPlugin,
	outputDir string,
	pb2ImportPrefix string,
	models []ModelDef,
	input *plugin.Input,
) ([]string, error) {
	gen, err := NewModelCodeGenerator(lang, orm)
	if err != nil {
		return nil, err
	}

	// Ensure output dir
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", outputDir, err)
	}

	opts := GenerateOptions{
		Language:        lang,
		ORM:             orm,
		OutputDir:       outputDir,
		Pb2ImportPrefix: pb2ImportPrefix,
	}
	// Derive package from first model
	if len(models) > 0 {
		opts.Package = models[0].Package
	}

	var generatedPaths []string

	// 1. Base model
	base, err := gen.GenerateBaseModel(opts)
	if err != nil {
		return nil, fmt.Errorf("base model: %w", err)
	}
	if base.Content != "" {
		p := filepath.Join(outputDir, base.Path)
		if err := writeFile(p, base.Content); err != nil {
			return nil, err
		}
		generatedPaths = append(generatedPaths, p)
	}

	// 2. Each model
	for _, model := range models {
		if model.Abstract || !shouldGenerateModel(model) {
			continue
		}
		files, err := gen.GenerateModel(model, opts)
		if err != nil {
			return nil, fmt.Errorf("model %s: %w", model.MessageName, err)
		}
		for _, f := range files {
			p := filepath.Join(outputDir, f.Path)
			if err := writeFile(p, f.Content); err != nil {
				return nil, err
			}
			generatedPaths = append(generatedPaths, p)
		}
	}

	// 3. Init file (Python __init__.py, Rust mod.rs)
	init, err := gen.GenerateInit(models, opts)
	if err != nil {
		return nil, fmt.Errorf("init: %w", err)
	}
	if init.Content != "" {
		p := filepath.Join(outputDir, init.Path)
		if err := writeFile(p, init.Content); err != nil {
			return nil, err
		}
		generatedPaths = append(generatedPaths, p)
	}

	return generatedPaths, nil
}

// writeFile writes content to a file, creating parent directories as needed.
func writeFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// shouldGenerateModel returns true if the model should be generated.
// If Generate is empty, all models are generated.
// If Generate lists specific types (e.g. "model", "repo"), "model" must be present.
func shouldGenerateModel(m ModelDef) bool {
	if len(m.Generate) == 0 {
		return true
	}
	for _, g := range m.Generate {
		if g == "model" {
			return true
		}
	}
	return false
}

// ──────────────────────────────────────────────────────────────────
//  Standalone generation (for CLI)
// ──────────────────────────────────────────────────────────────────

// GenerateModels is a standalone entry point for the CLI.
// It reads proto files, parses models, and generates code for the specified language.
// When fromProto is true, ALL messages are extracted (not just annotated ones).
// pb2ImportPrefix is an optional dotted prefix prepended to pb2 imports (Python only).
func GenerateModels(protoFiles []string, lang, ormRaw, outputDir, pkg, pb2ImportPrefix string, fromProto ...bool) ([]string, error) {
	orm := ParseORMPlugin(ormRaw)

	gen, err := NewModelCodeGenerator(lang, orm)
	if err != nil {
		return nil, err
	}

	useFromProto := len(fromProto) > 0 && fromProto[0]

	// Parse all models
	var allModels []ModelDef
	var allTopLevelEnums []EnumDef
	for _, pf := range protoFiles {
		content, err := os.ReadFile(pf)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", pf, err)
		}
		var models []ModelDef
		if useFromProto {
			models, err = ExtractAllMessages(string(content), pf)
		} else {
			models, err = ExtractModels(string(content), pf)
		}
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", pf, err)
		}
		allModels = append(allModels, models...)

		// Extract top-level enums (outside of message blocks)
		topEnums := ExtractTopLevelEnums(string(content), pf)
		allTopLevelEnums = append(allTopLevelEnums, topEnums...)
	}

	if len(allModels) == 0 {
		return nil, nil
	}

	// ── Detect file name collisions across packages ──
	fileNameSources := map[string][]string{} // snake_case name → proto files
	for _, m := range allModels {
		if m.Abstract || !shouldGenerateModel(m) {
			continue
		}
		sn := toSnakeCase(m.MessageName)
		fileNameSources[sn] = append(fileNameSources[sn], m.FilePath)
	}
	for name, sources := range fileNameSources {
		if len(sources) > 1 {
			fmt.Fprintf(os.Stderr, "WARNING: file name collision for '%s' from: %s (last one wins)\n",
				name, strings.Join(sources, ", "))
		}
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, err
	}

	opts := GenerateOptions{
		Language:        lang,
		ORM:             orm,
		OutputDir:       outputDir,
		Package:         pkg,
		Pb2ImportPrefix: pb2ImportPrefix,
	}

	var paths []string

	// Base model
	base, err := gen.GenerateBaseModel(opts)
	if err != nil {
		return nil, err
	}
	if base.Content != "" {
		p := filepath.Join(outputDir, base.Path)
		if err := writeFile(p, base.Content); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}

	// Each model
	for _, m := range allModels {
		if m.Abstract || !shouldGenerateModel(m) {
			continue
		}
		files, err := gen.GenerateModel(m, opts)
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			fp := filepath.Join(outputDir, f.Path)
			if err := writeFile(fp, f.Content); err != nil {
				return nil, err
			}
			paths = append(paths, fp)
		}
	}

	// Top-level enums (standalone enum files)
	for _, e := range allTopLevelEnums {
		ef, err := gen.GenerateEnum(e, opts)
		if err != nil {
			return nil, err
		}
		if ef.Content != "" {
			fp := filepath.Join(outputDir, ef.Path)
			if err := writeFile(fp, ef.Content); err != nil {
				return nil, err
			}
			paths = append(paths, fp)
		}
	}

	// Init
	init, err := gen.GenerateInit(allModels, opts)
	if err != nil {
		return nil, err
	}
	if init.Content != "" {
		p := filepath.Join(outputDir, init.Path)
		if err := writeFile(p, init.Content); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}

	return paths, nil
}

// ListModelAnnotations scans proto files and returns parsed ModelDef without generating.
// When fromProto is true, ALL messages are returned (not just annotated ones).
func ListModelAnnotations(protoFiles []string, fromProto ...bool) ([]ModelDef, error) {
	useFromProto := len(fromProto) > 0 && fromProto[0]

	var all []ModelDef
	for _, pf := range protoFiles {
		content, err := os.ReadFile(pf)
		if err != nil {
			return nil, err
		}
		var models []ModelDef
		if useFromProto {
			models, err = ExtractAllMessages(string(content), pf)
		} else {
			models, err = ExtractModels(string(content), pf)
		}
		if err != nil {
			return nil, err
		}
		all = append(all, models...)
	}
	return all, nil
}

// CheckORMDependencies validates that ORM dependencies are available.
func CheckORMDependencies(lang, ormRaw string) []string {
	orm := ParseORMPlugin(ormRaw)
	if orm.IsNone() {
		return nil
	}

	var warnings []string

	switch lang {
	case "python":
		switch orm.Name {
		case "pydantic":
			warnings = append(warnings,
				fmt.Sprintf("Ensure 'pydantic' is installed: pip install pydantic%s", pydanticVersionHint(orm.Version)))
		case "sqlalchemy":
			warnings = append(warnings,
				fmt.Sprintf("Ensure 'sqlalchemy' is installed: pip install sqlalchemy%s", versionHint(orm.Version)))
		}
	case "go":
		switch orm.Name {
		case "gorm":
			warnings = append(warnings, "Ensure 'gorm.io/gorm' is in go.mod")
		case "sqlx":
			warnings = append(warnings, "Ensure 'github.com/jmoiron/sqlx' is in go.mod")
		}
	case "rust":
		if orm.Name == "diesel" {
			warnings = append(warnings, "Ensure 'diesel' is in Cargo.toml dependencies")
		}
	}

	return warnings
}

func pydanticVersionHint(version string) string {
	if version == "" {
		return ""
	}
	if strings.HasPrefix(version, "2") {
		return ">=2.0"
	}
	return fmt.Sprintf(">=%s", version)
}

func versionHint(version string) string {
	if version == "" {
		return ""
	}
	return fmt.Sprintf(">=%s", version)
}
