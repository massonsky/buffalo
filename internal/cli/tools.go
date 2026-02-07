package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/massonsky/buffalo/internal/tools"
	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/spf13/cobra"
)

var (
	toolsLanguages   []string
	toolsForce       bool
	toolsDryRun      bool
	toolsInteractive bool
	toolsIncludeAll  bool
	toolsVerbose     bool

	toolsCmd = &cobra.Command{
		Use:   "tools",
		Short: "Manage language tools and dependencies",
		Long: `Manage language-specific tools required for protobuf/gRPC code generation.

Buffalo requires various tools depending on the target languages:
  - Go: protoc-gen-go, protoc-gen-go-grpc, protoc-gen-grpc-gateway
  - Python: grpcio-tools, grpcio, protobuf, mypy-protobuf
  - Rust: rustc, cargo, protobuf-codegen
  - C++: g++, cmake, libprotobuf-dev, grpc-dev

Use 'buffalo tools install' to automatically install required tools.
Use 'buffalo tools list' to see what tools are available.
Use 'buffalo tools check' to verify your environment.`,
	}

	toolsInstallCmd = &cobra.Command{
		Use:   "install [languages...]",
		Short: "Install language tools for protobuf/gRPC compilation",
		Long: `Install language-specific tools required for protobuf/gRPC code generation.

If no languages are specified, tools for languages enabled in buffalo.yaml will be installed.
You can also specify languages explicitly: go, python, rust, cpp.

Examples:
  # Install tools for all enabled languages (from buffalo.yaml)
  buffalo tools install

  # Install tools for specific languages
  buffalo tools install go python

  # Install all tools including optional ones
  buffalo tools install --all

  # Force reinstall
  buffalo tools install --force

  # Preview what would be installed
  buffalo tools install --dry-run

  # Interactive mode - confirm each tool
  buffalo tools install --interactive`,
		RunE: runToolsInstall,
	}

	toolsListCmd = &cobra.Command{
		Use:   "list [languages...]",
		Short: "List available language tools",
		Long: `List all available tools for protobuf/gRPC development.

Shows tools organized by language with installation status and commands.

Examples:
  # List all tools
  buffalo tools list

  # List tools for specific language
  buffalo tools list go

  # Include optional tools
  buffalo tools list --all`,
		RunE: runToolsList,
	}

	toolsCheckCmd = &cobra.Command{
		Use:   "check [languages...]",
		Short: "Check if required tools are installed",
		Long: `Check installation status of language tools.

Verifies that all required tools are installed and shows their versions.

Examples:
  # Check tools for all enabled languages
  buffalo tools check

  # Check tools for specific languages
  buffalo tools check go python`,
		RunE: runToolsCheck,
	}
)

func init() {
	rootCmd.AddCommand(toolsCmd)
	toolsCmd.AddCommand(toolsInstallCmd)
	toolsCmd.AddCommand(toolsListCmd)
	toolsCmd.AddCommand(toolsCheckCmd)

	// Install flags
	toolsInstallCmd.Flags().StringSliceVarP(&toolsLanguages, "lang", "l", nil, "Languages to install tools for (go, python, rust, cpp)")
	toolsInstallCmd.Flags().BoolVar(&toolsForce, "force", false, "Force reinstall even if already installed")
	toolsInstallCmd.Flags().BoolVar(&toolsDryRun, "dry-run", false, "Show what would be installed without installing")
	toolsInstallCmd.Flags().BoolVarP(&toolsInteractive, "interactive", "i", false, "Confirm each tool before installing")
	toolsInstallCmd.Flags().BoolVar(&toolsIncludeAll, "all", false, "Install all tools including optional ones")
	toolsInstallCmd.Flags().BoolVarP(&toolsVerbose, "verbose", "v", false, "Verbose output")

	// List flags
	toolsListCmd.Flags().BoolVar(&toolsIncludeAll, "all", false, "Include optional tools")

	// Check flags
	toolsCheckCmd.Flags().StringSliceVarP(&toolsLanguages, "lang", "l", nil, "Languages to check (go, python, rust, cpp)")
}

