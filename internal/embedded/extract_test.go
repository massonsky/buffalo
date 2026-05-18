package embedded

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/massonsky/buffalo/internal/version"
)

func TestProtoFS_Contains_ValidateProto(t *testing.T) {
	data, err := ProtoFS.ReadFile("proto/buffalo/validate/validate.proto")
	if err != nil {
		t.Fatalf("failed to read embedded validate.proto: %v", err)
	}
	if len(data) == 0 {
		t.Error("embedded validate.proto is empty")
	}
	content := string(data)
	if !strings.Contains(content, "package buffalo.validate") {
		t.Error("expected embedded proto to contain 'package buffalo.validate'")
	}
	if !strings.Contains(content, "FieldRules") {
		t.Error("expected embedded proto to contain 'FieldRules'")
	}
}

func TestProtoFS_Contains_ModelsProto(t *testing.T) {
	data, err := ProtoFS.ReadFile("proto/buffalo/models/models.proto")
	if err != nil {
		t.Fatalf("failed to read embedded models.proto: %v", err)
	}
	if len(data) == 0 {
		t.Error("embedded models.proto is empty")
	}
	content := string(data)
	if !strings.Contains(content, "package buffalo.models") {
		t.Error("expected embedded proto to contain 'package buffalo.models'")
	}
	if !strings.Contains(content, "ModelOptions") {
		t.Error("expected embedded proto to contain 'ModelOptions'")
	}
}

func TestListEmbeddedProtos(t *testing.T) {
	files, err := ListEmbeddedProtos()
	if err != nil {
		t.Fatalf("ListEmbeddedProtos failed: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected at least one embedded proto file")
	}

	found := false
	for _, f := range files {
		if strings.Contains(f, "validate.proto") {
			found = true
		}
	}
	if !found {
		t.Errorf("validate.proto not found in embedded protos: %v", files)
	}
}

func TestExtractAllProtos(t *testing.T) {
	tmpDir := t.TempDir()

	protoPath, err := ExtractAllProtos(tmpDir)
	if err != nil {
		t.Fatalf("ExtractAllProtos failed: %v", err)
	}

	expectedProtoPath := filepath.Join(tmpDir, "proto")
	if protoPath != expectedProtoPath {
		t.Errorf("expected protoPath %q, got %q", expectedProtoPath, protoPath)
	}

	// Verify the file was extracted
	validatePath := filepath.Join(tmpDir, "proto", "buffalo", "validate", "validate.proto")
	info, err := os.Stat(validatePath)
	if err != nil {
		t.Fatalf("extracted file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Error("extracted file is empty")
	}

	// Verify content
	data, err := os.ReadFile(validatePath)
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}
	if !strings.Contains(string(data), "package buffalo.validate") {
		t.Error("extracted file content mismatch")
	}
}

func TestExtractAllProtos_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()

	// Extract twice — should not error
	_, err := ExtractAllProtos(tmpDir)
	if err != nil {
		t.Fatalf("first extract failed: %v", err)
	}

	_, err = ExtractAllProtos(tmpDir)
	if err != nil {
		t.Fatalf("second extract (overwrite) failed: %v", err)
	}
}

