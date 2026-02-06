package graph

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Format represents the output format for rendering.
type Format string

const (
	FormatTree     Format = "tree"
	FormatDOT      Format = "dot"
	FormatMermaid  Format = "mermaid"
	FormatJSON     Format = "json"
	FormatPlantUML Format = "plantuml"
)

// Renderer renders graphs to various formats.
type Renderer interface {
	Render(w io.Writer, graph *Graph) error
}

// NewRenderer creates a renderer for the specified format.
func NewRenderer(format Format) Renderer {
	switch format {
	case FormatTree:
		return &TreeRenderer{}
	case FormatDOT:
		return &DOTRenderer{}
	case FormatMermaid:
		return &MermaidRenderer{}
	case FormatJSON:
		return &JSONRenderer{}
	case FormatPlantUML:
		return &PlantUMLRenderer{}
	default:
		return &TreeRenderer{}
	}
}

// TreeRenderer renders graphs as ASCII trees.
type TreeRenderer struct{}

func (r *TreeRenderer) Render(w io.Writer, graph *Graph) error {
	nodes := graph.SortedNodes()

	for _, node := range nodes {
		// Print node
		fmt.Fprintf(w, "%s\n", node.ID)

		// Print imports (outgoing edges)
		deps := graph.GetDependencies(node.ID)
		if len(deps) > 0 {
			fmt.Fprintf(w, "├── imports\n")
			for i, dep := range deps {
				prefix := "│   ├── "
				if i == len(deps)-1 {
					prefix = "│   └── "
				}
				fmt.Fprintf(w, "%s%s\n", prefix, dep)
			}
		}

		// Print imported by (incoming edges)
		dependents := graph.GetDependents(node.ID)
		if len(dependents) > 0 {
			fmt.Fprintf(w, "├── imported by\n")
			for i, dep := range dependents {
				prefix := "│   ├── "
				if i == len(dependents)-1 {
					prefix = "│   └── "
				}
				fmt.Fprintf(w, "%s%s\n", prefix, dep)
			}
		}

		// Print metadata
		if node.Type == NodeTypeFile {
			fmt.Fprintf(w, "├── package: %s\n", node.Package)
			if msgs, ok := node.Metadata["messages"].(int); ok && msgs > 0 {
				fmt.Fprintf(w, "├── messages: %d\n", msgs)
			}
			if svcs, ok := node.Metadata["services"].(int); ok && svcs > 0 {
				fmt.Fprintf(w, "├── services: %d\n", svcs)
			}
		}

		fmt.Fprintln(w)
	}

	return nil
}

// DOTRenderer renders graphs in Graphviz DOT format.
type DOTRenderer struct {
	// Title is the graph title.
	Title string
	// RankDir is the direction (TB, LR, BT, RL).
	RankDir string
}

func (r *DOTRenderer) Render(w io.Writer, graph *Graph) error {
	title := r.Title
	if title == "" {
		title = "dependencies"
	}
	rankDir := r.RankDir
	if rankDir == "" {
		rankDir = "TB"
	}

	fmt.Fprintf(w, "digraph %s {\n", sanitizeDOTID(title))
	fmt.Fprintf(w, "  rankdir=%s;\n", rankDir)
	fmt.Fprintf(w, "  node [shape=box, style=rounded];\n")
	fmt.Fprintln(w)

	// Define node styles by type
	nodeStyles := map[NodeType]string{
		NodeTypeFile:    `shape=box, style="rounded,filled", fillcolor="#e1f5fe"`,
		NodeTypePackage: `shape=folder, style=filled, fillcolor="#fff3e0"`,
		NodeTypeMessage: `shape=record, style=filled, fillcolor="#e8f5e9"`,
		NodeTypeService: `shape=component, style=filled, fillcolor="#fce4ec"`,
		NodeTypeEnum:    `shape=note, style=filled, fillcolor="#f3e5f5"`,
	}

	// Write nodes
	fmt.Fprintln(w, "  // Nodes")
	nodes := graph.SortedNodes()
	for _, node := range nodes {
		style := nodeStyles[node.Type]
		label := node.Name
		if node.Package != "" && node.Type != NodeTypePackage {
			label = node.Package + "." + node.Name
		}
		fmt.Fprintf(w, "  %s [label=%q, %s];\n",
			sanitizeDOTID(node.ID), label, style)
	}

	fmt.Fprintln(w)

	// Write edges
	fmt.Fprintln(w, "  // Edges")
	edgeStyles := map[string]string{
		"imports":    `style=solid, color="#1976d2"`,
		"depends":    `style=solid, color="#388e3c"`,
		"references": `style=dashed, color="#7b1fa2"`,
		"request":    `style=bold, color="#d32f2f", label="req"`,
		"response":   `style=bold, color="#f57c00", label="res"`,
		"defined_in": `style=dotted, color="#757575"`,
	}

	for _, edge := range graph.Edges {
		style := edgeStyles[edge.Type]
		if style == "" {
			style = `style=solid`
		}
		fmt.Fprintf(w, "  %s -> %s [%s];\n",
			sanitizeDOTID(edge.From), sanitizeDOTID(edge.To), style)
	}

	fmt.Fprintln(w, "}")

	return nil
}

