package cpp

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

// Compiler implements C++ protobuf/gRPC compiler
type Compiler struct {
	log     *logger.Logger
	options Options
}

// Options contains C++ compiler options
type Options struct {
	ProtocPath        string // Path to protoc executable
	GrpcCppPluginPath string // Path to grpc_cpp_plugin
	GenerateGrpc      bool   // Generate gRPC code
	Namespace         string // C++ namespace
}

// DefaultOptions returns default C++ compiler options
func DefaultOptions() Options {
	return Options{
		ProtocPath:        "protoc",
		GrpcCppPluginPath: "grpc_cpp_plugin",
		GenerateGrpc:      true,
		Namespace:         "",
	}
}

// New creates a new C++ compiler
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
	return "cpp"
}

// Validate checks if the compiler is properly configured
func (c *Compiler) Validate() error {
	// Check protoc
	if err := c.checkTool(c.options.ProtocPath, "--version"); err != nil {
		return errors.Wrap(err, errors.ErrConfig, "protoc not found")
	}

	// Check grpc_cpp_plugin if gRPC is enabled
	if c.options.GenerateGrpc {
		if err := c.checkTool(c.options.GrpcCppPluginPath, "--help"); err != nil {
			return errors.Wrap(err, errors.ErrConfig, "grpc_cpp_plugin not found, install gRPC C++ from https://grpc.io/docs/languages/cpp/quickstart/")
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

// Compile compiles proto files to C++
func (c *Compiler) Compile(ctx context.Context, files []compiler.ProtoFile, opts compiler.CompileOptions) (*compiler.CompileResult, error) {
	result := &compiler.CompileResult{
		GeneratedFiles: []string{},
		Warnings:       []string{},
		Success:        false,
	}

	for _, file := range files {
		c.log.Debug("Compiling proto file to C++",
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

	// Build protoc command for C++
	args := c.buildProtocArgs(file, opts, false)

	c.log.Debug("Running protoc for C++",
		logger.String("command", c.options.ProtocPath),
		logger.Any("args", args))

	cmd := exec.CommandContext(ctx, c.options.ProtocPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("protoc failed: %v\nOutput: %s", err, string(output))
	}

	// Add generated .pb.h and .pb.cc files
	baseName := strings.TrimSuffix(filepath.Base(file.Path), ".proto")
	relPath := filepath.Dir(file.Path)
	if relPath == "." {
		relPath = ""
	}

	headerFile := filepath.Join(opts.OutputDir, relPath, baseName+".pb.h")
	sourceFile := filepath.Join(opts.OutputDir, relPath, baseName+".pb.cc")
	generatedFiles = append(generatedFiles, headerFile, sourceFile)

	// Generate gRPC code if enabled
	if c.options.GenerateGrpc {
		grpcArgs := c.buildProtocArgs(file, opts, true)

		c.log.Debug("Running protoc for C++ gRPC",
			logger.String("command", c.options.ProtocPath),
			logger.Any("args", grpcArgs))

		grpcCmd := exec.CommandContext(ctx, c.options.ProtocPath, grpcArgs...)
		grpcOutput, err := grpcCmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("protoc gRPC failed: %v\nOutput: %s", err, string(grpcOutput))
		}

		// Add generated .grpc.pb.h and .grpc.pb.cc files
		grpcHeaderFile := filepath.Join(opts.OutputDir, relPath, baseName+".grpc.pb.h")
		grpcSourceFile := filepath.Join(opts.OutputDir, relPath, baseName+".grpc.pb.cc")
		generatedFiles = append(generatedFiles, grpcHeaderFile, grpcSourceFile)
	}

	return generatedFiles, nil
}

// buildProtocArgs builds the protoc command arguments
func (c *Compiler) buildProtocArgs(file compiler.ProtoFile, opts compiler.CompileOptions, grpc bool) []string {
	args := []string{}

	// Add current directory if no import paths specified
	if len(opts.ImportPaths) == 0 && len(file.ImportPaths) == 0 {
		args = append(args, "--proto_path=.")
	}

	// Add import paths
	for _, importPath := range opts.ImportPaths {
		args = append(args, "--proto_path="+importPath)
	}

	// Add import paths from file
	for _, importPath := range file.ImportPaths {
		args = append(args, "--proto_path="+importPath)
	}

	if grpc {
		// Generate gRPC code
		args = append(args, "--grpc_out="+opts.OutputDir)

		// Add plugin path if specified
		if c.options.GrpcCppPluginPath != "" && c.options.GrpcCppPluginPath != "grpc_cpp_plugin" {
			args = append(args, "--plugin=protoc-gen-grpc="+c.options.GrpcCppPluginPath)
		}
	} else {
		// Generate protobuf code
		args = append(args, "--cpp_out="+opts.OutputDir)
	}

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

	headerFile := baseName + ".pb.h"
	if relPath != "" {
		return filepath.Join(opts.OutputDir, relPath, headerFile)
	}
	return filepath.Join(opts.OutputDir, headerFile)
}

// RequiredTools returns the list of required external tools
func (c *Compiler) RequiredTools() []string {
	tools := []string{"protoc"}
	if c.options.GenerateGrpc {
		tools = append(tools, "grpc_cpp_plugin")
	}
	return tools
}