func runToolsInstall(cmd *cobra.Command, args []string) error {
	log := GetLogger()

	// Determine languages to install
	languages := args
	if len(languages) == 0 && len(toolsLanguages) > 0 {
		languages = toolsLanguages
	}

	// If still no languages, try to get from config
	if len(languages) == 0 {
		cfg, err := loadConfig(log)
		if err == nil {
			languages = cfg.GetEnabledLanguages()
		}
	}

	// Default to all languages if nothing specified
	if len(languages) == 0 {
		languages = []string{"go", "python", "rust", "cpp"}
		log.Info("No languages specified, installing tools for all languages")
	}

	log.Info("🔧 Installing language tools",
		logger.Any("languages", languages))

	if toolsDryRun {
		log.Info("DRY RUN MODE - No changes will be made")
	}

	installer := tools.NewInstaller(log)
	opts := tools.InstallOptions{
		Languages:   languages,
		Force:       toolsForce,
		DryRun:      toolsDryRun,
		Interactive: toolsInteractive,
		IncludeAll:  toolsIncludeAll,
		Verbose:     toolsVerbose,
	}

	results := installer.InstallAll(languages, opts)

	// Print results
	return printInstallResults(results, log)
}

func runToolsList(cmd *cobra.Command, args []string) error {
	log := GetLogger()

	languages := args
	if len(languages) == 0 {
		languages = []string{"go", "python", "rust", "cpp"}
	}

	installer := tools.NewInstaller(log)
	allTools := installer.ListTools(languages, toolsIncludeAll)

	// Group by language
	byLanguage := make(map[string][]tools.Tool)
	for _, tool := range allTools {
		lang := tool.Language
		if lang == "all" {
			lang = "core"
		}
		byLanguage[lang] = append(byLanguage[lang], tool)
	}

	fmt.Println()
	fmt.Println("📦 Available Tools for Protobuf/gRPC Development")
	fmt.Println("═══════════════════════════════════════════════════")

	// Sort languages
	sortedLangs := make([]string, 0, len(byLanguage))
	for lang := range byLanguage {
		sortedLangs = append(sortedLangs, lang)
	}
	sort.Strings(sortedLangs)

	// Print core first
	if coreTools, ok := byLanguage["core"]; ok {
		printToolsGroup("Core (Required)", coreTools, installer)
		delete(byLanguage, "core")
	}

	// Print other languages
	for _, lang := range sortedLangs {
		if lang == "core" {
			continue
		}
		toolsList := byLanguage[lang]
		printToolsGroup(strings.Title(lang), toolsList, installer)
	}

	fmt.Println()
	fmt.Println("Use 'buffalo tools install <language>' to install tools")
	fmt.Println("Use 'buffalo tools check' to verify installation status")
	fmt.Println()

	return nil
}

func printToolsGroup(title string, toolsList []tools.Tool, installer *tools.Installer) {
	fmt.Printf("\n%s:\n", title)
	fmt.Println(strings.Repeat("─", 60))

	for _, tool := range toolsList {
		// Check if installed
		installed, version, _ := installer.Check(tool)

		status := "❌"
		versionStr := ""
		if installed {
			status = "✅"
			versionStr = fmt.Sprintf(" (%s)", version)
		}

		criticalMark := ""
		if tool.Critical {
			criticalMark = " *"
		}

		fmt.Printf("  %s %s%s%s\n", status, tool.Name, criticalMark, versionStr)
		fmt.Printf("     %s\n", tool.Description)

		if !installed {
			platform := tools.GetPlatform()
			if cmd, ok := tool.InstallMethods[platform]; ok {
				fmt.Printf("     Install: %s\n", cmd)
			}
		}
	}
}

