package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/massonsky/buffalo/internal/builder"
	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/massonsky/buffalo/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	rebuildForce     bool
	rebuildLanguages []string
	rebuildOutputDir string

	rebuildCmd = &cobra.Command{
		Use:   "rebuild",
		Short: "Force a full rebuild, ignoring cache",
		Long: `Force a complete rebuild of all proto files, ignoring any cached results.

This is useful when you want to ensure a clean build or when you suspect
that cached results may be stale or corrupted.

Examples:
  # Full rebuild with cache cleared
  buffalo rebuild

  # Rebuild for specific languages
  buffalo rebuild --lang python,go

  # Rebuild with custom output
  buffalo rebuild --output ./generated`,
		RunE: runRebuild,
	}
)

func init() {
	rootCmd.AddCommand(rebuildCmd)

	rebuildCmd.Flags().BoolVarP(&rebuildForce, "force", "f", true, "force rebuild (always true)")
	rebuildCmd.Flags().StringSliceVarP(&rebuildLanguages, "lang", "l", []string{}, "target languages (python,go,rust,cpp)")
	rebuildCmd.Flags().StringVarP(&rebuildOutputDir, "output", "o", "./generated", "output directory for generated code")
}

func runRebuild(cmd *cobra.Command, args []string) error {
	log := GetLogger()
	ctx := context.Background()

	log.Info("🔨 Starting full rebuild (cache will be cleared)")

	// Load configuration
	cfg, err := loadConfig(log)
	if err != nil {
		log.Warn("Failed to load config, using defaults", logger.Any("error", err))
		cfg = getDefaultConfig()
	}

	// Override config with flags
	if rebuildOutputDir != "./generated" {
		cfg.Output.BaseDir = rebuildOutputDir
	}
	if len(rebuildLanguages) > 0 {
		enableLanguages(cfg, rebuildLanguages)
	}

	// Get enabled languages
	languages := cfg.GetEnabledLanguages()
	if len(rebuildLanguages) > 0 {
		languages = rebuildLanguages
	}

	if len(languages) == 0 {
		log.Warn("⚠️  No languages enabled")
		return fmt.Errorf("no languages enabled")
	}

	// Clear cache directory
	if cfg.Build.Cache.Enabled && cfg.Build.Cache.Directory != "" {
		cacheDir := cfg.Build.Cache.Directory
		if err := os.RemoveAll(cacheDir); err != nil {
			log.Warn("Failed to clear cache", logger.String("dir", cacheDir), logger.Any("error", err))
		} else {
			log.Info("🗑️  Cache cleared", logger.String("dir", cacheDir))
		}
	}

	// Clear output directory
	if err := os.RemoveAll(cfg.Output.BaseDir); err != nil {
		log.Warn("Failed to clear output", logger.String("dir", cfg.Output.BaseDir), logger.Any("error", err))
	} else {
		log.Info("🗑️  Output cleared", logger.String("dir", cfg.Output.BaseDir))
	}

	log.Info("Rebuild configuration",
		logger.String("output", cfg.Output.BaseDir),
		logger.Any("languages", languages),
		logger.Any("proto_paths", cfg.Proto.Paths),
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
		log.Warn("⚠️  No proto files found")
		return fmt.Errorf("no proto files found")
	}

	log.Info("Found proto files", logger.Int("count", len(allProtoFiles)))

	// Create builder with cache disabled for this rebuild
	cfgNoCahce := *cfg
	cfgNoCahce.Build.Cache.Enabled = false

	b, err := builder.New(&cfgNoCahce, builder.WithLogger(log))
	if err != nil {
		return err
	}

	// Create build plan
	plan := &builder.BuildPlan{
		ProtoFiles:  allProtoFiles,
		ImportPaths: cfg.Proto.ImportPaths,
		OutputDir:   cfg.Output.BaseDir,
		Languages:   languages,
		Options: builder.BuildOptions{
			Workers:     cfg.Build.Workers,
			Incremental: false, // Disable incremental for full rebuild
			Cache:       false, // Disable cache
			Verbose:     verbose,
		},
	}

	// Execute build
	startTime := time.Now()
	result, err := b.Build(ctx, plan)
	if err != nil {
		log.Error("❌ Rebuild failed", logger.Any("error", err))
		return err
	}

	elapsed := time.Since(startTime)

	// Display results
	log.Info("✅ Rebuild completed successfully!",
		logger.String("duration", elapsed.Round(time.Millisecond).String()),
		logger.Int("files_generated", result.FilesGenerated),
		logger.Int("files_processed", result.FilesProcessed),
	)

	if len(result.Warnings) > 0 {
		log.Warn("⚠️  Build completed with warnings:")
		for _, warning := range result.Warnings {
			log.Warn("  • " + warning)
		}
	}

	return nil
}
