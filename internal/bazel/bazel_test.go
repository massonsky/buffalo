package bazel

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectWorkspace_Bzlmod(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "MODULE.bazel"), []byte(`module(name = "myproject", version = "1.0")`), 0o644); err != nil {
		t.Fatal(err)
	}

	ws, err := DetectWorkspace(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ws == nil {
		t.Fatal("expected workspace, got nil")
	}
	if ws.Type != "bzlmod" {
		t.Errorf("expected type 'bzlmod', got %q", ws.Type)
	}
	if ws.ModuleName != "myproject" {
		t.Errorf("expected module name 'myproject', got %q", ws.ModuleName)
	}
}

func TestDetectWorkspace_Legacy(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "WORKSPACE"), []byte(`workspace(name = "legacy")`), 0o644); err != nil {
		t.Fatal(err)
	}

	ws, err := DetectWorkspace(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ws == nil {
		t.Fatal("expected workspace, got nil")
	}
	if ws.Type != "workspace" {
		t.Errorf("expected type 'workspace', got %q", ws.Type)
	}
}

func TestDetectWorkspace_None(t *testing.T) {
	tmp := t.TempDir()

	ws, err := DetectWorkspace(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ws != nil {
		t.Fatal("expected nil workspace")
	}
}

func TestDetectWorkspace_Subdir(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "MODULE.bazel"), []byte(`module(name = "root")`), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(tmp, "sub", "dir")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	ws, err := DetectWorkspace(sub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ws == nil {
		t.Fatal("expected workspace")
	}
	if ws.Root != tmp {
		t.Errorf("expected root %q, got %q", tmp, ws.Root)
	}
}

func TestFindBuildFiles(t *testing.T) {
	tmp := t.TempDir()

	// Create workspace marker
	os.WriteFile(filepath.Join(tmp, "MODULE.bazel"), []byte(""), 0o644)

	// Create BUILD files
	dirs := []string{"", "proto", filepath.Join("proto", "user")}
	for _, d := range dirs {
		dir := filepath.Join(tmp, d)
		os.MkdirAll(dir, 0o755)
		os.WriteFile(filepath.Join(dir, "BUILD.bazel"), []byte("# test"), 0o644)
	}

	// Create a .git dir that should be skipped
	os.MkdirAll(filepath.Join(tmp, ".git"), 0o755)
	os.WriteFile(filepath.Join(tmp, ".git", "BUILD"), []byte("# skip"), 0o644)

	files, err := FindBuildFiles(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("expected 3 BUILD files, got %d: %v", len(files), files)
	}
}

func TestParseBuildContent_ProtoLibrary(t *testing.T) {
	content := `
load("@rules_proto//proto:defs.bzl", "proto_library")

proto_library(
    name = "user_proto",
    srcs = ["user.proto"],
    deps = [
        "//proto/common:common_proto",
    ],
    visibility = ["//visibility:public"],
)

go_proto_library(
    name = "user_go_proto",
    proto = ":user_proto",
    importpath = "example.com/proto/user",
)
`
	targets := parseBuildContent(content, "//proto/user")
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(targets))
	}

	proto := targets[0]
	if proto.Rule != "proto_library" {
		t.Errorf("expected rule 'proto_library', got %q", proto.Rule)
	}
	if proto.Name != "user_proto" {
		t.Errorf("expected name 'user_proto', got %q", proto.Name)
	}
	if len(proto.Srcs) != 1 || proto.Srcs[0] != "user.proto" {
		t.Errorf("expected srcs ['user.proto'], got %v", proto.Srcs)
	}
	if len(proto.Deps) != 1 || proto.Deps[0] != "//proto/common:common_proto" {
		t.Errorf("expected deps ['//proto/common:common_proto'], got %v", proto.Deps)
	}

	goProto := targets[1]
	if goProto.Rule != "go_proto_library" {
		t.Errorf("expected rule 'go_proto_library', got %q", goProto.Rule)
	}
}

