package graph

// Stats contains graph statistics.
type Stats struct {
	// TotalNodes is the total number of nodes.
	TotalNodes int
	// TotalEdges is the total number of edges.
	TotalEdges int
	// NodesByType counts nodes by type.
	NodesByType map[NodeType]int
	// FileCount is the number of file nodes.
	FileCount int
	// PackageCount is the number of unique packages.
	PackageCount int
	// MessageCount is the number of message nodes.
	MessageCount int
	// ServiceCount is the number of service nodes.
	ServiceCount int
	// EnumCount is the number of enum nodes.
	EnumCount int
	// MaxDepth is the maximum dependency depth.
	MaxDepth int
	// AvgDepth is the average dependency depth.
	AvgDepth float64
	// MaxFanOut is the maximum number of outgoing edges.
	MaxFanOut int
	// AvgFanOut is the average number of outgoing edges.
	AvgFanOut float64
	// MaxFanIn is the maximum number of incoming edges.
	MaxFanIn int
	// AvgFanIn is the average number of incoming edges.
	AvgFanIn float64
	// Density is the graph density (edges / possible edges).
	Density float64
	// ConnectedComponents is the number of connected components.
	ConnectedComponents int
}

// CalculateStats calculates comprehensive statistics for a graph.
func CalculateStats(g *Graph) Stats {
	stats := Stats{
		TotalNodes:  len(g.Nodes),
		TotalEdges:  len(g.Edges),
		NodesByType: make(map[NodeType]int),
	}

	// Count nodes by type
	for nodeType, nodes := range g.NodesByType {
		stats.NodesByType[nodeType] = len(nodes)
	}

	stats.FileCount = stats.NodesByType[NodeTypeFile]
	stats.PackageCount = stats.NodesByType[NodeTypePackage]
	stats.MessageCount = stats.NodesByType[NodeTypeMessage]
	stats.ServiceCount = stats.NodesByType[NodeTypeService]
	stats.EnumCount = stats.NodesByType[NodeTypeEnum]

	if stats.TotalNodes == 0 {
		return stats
	}

	// Calculate fan-out and fan-in metrics
	totalFanOut := 0
	totalFanIn := 0
	totalDepth := 0

	for nodeID := range g.Nodes {
		fanOut := len(g.GetOutgoingEdges(nodeID))
		fanIn := len(g.GetIncomingEdges(nodeID))
		depth := len(g.GetTransitiveDependencies(nodeID))

		totalFanOut += fanOut
		totalFanIn += fanIn
		totalDepth += depth

		if fanOut > stats.MaxFanOut {
			stats.MaxFanOut = fanOut
		}
		if fanIn > stats.MaxFanIn {
			stats.MaxFanIn = fanIn
		}
		if depth > stats.MaxDepth {
			stats.MaxDepth = depth
		}
	}

	stats.AvgFanOut = float64(totalFanOut) / float64(stats.TotalNodes)
	stats.AvgFanIn = float64(totalFanIn) / float64(stats.TotalNodes)
	stats.AvgDepth = float64(totalDepth) / float64(stats.TotalNodes)

	// Calculate density
	// For a directed graph: density = edges / (nodes * (nodes - 1))
	if stats.TotalNodes > 1 {
		possibleEdges := stats.TotalNodes * (stats.TotalNodes - 1)
		stats.Density = float64(stats.TotalEdges) / float64(possibleEdges)
	}

	// Calculate connected components using union-find
	stats.ConnectedComponents = countConnectedComponents(g)

	return stats
}

// countConnectedComponents counts the number of weakly connected components.
func countConnectedComponents(g *Graph) int {
	if len(g.Nodes) == 0 {
		return 0
	}

	visited := make(map[string]bool)
	components := 0

	// DFS to mark all reachable nodes
	var dfs func(nodeID string)
	dfs = func(nodeID string) {
		if visited[nodeID] {
			return
		}
		visited[nodeID] = true

		// Visit neighbors (both directions for weakly connected)
		for _, edge := range g.GetOutgoingEdges(nodeID) {
			dfs(edge.To)
		}
		for _, edge := range g.GetIncomingEdges(nodeID) {
			dfs(edge.From)
		}
	}

	for nodeID := range g.Nodes {
		if !visited[nodeID] {
			components++
			dfs(nodeID)
		}
	}

	return components
}

// TopNodes returns the top N nodes by a metric.
type NodeMetric struct {
	NodeID string
	Value  int
}

// GetTopByFanOut returns nodes with the most outgoing edges.
func GetTopByFanOut(g *Graph, limit int) []NodeMetric {
	metrics := make([]NodeMetric, 0, len(g.Nodes))

	for nodeID := range g.Nodes {
		fanOut := len(g.GetOutgoingEdges(nodeID))
		metrics = append(metrics, NodeMetric{NodeID: nodeID, Value: fanOut})
	}

	// Sort descending
	for i := 0; i < len(metrics)-1; i++ {
		for j := i + 1; j < len(metrics); j++ {
			if metrics[j].Value > metrics[i].Value {
				metrics[i], metrics[j] = metrics[j], metrics[i]
			}
		}
	}

	if limit > 0 && limit < len(metrics) {
		return metrics[:limit]
	}
	return metrics
}

// GetTopByFanIn returns nodes with the most incoming edges.
func GetTopByFanIn(g *Graph, limit int) []NodeMetric {
	metrics := make([]NodeMetric, 0, len(g.Nodes))

	for nodeID := range g.Nodes {
		fanIn := len(g.GetIncomingEdges(nodeID))
		metrics = append(metrics, NodeMetric{NodeID: nodeID, Value: fanIn})
	}

	// Sort descending
	for i := 0; i < len(metrics)-1; i++ {
		for j := i + 1; j < len(metrics); j++ {
			if metrics[j].Value > metrics[i].Value {
				metrics[i], metrics[j] = metrics[j], metrics[i]
			}
		}
	}

	if limit > 0 && limit < len(metrics) {
		return metrics[:limit]
	}
	return metrics
}

// GetTopByTransitiveDeps returns nodes with the most transitive dependencies.
func GetTopByTransitiveDeps(g *Graph, limit int) []NodeMetric {
	metrics := make([]NodeMetric, 0, len(g.Nodes))

	for nodeID := range g.Nodes {
		deps := len(g.GetTransitiveDependencies(nodeID))
		metrics = append(metrics, NodeMetric{NodeID: nodeID, Value: deps})
	}

	// Sort descending
	for i := 0; i < len(metrics)-1; i++ {
		for j := i + 1; j < len(metrics); j++ {
			if metrics[j].Value > metrics[i].Value {
				metrics[i], metrics[j] = metrics[j], metrics[i]
			}
		}
	}

	if limit > 0 && limit < len(metrics) {
		return metrics[:limit]
	}
	return metrics
}
