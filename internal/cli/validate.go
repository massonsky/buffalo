package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/massonsky/buffalo/internal/embedded"
	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/massonsky/buffalo/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	validatePaths     []string
	validateStrict    bool
	validateWorkspace string

	validateCmd = &cobra.Command{
		Use:   "validate",
		Short: "Validate proto files and manage validation rules",
		Long: `Validate proto files and manage Buffalo's built-in validation system.

Without a subcommand, runs protoc in validation mode to check for
syntax and semantic errors.

Subcommands allow extracting the embedded validate.proto file and
managing validation rules.

Examples:
  # Validate all proto files
  buffalo validate

  # Extract validate.proto into your project
  buffalo validate init

  # List all embedded proto files
  buffalo validate list

  # Show available validation rules
  buffalo validate rules

  # Validate specific paths
  buffalo validate --proto-path ./protos

  # Strict mode (include warnings)
  buffalo validate --strict`,
		RunE: runValidate,
	}

	validateInitCmd = &cobra.Command{
		Use:   "init",
		Short: "Extract buffalo/validate/validate.proto into your project",
		Long: `Extract the embedded validate.proto file into your project workspace.

This creates:
  .buffalo/proto/buffalo/validate/validate.proto

The path .buffalo/proto is automatically added to proto import paths
during 'buffalo build', so you can immediately use:

  import "buffalo/validate/validate.proto";

  message User {
    string email = 1 [(buffalo.validate.rules).string = {email: true}];
  }

This command is idempotent — running it again overwrites the file to
match the current Buffalo version.`,
		RunE: runValidateInit,
	}

	validateListCmd = &cobra.Command{
		Use:   "list-protos",
		Short: "List all embedded validation proto files",
		RunE:  runValidateList,
	}

	validateRulesCmd = &cobra.Command{
		Use:   "rules",
		Short: "Show all supported validation rules",
		Run:   runValidateRules,
	}
)

func init() {
	rootCmd.AddCommand(validateCmd)

	validateCmd.Flags().StringSliceVarP(&validatePaths, "proto-path", "p", []string{"."}, "paths to validate")
	validateCmd.Flags().BoolVar(&validateStrict, "strict", false, "treat warnings as errors")

	// Subcommands
	validateCmd.AddCommand(validateInitCmd)
	validateCmd.AddCommand(validateListCmd)
	validateCmd.AddCommand(validateRulesCmd)

	validateInitCmd.Flags().StringVar(&validateWorkspace, "workspace", ".buffalo",
		"Buffalo workspace directory where proto files are extracted")
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

		// Run protoc --descriptor_set_out to validate without generating output
		devNull := "/dev/null"
		if runtime.GOOS == "windows" {
			devNull = "NUL"
		}
		args := []string{
			fmt.Sprintf("--descriptor_set_out=%s", devNull),
			"--proto_path=.",
		}

		// Auto-add .buffalo/proto if it exists (extracted buffalo imports)
		buffaloProtoDir := filepath.Join(".buffalo", "proto")
		if _, statErr := os.Stat(buffaloProtoDir); statErr == nil {
			args = append(args, fmt.Sprintf("--proto_path=%s", buffaloProtoDir))
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

// ── validate init ────────────────────────────────────────────────

func runValidateInit(cmd *cobra.Command, args []string) error {
	log := GetLogger()

	log.Info("📦 Extracting embedded validate proto files...",
		logger.String("workspace", validateWorkspace))

	protoPath, err := embedded.ExtractValidateProto(validateWorkspace)
	if err != nil {
		return fmt.Errorf("failed to extract proto files: %w", err)
	}

	log.Info("✅ Proto files extracted successfully",
		logger.String("proto_path", protoPath))

	fmt.Println()
	fmt.Println("Extracted:")
	fmt.Printf("  %s/buffalo/validate/validate.proto\n\n", protoPath)
	fmt.Println("Usage in your .proto files:")
	fmt.Println()
	fmt.Println(`  import "buffalo/validate/validate.proto";`)
	fmt.Println()
	fmt.Println("  message User {")
	fmt.Println(`    string email = 1 [(buffalo.validate.rules).string = {email: true}];`)
	fmt.Println(`    int32  age   = 2 [(buffalo.validate.rules).int32  = {gt: 0, lte: 150}];`)
	fmt.Println("  }")
	fmt.Println()
	fmt.Println("The import path is automatically resolved during 'buffalo build'.")

	return nil
}

// ── validate list-protos ─────────────────────────────────────────

func runValidateList(cmd *cobra.Command, args []string) error {
	files, err := embedded.ListEmbeddedProtos()
	if err != nil {
		return fmt.Errorf("failed to list embedded protos: %w", err)
	}

	fmt.Println("Embedded validation proto files:")
	for _, f := range files {
		fmt.Printf("  %s\n", f)
	}
	return nil
}

// ── validate rules ───────────────────────────────────────────────

func runValidateRules(cmd *cobra.Command, args []string) {
	fmt.Println("Supported buffalo.validate rules:")
	fmt.Println()
	fmt.Println("  ── Numeric (double, float, int32, int64, uint32, uint64) ──")
	fmt.Println("    gt, gte, lt, lte, const, in, not_in")
	fmt.Println()
	fmt.Println("  ── String ──")
	fmt.Println("    min_len, max_len, pattern, prefix, suffix, contains")
	fmt.Println("    email, uri, uuid, ip, ipv4, ipv6, hostname")
	fmt.Println("    not_empty, in, not_in")
	fmt.Println()
	fmt.Println("  ── Bool ──")
	fmt.Println("    const")
	fmt.Println()
	fmt.Println("  ── Bytes ──")
	fmt.Println("    min_len, max_len, pattern")
	fmt.Println()
	fmt.Println("  ── Enum ──")
	fmt.Println("    defined_only, in, not_in")
	fmt.Println()
	fmt.Println("  ── Repeated ──")
	fmt.Println("    min_items, max_items, unique")
	fmt.Println()
	fmt.Println("  ── Map ──")
	fmt.Println("    min_pairs, max_pairs")
	fmt.Println()
	fmt.Println("  ── Timestamp ──")
	fmt.Println("    gt_now, lt_now, within_seconds")
	fmt.Println()
	fmt.Println("  ── Field-level ──")
	fmt.Println("    required")
	fmt.Println()
	fmt.Println("  Example:")
	fmt.Println(`    double lat = 1 [(buffalo.validate.rules).double = {gte: -90, lte: 90}];`)
	fmt.Println(`    string email = 2 [(buffalo.validate.rules).string = {email: true, min_len: 5}];`)
}