// TestParseBuildContent_Filegroup tests parsing filegroup with proto glob patterns.
func TestParseBuildContent_Filegroup(t *testing.T) {
	content := `
load("@rules_python//python:defs.bzl", "py_binary", "py_library")

filegroup(
    name = "proto_files",
    srcs = glob(["proto/**/*.proto"]),
    visibility = ["//visibility:public"],
)

alias(
    name = "my_service",
    actual = "//services/my_service:my_service",
    visibility = ["//visibility:public"],
)
`
	targets := parseBuildContent(content, "//")
	if len(targets) < 1 {
		t.Fatalf("expected at least 1 target, got %d", len(targets))
	}

	var fg *BazelTarget
	for i, tgt := range targets {
		if tgt.Rule == "filegroup" && tgt.Name == "proto_files" {
			fg = &targets[i]
			break
		}
	}
	if fg == nil {
		t.Fatal("expected to find filegroup 'proto_files'")
	}

	if len(fg.GlobPatterns) != 1 {
		t.Fatalf("expected 1 glob pattern, got %d: %v", len(fg.GlobPatterns), fg.GlobPatterns)
	}
	if fg.GlobPatterns[0] != "proto/**/*.proto" {
		t.Errorf("expected glob 'proto/**/*.proto', got %q", fg.GlobPatterns[0])
	}

	// IsProtoSource should return true
	if !fg.IsProtoSource() {
		t.Error("expected filegroup with proto glob to be a proto source")
	}
}

// TestFilterProtoTargets_WithFilegroup tests that filegroup with proto globs is included.
func TestFilterProtoTargets_WithFilegroup(t *testing.T) {
	targets := []BazelTarget{
		{Rule: "proto_library", Name: "a_proto"},
		{Rule: "filegroup", Name: "proto_files", GlobPatterns: []string{"proto/**/*.proto"}},
		{Rule: "py_binary", Name: "my_app"},
		{Rule: "filegroup", Name: "other_files", GlobPatterns: []string{"data/**/*.csv"}},
		{Rule: "cc_library", Name: "util"},
	}

	filtered := FilterProtoTargets(targets)
	if len(filtered) != 2 {
		t.Errorf("expected 2 proto targets (proto_library + filegroup), got %d", len(filtered))
	}

	foundProtoLib := false
	foundFilegroup := false
	for _, f := range filtered {
		if f.Rule == "proto_library" {
			foundProtoLib = true
		}
		if f.Rule == "filegroup" && f.Name == "proto_files" {
			foundFilegroup = true
		}
	}
	if !foundProtoLib {
		t.Error("expected proto_library in filtered results")
	}
	if !foundFilegroup {
		t.Error("expected proto filegroup in filtered results")
	}
}

func TestFilterProtoTargets(t *testing.T) {
	targets := []BazelTarget{
		{Rule: "proto_library", Name: "a_proto"},
		{Rule: "go_proto_library", Name: "a_go_proto"},
		{Rule: "proto_library", Name: "b_proto"},
		{Rule: "cc_library", Name: "util"},
	}

	filtered := FilterProtoTargets(targets)
	if len(filtered) != 2 {
		t.Errorf("expected 2 proto targets, got %d", len(filtered))
	}
}

func TestResolveProtoFiles(t *testing.T) {
	ws := &Workspace{Root: "/workspace"}
	target := BazelTarget{
		Package: "//proto/user",
		Srcs:    []string{"user.proto", "types.proto"},
	}

	files := ResolveProtoFiles(ws, target)
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	if files[0] != "proto/user/user.proto" {
		t.Errorf("expected 'proto/user/user.proto', got %q", files[0])
	}
}

// TestResolveProtoFiles_WithGlob tests resolving filegroup with glob patterns.
func TestResolveProtoFiles_WithGlob(t *testing.T) {
	tmp := t.TempDir()

	// Create proto directory with files
	protoDir := filepath.Join(tmp, "proto", "v1")
	os.MkdirAll(protoDir, 0o755)
	os.WriteFile(filepath.Join(protoDir, "service.proto"), []byte("syntax = \"proto3\";"), 0o644)
	os.WriteFile(filepath.Join(protoDir, "types.proto"), []byte("syntax = \"proto3\";"), 0o644)
	os.WriteFile(filepath.Join(protoDir, "README.md"), []byte("# docs"), 0o644) // should be skipped

	ws := &Workspace{Root: tmp}
	target := BazelTarget{
		Rule:         "filegroup",
		Package:      "//",
		Name:         "proto_files",
		GlobPatterns: []string{"proto/**/*.proto"},
	}

	files := ResolveProtoFiles(ws, target)
	if len(files) != 2 {
		t.Fatalf("expected 2 proto files from glob, got %d: %v", len(files), files)
	}
}

