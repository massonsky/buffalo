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

	comp := New(logger.New(), nil)
	_, err := comp.Compile(context.Background(), []compiler.ProtoFile{{Path: filepath.Join("araviec", "test.proto")}}, compiler.CompileOptions{})
	if err == nil {
		t.Fatal("expected missing Cargo integration to fail")
	}
	if !strings.Contains(err.Error(), "prost generator requires manual Cargo setup") {
		t.Fatalf("expected helpful prost error, got: %v", err)
	}
}