// MermaidRenderer renders graphs in Mermaid format.
type MermaidRenderer struct {
	// Direction is the graph direction (TD, LR, BT, RL).
	Direction string
}

func (r *MermaidRenderer) Render(w io.Writer, graph *Graph) error {
	direction := r.Direction
	if direction == "" {
		direction = "TD"
	}

	fmt.Fprintf(w, "graph %s\n", direction)

	// Node styles
	nodeShapes := map[NodeType][2]string{
		NodeTypeFile:    {"[", "]"},
		NodeTypePackage: {"[[", "]]"},
		NodeTypeMessage: {"(", ")"},
		NodeTypeService: {"{{", "}}"},
		NodeTypeEnum:    {"[(", ")]"},
	}

	// Write nodes with shapes
	nodes := graph.SortedNodes()
	for _, node := range nodes {
		shape := nodeShapes[node.Type]
		if shape[0] == "" {
			shape = [2]string{"[", "]"}
		}
		nodeID := sanitizeMermaidID(node.ID)
		label := node.Name
		fmt.Fprintf(w, "    %s%s%q%s\n", nodeID, shape[0], label, shape[1])
	}

	fmt.Fprintln(w)

	// Write edges
	edgeArrows := map[string]string{
		"imports":    "-->",
		"depends":    "-->",
		"references": "-.->",
		"request":    "==>",
		"response":   "==>",
		"defined_in": "-.-",
	}

	for _, edge := range graph.Edges {
		arrow := edgeArrows[edge.Type]
		if arrow == "" {
			arrow = "-->"
		}
		fromID := sanitizeMermaidID(edge.From)
		toID := sanitizeMermaidID(edge.To)

		if edge.Type == "request" || edge.Type == "response" {
			fmt.Fprintf(w, "    %s %s|%s| %s\n", fromID, arrow, edge.Type, toID)
		} else {
			fmt.Fprintf(w, "    %s %s %s\n", fromID, arrow, toID)
		}
	}

	// Add styles
	fmt.Fprintln(w)
	fmt.Fprintln(w, "    %% Styles")

	// Group nodes by type for styling
	for nodeType, nodes := range graph.NodesByType {
		if len(nodes) == 0 {
			continue
		}
		var style string
		switch nodeType {
		case NodeTypeFile:
			style = "fill:#e1f5fe,stroke:#0288d1"
		case NodeTypePackage:
			style = "fill:#fff3e0,stroke:#f57c00"
		case NodeTypeMessage:
			style = "fill:#e8f5e9,stroke:#388e3c"
		case NodeTypeService:
			style = "fill:#fce4ec,stroke:#c2185b"
		case NodeTypeEnum:
			style = "fill:#f3e5f5,stroke:#7b1fa2"
		}

		for _, node := range nodes {
			nodeID := sanitizeMermaidID(node.ID)
			fmt.Fprintf(w, "    style %s %s\n", nodeID, style)
		}
	}

	return nil
}

// JSONRenderer renders graphs as JSON.
type JSONRenderer struct {
	// Indent enables pretty printing.
	Indent bool
}

// JSONGraph is the JSON representation of a graph.
type JSONGraph struct {
	Nodes []JSONNode `json:"nodes"`
	Edges []JSONEdge `json:"edges"`
	Stats JSONStats  `json:"stats"`
}

