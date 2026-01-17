// Package python implements the Python protobuf compiler.
package python

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

// Compiler implements the compiler.Compiler interface for Python
type Compiler struct {
	log     *logger.Logger
	options *Options
}

// Options contains Python-specific compilation options
type Options struct {
	// ProtocPath is the path to the protoc binary
	ProtocPath string

	// GrpcPythonPluginPath is the path to the grpc_python_plugin
	GrpcPythonPluginPath string

	// GenerateGrpc enables gRPC code generation
	GenerateGrpc bool

	// GenerateTyping enables .pyi stub file generation
	GenerateTyping bool

	// GenerateInit enables __init__.py generation
	GenerateInit bool

	// PythonPackagePrefix is the Python package prefix for imports
	PythonPackagePrefix string
}

// New creates a new Python compiler
func New(log *logger.Logger, opts *Options) *Compiler {
	if opts == nil {
		opts = DefaultOptions()
	}

	// Set defaults
	if opts.ProtocPath == "" {
		opts.ProtocPath = "protoc"
	}

	if opts.GrpcPythonPluginPath == "" {
		opts.GrpcPythonPluginPath = "grpc_python_plugin"
	}

	// Enable init generation by default
	if !opts.GenerateInit {
		opts.GenerateInit = true
	}

	return &Compiler{
		log:     log,
		options: opts,
	}
}

// DefaultOptions returns default options for Python compiler
func DefaultOptions() *Options {
	return &Options{
		ProtocPath:           "protoc",
		GrpcPythonPluginPath: "grpc_python_plugin",
		GenerateGrpc:         true,
		GenerateTyping:       false,
		GenerateInit:         true,
		PythonPackagePrefix:  "",
	}
}

// Name returns the compiler name
func (c *Compiler) Name() string {
	return "python"
}

// Validate checks if the compiler is properly configured
func (c *Compiler) Validate() error {
	// Check if protoc is available
	if err := c.checkTool(c.options.ProtocPath, "--version"); err != nil {
		return errors.Wrap(err, errors.ErrCompilerNotFound, "protoc not found")
	}

	// Check if grpc_python_plugin is available (if gRPC generation is enabled)
	if c.options.GenerateGrpc {
		// Note: grpc_python_plugin might not have a --version flag
		// We'll just check if it exists in PATH
		if _, err := exec.LookPath(c.options.GrpcPythonPluginPath); err != nil {
			c.log.Warn("grpc_python_plugin not found, gRPC generation may fail",
				logger.String("plugin", c.options.GrpcPythonPluginPath))
		}
	}

	return nil
}

// checkTool checks if a tool is available by running it with the given args
func (c *Compiler) checkTool(tool string, args ...string) error {
	cmd := exec.Command(tool, args...)
	if err := cmd.Run(); err != nil {
		if _, lookErr := exec.LookPath(tool); lookErr != nil {
			return fmt.Errorf("tool not found in PATH: %s", tool)
		}
		// Tool exists but command failed - might be expected for version checks
		return nil
	}
	return nil
}

// RequiredTools returns the list of required external tools
func (c *Compiler) RequiredTools() []string {
	tools := []string{"protoc"}
	if c.options.GenerateGrpc {
		tools = append(tools, "grpc_python_plugin")
	}
	return tools
}

