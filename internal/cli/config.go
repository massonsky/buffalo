package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/massonsky/buffalo/internal/config"
	"github.com/massonsky/buffalo/pkg/errors"
	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	configCmd = &cobra.Command{
		Use:   "config",
		Short: "Manage Buffalo configuration",
		Long: `Manage and validate Buffalo configuration files.

This command provides tools for working with buffalo.yaml configuration files,
including validation, inspection, and initialization.`,
		Example: `  # Validate current configuration
  buffalo config validate

  # Show configuration summary
  buffalo config show

  # Initialize new configuration
  buffalo config init`,
	}

	configValidateCmd = &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file",
		Long: `Validate buffalo.yaml configuration file.

Checks for:
  - Valid YAML syntax
  - Required fields presence
  - Path existence
  - Language configuration
  - Plugin configuration
  - Template configuration`,
		RunE: runConfigValidate,
	}

	configShowCmd = &cobra.Command{
		Use:   "show",
		Short: "Show configuration summary",
		Long:  "Display current configuration in a readable format",
		RunE:  runConfigShow,
	}

	configInitCmd = &cobra.Command{
		Use:   "init",
		Short: "Initialize new configuration",
		Long:  "Create a new buffalo.yaml configuration file with sensible defaults",
		RunE:  runConfigInit,
	}

	configSchemaCmd = &cobra.Command{
		Use:   "schema",
		Short: "Emit JSON Schema or default YAML for buffalo.yaml",
		Long: `Emit machine-readable artifacts describing the Buffalo configuration.

Useful for IDE integration (point your editor at the JSON Schema for inline
validation and autocomplete) and for regenerating configs/default.yaml from
the single source of truth (the Config struct in internal/config).`,
		Example: `  # Print JSON Schema to stdout
  buffalo config schema --format=json

  # Emit a YAML config for a given profile
  buffalo config schema --format=yaml --profile=full`,
		RunE: runConfigSchema,
	}

	configMigrateCmd = &cobra.Command{
		Use:   "migrate",
		Short: "Upgrade buffalo.yaml to the current schema version",
		Long: `Apply schema migrations in-place to bring buffalo.yaml up to the current schema_version.

This is YAML-preserving (comments and key order are kept). Use --dry-run to
print the migrated file to stdout without modifying the source.`,
		Example: `  buffalo config migrate -c buffalo.yaml
  buffalo config migrate -c buffalo.yaml --dry-run`,
		RunE: runConfigMigrate,
	}

	configFile    string
	configForce   bool
	configFormat  string
	configProfile string
	configDryRun  bool
)

func init() {
	rootCmd.AddCommand(configCmd)

	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configSchemaCmd)
	configCmd.AddCommand(configMigrateCmd)

	// Validate command flags
	configValidateCmd.Flags().StringVarP(&configFile, "config", "c", "buffalo.yaml", "configuration file to validate")
	configValidateCmd.Flags().StringVar(&configFormat, "format", "text", "output format: text, json")

	// Show command flags
	configShowCmd.Flags().StringVarP(&configFile, "config", "c", "buffalo.yaml", "configuration file to show")
	configShowCmd.Flags().StringVar(&configFormat, "format", "text", "output format: text, yaml, json")

	// Init command flags
	configInitCmd.Flags().BoolVarP(&configForce, "force", "f", false, "overwrite existing configuration")
	configInitCmd.Flags().StringVar(&configProfile, "profile", "full", "config profile: minimal, full, bazel")

	// Schema command flags
	configSchemaCmd.Flags().StringVar(&configFormat, "format", "json", "output format: json (JSON Schema) or yaml (default config)")
	configSchemaCmd.Flags().StringVar(&configProfile, "profile", "full", "YAML profile when --format=yaml: minimal, full, bazel")

	// Migrate command flags
	configMigrateCmd.Flags().StringVarP(&configFile, "config", "c", "buffalo.yaml", "configuration file to migrate")
	configMigrateCmd.Flags().BoolVar(&configDryRun, "dry-run", false, "print the migrated file to stdout without modifying the source")
}

// ValidationIssue represents a configuration validation issue
type ValidationIssue struct {
	Severity   string // "error", "warning", "info"
	Field      string
	Message    string
	Suggestion string
}

