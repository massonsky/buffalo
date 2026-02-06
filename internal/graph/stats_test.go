package graph

import (
	"testing"
)

func TestCalculateStats(t *testing.T) {
	g := NewGraph()

	// a -> b -> c -> d
	//      |
	//      v
	//      e
	g.AddNode(&Node{ID: "a", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "b", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "c", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "d", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "e", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "msg1", Type: NodeTypeMessage})
	g.AddNode(&Node{ID: "svc1", Type: NodeTypeService})

	g.AddEdge(&Edge{From: "a", To: "b", Type: "imports"})
	g.AddEdge(&Edge{From: "b", To: "c", Type: "imports"})
	g.AddEdge(&Edge{From: "c", To: "d", Type: "imports"})
	g.AddEdge(&Edge{From: "b", To: "e", Type: "imports"})

	stats := CalculateStats(g)

	if stats.TotalNodes != 7 {
		t.Errorf("Expected 7 nodes, got %d", stats.TotalNodes)
	}

	if stats.TotalEdges != 4 {
		t.Errorf("Expected 4 edges, got %d", stats.TotalEdges)
	}

	if stats.FileCount != 5 {
		t.Errorf("Expected 5 file nodes, got %d", stats.FileCount)
	}

	if stats.MessageCount != 1 {
		t.Errorf("Expected 1 message node, got %d", stats.MessageCount)
	}

	if stats.ServiceCount != 1 {
		t.Errorf("Expected 1 service node, got %d", stats.ServiceCount)
	}

	// Max fan-out should be 2 (node b has 2 outgoing)
	if stats.MaxFanOut != 2 {
		t.Errorf("Expected MaxFanOut=2, got %d", stats.MaxFanOut)
	}

	// Max fan-in should be 1 (each node has at most 1 incoming)
	if stats.MaxFanIn != 1 {
		t.Errorf("Expected MaxFanIn=1, got %d", stats.MaxFanIn)
	}

	// Max depth should be 3 (a -> b -> c -> d)
	if stats.MaxDepth != 4 {
		t.Errorf("Expected MaxDepth=4 (a has 4 transitive deps: b,c,d,e), got %d", stats.MaxDepth)
	}

	// Should have 2 connected components (main tree + 2 isolated nodes)
	if stats.ConnectedComponents != 3 {
		t.Errorf("Expected 3 connected components, got %d", stats.ConnectedComponents)
	}

	t.Logf("Stats: %+v", stats)
}

func TestCalculateStats_EmptyGraph(t *testing.T) {
	g := NewGraph()
	stats := CalculateStats(g)

	if stats.TotalNodes != 0 {
		t.Errorf("Expected 0 nodes, got %d", stats.TotalNodes)
	}

	if stats.TotalEdges != 0 {
		t.Errorf("Expected 0 edges, got %d", stats.TotalEdges)
	}

	if stats.ConnectedComponents != 0 {
		t.Errorf("Expected 0 components, got %d", stats.ConnectedComponents)
	}
}

func TestCalculateStats_SingleNode(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "single", Type: NodeTypeFile})

	stats := CalculateStats(g)

	if stats.TotalNodes != 1 {
		t.Errorf("Expected 1 node, got %d", stats.TotalNodes)
	}

	if stats.ConnectedComponents != 1 {
		t.Errorf("Expected 1 component, got %d", stats.ConnectedComponents)
	}

	if stats.Density != 0 {
		t.Errorf("Expected density 0 for single node, got %f", stats.Density)
	}
}

func TestCalculateStats_FullyConnected(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "a", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "b", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "c", Type: NodeTypeFile})

	// Fully connected (all possible edges)
	g.AddEdge(&Edge{From: "a", To: "b"})
	g.AddEdge(&Edge{From: "a", To: "c"})
	g.AddEdge(&Edge{From: "b", To: "a"})
	g.AddEdge(&Edge{From: "b", To: "c"})
	g.AddEdge(&Edge{From: "c", To: "a"})
	g.AddEdge(&Edge{From: "c", To: "b"})

	stats := CalculateStats(g)

	// Density should be 1.0 (6 edges / 6 possible edges)
	if stats.Density < 0.99 || stats.Density > 1.01 {
		t.Errorf("Expected density ~1.0, got %f", stats.Density)
	}

	if stats.ConnectedComponents != 1 {
		t.Errorf("Expected 1 component, got %d", stats.ConnectedComponents)
	}
}