// TestDetectSyncMode tests auto-detection of sync mode.
func TestDetectSyncMode(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "MODULE.bazel"), []byte(`module(name = "test")`), 0o644)

	ws, _ := DetectWorkspace(tmp)
	integrator := NewIntegratorFromWorkspace(ws)

	// Only filegroup targets → filegroup mode
	filegroupTargets := []BazelTarget{
		{Rule: "filegroup", Name: "protos", GlobPatterns: []string{"**/*.proto"}},
	}
	mode := integrator.detectSyncMode(filegroupTargets)
	if mode != SyncModeFilegroup {
		t.Errorf("expected filegroup mode, got %q", mode)
	}

	// proto_library targets → proto_library mode
	protoLibTargets := []BazelTarget{
		{Rule: "proto_library", Name: "user_proto"},
		{Rule: "filegroup", Name: "protos", GlobPatterns: []string{"**/*.proto"}},
	}
	mode = integrator.detectSyncMode(protoLibTargets)
	if mode != SyncModeProtoLibrary {
		t.Errorf("expected proto_library mode, got %q", mode)
	}

	// No proto targets at all → filegroup mode (default)
	mode = integrator.detectSyncMode(nil)
	if mode != SyncModeFilegroup {
		t.Errorf("expected filegroup mode (default), got %q", mode)
	}
}

// TestShouldPreserve tests that existing non-Buffalo BUILD files are not overwritten.
func TestShouldPreserve(t *testing.T) {
	tmp := t.TempDir()
	gen := NewGenerator(tmp, "test")

	// User-written BUILD file — should be preserved
	userBuild := filepath.Join(tmp, "user_BUILD.bazel")
	os.WriteFile(userBuild, []byte(`filegroup(
    name = "custom",
    srcs = ["foo.rs"],
)`), 0o644)

	if !gen.shouldPreserve(userBuild) {
		t.Error("expected user-written BUILD to be preserved")
	}

	// Buffalo-generated BUILD file — should NOT be preserved
	buffaloBuild := filepath.Join(tmp, "buffalo_BUILD.bazel")
	os.WriteFile(buffaloBuild, []byte(buffaloMarker+"\n# Language: rust\n"), 0o644)

	if gen.shouldPreserve(buffaloBuild) {
		t.Error("expected Buffalo-generated BUILD to NOT be preserved")
	}

	// Non-existent file — should NOT be preserved (safe to create)
	if gen.shouldPreserve(filepath.Join(tmp, "nonexistent")) {
		t.Error("expected non-existent file to not be preserved")
	}
}

