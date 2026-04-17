// Package typescript implements the TypeScript protobuf compiler.
package typescript

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/massonsky/buffalo/internal/compiler"
	"github.com/massonsky/buffalo/pkg/errors"
	"github.com/massonsky/buffalo/pkg/logger"
)

var nonIdentCharRegexp = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// Compiler implements the compiler.Compiler interface for TypeScript
type Compiler struct {
	log     *logger.Logger
	options *Options
}

// Options contains TypeScript-specific compilation options
type Options struct {
	// ProtocPath is the path to the protoc binary
	ProtocPath string

	// Generator selects the TS protobuf generator: "protoc-gen-ts" or "ts-proto"
	Generator string

	// ProtocGenTsPath is the path to the protoc-gen-ts plugin
	ProtocGenTsPath string

	// TsProtoPath is the path to the protoc-gen-ts_proto plugin
	TsProtoPath string

	// GenerateGrpc enables gRPC-Web or gRPC code generation
	GenerateGrpc bool

	// GenerateGrpcWeb uses grpc-web for browser compatibility
	GenerateGrpcWeb bool

	// GenerateNiceGrpc uses nice-grpc for server/client generation (ts-proto)
	GenerateNiceGrpc bool

	// ESModules enables ES module output (import/export) instead of CommonJS
	ESModules bool

	// OutputIndex generates an index.ts barrel file
	OutputIndex bool

	// PackageName is an optional npm package name for generated code
	PackageName string
}

// New creates a new TypeScript compiler
func New(log *logger.Logger, opts *Options) *Compiler {
	if opts == nil {
		opts = DefaultOptions()
	}

	if opts.ProtocPath == "" {
		opts.ProtocPath = "protoc"
	}

	if opts.Generator == "" {
		opts.Generator = "ts-proto"
	}

	if opts.ProtocGenTsPath == "" {
		opts.ProtocGenTsPath = "protoc-gen-ts"
	}

	if opts.TsProtoPath == "" {
		opts.TsProtoPath = "protoc-gen-ts_proto"
	}

	return &Compiler{
		log:     log,
		options: opts,
	}
}

// DefaultOptions returns default options for TypeScript compiler
func DefaultOptions() *Options {
	return &Options{
		ProtocPath:      "protoc",
		Generator:       "ts-proto",
		ProtocGenTsPath: "protoc-gen-ts",
		TsProtoPath:     "protoc-gen-ts_proto",
		GenerateGrpc:    true,
		GenerateGrpcWeb: false,
		ESModules:       true,
		OutputIndex:     true,
	}
}

// Name returns the compiler name
func (c *Compiler) Name() string {
	return "typescript"
}

