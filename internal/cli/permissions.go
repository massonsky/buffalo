package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/massonsky/buffalo/internal/permissions"
	"github.com/massonsky/buffalo/pkg/errors"
	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/spf13/cobra"
)

var permissionsCmd = &cobra.Command{
	Use:   "permissions",
	Short: "Manage gRPC permission annotations",
	Long: `Buffalo permissions commands for managing RBAC/ABAC in proto files.

Extract, analyze, and generate permission code from proto file annotations.
Supports generating code for Go, Casbin, and OPA frameworks.`,
	Aliases: []string{"perms", "perm"},
}

var permissionsGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate permission code",
	Long: `Generate permission enforcement code from proto annotations.

Supports multiple output frameworks:
  - go: Native Go permission checker
  - casbin: Casbin model and policy files
  - opa: Open Policy Agent Rego policy`,
	Example: `  # Generate Go code
  buffalo permissions generate --framework go --output permissions.go

  # Generate Casbin policy
  buffalo permissions generate --framework casbin --output policy.csv

  # Generate OPA policy
  buffalo permissions generate --framework opa --output policy.rego`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		protoDir, _ := cmd.Flags().GetString("proto")
		framework, _ := cmd.Flags().GetString("framework")
		output, _ := cmd.Flags().GetString("output")
		pkg, _ := cmd.Flags().GetString("package")
		constants, _ := cmd.Flags().GetBool("constants")

		// Parse proto files
		parser := permissions.NewParser()
		services, err := parser.ParseDirectory(ctx, protoDir)
		if err != nil {
			return err
		}

		if len(services) == 0 {
			log.Warn("No permission annotations found in proto files")
			return nil
		}

		log.Info("Found services with permission annotations", logger.Int("count", len(services)))

		// Generate code
		gen := permissions.NewGenerator(permissions.GeneratorOptions{
			Framework:         framework,
			Package:           pkg,
			GenerateConstants: constants,
		})

		code, err := gen.Generate(services)
		if err != nil {
			return err
		}

		// Write output
		if output == "" || output == "-" {
			fmt.Println(code)
		} else {
			if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(output, []byte(code), 0644); err != nil {
				return errors.Wrap(err, errors.ErrIO, "failed to write output")
			}
			log.Info("✅ Generated", logger.String("file", output))
		}

		return nil
	},
}

var permissionsMatrixCmd = &cobra.Command{
	Use:   "matrix",
	Short: "Generate permission access matrix",
	Long: `Generate a visual permission access matrix.

Creates a table showing which roles/scopes have access to each method.`,
	Example: `  # Generate text matrix
  buffalo permissions matrix

  # Generate HTML matrix
  buffalo permissions matrix --format html --output matrix.html

  # Generate Markdown
  buffalo permissions matrix --format markdown`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		protoDir, _ := cmd.Flags().GetString("proto")
		format, _ := cmd.Flags().GetString("format")
		output, _ := cmd.Flags().GetString("output")

		// Parse proto files
		parser := permissions.NewParser()
		services, err := parser.ParseDirectory(ctx, protoDir)
		if err != nil {
			return err
		}

		if len(services) == 0 {
			log.Warn("No permission annotations found")
			return nil
		}

		// Build matrix
		matrix := permissions.BuildMatrix(services)

		// Render
		var content string
		switch format {
		case "html":
			content, err = matrix.RenderHTML()
			if err != nil {
				return err
			}
		case "markdown", "md":
			content = matrix.RenderMarkdown()
		default:
			content = matrix.RenderText()
		}

		// Output
		if output == "" || output == "-" {
			fmt.Println(content)
		} else {
			if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(output, []byte(content), 0644); err != nil {
				return errors.Wrap(err, errors.ErrIO, "failed to write output")
			}
			log.Info("✅ Generated matrix", logger.String("file", output))
		}

		return nil
	},
}

var permissionsAuditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit permissions for security issues",
	Long: `Analyze permission annotations for potential security issues.

Checks for common problems like:
  - Missing roles or scopes (no authorization)
  - Admin access without MFA
  - Public write endpoints
  - Missing audit logging
  - Inconsistent naming`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		protoDir, _ := cmd.Flags().GetString("proto")
		severity, _ := cmd.Flags().GetString("severity")
		format, _ := cmd.Flags().GetString("format")

		// Parse proto files
		parser := permissions.NewParser()
		services, err := parser.ParseDirectory(ctx, protoDir)
		if err != nil {
			return err
		}

		if len(services) == 0 {
			log.Warn("No permission annotations found")
			return nil
		}

		// Run audit
		analyzer := permissions.NewAnalyzer()
		issues := analyzer.Audit(ctx, services)

		// Filter by severity
		var filtered []permissions.AuditIssue
		minSeverity := parseSeverity(severity)
		for _, issue := range issues {
			if issue.Severity >= minSeverity {
				filtered = append(filtered, issue)
			}
		}

		if len(filtered) == 0 {
			log.Info("✅ No issues found!")
			return nil
		}

		// Output
		if format == "json" {
			fmt.Println("[")
			for i, issue := range filtered {
				comma := ","
				if i == len(filtered)-1 {
					comma = ""
				}
				fmt.Printf(`  {"rule": "%s", "severity": "%d", "service": "%s", "method": "%s", "message": "%s", "fix": "%s"}%s`+"\n",
					issue.RuleID, issue.Severity, issue.Service, issue.Method, issue.Message, issue.Fix, comma)
			}
			fmt.Println("]")
		} else {
			// Group by severity
			var errors, warnings, infos []permissions.AuditIssue
			for _, issue := range filtered {
				switch issue.Severity {
				case permissions.SeverityError:
					errors = append(errors, issue)
				case permissions.SeverityWarning:
					warnings = append(warnings, issue)
				case permissions.SeverityInfo:
					infos = append(infos, issue)
				}
			}

			if len(errors) > 0 {
				log.Error(fmt.Sprintf("\nErrors (%d):", len(errors)))
				for _, issue := range errors {
					printIssue(issue)
				}
			}

			if len(warnings) > 0 {
				log.Warn(fmt.Sprintf("\nWarnings (%d):", len(warnings)))
				for _, issue := range warnings {
					printIssue(issue)
				}
			}

			if len(infos) > 0 {
				log.Info(fmt.Sprintf("\nInfo (%d):", len(infos)))
				for _, issue := range infos {
					printIssue(issue)
				}
			}

			fmt.Printf("\nTotal: %d errors, %d warnings, %d info\n", len(errors), len(warnings), len(infos))
		}

		// Exit with error if there are errors
		for _, issue := range filtered {
			if issue.Severity == permissions.SeverityError {
				return errors.New(errors.ErrValidation, "audit found errors")
			}
		}

		return nil
	},
}

