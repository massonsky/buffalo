package graph

import (
	"testing"

	"github.com/massonsky/buffalo/internal/builder"
	"github.com/massonsky/buffalo/pkg/logger"
)

func TestNewGraph(t *testing.T) {
	g := NewGraph()

	if g == nil {
		t.Fatal("NewGraph returned nil")
	}
	if g.Nodes == nil {
		t.Error("Nodes map is nil")
	}
	if g.Edges == nil {
		t.Error("Edges slice is nil")
	}
	if g.NodesByType == nil {
		t.Error("NodesByType map is nil")
	}
	if g.AdjacencyList == nil {
		t.Error("AdjacencyList map is nil")
	}
	if g.ReverseAdjacency == nil {
		t.Error("ReverseAdjacency map is nil")
	}
}

func TestGraph_AddNode(t *testing.T) {
	g := NewGraph()

	node := &Node{
		ID:      "test.proto",
		Type:    NodeTypeFile,
		Name:    "test.proto",
		Package: "test",
	}

	g.AddNode(node)

	if len(g.Nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(g.Nodes))
	}

	if g.Nodes["test.proto"] != node {
		t.Error("Node not found by ID")
	}

	if len(g.NodesByType[NodeTypeFile]) != 1 {
		t.Error("Node not indexed by type")
	}

	// Test nil node
	g.AddNode(nil)
	if len(g.Nodes) != 1 {
		t.Error("Adding nil node should not change count")
	}
}

func TestGraph_AddEdge(t *testing.T) {
	g := NewGraph()

	edge := &Edge{
		From: "a.proto",
		To:   "b.proto",
		Type: "imports",
	}

	g.AddEdge(edge)

	if len(g.Edges) != 1 {
		t.Errorf("Expected 1 edge, got %d", len(g.Edges))
	}

	if len(g.AdjacencyList["a.proto"]) != 1 {
		t.Error("Edge not in adjacency list")
	}

	if len(g.ReverseAdjacency["b.proto"]) != 1 {
		t.Error("Edge not in reverse adjacency")
	}

	// Test nil edge
	g.AddEdge(nil)
	if len(g.Edges) != 1 {
		t.Error("Adding nil edge should not change count")
	}
}

func TestGraph_GetDependencies(t *testing.T) {
	g := NewGraph()

	g.AddNode(&Node{ID: "a.proto", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "b.proto", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "c.proto", Type: NodeTypeFile})

	g.AddEdge(&Edge{From: "a.proto", To: "b.proto", Type: "imports"})
	g.AddEdge(&Edge{From: "a.proto", To: "c.proto", Type: "imports"})

	deps := g.GetDependencies("a.proto")
	if len(deps) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(deps))
	}

	deps = g.GetDependencies("b.proto")
	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies for b.proto, got %d", len(deps))
	}

	deps = g.GetDependencies("nonexistent")
	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies for nonexistent, got %d", len(deps))
	}
}

func TestGraph_GetDependents(t *testing.T) {
	g := NewGraph()

	g.AddNode(&Node{ID: "a.proto", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "b.proto", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "c.proto", Type: NodeTypeFile})

	g.AddEdge(&Edge{From: "a.proto", To: "c.proto", Type: "imports"})
	g.AddEdge(&Edge{From: "b.proto", To: "c.proto", Type: "imports"})

	dependents := g.GetDependents("c.proto")
	if len(dependents) != 2 {
		t.Errorf("Expected 2 dependents, got %d", len(dependents))
	}

	dependents = g.GetDependents("a.proto")
	if len(dependents) != 0 {
		t.Errorf("Expected 0 dependents for a.proto, got %d", len(dependents))
	}
}

func TestGraph_GetTransitiveDependencies(t *testing.T) {
	g := NewGraph()

	// a -> b -> c -> d
	g.AddNode(&Node{ID: "a.proto", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "b.proto", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "c.proto", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "d.proto", Type: NodeTypeFile})

	g.AddEdge(&Edge{From: "a.proto", To: "b.proto", Type: "imports"})
	g.AddEdge(&Edge{From: "b.proto", To: "c.proto", Type: "imports"})
	g.AddEdge(&Edge{From: "c.proto", To: "d.proto", Type: "imports"})

	deps := g.GetTransitiveDependencies("a.proto")
	if len(deps) != 3 {
		t.Errorf("Expected 3 transitive dependencies, got %d: %v", len(deps), deps)
	}

	deps = g.GetTransitiveDependencies("c.proto")
	if len(deps) != 1 {
		t.Errorf("Expected 1 transitive dependency, got %d", len(deps))
	}

	deps = g.GetTransitiveDependencies("d.proto")
	if len(deps) != 0 {
		t.Errorf("Expected 0 transitive dependencies for d.proto, got %d", len(deps))
	}
}

