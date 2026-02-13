package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/massonsky/buffalo/internal/models"
	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/spf13/cobra"
)

// ══════════════════════════════════════════════════════════════════
//  buffalo models  —  Code model generation commands
// ══════════════════════════════════════════════════════════════════

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Manage buffalo.models code generation",
	Long: `Buffalo models commands for generating typed code models from proto annotations.

Using buffalo.models annotations you can produce typed models for Go, Python,
Rust, and C++ with optional ORM/framework integration (pydantic, gorm, diesel, etc.).`,
	Aliases: []string{"model", "mdl"},
}

// ──────────────────────────────────────────────────────────────────
//  buffalo models generate
// ──────────────────────────────────────────────────────────────────

var modelsGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate code models from proto annotations",
	Long: `Generate typed code models from buffalo.models proto annotations.

Reads proto files, extracts buffalo.models annotations and generates
model source files for the specified language and ORM framework.`,
	Example: `  # Generate Python pydantic models
  buffalo models generate --lang python --orm pydantic --output gen/models

  # Generate Go structs with GORM tags
  buffalo models generate --lang go --orm gorm --output internal/models

  # Generate Rust structs with Diesel derives
  buffalo models generate --lang rust --orm diesel --output src/models

  # Generate plain C++ structs
  buffalo models generate --lang cpp --output include/models

  # Generate for all languages using buffalo.yaml config
  buffalo models generate --all`,
	Aliases: []string{"gen", "g"},
	RunE: func(cmd *cobra.Command, args []string) error {
		protoDir, _ := cmd.Flags().GetString("proto")
		lang, _ := cmd.Flags().GetString("lang")
		ormPlugin, _ := cmd.Flags().GetString("orm")
		outputDir, _ := cmd.Flags().GetString("output")
		pkg, _ := cmd.Flags().GetString("package")
		all, _ := cmd.Flags().GetBool("all")

		if all {
			return runGenerateAll(protoDir)
		}

		if lang == "" {
			return fmt.Errorf("--lang is required (python, go, rust, cpp) or use --all")
		}

		// Find proto files
		protoFiles, err := findProtoFiles(protoDir)
		if err != nil {
			return err
		}
		if len(protoFiles) == 0 {
			log.Warn("No .proto files found", logger.String("dir", protoDir))
			return nil
		}

		// Default output
		if outputDir == "" {
			outputDir = filepath.Join("generated", "models", lang)
		}
		// Default ORM
		if ormPlugin == "" {
			ormPlugin = "None"
		}

		log.Info("Generating models",
			logger.String("lang", lang),
			logger.String("orm", ormPlugin),
			logger.String("output", outputDir),
			logger.Int("protos", len(protoFiles)),
		)

		paths, err := models.GenerateModels(protoFiles, lang, ormPlugin, outputDir, pkg)
		if err != nil {
			return fmt.Errorf("generation failed: %w", err)
		}

		for _, p := range paths {
			fmt.Printf("  ✓ %s\n", p)
		}
		log.Info("Models generated", logger.Int("files", len(paths)))
		return nil
	},
}

func runGenerateAll(protoDir string) error {
	protoFiles, err := findProtoFiles(protoDir)
	if err != nil {
		return err
	}
	if len(protoFiles) == 0 {
		log.Warn("No .proto files found", logger.String("dir", protoDir))
		return nil
	}

	// Read buffalo.yaml config for per-language ORM settings
	cfg, cfgErr := loadModelsConfig()
	if cfgErr != nil {
		log.Warn("Config load warning, using defaults", logger.Any("error", cfgErr))
	}

	languages := []string{"python", "go", "rust", "cpp"}
	totalFiles := 0
	for _, lang := range languages {
		enabled := false
		ormPlugin := "None"
		outputDir := filepath.Join("generated", "models", lang)

		if cfg != nil {
			enabled = cfg.IsORMEnabled(lang)
			ormPlugin = cfg.GetORMPlugin(lang)
			if d := cfg.GetModelsOutputDir(lang); d != "" {
				outputDir = d
			}
		}

		if !enabled {
			continue
		}

		log.Info("Generating models", logger.String("lang", lang), logger.String("orm", ormPlugin), logger.String("output", outputDir))

		paths, err := models.GenerateModels(protoFiles, lang, ormPlugin, outputDir, "")
		if err != nil {
			log.Error("Generation failed", logger.String("lang", lang), logger.Any("error", err))
			continue
		}

		for _, p := range paths {
			fmt.Printf("  ✓ %s\n", p)
		}
		totalFiles += len(paths)
	}

	log.Info("All models generated", logger.Int("total_files", totalFiles))
	return nil
}

// ──────────────────────────────────────────────────────────────────
//  buffalo models list
// ──────────────────────────────────────────────────────────────────

var modelsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List model annotations in proto files",
	Long:  `Scan proto files and list all buffalo.models annotations found.`,
	Example: `  buffalo models list
  buffalo models list --proto ./protos`,
	Aliases: []string{"ls", "l"},
	RunE: func(cmd *cobra.Command, args []string) error {
		protoDir, _ := cmd.Flags().GetString("proto")
		protoFiles, err := findProtoFiles(protoDir)
		if err != nil {
			return err
		}

		allModels, err := models.ListModelAnnotations(protoFiles)
		if err != nil {
			return err
		}

		if len(allModels) == 0 {
			fmt.Println("No buffalo.models annotations found.")
			return nil
		}

		for _, m := range allModels {
			name := m.EffectiveName()
			extra := ""
			if m.TableName != "" {
				extra = fmt.Sprintf("  table=%s", m.TableName)
			}
			if m.Description != "" {
				extra += fmt.Sprintf("  \"%s\"", m.Description)
			}
			deprecated := ""
			if m.Deprecated {
				deprecated = " [DEPRECATED]"
			}
			fmt.Printf("  • %s  (%d fields)%s%s  [%s]\n",
				name, len(m.Fields), extra, deprecated, m.FilePath)
		}
		return nil
	},
}

// ──────────────────────────────────────────────────────────────────
//  buffalo models inspect
// ──────────────────────────────────────────────────────────────────

var modelsInspectCmd = &cobra.Command{
	Use:   "inspect [model]",
	Short: "Inspect a specific model in detail",
	Long:  `Show detailed information about a model including all fields, visibility, behavior, and relations.`,
	Example: `  buffalo models inspect User
  buffalo models inspect --proto ./protos`,
	RunE: func(cmd *cobra.Command, args []string) error {
		protoDir, _ := cmd.Flags().GetString("proto")
		protoFiles, err := findProtoFiles(protoDir)
		if err != nil {
			return err
		}

		allModels, err := models.ListModelAnnotations(protoFiles)
		if err != nil {
			return err
		}

		filter := ""
		if len(args) > 0 {
			filter = args[0]
		}

		for _, m := range allModels {
			name := m.EffectiveName()
			if filter != "" && !strings.EqualFold(name, filter) && !strings.EqualFold(m.MessageName, filter) {
				continue
			}

			fmt.Printf("Model: %s\n", name)
			if m.Description != "" {
				fmt.Printf("  Description: %s\n", m.Description)
			}
			if m.TableName != "" {
				fmt.Printf("  Table: %s\n", m.TableName)
			}
			if m.Schema != "" {
				fmt.Printf("  Schema: %s\n", m.Schema)
			}
			if m.Extends != "" {
				fmt.Printf("  Extends: %s\n", m.Extends)
			}
			if m.Deprecated {
				fmt.Printf("  ⚠ DEPRECATED: %s\n", m.DeprecatedMessage)
			}
			fmt.Printf("  Fields (%d):\n", len(m.Fields))
			for _, f := range m.Fields {
				typ := f.ProtoType
				if f.Repeated {
					typ = "[]" + typ
				}
				if f.Nullable {
					typ = "?" + typ
				}

				vis := ""
				if f.Visibility != models.VisibilityDefault {
					vis = fmt.Sprintf(" [%s]", f.Visibility.String())
				}
				beh := ""
				if f.Behavior != models.BehaviorDefault {
					beh = fmt.Sprintf(" [%s]", f.Behavior.String())
				}
				sens := ""
				if f.Sensitive {
					sens = " 🔒"
				}
				dep := ""
				if f.Deprecated {
					dep = " ⚠DEPRECATED"
				}

				fmt.Printf("    %s : %s%s%s%s%s\n", f.Name, typ, vis, beh, sens, dep)
			}
			fmt.Println()
		}
		return nil
	},
}

// ──────────────────────────────────────────────────────────────────
//  buffalo models check-deps
// ──────────────────────────────────────────────────────────────────

var modelsCheckDepsCmd = &cobra.Command{
	Use:   "check-deps",
	Short: "Check ORM dependencies for model generation",
	Long:  `Verify that required ORM/framework dependencies are available for model code generation.`,
	Example: `  buffalo models check-deps --lang python --orm pydantic
  buffalo models check-deps --lang go --orm gorm`,
	RunE: func(cmd *cobra.Command, args []string) error {
		lang, _ := cmd.Flags().GetString("lang")
		orm, _ := cmd.Flags().GetString("orm")

		if lang == "" {
			return fmt.Errorf("--lang is required")
		}
		if orm == "" {
			orm = "None"
		}

		warnings := models.CheckORMDependencies(lang, orm)
		if len(warnings) == 0 {
			fmt.Printf("✓ No dependency issues for %s/%s\n", lang, orm)
			return nil
		}

		for _, w := range warnings {
			fmt.Printf("  ⚠ %s\n", w)
		}
		return nil
	},
}

// ──────────────────────────────────────────────────────────────────
//  Helpers
// ──────────────────────────────────────────────────────────────────

