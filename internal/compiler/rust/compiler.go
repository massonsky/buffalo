package rust

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/massonsky/buffalo/internal/compiler"
	"github.com/massonsky/buffalo/pkg/errors"
	"github.com/massonsky/buffalo/pkg/logger"
)

// Compiler implements Rust protobuf compiler
type Compiler struct {
	log     *logger.Logger
	options Options
}

// Options contains Rust compiler options
type Options struct {
	ProtocPath     string // Path to protoc executable
	Generator      string // Generator to use: "rust-protobuf" or "prost"
	GenerateGrpc   bool   // Generate gRPC code (using tonic with prost)
	CargoIntegrate bool   // Generate Cargo.toml integration
}

// DefaultOptions returns default Rust compiler options
func DefaultOptions() Options {
	return Options{
		ProtocPath:     "protoc",
		Generator:      "prost", // prost is more modern
		GenerateGrpc:   true,
		CargoIntegrate: false,
	}
}

// New creates a new Rust compiler
func New(log *logger.Logger, options *Options) *Compiler {
	if options == nil {
		opts := DefaultOptions()
		options = &opts
	}

	return &Compiler{
		log:     log,
		options: *options,
	}
}

// Name returns the compiler name
func (c *Compiler) Name() string {
	return "rust"
}

// Validate checks if the compiler is properly configured
func (c *Compiler) Validate() error {
	// Check protoc
	if err := c.checkTool(c.options.ProtocPath, "--version"); err != nil {
		return errors.Wrap(err, errors.ErrConfig, "protoc not found")
	}

	// Check generator-specific tools
	switch c.options.Generator {
	case "prost":
		// prost is typically used as a Cargo dependency, not a CLI tool
		// We'll generate code via protoc with prost plugin
		return nil
	case "rust-protobuf":
		if err := c.checkTool("protoc-gen-rust", "--version"); err != nil {
			return errors.Wrap(err, errors.ErrConfig, "protoc-gen-rust not found, install: cargo install protobuf-codegen")
		}
	default:
		return errors.New(errors.ErrConfig, "unknown Rust generator: %s (use 'prost' or 'rust-protobuf')", c.options.Generator)
	}

	return nil
}

// checkTool checks if a tool is available
func (c *Compiler) checkTool(toolPath, versionArg string) error {
	cmd := exec.Command(toolPath, versionArg)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tool not found: %s", toolPath)
	}
	return nil
}

// Compile compiles proto files to Rust
func (c *Compiler) Compile(ctx context.Context, files []compiler.ProtoFile, opts compiler.CompileOptions) (*compiler.CompileResult, error) {
	result := &compiler.CompileResult{
		GeneratedFiles: []string{},
		Warnings:       []string{},
		Success:        false,
	}

	// Add warning about Rust compilation
	result.Warnings = append(result.Warnings,
		"Rust compilation uses cargo build integration. Manual setup may be required.")

	for _, file := range files {
		c.log.Debug("Compiling proto file to Rust",
			logger.String("file", file.Path),
			logger.String("output", opts.OutputDir))

		generatedFiles, err := c.compileFile(ctx, file, opts)
		if err != nil {
			return nil, errors.Wrap(err, errors.ErrCompilation, "failed to compile proto file: %s", file.Path)
		}

		result.GeneratedFiles = append(result.GeneratedFiles, generatedFiles...)
	}

	result.Success = true
	return result, nil
}

// compileFile compiles a single proto file
func (c *Compiler) compileFile(ctx context.Context, file compiler.ProtoFile, opts compiler.CompileOptions) ([]string, error) {
	var generatedFiles []string

	switch c.options.Generator {
	case "rust-protobuf":
		// Use protoc with rust plugin
		args := c.buildProtocArgs(file, opts)

		c.log.Debug("Running protoc for Rust",
			logger.String("command", c.options.ProtocPath),
			logger.Any("args", args))

		cmd := exec.CommandContext(ctx, c.options.ProtocPath, args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("protoc failed: %v\nOutput: %s", err, string(output))
		}

		// Add generated .rs file
		outputPath := c.GetOutputPath(file, opts)
		generatedFiles = append(generatedFiles, outputPath)

	case "prost":
		// For prost, we need to generate a build.rs file
		// This is typically done manually in a Rust project
		result := fmt.Sprintf("Prost requires Cargo integration. Add to build.rs:\n"+
			"    prost_build::compile_protos(&[\"%s\"], &[\".\"])?;", file.Path)
		return nil, fmt.Errorf("prost generator requires manual Cargo setup:\n%s", result)

	default:
		return nil, errors.New(errors.ErrConfig, "unknown Rust generator: %s", c.options.Generator)
	}

	return generatedFiles, nil
}

// buildProtocArgs builds the protoc command arguments for rust-protobuf
func (c *Compiler) buildProtocArgs(file compiler.ProtoFile, opts compiler.CompileOptions) []string {
	args := []string{}

	// Merge and deduplicate import paths
	importPaths := compiler.MergeImportPaths(opts, file)
	if len(importPaths) == 0 {
		args = append(args, "--proto_path=.")
	}

	// Add import paths
	for _, importPath := range importPaths {
		args = append(args, "--proto_path="+importPath)
	}

	// Add output directory
	args = append(args, "--rust_out="+opts.OutputDir)

	// Add the proto file
	args = append(args, file.Path)

	return args
}

// GetOutputPath returns the output path for a proto file
func (c *Compiler) GetOutputPath(file compiler.ProtoFile, opts compiler.CompileOptions) string {
	baseName := strings.TrimSuffix(filepath.Base(file.Path), ".proto")

	// Get the directory structure
	relPath := filepath.Dir(file.Path)
	if relPath == "." {
		relPath = ""
	}

	outputFile := baseName + ".rs"
	if relPath != "" {
		return filepath.Join(opts.OutputDir, relPath, outputFile)
	}
	return filepath.Join(opts.OutputDir, outputFile)
}

// RequiredTools returns the list of required external tools
func (c *Compiler) RequiredTools() []string {
	tools := []string{"protoc"}
	if c.options.Generator == "rust-protobuf" {
		tools = append(tools, "protoc-gen-rust")
	}
	return tools
}
