package cli

import (
	"fmt"
	"os/exec"

	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/massonsky/buffalo/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	validatePaths  []string
	validateStrict bool

	validateCmd = &cobra.Command{
		Use:   "validate",
		Short: "Validate proto files for syntax errors",
		Long: `Validate proto files using protoc to check for syntax and semantic errors.

This command runs protoc in validation mode to ensure all proto files
are syntactically correct and can be compiled. It checks for:
- Syntax errors
- Type mismatches
- Missing imports
- Duplicate definitions
- Invalid field numbers

Examples:
  # Validate all proto files
  buffalo validate

  # Validate specific paths
  buffalo validate --proto-path ./protos

  # Strict mode (include warnings)
  buffalo validate --strict`,
		RunE: runValidate,
	}
)

func init() {
	rootCmd.AddCommand(validateCmd)

	validateCmd.Flags().StringSliceVarP(&validatePaths, "proto-path", "p", []string{"."}, "paths to validate")
	validateCmd.Flags().BoolVar(&validateStrict, "strict", false, "treat warnings as errors")
}

func runValidate(cmd *cobra.Command, args []string) error {
	log := GetLogger()

	log.Info("✓ Validating proto files...")

	// Check if protoc is available
	protocPath, err := utils.FindExecutable("protoc")
	if err != nil {
		log.Error("❌ protoc not found in PATH")
		log.Info("💡 Please install Protocol Buffers compiler:")
		log.Info("   • Linux: apt-get install protobuf-compiler")
		log.Info("   • macOS: brew install protobuf")
		log.Info("   • Windows: choco install protoc")
		return fmt.Errorf("protoc not found")
	}

	log.Debug("Using protoc", logger.String("path", protocPath))

	// Load configuration
	cfg, err := loadConfig(log)
	if err == nil && len(validatePaths) == 1 && validatePaths[0] == "." {
		validatePaths = cfg.Proto.Paths
	}

	// Find all proto files
	var allProtoFiles []string
	for _, path := range validatePaths {
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

	// Validate each file with protoc
	errorCount := 0
	warningCount := 0
	validFiles := 0

	for _, file := range allProtoFiles {
		relPath := getRelativePath(file)

		// Run protoc --descriptor_set_out=/dev/null to validate without generating output
		args := []string{
			"--descriptor_set_out=/dev/null",
			"--proto_path=.",
		}

		// Add import paths if available
		if cfg != nil {
			for _, importPath := range cfg.Proto.ImportPaths {
				args = append(args, fmt.Sprintf("--proto_path=%s", importPath))
			}
		}

		args = append(args, file)

		cmd := exec.Command("protoc", args...)
		output, err := cmd.CombinedOutput()

		if err != nil {
			log.Error(fmt.Sprintf("❌ %s", relPath))
			if len(output) > 0 {
				log.Error(fmt.Sprintf("   %s", string(output)))
			}
			errorCount++
		} else if len(output) > 0 && validateStrict {
			log.Warn(fmt.Sprintf("⚠️  %s", relPath))
			log.Warn(fmt.Sprintf("   %s", string(output)))
			warningCount++
		} else {
			if verbose {
				log.Info(fmt.Sprintf("✅ %s", relPath))
			}
			validFiles++
		}
	}

	// Summary
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════╗")
	fmt.Println("║  Validation Summary                                     ║")
	fmt.Println("╚════════════════════════════════════════════════════════╝")
	fmt.Printf("   Files checked: %d\n", len(allProtoFiles))
	fmt.Printf("   Valid: %d\n", validFiles)
	fmt.Printf("   Errors: %d\n", errorCount)
	if validateStrict {
		fmt.Printf("   Warnings: %d\n", warningCount)
	}
	fmt.Println()

	if errorCount > 0 {
		return fmt.Errorf("validation failed with %d error(s)", errorCount)
	}

	if validateStrict && warningCount > 0 {
		return fmt.Errorf("validation failed with %d warning(s) in strict mode", warningCount)
	}

	log.Info("✅ All files are valid!")
	return nil
}
