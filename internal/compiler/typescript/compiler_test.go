package typescript

import (
	"os"
	"path/filepath"
	"strings"
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
			if tool == "protoc-gen-ts_proto" {
				hasPlugin = true
			}
		}

		if !hasProtoc {
			t.Error("Expected protoc in required tools")
		}
		if !hasPlugin {
			t.Error("Expected protoc-gen-ts_proto in required tools")
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

	if opts.Generator != "ts-proto" {
		t.Errorf("Expected default generator 'ts-proto', got '%s'", opts.Generator)
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

	t.Run("ts-proto-default", func(t *testing.T) {
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

func TestCompiler_GenerateIndexFile_NamespaceExports(t *testing.T) {
	log := logger.New()
	c := New(log, &Options{OutputIndex: true})

	outputDir := t.TempDir()
	files := []string{
		filepath.Join(outputDir, "araviec", "ais", "v1", "ais_service.ts"),
		filepath.Join(outputDir, "google", "rpc", "status.ts"),
	}

	if err := c.GenerateIndexFile(outputDir, files); err != nil {
		t.Fatalf("GenerateIndexFile failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "index.ts"))
	if err != nil {
		t.Fatalf("read index.ts failed: %v", err)
	}

	s := string(content)
	if strings.Contains(s, "export * from '") {
		t.Fatalf("unexpected wildcard re-export in index.ts: %s", s)
	}
	if !strings.Contains(s, "export * as araviec_ais_v1_ais_service from './araviec/ais/v1/ais_service';") {
		t.Fatalf("expected namespaced export for ais_service, got: %s", s)
	}
	if !strings.Contains(s, "export * as google_rpc_status from './google/rpc/status';") {
		t.Fatalf("expected namespaced export for google/rpc/status, got: %s", s)
	}
}

func TestModuleAlias(t *testing.T) {
	if got := moduleAlias("araviec/ais/v1/ais-service"); got != "araviec_ais_v1_ais_service" {
		t.Fatalf("unexpected alias: %s", got)
	}
	if got := moduleAlias("123/pkg"); got != "m_123_pkg" {
		t.Fatalf("unexpected numeric-prefixed alias: %s", got)
	}
}