var permissionsDiffCmd = &cobra.Command{
	Use:   "diff [old-dir] [new-dir]",
	Short: "Compare permission changes",
	Long: `Compare permissions between two versions.

Useful for code review and change tracking.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		oldDir := args[0]
		newDir := args[1]

		// Parse old
		oldParser := permissions.NewParser()
		oldServices, err := oldParser.ParseDirectory(ctx, oldDir)
		if err != nil {
			return errors.Wrap(err, errors.ErrIO, "failed to parse old permissions")
		}

		// Parse new
		newParser := permissions.NewParser()
		newServices, err := newParser.ParseDirectory(ctx, newDir)
		if err != nil {
			return errors.Wrap(err, errors.ErrIO, "failed to parse new permissions")
		}

		// Calculate diff
		analyzer := permissions.NewAnalyzer()
		diffs := analyzer.Diff(ctx, oldServices, newServices)

		if len(diffs) == 0 {
			log.Info("✅ No permission changes detected")
			return nil
		}

		// Display diffs
		var added, modified, removed int
		for _, d := range diffs {
			switch d.Type {
			case permissions.DiffAdded:
				added++
				fmt.Printf("+ %s.%s\n", d.Service, d.Method)
				if d.New != nil {
					fmt.Printf("  Action: %s\n", d.New.Action)
					fmt.Printf("  Roles: %v\n", d.New.Roles)
				}
			case permissions.DiffRemoved:
				removed++
				fmt.Printf("- %s.%s\n", d.Service, d.Method)
			case permissions.DiffModified:
				modified++
				fmt.Printf("~ %s.%s\n", d.Service, d.Method)
				if d.Old != nil && d.New != nil {
					if !stringSliceEqual(d.Old.Roles, d.New.Roles) {
						fmt.Printf("  Roles: %v -> %v\n", d.Old.Roles, d.New.Roles)
					}
					if !stringSliceEqual(d.Old.Scopes, d.New.Scopes) {
						fmt.Printf("  Scopes: %v -> %v\n", d.Old.Scopes, d.New.Scopes)
					}
					if d.Old.Public != d.New.Public {
						fmt.Printf("  Public: %v -> %v\n", d.Old.Public, d.New.Public)
					}
					if d.Old.RequireMFA != d.New.RequireMFA {
						fmt.Printf("  RequireMFA: %v -> %v\n", d.Old.RequireMFA, d.New.RequireMFA)
					}
				}
			}
		}

		fmt.Printf("\nSummary: +%d added, ~%d modified, -%d removed\n", added, modified, removed)

		return nil
	},
}

var permissionsSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show permission statistics",
	Long:  `Display summary statistics about permission annotations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		protoDir, _ := cmd.Flags().GetString("proto")

		// Parse proto files
		parser := permissions.NewParser()
		services, err := parser.ParseDirectory(ctx, protoDir)
		if err != nil {
			return err
		}

		if len(services) == 0 {
			log.Warn("No permission annotations found")
			return nil
		}

		// Get summary
		analyzer := permissions.NewAnalyzer()
		summary := analyzer.Summary(services)

		fmt.Println("Permission Summary")
		fmt.Println(strings.Repeat("=", 40))
		fmt.Printf("Services:        %d\n", summary.ServiceCount)
		fmt.Printf("Methods:         %d\n", summary.MethodCount)
		fmt.Printf("Public Methods:  %d\n", summary.PublicCount)
		fmt.Println()

		if len(summary.ByRole) > 0 {
			fmt.Println("Methods by Role:")
			for role, count := range summary.ByRole {
				fmt.Printf("  %-15s %d\n", role, count)
			}
			fmt.Println()
		}

		if len(summary.ByScope) > 0 {
			fmt.Println("Methods by Scope:")
			for scope, count := range summary.ByScope {
				fmt.Printf("  %-15s %d\n", scope, count)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(permissionsCmd)

	// Subcommands
	permissionsCmd.AddCommand(permissionsGenerateCmd)
	permissionsCmd.AddCommand(permissionsMatrixCmd)
	permissionsCmd.AddCommand(permissionsAuditCmd)
	permissionsCmd.AddCommand(permissionsDiffCmd)
	permissionsCmd.AddCommand(permissionsSummaryCmd)

	// Global flags
	permissionsCmd.PersistentFlags().StringP("proto", "p", "protos", "proto files directory")

	// Generate flags
	permissionsGenerateCmd.Flags().StringP("framework", "f", "go", "output framework (go, casbin, opa)")
	permissionsGenerateCmd.Flags().StringP("output", "o", "", "output file (default: stdout)")
	permissionsGenerateCmd.Flags().String("package", "permissions", "Go package name")
	permissionsGenerateCmd.Flags().Bool("constants", true, "generate constants for actions/roles/scopes")

	// Matrix flags
	permissionsMatrixCmd.Flags().StringP("format", "f", "text", "output format (text, html, markdown)")
	permissionsMatrixCmd.Flags().StringP("output", "o", "", "output file (default: stdout)")

	// Audit flags
	permissionsAuditCmd.Flags().StringP("severity", "s", "info", "minimum severity (info, warning, error)")
	permissionsAuditCmd.Flags().StringP("format", "f", "text", "output format (text, json)")
}

func parseSeverity(s string) permissions.IssueSeverity {
	switch strings.ToLower(s) {
	case "error":
		return permissions.SeverityError
	case "warning", "warn":
		return permissions.SeverityWarning
	default:
		return permissions.SeverityInfo
	}
}

func printIssue(issue permissions.AuditIssue) {
	fmt.Printf("  [%s] %s.%s\n", issue.RuleID, issue.Service, issue.Method)
	fmt.Printf("    %s\n", issue.Message)
	if issue.Fix != "" {
		fmt.Printf("    Fix: %s\n", issue.Fix)
	}
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
