package graph

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func createRendererTestGraph() *Graph {
	g := NewGraph()

	g.AddNode(&Node{
		ID:       "user.proto",
		Type:     NodeTypeFile,
		Name:     "user.proto",
		Package:  "users",
		FilePath: "protos/user.proto",
		Metadata: map[string]interface{}{
			"syntax":   "proto3",
			"messages": 2,
			"services": 1,
		},
	})

	g.AddNode(&Node{
		ID:       "common.proto",
		Type:     NodeTypeFile,
		Name:     "common.proto",
		Package:  "common",
		FilePath: "protos/common.proto",
		Metadata: map[string]interface{}{
			"syntax":   "proto3",
			"messages": 3,
		},
	})

	g.AddEdge(&Edge{
		From:   "user.proto",
		To:     "common.proto",
		Type:   "imports",
		Weight: 1,
	})

	return g
}

func TestNewRenderer(t *testing.T) {
	testCases := []struct {
		format   Format
		expected string
	}{
		{FormatTree, "*graph.TreeRenderer"},
		{FormatDOT, "*graph.DOTRenderer"},
		{FormatMermaid, "*graph.MermaidRenderer"},
		{FormatJSON, "*graph.JSONRenderer"},
		{FormatPlantUML, "*graph.PlantUMLRenderer"},
		{"unknown", "*graph.TreeRenderer"}, // Default to tree
	}

	for _, tc := range testCases {
		renderer := NewRenderer(tc.format)
		if renderer == nil {
			t.Errorf("NewRenderer(%s) returned nil", tc.format)
		}
	}
}

func TestTreeRenderer_Render(t *testing.T) {
	g := createRendererTestGraph()
	renderer := &TreeRenderer{}

	var buf bytes.Buffer
	err := renderer.Render(&buf, g)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := buf.String()

	// Check that node IDs are present
	if !strings.Contains(output, "user.proto") {
		t.Error("Output should contain user.proto")
	}

	if !strings.Contains(output, "common.proto") {
		t.Error("Output should contain common.proto")
	}

	// Check structure
	if !strings.Contains(output, "imports") {
		t.Error("Output should contain 'imports' section")
	}

	if !strings.Contains(output, "package:") {
		t.Error("Output should contain package info")
	}

	t.Logf("Tree output:\n%s", output)
}

func TestDOTRenderer_Render(t *testing.T) {
	g := createRendererTestGraph()
	renderer := &DOTRenderer{Title: "test_graph", RankDir: "LR"}

	var buf bytes.Buffer
	err := renderer.Render(&buf, g)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := buf.String()

	// Check DOT structure
	if !strings.Contains(output, "digraph") {
		t.Error("Output should contain 'digraph'")
	}

	if !strings.Contains(output, "rankdir=LR") {
		t.Error("Output should contain rankdir")
	}

	if !strings.Contains(output, "->") {
		t.Error("Output should contain edge arrows")
	}

	// Check that node IDs are sanitized (dots replaced)
	if !strings.Contains(output, "user_proto") {
		t.Error("Output should contain sanitized node ID")
	}

	t.Logf("DOT output:\n%s", output)
}

func TestMermaidRenderer_Render(t *testing.T) {
	g := createRendererTestGraph()
	renderer := &MermaidRenderer{Direction: "TD"}

	var buf bytes.Buffer
	err := renderer.Render(&buf, g)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := buf.String()

	// Check Mermaid structure
	if !strings.Contains(output, "graph TD") {
		t.Error("Output should contain 'graph TD'")
	}

	if !strings.Contains(output, "-->") {
		t.Error("Output should contain edge arrows")
	}

	// Check styles
	if !strings.Contains(output, "style") {
		t.Error("Output should contain style definitions")
	}

	t.Logf("Mermaid output:\n%s", output)
}

func TestJSONRenderer_Render(t *testing.T) {
	g := createRendererTestGraph()
	renderer := &JSONRenderer{Indent: true}

	var buf bytes.Buffer
	err := renderer.Render(&buf, g)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := buf.String()

	// Verify it's valid JSON
	var result JSONGraph
	err = json.Unmarshal([]byte(output), &result)
	if err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	// Check structure
	if len(result.Nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(result.Nodes))
	}

	if len(result.Edges) != 1 {
		t.Errorf("Expected 1 edge, got %d", len(result.Edges))
	}

	if result.Stats.TotalNodes != 2 {
		t.Errorf("Expected TotalNodes=2, got %d", result.Stats.TotalNodes)
	}

	if result.Stats.TotalEdges != 1 {
		t.Errorf("Expected TotalEdges=1, got %d", result.Stats.TotalEdges)
	}

	t.Logf("JSON output:\n%s", output)
}

