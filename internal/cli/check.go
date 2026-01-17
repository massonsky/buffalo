package cli

import (
	"fmt"
	"os"

	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/massonsky/buffalo/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	checkVerbose bool

	checkCmd = &cobra.Command{
		Use:   "check",
		Short: "Check project configuration and dependencies",
		Long: `Check project configuration, proto files, and required dependencies.

This command validates your Buffalo configuration, checks that proto files
exist and are readable, verifies that required tools (protoc, language-specific
generators) are installed, and reports any issues.

Examples:
  # Basic check
  buffalo check

  # Verbose output with details
  buffalo check --verbose`,
		RunE: runCheck,
	}
)

func init() {
	rootCmd.AddCommand(checkCmd)
	checkCmd.Flags().BoolVarP(&checkVerbose, "verbose", "v", false, "verbose output")
}

func runCheck(cmd *cobra.Command, args []string) error {
	log := GetLogger()

	log.Info("🔍 Checking project configuration...")

	issues := 0
	warnings := 0

	// 1. Check configuration file
	log.Info("\n📄 Configuration File")
	cfg, err := loadConfig(log)
	if err != nil {
		log.Error("  ❌ Config file not found or invalid", logger.Any("error", err))
		log.Info("  💡 Tip: Run 'buffalo init' to create a default configuration")
		issues++
	} else {
		log.Info("  ✅ Config file loaded successfully")
		if checkVerbose {
			log.Info(fmt.Sprintf("     Project: %s", cfg.Project.Name))
			log.Info(fmt.Sprintf("     Version: %s", cfg.Project.Version))
		}
	}

	if cfg == nil {
		cfg = getDefaultConfig()
		log.Warn("  ⚠️  Using default configuration")
		warnings++
	}

	// 2. Check proto files
	log.Info("\n📦 Proto Files")
	var allProtoFiles []string
	for _, path := range cfg.Proto.Paths {
		fileInfos, err := utils.FindFiles(path, utils.FindFilesOptions{
			Pattern:   "*.proto",
			Recursive: true,
		})
		if err != nil {
			log.Warn(fmt.Sprintf("  ⚠️  Failed to scan path: %s", path), logger.Any("error", err))
			warnings++
			continue
		}
		for _, fi := range fileInfos {
			allProtoFiles = append(allProtoFiles, fi.Path)
		}
	}

	if len(allProtoFiles) == 0 {
		log.Error("  ❌ No proto files found in specified paths")
		for _, path := range cfg.Proto.Paths {
			log.Info(fmt.Sprintf("     Searched: %s", path))
		}
		log.Info("  💡 Tip: Check your proto.paths configuration")
		issues++
	} else {
		log.Info(fmt.Sprintf("  ✅ Found %d proto file(s)", len(allProtoFiles)))
		if checkVerbose {
			for i, file := range allProtoFiles {
				if i < 10 { // Show first 10 files
					log.Info(fmt.Sprintf("     • %s", file))
				}
			}
			if len(allProtoFiles) > 10 {
				log.Info(fmt.Sprintf("     ... and %d more", len(allProtoFiles)-10))
			}
		}
	}

	// 3. Check output directory
	log.Info("\n📁 Output Directory")
	outputDir := cfg.Output.BaseDir
	if outputDir == "" {
		log.Error("  ❌ Output directory not configured")
		issues++
	} else {
		log.Info(fmt.Sprintf("  📂 Output: %s", outputDir))
		if _, err := os.Stat(outputDir); os.IsNotExist(err) {
			log.Info("  ℹ️  Output directory does not exist yet (will be created)")
		} else {
			log.Info("  ✅ Output directory exists")
		}
	}

	// 4. Check languages
	log.Info("\n🌐 Languages")
	enabledLangs := cfg.GetEnabledLanguages()
	if len(enabledLangs) == 0 {
		log.Warn("  ⚠️  No languages enabled")
		log.Info("  💡 Tip: Enable at least one language in your config or use --lang flag")
		warnings++
	} else {
		log.Info(fmt.Sprintf("  ✅ %d language(s) enabled:", len(enabledLangs)))
		for _, lang := range enabledLangs {
			log.Info(fmt.Sprintf("     • %s", lang))
		}
	}

	// 5. Check protoc
	log.Info("\n🔧 Dependencies")
	if _, err := utils.FindExecutable("protoc"); err != nil {
		log.Error("  ❌ protoc not found in PATH")
		log.Info("  💡 Tip: Install Protocol Buffers compiler")
		log.Info("     • Linux: apt-get install protobuf-compiler")
		log.Info("     • macOS: brew install protobuf")
		log.Info("     • Windows: choco install protoc")
		issues++
	} else {
		log.Info("  ✅ protoc found")
		if checkVerbose {
			if output, err := utils.ExecCommand("protoc", "--version"); err == nil {
				log.Info(fmt.Sprintf("     Version: %s", output))
			}
		}
	}

	// Check language-specific tools
	for _, lang := range enabledLangs {
		switch lang {
		case "python":
			if _, err := utils.FindExecutable("python"); err != nil {
				if _, err := utils.FindExecutable("python3"); err != nil {
					log.Warn(fmt.Sprintf("  ⚠️  Python not found for %s", lang))
					warnings++
				} else {
					log.Info(fmt.Sprintf("  ✅ python3 found for %s", lang))
				}
			} else {
				log.Info(fmt.Sprintf("  ✅ python found for %s", lang))
			}

		case "go":
			if _, err := utils.FindExecutable("protoc-gen-go"); err != nil {
				log.Warn(fmt.Sprintf("  ⚠️  protoc-gen-go not found for %s", lang))
				log.Info("  💡 Tip: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest")
				warnings++
			} else {
				log.Info(fmt.Sprintf("  ✅ protoc-gen-go found for %s", lang))
			}
			if _, err := utils.FindExecutable("protoc-gen-go-grpc"); err != nil {
				log.Warn(fmt.Sprintf("  ⚠️  protoc-gen-go-grpc not found for %s", lang))
				log.Info("  💡 Tip: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest")
				warnings++
			} else {
				log.Info(fmt.Sprintf("  ✅ protoc-gen-go-grpc found for %s", lang))
			}

		case "rust":
			if _, err := utils.FindExecutable("cargo"); err != nil {
				log.Warn(fmt.Sprintf("  ⚠️  cargo not found for %s", lang))
				log.Info("  💡 Tip: Install Rust from https://rustup.rs/")
				warnings++
			} else {
				log.Info(fmt.Sprintf("  ✅ cargo found for %s", lang))
			}

		case "cpp":
			if _, err := utils.FindExecutable("protoc"); err != nil {
				log.Warn(fmt.Sprintf("  ⚠️  protoc with C++ support needed for %s", lang))
				warnings++
			} else {
				log.Info(fmt.Sprintf("  ✅ protoc with C++ support for %s", lang))
			}
		}
	}

	// 6. Check cache configuration
	if cfg.Build.Cache.Enabled {
		log.Info("\n💾 Cache")
		cacheDir := cfg.Build.Cache.Directory
		if cacheDir == "" {
			log.Warn("  ⚠️  Cache enabled but directory not specified")
			warnings++
		} else {
			log.Info(fmt.Sprintf("  ✅ Cache directory: %s", cacheDir))
			if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
				log.Info("  ℹ️  Cache directory does not exist yet (will be created)")
			}
		}
	}

	// Summary
	log.Info("\n📊 Summary")
	if issues == 0 && warnings == 0 {
		log.Info("  ✅ All checks passed! Your project is ready to build.")
		return nil
	}

	if issues > 0 {
		log.Error(fmt.Sprintf("  ❌ Found %d critical issue(s)", issues))
	}
	if warnings > 0 {
		log.Warn(fmt.Sprintf("  ⚠️  Found %d warning(s)", warnings))
	}

	if issues > 0 {
		log.Info("\n  💡 Please fix the critical issues before building")
		return fmt.Errorf("configuration check failed with %d issue(s)", issues)
	}

	log.Info("\n  ℹ️  You can proceed with caution, but some features may not work")
	return nil
}
