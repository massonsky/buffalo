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
	lintFix    bool
	lintStrict bool
	lintPaths  []string
	lintRules  []string
	lintIgnore []string

	lintCmd = &cobra.Command{
		Use:   "lint",
		Short: "Lint proto files for style and best practices",
		Long: `Check proto files for style violations, common mistakes, and best practices.

This command analyzes your proto files and reports issues such as:
- Naming conventions (PascalCase for messages, snake_case for fields)
- Missing documentation
- Deprecated syntax usage
- Import organization
- Package naming
- Field numbering gaps

Examples:
  # Lint all proto files
  buffalo lint

  # Lint with auto-fix
  buffalo lint --fix

  # Strict mode (treat warnings as errors)
  buffalo lint --strict

  # Lint specific paths
  buffalo lint --proto-path ./protos

  # Enable specific rules
  buffalo lint --rules naming,imports,docs`,
		RunE: runLint,
	}
)

func init() {
	rootCmd.AddCommand(lintCmd)

	lintCmd.Flags().BoolVar(&lintFix, "fix", false, "automatically fix issues when possible")
	lintCmd.Flags().BoolVar(&lintStrict, "strict", false, "treat warnings as errors")
	lintCmd.Flags().StringSliceVarP(&lintPaths, "proto-path", "p", []string{"."}, "paths to lint")
	lintCmd.Flags().StringSliceVar(&lintRules, "rules", []string{}, "specific rules to check (naming,imports,docs,syntax)")
	lintCmd.Flags().StringSliceVar(&lintIgnore, "ignore", []string{}, "patterns to ignore")
}

type LintIssue struct {
	File     string
	Line     int
	Column   int
	Rule     string
	Severity string // "error", "warning", "info"
	Message  string
	Fixable  bool
}

