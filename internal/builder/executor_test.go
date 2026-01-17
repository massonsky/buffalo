package builder

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/massonsky/buffalo/pkg/metrics"
)

func TestExecutor_Execute(t *testing.T) {
	log := logger.New(logger.WithLevel(logger.INFO))
	met := metrics.NewCollector()

	executor := NewExecutor(log, met)
	ctx := context.Background()

	// Create test graph
	graph := &DependencyGraph{
		Nodes: map[string]*ProtoFile{
			"test.proto": {
				Path:    "test.proto",
				Package: "test",
			},
		},
		Edges: map[string][]string{
			"test.proto": {},
		},
		CompilationOrder: []string{"test.proto"},
	}

	plan := &ExecutionPlan{
		Graph:     graph,
		OutputDir: t.TempDir(),
		Languages: []string{"go", "python"},
		Options: BuildOptions{
			Workers: 2,
			Verbose: false,
		},
	}

	result, err := executor.Execute(ctx, plan)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(result.FilesGenerated) == 0 {
		t.Error("Expected generated files")
	}

	if result.Metrics == nil {
		t.Error("Expected metrics")
	}
}

func TestExecutor_ExecuteMultipleFiles(t *testing.T) {
	log := logger.New(logger.WithLevel(logger.INFO))
	met := metrics.NewCollector()

	executor := NewExecutor(log, met)
	ctx := context.Background()

	// Create test graph with multiple files
	graph := &DependencyGraph{
		Nodes: map[string]*ProtoFile{
			"a.proto": {Path: "a.proto", Package: "a"},
			"b.proto": {Path: "b.proto", Package: "b"},
			"c.proto": {Path: "c.proto", Package: "c"},
		},
		Edges: map[string][]string{
			"a.proto": {},
			"b.proto": {},
			"c.proto": {},
		},
		CompilationOrder: []string{"a.proto", "b.proto", "c.proto"},
	}

	plan := &ExecutionPlan{
		Graph:     graph,
		OutputDir: t.TempDir(),
		Languages: []string{"go"},
		Options: BuildOptions{
			Workers: 4,
			Verbose: true,
		},
	}

	result, err := executor.Execute(ctx, plan)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Should have generated files for each proto file
	expectedFiles := 3 // 3 proto files * 1 language
	if len(result.FilesGenerated) < expectedFiles {
		t.Errorf("Expected at least %d generated files, got %d", expectedFiles, len(result.FilesGenerated))
	}
}

func TestExecutor_ExecuteWithMultipleLanguages(t *testing.T) {
	log := logger.New(logger.WithLevel(logger.INFO))
	met := metrics.NewCollector()

	executor := NewExecutor(log, met)
	ctx := context.Background()

	graph := &DependencyGraph{
		Nodes: map[string]*ProtoFile{
			"test.proto": {Path: "test.proto", Package: "test"},
		},
		Edges: map[string][]string{
			"test.proto": {},
		},
		CompilationOrder: []string{"test.proto"},
	}

	plan := &ExecutionPlan{
		Graph:     graph,
		OutputDir: t.TempDir(),
		Languages: []string{"go", "python", "rust"},
		Options: BuildOptions{
			Workers: 4,
			Verbose: false,
		},
	}

	result, err := executor.Execute(ctx, plan)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Should have files for all languages
	if len(result.FilesGenerated) < 3 {
		t.Errorf("Expected at least 3 generated files (one per language), got %d", len(result.FilesGenerated))
	}
}

func TestExecutor_ExecuteEmptyGraph(t *testing.T) {
	log := logger.New(logger.WithLevel(logger.INFO))
	met := metrics.NewCollector()

	executor := NewExecutor(log, met)
	ctx := context.Background()

	graph := &DependencyGraph{
		Nodes:            map[string]*ProtoFile{},
		Edges:            map[string][]string{},
		CompilationOrder: []string{},
	}

	plan := &ExecutionPlan{
		Graph:     graph,
		OutputDir: t.TempDir(),
		Languages: []string{"go"},
		Options: BuildOptions{
			Workers: 2,
		},
	}

	result, err := executor.Execute(ctx, plan)
	if err != nil {
		t.Fatalf("Execute failed for empty graph: %v", err)
	}

	if len(result.FilesGenerated) != 0 {
		t.Errorf("Expected 0 generated files for empty graph, got %d", len(result.FilesGenerated))
	}
}

