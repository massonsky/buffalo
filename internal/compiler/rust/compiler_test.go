package rust

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/massonsky/buffalo/internal/compiler"
	"github.com/massonsky/buffalo/pkg/logger"
)

func TestCompileProstAcceptsExistingCargoIntegration(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	cargoToml := `[package]
name = "demo"
version = "0.1.0"
edition = "2021"

[build-dependencies]
prost-build = "0.14"
`
	buildRs := `fn main() -> Result<(), Box<dyn std::error::Error>> {
    prost_build::Config::new()
        .compile_protos(&["araviec/test.proto"], &["."])?;
    Ok(())
}`
	protoContent := `syntax = "proto3";
package demo.v1;
message Ping { string id = 1; }`

	if err := os.WriteFile(filepath.Join(tempDir, "Cargo.toml"), []byte(cargoToml), 0644); err != nil {
		t.Fatalf("failed to write Cargo.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "build.rs"), []byte(buildRs), 0644); err != nil {
		t.Fatalf("failed to write build.rs: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tempDir, "araviec"), 0755); err != nil {
		t.Fatalf("failed to create proto directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "araviec", "test.proto"), []byte(protoContent), 0644); err != nil {
		t.Fatalf("failed to write proto file: %v", err)
	}

	comp := New(logger.New(), nil)
	result, err := comp.Compile(context.Background(), []compiler.ProtoFile{{Path: filepath.Join("araviec", "test.proto")}}, compiler.CompileOptions{})
	if err != nil {
		t.Fatalf("expected prost compile validation to succeed, got error: %v", err)
	}
	if !result.Success {
		t.Fatal("expected compile result success=true")
	}
}

func TestCompileProstFailsWithoutCargoIntegration(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	if err := os.MkdirAll(filepath.Join(tempDir, "araviec"), 0755); err != nil {
		t.Fatalf("failed to create proto directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "araviec", "test.proto"), []byte("syntax = \"proto3\"; package demo.v1;"), 0644); err != nil {
		t.Fatalf("failed to write proto file: %v", err)
	}

	// Without a Cargo project alongside the protos, buffalo no longer treats
	// missing Cargo integration as a hard error: it falls back to direct
	// generation via `protoc --prost_out=` (the hermetic Bazel sandbox path).
	// In the test environment protoc itself isn't available, so the call is
	// expected to fail — but the error must come from protoc, NOT from the
	// Cargo-integration validator.
	comp := New(logger.New(), &Options{
		ProtocPath:   "protoc",
		Generator:    "prost",
		GenerateGrpc: false,
	})
	_, err := comp.Compile(context.Background(), []compiler.ProtoFile{{Path: filepath.Join("araviec", "test.proto")}}, compiler.CompileOptions{OutputDir: filepath.Join(tempDir, "out")})
	if err == nil {
		// protoc happens to be on PATH — that's fine, Compile succeeded.
		return
	}
	if strings.Contains(err.Error(), "prost generator requires manual Cargo setup") {
		t.Fatalf("Cargo-integration check should be a soft warning, not a hard error; got: %v", err)
	}
}
