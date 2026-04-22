// Package compiler defines the common interface for language-specific compilers.
package compiler

import (
	"context"
	"path/filepath"
	"strings"
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

	// ProjectDir is the project root directory used for calculating relative paths.
	// If empty, os.Getwd() is used as fallback.
	ProjectDir string
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

// ResolveProtoFileArg returns the proto file path that should be passed to
// protoc as the input argument, expressed relative to the longest matching
// --proto_path entry. This avoids the "Input is shadowed in the --proto_path"
// error when the file is reachable via multiple paths (e.g. the file is given
// as `proto/foo.proto` while `--proto_path=proto` is also passed, making the
// file also reachable as `foo.proto`).
//
// If no import path is a prefix of filePath, filePath is returned unchanged.
func ResolveProtoFileArg(filePath string, importPaths []string) string {
	clean := filepath.ToSlash(filepath.Clean(filePath))
	bestRel := ""
	bestLen := -1
	for _, p := range importPaths {
		if p == "" {
			continue
		}
		base := filepath.ToSlash(filepath.Clean(p))
		if base == "." {
			continue
		}
		prefix := base + "/"
		if strings.HasPrefix(clean, prefix) {
			if len(base) > bestLen {
				bestLen = len(base)
				bestRel = strings.TrimPrefix(clean, prefix)
			}
		} else if clean == base {
			if len(base) > bestLen {
				bestLen = len(base)
				bestRel = filepath.Base(clean)
			}
		}
	}
	if bestRel == "" {
		return filePath
	}
	return bestRel
}
