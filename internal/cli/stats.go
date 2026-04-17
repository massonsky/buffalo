package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/massonsky/buffalo/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	statsDetailed bool
	statsJSON     bool

	statsCmd = &cobra.Command{
		Use:   "stats",
		Short: "Show project statistics",
		Long: `Show detailed statistics about the project including proto files,
generated code, cache usage, and build history.

This command provides insights into your project's structure and
build performance.

Examples:
  # Basic statistics
  buffalo stats

  # Detailed statistics
  buffalo stats --detailed

  # Output as JSON
  buffalo stats --json`,
		RunE: runStats,
	}
)

func init() {
	rootCmd.AddCommand(statsCmd)

	statsCmd.Flags().BoolVarP(&statsDetailed, "detailed", "d", false, "show detailed statistics")
	statsCmd.Flags().BoolVar(&statsJSON, "json", false, "output as JSON")
}

type ProjectStats struct {
	ProtoFiles struct {
		Count       int
		TotalSize   int64
		Directories int
		LargestFile struct {
			Path string
			Size int64
		}
	}
	GeneratedCode struct {
		Python struct {
			Exists bool
			Files  int
			Size   int64
		}
		Go struct {
			Exists bool
			Files  int
			Size   int64
		}
		Rust struct {
			Exists bool
			Files  int
			Size   int64
		}
		Cpp struct {
			Exists bool
			Files  int
			Size   int64
		}
		TotalFiles int
		TotalSize  int64
	}
	Cache struct {
		Exists bool
		Size   int64
		Files  int
	}
	Config struct {
		Exists    bool
		Languages []string
	}
}