func runToolsCheck(cmd *cobra.Command, args []string) error {
	log := GetLogger()

	languages := args
	if len(languages) == 0 && len(toolsLanguages) > 0 {
		languages = toolsLanguages
	}

	// If still no languages, try to get from config
	if len(languages) == 0 {
		cfg, err := loadConfig(log)
		if err == nil {
			languages = cfg.GetEnabledLanguages()
		}
	}

	if len(languages) == 0 {
		languages = []string{"go", "python", "rust", "cpp"}
	}

	log.Info("🔍 Checking language tools",
		logger.Any("languages", languages))

	installer := tools.NewInstaller(log)
	results := installer.CheckAll(languages)

	fmt.Println()
	fmt.Println("📋 Tool Installation Status")
	fmt.Println("═══════════════════════════════════════════════════")

	allOK := true
	criticalMissing := false

	// Print core first
	if coreResults, ok := results["core"]; ok {
		fmt.Println("\nCore Tools:")
		fmt.Println(strings.Repeat("─", 50))
		for _, result := range coreResults {
			printCheckResult(result)
			if !result.Success {
				allOK = false
				if result.Tool.Critical {
					criticalMissing = true
				}
			}
		}
	}

	// Print language-specific
	for lang, langResults := range results {
		if lang == "core" {
			continue
		}
		fmt.Printf("\n%s:\n", strings.Title(lang))
		fmt.Println(strings.Repeat("─", 50))
		for _, result := range langResults {
			printCheckResult(result)
			if !result.Success {
				allOK = false
				if result.Tool.Critical {
					criticalMissing = true
				}
			}
		}
	}

	fmt.Println()

	if allOK {
		fmt.Println("✅ All tools are installed and ready!")
	} else if criticalMissing {
		fmt.Println("❌ Some critical tools are missing!")
		fmt.Println("   Run 'buffalo tools install' to install them.")
		os.Exit(1)
	} else {
		fmt.Println("⚠️  Some optional tools are missing.")
		fmt.Println("   Run 'buffalo tools install --all' to install all tools.")
	}

	fmt.Println()
	return nil
}

func printCheckResult(result tools.InstallResult) {
	status := "❌"
	if result.Success {
		status = "✅"
	}

	criticalMark := ""
	if result.Tool.Critical {
		criticalMark = " *"
	}

	versionStr := ""
	if result.Version != "" && result.Success {
		versionStr = fmt.Sprintf(" - %s", result.Version)
	}

	fmt.Printf("  %s %s%s%s\n", status, result.Tool.Name, criticalMark, versionStr)

	if !result.Success && result.Tool.InstallMethods != nil {
		platform := tools.GetPlatform()
		if cmd, ok := result.Tool.InstallMethods[platform]; ok {
			fmt.Printf("     Install: %s\n", cmd)
		}
	}
}

func printInstallResults(results map[string][]tools.InstallResult, log *logger.Logger) error {
	fmt.Println()
	fmt.Println("📋 Installation Results")
	fmt.Println("═══════════════════════════════════════════════════")

	successCount := 0
	failCount := 0
	skippedCount := 0
	alreadyCount := 0

	for category, categoryResults := range results {
		fmt.Printf("\n%s:\n", strings.Title(category))
		fmt.Println(strings.Repeat("─", 50))

		for _, result := range categoryResults {
			status := "❌ FAILED"
			if result.Success {
				if result.AlreadyOK {
					status = "✅ Already installed"
					alreadyCount++
				} else {
					status = "✅ Installed"
					successCount++
				}
			} else if result.Skipped {
				status = "⏭️  Skipped"
				skippedCount++
			} else {
				failCount++
			}

			fmt.Printf("  %s: %s\n", result.Tool.Name, status)
			if result.Version != "" && result.Success {
				fmt.Printf("     Version: %s\n", result.Version)
			}
			if result.Error != nil {
				fmt.Printf("     Error: %s\n", result.Error)
			}
		}
	}

	fmt.Println()
	fmt.Println("Summary:")
	fmt.Printf("  ✅ Installed: %d\n", successCount)
	fmt.Printf("  ✅ Already OK: %d\n", alreadyCount)
	fmt.Printf("  ⏭️  Skipped: %d\n", skippedCount)
	fmt.Printf("  ❌ Failed: %d\n", failCount)
	fmt.Println()

	if failCount > 0 {
		return fmt.Errorf("some tools failed to install")
	}

	return nil
}
