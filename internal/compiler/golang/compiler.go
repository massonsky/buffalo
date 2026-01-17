package golang

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/massonsky/buffalo/internal/compiler"
	"github.com/massonsky/buffalo/pkg/errors"
	"github.com/massonsky/buffalo/pkg/logger"
)

// Compiler implements Go protobuf/gRPC compiler
type Compiler struct {
	log     *logger.Logger
	options Options
}

// Options contains Go compiler options
type Options struct {
	ProtocPath          string // Path to protoc executable
	ProtocGenGoPath     string // Path to protoc-gen-go plugin
	ProtocGenGoGrpcPath string // Path to protoc-gen-go-grpc plugin
	GoModule            string // Go module path
	GenerateGrpc        bool   // Generate gRPC code
	GenerateGateway     bool   // Generate gRPC-Gateway code
	GenerateOpenAPIV2   bool   // Generate OpenAPI v2 spec
	GenerateValidate    bool   // Generate validation code
	GoPackageSuffix     string // Suffix to add to Go package path
}

// DefaultOptions returns default Go compiler options
func DefaultOptions() Options {
	return Options{
		ProtocPath:          "protoc",
		ProtocGenGoPath:     "protoc-gen-go",
		ProtocGenGoGrpcPath: "protoc-gen-go-grpc",
		GenerateGrpc:        true,
		GenerateGateway:     false,
		GenerateOpenAPIV2:   false,
		GenerateValidate:    false,
		GoPackageSuffix:     "",
	}
}

// New creates a new Go compiler
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
	return "go"
}

// Validate checks if the compiler is properly configured
func (c *Compiler) Validate() error {
	// Check protoc
	if err := c.checkTool(c.options.ProtocPath, "--version"); err != nil {
		return errors.Wrap(err, errors.ErrConfig, "protoc not found")
	}

	// Check protoc-gen-go
	if err := c.checkTool(c.options.ProtocGenGoPath, "--version"); err != nil {
		return errors.Wrap(err, errors.ErrConfig, "protoc-gen-go not found, install: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest")
	}

	// Check protoc-gen-go-grpc if gRPC is enabled
	if c.options.GenerateGrpc {
		if err := c.checkTool(c.options.ProtocGenGoGrpcPath, "--version"); err != nil {
			return errors.Wrap(err, errors.ErrConfig, "protoc-gen-go-grpc not found, install: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest")
		}
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

// Compile compiles proto files to Go
func (c *Compiler) Compile(ctx context.Context, files []compiler.ProtoFile, opts compiler.CompileOptions) (*compiler.CompileResult, error) {
	result := &compiler.CompileResult{
		GeneratedFiles: []string{},
		Warnings:       []string{},
		Success:        false,
	}

	for _, file := range files {
		c.log.Debug("Compiling proto file to Go",
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

	// Use output directory as-is (protoc will create subdirs from file path if needed)
	outputDir := opts.OutputDir

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %v", err)
	}

	// Determine which file path to use for protoc
	protoFilePath := file.Path
	protoDir := "."
	if !opts.PreserveProtoStructure {
		// Use only the base name to avoid creating subdirectories
		protoFilePath = filepath.Base(file.Path)
		// And add the directory as proto_path
		protoDir = filepath.Dir(file.Path)
	}

	// Build protoc command for Go
	args := c.buildProtocArgs(file, opts, protoDir, false)
	args = append(args, protoFilePath)

	c.log.Debug("Running protoc for Go",
		logger.String("command", c.options.ProtocPath),
		logger.Any("args", args))

	cmd := exec.CommandContext(ctx, c.options.ProtocPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("protoc failed: %v\nOutput: %s", err, string(output))
	}

	// Add generated .pb.go file
	outputPath := c.getOutputPathWithDir(file, opts)
	generatedFiles = append(generatedFiles, outputPath)

	// Generate gRPC code if enabled
	if c.options.GenerateGrpc {
		grpcArgs := c.buildProtocArgs(file, opts, protoDir, true)
		grpcArgs = append(grpcArgs, protoFilePath)

		c.log.Debug("Running protoc for Go gRPC",
			logger.String("command", c.options.ProtocPath),
			logger.Any("args", grpcArgs))

		grpcCmd := exec.CommandContext(ctx, c.options.ProtocPath, grpcArgs...)
		grpcOutput, err := grpcCmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("protoc gRPC failed: %v\nOutput: %s", err, string(grpcOutput))
		}

		// Add generated _grpc.pb.go file
		grpcPath := strings.TrimSuffix(outputPath, ".pb.go") + "_grpc.pb.go"
		generatedFiles = append(generatedFiles, grpcPath)
	}

	return generatedFiles, nil
}

// buildProtocArgs builds the protoc command arguments
func (c *Compiler) buildProtocArgs(file compiler.ProtoFile, opts compiler.CompileOptions, protoDir string, grpc bool) []string {
	args := []string{}

	// Add proto_path for the proto file directory if needed
	if protoDir != "." {
		args = append(args, "--proto_path="+protoDir)
	}

	// Merge and deduplicate import paths
	importPaths := compiler.MergeImportPaths(opts, file)
	if len(importPaths) == 0 {
		args = append(args, "--proto_path=.")
	}

	// Add import paths
	for _, importPath := range importPaths {
		args = append(args, "--proto_path="+importPath)
	}

	// Build go_opt with module path
	goOpt := fmt.Sprintf("paths=source_relative")
	if c.options.GoModule != "" {
		// Get relative path from proto file to determine package
		relPath := filepath.Dir(file.Path)
		if relPath != "." {
			relPath = strings.ReplaceAll(relPath, "\\", "/")
		}
		goOpt = fmt.Sprintf("module=%s", c.options.GoModule)
	}

	if grpc {
		// Generate gRPC code
		args = append(args, "--go-grpc_out="+opts.OutputDir)
		args = append(args, "--go-grpc_opt="+goOpt)

		// Add plugin path if specified
		if c.options.ProtocGenGoGrpcPath != "" && c.options.ProtocGenGoGrpcPath != "protoc-gen-go-grpc" {
			args = append(args, "--plugin=protoc-gen-go-grpc="+c.options.ProtocGenGoGrpcPath)
		}
	} else {
		// Generate protobuf code
		args = append(args, "--go_out="+opts.OutputDir)
		args = append(args, "--go_opt="+goOpt)

		// Add plugin path if specified
		if c.options.ProtocGenGoPath != "" && c.options.ProtocGenGoPath != "protoc-gen-go" {
			args = append(args, "--plugin=protoc-gen-go="+c.options.ProtocGenGoPath)
		}
	}

	// Note: file path should be added by caller
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

	outputFile := baseName + ".pb.go"
	if relPath != "" {
		return filepath.Join(opts.OutputDir, relPath, outputFile)
	}
	return filepath.Join(opts.OutputDir, outputFile)
}

// getOutputPathWithDir is a helper that computes the output path with already-adjusted outputDir
func (c *Compiler) getOutputPathWithDir(file compiler.ProtoFile, opts compiler.CompileOptions) string {
	baseName := strings.TrimSuffix(filepath.Base(file.Path), ".proto")
	return filepath.Join(opts.OutputDir, baseName+".pb.go")
}

// RequiredTools returns the list of required external tools
func (c *Compiler) RequiredTools() []string {
	tools := []string{"protoc", "protoc-gen-go"}
	if c.options.GenerateGrpc {
		tools = append(tools, "protoc-gen-go-grpc")
	}
	return tools
}