func runStats(cmd *cobra.Command, args []string) error {
	log := GetLogger()

	log.Info("📊 Collecting project statistics...")

	stats := ProjectStats{}

	// Load configuration
	cfg, err := loadConfig(log)
	if err == nil {
		stats.Config.Exists = true
		stats.Config.Languages = cfg.GetEnabledLanguages()
	}

	// 1. Proto files statistics
	if cfg != nil {
		var allProtoFiles []utils.FileInfo
		uniqueDirs := make(map[string]bool)
		maxSize := int64(0)
		maxFile := ""

		for _, path := range cfg.Proto.Paths {
			fileInfos, err := utils.FindFiles(path, utils.FindFilesOptions{
				Pattern:   "*.proto",
				Recursive: true,
			})
			if err != nil {
				continue
			}

			for _, fi := range fileInfos {
				allProtoFiles = append(allProtoFiles, fi)
				stats.ProtoFiles.TotalSize += fi.Size
				uniqueDirs[filepath.Dir(fi.Path)] = true

				if fi.Size > maxSize {
					maxSize = fi.Size
					maxFile = fi.Path
				}
			}
		}

		stats.ProtoFiles.Count = len(allProtoFiles)
		stats.ProtoFiles.Directories = len(uniqueDirs)
		stats.ProtoFiles.LargestFile.Path = maxFile
		stats.ProtoFiles.LargestFile.Size = maxSize
	}

	// 2. Generated code statistics
	if cfg != nil && cfg.Output.BaseDir != "" {
		baseDir := cfg.Output.BaseDir

		// Python
		pythonDir := filepath.Join(baseDir, cfg.Output.Directories["python"])
		if info, err := os.Stat(pythonDir); err == nil && info.IsDir() {
			stats.GeneratedCode.Python.Exists = true
			_ = countFilesRecursive(pythonDir, &stats.GeneratedCode.Python.Files, &stats.GeneratedCode.Python.Size)
		}

		// Go
		goDir := filepath.Join(baseDir, cfg.Output.Directories["go"])
		if info, err := os.Stat(goDir); err == nil && info.IsDir() {
			stats.GeneratedCode.Go.Exists = true
			_ = countFilesRecursive(goDir, &stats.GeneratedCode.Go.Files, &stats.GeneratedCode.Go.Size)
		}

		// Rust
		rustDir := filepath.Join(baseDir, cfg.Output.Directories["rust"])
		if info, err := os.Stat(rustDir); err == nil && info.IsDir() {
			stats.GeneratedCode.Rust.Exists = true
			_ = countFilesRecursive(rustDir, &stats.GeneratedCode.Rust.Files, &stats.GeneratedCode.Rust.Size)
		}

		// C++
		cppDir := filepath.Join(baseDir, cfg.Output.Directories["cpp"])
		if info, err := os.Stat(cppDir); err == nil && info.IsDir() {
			stats.GeneratedCode.Cpp.Exists = true
			_ = countFilesRecursive(cppDir, &stats.GeneratedCode.Cpp.Files, &stats.GeneratedCode.Cpp.Size)
		}

		stats.GeneratedCode.TotalFiles = stats.GeneratedCode.Python.Files +
			stats.GeneratedCode.Go.Files +
			stats.GeneratedCode.Rust.Files +
			stats.GeneratedCode.Cpp.Files

		stats.GeneratedCode.TotalSize = stats.GeneratedCode.Python.Size +
			stats.GeneratedCode.Go.Size +
			stats.GeneratedCode.Rust.Size +
			stats.GeneratedCode.Cpp.Size
	}

	// 3. Cache statistics
	if cfg != nil && cfg.Build.Cache.Enabled && cfg.Build.Cache.Directory != "" {
		cacheDir := cfg.Build.Cache.Directory
		if info, err := os.Stat(cacheDir); err == nil && info.IsDir() {
			stats.Cache.Exists = true
			_ = countFilesRecursive(cacheDir, &stats.Cache.Files, &stats.Cache.Size)
		}
	}

	// Output results
	if statsJSON {
		// TODO: Output as JSON
		fmt.Println("JSON output not yet implemented")
		return nil
	}

	// Text output
	fmt.Println("\n╔════════════════════════════════════════════════════════╗")
	fmt.Println("║         Buffalo Project Statistics                     ║")
	fmt.Println("╚════════════════════════════════════════════════════════╝")

	// Configuration
	fmt.Println("\n📋 Configuration")
	if stats.Config.Exists {
		fmt.Println("   Status: ✅ Found")
		if len(stats.Config.Languages) > 0 {
			fmt.Printf("   Enabled Languages: %s\n", formatList(stats.Config.Languages))
		} else {
			fmt.Println("   Enabled Languages: ⚠️  None")
		}
	} else {
		fmt.Println("   Status: ❌ Not found")
	}

	// Proto files
	fmt.Println("\n📦 Proto Files")
	if stats.ProtoFiles.Count > 0 {
		fmt.Printf("   Count: %d file(s)\n", stats.ProtoFiles.Count)
		fmt.Printf("   Total Size: %s\n", formatBytes(stats.ProtoFiles.TotalSize))
		fmt.Printf("   Directories: %d\n", stats.ProtoFiles.Directories)
		if statsDetailed && stats.ProtoFiles.LargestFile.Path != "" {
			fmt.Printf("   Largest File: %s (%s)\n",
				filepath.Base(stats.ProtoFiles.LargestFile.Path),
				formatBytes(stats.ProtoFiles.LargestFile.Size))
		}
	} else {
		fmt.Println("   Status: ⚠️  No files found")
	}

	// Generated code
	fmt.Println("\n🔨 Generated Code")
	if stats.GeneratedCode.TotalFiles > 0 {
		fmt.Printf("   Total Files: %d\n", stats.GeneratedCode.TotalFiles)
		fmt.Printf("   Total Size: %s\n", formatBytes(stats.GeneratedCode.TotalSize))

		if statsDetailed {
			if stats.GeneratedCode.Python.Exists {
				fmt.Printf("   Python: %d files (%s)\n",
					stats.GeneratedCode.Python.Files,
					formatBytes(stats.GeneratedCode.Python.Size))
			}
			if stats.GeneratedCode.Go.Exists {
				fmt.Printf("   Go: %d files (%s)\n",
					stats.GeneratedCode.Go.Files,
					formatBytes(stats.GeneratedCode.Go.Size))
			}
			if stats.GeneratedCode.Rust.Exists {
				fmt.Printf("   Rust: %d files (%s)\n",
					stats.GeneratedCode.Rust.Files,
					formatBytes(stats.GeneratedCode.Rust.Size))
			}
			if stats.GeneratedCode.Cpp.Exists {
				fmt.Printf("   C++: %d files (%s)\n",
					stats.GeneratedCode.Cpp.Files,
					formatBytes(stats.GeneratedCode.Cpp.Size))
			}
		}
	} else {
		fmt.Println("   Status: ⚠️  No generated files found")
	}

	// Cache
	fmt.Println("\n💾 Build Cache")
	if stats.Cache.Exists {
		fmt.Printf("   Status: ✅ Active\n")
		fmt.Printf("   Files: %d\n", stats.Cache.Files)
		fmt.Printf("   Size: %s\n", formatBytes(stats.Cache.Size))
	} else {
		fmt.Println("   Status: ❌ Not found or disabled")
	}

	// Summary
	fmt.Println("\n╔════════════════════════════════════════════════════════╗")
	fmt.Println("║  Quick Summary                                          ║")
	fmt.Println("╚════════════════════════════════════════════════════════╝")
	fmt.Printf("   Proto → Generated: %d → %d files\n",
		stats.ProtoFiles.Count,
		stats.GeneratedCode.TotalFiles)
	fmt.Printf("   Compression Ratio: %.1fx\n",
		float64(stats.GeneratedCode.TotalSize)/float64(max(stats.ProtoFiles.TotalSize, 1)))

	if stats.Cache.Exists {
		fmt.Printf("   Cache Overhead: %s\n", formatBytes(stats.Cache.Size))
	}

	fmt.Println()

	return nil
}

// countFilesRecursive counts files and total size in a directory recursively
func countFilesRecursive(dir string, count *int, size *int64) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() {
			*count++
			*size += info.Size()
		}
		return nil
	})
}

// formatList formats a list of strings
func formatList(items []string) string {
	if len(items) == 0 {
		return ""
	}
	sort.Strings(items)
	return fmt.Sprintf("%s", items)
}

// max returns the maximum of two int64 values
func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
