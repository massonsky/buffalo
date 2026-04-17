package graph

import (
	"github.com/massonsky/buffalo/pkg/errors"
)

// CycleInfo contains information about a detected cycle.
type CycleInfo struct {
	// Nodes are the node IDs forming the cycle.
	Nodes []string
	// Type is the type of nodes in the cycle.
	Type NodeType
}

// CouplingMetrics contains coupling analysis results.
type CouplingMetrics struct {
	// NodeID is the analyzed node.
	NodeID string
	// AfferentCoupling (Ca) - number of nodes that depend on this node.
	AfferentCoupling int
	// EfferentCoupling (Ce) - number of nodes this node depends on.
	EfferentCoupling int
	// Instability = Ce / (Ca + Ce), ranges from 0 (stable) to 1 (unstable).
	Instability float64
}

// AnalysisResult contains the results of graph analysis.
type AnalysisResult struct {
	// Cycles are detected cycles in the graph.
	Cycles []CycleInfo
	// Orphans are nodes with no incoming or outgoing edges.
	Orphans []string
	// HeavyNodes are nodes with many transitive dependencies.
	HeavyNodes []HeavyNode
	// Coupling contains coupling metrics for each node.
	Coupling []CouplingMetrics
	// Layers suggests a layered architecture based on dependencies.
	Layers []Layer
}

// HeavyNode represents a node with many dependencies.
type HeavyNode struct {
	NodeID                 string
	TransitiveDependencies int
	TransitiveDependents   int
}

// Layer represents a suggested layer in the architecture.
type Layer struct {
	// Name is the layer name (e.g., "domain", "api", "infra").
	Name string
	// Level is the layer level (0 = lowest/foundation).
	Level int
	// Nodes are the node IDs in this layer.
	Nodes []string
}

// Analyzer performs various analyzes on graphs.
type Analyzer struct {
	graph *Graph
}

// NewAnalyzer creates a new graph analyzer.
func NewAnalyzer(graph *Graph) *Analyzer {
	return &Analyzer{graph: graph}
}

