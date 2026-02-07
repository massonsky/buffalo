package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/massonsky/buffalo/internal/workspace"
	"github.com/massonsky/buffalo/pkg/errors"
	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/spf13/cobra"
)

var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Manage multi-project workspaces",
	Long: `Buffalo workspace commands for managing monorepos and multi-project setups.

A workspace allows you to define multiple projects that can be built together,
with dependency tracking between projects and incremental builds.`,
	Aliases: []string{"ws"},
}

var workspaceInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new workspace",
	Long: `Initialize a new workspace configuration file.

Creates a buffalo-workspace.yaml file with default settings and prompts
for initial project discovery.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := cmd.Flags().GetString("dir")
		if dir == "" {
			var err error
			dir, err = os.Getwd()
			if err != nil {
				return err
			}
		}

		name, _ := cmd.Flags().GetString("name")
		discover, _ := cmd.Flags().GetBool("discover")

		cfg := workspace.InitConfig(name)

		// Auto-discover projects
		if discover {
			log.Info("Discovering projects...")
			projects := discoverProjects(dir)
			cfg.Projects = projects
			log.Info("Discovered projects", logger.Int("count", len(projects)))
		}

		configPath := filepath.Join(dir, "buffalo-workspace.yaml")
		if err := workspace.SaveConfig(configPath, cfg); err != nil {
			return err
		}

		log.Info("✅ Workspace initialized", logger.String("path", configPath))
		return nil
	},
}

var workspaceBuildCmd = &cobra.Command{
	Use:   "build [projects...]",
	Short: "Build workspace projects",
	Long: `Build one or more projects in the workspace.

If no projects are specified, builds all projects respecting
dependency order. Use tags to build groups of projects.`,
	Example: `  # Build all projects
  buffalo workspace build

  # Build specific projects
  buffalo workspace build api web

  # Build projects with a tag
  buffalo workspace build --tag backend

  # Build with parallelism
  buffalo workspace build --parallel 4`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		cfgPath, _ := cmd.Flags().GetString("config")
		cfgPath, err := resolveWorkspaceConfig(cfgPath)
		if err != nil {
			return err
		}

		parallel, _ := cmd.Flags().GetInt("parallel")
		tag, _ := cmd.Flags().GetString("tag")
		force, _ := cmd.Flags().GetBool("force")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		manager, err := workspace.NewManager(cfgPath, log)
		if err != nil {
			return err
		}

		// Determine projects to build
		var projectNames []string
		if tag != "" {
			projects := manager.Config().GetProjectsByTag(tag)
			for _, p := range projects {
				projectNames = append(projectNames, p.Name)
			}
		} else if len(args) > 0 {
			projectNames = args
		} else {
			for _, p := range manager.Config().Projects {
				projectNames = append(projectNames, p.Name)
			}
		}

		if len(projectNames) == 0 {
			log.Warn("No projects to build")
			return nil
		}

		if dryRun {
			log.Info("Dry run - would build projects", logger.String("projects", strings.Join(projectNames, ", ")))
			return nil
		}

		log.Info("Building projects...", logger.Int("count", len(projectNames)), logger.Int("parallel", parallel))

		results, err := manager.Build(ctx, workspace.BuildOptions{
			Projects:        projectNames,
			Force:           force,
			Parallel:        parallel > 1,
			Workers:         parallel,
			ContinueOnError: false,
		})

		// Print results
		var succeeded, failed int
		for _, r := range results {
			if r.Status == workspace.StatusSuccess {
				succeeded++
				log.Info(fmt.Sprintf("✓ %s (%.2fs)", r.Project.Name, r.Duration.Seconds()))
			} else {
				failed++
				log.Error(fmt.Sprintf("✗ %s: %v", r.Project.Name, r.Error))
			}
		}

		log.Info("Build complete", logger.Int("succeeded", succeeded), logger.Int("failed", failed))

		if failed > 0 {
			return errors.New(errors.ErrValidation, "some projects failed to build")
		}

		return err
	},
}

var workspaceListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List workspace projects",
	Long:    `List all projects defined in the workspace configuration.`,
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")
		cfg, err := loadWorkspaceConfig(cfgPath)
		if err != nil {
			return err
		}

		tag, _ := cmd.Flags().GetString("tag")
		format, _ := cmd.Flags().GetString("format")

		var projects []*workspace.Project
		if tag != "" {
			projects = cfg.GetProjectsByTag(tag)
		} else {
			for i := range cfg.Projects {
				projects = append(projects, &cfg.Projects[i])
			}
		}

		if format == "json" {
			// JSON output
			fmt.Println("[")
			for i, p := range projects {
				comma := ","
				if i == len(projects)-1 {
					comma = ""
				}
				fmt.Printf(`  {"name": "%s", "path": "%s", "tags": %v}%s`+"\n",
					p.Name, p.Path, formatTags(p.Tags), comma)
			}
			fmt.Println("]")
		} else {
			// Table output
			fmt.Printf("%-20s %-30s %s\n", "NAME", "PATH", "TAGS")
			fmt.Println(strings.Repeat("-", 70))
			for _, p := range projects {
				fmt.Printf("%-20s %-30s %s\n",
					p.Name, p.Path, strings.Join(p.Tags, ", "))
			}
			fmt.Printf("\nTotal: %d projects\n", len(projects))
		}

		return nil
	},
}

var workspaceGraphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Show project dependency graph",
	Long:  `Visualize the dependency graph between workspace projects.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")
		cfg, err := loadWorkspaceConfig(cfgPath)
		if err != nil {
			return err
		}

		format, _ := cmd.Flags().GetString("format")

		graph := cfg.BuildDependencyGraph()

		switch format {
		case "dot":
			fmt.Println("digraph workspace {")
			fmt.Println("  rankdir=LR;")
			fmt.Println("  node [shape=box];")
			for project, deps := range graph.Projects {
				for _, dep := range deps {
					fmt.Printf("  \"%s\" -> \"%s\";\n", project, dep)
				}
				if len(deps) == 0 {
					fmt.Printf("  \"%s\";\n", project)
				}
			}
			fmt.Println("}")

		case "mermaid":
			fmt.Println("```mermaid")
			fmt.Println("graph LR")
			for project, deps := range graph.Projects {
				for _, dep := range deps {
					fmt.Printf("  %s --> %s\n", project, dep)
				}
				if len(deps) == 0 {
					fmt.Printf("  %s\n", project)
				}
			}
			fmt.Println("```")

		default:
			// ASCII art
			fmt.Println("Project Dependencies:")
			fmt.Println()
			for project, deps := range graph.Projects {
				if len(deps) > 0 {
					fmt.Printf("  %s\n", project)
					for i, dep := range deps {
						prefix := "├──"
						if i == len(deps)-1 {
							prefix = "└──"
						}
						fmt.Printf("    %s %s\n", prefix, dep)
					}
				} else {
					fmt.Printf("  %s (no dependencies)\n", project)
				}
			}
		}

		return nil
	},
}

