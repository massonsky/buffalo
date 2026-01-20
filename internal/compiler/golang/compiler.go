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

	// Generate go.mod file if GoModule is specified
	if c.options.GoModule != "" {
		if err := c.generateGoMod(opts.OutputDir); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("failed to generate go.mod: %v", err))
		}
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

// generateGoMod creates a go.mod file in the output directory and runs go mod tidy
// It analyzes generated .go files to determine required dependencies
func (c *Compiler) generateGoMod(outputDir string) error {
	goModPath := filepath.Join(outputDir, "go.mod")

	// Check if go.mod already exists
	if _, err := os.Stat(goModPath); err == nil {
		c.log.Debug("go.mod already exists, skipping generation",
			logger.String("path", goModPath))
		return nil
	}

	// Analyze generated Go files to detect required imports
	requiredDeps := c.analyzeGoImports(outputDir)

	// Build require block
	var requireBlock strings.Builder
	for pkg, version := range requiredDeps {
		requireBlock.WriteString(fmt.Sprintf("\t%s %s\n", pkg, version))
	}

	// Create go.mod content based on analysis
	goModContent := fmt.Sprintf(`module %s

go 1.23

require (
%s)
`, c.options.GoModule, requireBlock.String())

	// Write go.mod file
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		return fmt.Errorf("failed to write go.mod: %v", err)
	}

	c.log.Info("Generated go.mod",
		logger.String("path", goModPath),
		logger.String("module", c.options.GoModule),
		logger.Int("dependencies", len(requiredDeps)))

	// Run go mod tidy to resolve transitive dependencies
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = outputDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		c.log.Warn("go mod tidy failed, go.mod may have incomplete dependencies",
			logger.String("error", err.Error()),
			logger.String("output", string(output)))
		// Don't return error - go.mod is still usable, just might need manual tidy
	} else {
		c.log.Info("Ran go mod tidy successfully",
			logger.String("path", outputDir))
	}

	return nil
}

// analyzeGoImports scans generated .go files and returns a map of required external packages
func (c *Compiler) analyzeGoImports(outputDir string) map[string]string {
	// Known protobuf/grpc package versions (latest stable)
	knownPackages := map[string]string{
		"google.golang.org/protobuf":                      "v1.36.2",
		"google.golang.org/grpc":                          "v1.69.4",
		"google.golang.org/genproto/googleapis/api":       "v0.0.0-20241015192408-796eee8c2d53",
		"google.golang.org/genproto/googleapis/rpc":       "v0.0.0-20241015192408-796eee8c2d53",
		"github.com/grpc-ecosystem/grpc-gateway/v2":       "v2.24.0",
		"github.com/envoyproxy/protoc-gen-validate":       "v1.1.0",
	}

	requiredDeps := make(map[string]string)

	// Walk through all .go files
	filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		fileContent := string(content)

		// Check for common protobuf imports
		for pkg, version := range knownPackages {
			// Check if this package is imported (simple string match)
			importPath := `"` + pkg
			if strings.Contains(fileContent, importPath) {
				requiredDeps[pkg] = version
				c.log.Debug("Found import",
					logger.String("file", path),
					logger.String("package", pkg))
			}
		}

		// Always include protobuf as base dependency for generated code
		if strings.Contains(fileContent, "google.golang.org/protobuf") ||
			strings.Contains(fileContent, "proto.Message") ||
			strings.Contains(fileContent, "protoreflect") {
			requiredDeps["google.golang.org/protobuf"] = knownPackages["google.golang.org/protobuf"]
		}

		// Check for grpc imports
		if strings.Contains(fileContent, "google.golang.org/grpc") {
			requiredDeps["google.golang.org/grpc"] = knownPackages["google.golang.org/grpc"]
		}

		return nil
	})

	// Ensure at least protobuf is present (all generated code needs it)
	if len(requiredDeps) == 0 {
		requiredDeps["google.golang.org/protobuf"] = knownPackages["google.golang.org/protobuf"]
	}

	return requiredDeps
}

// RequiredTools returns the list of required external tools
func (c *Compiler) RequiredTools() []string {
	tools := []string{"protoc", "protoc-gen-go"}
	if c.options.GenerateGrpc {
		tools = append(tools, "protoc-gen-go-grpc")
	}
	return tools
}
