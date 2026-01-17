package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/massonsky/buffalo/internal/builder"
	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/massonsky/buffalo/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	diffLang       []string
	diffOutput     string
	diffFormat     string
	diffContext    int
	diffShowSame   bool
	diffColorize   bool
	diffExclude    []string
	
	diffCmd = &cobra.Command{
		Use:   "diff",
		Short: "Show differences between current and new generated files",
		Long: `Show what changes would be made to generated files if build was run.

This command performs a dry-run build to generate new files in a temporary
directory, then compares them with existing generated files.

Useful for:
  - Reviewing changes before committing
  - Checking impact of proto modifications
  - Verifying generated code changes

Output formats:
  - unified (default): Standard unified diff format
  - side-by-side: Show old and new side by side
  - summary: Just show which files changed`,
		Example: `  # Show all differences
  buffalo diff

  # Show differences for specific languages
  buffalo diff --lang go,python

  # Show summary only
  buffalo diff --format summary

  # Save diff to file
  buffalo diff --output changes.diff

  # Show more context lines
  buffalo diff --context 5`,
		RunE: runDiff,
	}
)

func init() {
	rootCmd.AddCommand(diffCmd)
	
	diffCmd.Flags().StringSliceVarP(&diffLang, "lang", "l", []string{}, "languages to diff (python,go,rust,cpp)")
	diffCmd.Flags().StringVarP(&diffOutput, "output", "o", "", "save diff to file instead of stdout")
	diffCmd.Flags().StringVarP(&diffFormat, "format", "f", "unified", "output format: unified, side-by-side, summary")
	diffCmd.Flags().IntVarP(&diffContext, "context", "c", 3, "number of context lines")
	diffCmd.Flags().BoolVar(&diffShowSame, "show-same", false, "show unchanged files")
	diffCmd.Flags().BoolVar(&diffColorize, "color", true, "colorize output (auto-disabled when piping)")
	diffCmd.Flags().StringSliceVar(&diffExclude, "exclude", []string{".buffalo/depends"}, "paths to exclude from diff")
}

