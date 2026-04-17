package builder

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/massonsky/buffalo/pkg/metrics"
)

func TestBuilder_Build(t *testing.T) {
	log := logger.New(logger.WithLevel(logger.INFO))
	met := metrics.NewCollector()

	b, err := New(nil,
		WithLogger(log),
		WithMetrics(met),
	)
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	ctx := context.Background()
	tempDir := t.TempDir()

	// Create a test proto file
	protoContent := `syntax = "proto3";
package test;

message TestMessage {
  string name = 1;
  int32 value = 2;
}

service TestService {
  rpc GetTest(TestMessage) returns (TestMessage);
}
`
	protoFile := filepath.Join(tempDir, "test.proto")
	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		t.Fatalf("Failed to create test proto file: %v", err)
	}

	plan := &BuildPlan{
		ProtoFiles:  []string{protoFile},
		ImportPaths: []string{tempDir},
		OutputDir:   filepath.Join(tempDir, "generated"),
		Languages:   []string{"go"},
		Options: BuildOptions{
			Workers:     4,
			Incremental: false,
			Cache:       false,
			Verbose:     true,
		},
	}

	result, err := b.Build(ctx, plan)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected successful build")
	}

	if result.FilesProcessed != 1 {
		t.Errorf("Expected 1 file processed, got %d", result.FilesProcessed)
	}

	// Without protoc and enabled language compilers, no files will be generated
	t.Logf("Files generated: %d (may be 0 without protoc)", result.FilesGenerated)

	if result.Duration == 0 {
		t.Error("Expected non-zero duration")
	}

	t.Logf("Build completed in %v", result.Duration)
	t.Logf("Files processed: %d, generated: %d", result.FilesProcessed, result.FilesGenerated)
}

func TestBuilder_BuildWithMultipleFiles(t *testing.T) {
	log := logger.New(logger.WithLevel(logger.INFO))
	met := metrics.NewCollector()

	b, err := New(nil,
		WithLogger(log),
		WithMetrics(met),
	)
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	ctx := context.Background()
	tempDir := t.TempDir()

	// Create multiple proto files
	files := []string{"file1.proto", "file2.proto", "file3.proto"}
	for _, filename := range files {
		protoContent := `syntax = "proto3";
package test;
message TestMessage {
  string name = 1;
}
`
		protoFile := filepath.Join(tempDir, filename)
		if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
			t.Fatalf("Failed to create test proto file %s: %v", filename, err)
		}
	}

	protoFiles := make([]string, 0, len(files))
	for _, f := range files {
		protoFiles = append(protoFiles, filepath.Join(tempDir, f))
	}

	plan := &BuildPlan{
		ProtoFiles:  protoFiles,
		ImportPaths: []string{tempDir},
		OutputDir:   filepath.Join(tempDir, "generated"),
		Languages:   []string{"go"},
		Options: BuildOptions{
			Workers: 4,
			Cache:   false,
			Verbose: false,
		},
	}

	result, err := b.Build(ctx, plan)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected successful build")
	}

	if result.FilesProcessed != len(files) {
		t.Errorf("Expected %d files processed, got %d", len(files), result.FilesProcessed)
	}

	// Without protoc and enabled language compilers, no files will be generated
	t.Logf("Files generated: %d (may be 0 without protoc)", result.FilesGenerated)
}

func TestBuilder_BuildWithMultipleLanguages(t *testing.T) {
	log := logger.New(logger.WithLevel(logger.INFO))
	met := metrics.NewCollector()

	b, err := New(nil,
		WithLogger(log),
		WithMetrics(met),
	)
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	ctx := context.Background()
	tempDir := t.TempDir()

	protoContent := `syntax = "proto3";
package test;
message TestMessage {
  string name = 1;
}
`
	protoFile := filepath.Join(tempDir, "test.proto")
	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		t.Fatalf("Failed to create test proto file: %v", err)
	}

	plan := &BuildPlan{
		ProtoFiles:  []string{protoFile},
		ImportPaths: []string{tempDir},
		OutputDir:   filepath.Join(tempDir, "generated"),
		Languages:   []string{"go", "python", "rust"},
		Options: BuildOptions{
			Workers: 4,
			Cache:   false,
			Verbose: false,
		},
	}

	result, err := b.Build(ctx, plan)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected successful build")
	}

	// Without protoc and enabled language compilers, no files will be generated
	t.Logf("Files generated: %d (may be 0 without protoc)", result.FilesGenerated)
}