// Compile compiles the given proto files to Python
func (c *Compiler) Compile(ctx context.Context, files []compiler.ProtoFile, opts compiler.CompileOptions) (*compiler.CompileResult, error) {
	result := &compiler.CompileResult{
		GeneratedFiles: []string{},
		Warnings:       []string{},
		Success:        false,
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return nil, errors.Wrap(err, errors.ErrFileWrite, "failed to create output directory")
	}

	// Compile each proto file
	for _, file := range files {
		c.log.Debug("Compiling proto file to Python",
			logger.String("file", file.Path),
			logger.String("output", opts.OutputDir))

		generatedFiles, err := c.compileFile(ctx, file, opts)
		if err != nil {
			return nil, errors.Wrap(err, errors.ErrCompilation, "failed to compile proto file: %s", file.Path)
		}

		result.GeneratedFiles = append(result.GeneratedFiles, generatedFiles...)
	}

	// Generate __init__.py files if enabled
	if c.options.GenerateInit {
		if err := c.generateInitFiles(opts.OutputDir); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("failed to generate __init__.py: %v", err))
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
		if protoDir == "." {
			protoDir = "."
		}
	}

	// On Windows, try using python -m grpc_tools.protoc for better gRPC support
	useGrpcTools := c.options.GenerateGrpc && c.isGrpcToolsAvailable()

	if useGrpcTools {
		// Use python -m grpc_tools.protoc which includes gRPC plugin
		args := []string{"-m", "grpc_tools.protoc"}

		// Add proto_path for the proto file directory
		if !opts.PreserveProtoStructure && protoDir != "." {
			args = append(args, "--proto_path="+protoDir)
		}

		// Add current directory if no import paths specified
		if len(opts.ImportPaths) == 0 && len(file.ImportPaths) == 0 {
			args = append(args, "--proto_path=.")
		}

		for _, importPath := range file.ImportPaths {
			args = append(args, "--proto_path="+importPath)
		}

		// Add output directories
		args = append(args, "--python_out="+outputDir)
		args = append(args, "--grpc_python_out="+outputDir)

		// Add the proto file
		args = append(args, protoFilePath)

		c.log.Debug("Running python -m grpc_tools.protoc",
			logger.Any("args", args))

		cmd := exec.CommandContext(ctx, "python", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("grpc_tools.protoc failed: %v\nOutput: %s", err, string(output))
		}

		// Add generated files
		baseName := strings.TrimSuffix(filepath.Base(file.Path), ".proto")
		generatedFiles = append(generatedFiles,
			filepath.Join(outputDir, baseName+"_pb2.py"),
			filepath.Join(outputDir, baseName+"_pb2_grpc.py"))
	} else {
		// Use standard protoc
		args := c.buildProtocArgs(file, opts, outputDir, protoDir, false)
		args = append(args, protoFilePath)

		c.log.Debug("Running protoc for Python",
			logger.String("command", c.options.ProtocPath),
			logger.Any("args", args))

		cmd := exec.CommandContext(ctx, c.options.ProtocPath, args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("protoc failed: %v\nOutput: %s", err, string(output))
		}

		// Add generated _pb2.py file
		baseName := strings.TrimSuffix(filepath.Base(file.Path), ".proto")
		pbFile := filepath.Join(outputDir, baseName+"_pb2.py")
		generatedFiles = append(generatedFiles, pbFile)

		// Generate gRPC code if enabled
		if c.options.GenerateGrpc {
			grpcArgs := c.buildProtocArgs(file, opts, outputDir, protoDir, true)
			grpcArgs = append(grpcArgs, protoFilePath)

			c.log.Debug("Running protoc for Python gRPC",
				logger.String("command", c.options.ProtocPath),
				logger.Any("args", grpcArgs))

			grpcCmd := exec.CommandContext(ctx, c.options.ProtocPath, grpcArgs...)
			grpcOutput, err := grpcCmd.CombinedOutput()
			if err != nil {
				return nil, fmt.Errorf("protoc gRPC failed: %v\nOutput: %s", err, string(grpcOutput))
			}

			// Add generated _pb2_grpc.py file
			grpcFile := filepath.Join(outputDir, baseName+"_pb2_grpc.py")
			generatedFiles = append(generatedFiles, grpcFile)
		}
	}

	return generatedFiles, nil
}

// isGrpcToolsAvailable checks if grpc_tools is available in Python
func (c *Compiler) isGrpcToolsAvailable() bool {
	cmd := exec.Command("python", "-c", "import grpc_tools")
	return cmd.Run() == nil
}

// buildProtocArgs builds the protoc command arguments
func (c *Compiler) buildProtocArgs(file compiler.ProtoFile, opts compiler.CompileOptions, outputDir string, protoDir string, grpc bool) []string {
	args := []string{}

	// Add proto_path for the proto file directory if needed
	if protoDir != "." {
		args = append(args, "--proto_path="+protoDir)
	}

	// Add import paths
	for _, importPath := range opts.ImportPaths {
		args = append(args, "--proto_path="+importPath)
	}

	// Add import paths from file
	for _, importPath := range file.ImportPaths {
		args = append(args, "--proto_path="+importPath)
	}

	// Add output directory
	if grpc {
		args = append(args, "--grpc_python_out="+outputDir)
		// Add plugin path if specified
		if c.options.GrpcPythonPluginPath != "" && c.options.GrpcPythonPluginPath != "grpc_python_plugin" {
			args = append(args, "--plugin=protoc-gen-grpc_python="+c.options.GrpcPythonPluginPath)
		}
	} else {
		args = append(args, "--python_out="+outputDir)
	}

	// Note: file path should be added by caller
	return args
}

// GetOutputPath returns the output path for a proto file
func (c *Compiler) GetOutputPath(file compiler.ProtoFile, opts compiler.CompileOptions) string {
	baseName := strings.TrimSuffix(filepath.Base(file.Path), ".proto")
	return filepath.Join(opts.OutputDir, baseName+"_pb2.py")
}

// generateInitFiles generates __init__.py files in the output directory and subdirectories
func (c *Compiler) generateInitFiles(outputDir string) error {
	c.log.Debug("Generating __init__.py files", logger.String("dir", outputDir))

	// Walk through the output directory
	return filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a directory
		if !info.IsDir() {
			return nil
		}

		// Check if directory contains any .py files
		hasPyFiles := false
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".py") {
				hasPyFiles = true
				break
			}
		}

		// Create __init__.py if directory has Python files
		if hasPyFiles {
			initPath := filepath.Join(path, "__init__.py")

			// Check if __init__.py already exists
			if _, err := os.Stat(initPath); err == nil {
				c.log.Debug("__init__.py already exists, skipping", logger.String("path", initPath))
				return nil
			}

			// Create empty __init__.py
			c.log.Debug("Creating __init__.py", logger.String("path", initPath))
			if err := os.WriteFile(initPath, []byte("# Generated by Buffalo\n"), 0644); err != nil {
				return err
			}
		}

		return nil
	})
}
