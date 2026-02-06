// Package graph provides dependency graph analysis and visualization for proto files.
package graph

import (
	"sort"
	"strings"

	"github.com/massonsky/buffalo/internal/builder"
	"github.com/massonsky/buffalo/pkg/errors"
)

// Scope defines the level of graph analysis.
type Scope string

const (
	// ScopeFile shows dependencies between files.
	ScopeFile Scope = "file"
	// ScopePackage shows dependencies between packages.
	ScopePackage Scope = "package"
	// ScopeMessage shows relationships between messages.
	ScopeMessage Scope = "message"
	// ScopeService shows service to message mappings.
	ScopeService Scope = "service"
	// ScopeFull shows all relationships.
	ScopeFull Scope = "full"
)

// NodeType represents the type of a graph node.
type NodeType string

const (
	NodeTypeFile    NodeType = "file"
	NodeTypePackage NodeType = "package"
	NodeTypeMessage NodeType = "message"
	NodeTypeService NodeType = "service"
	NodeTypeEnum    NodeType = "enum"
)

// Node represents a node in the enhanced dependency graph.
type Node struct {
	// ID is the unique identifier for the node.
	ID string
	// Type is the type of node.
	Type NodeType
	// Name is the display name.
	Name string
	// Package is the proto package name (for file nodes).
	Package string
	// FilePath is the file path (for non-file nodes, indicates source file).
	FilePath string
	// Metadata contains additional node information.
	Metadata map[string]interface{}
}

// Edge represents a directed edge in the graph.
type Edge struct {
	// From is the source node ID.
	From string
	// To is the target node ID.
	To string
	// Type describes the relationship.
	Type string
	// Weight is optional edge weight (for analysis).
	Weight int
}

// Graph represents an enhanced dependency graph with metadata.
type Graph struct {
	// Nodes are all nodes in the graph.
	Nodes map[string]*Node
	// Edges are all directed edges.
	Edges []*Edge
	// NodesByType indexes nodes by type.
	NodesByType map[NodeType][]*Node
	// AdjacencyList maps node ID to outgoing edges.
	AdjacencyList map[string][]*Edge
	// ReverseAdjacency maps node ID to incoming edges.
	ReverseAdjacency map[string][]*Edge
}

// NewGraph creates a new empty graph.
func NewGraph() *Graph {
	return &Graph{
		Nodes:            make(map[string]*Node),
		Edges:            make([]*Edge, 0),
		NodesByType:      make(map[NodeType][]*Node),
		AdjacencyList:    make(map[string][]*Edge),
		ReverseAdjacency: make(map[string][]*Edge),
	}
}

// AddNode adds a node to the graph.
func (g *Graph) AddNode(node *Node) {
	if node == nil {
		return
	}
	g.Nodes[node.ID] = node
	g.NodesByType[node.Type] = append(g.NodesByType[node.Type], node)
}

// AddEdge adds an edge to the graph.
func (g *Graph) AddEdge(edge *Edge) {
	if edge == nil {
		return
	}
	g.Edges = append(g.Edges, edge)
	g.AdjacencyList[edge.From] = append(g.AdjacencyList[edge.From], edge)
	g.ReverseAdjacency[edge.To] = append(g.ReverseAdjacency[edge.To], edge)
}

// GetNode returns a node by ID.
func (g *Graph) GetNode(id string) *Node {
	return g.Nodes[id]
}

// GetOutgoingEdges returns edges originating from a node.
func (g *Graph) GetOutgoingEdges(nodeID string) []*Edge {
	return g.AdjacencyList[nodeID]
}

// GetIncomingEdges returns edges pointing to a node.
func (g *Graph) GetIncomingEdges(nodeID string) []*Edge {
	return g.ReverseAdjacency[nodeID]
}

// GetDependencies returns node IDs that the given node depends on.
func (g *Graph) GetDependencies(nodeID string) []string {
	edges := g.GetOutgoingEdges(nodeID)
	result := make([]string, 0, len(edges))
	for _, e := range edges {
		result = append(result, e.To)
	}
	return result
}

