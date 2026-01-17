package cli

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/massonsky/buffalo/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	listPaths     []string
	listRecursive bool
	listFullPath  bool
	listGrouped   bool

	listCmd = &cobra.Command{
		Use:   "list",
		Short: "List all proto files in the project",
		Long: `List all proto files found in the specified paths.

This command helps you see which proto files will be processed during
a build. You can filter by path, see grouped output, and control the
level of detail.

Examples:
  # List all proto files
  buffalo list

  # List from specific paths
  buffalo list --proto-path ./protos --proto-path ./api

  # Show full paths
  buffalo list --full

  # Group by directory
  buffalo list --grouped`,
		RunE: runList,
	}
)

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringSliceVarP(&listPaths, "proto-path", "p", []string{"."}, "paths to search for proto files")
	listCmd.Flags().BoolVarP(&listRecursive, "recursive", "r", true, "search recursively")
	listCmd.Flags().BoolVarP(&listFullPath, "full", "f", false, "show full paths")
	listCmd.Flags().BoolVarP(&listGrouped, "grouped", "g", false, "group by directory")
}

func runList(cmd *cobra.Command, args []string) error {
	log := GetLogger()

	// Load configuration
	cfg, err := loadConfig(log)
	if err == nil && len(listPaths) == 1 && listPaths[0] == "." {
		// Use paths from config if not explicitly provided
		listPaths = cfg.Proto.Paths
	}

	log.Info("📋 Listing proto files...")
	log.Debug("Search paths", logger.Any("paths", listPaths))

	// Find all proto files
	type FileInfo struct {
		Path      string
		Directory string
		Name      string
		Size      int64
	}

	var allFiles []FileInfo
	for _, path := range listPaths {
		fileInfos, err := utils.FindFiles(path, utils.FindFilesOptions{
			Pattern:   "*.proto",
			Recursive: listRecursive,
		})
		if err != nil {
			log.Warn("Failed to scan path", logger.String("path", path), logger.Any("error", err))
			continue
		}

		for _, fi := range fileInfos {
			absPath, _ := filepath.Abs(fi.Path)
			dir := filepath.Dir(absPath)
			name := filepath.Base(absPath)

			allFiles = append(allFiles, FileInfo{
				Path:      absPath,
				Directory: dir,
				Name:      name,
				Size:      fi.Size,
			})
		}
	}

	if len(allFiles) == 0 {
		log.Warn("⚠️  No proto files found")
		return nil
	}

	// Sort files
	sort.Slice(allFiles, func(i, j int) bool {
		if allFiles[i].Directory == allFiles[j].Directory {
			return allFiles[i].Name < allFiles[j].Name
		}
		return allFiles[i].Directory < allFiles[j].Directory
	})

	log.Info(fmt.Sprintf("✅ Found %d proto file(s)\n", len(allFiles)))

	if listGrouped {
		// Group by directory
		dirMap := make(map[string][]FileInfo)
		for _, file := range allFiles {
			dir := file.Directory
			if !listFullPath {
				dir = getRelativePath(dir)
			}
			dirMap[dir] = append(dirMap[dir], file)
		}

		// Sort directories
		dirs := make([]string, 0, len(dirMap))
		for dir := range dirMap {
			dirs = append(dirs, dir)
		}
		sort.Strings(dirs)

		// Print grouped
		for _, dir := range dirs {
			files := dirMap[dir]
			fmt.Printf("📁 %s (%d file(s))\n", dir, len(files))
			for _, file := range files {
				if listFullPath {
					fmt.Printf("   • %s\n", file.Path)
				} else {
					fmt.Printf("   • %s\n", file.Name)
				}
			}
			fmt.Println()
		}
	} else {
		// Simple list
		for _, file := range allFiles {
			if listFullPath {
				fmt.Printf("• %s\n", file.Path)
			} else {
				relPath := getRelativePath(file.Path)
				fmt.Printf("• %s\n", relPath)
			}
		}
		fmt.Println()
	}

	// Summary
	totalSize := int64(0)
	for _, file := range allFiles {
		totalSize += file.Size
	}

	fmt.Printf("📊 Summary:\n")
	fmt.Printf("   Files: %d\n", len(allFiles))
	fmt.Printf("   Total size: %s\n", formatBytes(totalSize))

	// Count unique directories
	uniqueDirs := make(map[string]bool)
	for _, file := range allFiles {
		uniqueDirs[file.Directory] = true
	}
	fmt.Printf("   Directories: %d\n", len(uniqueDirs))

	return nil
}

// getRelativePath returns a relative path from current directory
func getRelativePath(path string) string {
	cwd, err := filepath.Abs(".")
	if err != nil {
		return path
	}

	relPath, err := filepath.Rel(cwd, path)
	if err != nil {
		return path
	}

	return relPath
}

// formatBytes formats byte size in human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
