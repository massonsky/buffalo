package cli

import (
	"context"
	"fmt"

	"github.com/massonsky/buffalo/internal/builder"
	"github.com/massonsky/buffalo/internal/config"
	"github.com/massonsky/buffalo/internal/dependency"
	"github.com/massonsky/buffalo/internal/embedded"
	"github.com/massonsky/buffalo/internal/plugin"
	"github.com/massonsky/buffalo/internal/system"
	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/massonsky/buffalo/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	buildOutputDir       string
	buildLanguages       []string
	buildProtoPath       []string
	buildDryRun          bool
	buildSkipSystemCheck bool
	buildSkipLock        bool
	buildForceLock       bool

	buildCmd = &cobra.Command{
		Use:   "build",
		Short: "Build protobuf files",
		Long: `Build protobuf files and generate code for specified languages.

Examples:
  # Build with default config
  buffalo build

  # Build for specific languages
  buffalo build --lang python,go

  # Build with custom output directory
  buffalo build --output ./generated

  # Dry run to see what would be built
  buffalo build --dry-run
  
  # Skip system readiness check
  buffalo build --skip-system-check
  
  # Force regenerate lock file
  buffalo build --force-lock
  
  # Skip lock file (build directly from config)
  buffalo build --skip-lock`,
		RunE: runBuild,
	}
)

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().StringVarP(&buildOutputDir, "output", "o", "", "output directory for generated code")
	buildCmd.Flags().StringSliceVarP(&buildLanguages, "lang", "l", []string{}, "target languages (python,go,rust,cpp,typescript)")
	buildCmd.Flags().StringSliceVarP(&buildProtoPath, "proto-path", "p", []string{}, "paths to search for proto files")
	buildCmd.Flags().BoolVar(&buildDryRun, "dry-run", false, "show what would be built without building")
	buildCmd.Flags().BoolVar(&buildSkipSystemCheck, "skip-system-check", false, "skip system readiness check before build")
	buildCmd.Flags().BoolVar(&buildSkipLock, "skip-lock", false, "skip lock file and build directly from config")
	buildCmd.Flags().BoolVar(&buildForceLock, "force-lock", false, "force regenerate lock file")
}