func runDiff(cmd *cobra.Command, args []string) error {
	log := GetLogger()
	ctx := context.Background()
	
	log.Info("🔍 Computing differences in generated files")
	
	// Load configuration
	cfg, err := loadConfig(log)
	if err != nil {
		log.Warn("Failed to load config, using defaults", logger.Any("error", err))
		cfg = getDefaultConfig()
	}
	
	// Override languages if specified
	if len(diffLang) > 0 {
		enableLanguages(cfg, diffLang)
	}
	
	languages := cfg.GetEnabledLanguages()
	if len(diffLang) > 0 {
		languages = diffLang
	}
	
	if len(languages) == 0 {
		log.Warn("⚠️  No languages enabled")
		return nil
	}
	
	// Check if output directory exists
	if _, err := os.Stat(cfg.Output.BaseDir); os.IsNotExist(err) {
		log.Info("No existing generated files found. Run 'buffalo build' first.")
		return nil
	}
	
	// Create temporary directory for new generation
	tempDir, err := os.MkdirTemp("", "buffalo-diff-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)
	
	log.Debug("Temporary directory created", logger.String("path", tempDir))
	
	// Build proto files in temp directory
	log.Info("Generating new files for comparison...")
	
	// Save original output dir
	originalOutputDir := cfg.Output.BaseDir
	cfg.Output.BaseDir = tempDir
	
	// Find proto files
	var allProtoFiles []string
	for _, path := range cfg.Proto.Paths {
		// Check if path should be excluded
		shouldExclude := false
		for _, excludePattern := range diffExclude {
			if strings.Contains(path, excludePattern) || strings.Contains(filepath.Clean(path), filepath.Clean(excludePattern)) {
				shouldExclude = true
				log.Debug("Excluding path from diff", logger.String("path", path), logger.String("pattern", excludePattern))
				break
			}
		}
		if shouldExclude {
			continue
		}
		
		fileInfos, err := utils.FindFiles(path, utils.FindFilesOptions{
			Pattern:   "*.proto",
			Recursive: true,
		})
		if err != nil {
			log.Warn("Failed to find proto files", logger.String("path", path))
			continue
		}
		for _, fi := range fileInfos {
			// Check if file path should be excluded
			fileExcluded := false
			for _, excludePattern := range diffExclude {
				if strings.Contains(fi.Path, excludePattern) || strings.Contains(filepath.Clean(fi.Path), filepath.Clean(excludePattern)) {
					fileExcluded = true
					break
				}
			}
			if !fileExcluded {
				allProtoFiles = append(allProtoFiles, fi.Path)
			}
		}
	}
	
	if len(allProtoFiles) == 0 {
		log.Warn("⚠️  No proto files found")
		return nil
	}
	
	// Create builder
	b, err := builder.New(cfg, builder.WithLogger(log))
	if err != nil {
		return err
	}
	
	// Build in temp directory
	plan := &builder.BuildPlan{
		ProtoFiles:  allProtoFiles,
		ImportPaths: cfg.Proto.ImportPaths,
		OutputDir:   tempDir,
		Languages:   languages,
		Options: builder.BuildOptions{
			Workers:     cfg.Build.Workers,
			Incremental: false, // Always rebuild for diff
			Cache:       false, // No cache for diff
			DryRun:      false,
			Verbose:     false,
		},
	}
	
	_, err = b.Build(ctx, plan)
	if err != nil {
		log.Error("Failed to generate new files", logger.Any("error", err))
		return err
	}
	
	// Compare directories
	log.Info("Comparing files...")
	
	diffs, err := compareDirectories(originalOutputDir, tempDir, languages)
	if err != nil {
		return fmt.Errorf("failed to compare directories: %w", err)
	}
	
	// Output results
	var output io.Writer = os.Stdout
	if diffOutput != "" {
		f, err := os.Create(diffOutput)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer f.Close()
		output = f
		diffColorize = false // No colors in file
	}
	
	// Check if stdout is a terminal for colorization
	if diffColorize && diffOutput == "" {
		fileInfo, _ := os.Stdout.Stat()
		if (fileInfo.Mode() & os.ModeCharDevice) == 0 {
			diffColorize = false // Piped output, no colors
		}
	}
	
	switch diffFormat {
	case "summary":
		printSummary(output, diffs, diffShowSame, diffColorize)
	case "side-by-side":
		printSideBySide(output, diffs, diffContext, diffShowSame, diffColorize)
	default: // unified
		printUnified(output, diffs, diffContext, diffShowSame, diffColorize)
	}
	
	// Print statistics
	added := 0
	modified := 0
	deleted := 0
	unchanged := 0
	
	for _, diff := range diffs {
		switch diff.Status {
		case "added":
			added++
		case "modified":
			modified++
		case "deleted":
			deleted++
		case "unchanged":
			unchanged++
		}
	}
	
	log.Info("")
	log.Info("📊 Summary:",
		logger.Int("added", added),
		logger.Int("modified", modified),
		logger.Int("deleted", deleted),
		logger.Int("unchanged", unchanged),
	)
	
	if added+modified+deleted == 0 {
		log.Info("✅ No changes detected - all generated files are up to date!")
	}
	
	return nil
}

type FileDiff struct {
	Path     string
	Status   string // "added", "modified", "deleted", "unchanged"
	OldLines []string
	NewLines []string
}

func compareDirectories(oldDir, newDir string, languages []string) ([]FileDiff, error) {
	diffs := []FileDiff{}
	
	// Map to track all files
	allFiles := make(map[string]bool)
	
	// Scan old directory
	oldFiles := make(map[string]string) // relative path -> absolute path
	for _, lang := range languages {
		langDir := filepath.Join(oldDir, lang)
		if _, err := os.Stat(langDir); os.IsNotExist(err) {
			continue
		}
		
		filepath.Walk(langDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			relPath, _ := filepath.Rel(oldDir, path)
			oldFiles[relPath] = path
			allFiles[relPath] = true
			return nil
		})
	}
	
	// Scan new directory
	newFiles := make(map[string]string)
	for _, lang := range languages {
		langDir := filepath.Join(newDir, lang)
		if _, err := os.Stat(langDir); os.IsNotExist(err) {
			continue
		}
		
		filepath.Walk(langDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			relPath, _ := filepath.Rel(newDir, path)
			newFiles[relPath] = path
			allFiles[relPath] = true
			return nil
		})
	}
	
	// Compare files
	for relPath := range allFiles {
		oldPath, oldExists := oldFiles[relPath]
		newPath, newExists := newFiles[relPath]
		
		if !oldExists && newExists {
			// Added
			newLines, _ := readFileLines(newPath)
			diffs = append(diffs, FileDiff{
				Path:     relPath,
				Status:   "added",
				NewLines: newLines,
			})
		} else if oldExists && !newExists {
			// Deleted
			oldLines, _ := readFileLines(oldPath)
			diffs = append(diffs, FileDiff{
				Path:     relPath,
				Status:   "deleted",
				OldLines: oldLines,
			})
		} else {
			// Check if modified
			oldLines, _ := readFileLines(oldPath)
			newLines, _ := readFileLines(newPath)
			
			if !linesEqual(oldLines, newLines) {
				diffs = append(diffs, FileDiff{
					Path:     relPath,
					Status:   "modified",
					OldLines: oldLines,
					NewLines: newLines,
				})
			} else {
				diffs = append(diffs, FileDiff{
					Path:   relPath,
					Status: "unchanged",
				})
			}
		}
	}
	
	return diffs, nil
}

func readFileLines(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(data), "\n"), nil
}

func linesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func printSummary(w io.Writer, diffs []FileDiff, showSame bool, colorize bool) {
	for _, diff := range diffs {
		if diff.Status == "unchanged" && !showSame {
			continue
		}
		
		symbol := " "
		color := ""
		resetColor := ""
		
		if colorize {
			resetColor = "\033[0m"
		}
		
		switch diff.Status {
		case "added":
			symbol = "+"
			if colorize {
				color = "\033[32m" // Green
			}
		case "deleted":
			symbol = "-"
			if colorize {
				color = "\033[31m" // Red
			}
		case "modified":
			symbol = "M"
			if colorize {
				color = "\033[33m" // Yellow
			}
		case "unchanged":
			symbol = " "
			if colorize {
				color = "\033[90m" // Gray
			}
		}
		
		fmt.Fprintf(w, "%s%s %s%s\n", color, symbol, diff.Path, resetColor)
	}
}

func printUnified(w io.Writer, diffs []FileDiff, contextLines int, showSame bool, colorize bool) {
	for _, diff := range diffs {
		if diff.Status == "unchanged" && !showSame {
			continue
		}
		
		fmt.Fprintf(w, "diff --buffalo %s\n", diff.Path)
		
		switch diff.Status {
		case "added":
			if colorize {
				fmt.Fprintf(w, "\033[32m+++ %s\033[0m\n", diff.Path)
			} else {
				fmt.Fprintf(w, "+++ %s\n", diff.Path)
			}
			for _, line := range diff.NewLines {
				if colorize {
					fmt.Fprintf(w, "\033[32m+%s\033[0m\n", line)
				} else {
					fmt.Fprintf(w, "+%s\n", line)
				}
			}
			
		case "deleted":
			if colorize {
				fmt.Fprintf(w, "\033[31m--- %s\033[0m\n", diff.Path)
			} else {
				fmt.Fprintf(w, "--- %s\n", diff.Path)
			}
			for _, line := range diff.OldLines {
				if colorize {
					fmt.Fprintf(w, "\033[31m-%s\033[0m\n", line)
				} else {
					fmt.Fprintf(w, "-%s\n", line)
				}
			}
			
		case "modified":
			fmt.Fprintf(w, "--- %s (old)\n", diff.Path)
			fmt.Fprintf(w, "+++ %s (new)\n", diff.Path)
			
			// Simple diff algorithm
			for i := 0; i < len(diff.OldLines) || i < len(diff.NewLines); i++ {
				if i < len(diff.OldLines) && i < len(diff.NewLines) {
					if diff.OldLines[i] == diff.NewLines[i] {
						fmt.Fprintf(w, " %s\n", diff.OldLines[i])
					} else {
						if colorize {
							fmt.Fprintf(w, "\033[31m-%s\033[0m\n", diff.OldLines[i])
							fmt.Fprintf(w, "\033[32m+%s\033[0m\n", diff.NewLines[i])
						} else {
							fmt.Fprintf(w, "-%s\n", diff.OldLines[i])
							fmt.Fprintf(w, "+%s\n", diff.NewLines[i])
						}
					}
				} else if i < len(diff.OldLines) {
					if colorize {
						fmt.Fprintf(w, "\033[31m-%s\033[0m\n", diff.OldLines[i])
					} else {
						fmt.Fprintf(w, "-%s\n", diff.OldLines[i])
					}
				} else {
					if colorize {
						fmt.Fprintf(w, "\033[32m+%s\033[0m\n", diff.NewLines[i])
					} else {
						fmt.Fprintf(w, "+%s\n", diff.NewLines[i])
					}
				}
			}
		}
		
		fmt.Fprintln(w)
	}
}

func printSideBySide(w io.Writer, diffs []FileDiff, contextLines int, showSame bool, colorize bool) {
	for _, diff := range diffs {
		if diff.Status == "unchanged" && !showSame {
			continue
		}
		
		fmt.Fprintf(w, "=== %s (%s) ===\n", diff.Path, diff.Status)
		
		switch diff.Status {
		case "added":
			for _, line := range diff.NewLines {
				if colorize {
					fmt.Fprintf(w, "%-50s | \033[32m+ %s\033[0m\n", "", line)
				} else {
					fmt.Fprintf(w, "%-50s | + %s\n", "", line)
				}
			}
			
		case "deleted":
			for _, line := range diff.OldLines {
				if colorize {
					fmt.Fprintf(w, "\033[31m- %s\033[0m | %-50s\n", line, "")
				} else {
					fmt.Fprintf(w, "- %s | %-50s\n", line, "")
				}
			}
			
		case "modified":
			maxLen := len(diff.OldLines)
			if len(diff.NewLines) > maxLen {
				maxLen = len(diff.NewLines)
			}
			
			for i := 0; i < maxLen; i++ {
				oldLine := ""
				if i < len(diff.OldLines) {
					oldLine = diff.OldLines[i]
				}
				newLine := ""
				if i < len(diff.NewLines) {
					newLine = diff.NewLines[i]
				}
				
				if oldLine == newLine {
					fmt.Fprintf(w, "  %-48s |   %-48s\n", oldLine, newLine)
				} else {
					if colorize {
						fmt.Fprintf(w, "\033[31m- %-48s\033[0m | \033[32m+ %-48s\033[0m\n", oldLine, newLine)
					} else {
						fmt.Fprintf(w, "- %-48s | + %-48s\n", oldLine, newLine)
					}
				}
			}
		}
		
		fmt.Fprintln(w)
	}
}
