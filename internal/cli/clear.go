package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/spf13/cobra"
)

var (
	clearCache   bool
	clearOutput  bool
	clearAll     bool
	clearConfirm bool

	clearCmd = &cobra.Command{
		Use:   "clear",
		Short: "Clear cache and generated files",
		Long: `Clear cached build artifacts and/or generated output files.

This command helps you clean up your project by removing cached data
and generated files. You can choose to clear specific items or everything.

Examples:
  # Clear only cache
  buffalo clear --cache

  # Clear only generated output
  buffalo clear --output

  # Clear everything (cache + output)
  buffalo clear --all

  # Clear with confirmation prompt
  buffalo clear --all --confirm`,
		RunE: runClear,
	}
)

func init() {
	rootCmd.AddCommand(clearCmd)

	clearCmd.Flags().BoolVar(&clearCache, "cache", false, "clear build cache")
	clearCmd.Flags().BoolVar(&clearOutput, "output", false, "clear generated output files")
	clearCmd.Flags().BoolVar(&clearAll, "all", false, "clear everything (cache + output)")
	clearCmd.Flags().BoolVarP(&clearConfirm, "confirm", "y", false, "skip confirmation prompt")
}

func runClear(cmd *cobra.Command, args []string) error {
	log := GetLogger()

	// If no flags specified, default to --all
	if !clearCache && !clearOutput && !clearAll {
		clearAll = true
	}

	if clearAll {
		clearCache = true
		clearOutput = true
	}

	// Load configuration to get directories
	cfg, err := loadConfig(log)
	if err != nil {
		log.Warn("Failed to load config, using defaults", logger.Any("error", err))
		cfg = getDefaultConfig()
	}

	var itemsToClear []string
	if clearCache && cfg.Build.Cache.Directory != "" {
		itemsToClear = append(itemsToClear, fmt.Sprintf("Cache: %s", cfg.Build.Cache.Directory))
	}
	if clearOutput && cfg.Output.BaseDir != "" {
		itemsToClear = append(itemsToClear, fmt.Sprintf("Output: %s", cfg.Output.BaseDir))
	}

	if len(itemsToClear) == 0 {
		log.Info("Nothing to clear")
		return nil
	}

	// Confirmation prompt
	if !clearConfirm {
		fmt.Println("🗑️  The following will be deleted:")
		for _, item := range itemsToClear {
			fmt.Printf("  • %s\n", item)
		}
		fmt.Print("\nAre you sure? (y/N): ")
		var response string
		_, _ = fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			log.Info("❌ Canceled")
			return nil
		}
	}

	log.Info("🧹 Clearing...")

	// Clear cache
	if clearCache && cfg.Build.Cache.Directory != "" {
		cacheDir := cfg.Build.Cache.Directory
		if err := clearDirectory(cacheDir, log); err != nil {
			log.Warn("Failed to clear cache", logger.String("dir", cacheDir), logger.Any("error", err))
		} else {
			log.Info("✅ Cache cleared", logger.String("dir", cacheDir))
		}
	}

	// Clear output
	if clearOutput && cfg.Output.BaseDir != "" {
		outputDir := cfg.Output.BaseDir
		if err := clearDirectory(outputDir, log); err != nil {
			log.Warn("Failed to clear output", logger.String("dir", outputDir), logger.Any("error", err))
		} else {
			log.Info("✅ Output cleared", logger.String("dir", outputDir))
		}
	}

	log.Info("✨ Clear completed successfully!")
	return nil
}

// clearDirectory removes all contents of a directory
func clearDirectory(dir string, log *logger.Logger) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Safety check: don't delete root or home directory
	if absDir == "/" || absDir == filepath.Dir(absDir) {
		return fmt.Errorf("refusing to delete root directory: %s", absDir)
	}

	homeDir, _ := os.UserHomeDir()
	if absDir == homeDir {
		return fmt.Errorf("refusing to delete home directory: %s", absDir)
	}

	// Check if directory exists
	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		log.Debug("Directory does not exist, skipping", logger.String("dir", absDir))
		return nil
	}

	// Remove directory and all contents
	if err := os.RemoveAll(absDir); err != nil {
		return fmt.Errorf("failed to remove directory: %w", err)
	}

	return nil
}