// TestGenerateFilegroupBuilds tests generating filegroup BUILD files for generated code.
func TestGenerateFilegroupBuilds(t *testing.T) {
	tmp := t.TempDir()
	gen := NewGenerator(tmp, "test")

	// Create generated Rust files
	rustDir := filepath.Join(tmp, "gen", "rust")
	os.MkdirAll(rustDir, 0o755)
	os.WriteFile(filepath.Join(rustDir, "service.rs"), []byte("// gen"), 0o644)
	os.WriteFile(filepath.Join(rustDir, "types.rs"), []byte("// gen"), 0o644)
	os.WriteFile(filepath.Join(rustDir, "descriptor.bin"), []byte{0x00}, 0o644)

	// Create generated Python files
	pyDir := filepath.Join(tmp, "gen", "python")
	os.MkdirAll(pyDir, 0o755)
	os.WriteFile(filepath.Join(pyDir, "service_pb2.py"), []byte("# gen"), 0o644)
	os.WriteFile(filepath.Join(pyDir, "__init__.py"), []byte(""), 0o644)

	builds, err := gen.GenerateFilegroupBuilds([]string{"rust", "python"}, filepath.Join(tmp, "gen"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(builds) < 2 {
		t.Fatalf("expected at least 2 BUILD files, got %d", len(builds))
	}

	// Check Rust BUILD has filegroup
	var rustBuild *GeneratedBuild
	var pyBuild *GeneratedBuild
	for i, b := range builds {
		if contains(b.Path, "rust") {
			rustBuild = &builds[i]
		}
		if contains(b.Path, "python") {
			pyBuild = &builds[i]
		}
	}

	if rustBuild == nil {
		t.Fatal("expected Rust BUILD file")
	}
	if !contains(rustBuild.Content, "filegroup") {
		t.Error("expected Rust BUILD to contain filegroup")
	}
	if !contains(rustBuild.Content, buffaloMarker) {
		t.Error("expected Rust BUILD to contain Buffalo marker")
	}

	if pyBuild == nil {
		t.Fatal("expected Python BUILD file")
	}
	if !contains(pyBuild.Content, "py_library") {
		t.Error("expected Python BUILD to contain py_library")
	}
}

// TestGenerateFilegroupBuilds_PreserveExisting tests that existing custom BUILD is not overwritten.
func TestGenerateFilegroupBuilds_PreserveExisting(t *testing.T) {
	tmp := t.TempDir()
	gen := NewGenerator(tmp, "test")

	// Create generated Rust files
	rustDir := filepath.Join(tmp, "gen", "rust")
	os.MkdirAll(rustDir, 0o755)
	os.WriteFile(filepath.Join(rustDir, "service.rs"), []byte("// gen"), 0o644)

	// Create an existing custom BUILD file (should be preserved)
	customBuild := filepath.Join(rustDir, "BUILD.bazel")
	customContent := `filegroup(
    name = "nats_orchestrator_proto_files",
    srcs = ["proxy.v1.rs", "google.protobuf.rs"],
    visibility = ["//visibility:public"],
)
`
	os.WriteFile(customBuild, []byte(customContent), 0o644)

	builds, err := gen.GenerateFilegroupBuilds([]string{"rust"}, filepath.Join(tmp, "gen"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should produce no BUILD for Rust since existing non-Buffalo BUILD exists
	for _, b := range builds {
		if contains(b.Path, "rust") {
			t.Error("expected Rust BUILD to be skipped (preserve existing)")
		}
	}

	// Verify original file is untouched
	data, _ := os.ReadFile(customBuild)
	if string(data) != customContent {
		t.Error("existing custom BUILD file was modified")
	}
}

func TestGenerator_GenerateBuildFiles(t *testing.T) {
	gen := NewGenerator("/workspace", "example.com/myproject")

	targets := []BazelTarget{
		{
			Rule:    "proto_library",
			Package: "//proto/user",
			Name:    "user_proto",
			Srcs:    []string{"user.proto"},
			Deps:    []string{"//proto/common:common_proto"},
		},
	}

	builds, err := gen.GenerateBuildFiles(targets, []string{"go", "python"}, "/workspace/generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(builds) != 2 {
		t.Fatalf("expected 2 BUILD files (one per language), got %d", len(builds))
	}

	// Check Go build
	var goBuild *GeneratedBuild
	for i, b := range builds {
		if filepath.Base(filepath.Dir(b.Path)) != "user" {
			continue
		}
		if len(b.Bindings) > 0 && b.Bindings[0].Language == "go" {
			goBuild = &builds[i]
		}
	}
	if goBuild == nil {
		t.Fatal("expected Go BUILD file")
	}
	if len(goBuild.Bindings) != 1 {
		t.Fatalf("expected 1 binding, got %d", len(goBuild.Bindings))
	}
	binding := goBuild.Bindings[0]
	if binding.TargetName != "user_go_proto" {
		t.Errorf("expected target name 'user_go_proto', got %q", binding.TargetName)
	}
	if binding.Importpath != "example.com/myproject/proto/user" {
		t.Errorf("expected importpath 'example.com/myproject/proto/user', got %q", binding.Importpath)
	}
}

func TestParseModuleName(t *testing.T) {
	tests := []struct {
		content  string
		expected string
	}{
		{`module(name = "mymod", version = "1.0")`, "mymod"},
		{"module(\n    name = \"multi\",\n    version = \"2.0\",\n)", "multi"},
		{`workspace(name = "notmod")`, ""},
		{``, ""},
	}

	for _, tt := range tests {
		got := parseModuleName(tt.content)
		if got != tt.expected {
			t.Errorf("parseModuleName(%q) = %q, want %q", tt.content[:min(40, len(tt.content))], got, tt.expected)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestSplitLabel(t *testing.T) {
	tests := []struct {
		label string
		pkg   string
		name  string
	}{
		{"//proto/user:user_proto", "//proto/user", "user_proto"},
		{"//proto/user", "//proto/user", "user"},
	}

	for _, tt := range tests {
		pkg, name := splitLabel(tt.label)
		if pkg != tt.pkg || name != tt.name {
			t.Errorf("splitLabel(%q) = (%q, %q), want (%q, %q)", tt.label, pkg, name, tt.pkg, tt.name)
		}
	}
}

// TestIsProtoSource tests the IsProtoSource method on BazelTarget.
func TestIsProtoSource(t *testing.T) {
	tests := []struct {
		name     string
		target   BazelTarget
		expected bool
	}{
		{"proto_library", BazelTarget{Rule: "proto_library"}, true},
		{"filegroup with proto glob", BazelTarget{Rule: "filegroup", GlobPatterns: []string{"**/*.proto"}}, true},
		{"filegroup with proto srcs", BazelTarget{Rule: "filegroup", Srcs: []string{"user.proto"}}, true},
		{"filegroup without proto", BazelTarget{Rule: "filegroup", GlobPatterns: []string{"**/*.csv"}}, false},
		{"py_binary", BazelTarget{Rule: "py_binary"}, false},
		{"cc_library", BazelTarget{Rule: "cc_library"}, false},
		{"filegroup no srcs", BazelTarget{Rule: "filegroup"}, false},
	}

	for _, tt := range tests {
		got := tt.target.IsProtoSource()
		if got != tt.expected {
			t.Errorf("IsProtoSource(%q) = %v, want %v", tt.name, got, tt.expected)
		}
	}
}

// TestParseGlobPatterns tests extracting glob patterns from BUILD attributes.
func TestParseGlobPatterns(t *testing.T) {
	tests := []struct {
		body     string
		expected []string
	}{
		{`    srcs = glob(["proto/**/*.proto"]),`, []string{"proto/**/*.proto"}},
		{`    srcs = glob(["*.proto", "**/*.proto"]),`, []string{"*.proto", "**/*.proto"}},
		{`    srcs = ["explicit.proto"],`, nil},
		{`    srcs = glob(["src/**/*.rs"]),`, []string{"src/**/*.rs"}},
	}

	for _, tt := range tests {
		got := parseGlobPatterns(tt.body, "srcs")
		if len(got) != len(tt.expected) {
			t.Errorf("parseGlobPatterns(%q) = %v, want %v", tt.body[:min(40, len(tt.body))], got, tt.expected)
			continue
		}
		for i := range got {
			if got[i] != tt.expected[i] {
				t.Errorf("parseGlobPatterns[%d] = %q, want %q", i, got[i], tt.expected[i])
			}
		}
	}
}

// TestReadExistingBuild tests reading and parsing an existing BUILD file.
func TestReadExistingBuild(t *testing.T) {
	tmp := t.TempDir()
	buildPath := filepath.Join(tmp, "BUILD.bazel")

	content := `filegroup(
    name = "proto_files",
    srcs = ["a.rs", "b.rs"],
    visibility = ["//visibility:public"],
)
`
	os.WriteFile(buildPath, []byte(content), 0o644)

	info, err := ReadExistingBuild(buildPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.HasBuffaloMarker {
		t.Error("expected no Buffalo marker in custom BUILD")
	}
	if len(info.Targets) == 0 {
		t.Error("expected to parse targets from BUILD")
	}

	// Now test with Buffalo marker
	buffaloPath := filepath.Join(tmp, "BUFFALO_BUILD.bazel")
	os.WriteFile(buffaloPath, []byte(buffaloMarker+"\n"+content), 0o644)

	info2, _ := ReadExistingBuild(buffaloPath)
	if !info2.HasBuffaloMarker {
		t.Error("expected Buffalo marker in generated BUILD")
	}
}