// DetectCycles finds all cycles in the graph using DFS.
func (a *Analyzer) DetectCycles() []CycleInfo {
	var cycles []CycleInfo

	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	parent := make(map[string]string)

	var dfs func(nodeID string, path []string) bool
	dfs = func(nodeID string, path []string) bool {
		visited[nodeID] = true
		recStack[nodeID] = true
		currentPath := append(path, nodeID)

		for _, edge := range a.graph.GetOutgoingEdges(nodeID) {
			neighbor := edge.To
			if !visited[neighbor] {
				parent[neighbor] = nodeID
				if dfs(neighbor, currentPath) {
					return true
				}
			} else if recStack[neighbor] {
				// Found cycle - extract it
				cycleStart := -1
				for i, n := range currentPath {
					if n == neighbor {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					cycleNodes := append(currentPath[cycleStart:], neighbor)

					// Determine node type
					nodeType := NodeTypeFile
					if node := a.graph.GetNode(neighbor); node != nil {
						nodeType = node.Type
					}

					cycles = append(cycles, CycleInfo{
						Nodes: cycleNodes,
						Type:  nodeType,
					})
				}
			}
		}

		recStack[nodeID] = false
		return false
	}

	for nodeID := range a.graph.Nodes {
		if !visited[nodeID] {
			dfs(nodeID, nil)
		}
	}

	return cycles
}

// FindOrphans finds nodes with no connections.
func (a *Analyzer) FindOrphans() []string {
	var orphans []string

	for nodeID := range a.graph.Nodes {
		incoming := len(a.graph.GetIncomingEdges(nodeID))
		outgoing := len(a.graph.GetOutgoingEdges(nodeID))

		if incoming == 0 && outgoing == 0 {
			orphans = append(orphans, nodeID)
		}
	}

	return orphans
}

// FindUnusedFiles finds files that are not imported by anyone (potential orphans).
func (a *Analyzer) FindUnusedFiles() []string {
	var unused []string

	for nodeID, node := range a.graph.Nodes {
		if node.Type != NodeTypeFile {
			continue
		}

		// Check if this file is imported by any other file
		incoming := a.graph.GetIncomingEdges(nodeID)
		if len(incoming) == 0 {
			unused = append(unused, nodeID)
		}
	}

	return unused
}

// FindHeavyNodes finds nodes with the most transitive dependencies.
func (a *Analyzer) FindHeavyNodes(limit int) []HeavyNode {
	heavyNodes := make([]HeavyNode, 0, len(a.graph.Nodes))

	for nodeID := range a.graph.Nodes {
		transDeps := len(a.graph.GetTransitiveDependencies(nodeID))
		transDependents := len(a.graph.GetTransitiveDependents(nodeID))

		heavyNodes = append(heavyNodes, HeavyNode{
			NodeID:                 nodeID,
			TransitiveDependencies: transDeps,
			TransitiveDependents:   transDependents,
		})
	}

	// Sort by total connections (dependencies + dependents)
	for i := 0; i < len(heavyNodes)-1; i++ {
		for j := i + 1; j < len(heavyNodes); j++ {
			totalI := heavyNodes[i].TransitiveDependencies + heavyNodes[i].TransitiveDependents
			totalJ := heavyNodes[j].TransitiveDependencies + heavyNodes[j].TransitiveDependents
			if totalJ > totalI {
				heavyNodes[i], heavyNodes[j] = heavyNodes[j], heavyNodes[i]
			}
		}
	}

	if limit > 0 && limit < len(heavyNodes) {
		return heavyNodes[:limit]
	}

	return heavyNodes
}

// CalculateCoupling calculates coupling metrics for all nodes.
func (a *Analyzer) CalculateCoupling() []CouplingMetrics {
	metrics := make([]CouplingMetrics, 0, len(a.graph.Nodes))

	for nodeID := range a.graph.Nodes {
		ca := len(a.graph.GetIncomingEdges(nodeID)) // Afferent (dependents)
		ce := len(a.graph.GetOutgoingEdges(nodeID)) // Efferent (dependencies)

		instability := float64(0)
		if ca+ce > 0 {
			instability = float64(ce) / float64(ca+ce)
		}

		metrics = append(metrics, CouplingMetrics{
			NodeID:           nodeID,
			AfferentCoupling: ca,
			EfferentCoupling: ce,
			Instability:      instability,
		})
	}

	return metrics
}

// SuggestLayers suggests a layered architecture based on dependency depth.
func (a *Analyzer) SuggestLayers() []Layer {
	// Return empty slice for empty graph
	if len(a.graph.Nodes) == 0 {
		return []Layer{}
	}

	// Calculate depth for each node (longest path from any leaf)
	depths := make(map[string]int)

	// Find leaf nodes (nodes with no dependencies)
	var calculateDepth func(nodeID string, visited map[string]bool) int
	calculateDepth = func(nodeID string, visited map[string]bool) int {
		if d, exists := depths[nodeID]; exists {
			return d
		}

		if visited[nodeID] {
			return 0 // Cycle protection
		}
		visited[nodeID] = true

		deps := a.graph.GetDependencies(nodeID)
		if len(deps) == 0 {
			depths[nodeID] = 0
			return 0
		}

		maxDepth := 0
		for _, dep := range deps {
			d := calculateDepth(dep, visited)
			if d+1 > maxDepth {
				maxDepth = d + 1
			}
		}

		depths[nodeID] = maxDepth
		return maxDepth
	}

	for nodeID := range a.graph.Nodes {
		calculateDepth(nodeID, make(map[string]bool))
	}

	// Group nodes by depth
	layerMap := make(map[int][]string)
	maxDepth := 0
	for nodeID, depth := range depths {
		layerMap[depth] = append(layerMap[depth], nodeID)
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	// Create layers
	layers := make([]Layer, 0, maxDepth+1)
	layerNames := []string{"foundation", "core", "domain", "application", "api", "presentation"}

	for level := 0; level <= maxDepth; level++ {
		name := "layer-" + string(rune('A'+level))
		if level < len(layerNames) {
			name = layerNames[level]
		}

		layers = append(layers, Layer{
			Name:  name,
			Level: level,
			Nodes: layerMap[level],
		})
	}

	return layers
}

// Analyze performs comprehensive analysis on the graph.
func (a *Analyzer) Analyze() *AnalysisResult {
	cycles := a.DetectCycles()
	if cycles == nil {
		cycles = []CycleInfo{}
	}

	orphans := a.FindOrphans()
	if orphans == nil {
		orphans = []string{}
	}

	heavyNodes := a.FindHeavyNodes(10)
	if heavyNodes == nil {
		heavyNodes = []HeavyNode{}
	}

	coupling := a.CalculateCoupling()
	if coupling == nil {
		coupling = []CouplingMetrics{}
	}

	layers := a.SuggestLayers()
	if layers == nil {
		layers = []Layer{}
	}

	return &AnalysisResult{
		Cycles:     cycles,
		Orphans:    orphans,
		HeavyNodes: heavyNodes,
		Coupling:   coupling,
		Layers:     layers,
	}
}

// HasCycles returns true if the graph has cycles.
func (a *Analyzer) HasCycles() bool {
	cycles := a.DetectCycles()
	return len(cycles) > 0
}

// ValidateAcyclic validates that the graph has no cycles.
func (a *Analyzer) ValidateAcyclic() error {
	cycles := a.DetectCycles()
	if len(cycles) > 0 {
		return errors.New(errors.ErrDependency, "graph contains %d cycle(s)", len(cycles))
	}
	return nil
}
