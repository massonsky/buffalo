package graph

import (
	"testing"
)

func createTestGraph() *Graph {
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

	g.AddEdge(&Edge{From: "a", To: "b", Type: "imports"})
	g.AddEdge(&Edge{From: "b", To: "c", Type: "imports"})
	g.AddEdge(&Edge{From: "c", To: "d", Type: "imports"})
	g.AddEdge(&Edge{From: "b", To: "e", Type: "imports"})

	return g
}

func createCyclicGraph() *Graph {
	g := NewGraph()

	// a -> b -> c -> a (cycle)
	g.AddNode(&Node{ID: "a", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "b", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "c", Type: NodeTypeFile})

	g.AddEdge(&Edge{From: "a", To: "b", Type: "imports"})
	g.AddEdge(&Edge{From: "b", To: "c", Type: "imports"})
	g.AddEdge(&Edge{From: "c", To: "a", Type: "imports"})

	return g
}

func TestAnalyzer_DetectCycles_NoCycles(t *testing.T) {
	g := createTestGraph()
	analyzer := NewAnalyzer(g)

	cycles := analyzer.DetectCycles()
	if len(cycles) != 0 {
		t.Errorf("Expected no cycles, found %d", len(cycles))
	}
}

func TestAnalyzer_DetectCycles_WithCycle(t *testing.T) {
	g := createCyclicGraph()
	analyzer := NewAnalyzer(g)

	cycles := analyzer.DetectCycles()
	if len(cycles) == 0 {
		t.Error("Expected to detect cycle, found none")
	}
}

func TestAnalyzer_HasCycles(t *testing.T) {
	t.Run("NoCycles", func(t *testing.T) {
		g := createTestGraph()
		analyzer := NewAnalyzer(g)

		if analyzer.HasCycles() {
			t.Error("Expected no cycles")
		}
	})

	t.Run("WithCycles", func(t *testing.T) {
		g := createCyclicGraph()
		analyzer := NewAnalyzer(g)

		if !analyzer.HasCycles() {
			t.Error("Expected cycles to be detected")
		}
	})
}

func TestAnalyzer_ValidateAcyclic(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		g := createTestGraph()
		analyzer := NewAnalyzer(g)

		err := analyzer.ValidateAcyclic()
		if err != nil {
			t.Errorf("Expected valid acyclic graph, got error: %v", err)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		g := createCyclicGraph()
		analyzer := NewAnalyzer(g)

		err := analyzer.ValidateAcyclic()
		if err == nil {
			t.Error("Expected error for cyclic graph")
		}
	})
}

func TestAnalyzer_FindOrphans(t *testing.T) {
	g := NewGraph()

	g.AddNode(&Node{ID: "connected1", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "connected2", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "orphan1", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "orphan2", Type: NodeTypeFile})

	g.AddEdge(&Edge{From: "connected1", To: "connected2", Type: "imports"})

	analyzer := NewAnalyzer(g)
	orphans := analyzer.FindOrphans()

	if len(orphans) != 2 {
		t.Errorf("Expected 2 orphans, got %d: %v", len(orphans), orphans)
	}

	orphanSet := make(map[string]bool)
	for _, o := range orphans {
		orphanSet[o] = true
	}

	if !orphanSet["orphan1"] || !orphanSet["orphan2"] {
		t.Error("Expected orphan1 and orphan2 to be identified as orphans")
	}
}

func TestAnalyzer_FindUnusedFiles(t *testing.T) {
	g := NewGraph()

	// a imports b, c is unused
	g.AddNode(&Node{ID: "a.proto", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "b.proto", Type: NodeTypeFile})
	g.AddNode(&Node{ID: "c.proto", Type: NodeTypeFile})

	g.AddEdge(&Edge{From: "a.proto", To: "b.proto", Type: "imports"})

	analyzer := NewAnalyzer(g)
	unused := analyzer.FindUnusedFiles()

	// Both a.proto and c.proto have no incoming edges
	// (a.proto is not imported by anyone, c.proto is not imported by anyone)
	if len(unused) != 2 {
		t.Errorf("Expected 2 unused files, got %d: %v", len(unused), unused)
	}
}

func TestAnalyzer_FindHeavyNodes(t *testing.T) {
	g := createTestGraph()
	analyzer := NewAnalyzer(g)

	heavy := analyzer.FindHeavyNodes(3)

	if len(heavy) > 3 {
		t.Errorf("Expected at most 3 heavy nodes, got %d", len(heavy))
	}

	// 'a' should be heavy because it has most transitive dependencies
	if len(heavy) > 0 {
		// First should be 'a' with 4 transitive deps (b, c, d, e)
		// Or 'd' with 3 transitive dependents (c, b, a)
		t.Logf("Heavy nodes: %+v", heavy)
	}
}

