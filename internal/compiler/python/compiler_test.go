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

func TestCompiler_FixFileImports(t *testing.T) {
	log := logger.New()
	c := New(log, nil)

	tempDir := t.TempDir()

	testCases := []struct {
		name         string
		content      string
		modulePrefix string
		expected     string
		shouldModify bool
	}{
		{
			name: "fix_from_import",
			content: `# Generated by protoc
from module1.v1 import service_pb2
from module1.v1 import service_pb2_grpc

def foo():
    pass
`,
			modulePrefix: "api.generated.python",
			expected: `# Generated by protoc
from api.generated.python.module1.v1 import service_pb2
from api.generated.python.module1.v1 import service_pb2_grpc

def foo():
    pass
`,
			shouldModify: true,
		},
		{
			name: "fix_simple_from_import",
			content: `from mypackage import message_pb2
`,
			modulePrefix: "generated.python",
			expected: `from generated.python.mypackage import message_pb2
`,
			shouldModify: true,
		},
		{
			name: "already_has_prefix",
			content: `from api.generated.python.module1.v1 import service_pb2
`,
			modulePrefix: "api.generated.python",
			expected: `from api.generated.python.module1.v1 import service_pb2
`,
			shouldModify: false,
		},
		{
			name: "fix_direct_import",
			content: `import module1.v1_pb2
import module1.v1_pb2_grpc
`,
			modulePrefix: "api.generated.python",
			expected: `import api.generated.python.module1.v1_pb2
import api.generated.python.module1.v1_pb2_grpc
`,
			shouldModify: true,
		},
		{
			name: "skip_standard_imports",
			content: `import os
from typing import Optional
from concurrent import futures
from module1.v1 import service_pb2
`,
			modulePrefix: "api.generated.python",
			expected: `import os
from typing import Optional
from concurrent import futures
from api.generated.python.module1.v1 import service_pb2
`,
			shouldModify: true,
		},
		{
			name: "preserve_indentation",
			content: `class MyClass:
    from module1.v1 import service_pb2
`,
			modulePrefix: "api.generated.python",
			expected: `class MyClass:
    from api.generated.python.module1.v1 import service_pb2
`,
			shouldModify: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create temp file
			pyFile := filepath.Join(tempDir, tc.name+"_pb2.py")
			if err := os.WriteFile(pyFile, []byte(tc.content), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Fix imports
			if err := c.fixFileImports(pyFile, tc.modulePrefix); err != nil {
				t.Fatalf("fixFileImports failed: %v", err)
			}

			// Read result
			result, err := os.ReadFile(pyFile)
			if err != nil {
				t.Fatalf("Failed to read result file: %v", err)
			}

			if string(result) != tc.expected {
				t.Errorf("Unexpected result.\nExpected:\n%s\nGot:\n%s", tc.expected, string(result))
			}
		})
	}
}

func TestCompiler_FixImports_Integration(t *testing.T) {
	log := logger.New()
	c := New(log, nil)

	// Create a temp directory structure that mimics real output
	tempDir := t.TempDir()
	outputDir := filepath.Join(tempDir, "api", "generated", "python")
	moduleDir := filepath.Join(outputDir, "module1", "v1")

	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("Failed to create directory structure: %v", err)
	}

	// Create a _pb2_grpc.py file with relative imports
	grpcContent := `# Generated by protoc-gen-grpc
import grpc
from module1.v1 import service_pb2 as service__pb2

class MyServiceStub:
    pass
`
	grpcFile := filepath.Join(moduleDir, "service_pb2_grpc.py")
	if err := os.WriteFile(grpcFile, []byte(grpcContent), 0644); err != nil {
		t.Fatalf("Failed to create grpc file: %v", err)
	}

	// Create a _pb2.py file (should not need fixing in most cases)
	pb2Content := `# Generated by protoc
# No pb2 imports here
`
	pb2File := filepath.Join(moduleDir, "service_pb2.py")
	if err := os.WriteFile(pb2File, []byte(pb2Content), 0644); err != nil {
		t.Fatalf("Failed to create pb2 file: %v", err)
	}

	// Change to temp dir to simulate working directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Fix imports
	generatedFiles := []string{grpcFile, pb2File}
	if err := c.fixImports(outputDir, generatedFiles); err != nil {
		t.Fatalf("fixImports failed: %v", err)
	}

	// Check that the grpc file was fixed
	result, err := os.ReadFile(grpcFile)
	if err != nil {
		t.Fatalf("Failed to read result file: %v", err)
	}

	expectedImport := "from api.generated.python.module1.v1 import service_pb2 as service__pb2"
	if !contains(string(result), expectedImport) {
		t.Errorf("Expected import to be fixed.\nExpected to contain:\n%s\nGot:\n%s", expectedImport, string(result))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
