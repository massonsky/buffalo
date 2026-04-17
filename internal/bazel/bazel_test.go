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
		label   string
		pkg     string
		name    string
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