func runBuild(cmd *cobra.Command, args []string) error {
	log := GetLogger()
	ctx := context.Background()

	log.Info("🔨 Starting build process")

	// Load configuration
	cfg, configPath, err := loadConfigWithPath(log)
	if err != nil {
		log.Warn("Failed to load config, using defaults", logger.Any("error", err))
		cfg = getDefaultConfig()
		configPath = "buffalo.yaml"
	}

	// Override config with flags (only if explicitly set)
	if buildOutputDir != "" {
		cfg.Output.BaseDir = buildOutputDir
	}
	if len(buildLanguages) > 0 {
		// Enable specified languages
		enableLanguages(cfg, buildLanguages)
	}
	if len(buildProtoPath) > 0 {
		cfg.Proto.Paths = buildProtoPath
	}

	// Get enabled languages
	languages := cfg.GetEnabledLanguages()
	if len(buildLanguages) > 0 {
		languages = buildLanguages
	}

	if len(languages) == 0 {
		log.Warn("⚠️  No languages enabled. Please enable at least one language in config or use --lang flag")
		return nil
	}

	// Проверка готовности системы (если не пропущена)
	if !buildSkipSystemCheck {
		log.Info("🔍 Checking system readiness...")
		systemChecker := system.NewSystemChecker(cfg)
		sysResults, err := systemChecker.CheckReadiness()
		if err != nil {
			log.Error("Failed to check system readiness", logger.Any("error", err))
			return fmt.Errorf("system check failed: %w", err)
		}

		// Проверяем критичные требования
		criticalMissing := system.GetMissingCritical(sysResults)
		if len(criticalMissing) > 0 {
			log.Error("❌ Критичные требования не выполнены:")
			for _, result := range criticalMissing {
				log.Error(fmt.Sprintf("   • %s: %v", result.Requirement.Name, result.Error))
				if result.InstallCommand != "" {
					log.Info(fmt.Sprintf("     💡 Установка: %s", result.InstallCommand))
				}
			}
			log.Error("")
			log.Error("Пожалуйста, установите недостающие компоненты перед сборкой.")
			log.Info("Используйте 'buffalo doctor --config-only' для подробной диагностики.")
			log.Info("Или запустите с флагом --skip-system-check, чтобы пропустить эту проверку.")
			return fmt.Errorf("system requirements not met")
		}

		// Выводим предупреждения
		hasWarnings := false
		for _, result := range sysResults {
			if !result.Installed && !result.Requirement.Critical {
				if !hasWarnings {
					log.Warn("⚠️  Некоторые опциональные компоненты отсутствуют:")
					hasWarnings = true
				}
				log.Warn(fmt.Sprintf("   • %s", result.Requirement.Name))
			}
		}
		if hasWarnings {
			log.Info("")
		}

		log.Info("✅ Система готова к сборке")
		log.Info("")
	}

	log.Info("Build configuration",
		logger.String("output", cfg.Output.BaseDir),
		logger.Any("languages", languages),
		logger.Any("proto_paths", cfg.Proto.Paths),
		logger.Bool("dry_run", buildDryRun),
		logger.Bool("cache", cfg.Build.Cache.Enabled),
		logger.Int("workers", cfg.Build.Workers),
	)

	// Find proto files
	var allProtoFiles []string
	for _, path := range cfg.Proto.Paths {
		fileInfos, err := utils.FindFiles(path, utils.FindFilesOptions{
			Pattern:   "*.proto",
			Recursive: true,
		})
		if err != nil {
			log.Warn("Failed to find proto files", logger.String("path", path), logger.Any("error", err))
			continue
		}
		for _, fi := range fileInfos {
			allProtoFiles = append(allProtoFiles, fi.Path)
		}
	}

	if len(allProtoFiles) == 0 {
		log.Warn("⚠️  No proto files found in specified paths")
		return nil
	}

	log.Info("Found proto files", logger.Int("count", len(allProtoFiles)))

	// Handle lock file
	var lockFile *config.LockFile
	if !buildSkipLock {
		lockManager := config.NewLockFileManager(configPath)

		needsRegen, reason, err := lockManager.NeedsRegeneration()
		if err != nil {
			log.Warn("Failed to check lock file", logger.Any("error", err))
			needsRegen = true
			reason = "check failed"
		}

		if buildForceLock || needsRegen {
			if buildForceLock {
				log.Info("🔒 Regenerating lock file (forced)")
			} else {
				log.Info("🔒 Regenerating lock file", logger.String("reason", reason))
			}

			lockFile, err = lockManager.Generate(cfg, allProtoFiles)
			if err != nil {
				log.Warn("Failed to generate lock file", logger.Any("error", err))
			} else {
				if err := lockManager.Save(lockFile); err != nil {
					log.Warn("Failed to save lock file", logger.Any("error", err))
				} else {
					log.Info("🔒 Lock file saved", logger.String("path", lockManager.GetLockPath()))
				}
			}
		} else {
			log.Debug("Lock file is up to date")
			lockFile, err = lockManager.Load()
			if err != nil {
				log.Warn("Failed to load lock file, regenerating", logger.Any("error", err))
				lockFile, _ = lockManager.Generate(cfg, allProtoFiles)
				lockManager.Save(lockFile)
			}
		}

		// Apply resolved settings from lock file to config
		if lockFile != nil && lockFile.Languages.Python != nil {
			// Use resolved exclude imports from lock file
			if len(lockFile.Languages.Python.ExcludeImports) > 0 {
				cfg.Languages.Python.ExcludeImports = lockFile.Languages.Python.ExcludeImports
				log.Debug("Applied exclude imports from lock file",
					logger.Int("count", len(cfg.Languages.Python.ExcludeImports)))
			}
		}
	}

	// Add dependencies to import paths
	importPaths := cfg.Proto.ImportPaths

	// Add proto.paths to import paths so protoc can find project files
	// Using proto.paths directly allows proper structure: protos/file.proto with --proto_path=protos → file.pb.h
	if len(cfg.Proto.Paths) > 0 {
		importPaths = append(importPaths, cfg.Proto.Paths...)
	}

	if len(cfg.Dependencies) > 0 {
		log.Info("Loading dependencies", logger.Int("count", len(cfg.Dependencies)))
		depManager, err := dependency.NewManager(".buffalo", log)
		if err != nil {
			log.Warn("Failed to create dependency manager", logger.Any("error", err))
		} else {
			depPaths := depManager.GetProtoPaths()
			if len(depPaths) > 0 {
				log.Info("Adding dependency paths", logger.Int("count", len(depPaths)))
				importPaths = append(importPaths, depPaths...)
			} else {
				log.Warn("⚠️  Dependencies configured but not installed. Run 'buffalo install' first.")
			}
		}
	}

	// Auto-extract and add buffalo/validate proto import path.
	// This ensures `import "buffalo/validate/validate.proto"` resolves
	// even after `go install` when proto files are embedded in the binary.
	validateProtoPath, err := embedded.ValidateProtoImportPath(".buffalo")
	if err != nil {
		log.Warn("Failed to resolve validate proto path", logger.Any("error", err))
	} else {
		importPaths = append(importPaths, validateProtoPath)
		log.Debug("Added validate proto path", logger.String("path", validateProtoPath))
	}

	// Initialize plugin registry
	pluginRegistry := plugin.NewRegistry(log)

	// Register built-in plugins
	if len(cfg.Plugins) > 0 {
		log.Info("Loading plugins", logger.Int("count", len(cfg.Plugins)))

		for _, pluginCfg := range cfg.Plugins {
			if !pluginCfg.Enabled {
				log.Debug("Plugin disabled", logger.String("name", pluginCfg.Name))
				continue
			}

			// Check for built-in plugins
			if pluginCfg.Name == "naming-validator" {
				namingValidator := plugin.NewSimpleNamingValidator()

				// Convert config
				hookPoints := make([]plugin.HookPoint, len(pluginCfg.HookPoints))
				for i, hp := range pluginCfg.HookPoints {
					hookPoints[i] = plugin.HookPoint(hp)
				}

				plgCfg := plugin.Config{
					Name:       pluginCfg.Name,
					Enabled:    pluginCfg.Enabled,
					HookPoints: hookPoints,
					Priority:   pluginCfg.Priority,
					Options:    pluginCfg.Options,
				}

				if err := pluginRegistry.Register(namingValidator, plgCfg); err != nil {
					log.Warn("Failed to register plugin",
						logger.String("name", pluginCfg.Name),
						logger.Any("error", err))
				} else {
					log.Info("Plugin registered", logger.String("name", pluginCfg.Name))
				}
			}
		}

		// Initialize all plugins
		if err := pluginRegistry.InitAll(); err != nil {
			return err
		}
	}

	// Create builder
	b, err := builder.New(
		cfg,
		builder.WithLogger(log),
		builder.WithPluginRegistry(pluginRegistry),
	)
	if err != nil {
		return err
	}

	// Create build plan
	plan := &builder.BuildPlan{
		ProtoFiles:  allProtoFiles,
		ImportPaths: importPaths,
		OutputDir:   cfg.Output.BaseDir,
		Languages:   languages,
		Options: builder.BuildOptions{
			Workers:     cfg.Build.Workers,
			Incremental: cfg.Build.Incremental,
			Cache:       cfg.Build.Cache.Enabled,
			CacheDir:    cfg.Build.Cache.Directory,
			DryRun:      buildDryRun,
			Verbose:     verbose,
		},
	}

	// Execute build
	result, err := b.Build(ctx, plan)
	if err != nil {
		log.Error("❌ Build failed", logger.Any("error", err))
		return err
	}

	// Display results
	log.Info("✅ Build completed successfully!",
		logger.String("duration", result.Duration.String()),
		logger.Int("files_processed", result.FilesProcessed),
		logger.Int("files_generated", result.FilesGenerated),
	)

	if cfg.Build.Cache.Enabled {
		log.Info("Cache statistics",
			logger.Int("hits", result.CacheHits),
			logger.Int("misses", result.CacheMisses),
		)
	}

	if len(result.Warnings) > 0 {
		log.Warn("Build completed with warnings", logger.Int("count", len(result.Warnings)))
		for _, warning := range result.Warnings {
			log.Warn("⚠️  " + warning)
		}
	}

	// Display metrics if verbose
	if verbose {
		metrics := b.GetMetrics()
		snapshot := metrics.Snapshot()
		log.Debug("Build metrics",
			logger.Int("total_metrics", len(snapshot.Metrics)),
		)
	}

	return nil
}

