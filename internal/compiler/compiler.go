// Package compiler defines the common interface for language-specific compilers.
package compiler

import (
	"context"
)

// ProtoFile represents a protobuf file to compile
type ProtoFile struct {
	Path        string   // Absolute path to the proto file
	Package     string   // Proto package name
	ImportPaths []string // Import paths for dependencies
}

// CompileOptions contains options for compilation
type CompileOptions struct {
	// OutputDir is the directory where generated files will be placed
	OutputDir string

	// ImportPaths are additional import paths for proto files
	ImportPaths []string

	// Plugins are additional plugins to use (e.g., "grpc")
	Plugins []string

	// CustomOptions are compiler-specific options
	CustomOptions map[string]string

	// Verbose enables detailed logging
	Verbose bool

	// PreserveProtoStructure preserves the proto directory structure in output
	PreserveProtoStructure bool
}

// CompileResult contains the results of compilation
type CompileResult struct {
	// GeneratedFiles are the paths to generated files
	GeneratedFiles []string

	// Warnings are non-fatal issues encountered during compilation
	Warnings []string

	// Success indicates whether compilation was successful
	Success bool
}

// Compiler is the interface that all language-specific compilers must implement
type Compiler interface {
	// Name returns the name of the compiler (e.g., "python", "go", "rust")
	Name() string

	// Compile compiles the given proto files
	Compile(ctx context.Context, files []ProtoFile, opts CompileOptions) (*CompileResult, error)

	// Validate checks if the compiler is properly configured and available
	Validate() error

	// GetOutputPath returns the output path for the given proto file
	GetOutputPath(protoFile ProtoFile, opts CompileOptions) string

	// RequiredTools returns a list of external tools required by this compiler
	RequiredTools() []string
}

// MergeImportPaths merges import paths from options and file, removing duplicates
func MergeImportPaths(opts CompileOptions, file ProtoFile) []string {
	seen := make(map[string]bool)
	var result []string

	// Add paths from options first
	for _, p := range opts.ImportPaths {
		if !seen[p] {
			seen[p] = true
			result = append(result, p)
		}
	}

	// Add paths from file (skip duplicates)
	for _, p := range file.ImportPaths {
		if !seen[p] {
			seen[p] = true
			result = append(result, p)
		}
	}

	return result
}
