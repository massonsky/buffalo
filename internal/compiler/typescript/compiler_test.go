package typescript

import (
	"path/filepath"
	"testing"

	"github.com/massonsky/buffalo/internal/compiler"
	"github.com/massonsky/buffalo/pkg/logger"
)

func TestCompiler_Name(t *testing.T) {
	log := logger.New()
	c := New(log, nil)

	if c.Name() != "typescript" {
		t.Errorf("Expected name 'typescript', got '%s'", c.Name())
	}
}

func TestCompiler_RequiredTools(t *testing.T) {
	log := logger.New()

	t.Run("default", func(t *testing.T) {
		opts := &Options{GenerateGrpcWeb: false}
		c := New(log, opts)

		tools := c.RequiredTools()
		if len(tools) != 2 {
			t.Errorf("Expected 2 tools, got %d", len(tools))
		}

		hasProtoc := false
		hasPlugin := false
		for _, tool := range tools {
			if tool == "protoc" {
				hasProtoc = true
			}
			if tool == "protoc-gen-ts" {
				hasPlugin = true
			}
		}

		if !hasProtoc {
			t.Error("Expected protoc in required tools")
		}
		if !hasPlugin {
			t.Error("Expected protoc-gen-ts in required tools")
		}
	})

	t.Run("with_grpc_web", func(t *testing.T) {
		opts := &Options{GenerateGrpcWeb: true}
		c := New(log, opts)

		tools := c.RequiredTools()
		if len(tools) != 3 {
			t.Errorf("Expected 3 tools, got %d", len(tools))
		}

		hasGrpcWeb := false
		for _, tool := range tools {
			if tool == "protoc-gen-grpc-web" {
				hasGrpcWeb = true
			}
		}

		if !hasGrpcWeb {
			t.Error("Expected protoc-gen-grpc-web in required tools")
		}
	})
}

func TestCompiler_DefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.ProtocPath != "protoc" {
		t.Errorf("Expected default protoc path 'protoc', got '%s'", opts.ProtocPath)
	}

	if opts.Generator != "protoc-gen-ts" {
		t.Errorf("Expected default generator 'protoc-gen-ts', got '%s'", opts.Generator)
	}

	if !opts.GenerateGrpc {
		t.Error("Expected GenerateGrpc to be true by default")
	}

	if !opts.ESModules {
		t.Error("Expected ESModules to be true by default")
	}

	if !opts.OutputIndex {
		t.Error("Expected OutputIndex to be true by default")
	}
}

func TestCompiler_GetOutputPath(t *testing.T) {
	log := logger.New()

	t.Run("protoc-gen-ts", func(t *testing.T) {
		c := New(log, &Options{Generator: "protoc-gen-ts"})

		file := compiler.ProtoFile{
			Path:    "/path/to/test.proto",
			Package: "test",
		}

		opts := compiler.CompileOptions{
			OutputDir: "/output",
		}

		outputPath := c.GetOutputPath(file, opts)
		expected := filepath.Join("/output", "test_pb.ts")

		if outputPath != expected {
			t.Errorf("Expected output path '%s', got '%s'", expected, outputPath)
		}
	})

	t.Run("ts-proto", func(t *testing.T) {
		c := New(log, &Options{Generator: "ts-proto"})

		file := compiler.ProtoFile{
			Path:    "/path/to/test.proto",
			Package: "test",
		}

		opts := compiler.CompileOptions{
			OutputDir: "/output",
		}

		outputPath := c.GetOutputPath(file, opts)
		expected := filepath.Join("/output", "test.ts")

		if outputPath != expected {
			t.Errorf("Expected output path '%s', got '%s'", expected, outputPath)
		}
	})
}
