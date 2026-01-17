package python

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/massonsky/buffalo/internal/compiler"
	"github.com/massonsky/buffalo/pkg/logger"
)

func TestCompiler_Name(t *testing.T) {
	log := logger.New()
	c := New(log, nil)

	if c.Name() != "python" {
		t.Errorf("Expected name 'python', got '%s'", c.Name())
	}
}

func TestCompiler_RequiredTools(t *testing.T) {
	log := logger.New()

	t.Run("with_grpc", func(t *testing.T) {
		opts := &Options{GenerateGrpc: true}
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
			if tool == "grpc_python_plugin" {
				hasPlugin = true
			}
		}

		if !hasProtoc {
			t.Error("Expected protoc in required tools")
		}
		if !hasPlugin {
			t.Error("Expected grpc_python_plugin in required tools")
		}
	})

	t.Run("without_grpc", func(t *testing.T) {
		opts := &Options{GenerateGrpc: false}
		c := New(log, opts)

		tools := c.RequiredTools()
		if len(tools) != 1 {
			t.Errorf("Expected 1 tool, got %d", len(tools))
		}

		if tools[0] != "protoc" {
			t.Errorf("Expected 'protoc', got '%s'", tools[0])
		}
	})
}

func TestCompiler_DefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.ProtocPath != "protoc" {
		t.Errorf("Expected default protoc path 'protoc', got '%s'", opts.ProtocPath)
	}

	if !opts.GenerateGrpc {
		t.Error("Expected GenerateGrpc to be true by default")
	}

	if !opts.GenerateInit {
		t.Error("Expected GenerateInit to be true by default")
	}
}

func TestCompiler_GetOutputPath(t *testing.T) {
	log := logger.New()
	c := New(log, nil)

	file := compiler.ProtoFile{
		Path:    "/path/to/test.proto",
		Package: "test",
	}

	opts := compiler.CompileOptions{
		OutputDir: "/output",
	}

	outputPath := c.GetOutputPath(file, opts)
	expected := filepath.Join("/output", "test_pb2.py")

	if outputPath != expected {
		t.Errorf("Expected output path '%s', got '%s'", expected, outputPath)
	}
}

func TestCompiler_GenerateInitFiles(t *testing.T) {
	log := logger.New()
	opts := &Options{GenerateInit: true}
	c := New(log, opts)

	tempDir := t.TempDir()

	// Create a subdirectory structure
	subDir := filepath.Join(tempDir, "package")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create a Python file
	pyFile := filepath.Join(subDir, "test_pb2.py")
	if err := os.WriteFile(pyFile, []byte("# test"), 0644); err != nil {
		t.Fatalf("Failed to create Python file: %v", err)
	}

	// Generate __init__.py files
	if err := c.generateInitFiles(tempDir); err != nil {
		t.Fatalf("generateInitFiles failed: %v", err)
	}

	// Check if __init__.py was created
	initFile := filepath.Join(subDir, "__init__.py")
	if _, err := os.Stat(initFile); os.IsNotExist(err) {
		t.Error("__init__.py was not created")
	}

	// Verify content
	content, err := os.ReadFile(initFile)
	if err != nil {
		t.Fatalf("Failed to read __init__.py: %v", err)
	}

	if len(content) == 0 {
		t.Error("__init__.py is empty")
	}
}

func TestCompiler_Compile_DryRun(t *testing.T) {
	// This test doesn't require protoc to be installed
	log := logger.New()
	c := New(log, nil)

	tempDir := t.TempDir()

	// Create a dummy proto file
	protoFile := filepath.Join(tempDir, "test.proto")
	protoContent := `syntax = "proto3";
package test;

message TestMessage {
  string name = 1;
}
`
	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		t.Fatalf("Failed to create proto file: %v", err)
	}

	files := []compiler.ProtoFile{
		{
			Path:        protoFile,
			Package:     "test",
			ImportPaths: []string{tempDir},
		},
	}

	opts := compiler.CompileOptions{
		OutputDir:   filepath.Join(tempDir, "output"),
		ImportPaths: []string{tempDir},
	}

	ctx := context.Background()

	// Note: This will fail if protoc is not installed
	// In a real test environment, you would mock exec.Command
	_, err := c.Compile(ctx, files, opts)

	// We expect this to fail without protoc installed
	// This is just to test the structure
	if err != nil {
		t.Logf("Compilation failed (expected without protoc): %v", err)
	}
}

func TestCompiler_BuildProtocArgs(t *testing.T) {
	log := logger.New()
	c := New(log, nil)

	file := compiler.ProtoFile{
		Path:        "/path/to/test.proto",
		Package:     "test",
		ImportPaths: []string{"/import1", "/import2"},
	}

	opts := compiler.CompileOptions{
		OutputDir:   "/output",
		ImportPaths: []string{"/import3", "/import4"},
	}

	t.Run("python_out", func(t *testing.T) {
		args := c.buildProtocArgs(file, opts, opts.OutputDir, "/path/to", false)

		// Check for import paths
		hasImport1 := false
		hasImport2 := false
		hasImport3 := false
		hasImport4 := false
		hasPythonOut := false
		hasProtoFile := false

		for _, arg := range args {
			if arg == "--proto_path=/import1" {
				hasImport1 = true
			}
			if arg == "--proto_path=/import2" {
				hasImport2 = true
			}
			if arg == "--proto_path=/import3" {
				hasImport3 = true
			}
			if arg == "--proto_path=/import4" {
				hasImport4 = true
			}
			if arg == "--python_out=/output" {
				hasPythonOut = true
			}
			if arg == "/path/to/test.proto" {
				hasProtoFile = true
			}
		}

		if !hasImport3 || !hasImport4 {
			t.Error("Expected import paths from opts")
		}
		if !hasImport1 || !hasImport2 {
			t.Error("Expected import paths from file")
		}
		if !hasPythonOut {
			t.Error("Expected --python_out flag")
		}
		if !hasProtoFile {
			t.Error("Expected proto file path in args")
		}
	})

	t.Run("grpc_python_out", func(t *testing.T) {
		args := c.buildProtocArgs(file, opts, opts.OutputDir, "/path/to", true)

		hasGrpcOut := false
		for _, arg := range args {
			if arg == "--grpc_python_out=/output" {
				hasGrpcOut = true
			}
		}

		if !hasGrpcOut {
			t.Error("Expected --grpc_python_out flag")
		}
	})
}

func TestNew_WithNilOptions(t *testing.T) {
	log := logger.New()
	c := New(log, nil)

	if c.options == nil {
		t.Fatal("Expected options to be initialized")
	}

	if c.options.ProtocPath != "protoc" {
		t.Error("Expected default protoc path")
	}

	if !c.options.GenerateInit {
		t.Error("Expected GenerateInit to be true")
	}
}