// findProtoFiles locates all .proto files in a directory tree.
func findProtoFiles(dir string) ([]string, error) {
	if dir == "" {
		dir = "."
	}
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("proto dir %s: %w", dir, err)
	}
	if !info.IsDir() {
		return []string{dir}, nil
	}

	var files []string
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".proto") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// loadModelsConfig loads buffalo config for ORM settings.
func loadModelsConfig() (*modelsConfigWrapper, error) {
	paths := []string{"buffalo.yaml", "buffalo.yml"}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			cfg, err := loadBuffaloConfigForModels(p)
			if err != nil {
				return nil, err
			}
			return cfg, nil
		}
	}
	return nil, fmt.Errorf("buffalo.yaml not found")
}

// modelsConfigWrapper is a lightweight wrapper around config for models CLI.
type modelsConfigWrapper struct {
	python modelsLangCfg
	golang modelsLangCfg
	rust   modelsLangCfg
	cpp    modelsLangCfg
}

type modelsLangCfg struct {
	orm          bool
	ormPlugin    string
	modelsOutput string
}

func (w *modelsConfigWrapper) IsORMEnabled(lang string) bool {
	switch lang {
	case "python":
		return w.python.orm
	case "go":
		return w.golang.orm
	case "rust":
		return w.rust.orm
	case "cpp":
		return w.cpp.orm
	}
	return false
}

func (w *modelsConfigWrapper) GetORMPlugin(lang string) string {
	switch lang {
	case "python":
		return w.python.ormPlugin
	case "go":
		return w.golang.ormPlugin
	case "rust":
		return w.rust.ormPlugin
	case "cpp":
		return w.cpp.ormPlugin
	}
	return "None"
}

func (w *modelsConfigWrapper) GetModelsOutputDir(lang string) string {
	switch lang {
	case "python":
		return w.python.modelsOutput
	case "go":
		return w.golang.modelsOutput
	case "rust":
		return w.rust.modelsOutput
	case "cpp":
		return w.cpp.modelsOutput
	}
	return ""
}

func loadBuffaloConfigForModels(path string) (*modelsConfigWrapper, error) {
	// Use the standard config loader
	cfg, err := configLoadFunc(path)
	if err != nil {
		return nil, err
	}
	w := &modelsConfigWrapper{
		python: modelsLangCfg{
			orm:          cfg.IsORMEnabled("python"),
			ormPlugin:    cfg.GetORMPlugin("python"),
			modelsOutput: cfg.GetModelsOutputDir("python"),
		},
		golang: modelsLangCfg{
			orm:          cfg.IsORMEnabled("go"),
			ormPlugin:    cfg.GetORMPlugin("go"),
			modelsOutput: cfg.GetModelsOutputDir("go"),
		},
		rust: modelsLangCfg{
			orm:          cfg.IsORMEnabled("rust"),
			ormPlugin:    cfg.GetORMPlugin("rust"),
			modelsOutput: cfg.GetModelsOutputDir("rust"),
		},
		cpp: modelsLangCfg{
			orm:          cfg.IsORMEnabled("cpp"),
			ormPlugin:    cfg.GetORMPlugin("cpp"),
			modelsOutput: cfg.GetModelsOutputDir("cpp"),
		},
	}
	return w, nil
}

// configLoadFunc is assigned at init to break import cycles.
var configLoadFunc = defaultConfigLoadFunc

func defaultConfigLoadFunc(path string) (configInterface, error) {
	c, err := configLoad(path)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// configInterface abstracts config.Config for CLI use.
type configInterface interface {
	IsORMEnabled(lang string) bool
	GetORMPlugin(lang string) string
	GetModelsOutputDir(lang string) string
}

// configLoad is set at init to avoid hard import of config package.
var configLoad func(string) (configInterface, error)

func init() {
	// Register subcommands
	modelsCmd.AddCommand(modelsGenerateCmd)
	modelsCmd.AddCommand(modelsListCmd)
	modelsCmd.AddCommand(modelsInspectCmd)
	modelsCmd.AddCommand(modelsCheckDepsCmd)

	// Register with root
	rootCmd.AddCommand(modelsCmd)

	// Common flags for proto directory
	modelsCmd.PersistentFlags().StringP("proto", "p", ".", "Proto files directory")

	// Generate flags
	modelsGenerateCmd.Flags().StringP("lang", "l", "", "Target language (python, go, rust, cpp)")
	modelsGenerateCmd.Flags().StringP("orm", "r", "", "ORM framework (pydantic, gorm, diesel, sqlalchemy, sqlx, None)")
	modelsGenerateCmd.Flags().StringP("output", "o", "", "Output directory for generated models")
	modelsGenerateCmd.Flags().String("package", "", "Package name for generated code")
	modelsGenerateCmd.Flags().Bool("all", false, "Generate for all configured languages from buffalo.yaml")

	// Check-deps flags
	modelsCheckDepsCmd.Flags().StringP("lang", "l", "", "Target language")
	modelsCheckDepsCmd.Flags().StringP("orm", "r", "", "ORM framework")
}
