package rust

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
	if err := c.checkTool(c.options.ProtocPath); err != nil {
		return errors.Wrap(err, errors.ErrConfig, "protoc not found")
	}

	// Check generator-specific tools
	switch c.options.Generator {
	case "prost":
		if err := c.checkTool("protoc-gen-prost"); err != nil {
			return errors.Wrap(err, errors.ErrConfig, "protoc-gen-prost not found, install: cargo install protoc-gen-prost")
		}
		if c.options.GenerateGrpc {
			if err := c.checkTool("protoc-gen-tonic"); err != nil {
				return errors.Wrap(err, errors.ErrConfig, "protoc-gen-tonic not found (optional for gRPC), install: cargo install protoc-gen-tonic")
			}
		}
	case "rust-protobuf", "protoc-gen-rs":
		if err := c.checkTool("protoc-gen-rs"); err != nil {
			return errors.Wrap(err, errors.ErrConfig, "protoc-gen-rs not found, install: cargo install protobuf-codegen")
		}
	default:
		return errors.New(errors.ErrConfig, "unknown Rust generator: %s (use 'prost' or 'rust-protobuf')", c.options.Generator)
	}

	return nil
}

// checkTool checks if a tool is available
func (c *Compiler) checkTool(toolPath string) error {
	cmd := exec.Command(toolPath, "--version")
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

	if c.options.Generator == "prost" {
		warning, err := c.validateProstCargoSetup(files)
		if err == nil {
			// Cargo project (Cargo.toml + build.rs) exists alongside the
			// protos — defer .rs generation to `cargo build` via
			// prost_build::compile_protos. Buffalo only validates the
			// integration in this mode.
			result.Warnings = append(result.Warnings, warning)
			result.Success = true
			return result, nil
		}

		// No Cargo project found: fall back to direct, hermetic generation
		// via `protoc --prost_out=` (+ `--tonic_out=` for gRPC). This is the
		// path used by Bazel's `buffalo_proto_compile`, where the sandbox
		// stages `protoc-gen-prost` / `protoc-gen-tonic` next to protoc and
		// no Cargo project lives next to the .proto sources. The validation
		// error is downgraded to a warning so callers can see why the
		// Cargo-integration mode was skipped.
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Rust/prost: Cargo integration not detected (%v) — generating .rs files directly via protoc-gen-prost.", err))
	}

	for _, file := range files {
		c.log.Debug("Compiling proto file to Rust",
			logger.String("file", file.Path),
			logger.String("output", opts.OutputDir),
			logger.String("generator", c.options.Generator))

		generatedFiles, err := c.compileFile(ctx, file, opts)
		if err != nil {
			return nil, errors.Wrap(err, errors.ErrCompilation, "failed to compile proto file: %s", file.Path)
		}

		result.GeneratedFiles = append(result.GeneratedFiles, generatedFiles...)
	}

	result.Success = true
	return result, nil
}

func (c *Compiler) validateProstCargoSetup(files []compiler.ProtoFile) (string, error) {
	projectRoot, cargoTomlPath, buildRsPath, err := c.findCargoProject(files)
	if err != nil {
		protoFile := "<unknown>"
		if len(files) > 0 {
			protoFile = files[0].Path
		}

		result := fmt.Sprintf("Prost requires Cargo integration. Add to build.rs:\n"+
			"    prost_build::compile_protos(&[\"%s\"], &[\".\"])?;", protoFile)
		return "", fmt.Errorf("prost generator requires manual Cargo setup:\n%s", result)
	}

	buildRsContent, err := os.ReadFile(buildRsPath)
	if err != nil {
		return "", fmt.Errorf("failed to read build.rs: %w", err)
	}

	buildRsSource := string(buildRsContent)
	if !strings.Contains(buildRsSource, "compile_protos(") {
		return "", fmt.Errorf("prost generator requires manual Cargo setup:\nfound build.rs at %s but it does not call compile_protos(...)", buildRsPath)
	}

	return fmt.Sprintf(
		"Rust/prost uses Cargo integration at %s (Cargo.toml: %s, build.rs: %s). Generated Rust sources are produced during `cargo build`, not directly by `buffalo build`.",
		projectRoot,
		cargoTomlPath,
		buildRsPath,
	), nil
}

