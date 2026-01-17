package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/massonsky/buffalo/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	formatWrite  bool
	formatCheck  bool
	formatPaths  []string
	formatIndent int

	formatCmd = &cobra.Command{
		Use:     "format",
		Aliases: []string{"fmt"},
		Short:   "Format proto files",
		Long: `Format proto files according to style guidelines.

This command formats your proto files with consistent indentation,
spacing, and organization. It can either write changes to files
or just check if files are properly formatted.

Examples:
  # Check formatting (dry run)
  buffalo format

  # Format and write changes
  buffalo format --write

  # Format with custom indentation
  buffalo format --write --indent 4

  # Check if files are formatted
  buffalo format --check`,
		RunE: runFormat,
	}
)

func init() {
	rootCmd.AddCommand(formatCmd)

	formatCmd.Flags().BoolVarP(&formatWrite, "write", "w", false, "write formatted output to files")
	formatCmd.Flags().BoolVar(&formatCheck, "check", false, "check if files are formatted (exit with error if not)")
	formatCmd.Flags().StringSliceVarP(&formatPaths, "proto-path", "p", []string{"."}, "paths to format")
	formatCmd.Flags().IntVar(&formatIndent, "indent", 2, "indentation spaces")
}

func runFormat(cmd *cobra.Command, args []string) error {
	log := GetLogger()

	if formatCheck && formatWrite {
		return fmt.Errorf("--check and --write cannot be used together")
	}

	mode := "check"
	if formatWrite {
		mode = "write"
	} else if formatCheck {
		mode = "check-strict"
	}

	log.Info("🎨 Formatting proto files", logger.String("mode", mode))

	// Load configuration
	cfg, err := loadConfig(log)
	if err == nil && len(formatPaths) == 1 && formatPaths[0] == "." {
		formatPaths = cfg.Proto.Paths
	}

	// Find all proto files
	var allProtoFiles []string
	for _, path := range formatPaths {
		fileInfos, err := utils.FindFiles(path, utils.FindFilesOptions{
			Pattern:   "*.proto",
			Recursive: true,
		})
		if err != nil {
			log.Warn("Failed to scan path", logger.String("path", path), logger.Any("error", err))
			continue
		}
		for _, fi := range fileInfos {
			allProtoFiles = append(allProtoFiles, fi.Path)
		}
	}

	if len(allProtoFiles) == 0 {
		log.Warn("⚠️  No proto files found")
		return nil
	}

	log.Info("Found proto files", logger.Int("count", len(allProtoFiles)))

	// Format each file
	formattedCount := 0
	unformattedFiles := []string{}

	for _, file := range allProtoFiles {
		formatted, err := formatFile(file, formatWrite, formatIndent, log)
		if err != nil {
			log.Warn("Failed to format file", logger.String("file", file), logger.Any("error", err))
			continue
		}

		if formatted {
			formattedCount++
			if !formatWrite {
				unformattedFiles = append(unformattedFiles, file)
			}
		}
	}

	// Results
	if formatWrite {
		if formattedCount > 0 {
			log.Info("✅ Formatted files", logger.Int("count", formattedCount))
		} else {
			log.Info("✅ All files already formatted")
		}
		return nil
	}

	if len(unformattedFiles) > 0 {
		log.Warn("⚠️  Found unformatted files:")
		for _, file := range unformattedFiles {
			log.Warn(fmt.Sprintf("  • %s", getRelativePath(file)))
		}
		if formatCheck {
			return fmt.Errorf("found %d unformatted file(s)", len(unformattedFiles))
		}
		log.Info("\n💡 Run 'buffalo format --write' to format them")
		return nil
	}

	log.Info("✅ All files are properly formatted")
	return nil
}

// formatFile formats a single proto file
func formatFile(file string, write bool, indent int, log *logger.Logger) (bool, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return false, err
	}

	original := string(content)
	formatted := formatProtoContent(original, indent)

	// Check if formatting changed anything
	if original == formatted {
		return false, nil
	}

	if write {
		if err := os.WriteFile(file, []byte(formatted), 0644); err != nil {
			return false, err
		}
		log.Debug("Formatted", logger.String("file", file))
	}

	return true, nil
}

// formatProtoContent formats proto file content
func formatProtoContent(content string, indent int) string {
	lines := strings.Split(content, "\n")
	var result []string
	currentIndent := 0
	indentStr := strings.Repeat(" ", indent)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines (but preserve them)
		if trimmed == "" {
			result = append(result, "")
			continue
		}

		// Skip comments at their current position
		if strings.HasPrefix(trimmed, "//") {
			result = append(result, strings.Repeat(indentStr, currentIndent)+trimmed)
			continue
		}

		// Decrease indent for closing braces
		if trimmed == "}" {
			currentIndent--
			if currentIndent < 0 {
				currentIndent = 0
			}
		}

		// Add properly indented line
		result = append(result, strings.Repeat(indentStr, currentIndent)+trimmed)

		// Increase indent for opening braces
		if strings.HasSuffix(trimmed, "{") {
			currentIndent++
		}
	}

	return strings.Join(result, "\n")
}
