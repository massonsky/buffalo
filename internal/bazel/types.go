// Package bazel provides integration between Buffalo and the Bazel build system.
//
// Buffalo can detect Bazel workspaces, parse proto_library targets from BUILD files,
// generate code respecting Bazel's dependency graph, and produce BUILD.bazel files
// for the generated output so that downstream Bazel targets can depend on it.
package bazel

// BazelTarget represents a target parsed from a BUILD/BUILD.bazel file.
type BazelTarget struct {
	// Name is the Bazel target name (e.g., "user_proto").
	Name string

	// Rule is the Bazel rule name (e.g., "proto_library").
	Rule string

	// Package is the Bazel package path (e.g., "//proto/user").
	Package string

	// Srcs lists source files declared in srcs = [...].
	Srcs []string

	// Deps lists dependency labels declared in deps = [...].
	Deps []string

	// Visibility lists visibility labels.
	Visibility []string

	// StripImportPrefix is the strip_import_prefix attribute.
	StripImportPrefix string

	// ImportPrefix is the import_prefix attribute.
	ImportPrefix string

	// ProtoSourceRoot is the proto_source_root attribute.
	ProtoSourceRoot string

	// Tags are build tags.
	Tags []string

	// Extra holds any additional attributes as key-value pairs.
	Extra map[string]string
}

// LanguageBinding maps a proto_library target to language-specific
// generated library targets (go_proto_library, py_proto_library, etc.).
type LanguageBinding struct {
	// Language is the target language (e.g., "go", "python", "cpp", "rust", "typescript").
	Language string

	// Rule is the Bazel rule to generate (e.g., "go_proto_library").
	Rule string

	// TargetName is the generated target name.
	TargetName string

	// ProtoTarget is the label of the source proto_library.
	ProtoTarget string

	// Deps are language-specific dependencies.
	Deps []string

	// Importpath is language-specific import path (Go module, Python package, etc.).
	Importpath string

	// Options are extra rule attributes.
	Options map[string]string
}

// Workspace describes a detected Bazel workspace.
type Workspace struct {
	// Root is the absolute path to the workspace root.
	Root string

	// Type is "bzlmod" (MODULE.bazel) or "workspace" (WORKSPACE/WORKSPACE.bazel).
	Type string

	// ModuleName is the module name from MODULE.bazel (bzlmod only).
	ModuleName string

	// ProtoTargets are all proto_library targets discovered in the workspace.
	ProtoTargets []BazelTarget

	// BuildFiles maps Bazel package paths to their BUILD file paths on disk.
	BuildFiles map[string]string
}

// GeneratedBuild represents a BUILD.bazel file to be written for generated code.
type GeneratedBuild struct {
	// Path is the relative path where the BUILD.bazel should be written.
	Path string

	// Content is the rendered BUILD.bazel content.
	Content string

	// Bindings are the language bindings declared in this BUILD file.
	Bindings []LanguageBinding
}

// QueryResult holds results from `bazel query` or `bazel cquery`.
type QueryResult struct {
	// Targets lists discovered target labels.
	Targets []string

	// Deps maps a target label to its transitive dependency labels.
	Deps map[string][]string

	// ProtoFiles maps a target label to its resolved proto source files.
	ProtoFiles map[string][]string
}

// SyncPlan describes what Buffalo will do to synchronize with Bazel.
type SyncPlan struct {
	// TargetsToCompile lists proto_library targets to compile.
	TargetsToCompile []BazelTarget

	// BuildFilesToGenerate lists BUILD.bazel files to create/update.
	BuildFilesToGenerate []GeneratedBuild

	// Languages are the target languages.
	Languages []string

	// OutputDir is the output root.
	OutputDir string
}