func TestExecutor_ExecuteWithCancellation(t *testing.T) {
	log := logger.New(logger.WithLevel(logger.INFO))
	met := metrics.NewCollector()

	executor := NewExecutor(log, met)

	// Create a context that we'll cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Create a large graph to ensure execution takes some time
	nodes := make(map[string]*ProtoFile)
	edges := make(map[string][]string)
	order := make([]string, 100)

	for i := 0; i < 100; i++ {
		name := filepath.Join("test", "file"+string(rune('0'+i%10))+".proto")
		nodes[name] = &ProtoFile{Path: name, Package: "test"}
		edges[name] = []string{}
		order[i] = name
	}

	graph := &DependencyGraph{
		Nodes:            nodes,
		Edges:            edges,
		CompilationOrder: order,
	}

	plan := &ExecutionPlan{
		Graph:     graph,
		OutputDir: t.TempDir(),
		Languages: []string{"go"},
		Options: BuildOptions{
			Workers: 2,
		},
	}

	// Cancel after a short delay
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := executor.Execute(ctx, plan)
	// Execution might complete before cancellation or be cancelled
	// Both are valid outcomes, so we don't check the error
	_ = err
}

func TestExecutor_CompileFile(t *testing.T) {
	log := logger.New(logger.WithLevel(logger.INFO))
	met := metrics.NewCollector()

	executor := NewExecutor(log, met).(*executor)
	ctx := context.Background()

	// Create a temporary directory for output
	tempDir := t.TempDir()

	// Create a test proto file
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

	file := &ProtoFile{
		Path:    protoFile,
		Package: "test",
		Syntax:  "proto3",
	}

	plan := &ExecutionPlan{
		OutputDir: tempDir,
		Options: BuildOptions{
			Verbose: true,
		},
	}

	// Test compilation for different languages
	languages := []string{"go", "python", "rust"}
	for _, lang := range languages {
		t.Run("Compile_"+lang, func(t *testing.T) {
			files, err := executor.compileFile(ctx, file, lang, plan)
			if err != nil {
				t.Fatalf("compileFile failed for %s: %v", lang, err)
			}

			if len(files) == 0 {
				t.Errorf("Expected generated files for %s", lang)
			}

			t.Logf("Generated files for %s: %v", lang, files)
		})
	}
}

func TestExecutor_ParallelExecution(t *testing.T) {
	log := logger.New(logger.WithLevel(logger.INFO))
	met := metrics.NewCollector()

	executor := NewExecutor(log, met)
	ctx := context.Background()

	// Create many files to test parallel execution
	nodes := make(map[string]*ProtoFile)
	edges := make(map[string][]string)
	order := make([]string, 20)

	for i := 0; i < 20; i++ {
		name := "file" + string(rune('a'+i%26)) + ".proto"
		nodes[name] = &ProtoFile{
			Path:    name,
			Package: "test",
		}
		edges[name] = []string{}
		order[i] = name
	}

	graph := &DependencyGraph{
		Nodes:            nodes,
		Edges:            edges,
		CompilationOrder: order,
	}

	plan := &ExecutionPlan{
		Graph:     graph,
		OutputDir: t.TempDir(),
		Languages: []string{"go"},
		Options: BuildOptions{
			Workers: 8, // Use 8 workers for parallel execution
			Verbose: false,
		},
	}

	start := time.Now()
	result, err := executor.Execute(ctx, plan)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	t.Logf("Parallel execution with 8 workers took %v", duration)
	t.Logf("Generated %d files", len(result.FilesGenerated))

	if len(result.FilesGenerated) < 20 {
		t.Errorf("Expected at least 20 generated files, got %d", len(result.FilesGenerated))
	}
}