func TestGraph_GetTransitiveDependents(t *testing.T) {
	g := NewGraph()

	// a -> b -> c -> d
	g.AddNode(&Node{ID: "a.proto", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "b.proto", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "c.proto", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "d.proto", Type: NodeTypeFile})

	g.AddEdge(&Edge{From: "a.proto", To: "b.proto", Type: "imports"})
	g.AddEdge(&Edge{From: "b.proto", To: "c.proto", Type: "imports"})
	g.AddEdge(&Edge{From: "c.proto", To: "d.proto", Type: "imports"})

	dependents := g.GetTransitiveDependents("d.proto")
	if len(dependents) != 3 {
		t.Errorf("Expected 3 transitive dependents, got %d: %v", len(dependents), dependents)
	}

	dependents = g.GetTransitiveDependents("a.proto")
	if len(dependents) != 0 {
		t.Errorf("Expected 0 transitive dependents for a.proto, got %d", len(dependents))
	}
}

func TestGraph_FilterByType(t *testing.T) {
	g := NewGraph()

	g.AddNode(&Node{ID: "a.proto", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "b.proto", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "pkg.User", Type: NodeTypeMessage})
	g.AddNode(&Node{ID: "pkg.Order", Type: NodeTypeMessage})

	g.AddEdge(&Edge{From: "a.proto", To: "b.proto", Type: "imports"})
	g.AddEdge(&Edge{From: "pkg.User", To: "pkg.Order", Type: "references"})

	filtered := g.FilterByType(NodeTypeFile)

	if len(filtered.Nodes) != 2 {
		t.Errorf("Expected 2 file nodes, got %d", len(filtered.Nodes))
	}

	if len(filtered.Edges) != 1 {
		t.Errorf("Expected 1 edge between files, got %d", len(filtered.Edges))
	}

	filteredMsg := g.FilterByType(NodeTypeMessage)
	if len(filteredMsg.Nodes) != 2 {
		t.Errorf("Expected 2 message nodes, got %d", len(filteredMsg.Nodes))
	}
}

func TestGraph_SortedNodes(t *testing.T) {
	g := NewGraph()

	g.AddNode(&Node{ID: "c.proto", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "a.proto", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "b.proto", Type: NodeTypeFile})

	nodes := g.SortedNodes()

	if len(nodes) != 3 {
		t.Fatalf("Expected 3 nodes, got %d", len(nodes))
	}

	if nodes[0].ID != "a.proto" || nodes[1].ID != "b.proto" || nodes[2].ID != "c.proto" {
		t.Errorf("Nodes not sorted: %v", nodes)
	}
}

func TestBuilder_BuildFromDependencyGraph(t *testing.T) {
	log := logger.New(logger.WithLevel(logger.DEBUG))
	logAdapter := builder.NewLoggerAdapter(log)
	b := NewBuilder(logAdapter)

	t.Run("FileScope", func(t *testing.T) {
		depGraph := &builder.DependencyGraph{
			Nodes: map[string]*builder.ProtoFile{
				"user.proto": {
					Path:    "user.proto",
					Package: "users",
					Syntax:  "proto3",
					Imports: []string{"common.proto"},
					Messages: []builder.Message{
						{Name: "User", Fields: []builder.Field{{Name: "id", Type: "string"}}},
					},
					Services: []builder.Service{
						{Name: "UserService", Methods: []builder.Method{{Name: "GetUser"}}},
					},
				},
				"common.proto": {
					Path:     "common.proto",
					Package:  "common",
					Syntax:   "proto3",
					Imports:  []string{},
					Messages: []builder.Message{{Name: "Empty"}},
				},
			},
			Edges: map[string][]string{
				"user.proto":   {"common.proto"},
				"common.proto": {},
			},
		}

		graph, err := b.BuildFromDependencyGraph(depGraph, ScopeFile)
		if err != nil {
			t.Fatalf("BuildFromDependencyGraph failed: %v", err)
		}

		if len(graph.Nodes) != 2 {
			t.Errorf("Expected 2 nodes, got %d", len(graph.Nodes))
		}

		if len(graph.Edges) != 1 {
			t.Errorf("Expected 1 edge, got %d", len(graph.Edges))
		}

		userNode := graph.GetNode("user.proto")
		if userNode == nil {
			t.Fatal("user.proto node not found")
		}
		if userNode.Type != NodeTypeFile {
			t.Errorf("Expected file type, got %s", userNode.Type)
		}
		if userNode.Package != "users" {
			t.Errorf("Expected package 'users', got %s", userNode.Package)
		}
	})

	t.Run("PackageScope", func(t *testing.T) {
		depGraph := &builder.DependencyGraph{
			Nodes: map[string]*builder.ProtoFile{
				"users/user.proto":   {Path: "users/user.proto", Package: "users", Imports: []string{"common/types.proto"}},
				"users/admin.proto":  {Path: "users/admin.proto", Package: "users", Imports: []string{"common/types.proto"}},
				"common/types.proto": {Path: "common/types.proto", Package: "common", Imports: []string{}},
			},
			Edges: map[string][]string{
				"users/user.proto":   {"common/types.proto"},
				"users/admin.proto":  {"common/types.proto"},
				"common/types.proto": {},
			},
		}

		graph, err := b.BuildFromDependencyGraph(depGraph, ScopePackage)
		if err != nil {
			t.Fatalf("BuildFromDependencyGraph failed: %v", err)
		}

		if len(graph.Nodes) != 2 {
			t.Errorf("Expected 2 package nodes, got %d", len(graph.Nodes))
		}

		// Should have only 1 edge (users -> common), deduplicated
		if len(graph.Edges) != 1 {
			t.Errorf("Expected 1 deduplicated edge, got %d", len(graph.Edges))
		}
	})

	t.Run("NilGraph", func(t *testing.T) {
		_, err := b.BuildFromDependencyGraph(nil, ScopeFile)
		if err == nil {
			t.Error("Expected error for nil graph")
		}
	})
}