func (c *Compiler) findCargoProject(files []compiler.ProtoFile) (string, string, string, error) {
	seen := make(map[string]struct{})
	candidates := make([]string, 0, len(files)+1)

	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, wd)
	}

	for _, file := range files {
		absFilePath, err := filepath.Abs(file.Path)
		if err != nil {
			continue
		}
		candidates = append(candidates, filepath.Dir(absFilePath))
	}

	for _, candidate := range candidates {
		for dir := candidate; dir != ""; dir = filepath.Dir(dir) {
			if _, ok := seen[dir]; ok {
				if dir == filepath.Dir(dir) {
					break
				}
				continue
			}
			seen[dir] = struct{}{}

			cargoTomlPath := filepath.Join(dir, "Cargo.toml")
			buildRsPath := filepath.Join(dir, "build.rs")
			if fileExists(cargoTomlPath) && fileExists(buildRsPath) {
				return dir, cargoTomlPath, buildRsPath, nil
			}

			if dir == filepath.Dir(dir) {
				break
			}
		}
	}

	return "", "", "", fmt.Errorf("Cargo.toml and build.rs were not found in the current project or its parent directories")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// compileFile compiles a single proto file
func (c *Compiler) compileFile(ctx context.Context, file compiler.ProtoFile, opts compiler.CompileOptions) ([]string, error) {
	var generatedFiles []string

	switch c.options.Generator {
	case "prost":
		// Direct, hermetic generation via protoc-gen-prost (+ tonic for gRPC).
		// Reached only when no Cargo project sits next to the .proto sources;
		// the Cargo-integration path returns early in Compile().
		if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create output directory: %v", err)
		}

		args := c.buildProtocArgs(file, opts, "prost")

		c.log.Debug("Running protoc for Rust (prost)",
			logger.String("command", c.options.ProtocPath),
			logger.Any("args", args))

		cmd := exec.CommandContext(ctx, c.options.ProtocPath, args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("protoc failed: %v\nOutput: %s", err, string(output))
		}

		generatedFiles = append(generatedFiles, c.GetOutputPath(file, opts))
		return generatedFiles, nil

	case "rust-protobuf":
		// Create output directory if it doesn't exist
		if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create output directory: %v", err)
		}

		// Use protoc with rust-protobuf plugin
		args := c.buildProtocArgs(file, opts, "rust-protobuf")

		c.log.Debug("Running protoc for Rust (rust-protobuf)",
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

	default:
		return nil, errors.New(errors.ErrConfig, "unknown Rust generator: %s", c.options.Generator)
	}

	return generatedFiles, nil
}

// buildProtocArgs builds the protoc command arguments
func (c *Compiler) buildProtocArgs(file compiler.ProtoFile, opts compiler.CompileOptions, generator string) []string {
	args := []string{}

	// Always add current directory as proto path
	args = append(args, "--proto_path=.")

	// Merge and deduplicate import paths
	importPaths := compiler.MergeImportPaths(opts, file)

	// Add import paths
	for _, importPath := range importPaths {
		if importPath != "." {
			args = append(args, "--proto_path="+importPath)
		}
	}

	// Add plugin-specific output arguments
	switch generator {
	case "prost":
		args = append(args, "--prost_out="+opts.OutputDir)
		if c.options.GenerateGrpc {
			args = append(args, "--tonic_out="+opts.OutputDir)
		}
	case "rust-protobuf":
		args = append(args, "--rs_out="+opts.OutputDir)
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
		tools = append(tools, "protoc-gen-rs")
	}
	return tools
}