func TestValidateProtoImportPath_AutoExtract(t *testing.T) {
	tmpDir := t.TempDir()

	// File doesn't exist yet → auto-extract
	importPath, err := ValidateProtoImportPath(tmpDir)
	if err != nil {
		t.Fatalf("ValidateProtoImportPath failed: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, "proto")
	if importPath != expectedPath {
		t.Errorf("expected import path %q, got %q", expectedPath, importPath)
	}

	// File now exists → no re-extract needed
	importPath2, err := ValidateProtoImportPath(tmpDir)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if importPath2 != expectedPath {
		t.Errorf("expected import path %q on second call, got %q", expectedPath, importPath2)
	}
}

func TestExtractValidateProto(t *testing.T) {
	tmpDir := t.TempDir()

	protoPath, err := ExtractValidateProto(tmpDir)
	if err != nil {
		t.Fatalf("ExtractValidateProto failed: %v", err)
	}

	// Verify directory structure
	expectedFile := filepath.Join(tmpDir, "proto", "buffalo", "validate", "validate.proto")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Fatalf("validate.proto not extracted to expected path: %s", expectedFile)
	}

	// protoPath should be the directory to pass to protoc --proto_path
	if !strings.HasSuffix(protoPath, "proto") {
		t.Errorf("protoPath should end with 'proto', got %q", protoPath)
	}
}

func TestBazelRules_DefaultBuffaloReleaseIsTemplatedAndPosixWrapperQuoting(t *testing.T) {
	repositories, err := BazelFS.ReadFile("bazel/rules_buffalo/buffalo/repositories.bzl")
	if err != nil {
		t.Fatalf("failed to read embedded repositories.bzl: %v", err)
	}
	if !strings.Contains(string(repositories), `DEFAULT_BUFFALO_VERSION = "{{BUFFALO_VERSION}}"`) {
		t.Fatal("embedded rules_buffalo must template the Buffalo release version")
	}

	defs, err := BazelFS.ReadFile("bazel/rules_buffalo/buffalo/defs.bzl")
	if err != nil {
		t.Fatalf("failed to read embedded defs.bzl: %v", err)
	}
	content := string(defs)
	for _, literal := range []string{"'$STAGE/", "'$TOOLS/", "'$EXECROOT/"} {
		if strings.Contains(content, literal) {
			t.Fatalf("POSIX wrapper must not single-quote shell variables literally: found %s", literal)
		}
	}
	for _, expanded := range []string{`\"$STAGE\"/`, `\"$TOOLS\"/`, `\"$EXECROOT\"/`} {
		if !strings.Contains(content, expanded) {
			t.Fatalf("POSIX wrapper should expand shell variables in double quotes: missing %s", expanded)
		}
	}
}

func TestBazelRules_DefaultBazelOutputMatchesInitConfig(t *testing.T) {
	defs, err := BazelFS.ReadFile("bazel/rules_buffalo/buffalo/defs.bzl")
	if err != nil {
		t.Fatalf("failed to read embedded defs.bzl: %v", err)
	}
	content := string(defs)
	for _, want := range []string{
		`default = "generated"`,
		`Keep this aligned with output.base_dir in buffalo.yaml.`,
		`compile_out = None`,
		`compile_out = out if package_name == "" else package_name + "/" + out`,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("defs.bzl should keep Bazel output aligned with Buffalo config output: missing %q", want)
		}
	}

	readme, err := BazelFS.ReadFile("bazel/rules_buffalo/README.md")
	if err != nil {
		t.Fatalf("failed to read embedded rules_buffalo README.md: %v", err)
	}
	if !strings.Contains(string(readme), `out = "generated"`) {
		t.Fatal("rules_buffalo README should show out matching output.base_dir")
	}
}

func TestBazelRules_RunTargetUsesHermeticCompileOutput(t *testing.T) {
	defs, err := BazelFS.ReadFile("bazel/rules_buffalo/buffalo/defs.bzl")
	if err != nil {
		t.Fatalf("failed to read embedded defs.bzl: %v", err)
	}
	content := string(defs)
	for _, want := range []string{
		`srcs = None`,
		`copy_from_bazel_bin = None`,
		`copy_from_bazel_bin = srcs != None or compile_target != None`,
		`buffalo_proto_compile(`,
		`name = compile_name`,
		`compile_target = ":" + compile_name`,
		`data.append(compile_target)`,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("buffalo_proto_gen should build a hermetic compile target before bazel run copy mode: missing %q", want)
		}
	}

	runner, err := BazelFS.ReadFile("bazel/rules_buffalo/buffalo/gen_runner.py")
	if err != nil {
		t.Fatalf("failed to read embedded gen_runner.py: %v", err)
	}
	runnerContent := string(runner)
	for _, want := range []string{
		`path or "generated"`,
		`--copy-from-bazel-bin`,
		`src_dir = os.path.join(root, "bazel-bin", known.copy_from_bazel_bin)`,
		`dst_rel = _read_output_dir(config_path)`,
	} {
		if !strings.Contains(runnerContent, want) {
			t.Fatalf("gen_runner.py should copy Bazel output into config output dir: missing %q", want)
		}
	}
}