func TestAnalyzer_CalculateCoupling(t *testing.T) {
	g := createTestGraph()
	analyzer := NewAnalyzer(g)

	metrics := analyzer.CalculateCoupling()

	if len(metrics) != 5 {
		t.Errorf("Expected 5 coupling metrics, got %d", len(metrics))
	}

	// Find metrics for node 'b' which has 1 incoming (from a) and 2 outgoing (to c, e)
	var bMetrics *CouplingMetrics
	for i := range metrics {
		if metrics[i].NodeID == "b" {
			bMetrics = &metrics[i]
			break
		}
	}

	if bMetrics == nil {
		t.Fatal("Metrics for node 'b' not found")
	}

	if bMetrics.AfferentCoupling != 1 {
		t.Errorf("Expected Ca=1 for 'b', got %d", bMetrics.AfferentCoupling)
	}

	if bMetrics.EfferentCoupling != 2 {
		t.Errorf("Expected Ce=2 for 'b', got %d", bMetrics.EfferentCoupling)
	}

	// Instability = Ce / (Ca + Ce) = 2 / 3 ≈ 0.67
	expectedInstability := 2.0 / 3.0
	if bMetrics.Instability < expectedInstability-0.01 || bMetrics.Instability > expectedInstability+0.01 {
		t.Errorf("Expected instability ~%.2f for 'b', got %.2f", expectedInstability, bMetrics.Instability)
	}
}

func TestAnalyzer_SuggestLayers(t *testing.T) {
	g := createTestGraph()
	analyzer := NewAnalyzer(g)

	layers := analyzer.SuggestLayers()

	if len(layers) == 0 {
		t.Error("Expected at least one layer")
	}

	// Nodes d and e should be at level 0 (no dependencies)
	// c should be at level 1
	// b should be at level 2
	// a should be at level 3
	t.Logf("Layers: %+v", layers)

	// Check that foundation layer contains leaf nodes
	foundD := false
	foundE := false
	for _, layer := range layers {
		if layer.Level == 0 {
			for _, node := range layer.Nodes {
				if node == "d" {
					foundD = true
				}
				if node == "e" {
					foundE = true
				}
			}
		}
	}

	if !foundD || !foundE {
		t.Error("Expected d and e to be in foundation layer (level 0)")
	}
}

func TestAnalyzer_Analyze(t *testing.T) {
	g := createTestGraph()
	analyzer := NewAnalyzer(g)

	result := analyzer.Analyze()

	if result == nil {
		t.Fatal("Analyze returned nil")
	}

	if result.Cycles == nil {
		t.Error("Cycles should not be nil")
	}

	if result.Orphans == nil {
		t.Error("Orphans should not be nil")
	}

	if result.HeavyNodes == nil {
		t.Error("HeavyNodes should not be nil")
	}

	if result.Coupling == nil {
		t.Error("Coupling should not be nil")
	}

	if result.Layers == nil {
		t.Error("Layers should not be nil")
	}
}

func TestAnalyzer_EmptyGraph(t *testing.T) {
	g := NewGraph()
	analyzer := NewAnalyzer(g)

	cycles := analyzer.DetectCycles()
	if len(cycles) != 0 {
		t.Error("Expected no cycles in empty graph")
	}

	orphans := analyzer.FindOrphans()
	if len(orphans) != 0 {
		t.Error("Expected no orphans in empty graph")
	}

	heavy := analyzer.FindHeavyNodes(5)
	if len(heavy) != 0 {
		t.Error("Expected no heavy nodes in empty graph")
	}

	coupling := analyzer.CalculateCoupling()
	if len(coupling) != 0 {
		t.Error("Expected no coupling metrics in empty graph")
	}

	layers := analyzer.SuggestLayers()
	if len(layers) != 0 {
		t.Error("Expected no layers in empty graph")
	}
}

func TestAnalyzer_SingleNode(t *testing.T) {
	g := NewGraph()
	g.AddNode(&Node{ID: "single", Type: NodeTypeFile})

	analyzer := NewAnalyzer(g)

	if analyzer.HasCycles() {
		t.Error("Single node should have no cycles")
	}

	orphans := analyzer.FindOrphans()
	if len(orphans) != 1 {
		t.Errorf("Expected 1 orphan, got %d", len(orphans))
	}

	coupling := analyzer.CalculateCoupling()
	if len(coupling) != 1 {
		t.Errorf("Expected 1 coupling metric, got %d", len(coupling))
	}

	// Single node should have instability 0 (0 deps, 0 dependents, so 0/0 = 0)
	if coupling[0].Instability != 0 {
		t.Errorf("Expected instability 0 for isolated node, got %f", coupling[0].Instability)
	}
}