// GetDependents returns node IDs that depend on the given node.
func (g *Graph) GetDependents(nodeID string) []string {
	edges := g.GetIncomingEdges(nodeID)
	result := make([]string, 0, len(edges))
	for _, e := range edges {
		result = append(result, e.From)
	}
	return result
}

// GetTransitiveDependencies returns all transitive dependencies.
func (g *Graph) GetTransitiveDependencies(nodeID string) []string {
	visited := make(map[string]bool)
	var result []string

	var visit func(id string)
	visit = func(id string) {
		for _, dep := range g.GetDependencies(id) {
			if !visited[dep] {
				visited[dep] = true
				result = append(result, dep)
				visit(dep)
			}
		}
	}

	visit(nodeID)
	return result
}

// GetTransitiveDependents returns all nodes that transitively depend on this node.
func (g *Graph) GetTransitiveDependents(nodeID string) []string {
	visited := make(map[string]bool)
	var result []string

	var visit func(id string)
	visit = func(id string) {
		for _, dep := range g.GetDependents(id) {
			if !visited[dep] {
				visited[dep] = true
				result = append(result, dep)
				visit(dep)
			}
		}
	}

	visit(nodeID)
	return result
}

// FilterByType returns a subgraph containing only nodes of the specified type.
func (g *Graph) FilterByType(nodeType NodeType) *Graph {
	filtered := NewGraph()

	// Add nodes of the specified type
	nodeSet := make(map[string]bool)
	for _, node := range g.NodesByType[nodeType] {
		filtered.AddNode(node)
		nodeSet[node.ID] = true
	}

	// Add edges between included nodes
	for _, edge := range g.Edges {
		if nodeSet[edge.From] && nodeSet[edge.To] {
			filtered.AddEdge(edge)
		}
	}

	return filtered
}

// SortedNodes returns nodes sorted by ID.
func (g *Graph) SortedNodes() []*Node {
	nodes := make([]*Node, 0, len(g.Nodes))
	for _, n := range g.Nodes {
		nodes = append(nodes, n)
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})
	return nodes
}

// Builder builds graphs from proto files.
type Builder struct {
	log builder.Logger
}

// NewBuilder creates a new graph builder.
func NewBuilder(log builder.Logger) *Builder {
	return &Builder{log: log}
}

// BuildFromDependencyGraph creates an enhanced graph from a DependencyGraph.
func (b *Builder) BuildFromDependencyGraph(depGraph *builder.DependencyGraph, scope Scope) (*Graph, error) {
	if depGraph == nil {
		return nil, errors.New(errors.ErrInvalidInput, "dependency graph is nil")
	}

	b.log.Debug("Building graph", "scope", string(scope), "nodes", len(depGraph.Nodes))

	graph := NewGraph()

	switch scope {
	case ScopeFile:
		b.buildFileGraph(graph, depGraph)
	case ScopePackage:
		b.buildPackageGraph(graph, depGraph)
	case ScopeMessage:
		b.buildMessageGraph(graph, depGraph)
	case ScopeService:
		b.buildServiceGraph(graph, depGraph)
	case ScopeFull:
		b.buildFullGraph(graph, depGraph)
	default:
		b.buildFileGraph(graph, depGraph)
	}

	return graph, nil
}

// buildFileGraph builds a file-level dependency graph.
func (b *Builder) buildFileGraph(graph *Graph, depGraph *builder.DependencyGraph) {
	// Add file nodes
	for path, protoFile := range depGraph.Nodes {
		node := &Node{
			ID:       path,
			Type:     NodeTypeFile,
			Name:     extractFileName(path),
			Package:  protoFile.Package,
			FilePath: path,
			Metadata: map[string]interface{}{
				"syntax":   protoFile.Syntax,
				"messages": len(protoFile.Messages),
				"services": len(protoFile.Services),
				"enums":    len(protoFile.Enums),
			},
		}
		graph.AddNode(node)
	}

	// Add import edges
	for path, imports := range depGraph.Edges {
		for _, imp := range imports {
			edge := &Edge{
				From:   path,
				To:     imp,
				Type:   "imports",
				Weight: 1,
			}
			graph.AddEdge(edge)
		}
	}
}