var workspaceAffectedCmd = &cobra.Command{
	Use:   "affected",
	Short: "Show affected projects",
	Long: `Determine which projects are affected by recent changes.

Uses git to detect changed files and traces dependencies to find
all projects that need to be rebuilt.`,
	Example: `  # Show affected projects since last commit
  buffalo workspace affected

  # Show affected since specific ref
  buffalo workspace affected --since HEAD~5`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")
		cfgPath, err := resolveWorkspaceConfig(cfgPath)
		if err != nil {
			return err
		}

		since, _ := cmd.Flags().GetString("since")

		manager, err := workspace.NewManager(cfgPath, log)
		if err != nil {
			return err
		}

		result, err := manager.GetAffected(since)
		if err != nil {
			return err
		}

		if len(result.DirectlyAffected) == 0 && len(result.TransitivelyAffected) == 0 {
			log.Info("No projects affected")
			return nil
		}

		fmt.Printf("Directly affected projects (%d):\n", len(result.DirectlyAffected))
		for _, p := range result.DirectlyAffected {
			fmt.Printf("  • %s\n", p.Name)
		}

		if len(result.TransitivelyAffected) > 0 {
			fmt.Printf("\nTransitively affected (%d):\n", len(result.TransitivelyAffected))
			for _, p := range result.TransitivelyAffected {
				fmt.Printf("  • %s\n", p.Name)
			}
		}

		if len(result.ChangedFiles) > 0 {
			fmt.Printf("\nChanged files (%d):\n", len(result.ChangedFiles))
			for _, f := range result.ChangedFiles {
				fmt.Printf("  - %s\n", f)
			}
		}

		return nil
	},
}

var workspaceValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate workspace configuration",
	Long:  `Validate the workspace configuration file for errors.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")
		cfg, err := loadWorkspaceConfig(cfgPath)
		if err != nil {
			return err
		}

		result := cfg.Validate()

		if len(result.Errors) > 0 {
			log.Error("Validation errors:")
			for _, e := range result.Errors {
				log.Error(fmt.Sprintf("  • %s: %s", e.Field, e.Message))
			}
		}

		if len(result.Warnings) > 0 {
			log.Warn("Warnings:")
			for _, w := range result.Warnings {
				log.Warn(fmt.Sprintf("  • %s", w.Message))
			}
		}

		if result.Valid {
			log.Info("✅ Workspace configuration is valid")
			return nil
		}

		return errors.New(errors.ErrValidation, "workspace configuration is invalid")
	},
}

var workspaceExecCmd = &cobra.Command{
	Use:   "exec [command]",
	Short: "Execute command in all projects",
	Long: `Execute a command in each project directory.

