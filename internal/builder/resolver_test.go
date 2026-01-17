package builder

import (
	"context"
	"testing"

	"github.com/massonsky/buffalo/pkg/logger"
)

func TestDependencyResolver_Resolve(t *testing.T) {
	log := logger.New(logger.WithLevel(logger.INFO))
	logAdapter := NewLoggerAdapter(log)
	resolver := NewDependencyResolver(logAdapter)
	ctx := context.Background()

	// Test case 1: Simple linear dependency
	t.Run("LinearDependency", func(t *testing.T) {
		files := []*ProtoFile{
			{Path: "a.proto", Package: "a", Imports: []string{"b.proto"}},
			{Path: "b.proto", Package: "b", Imports: []string{}},
		}

		graph, err := resolver.Resolve(ctx, files)
		if err != nil {
			t.Fatalf("Resolve failed: %v", err)
		}

		if len(graph.Nodes) != 2 {
			t.Errorf("Expected 2 nodes, got %d", len(graph.Nodes))
		}

		if len(graph.CompilationOrder) != 2 {
			t.Errorf("Expected 2 files in compilation order, got %d", len(graph.CompilationOrder))
		}

		// b.proto should come before a.proto
		if graph.CompilationOrder[0] != "b.proto" {
			t.Errorf("Expected b.proto first, got %s", graph.CompilationOrder[0])
		}
		if graph.CompilationOrder[1] != "a.proto" {
			t.Errorf("Expected a.proto second, got %s", graph.CompilationOrder[1])
		}
	})

	// Test case 2: Complex dependency graph
	t.Run("ComplexDependency", func(t *testing.T) {
		files := []*ProtoFile{
			{Path: "a.proto", Package: "a", Imports: []string{"b.proto", "c.proto"}},
			{Path: "b.proto", Package: "b", Imports: []string{"d.proto"}},
			{Path: "c.proto", Package: "c", Imports: []string{"d.proto"}},
			{Path: "d.proto", Package: "d", Imports: []string{}},
		}

		graph, err := resolver.Resolve(ctx, files)
		if err != nil {
			t.Fatalf("Resolve failed: %v", err)
		}

		if len(graph.Nodes) != 4 {
			t.Errorf("Expected 4 nodes, got %d", len(graph.Nodes))
		}

		if len(graph.CompilationOrder) != 4 {
			t.Errorf("Expected 4 files in compilation order, got %d", len(graph.CompilationOrder))
		}

		// d.proto should be first
		if graph.CompilationOrder[0] != "d.proto" {
			t.Errorf("Expected d.proto first, got %s", graph.CompilationOrder[0])
		}

		// a.proto should be last
		if graph.CompilationOrder[3] != "a.proto" {
			t.Errorf("Expected a.proto last, got %s", graph.CompilationOrder[3])
		}
	})

	// Test case 3: Circular dependency detection
	t.Run("CircularDependency", func(t *testing.T) {
		files := []*ProtoFile{
			{Path: "a.proto", Package: "a", Imports: []string{"b.proto"}},
			{Path: "b.proto", Package: "b", Imports: []string{"c.proto"}},
			{Path: "c.proto", Package: "c", Imports: []string{"a.proto"}},
		}

		_, err := resolver.Resolve(ctx, files)
		if err == nil {
			t.Error("Expected error for circular dependency, got nil")
		}

		if err != nil {
			t.Logf("Got expected error: %v", err)
		}
	})

	// Test case 4: No dependencies
	t.Run("NoDependencies", func(t *testing.T) {
		files := []*ProtoFile{
			{Path: "a.proto", Package: "a", Imports: []string{}},
			{Path: "b.proto", Package: "b", Imports: []string{}},
		}

		graph, err := resolver.Resolve(ctx, files)
		if err != nil {
			t.Fatalf("Resolve failed: %v", err)
		}

		if len(graph.Nodes) != 2 {
			t.Errorf("Expected 2 nodes, got %d", len(graph.Nodes))
		}

		if len(graph.CompilationOrder) != 2 {
			t.Errorf("Expected 2 files in compilation order, got %d", len(graph.CompilationOrder))
		}
	})

	// Test case 5: Empty file list
	t.Run("EmptyFileList", func(t *testing.T) {
		graph, err := resolver.Resolve(ctx, []*ProtoFile{})
		if err != nil {
			t.Fatalf("Resolve failed for empty list: %v", err)
		}

		if len(graph.Nodes) != 0 {
			t.Errorf("Expected 0 nodes, got %d", len(graph.Nodes))
		}

		if len(graph.CompilationOrder) != 0 {
			t.Errorf("Expected empty compilation order, got %d", len(graph.CompilationOrder))
		}
	})
}

