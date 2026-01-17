package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/massonsky/buffalo/internal/builder"
	"github.com/massonsky/buffalo/internal/config"
	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/massonsky/buffalo/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	watchPaths     []string
	watchLanguages []string
	watchDebounce  int
	watchOutputDir string

	watchCmd = &cobra.Command{
		Use:   "watch",
		Short: "Watch proto files and rebuild on changes",
		Long: `Watch proto files for changes and automatically rebuild when they change.

This command will monitor the specified paths for any changes to .proto files
and trigger a rebuild whenever a change is detected. It uses a debounce 
mechanism to avoid rebuilding too frequently.

Examples:
  # Watch default paths
  buffalo watch

  # Watch specific paths
  buffalo watch --proto-path ./protos --proto-path ./api

  # Watch with custom debounce (milliseconds)
  buffalo watch --debounce 1000

  # Watch and build for specific languages
  buffalo watch --lang python,go`,
		RunE: runWatch,
	}
)

func init() {
	rootCmd.AddCommand(watchCmd)

	watchCmd.Flags().StringSliceVarP(&watchPaths, "proto-path", "p", []string{"."}, "paths to watch for proto files")
	watchCmd.Flags().StringSliceVarP(&watchLanguages, "lang", "l", []string{}, "target languages (python,go,rust,cpp)")
	watchCmd.Flags().IntVar(&watchDebounce, "debounce", 500, "debounce delay in milliseconds")
	watchCmd.Flags().StringVarP(&watchOutputDir, "output", "o", "./generated", "output directory for generated code")
}

func runWatch(cmd *cobra.Command, args []string) error {
	log := GetLogger()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Info("👀 Starting watch mode")

	// Load configuration
	cfg, err := loadConfig(log)
	if err != nil {
		log.Warn("Failed to load config, using defaults", logger.Any("error", err))
		cfg = getDefaultConfig()
	}

	// Override config with flags
	if watchOutputDir != "./generated" {
		cfg.Output.BaseDir = watchOutputDir
	}
	if len(watchLanguages) > 0 {
		enableLanguages(cfg, watchLanguages)
	}
	if len(watchPaths) > 0 {
		cfg.Proto.Paths = watchPaths
	}

	// Get enabled languages
	languages := cfg.GetEnabledLanguages()
	if len(watchLanguages) > 0 {
		languages = watchLanguages
	}

	if len(languages) == 0 {
		log.Warn("⚠️  No languages enabled. Please enable at least one language in config or use --lang flag")
		return fmt.Errorf("no languages enabled")
	}

	log.Info("Watch configuration",
		logger.Any("paths", cfg.Proto.Paths),
		logger.Any("languages", languages),
		logger.String("output", cfg.Output.BaseDir),
		logger.Int("debounce_ms", watchDebounce),
	)

	// Perform initial build
	log.Info("🔨 Performing initial build...")
	if err := performBuild(ctx, cfg, languages, log); err != nil {
		log.Error("Initial build failed", logger.Any("error", err))
		return err
	}

	// Setup file watcher
	watcher, err := utils.NewFileWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	defer watcher.Close()

	// Add watch paths
	for _, path := range cfg.Proto.Paths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			log.Warn("Failed to get absolute path", logger.String("path", path), logger.Any("error", err))
			continue
		}

		if err := watcher.Add(absPath); err != nil {
			log.Warn("Failed to watch path", logger.String("path", absPath), logger.Any("error", err))
			continue
		}

		log.Info("👁️  Watching path", logger.String("path", absPath))
	}

	// Setup signal handler
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Debounce timer
	var debounceTimer *time.Timer
	debounceDuration := time.Duration(watchDebounce) * time.Millisecond

	log.Info("✅ Watch mode started. Press Ctrl+C to stop.")
	log.Info("⏱️  Debounce: %dms", logger.Int("ms", watchDebounce))

	for {
		select {
		case event, ok := <-watcher.Events():
			if !ok {
				return nil
			}

			// Only process .proto files
			if filepath.Ext(event.Name) != ".proto" {
				continue
			}

			log.Debug("File event",
				logger.String("file", event.Name),
				logger.String("op", event.Op.String()),
			)

			// Reset debounce timer
			if debounceTimer != nil {
				debounceTimer.Stop()
			}

			debounceTimer = time.AfterFunc(debounceDuration, func() {
				log.Info("🔄 Change detected, rebuilding...",
					logger.String("file", filepath.Base(event.Name)),
				)

				startTime := time.Now()
				if err := performBuild(ctx, cfg, languages, log); err != nil {
					log.Error("❌ Rebuild failed", logger.Any("error", err))
				} else {
					elapsed := time.Since(startTime)
					log.Info("✅ Rebuild completed",
						logger.String("duration", elapsed.Round(time.Millisecond).String()),
					)
				}
			})

		case err, ok := <-watcher.Errors():
			if !ok {
				return nil
			}
			log.Error("Watcher error", logger.Any("error", err))

		case <-sigChan:
			log.Info("\n🛑 Stopping watch mode...")
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return nil

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// performBuild executes a build with the given configuration
func performBuild(ctx context.Context, cfg *config.Config, languages []string, log *logger.Logger) error {
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
		return fmt.Errorf("no proto files found")
	}

	// Create builder
	b, err := builder.New(cfg, builder.WithLogger(log))
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
			Incremental: cfg.Build.Incremental,
			Cache:       cfg.Build.Cache.Enabled,
			CacheDir:    cfg.Build.Cache.Directory,
			Verbose:     verbose,
		},
	}

	// Execute build
	_, err = b.Build(ctx, plan)
	return err
}