// loadConfig loads configuration from viper
func loadConfig(log *logger.Logger) (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// loadConfigWithPath loads configuration and returns the config file path
func loadConfigWithPath(log *logger.Logger) (*config.Config, string, error) {
	// Try to find config file in current directory
	configPaths := []string{
		"buffalo.yaml",
		"buffalo.yml",
		".buffalo.yaml",
		".buffalo.yml",
	}

	for _, path := range configPaths {
		cfg, err := config.LoadFromFile(path)
		if err == nil {
			return cfg, path, nil
		}
	}

	// Fall back to viper config
	cfg, err := config.Load()
	if err != nil {
		return nil, "", err
	}
	return cfg, "buffalo.yaml", nil
}

// getDefaultConfig returns default configuration
func getDefaultConfig() *config.Config {
	return &config.Config{
		Project: config.ProjectConfig{
			Name:    "unnamed",
			Version: "0.1.0",
		},
		Proto: config.ProtoConfig{
			Paths:       []string{"."},
			ImportPaths: []string{},
		},
		Output: config.OutputConfig{
			BaseDir: "./generated",
		},
		Languages: config.LanguagesConfig{},
		Build: config.BuildConfig{
			Workers:     0,
			Incremental: true,
			Cache: config.CacheConfig{
				Enabled:   true,
				Directory: ".buffalo-cache",
			},
		},
	}
}

// enableLanguages enables specified languages in config
func enableLanguages(cfg *config.Config, languages []string) {
	for _, lang := range languages {
		switch lang {
		case "python":
			cfg.Languages.Python.Enabled = true
		case "go":
			cfg.Languages.Go.Enabled = true
		case "rust":
			cfg.Languages.Rust.Enabled = true
		case "cpp":
			cfg.Languages.Cpp.Enabled = true
		case "typescript":
			cfg.Languages.Typescript.Enabled = true
		}
	}
}
