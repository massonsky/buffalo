package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/massonsky/buffalo/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	depsPaths      []string
	depsReverse    bool
	depsTransitive bool
	depsFormat     string

	depsCmd = &cobra.Command{
		Use:     "deps",
		Aliases: []string{"dependencies"},
		Short:   "Show dependencies between proto files",
		Long: `Analyze and display dependencies between proto files.

This command parses import statements in proto files and shows
the dependency graph. It can show direct dependencies or the
full transitive dependency tree.

Examples:
  # Show all dependencies
  buffalo deps

  # Show what depends on a file (reverse dependencies)
  buffalo deps --reverse

  # Include transitive dependencies
  buffalo deps --transitive

  # Output as tree
  buffalo deps --format tree`,
		RunE: runDeps,
	}
)

func init() {
	rootCmd.AddCommand(depsCmd)

	depsCmd.Flags().StringSliceVarP(&depsPaths, "proto-path", "p", []string{"."}, "paths to analyze")
	depsCmd.Flags().BoolVarP(&depsReverse, "reverse", "r", false, "show reverse dependencies (what depends on what)")
	depsCmd.Flags().BoolVarP(&depsTransitive, "transitive", "t", false, "include transitive dependencies")
	depsCmd.Flags().StringVar(&depsFormat, "format", "list", "output format: list, tree, dot")
}

type ProtoDependency struct {
	File       string
	Imports    []string
	ImportedBy []string
}

func runDeps(cmd *cobra.Command, args []string) error {
	log := GetLogger()

	log.Info("🔗 Analyzing dependencies...")

	// Load configuration
	cfg, err := loadConfig(log)
	if err == nil && len(depsPaths) == 1 && depsPaths[0] == "." {
		depsPaths = cfg.Proto.Paths
	}

	// Find all proto files
	var allProtoFiles []string
	for _, path := range depsPaths {
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

	// Parse dependencies
	deps := make(map[string]*ProtoDependency)
	for _, file := range allProtoFiles {
		imports := parseImports(file)
		deps[file] = &ProtoDependency{
			File:    file,
			Imports: imports,
		}
	}

	// Build reverse dependencies
	for file, dep := range deps {
		for _, imp := range dep.Imports {
			// Find the full path of the imported file
			for depFile := range deps {
				if strings.HasSuffix(depFile, imp) {
					if deps[depFile].ImportedBy == nil {
						deps[depFile].ImportedBy = []string{}
					}
					deps[depFile].ImportedBy = append(deps[depFile].ImportedBy, file)
				}
			}
		}
	}

	// Display results based on format
	switch depsFormat {
	case "tree":
		displayDepsTree(deps, allProtoFiles)
	case "dot":
		displayDepsDot(deps, allProtoFiles)
	default:
		displayDepsList(deps, allProtoFiles, depsReverse)
	}

	// Statistics
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════╗")
	fmt.Println("║  Dependency Statistics                                  ║")
	fmt.Println("╚════════════════════════════════════════════════════════╝")

	totalImports := 0
	maxImports := 0
	maxImportFile := ""
	filesWithoutDeps := 0

	for file, dep := range deps {
		importCount := len(dep.Imports)
		totalImports += importCount

		if importCount == 0 && len(dep.ImportedBy) == 0 {
			filesWithoutDeps++
		}

		if importCount > maxImports {
			maxImports = importCount
			maxImportFile = file
		}
	}

	fmt.Printf("   Total files: %d\n", len(allProtoFiles))
	fmt.Printf("   Total imports: %d\n", totalImports)
	fmt.Printf("   Avg imports per file: %.1f\n", float64(totalImports)/float64(len(allProtoFiles)))
	fmt.Printf("   Files without dependencies: %d\n", filesWithoutDeps)
	if maxImportFile != "" {
		fmt.Printf("   Most imports: %s (%d)\n", filepath.Base(maxImportFile), maxImports)
	}
	fmt.Println()

	return nil
}

// parseImports extracts import statements from a proto file
func parseImports(file string) []string {
	content, err := os.ReadFile(file)
	if err != nil {
		return nil
	}

	var imports []string
	importRegex := regexp.MustCompile(`import\s+"([^"]+)";`)
	matches := importRegex.FindAllSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 {
			imports = append(imports, string(match[1]))
		}
	}

	return imports
}

// displayDepsList displays dependencies as a list
func displayDepsList(deps map[string]*ProtoDependency, files []string, reverse bool) {
	sort.Strings(files)

	fmt.Println()
	if reverse {
		fmt.Println("📦 Reverse Dependencies (what imports what):")
	} else {
		fmt.Println("📦 Dependencies (what each file imports):")
	}
	fmt.Println()

	for _, file := range files {
		dep := deps[file]
		relPath := getRelativePath(file)

		if reverse {
			if len(dep.ImportedBy) > 0 {
				fmt.Printf("📄 %s\n", relPath)
				fmt.Printf("   Imported by %d file(s):\n", len(dep.ImportedBy))
				for _, importer := range dep.ImportedBy {
					fmt.Printf("   • %s\n", getRelativePath(importer))
				}
				fmt.Println()
			}
		} else {
			if len(dep.Imports) > 0 {
				fmt.Printf("📄 %s\n", relPath)
				fmt.Printf("   Imports %d file(s):\n", len(dep.Imports))
				for _, imp := range dep.Imports {
					fmt.Printf("   • %s\n", imp)
				}
				fmt.Println()
			}
		}
	}
}

// displayDepsTree displays dependencies as a tree
func displayDepsTree(deps map[string]*ProtoDependency, files []string) {
	fmt.Println()
	fmt.Println("🌳 Dependency Tree:")
	fmt.Println()

	// Find root files (files with no dependencies or not imported)
	var roots []string
	for _, file := range files {
		dep := deps[file]
		if len(dep.Imports) == 0 || len(dep.ImportedBy) == 0 {
			roots = append(roots, file)
		}
	}

	visited := make(map[string]bool)
	for _, root := range roots {
		displayTreeNode(root, deps, 0, visited)
	}
}

func displayTreeNode(file string, deps map[string]*ProtoDependency, level int, visited map[string]bool) {
	if visited[file] {
		return
	}
	visited[file] = true

	indent := strings.Repeat("  ", level)
	prefix := "├─"
	if level == 0 {
		prefix = ""
	}

	relPath := getRelativePath(file)
	fmt.Printf("%s%s 📄 %s\n", indent, prefix, relPath)

	dep := deps[file]
	for i, imp := range dep.Imports {
		isLast := i == len(dep.Imports)-1
		childPrefix := "├─"
		if isLast {
			childPrefix = "└─"
		}
		fmt.Printf("%s  %s %s\n", indent, childPrefix, imp)
	}
}

// displayDepsDot displays dependencies in DOT format for graphviz
func displayDepsDot(deps map[string]*ProtoDependency, files []string) {
	fmt.Println()
	fmt.Println("digraph Dependencies {")
	fmt.Println("  rankdir=LR;")
	fmt.Println("  node [shape=box];")
	fmt.Println()

	for _, file := range files {
		dep := deps[file]
		fileName := filepath.Base(file)

		for _, imp := range dep.Imports {
			impName := filepath.Base(imp)
			fmt.Printf("  \"%s\" -> \"%s\";\n", fileName, impName)
		}
	}

	fmt.Println("}")
	fmt.Println()
	fmt.Println("💡 Save to file and visualize with: dot -Tpng deps.dot -o deps.png")
}