func runConfigValidate(cmd *cobra.Command, args []string) error {
	log := GetLogger()

	log.Info("🔍 Validating configuration", logger.String("file", configFile))

	// Check if file exists
	absPath, err := filepath.Abs(configFile)
	if err != nil {
		return fmt.Errorf("invalid config path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		log.Error("Configuration file not found", logger.String("path", absPath))
		log.Info("\nTo create a new configuration, run: buffalo config init")
		return errors.New(errors.ErrConfig, "configuration file not found")
	}

	// Read and parse YAML
	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Validate YAML syntax
	var rawConfig map[string]interface{}
	if err := yaml.Unmarshal(data, &rawConfig); err != nil {
		log.Error("❌ Invalid YAML syntax", logger.Any("error", err))
		return errors.Wrap(err, errors.ErrConfig, "invalid YAML syntax")
	}

	log.Info("✅ Valid YAML syntax")

	// Load config from the specific file (not from CWD-bound viper).
	cfg, err := config.LoadFromFile(absPath)
	if err != nil {
		log.Error("❌ Configuration validation failed", logger.Any("error", err))
		return err
	}

	// Collect validation issues
	issues := []ValidationIssue{}

	// Validate project section
	if cfg.Project.Name == "" {
		issues = append(issues, ValidationIssue{
			Severity:   "warning",
			Field:      "project.name",
			Message:    "Project name is not set",
			Suggestion: "Add 'project.name' to identify your project",
		})
	}

	// Validate proto paths
	if len(cfg.Proto.Paths) == 0 {
		issues = append(issues, ValidationIssue{
			Severity:   "error",
			Field:      "proto.paths",
			Message:    "No proto paths defined",
			Suggestion: "Add paths to your .proto files in 'proto.paths'",
		})
	} else {
		for _, path := range cfg.Proto.Paths {
			absProtoPath := path
			if !filepath.IsAbs(path) {
				absProtoPath = filepath.Join(filepath.Dir(absPath), path)
			}
			if _, err := os.Stat(absProtoPath); os.IsNotExist(err) {
				issues = append(issues, ValidationIssue{
					Severity:   "error",
					Field:      "proto.paths",
					Message:    fmt.Sprintf("Proto path does not exist: %s", path),
					Suggestion: "Create the directory or update the path",
				})
			}
		}
	}

	// Validate output configuration
	if cfg.Output.BaseDir == "" {
		issues = append(issues, ValidationIssue{
			Severity:   "error",
			Field:      "output.base_dir",
			Message:    "Output base directory is not set",
			Suggestion: "Add 'output.base_dir' to specify where generated files should go",
		})
	}

	// Validate languages
	enabledLangs := cfg.GetEnabledLanguages()
	if len(enabledLangs) == 0 {
		issues = append(issues, ValidationIssue{
			Severity:   "warning",
			Field:      "languages",
			Message:    "No languages enabled",
			Suggestion: "Enable at least one language (python, go, rust, cpp, typescript)",
		})
	}

	// Validate Go-specific settings
	if cfg.Languages.Go.Enabled {
		if cfg.Languages.Go.Module == "" {
			issues = append(issues, ValidationIssue{
				Severity:   "warning",
				Field:      "languages.go.module",
				Message:    "Go module path is not set",
				Suggestion: "Add 'languages.go.module' for proper import paths",
			})
		}
	}

	// Validate build settings
	if cfg.Build.Workers <= 0 {
		issues = append(issues, ValidationIssue{
			Severity:   "info",
			Field:      "build.workers",
			Message:    "Build workers not set, will use default",
			Suggestion: "Set 'build.workers' for optimal parallelism",
		})
	}

	// Validate plugins
	for i, plugin := range cfg.Plugins {
		if plugin.Name == "" {
			issues = append(issues, ValidationIssue{
				Severity:   "error",
				Field:      fmt.Sprintf("plugins[%d].name", i),
				Message:    "Plugin name is empty",
				Suggestion: "Add a name for the plugin",
			})
		}
	}

	// Validate templates
	for i, tmpl := range cfg.Templates {
		if tmpl.Name == "" {
			issues = append(issues, ValidationIssue{
				Severity:   "error",
				Field:      fmt.Sprintf("templates[%d].name", i),
				Message:    "Template name is empty",
				Suggestion: "Add a name for the template",
			})
		}
		if tmpl.Path != "" {
			absTmplPath := tmpl.Path
			if !filepath.IsAbs(tmpl.Path) {
				absTmplPath = filepath.Join(filepath.Dir(absPath), tmpl.Path)
			}
			if _, err := os.Stat(absTmplPath); os.IsNotExist(err) {
				issues = append(issues, ValidationIssue{
					Severity:   "warning",
					Field:      fmt.Sprintf("templates[%d].path", i),
					Message:    fmt.Sprintf("Template path does not exist: %s", tmpl.Path),
					Suggestion: "Create the template directory or update the path",
				})
			}
		}
	}

	// Print results
	errorCount := 0
	warningCount := 0
	infoCount := 0

	if len(issues) > 0 {
		log.Info("\n📋 Validation Issues:\n")

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, issue := range issues {
			icon := "ℹ️"
			switch issue.Severity {
			case "error":
				icon = "❌"
				errorCount++
			case "warning":
				icon = "⚠️"
				warningCount++
			case "info":
				icon = "ℹ️"
				infoCount++
			}

			fmt.Fprintf(w, "%s\t[%s]\t%s\n", icon, issue.Field, issue.Message)
			if issue.Suggestion != "" {
				fmt.Fprintf(w, "\t\t  💡 %s\n", issue.Suggestion)
			}
		}
		w.Flush()
	}

	// Summary
	log.Info("")
	if errorCount > 0 {
		log.Error(fmt.Sprintf("❌ Validation failed: %d error(s), %d warning(s)", errorCount, warningCount))
		return errors.New(errors.ErrConfig, fmt.Sprintf("configuration has %d error(s)", errorCount))
	} else if warningCount > 0 {
		log.Warn(fmt.Sprintf("⚠️  Validation passed with %d warning(s)", warningCount))
	} else {
		log.Info("✅ Configuration is valid!")
	}

	// Show enabled languages
	if len(enabledLangs) > 0 {
		log.Info(fmt.Sprintf("\n📝 Enabled languages: %s", strings.Join(enabledLangs, ", ")))
	}

	return nil
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	log := GetLogger()

	// Check if file exists
	absPath, err := filepath.Abs(configFile)
	if err != nil {
		return fmt.Errorf("invalid config path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		log.Error("Configuration file not found", logger.String("path", absPath))
		return errors.New(errors.ErrConfig, "configuration file not found")
	}

	// Load config
	cfg, err := loadConfig(log)
	if err != nil {
		return err
	}

	// Output based on format
	switch configFormat {
	case "yaml":
		data, _ := os.ReadFile(absPath)
		fmt.Println(string(data))
	case "json":
		// TODO: JSON output
		log.Info("JSON format not yet implemented")
	default:
		// Text format - summary
		log.Info("📋 Buffalo Configuration Summary\n")
		log.Info(fmt.Sprintf("  File: %s", absPath))
		log.Info("")

		// Project
		log.Info("📦 Project:")
		log.Info(fmt.Sprintf("  Name: %s", cfg.Project.Name))
		log.Info(fmt.Sprintf("  Version: %s", cfg.Project.Version))
		log.Info("")

		// Proto
		log.Info("📄 Proto:")
		log.Info(fmt.Sprintf("  Paths: %v", cfg.Proto.Paths))
		log.Info(fmt.Sprintf("  Import Paths: %v", cfg.Proto.ImportPaths))
		log.Info(fmt.Sprintf("  Exclude: %v", cfg.Proto.Exclude))
		log.Info("")

		// Output
		log.Info("📤 Output:")
		log.Info(fmt.Sprintf("  Base Dir: %s", cfg.Output.BaseDir))
		log.Info(fmt.Sprintf("  Preserve Structure: %v", cfg.Output.PreserveProtoStructure))
		log.Info("")

		// Languages
		log.Info("🌐 Languages:")
		log.Info(fmt.Sprintf("  Python: enabled=%v, package=%s", cfg.Languages.Python.Enabled, cfg.Languages.Python.Package))
		log.Info(fmt.Sprintf("  Go: enabled=%v, module=%s", cfg.Languages.Go.Enabled, cfg.Languages.Go.Module))
		log.Info(fmt.Sprintf("  Rust: enabled=%v", cfg.Languages.Rust.Enabled))
		log.Info(fmt.Sprintf("  C++: enabled=%v", cfg.Languages.Cpp.Enabled))
		log.Info(fmt.Sprintf("  TypeScript: enabled=%v, generator=%s", cfg.Languages.Typescript.Enabled, cfg.Languages.Typescript.Generator))
		log.Info("")

		// Build
		log.Info("🔨 Build:")
		log.Info(fmt.Sprintf("  Workers: %d", cfg.Build.Workers))
		log.Info(fmt.Sprintf("  Incremental: %v", cfg.Build.Incremental))
		log.Info(fmt.Sprintf("  Cache: enabled=%v, dir=%s", cfg.Build.Cache.Enabled, cfg.Build.Cache.Directory))
		log.Info("")

		// Plugins
		if len(cfg.Plugins) > 0 {
			log.Info("🔌 Plugins:")
			for _, p := range cfg.Plugins {
				log.Info(fmt.Sprintf("  - %s (enabled=%v, priority=%d)", p.Name, p.Enabled, p.Priority))
			}
			log.Info("")
		}

		// Templates
		if len(cfg.Templates) > 0 {
			log.Info("📋 Templates:")
			for _, t := range cfg.Templates {
				log.Info(fmt.Sprintf("  - %s (language=%s, path=%s)", t.Name, t.Language, t.Path))
			}
			log.Info("")
		}
	}

	return nil
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	log := GetLogger()

	targetFile := "buffalo.yaml"
	if configFile != "" && configFile != "buffalo.yaml" {
		targetFile = configFile
	}

	absPath, err := filepath.Abs(targetFile)
	if err != nil {
		return fmt.Errorf("invalid config path: %w", err)
	}

	if _, err := os.Stat(absPath); err == nil && !configForce {
		log.Error("Configuration file already exists", logger.String("path", absPath))
		log.Info("\nUse --force to overwrite")
		return errors.New(errors.ErrConfig, "configuration file already exists")
	}

	profile := config.Profile(configProfile)
	if profile == "" {
		profile = config.ProfileFull
	}
	data, err := config.MarshalDefaultYAML(profile)
	if err != nil {
		return fmt.Errorf("generate default config: %w", err)
	}

	if err := os.WriteFile(absPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	log.Info("✅ Configuration file created",
		logger.String("path", absPath),
		logger.String("profile", string(profile)))
	log.Info("\nNext steps:")
	log.Info("  1. Edit buffalo.yaml to match your project")
	log.Info("  2. Create your proto files in ./protos")
	log.Info("  3. Run 'buffalo build' to generate code")

	return nil
}

func runConfigSchema(cmd *cobra.Command, args []string) error {
	switch strings.ToLower(configFormat) {
	case "json", "":
		data, err := config.GenerateJSONSchema()
		if err != nil {
			return fmt.Errorf("generate JSON schema: %w", err)
		}
		_, err = os.Stdout.Write(append(data, '\n'))
		return err
	case "yaml", "yml":
		profile := config.Profile(configProfile)
		if profile == "" {
			profile = config.ProfileFull
		}
		data, err := config.MarshalDefaultYAML(profile)
		if err != nil {
			return fmt.Errorf("generate default YAML: %w", err)
		}
		_, err = os.Stdout.Write(data)
		return err
	default:
		return fmt.Errorf("unknown --format=%q (expected json or yaml)", configFormat)
	}
}

func runConfigMigrate(cmd *cobra.Command, args []string) error {
	log := GetLogger()

	absPath, err := filepath.Abs(configFile)
	if err != nil {
		return fmt.Errorf("invalid config path: %w", err)
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return errors.New(errors.ErrConfig, fmt.Sprintf("configuration file not found: %s", absPath))
	}

	out, from, to, err := config.MigrateFile(absPath, configDryRun)
	if err != nil {
		return err
	}

	if from == to {
		log.Info("✅ Already at current schema version",
			logger.String("file", absPath),
			logger.Any("schema_version", to))
		return nil
	}

	if configDryRun {
		log.Info("ℹ️  Dry run — file not modified",
			logger.Any("from", from), logger.Any("to", to))
		_, err = os.Stdout.Write(out)
		return err
	}

	log.Info("✅ Migrated configuration",
		logger.String("file", absPath),
		logger.Any("from", from),
		logger.Any("to", to))
	return nil
}