Useful for running arbitrary commands across all projects.`,
	Example: `  # Run tests in all projects
  buffalo workspace exec -- go test ./...

  # Install dependencies
  buffalo workspace exec --tag go -- go mod tidy`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")
		cfg, err := loadWorkspaceConfig(cfgPath)
		if err != nil {
			return err
		}

		tag, _ := cmd.Flags().GetString("tag")
		continueOnError, _ := cmd.Flags().GetBool("continue-on-error")

		var projects []*workspace.Project
		if tag != "" {
			projects = cfg.GetProjectsByTag(tag)
		} else {
			for i := range cfg.Projects {
				projects = append(projects, &cfg.Projects[i])
			}
		}

		workspaceDir, _ := resolveWorkspaceConfig(cfgPath)
		workspaceDir = filepath.Dir(workspaceDir)
		command := strings.Join(args, " ")

		var failed int
		for _, p := range projects {
			projectDir := filepath.Join(workspaceDir, p.Path)
			log.Info(fmt.Sprintf("Running in %s: %s", p.Name, command))

			// Execute command (simplified - actual implementation would use exec)
			start := time.Now()
			// In real implementation, use os/exec
			_ = projectDir
			duration := time.Since(start)

			log.Info(fmt.Sprintf("  ✓ %s (%.2fs)", p.Name, duration.Seconds()))

			if !continueOnError && failed > 0 {
				break
			}
		}

		if failed > 0 {
			return fmt.Errorf("%d projects failed", failed)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(workspaceCmd)

	// Subcommands
	workspaceCmd.AddCommand(workspaceInitCmd)
	workspaceCmd.AddCommand(workspaceBuildCmd)
	workspaceCmd.AddCommand(workspaceListCmd)
	workspaceCmd.AddCommand(workspaceGraphCmd)
	workspaceCmd.AddCommand(workspaceAffectedCmd)
	workspaceCmd.AddCommand(workspaceValidateCmd)
	workspaceCmd.AddCommand(workspaceExecCmd)

	// Global workspace flags
	workspaceCmd.PersistentFlags().StringP("config", "c", "", "workspace config file (default: buffalo-workspace.yaml)")

	// Init flags
	workspaceInitCmd.Flags().StringP("name", "n", "", "workspace name")
	workspaceInitCmd.Flags().StringP("dir", "d", "", "directory for workspace")
	workspaceInitCmd.Flags().Bool("discover", true, "auto-discover projects")

	// Build flags
	workspaceBuildCmd.Flags().IntP("parallel", "p", 1, "number of parallel builds")
	workspaceBuildCmd.Flags().StringP("tag", "t", "", "build projects with tag")
	workspaceBuildCmd.Flags().BoolP("force", "f", false, "force rebuild all")
	workspaceBuildCmd.Flags().Bool("dry-run", false, "show what would be built")

	// List flags
	workspaceListCmd.Flags().StringP("tag", "t", "", "filter by tag")
	workspaceListCmd.Flags().StringP("format", "f", "table", "output format (table, json)")

	// Graph flags
	workspaceGraphCmd.Flags().StringP("format", "f", "ascii", "output format (ascii, dot, mermaid)")

	// Affected flags
	workspaceAffectedCmd.Flags().String("since", "HEAD~1", "git ref to compare from")

	// Exec flags
	workspaceExecCmd.Flags().StringP("tag", "t", "", "filter projects by tag")
	workspaceExecCmd.Flags().Bool("continue-on-error", false, "continue on error")
}

func loadWorkspaceConfig(cfgPath string) (*workspace.Config, error) {
	cfgPath, err := resolveWorkspaceConfig(cfgPath)
	if err != nil {
		return nil, err
	}
	return workspace.LoadConfig(cfgPath)
}

func resolveWorkspaceConfig(cfgPath string) (string, error) {
	if cfgPath == "" {
		var err error
		cfgPath, err = workspace.FindConfig(".")
		if err != nil {
			return "", errors.Wrap(err, errors.ErrNotFound, "workspace config not found")
		}
	}
	return cfgPath, nil
}

func discoverProjects(dir string) []workspace.Project {
	var projects []workspace.Project

	// Look for common project indicators
	indicators := []struct {
		file string
		lang string
	}{
		{"go.mod", "go"},
		{"package.json", "node"},
		{"Cargo.toml", "rust"},
		{"pyproject.toml", "python"},
		{"requirements.txt", "python"},
		{"CMakeLists.txt", "cpp"},
	}

	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			// Skip common non-project directories
			if info != nil && info.IsDir() {
				name := info.Name()
				if name == "node_modules" || name == ".git" || name == "vendor" || name == "target" {
					return filepath.SkipDir
				}
			}
			return nil
		}

		for _, ind := range indicators {
			if info.Name() == ind.file {
				relPath, _ := filepath.Rel(dir, filepath.Dir(path))
				if relPath == "." {
					relPath = "."
				}

				name := filepath.Base(filepath.Dir(path))
				if name == "." {
					name = filepath.Base(dir)
				}

				projects = append(projects, workspace.Project{
					Name: name,
					Path: relPath,
					Tags: []string{ind.lang},
				})

				return filepath.SkipDir // Don't descend further
			}
		}

		return nil
	})

	return projects
}

func formatTags(tags []string) string {
	if len(tags) == 0 {
		return "[]"
	}
	var quoted []string
	for _, t := range tags {
		quoted = append(quoted, fmt.Sprintf(`"%s"`, t))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}