func TestDependencyGraph_GetDependencies(t *testing.T) {
	graph := &DependencyGraph{
		Nodes: map[string]*ProtoFile{
			"a.proto": {Path: "a.proto", Package: "a"},
			"b.proto": {Path: "b.proto", Package: "b"},
			"c.proto": {Path: "c.proto", Package: "c"},
		},
		Edges: map[string][]string{
			"a.proto": {"b.proto", "c.proto"},
			"b.proto": {},
			"c.proto": {},
		},
	}

	deps := graph.GetDependencies("a.proto")
	if len(deps) != 2 {
		t.Errorf("Expected 2 dependencies for a.proto, got %d", len(deps))
	}

	deps = graph.GetDependencies("b.proto")
	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies for b.proto, got %d", len(deps))
	}

	deps = graph.GetDependencies("nonexistent.proto")
	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies for nonexistent file, got %d", len(deps))
	}
}

func TestDependencyGraph_GetTransitiveDependencies(t *testing.T) {
	graph := &DependencyGraph{
		Nodes: map[string]*ProtoFile{
			"a.proto": {Path: "a.proto", Package: "a"},
			"b.proto": {Path: "b.proto", Package: "b"},
			"c.proto": {Path: "c.proto", Package: "c"},
			"d.proto": {Path: "d.proto", Package: "d"},
		},
		Edges: map[string][]string{
			"a.proto": {"b.proto"},
			"b.proto": {"c.proto"},
			"c.proto": {"d.proto"},
			"d.proto": {},
		},
	}

	deps := graph.GetTransitiveDependencies("a.proto")
	// a.proto depends on b.proto, which depends on c.proto, which depends on d.proto
	// So transitive deps should be: b.proto, c.proto, d.proto
	if len(deps) != 3 {
		t.Errorf("Expected 3 transitive dependencies for a.proto, got %d: %v", len(deps), deps)
	}

	deps = graph.GetTransitiveDependencies("b.proto")
	// b.proto -> c.proto -> d.proto
	if len(deps) != 2 {
		t.Errorf("Expected 2 transitive dependencies for b.proto, got %d: %v", len(deps), deps)
	}

	deps = graph.GetTransitiveDependencies("d.proto")
	if len(deps) != 0 {
		t.Errorf("Expected 0 transitive dependencies for d.proto, got %d", len(deps))
	}
}

func TestDependencyGraph_Validate(t *testing.T) {
	t.Run("ValidGraph", func(t *testing.T) {
		graph := &DependencyGraph{
			Nodes: map[string]*ProtoFile{
				"a.proto": {Path: "a.proto", Package: "a"},
				"b.proto": {Path: "b.proto", Package: "b"},
			},
			Edges: map[string][]string{
				"a.proto": {"b.proto"},
				"b.proto": {},
			},
		}

		if err := graph.Validate(); err != nil {
			t.Errorf("Expected valid graph, got error: %v", err)
		}
	})

	t.Run("MissingDependency", func(t *testing.T) {
		graph := &DependencyGraph{
			Nodes: map[string]*ProtoFile{
				"a.proto": {Path: "a.proto", Package: "a"},
			},
			Edges: map[string][]string{
				"a.proto": {"nonexistent.proto"},
			},
		}

		err := graph.Validate()
		if err == nil {
			t.Error("Expected error for missing dependency, got nil")
		}
	})

	t.Run("EmptyGraph", func(t *testing.T) {
		graph := &DependencyGraph{
			Nodes: map[string]*ProtoFile{},
			Edges: map[string][]string{},
		}

		if err := graph.Validate(); err != nil {
			t.Errorf("Expected valid empty graph, got error: %v", err)
		}
	})
}
