package cli

import (
	"fmt"
	"path/filepath"

	"github.com/massonsky/buffalo/internal/embedded"
	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/massonsky/buffalo/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	initForce bool

	initCmd = &cobra.Command{
		Use:   "init",
		Short: "Initialize a new Buffalo project",
		Long: `Initialize a new Buffalo project with default configuration.

This will create a buffalo.yaml configuration file in the current directory
with sensible defaults.`,
		RunE: runInit,
	}
)

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "overwrite existing config file")
}

func runInit(cmd *cobra.Command, args []string) error {
	log := GetLogger()

	configFile := "buffalo.yaml"
	if utils.FileExists(configFile) && !initForce {
		return fmt.Errorf("config file %s already exists (use --force to overwrite)", configFile)
	}

	log.Info("🚀 Initializing Buffalo project")

	// Create default config
	defaultConfig := `# Buffalo Configuration File
# See https://github.com/massonsky/buffalo for full documentation

# Project settings
project:
  name: my-proto-project
  version: 0.1.0

# Proto files configuration
proto:
  # Paths to search for .proto files (can use glob patterns)
  paths:
    - ./protos
  # Files to exclude (glob patterns)
  exclude:
    - "**/*_test.proto"
  # Import paths for proto dependencies
  import_paths:
    - ./third_party

# Output configuration
output:
  # Base directory for generated code
  base_dir: ./generated
  # Per-language output directories (relative to base_dir)
  directories:
    python: python
    go: go
    rust: rust
    cpp: cpp

# Language-specific settings
languages:
  # Python settings
  python:
    enabled: false
    # Package name for generated Python code
    package: proto_gen
    # Use grpcio-tools or betterproto
    generator: grpcio-tools
    
  # Go settings
  go:
    enabled: false
    # Go module path
    module: github.com/yourorg/yourproject
    # Use protoc-gen-go or other generator
    generator: protoc-gen-go
    
  # Rust settings
  rust:
    enabled: false
    # Use prost or other generator
    generator: prost
    
  # C++ settings
  cpp:
    enabled: false
    # C++ namespace
    namespace: myproject

# Build options
build:
  # Number of parallel workers (0 = auto-detect)
  workers: 0
  # Enable incremental builds
  incremental: true
  # Enable caching
  cache:
    enabled: true
    directory: .buffalo-cache

# Logging
logging:
  # Log level: debug, info, warn, error
  level: info
  # Log format: json, text, colored
  format: colored
  # Log output: stdout, stderr, file
  output: stdout
  # Log file path (if output is file)
  file: buffalo.log
`

	if err := utils.WriteFile(configFile, []byte(defaultConfig)); err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	log.Info("✅ Created config file", logger.String("file", configFile))

	// Create default directory structure
	dirs := []string{"./protos", "./generated"}
	for _, dir := range dirs {
		if err := utils.EnsureDir(dir); err != nil {
			log.Warn("Failed to create directory", logger.String("dir", dir), logger.Any("error", err))
		} else {
			log.Info("📁 Created directory", logger.String("dir", dir))
		}
	}

	// Create example proto file
	exampleProto := filepath.Join("protos", "example.proto")
	if !utils.FileExists(exampleProto) {
		exampleContent := `syntax = "proto3";

package example;

// Example service
service ExampleService {
  // Example RPC method
  rpc GetExample(ExampleRequest) returns (ExampleResponse);
}

// Example request message
message ExampleRequest {
  string id = 1;
}

// Example response message
message ExampleResponse {
  string id = 1;
  string name = 2;
  int32 value = 3;
}
`
		if err := utils.WriteFile(exampleProto, []byte(exampleContent)); err != nil {
			log.Warn("Failed to create example proto", logger.Any("error", err))
		} else {
			log.Info("📝 Created example proto file", logger.String("file", exampleProto))
		}
	}

	// Extract built-in validate.proto so users can use buffalo.validate annotations
	protoPath, err := embedded.ExtractValidateProto(".buffalo")
	if err != nil {
		log.Warn("Failed to extract validate.proto", logger.Any("error", err))
	} else {
		log.Info("📦 Extracted validate.proto",
			logger.String("path", protoPath+"/buffalo/validate/validate.proto"))
	}

	fmt.Println()
	log.Info("🎉 Buffalo project initialized successfully!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Edit buffalo.yaml to configure your project")
	fmt.Println("  2. Add your .proto files to ./protos directory")
	fmt.Println("  3. Run 'buffalo build' to generate code")
	fmt.Println()

	return nil
}