func TestBazelRules_RustPluginsAreWiredThroughBuffaloRust(t *testing.T) {
	moduleFile, err := BazelFS.ReadFile("bazel/rules_buffalo/MODULE.bazel")
	if err != nil {
		t.Fatalf("failed to read embedded rules_buffalo MODULE.bazel: %v", err)
	}
	moduleContent := string(moduleFile)
	for _, want := range []string{
		`bazel_dep(name = "rules_rust", version = "0.66.0")`,
		`rust_host_tools = use_extension("@rules_rust//rust:extensions.bzl", "rust_host_tools")`,
		`rust_host_tools.host_tools(`,
		`name = "rust_host_tools_nightly"`,
		`version = "nightly/2025-02-17"`,
		`use_repo(rust_host_tools, "rust_host_tools_nightly")`,
		`rust_plugins = use_extension("@rules_rust//crate_universe:extensions.bzl", "crate")`,
		`package = "protoc-gen-prost"`,
		`package = "protoc-gen-tonic"`,
		`artifact = "bin"`,
		`rust_plugins.from_specs(`,
		`name = "buffalo_rust_plugins"`,
		`generate_binaries = True`,
		`host_tools = "@rust_host_tools_nightly//:rust_host_tools"`,
		`use_repo(rust_plugins, "buffalo_rust_plugins")`,
	} {
		if !strings.Contains(moduleContent, want) {
			t.Fatalf("MODULE.bazel is missing bundled Rust plugin dependency %q", want)
		}
	}
	if strings.Contains(moduleContent, `host_tools = "@rust_host_tools_nightly"`) {
		t.Fatal("rules_buffalo MODULE.bazel should pass a fully qualified rust_host_tools target label")
	}

	defs, err := BazelFS.ReadFile("bazel/rules_buffalo/buffalo/defs.bzl")
	if err != nil {
		t.Fatalf("failed to read embedded defs.bzl: %v", err)
	}
	defsContent := string(defs)
	for _, want := range []string{
		`"protoc_gen_prost": attr.label(`,
		`default = Label("@buffalo_rust_plugins//:protoc-gen-prost__protoc-gen-prost")`,
		`"protoc_gen_tonic": attr.label(`,
		`default = Label("@buffalo_rust_plugins//:protoc-gen-tonic__protoc-gen-tonic")`,
		`if rust_enabled and ctx.file.protoc_gen_prost:`,
		`if rust_enabled and ctx.file.protoc_gen_tonic:`,
	} {
		if !strings.Contains(defsContent, want) {
			t.Fatalf("defs.bzl is missing direct Rust plugin action input wiring %q", want)
		}
	}
	if strings.Contains(defsContent, `@buffalo_toolchain//:protoc_gen_prost_bin`) ||
		strings.Contains(defsContent, `@buffalo_toolchain//:protoc_gen_tonic_bin`) {
		t.Fatal("Rust plugin defaults must not point at @buffalo_toolchain aliases")
	}

	repositories, err := BazelFS.ReadFile("bazel/rules_buffalo/buffalo/repositories.bzl")
	if err != nil {
		t.Fatalf("failed to read embedded repositories.bzl: %v", err)
	}
	repoContent := string(repositories)
	for _, forbidden := range []string{
		`_stage_label_tool`,
		`protoc_gen_prost_bin`,
		`protoc_gen_tonic_bin`,
		`"protoc_gen_prost": attr.label(allow_single_file = True)`,
		`"protoc_gen_tonic": attr.label(allow_single_file = True)`,
	} {
		if strings.Contains(repoContent, forbidden) {
			t.Fatalf("buffalo_toolchain repository must not stage Rust build outputs: found %q", forbidden)
		}
	}
}

func TestExtractBazelRules_RendersBuffaloVersionTemplate(t *testing.T) {
	tmpDir := t.TempDir()

	rulesPath, err := ExtractBazelRules(tmpDir)
	if err != nil {
		t.Fatalf("ExtractBazelRules failed: %v", err)
	}

	repositoriesPath := filepath.Join(rulesPath, "buffalo", "repositories.bzl")
	data, err := os.ReadFile(repositoriesPath)
	if err != nil {
		t.Fatalf("failed to read extracted repositories.bzl: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "{{BUFFALO_VERSION}}") {
		t.Fatal("extracted repositories.bzl still contains the Buffalo version template")
	}
	if !strings.Contains(content, `DEFAULT_BUFFALO_VERSION = "1.33.18"`) {
		t.Fatalf("dev builds should render the fallback Buffalo release version, got:\n%s", content)
	}
}

func TestExtractBazelRules_UsesInjectedReleaseVersion(t *testing.T) {
	original := version.Version
	version.Version = "9.8.7"
	t.Cleanup(func() {
		version.Version = original
	})

	tmpDir := t.TempDir()
	rulesPath, err := ExtractBazelRules(tmpDir)
	if err != nil {
		t.Fatalf("ExtractBazelRules failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(rulesPath, "buffalo", "repositories.bzl"))
	if err != nil {
		t.Fatalf("failed to read extracted repositories.bzl: %v", err)
	}
	if !strings.Contains(string(data), `DEFAULT_BUFFALO_VERSION = "9.8.7"`) {
		t.Fatal("extracted rules_buffalo should use the build-time injected Buffalo version")
	}
}
