package builder

import (
	"context"
	"fmt"

	"github.com/massonsky/buffalo/pkg/errors"
)

// DependencyGraph represents the dependency graph of proto files
type DependencyGraph struct {
	// Nodes are the proto files
	Nodes map[string]*ProtoFile

	// Edges represent dependencies (file -> dependencies)
	Edges map[string][]string

	// CompilationOrder is the topologically sorted order
	CompilationOrder []string
}

// DependencyResolver resolves proto file dependencies
type DependencyResolver interface {
	// Resolve builds the dependency graph and determines compilation order
	Resolve(ctx context.Context, files []*ProtoFile) (*DependencyGraph, error)
}

// dependencyResolver implements DependencyResolver
type dependencyResolver struct {
	log Logger
}

// NewDependencyResolver creates a new DependencyResolver
func NewDependencyResolver(log Logger) DependencyResolver {
	return &dependencyResolver{log: log}
}

// Resolve builds the dependency graph
func (r *dependencyResolver) Resolve(ctx context.Context, files []*ProtoFile) (*DependencyGraph, error) {
	r.log.Debug("Resolving dependencies", "files", len(files))

	graph := &DependencyGraph{
		Nodes: make(map[string]*ProtoFile),
		Edges: make(map[string][]string),
	}

	// Build nodes
	for _, file := range files {
		graph.Nodes[file.Path] = file
		graph.Edges[file.Path] = file.Imports
	}

	// Compute compilation order (topological sort)
	order, err := r.topologicalSort(graph)
	if err != nil {
		return nil, err
	}
	graph.CompilationOrder = order

	r.log.Debug("Dependencies resolved", "order", order)

	return graph, nil
}

// topologicalSort performs topological sorting on the dependency graph
func (r *dependencyResolver) topologicalSort(graph *DependencyGraph) ([]string, error) {
	// Kahn's algorithm for topological sorting
	// Edges[node] = list of dependencies (node depends on each dep)
	// So dep must come before node in the order
	inDegree := make(map[string]int)

	// Initialize in-degrees
	for node := range graph.Nodes {
		inDegree[node] = 0
	}

	// Build reverse edges: dep -> list of nodes that depend on dep
	dependents := make(map[string][]string)
	for node, deps := range graph.Edges {
		for _, dep := range deps {
			if _, exists := graph.Nodes[dep]; exists {
				inDegree[node]++
				dependents[dep] = append(dependents[dep], node)
			}
		}
	}

	// Find nodes with in-degree 0 (no dependencies)
	queue := []string{}
	for node, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, node)
		}
	}

	result := []string{}

	for len(queue) > 0 {
		// Dequeue
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		// Reduce in-degree for nodes that depend on this node
		for _, dependent := range dependents[node] {
			if _, exists := graph.Nodes[dependent]; !exists {
				continue
			}
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	// Check for cycles
	if len(result) != len(graph.Nodes) {
		return nil, errors.New(errors.ErrDependency, "circular dependency detected in proto files")
	}

	return result, nil
}

// GetDependencies returns direct dependencies of a file
func (g *DependencyGraph) GetDependencies(path string) []string {
	return g.Edges[path]
}

// GetTransitiveDependencies returns all transitive dependencies
func (g *DependencyGraph) GetTransitiveDependencies(path string) []string {
	visited := make(map[string]bool)
	var result []string

	var visit func(string)
	visit = func(p string) {
		if visited[p] {
			return
		}
		visited[p] = true

		for _, dep := range g.Edges[p] {
			if !visited[dep] {
				result = append(result, dep)
				visit(dep)
			}
		}
	}

	visit(path)
	return result
}

// Validate checks if the graph is valid
func (g *DependencyGraph) Validate() error {
	// Check for missing dependencies
	for file, deps := range g.Edges {
		for _, dep := range deps {
			if _, exists := g.Nodes[dep]; !exists {
				return errors.New(
					errors.ErrDependency,
					fmt.Sprintf("file %s depends on missing file %s", file, dep),
				)
			}
		}
	}

	return nil
}
