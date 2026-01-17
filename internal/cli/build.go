package cli

import (
	"context"

	"github.com/massonsky/buffalo/internal/builder"
	"github.com/massonsky/buffalo/internal/config"
	"github.com/massonsky/buffalo/internal/dependency"
	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/massonsky/buffalo/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	buildOutputDir string
	buildLanguages []string
	buildProtoPath []string
	buildDryRun    bool

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
  buffalo build --dry-run`,
		RunE: runBuild,
	}
)

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().StringVarP(&buildOutputDir, "output", "o", "", "output directory for generated code")
	buildCmd.Flags().StringSliceVarP(&buildLanguages, "lang", "l", []string{}, "target languages (python,go,rust,cpp)")
	buildCmd.Flags().StringSliceVarP(&buildProtoPath, "proto-path", "p", []string{}, "paths to search for proto files")
	buildCmd.Flags().BoolVar(&buildDryRun, "dry-run", false, "show what would be built without building")
}

func runBuild(cmd *cobra.Command, args []string) error {
	log := GetLogger()
	ctx := context.Background()

	log.Info("🔨 Starting build process")

	// Load configuration
	cfg, err := loadConfig(log)
	if err != nil {
		log.Warn("Failed to load config, using defaults", logger.Any("error", err))
		cfg = getDefaultConfig()
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

	// Add dependencies to import paths
	importPaths := cfg.Proto.ImportPaths
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

	// Create builder
	b, err := builder.New(
		cfg,
		builder.WithLogger(log),
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
		}
	}
}