// JSONNode is the JSON representation of a node.
type JSONNode struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Name     string                 `json:"name"`
	Package  string                 `json:"package,omitempty"`
	FilePath string                 `json:"file_path,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// JSONEdge is the JSON representation of an edge.
type JSONEdge struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Type   string `json:"type"`
	Weight int    `json:"weight,omitempty"`
}

// JSONStats contains graph statistics.
type JSONStats struct {
	TotalNodes  int            `json:"total_nodes"`
	TotalEdges  int            `json:"total_edges"`
	NodesByType map[string]int `json:"nodes_by_type"`
	MaxDepth    int            `json:"max_depth"`
	AvgFanOut   float64        `json:"avg_fan_out"`
}

func (r *JSONRenderer) Render(w io.Writer, graph *Graph) error {
	jg := JSONGraph{
		Nodes: make([]JSONNode, 0, len(graph.Nodes)),
		Edges: make([]JSONEdge, 0, len(graph.Edges)),
	}

	// Add nodes
	nodes := graph.SortedNodes()
	for _, node := range nodes {
		jn := JSONNode{
			ID:       node.ID,
			Type:     string(node.Type),
			Name:     node.Name,
			Package:  node.Package,
			FilePath: node.FilePath,
			Metadata: node.Metadata,
		}
		jg.Nodes = append(jg.Nodes, jn)
	}

	// Add edges
	for _, edge := range graph.Edges {
		je := JSONEdge{
			From:   edge.From,
			To:     edge.To,
			Type:   edge.Type,
			Weight: edge.Weight,
		}
		jg.Edges = append(jg.Edges, je)
	}

	// Calculate stats
	jg.Stats = calculateStats(graph)

	// Marshal
	var data []byte
	var err error
	if r.Indent {
		data, err = json.MarshalIndent(jg, "", "  ")
	} else {
		data, err = json.Marshal(jg)
	}
	if err != nil {
		return err
	}

	_, err = w.Write(data)
	return err
}

func calculateStats(graph *Graph) JSONStats {
	stats := JSONStats{
		TotalNodes:  len(graph.Nodes),
		TotalEdges:  len(graph.Edges),
		NodesByType: make(map[string]int),
	}

	for nodeType, nodes := range graph.NodesByType {
		stats.NodesByType[string(nodeType)] = len(nodes)
	}

	// Calculate max depth and average fan-out
	totalFanOut := 0
	maxDepth := 0

	for nodeID := range graph.Nodes {
		fanOut := len(graph.GetOutgoingEdges(nodeID))
		totalFanOut += fanOut

		depth := len(graph.GetTransitiveDependencies(nodeID))
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	stats.MaxDepth = maxDepth
	if len(graph.Nodes) > 0 {
		stats.AvgFanOut = float64(totalFanOut) / float64(len(graph.Nodes))
	}

	return stats
}

// PlantUMLRenderer renders graphs in PlantUML format.
type PlantUMLRenderer struct{}

func (r *PlantUMLRenderer) Render(w io.Writer, graph *Graph) error {
	fmt.Fprintln(w, "@startuml")
	fmt.Fprintln(w, "skinparam componentStyle rectangle")
	fmt.Fprintln(w, "skinparam linetype ortho")
	fmt.Fprintln(w)

	// Define stereotypes
	fmt.Fprintln(w, "skinparam component {")
	fmt.Fprintln(w, "  BackgroundColor<<file>> #e1f5fe")
	fmt.Fprintln(w, "  BackgroundColor<<package>> #fff3e0")
	fmt.Fprintln(w, "  BackgroundColor<<message>> #e8f5e9")
	fmt.Fprintln(w, "  BackgroundColor<<service>> #fce4ec")
	fmt.Fprintln(w, "  BackgroundColor<<enum>> #f3e5f5")
	fmt.Fprintln(w, "}")
	fmt.Fprintln(w)

	// Write nodes
	nodes := graph.SortedNodes()
	for _, node := range nodes {
		stereotype := string(node.Type)
		alias := sanitizePlantUMLID(node.ID)
		label := node.Name
		if node.Package != "" && node.Type != NodeTypePackage {
			label = node.Package + "\\n" + node.Name
		}
		fmt.Fprintf(w, "component \"%s\" as %s <<%s>>\n", label, alias, stereotype)
	}

	fmt.Fprintln(w)

	// Write edges
	for _, edge := range graph.Edges {
		fromAlias := sanitizePlantUMLID(edge.From)
		toAlias := sanitizePlantUMLID(edge.To)

		arrow := "-->"
		switch edge.Type {
		case "imports", "depends":
			arrow = "-->"
		case "references":
			arrow = "..>"
		case "request":
			arrow = "==>"
		case "response":
			arrow = "==>"
		case "defined_in":
			arrow = "..>"
		}

		if edge.Type != "" && edge.Type != "imports" {
			fmt.Fprintf(w, "%s %s %s : %s\n", fromAlias, arrow, toAlias, edge.Type)
		} else {
			fmt.Fprintf(w, "%s %s %s\n", fromAlias, arrow, toAlias)
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "@enduml")

	return nil
}

// Helper functions for sanitizing IDs

func sanitizeDOTID(id string) string {
	// DOT IDs must start with a letter or underscore
	// Replace special characters
	id = strings.ReplaceAll(id, "/", "_")
	id = strings.ReplaceAll(id, ".", "_")
	id = strings.ReplaceAll(id, "-", "_")
	id = strings.ReplaceAll(id, ":", "_")
	id = strings.ReplaceAll(id, " ", "_")

	// Ensure starts with letter or underscore
	if len(id) > 0 && !isLetter(id[0]) && id[0] != '_' {
		id = "_" + id
	}

	return id
}

func sanitizeMermaidID(id string) string {
	// Mermaid IDs should be alphanumeric
	id = strings.ReplaceAll(id, "/", "_")
	id = strings.ReplaceAll(id, ".", "_")
	id = strings.ReplaceAll(id, "-", "_")
	id = strings.ReplaceAll(id, ":", "_")
	id = strings.ReplaceAll(id, " ", "_")

	return id
}

func sanitizePlantUMLID(id string) string {
	// PlantUML aliases should be alphanumeric
	id = strings.ReplaceAll(id, "/", "_")
	id = strings.ReplaceAll(id, ".", "_")
	id = strings.ReplaceAll(id, "-", "_")
	id = strings.ReplaceAll(id, ":", "_")
	id = strings.ReplaceAll(id, " ", "_")

	return id
}

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// RenderAnalysis renders analysis results.
func RenderAnalysis(w io.Writer, result *AnalysisResult) error {
	// Cycles
	fmt.Fprintln(w, "=== Cycle Analysis ===")
	if len(result.Cycles) == 0 {
		fmt.Fprintln(w, "✅ No cycles detected")
	} else {
		fmt.Fprintf(w, "❌ Found %d cycle(s):\n", len(result.Cycles))
		for i, cycle := range result.Cycles {
			fmt.Fprintf(w, "  %d. %s\n", i+1, strings.Join(cycle.Nodes, " → "))
		}
	}
	fmt.Fprintln(w)

	// Orphans
	fmt.Fprintln(w, "=== Orphan Analysis ===")
	if len(result.Orphans) == 0 {
		fmt.Fprintln(w, "✅ No orphan nodes")
	} else {
		fmt.Fprintf(w, "⚠️  Found %d orphan(s):\n", len(result.Orphans))
		for _, orphan := range result.Orphans {
			fmt.Fprintf(w, "  - %s\n", orphan)
		}
	}
	fmt.Fprintln(w)

	// Heavy nodes
	fmt.Fprintln(w, "=== Heavy Nodes (most connections) ===")
	for i, node := range result.HeavyNodes {
		if i >= 5 {
			break
		}
		total := node.TransitiveDependencies + node.TransitiveDependents
		fmt.Fprintf(w, "  %d. %s (deps: %d, dependents: %d, total: %d)\n",
			i+1, node.NodeID, node.TransitiveDependencies, node.TransitiveDependents, total)
	}
	fmt.Fprintln(w)

	// Coupling metrics
	fmt.Fprintln(w, "=== Coupling Metrics ===")
	// Sort by instability
	sorted := make([]CouplingMetrics, len(result.Coupling))
	copy(sorted, result.Coupling)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Instability > sorted[j].Instability
	})

	fmt.Fprintln(w, "Most unstable (high instability = many outgoing deps, few incoming):")
	for i, m := range sorted {
		if i >= 5 {
			break
		}
		fmt.Fprintf(w, "  %d. %s (Ca=%d, Ce=%d, I=%.2f)\n",
			i+1, m.NodeID, m.AfferentCoupling, m.EfferentCoupling, m.Instability)
	}
	fmt.Fprintln(w)

	// Layers
	fmt.Fprintln(w, "=== Suggested Layers ===")
	for _, layer := range result.Layers {
		fmt.Fprintf(w, "Level %d (%s): %d nodes\n", layer.Level, layer.Name, len(layer.Nodes))
		for _, nodeID := range layer.Nodes {
			fmt.Fprintf(w, "  - %s\n", nodeID)
		}
	}

	return nil
}