// buildPackageGraph builds a package-level dependency graph.
func (b *Builder) buildPackageGraph(graph *Graph, depGraph *builder.DependencyGraph) {
	// Collect packages
	packages := make(map[string]*Node)
	fileToPackage := make(map[string]string)

	for path, protoFile := range depGraph.Nodes {
		pkg := protoFile.Package
		if pkg == "" {
			pkg = "(default)"
		}
		fileToPackage[path] = pkg

		if _, exists := packages[pkg]; !exists {
			node := &Node{
				ID:      pkg,
				Type:    NodeTypePackage,
				Name:    pkg,
				Package: pkg,
				Metadata: map[string]interface{}{
					"files": []string{},
				},
			}
			packages[pkg] = node
		}

		// Track files in package
		files := packages[pkg].Metadata["files"].([]string)
		packages[pkg].Metadata["files"] = append(files, path)
	}

	// Add package nodes
	for _, node := range packages {
		graph.AddNode(node)
	}

	// Add package edges (deduplicated)
	edgeSet := make(map[string]bool)
	for path, imports := range depGraph.Edges {
		fromPkg := fileToPackage[path]
		for _, imp := range imports {
			if toPkg, exists := fileToPackage[imp]; exists && fromPkg != toPkg {
				edgeKey := fromPkg + "->" + toPkg
				if !edgeSet[edgeKey] {
					edgeSet[edgeKey] = true
					edge := &Edge{
						From:   fromPkg,
						To:     toPkg,
						Type:   "depends",
						Weight: 1,
					}
					graph.AddEdge(edge)
				}
			}
		}
	}
}

// buildMessageGraph builds a message-level relationship graph.
func (b *Builder) buildMessageGraph(graph *Graph, depGraph *builder.DependencyGraph) {
	// Collect all messages and their field types
	for filePath, protoFile := range depGraph.Nodes {
		for _, msg := range protoFile.Messages {
			msgID := protoFile.Package + "." + msg.Name
			node := &Node{
				ID:       msgID,
				Type:     NodeTypeMessage,
				Name:     msg.Name,
				Package:  protoFile.Package,
				FilePath: filePath,
				Metadata: map[string]interface{}{
					"fields": len(msg.Fields),
				},
			}
			graph.AddNode(node)
		}
	}

	// Add edges for field type references
	for _, protoFile := range depGraph.Nodes {
		for _, msg := range protoFile.Messages {
			fromID := protoFile.Package + "." + msg.Name
			for _, field := range msg.Fields {
				// Check if field type references another message
				fieldType := field.Type
				// Try full qualified name first
				if _, exists := graph.Nodes[fieldType]; exists {
					edge := &Edge{
						From:   fromID,
						To:     fieldType,
						Type:   "references",
						Weight: 1,
					}
					graph.AddEdge(edge)
				} else {
					// Try with current package prefix
					qualifiedType := protoFile.Package + "." + fieldType
					if _, exists := graph.Nodes[qualifiedType]; exists {
						edge := &Edge{
							From:   fromID,
							To:     qualifiedType,
							Type:   "references",
							Weight: 1,
						}
						graph.AddEdge(edge)
					}
				}
			}
		}
	}
}