func TestBuilder_BuildMessageGraph(t *testing.T) {
	log := logger.New(logger.WithLevel(logger.DEBUG))
	logAdapter := builder.NewLoggerAdapter(log)
	b := NewBuilder(logAdapter)

	depGraph := &builder.DependencyGraph{
		Nodes: map[string]*builder.ProtoFile{
			"order.proto": {
				Path:    "order.proto",
				Package: "orders",
				Messages: []builder.Message{
					{
						Name: "Order",
						Fields: []builder.Field{
							{Name: "id", Type: "string"},
							{Name: "user", Type: "User"}, // References User message
						},
					},
				},
			},
			"user.proto": {
				Path:    "user.proto",
				Package: "orders",
				Messages: []builder.Message{
					{Name: "User", Fields: []builder.Field{{Name: "id", Type: "string"}}},
				},
			},
		},
		Edges: map[string][]string{
			"order.proto": {"user.proto"},
			"user.proto":  {},
		},
	}

	graph, err := b.BuildFromDependencyGraph(depGraph, ScopeMessage)
	if err != nil {
		t.Fatalf("BuildFromDependencyGraph failed: %v", err)
	}

	if len(graph.Nodes) != 2 {
		t.Errorf("Expected 2 message nodes, got %d", len(graph.Nodes))
	}

	// Check that Order references User
	orderDeps := graph.GetDependencies("orders.Order")
	found := false
	for _, dep := range orderDeps {
		if dep == "orders.User" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected Order to reference User, deps: %v", orderDeps)
	}
}

func TestBuilder_BuildServiceGraph(t *testing.T) {
	log := logger.New(logger.WithLevel(logger.DEBUG))
	logAdapter := builder.NewLoggerAdapter(log)
	b := NewBuilder(logAdapter)

	depGraph := &builder.DependencyGraph{
		Nodes: map[string]*builder.ProtoFile{
			"user.proto": {
				Path:    "user.proto",
				Package: "users",
				Messages: []builder.Message{
					{Name: "GetUserRequest"},
					{Name: "User"},
				},
				Services: []builder.Service{
					{
						Name: "UserService",
						Methods: []builder.Method{
							{
								Name:       "GetUser",
								InputType:  "GetUserRequest",
								OutputType: "User",
							},
						},
					},
				},
			},
		},
		Edges: map[string][]string{"user.proto": {}},
	}

	graph, err := b.BuildFromDependencyGraph(depGraph, ScopeService)
	if err != nil {
		t.Fatalf("BuildFromDependencyGraph failed: %v", err)
	}

	// Should have service + message nodes
	serviceNodes := graph.NodesByType[NodeTypeService]
	if len(serviceNodes) != 1 {
		t.Errorf("Expected 1 service node, got %d", len(serviceNodes))
	}

	// Service should have edges to request/response types
	svcDeps := graph.GetDependencies("users.UserService")
	if len(svcDeps) != 2 {
		t.Errorf("Expected 2 edges from service (req+res), got %d", len(svcDeps))
	}
}

func TestBuilder_BuildFullGraph(t *testing.T) {
	log := logger.New(logger.WithLevel(logger.DEBUG))
	logAdapter := builder.NewLoggerAdapter(log)
	b := NewBuilder(logAdapter)

	depGraph := &builder.DependencyGraph{
		Nodes: map[string]*builder.ProtoFile{
			"user.proto": {
				Path:    "user.proto",
				Package: "users",
				Syntax:  "proto3",
				Messages: []builder.Message{
					{Name: "User"},
				},
				Services: []builder.Service{
					{Name: "UserService"},
				},
				Enums: []builder.Enum{
					{Name: "UserStatus"},
				},
			},
		},
		Edges: map[string][]string{"user.proto": {}},
	}

	graph, err := b.BuildFromDependencyGraph(depGraph, ScopeFull)
	if err != nil {
		t.Fatalf("BuildFromDependencyGraph failed: %v", err)
	}

	// Should have file + message + service + enum nodes
	if len(graph.NodesByType[NodeTypeFile]) != 1 {
		t.Error("Expected 1 file node")
	}
	if len(graph.NodesByType[NodeTypeMessage]) != 1 {
		t.Error("Expected 1 message node")
	}
	if len(graph.NodesByType[NodeTypeService]) != 1 {
		t.Error("Expected 1 service node")
	}
	if len(graph.NodesByType[NodeTypeEnum]) != 1 {
		t.Error("Expected 1 enum node")
	}
}