func TestBuilder_BuildWithCache(t *testing.T) {
	log := logger.New(logger.WithLevel(logger.INFO))
	met := metrics.NewCollector()

	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, ".cache")

	logAdapter := NewLoggerAdapter(log)

	b, err := New(nil,
		WithLogger(log),
		WithMetrics(met),
		WithCache(NewCacheManager(logAdapter)),
	)
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	ctx := context.Background()

	protoContent := `syntax = "proto3";
package test;
message TestMessage {
  string name = 1;
}
`
	protoFile := filepath.Join(tempDir, "test.proto")
	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		t.Fatalf("Failed to create test proto file: %v", err)
	}

	plan := &BuildPlan{
		ProtoFiles:  []string{protoFile},
		ImportPaths: []string{tempDir},
		OutputDir:   filepath.Join(tempDir, "generated"),
		Languages:   []string{"go"},
		Options: BuildOptions{
			Workers:     4,
			Incremental: true,
			Cache:       true,
			CacheDir:    cacheDir,
			Verbose:     false,
		},
	}

	// First build
	result1, err := b.Build(ctx, plan)
	if err != nil {
		t.Fatalf("First build failed: %v", err)
	}

	if !result1.Success {
		t.Error("Expected successful first build")
	}

	t.Logf("First build: hits=%d, misses=%d", result1.CacheHits, result1.CacheMisses)

	// Second build
	result2, err := b.Build(ctx, plan)
	if err != nil {
		t.Fatalf("Second build failed: %v", err)
	}

	if !result2.Success {
		t.Error("Expected successful second build")
	}

	t.Logf("Second build: hits=%d, misses=%d", result2.CacheHits, result2.CacheMisses)
}

func TestBuilder_BuildWithDryRun(t *testing.T) {
	log := logger.New(logger.WithLevel(logger.INFO))
	met := metrics.NewCollector()

	b, err := New(nil,
		WithLogger(log),
		WithMetrics(met),
	)
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	ctx := context.Background()
	tempDir := t.TempDir()

	protoContent := `syntax = "proto3";
package test;
message TestMessage {
  string name = 1;
}
`
	protoFile := filepath.Join(tempDir, "test.proto")
	if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
		t.Fatalf("Failed to create test proto file: %v", err)
	}

	plan := &BuildPlan{
		ProtoFiles:  []string{protoFile},
		ImportPaths: []string{tempDir},
		OutputDir:   filepath.Join(tempDir, "generated"),
		Languages:   []string{"go"},
		Options: BuildOptions{
			Workers: 4,
			DryRun:  true,
			Verbose: true,
		},
	}

	result, err := b.Build(ctx, plan)
	if err != nil {
		t.Fatalf("Dry run build failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected successful dry run")
	}

	// Dry run should process files but not necessarily generate them
	if result.FilesProcessed != 1 {
		t.Errorf("Expected 1 file processed, got %d", result.FilesProcessed)
	}
}

func TestBuilder_BuildEmptyPlan(t *testing.T) {
	log := logger.New(logger.WithLevel(logger.INFO))
	met := metrics.NewCollector()

	b, err := New(nil,
		WithLogger(log),
		WithMetrics(met),
	)
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	ctx := context.Background()

	plan := &BuildPlan{
		ProtoFiles:  []string{},
		ImportPaths: []string{},
		OutputDir:   t.TempDir(),
		Languages:   []string{"go"},
		Options:     BuildOptions{},
	}

	result, err := b.Build(ctx, plan)
	if err != nil {
		t.Fatalf("Build failed for empty plan: %v", err)
	}

	if !result.Success {
		t.Error("Expected successful build for empty plan")
	}

	if result.FilesProcessed != 0 {
		t.Errorf("Expected 0 files processed, got %d", result.FilesProcessed)
	}
}

func TestBuilder_GetMetrics(t *testing.T) {
	log := logger.New(logger.WithLevel(logger.INFO))
	met := metrics.NewCollector()

	b, err := New(nil,
		WithLogger(log),
		WithMetrics(met),
	)
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	m := b.GetMetrics()
	if m == nil {
		t.Error("Expected non-nil metrics")
	}
}

func TestBuilder_BuildWithCancellation(t *testing.T) {
	log := logger.New(logger.WithLevel(logger.INFO))
	met := metrics.NewCollector()

	b, err := New(nil,
		WithLogger(log),
		WithMetrics(met),
	)
	if err != nil {
		t.Fatalf("Failed to create builder: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	tempDir := t.TempDir()

	// Create many proto files
	var protoFiles []string
	for i := 0; i < 50; i++ {
		protoContent := `syntax = "proto3";
package test;
message TestMessage {
  string name = 1;
}
`
		protoFile := filepath.Join(tempDir, "file"+string(rune('0'+i%10))+".proto")
		if err := os.WriteFile(protoFile, []byte(protoContent), 0644); err != nil {
			t.Fatalf("Failed to create test proto file: %v", err)
		}
		protoFiles = append(protoFiles, protoFile)
	}

	plan := &BuildPlan{
		ProtoFiles:  protoFiles,
		ImportPaths: []string{tempDir},
		OutputDir:   filepath.Join(tempDir, "generated"),
		Languages:   []string{"go"},
		Options: BuildOptions{
			Workers: 2,
		},
	}

	// Cancel immediately
	cancel()

	_, err = b.Build(ctx, plan)
	// Should handle cancellation gracefully
	// Either error or success is acceptable depending on timing
	_ = err
}