func TestJSONRenderer_RenderNoIndent(t *testing.T) {
	g := createRendererTestGraph()
	renderer := &JSONRenderer{Indent: false}

	var buf bytes.Buffer
	err := renderer.Render(&buf, g)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := buf.String()

	// Should be a single line (no newlines in middle)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 1 {
		t.Errorf("Expected single line output, got %d lines", len(lines))
	}

	// Still should be valid JSON
	var result JSONGraph
	err = json.Unmarshal([]byte(output), &result)
	if err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}
}

func TestPlantUMLRenderer_Render(t *testing.T) {
	g := createRendererTestGraph()
	renderer := &PlantUMLRenderer{}

	var buf bytes.Buffer
	err := renderer.Render(&buf, g)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := buf.String()

	// Check PlantUML structure
	if !strings.Contains(output, "@startuml") {
		t.Error("Output should start with @startuml")
	}

	if !strings.Contains(output, "@enduml") {
		t.Error("Output should end with @enduml")
	}

	if !strings.Contains(output, "component") {
		t.Error("Output should contain component definitions")
	}

	if !strings.Contains(output, "-->") {
		t.Error("Output should contain edge arrows")
	}

	t.Logf("PlantUML output:\n%s", output)
}

func TestRenderAnalysis(t *testing.T) {
	result := &AnalysisResult{
		Cycles: []CycleInfo{
			{Nodes: []string{"a", "b", "c", "a"}, Type: NodeTypeFile},
		},
		Orphans: []string{"orphan1", "orphan2"},
		HeavyNodes: []HeavyNode{
			{NodeID: "heavy1", TransitiveDependencies: 10, TransitiveDependents: 5},
		},
		Coupling: []CouplingMetrics{
			{NodeID: "node1", AfferentCoupling: 2, EfferentCoupling: 3, Instability: 0.6},
		},
		Layers: []Layer{
			{Name: "foundation", Level: 0, Nodes: []string{"base1", "base2"}},
			{Name: "core", Level: 1, Nodes: []string{"core1"}},
		},
	}

	var buf bytes.Buffer
	err := RenderAnalysis(&buf, result)
	if err != nil {
		t.Fatalf("RenderAnalysis failed: %v", err)
	}

	output := buf.String()

	// Check sections
	if !strings.Contains(output, "Cycle Analysis") {
		t.Error("Output should contain Cycle Analysis section")
	}

	if !strings.Contains(output, "Orphan Analysis") {
		t.Error("Output should contain Orphan Analysis section")
	}

	if !strings.Contains(output, "Heavy Nodes") {
		t.Error("Output should contain Heavy Nodes section")
	}

	if !strings.Contains(output, "Coupling Metrics") {
		t.Error("Output should contain Coupling Metrics section")
	}

	if !strings.Contains(output, "Suggested Layers") {
		t.Error("Output should contain Suggested Layers section")
	}

	// Check content
	if !strings.Contains(output, "a → b → c → a") {
		t.Error("Output should contain cycle path")
	}

	if !strings.Contains(output, "orphan1") {
		t.Error("Output should contain orphan nodes")
	}

	t.Logf("Analysis output:\n%s", output)
}

func TestRenderAnalysis_NoCycles(t *testing.T) {
	result := &AnalysisResult{
		Cycles:     []CycleInfo{},
		Orphans:    []string{},
		HeavyNodes: []HeavyNode{},
		Coupling:   []CouplingMetrics{},
		Layers:     []Layer{},
	}

	var buf bytes.Buffer
	err := RenderAnalysis(&buf, result)
	if err != nil {
		t.Fatalf("RenderAnalysis failed: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "✅ No cycles detected") {
		t.Error("Output should indicate no cycles")
	}

	if !strings.Contains(output, "✅ No orphan nodes") {
		t.Error("Output should indicate no orphans")
	}
}

func TestSanitizeDOTID(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"file.proto", "file_proto"},
		{"path/to/file.proto", "path_to_file_proto"},
		{"my-file", "my_file"},
		{"file:name", "file_name"},
		{"123start", "_123start"},
	}

	for _, tc := range testCases {
		result := sanitizeDOTID(tc.input)
		if result != tc.expected {
			t.Errorf("sanitizeDOTID(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

func TestSanitizeMermaidID(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"file.proto", "file_proto"},
		{"path/to/file", "path_to_file"},
	}

	for _, tc := range testCases {
		result := sanitizeMermaidID(tc.input)
		if result != tc.expected {
			t.Errorf("sanitizeMermaidID(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

func TestEmptyGraphRendering(t *testing.T) {
	g := NewGraph()

	renderers := []struct {
		name     string
		renderer Renderer
	}{
		{"Tree", &TreeRenderer{}},
		{"DOT", &DOTRenderer{}},
		{"Mermaid", &MermaidRenderer{}},
		{"JSON", &JSONRenderer{Indent: true}},
		{"PlantUML", &PlantUMLRenderer{}},
	}

	for _, tc := range renderers {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tc.renderer.Render(&buf, g)
			if err != nil {
				t.Errorf("%s renderer failed on empty graph: %v", tc.name, err)
			}
		})
	}
}