func TestGetTopByFanOut(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "a", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "b", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "c", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "d", Type: NodeTypeFile})

	// b has most outgoing edges (3)
	g.AddEdge(&Edge{From: "b", To: "a"})
	g.AddEdge(&Edge{From: "b", To: "c"})
	g.AddEdge(&Edge{From: "b", To: "d"})
	// a has 1
	g.AddEdge(&Edge{From: "a", To: "c"})

	top := GetTopByFanOut(g, 2)

	if len(top) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(top))
	}

	if top[0].NodeID != "b" {
		t.Errorf("Expected first to be 'b', got %s", top[0].NodeID)
	}

	if top[0].Value != 3 {
		t.Errorf("Expected value 3 for 'b', got %d", top[0].Value)
	}
}

func TestGetTopByFanIn(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "a", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "b", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "c", Type: NodeTypeFile})

	// c has most incoming edges (2)
	g.AddEdge(&Edge{From: "a", To: "c"})
	g.AddEdge(&Edge{From: "b", To: "c"})

	top := GetTopByFanIn(g, 3)

	if len(top) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(top))
	}

	if top[0].NodeID != "c" {
		t.Errorf("Expected first to be 'c', got %s", top[0].NodeID)
	}

	if top[0].Value != 2 {
		t.Errorf("Expected value 2 for 'c', got %d", top[0].Value)
	}
}

func TestGetTopByTransitiveDeps(t *testing.T) {
	g := NewGraph()
	// a -> b -> c -> d
	g.AddNode(&Node{ID: "a", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "b", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "c", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "d", Type: NodeTypeFile})

	g.AddEdge(&Edge{From: "a", To: "b"})
	g.AddEdge(&Edge{From: "b", To: "c"})
	g.AddEdge(&Edge{From: "c", To: "d"})

	top := GetTopByTransitiveDeps(g, 2)

	if len(top) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(top))
	}

	// 'a' should have most transitive deps (3: b, c, d)
	if top[0].NodeID != "a" {
		t.Errorf("Expected first to be 'a', got %s", top[0].NodeID)
	}

	if top[0].Value != 3 {
		t.Errorf("Expected value 3 for 'a', got %d", top[0].Value)
	}
}

func TestCountConnectedComponents(t *testing.T) {
	testCases := []struct {
		name     string
		setup    func(*Graph)
		expected int
	}{
		{
			name:     "Empty",
			setup:    func(g *Graph) {},
			expected: 0,
		},
		{
			name: "SingleNode",
			setup: func(g *Graph) {
				g.AddNode(&Node{ID: "a", Type: NodeTypeFile})
			},
			expected: 1,
		},
		{
			name: "TwoConnected",
			setup: func(g *Graph) {
				g.AddNode(&Node{ID: "a", Type: NodeTypeFile})
				g.AddNode(&Node{ID: "b", Type: NodeTypeFile})
				g.AddEdge(&Edge{From: "a", To: "b"})
			},
			expected: 1,
		},
		{
			name: "TwoSeparate",
			setup: func(g *Graph) {
				g.AddNode(&Node{ID: "a", Type: NodeTypeFile})
				g.AddNode(&Node{ID: "b", Type: NodeTypeFile})
			},
			expected: 2,
		},
		{
			name: "ThreeSeparate",
			setup: func(g *Graph) {
				g.AddNode(&Node{ID: "a", Type: NodeTypeFile})
				g.AddNode(&Node{ID: "b", Type: NodeTypeFile})
				g.AddNode(&Node{ID: "c", Type: NodeTypeFile})
			},
			expected: 3,
		},
		{
			name: "MixedComponents",
			setup: func(g *Graph) {
				// Component 1: a -> b
				g.AddNode(&Node{ID: "a", Type: NodeTypeFile})
				g.AddNode(&Node{ID: "b", Type: NodeTypeFile})
				g.AddEdge(&Edge{From: "a", To: "b"})
				// Component 2: c -> d
				g.AddNode(&Node{ID: "c", Type: NodeTypeFile})
				g.AddNode(&Node{ID: "d", Type: NodeTypeFile})
				g.AddEdge(&Edge{From: "c", To: "d"})
				// Component 3: isolated e
				g.AddNode(&Node{ID: "e", Type: NodeTypeFile})
			},
			expected: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewGraph()
			tc.setup(g)
			count := countConnectedComponents(g)
			if count != tc.expected {
				t.Errorf("Expected %d components, got %d", tc.expected, count)
			}
		})
	}
}