// buildServiceGraph builds a service to message relationship graph.
func (b *Builder) buildServiceGraph(graph *Graph, depGraph *builder.DependencyGraph) {
	// Add service nodes
	for path, protoFile := range depGraph.Nodes {
		for _, svc := range protoFile.Services {
			svcID := protoFile.Package + "." + svc.Name
			node := &Node{
				ID:       svcID,
				Type:     NodeTypeService,
				Name:     svc.Name,
				Package:  protoFile.Package,
				FilePath: path,
				Metadata: map[string]interface{}{
					"methods": len(svc.Methods),
				},
			}
			graph.AddNode(node)

			// Add edges to request/response types
			for _, method := range svc.Methods {
				// Input type
				inputID := resolveTypeName(protoFile.Package, method.InputType)
				edge := &Edge{
					From:   svcID,
					To:     inputID,
					Type:   "request",
					Weight: 1,
				}
				graph.AddEdge(edge)

				// Output type
				outputID := resolveTypeName(protoFile.Package, method.OutputType)
				edge = &Edge{
					From:   svcID,
					To:     outputID,
					Type:   "response",
					Weight: 1,
				}
				graph.AddEdge(edge)
			}
		}
	}

	// Add message nodes that are referenced by services
	for filePath, protoFile := range depGraph.Nodes {
		for _, msg := range protoFile.Messages {
			msgID := protoFile.Package + "." + msg.Name
			if _, exists := graph.Nodes[msgID]; !exists {
				node := &Node{
					ID:       msgID,
					Type:     NodeTypeMessage,
					Name:     msg.Name,
					Package:  protoFile.Package,
					FilePath: filePath,
					Metadata: map[string]interface{}{
						"fields": len(msg.Fields),
					},
				}
				graph.AddNode(node)
			}
		}
	}
}

// buildFullGraph builds a comprehensive graph with all relationships.
func (b *Builder) buildFullGraph(graph *Graph, depGraph *builder.DependencyGraph) {
	// Add all file nodes
	for path, protoFile := range depGraph.Nodes {
		fileNode := &Node{
			ID:       "file:" + path,
			Type:     NodeTypeFile,
			Name:     extractFileName(path),
			Package:  protoFile.Package,
			FilePath: path,
			Metadata: map[string]interface{}{
				"syntax": protoFile.Syntax,
			},
		}
		graph.AddNode(fileNode)

		// Add messages
		for _, msg := range protoFile.Messages {
			msgID := "msg:" + protoFile.Package + "." + msg.Name
			msgNode := &Node{
				ID:       msgID,
				Type:     NodeTypeMessage,
				Name:     msg.Name,
				Package:  protoFile.Package,
				FilePath: path,
				Metadata: map[string]interface{}{
					"fields": len(msg.Fields),
				},
			}
			graph.AddNode(msgNode)

			// Message belongs to file
			graph.AddEdge(&Edge{From: msgID, To: "file:" + path, Type: "defined_in"})
		}

		// Add services
		for _, svc := range protoFile.Services {
			svcID := "svc:" + protoFile.Package + "." + svc.Name
			svcNode := &Node{
				ID:       svcID,
				Type:     NodeTypeService,
				Name:     svc.Name,
				Package:  protoFile.Package,
				FilePath: path,
				Metadata: map[string]interface{}{
					"methods": len(svc.Methods),
				},
			}
			graph.AddNode(svcNode)

			// Service belongs to file
			graph.AddEdge(&Edge{From: svcID, To: "file:" + path, Type: "defined_in"})
		}

		// Add enums
		for _, enum := range protoFile.Enums {
			enumID := "enum:" + protoFile.Package + "." + enum.Name
			enumNode := &Node{
				ID:       enumID,
				Type:     NodeTypeEnum,
				Name:     enum.Name,
				Package:  protoFile.Package,
				FilePath: path,
				Metadata: map[string]interface{}{
					"values": len(enum.Values),
				},
			}
			graph.AddNode(enumNode)

			// Enum belongs to file
			graph.AddEdge(&Edge{From: enumID, To: "file:" + path, Type: "defined_in"})
		}
	}

	// Add file import edges
	for path, imports := range depGraph.Edges {
		for _, imp := range imports {
			graph.AddEdge(&Edge{
				From: "file:" + path,
				To:   "file:" + imp,
				Type: "imports",
			})
		}
	}
}

// Helper functions

func extractFileName(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

func resolveTypeName(currentPackage, typeName string) string {
	if strings.Contains(typeName, ".") {
		return typeName
	}
	return currentPackage + "." + typeName
}