// Validate checks if the compiler is properly configured
func (c *Compiler) Validate() error {
	// Check if protoc is available
	if err := c.checkTool(c.options.ProtocPath, "--version"); err != nil {
		return errors.Wrap(err, errors.ErrCompilerNotFound, "protoc not found")
	}

	// Check if the TS plugin is available
	switch c.options.Generator {
	case "ts-proto":
		if _, err := exec.LookPath(c.options.TsProtoPath); err != nil {
			return errors.Wrap(err, errors.ErrCompilerNotFound,
				fmt.Sprintf("%s not found, install: npm install -g ts-proto",
					c.options.TsProtoPath))
		}
	default:
		if _, err := exec.LookPath(c.options.ProtocGenTsPath); err != nil {
			return errors.Wrap(err, errors.ErrCompilerNotFound,
				fmt.Sprintf("%s not found, install: npm install -g %s",
					c.options.ProtocGenTsPath, c.options.Generator))
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
		return nil
	}
	return nil
}

// RequiredTools returns the list of required external tools
func (c *Compiler) RequiredTools() []string {
	tools := []string{"protoc"}
	switch c.options.Generator {
	case "ts-proto":
		tools = append(tools, c.options.TsProtoPath)
	default:
		tools = append(tools, c.options.ProtocGenTsPath)
	}
	if c.options.GenerateGrpcWeb {
		tools = append(tools, "protoc-gen-grpc-web")
	}
	return tools
}

// Compile compiles the given proto files to TypeScript
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
		c.log.Debug("Compiling proto file to TypeScript",
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

// compileFile compiles a single proto file to TypeScript
func (c *Compiler) compileFile(ctx context.Context, file compiler.ProtoFile, opts compiler.CompileOptions) ([]string, error) {
	var generatedFiles []string

	outputDir := opts.OutputDir

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %v", err)
	}

	protoFilePath := file.Path
	protoDir := "."
	if !opts.PreserveProtoStructure {
		protoFilePath = filepath.Base(file.Path)
		protoDir = filepath.Dir(file.Path)
		if protoDir == "." {
			protoDir = "."
		}
	}

	args := c.buildProtocArgs(file, opts, outputDir, protoDir)
	args = append(args, protoFilePath)

	c.log.Debug("Running protoc for TypeScript",
		logger.String("command", c.options.ProtocPath),
		logger.Any("args", args))

	cmd := exec.CommandContext(ctx, c.options.ProtocPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("protoc failed: %v\nOutput: %s", err, string(output))
	}

	baseName := strings.TrimSuffix(filepath.Base(file.Path), ".proto")

	switch c.options.Generator {
	case "ts-proto":
		// ts-proto generates a single .ts file per proto
		tsFile := filepath.Join(outputDir, baseName+".ts")
		generatedFiles = append(generatedFiles, tsFile)
	default:
		// protoc-gen-ts generates _pb.ts (and optionally _grpc_pb.ts)
		pbFile := filepath.Join(outputDir, baseName+"_pb.ts")
		generatedFiles = append(generatedFiles, pbFile)

		if c.options.GenerateGrpc {
			grpcFile := filepath.Join(outputDir, baseName+"_grpc_pb.ts")
			generatedFiles = append(generatedFiles, grpcFile)
		}
	}

	return generatedFiles, nil
}

// buildProtocArgs builds the protoc command arguments for TypeScript
func (c *Compiler) buildProtocArgs(file compiler.ProtoFile, opts compiler.CompileOptions, outputDir string, protoDir string) []string {
	args := []string{}

	if protoDir != "." {
		args = append(args, "--proto_path="+protoDir)
	}

	for _, importPath := range compiler.MergeImportPaths(opts, file) {
		args = append(args, "--proto_path="+importPath)
	}

	switch c.options.Generator {
	case "ts-proto":
		if c.options.TsProtoPath != "protoc-gen-ts_proto" {
			args = append(args, "--plugin=protoc-gen-ts_proto="+c.options.TsProtoPath)
		}
		tsProtoOpts := []string{"outputServices=default", "esModuleInterop=true"}
		if c.options.ESModules {
			tsProtoOpts = append(tsProtoOpts, "importSuffix=.js")
		}
		if c.options.GenerateNiceGrpc {
			tsProtoOpts = append(tsProtoOpts, "outputServices=nice-grpc", "outputServices=generic-definitions")
		}
		optStr := strings.Join(tsProtoOpts, ",")
		args = append(args, fmt.Sprintf("--ts_proto_out=%s", outputDir))
		args = append(args, fmt.Sprintf("--ts_proto_opt=%s", optStr))
	default:
		// protoc-gen-ts
		args = append(args, fmt.Sprintf("--ts_out=%s", outputDir))
		if c.options.ProtocGenTsPath != "protoc-gen-ts" {
			args = append(args, "--plugin=protoc-gen-ts="+c.options.ProtocGenTsPath)
		}
	}

	// gRPC-Web support
	if c.options.GenerateGrpcWeb {
		args = append(args, fmt.Sprintf("--grpc-web_out=import_style=typescript,mode=grpcwebtext:%s", outputDir))
	}

	return args
}

// GenerateIndexFile creates a single index.ts barrel after all TS files are generated.
func (c *Compiler) GenerateIndexFile(outputDir string, generatedFiles []string) error {
	if !c.options.OutputIndex {
		return nil
	}
	return c.generateIndexFile(outputDir, generatedFiles)
}

// GetOutputPath returns the output path for a proto file
func (c *Compiler) GetOutputPath(file compiler.ProtoFile, opts compiler.CompileOptions) string {
	baseName := strings.TrimSuffix(filepath.Base(file.Path), ".proto")
	switch c.options.Generator {
	case "ts-proto":
		return filepath.Join(opts.OutputDir, baseName+".ts")
	default:
		return filepath.Join(opts.OutputDir, baseName+"_pb.ts")
	}
}

// generateIndexFile generates an index.ts barrel file with namespace re-exports,
// avoiding collisions from wildcard exports.
func (c *Compiler) generateIndexFile(outputDir string, generatedFiles []string) error {
	c.log.Debug("Generating index.ts", logger.String("dir", outputDir))

	seenModules := make(map[string]struct{})
	seenAliases := make(map[string]int)
	exports := make([]string, 0, len(generatedFiles))
	for _, f := range generatedFiles {
		rel, err := filepath.Rel(outputDir, f)
		if err != nil {
			continue
		}
		if rel == "" {
			continue
		}
		// Convert to module path (no extension for imports)
		module := strings.TrimSuffix(rel, filepath.Ext(rel))
		module = strings.ReplaceAll(module, string(filepath.Separator), "/")
		if _, ok := seenModules[module]; ok {
			continue
		}
		seenModules[module] = struct{}{}

		alias := moduleAlias(module)
		if n, exists := seenAliases[alias]; exists {
			n++
			seenAliases[alias] = n
			alias = fmt.Sprintf("%s_%d", alias, n)
		} else {
			seenAliases[alias] = 0
		}

		exports = append(exports, fmt.Sprintf("export * as %s from './%s';", alias, module))
	}

	if len(exports) == 0 {
		return nil
	}

	sort.Strings(exports)

	content := "// Generated by Buffalo\n" + strings.Join(exports, "\n") + "\n"
	indexPath := filepath.Join(outputDir, "index.ts")

	return os.WriteFile(indexPath, []byte(content), 0600)
}

func moduleAlias(module string) string {
	alias := strings.ReplaceAll(module, "/", "_")
	alias = strings.ReplaceAll(alias, "-", "_")
	alias = nonIdentCharRegexp.ReplaceAllString(alias, "_")
	alias = strings.Trim(alias, "_")
	if alias == "" {
		return "module"
	}
	if alias[0] >= '0' && alias[0] <= '9' {
		alias = "m_" + alias
	}
	return alias
}