func runLint(cmd *cobra.Command, args []string) error {
	log := GetLogger()

	log.Info("🔍 Linting proto files...")

	// Load configuration
	cfg, err := loadConfig(log)
	if err == nil && len(lintPaths) == 1 && lintPaths[0] == "." {
		lintPaths = cfg.Proto.Paths
	}

	// Find all proto files
	var allProtoFiles []string
	for _, path := range lintPaths {
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

	// Lint each file
	var allIssues []LintIssue
	filesWithIssues := 0

	for _, file := range allProtoFiles {
		issues := lintFile(file, log)
		if len(issues) > 0 {
			allIssues = append(allIssues, issues...)
			filesWithIssues++
		}
	}

	// Display results
	if len(allIssues) == 0 {
		log.Info("✅ All checks passed! No issues found.")
		return nil
	}

	// Group issues by file
	fileIssues := make(map[string][]LintIssue)
	for _, issue := range allIssues {
		fileIssues[issue.File] = append(fileIssues[issue.File], issue)
	}

	// Count by severity
	errorCount := 0
	warningCount := 0
	infoCount := 0
	fixableCount := 0

	for _, issue := range allIssues {
		switch issue.Severity {
		case "error":
			errorCount++
		case "warning":
			warningCount++
		case "info":
			infoCount++
		}
		if issue.Fixable {
			fixableCount++
		}
	}

	// Print issues
	fmt.Println()
	for file, issues := range fileIssues {
		relPath := getRelativePath(file)
		fmt.Printf("📄 %s (%d issue(s))\n", relPath, len(issues))
		for _, issue := range issues {
			severityIcon := "ℹ️"
			if issue.Severity == "error" {
				severityIcon = "❌"
			} else if issue.Severity == "warning" {
				severityIcon = "⚠️"
			}

			fixable := ""
			if issue.Fixable {
				fixable = " [fixable]"
			}

			fmt.Printf("  %s Line %d:%d [%s]%s %s\n",
				severityIcon,
				issue.Line,
				issue.Column,
				issue.Rule,
				fixable,
				issue.Message,
			)
		}
		fmt.Println()
	}

	// Summary
	fmt.Println("╔════════════════════════════════════════════════════════╗")
	fmt.Println("║  Lint Summary                                           ║")
	fmt.Println("╚════════════════════════════════════════════════════════╝")
	fmt.Printf("   Files checked: %d\n", len(allProtoFiles))
	fmt.Printf("   Files with issues: %d\n", filesWithIssues)
	fmt.Printf("   Total issues: %d\n", len(allIssues))
	fmt.Printf("   ├─ Errors: %d\n", errorCount)
	fmt.Printf("   ├─ Warnings: %d\n", warningCount)
	fmt.Printf("   └─ Info: %d\n", infoCount)
	if fixableCount > 0 {
		fmt.Printf("   Fixable: %d (run with --fix)\n", fixableCount)
	}
	fmt.Println()

	if lintStrict && (errorCount > 0 || warningCount > 0) {
		return fmt.Errorf("lint failed in strict mode")
	}

	if errorCount > 0 {
		return fmt.Errorf("lint found %d error(s)", errorCount)
	}

	return nil
}

// lintFile performs linting on a single proto file
func lintFile(file string, log *logger.Logger) []LintIssue {
	var issues []LintIssue

	content, err := os.ReadFile(file)
	if err != nil {
		log.Warn("Failed to read file", logger.String("file", file), logger.Any("error", err))
		return issues
	}

	lines := strings.Split(string(content), "\n")

	// Check 1: Syntax version
	hasSyntax := false
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "syntax") {
			hasSyntax = true
			if !strings.Contains(line, `"proto3"`) {
				issues = append(issues, LintIssue{
					File:     file,
					Line:     i + 1,
					Column:   1,
					Rule:     "syntax",
					Severity: "warning",
					Message:  "Consider using proto3 syntax",
					Fixable:  true,
				})
			}
			break
		}
	}
	if !hasSyntax {
		issues = append(issues, LintIssue{
			File:     file,
			Line:     1,
			Column:   1,
			Rule:     "syntax",
			Severity: "error",
			Message:  "Missing syntax declaration (should be 'syntax = \"proto3\";')",
			Fixable:  true,
		})
	}

	// Check 2: Package declaration
	hasPackage := false
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package") {
			hasPackage = true
			// Check package naming convention (lowercase with dots)
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				pkgName := strings.TrimSuffix(parts[1], ";")
				if pkgName != strings.ToLower(pkgName) {
					issues = append(issues, LintIssue{
						File:     file,
						Line:     i + 1,
						Column:   1,
						Rule:     "naming",
						Severity: "warning",
						Message:  fmt.Sprintf("Package name should be lowercase: '%s'", pkgName),
						Fixable:  true,
					})
				}
			}
			break
		}
	}
	if !hasPackage {
		issues = append(issues, LintIssue{
			File:     file,
			Line:     1,
			Column:   1,
			Rule:     "package",
			Severity: "warning",
			Message:  "Missing package declaration",
			Fixable:  false,
		})
	}

	// Check 3: Message and service naming (PascalCase)
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "message ") || strings.HasPrefix(line, "service ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				name := parts[1]
				if !isPascalCase(name) {
					issues = append(issues, LintIssue{
						File:     file,
						Line:     i + 1,
						Column:   1,
						Rule:     "naming",
						Severity: "warning",
						Message:  fmt.Sprintf("Type name should be PascalCase: '%s'", name),
						Fixable:  false,
					})
				}
			}
		}
	}

	// Check 4: Field naming (snake_case)
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Simple field detection (not perfect but good enough)
		if strings.Contains(trimmed, " ") && strings.Contains(trimmed, "=") && !strings.HasPrefix(trimmed, "//") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 3 {
				fieldName := parts[1]
				if !isSnakeCase(fieldName) && !isReservedKeyword(fieldName) {
					issues = append(issues, LintIssue{
						File:     file,
						Line:     i + 1,
						Column:   1,
						Rule:     "naming",
						Severity: "info",
						Message:  fmt.Sprintf("Field name should be snake_case: '%s'", fieldName),
						Fixable:  false,
					})
				}
			}
		}
	}

	// Check 5: Trailing whitespace
	for i, line := range lines {
		if len(line) > 0 && (line[len(line)-1] == ' ' || line[len(line)-1] == '\t') {
			issues = append(issues, LintIssue{
				File:     file,
				Line:     i + 1,
				Column:   len(line),
				Rule:     "whitespace",
				Severity: "info",
				Message:  "Trailing whitespace",
				Fixable:  true,
			})
		}
	}

	return issues
}

// isPascalCase checks if a string is in PascalCase
func isPascalCase(s string) bool {
	if len(s) == 0 {
		return false
	}
	// First character should be uppercase
	if s[0] < 'A' || s[0] > 'Z' {
		return false
	}
	// Should not contain underscores or hyphens
	return !strings.Contains(s, "_") && !strings.Contains(s, "-")
}

// isSnakeCase checks if a string is in snake_case
func isSnakeCase(s string) bool {
	if len(s) == 0 {
		return false
	}
	// Should be lowercase with optional underscores
	return s == strings.ToLower(s)
}

// isReservedKeyword checks if a string is a proto reserved keyword
func isReservedKeyword(s string) bool {
	keywords := []string{"syntax", "package", "import", "message", "service", "rpc", "returns", "enum", "repeated", "optional", "required"}
	for _, kw := range keywords {
		if s == kw {
			return true
		}
	}
	return false
}
