package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/massonsky/buffalo/internal/builder"
	"github.com/massonsky/buffalo/internal/config"
	"github.com/massonsky/buffalo/internal/graph"
	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/massonsky/buffalo/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	graphFormat    string
	graphOutput    string
	graphScope     string
	graphFile      string
	graphShowStats bool

	graphCmd = &cobra.Command{
		Use:   "graph",
		Short: "Visualize proto file dependencies",
		Long: `Generate dependency graphs and diagrams for proto files.

Supports multiple output formats: tree, dot, mermaid, json, plantuml.
Supports multiple scopes: file, package, message, service, full.

Examples:
  # Show dependency tree in terminal
  buffalo graph

  # Generate Graphviz DOT file
  buffalo graph --format dot --output deps.dot

  # Generate Mermaid diagram for README
  buffalo graph --format mermaid --output deps.md

  # Show package-level dependencies
  buffalo graph --scope package

  # Show message relationships
  buffalo graph --scope message

  # Analyze specific file
  buffalo graph --file protos/user.proto

  # Detect cycles
  buffalo graph analyze --cycles

  # Find orphan files
  buffalo graph analyze --orphans

  # Show coupling metrics
  buffalo graph analyze --coupling

  # Show graph statistics
  buffalo graph --stats`,
		RunE: runGraph,
	}

	graphAnalyzeCmd = &cobra.Command{
		Use:   "analyze",
		Short: "Analyze dependency graph",
		Long: `Perform various analyzes on the dependency graph.

Examples:
  buffalo graph analyze --cycles     # Find circular dependencies
  buffalo graph analyze --orphans    # Find unused files
  buffalo graph analyze --heavy      # Find files with most dependencies
  buffalo graph analyze --coupling   # Calculate coupling metrics
  buffalo graph analyze --layers     # Suggest architectural layers`,
		RunE: runGraphAnalyze,
	}

	graphStatsCmd = &cobra.Command{
		Use:   "stats",
		Short: "Show graph statistics",
		Long: `Display comprehensive statistics about the dependency graph.

Examples:
  buffalo graph stats`,
		RunE: runGraphStats,
	}
)

func init() {
	rootCmd.AddCommand(graphCmd)
	graphCmd.AddCommand(graphAnalyzeCmd)
	graphCmd.AddCommand(graphStatsCmd)

	// Graph command flags
	graphCmd.Flags().StringVarP(&graphFormat, "format", "f", "tree", "output format (tree, dot, mermaid, json, plantuml)")
	graphCmd.Flags().StringVarP(&graphOutput, "output", "o", "", "output file (default: stdout)")
	graphCmd.Flags().StringVarP(&graphScope, "scope", "s", "file", "analysis scope (file, package, message, service, full)")
	graphCmd.Flags().StringVar(&graphFile, "file", "", "analyze specific proto file")
	graphCmd.Flags().BoolVar(&graphShowStats, "stats", false, "show graph statistics")

	// Analyze command flags
	graphAnalyzeCmd.Flags().Bool("cycles", false, "detect circular dependencies")
	graphAnalyzeCmd.Flags().Bool("orphans", false, "find unused/orphan files")
	graphAnalyzeCmd.Flags().Bool("heavy", false, "find files with most dependencies")
	graphAnalyzeCmd.Flags().Bool("coupling", false, "calculate coupling metrics")
	graphAnalyzeCmd.Flags().Bool("layers", false, "suggest architectural layers")
	graphAnalyzeCmd.Flags().Bool("all", false, "run all analyzes")
}

func runGraph(cmd *cobra.Command, args []string) error {
	log := GetLogger()
	ctx := context.Background()

	log.Info("📊 Generating dependency graph")

	// Load configuration
	cfg, err := loadConfig(log)
	if err != nil {
		log.Warn("Failed to load config, using defaults", logger.Any("error", err))
		cfg = getDefaultConfig()
	}

	// Build dependency graph
	depGraph, err := buildDependencyGraph(ctx, log, cfg)
	if err != nil {
		return fmt.Errorf("failed to build dependency graph: %w", err)
	}

	// Convert scope string to graph.Scope
	scope := parseScope(graphScope)

	// Build enhanced graph
	logAdapter := builder.NewLoggerAdapter(log)
	graphBuilder := graph.NewBuilder(logAdapter)
	g, err := graphBuilder.BuildFromDependencyGraph(depGraph, scope)
	if err != nil {
		return fmt.Errorf("failed to build graph: %w", err)
	}

	// If specific file requested, filter to show its context
	if graphFile != "" {
		g = filterGraphForFile(g, graphFile)
	}

	// Show stats if requested
	if graphShowStats {
		return runGraphStatsInternal(g)
	}

	// Render graph
	format := parseFormat(graphFormat)
	renderer := graph.NewRenderer(format)

	var output bytes.Buffer
	if err := renderer.Render(&output, g); err != nil {
		return fmt.Errorf("failed to render graph: %w", err)
	}

	// Write output
	if graphOutput != "" {
		// Ensure directory exists
		dir := filepath.Dir(graphOutput)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		if err := os.WriteFile(graphOutput, output.Bytes(), 0600); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		log.Info("Graph written", logger.String("file", graphOutput), logger.String("format", graphFormat))
	} else {
		fmt.Println(output.String())
	}

	return nil
}

func runGraphAnalyze(cmd *cobra.Command, args []string) error {
	log := GetLogger()
	ctx := context.Background()

	log.Info("🔍 Analyzing dependency graph")

	// Load configuration
	cfg, err := loadConfig(log)
	if err != nil {
		log.Warn("Failed to load config, using defaults", logger.Any("error", err))
		cfg = getDefaultConfig()
	}

	// Build dependency graph
	depGraph, err := buildDependencyGraph(ctx, log, cfg)
	if err != nil {
		return fmt.Errorf("failed to build dependency graph: %w", err)
	}

	// Build enhanced graph
	logAdapter := builder.NewLoggerAdapter(log)
	graphBuilder := graph.NewBuilder(logAdapter)
	g, err := graphBuilder.BuildFromDependencyGraph(depGraph, graph.ScopeFile)
	if err != nil {
		return fmt.Errorf("failed to build graph: %w", err)
	}

	analyzer := graph.NewAnalyzer(g)

	// Check flags
	runAll, _ := cmd.Flags().GetBool("all")
	checkCycles, _ := cmd.Flags().GetBool("cycles")
	checkOrphans, _ := cmd.Flags().GetBool("orphans")
	checkHeavy, _ := cmd.Flags().GetBool("heavy")
	checkCoupling, _ := cmd.Flags().GetBool("coupling")
	checkLayers, _ := cmd.Flags().GetBool("layers")

	// If no specific flag, run all
	if !checkCycles && !checkOrphans && !checkHeavy && !checkCoupling && !checkLayers {
		runAll = true
	}

	fmt.Println()

	if runAll || checkCycles {
		printCycleAnalysis(analyzer)
	}

	if runAll || checkOrphans {
		printOrphanAnalysis(analyzer)
	}

	if runAll || checkHeavy {
		printHeavyNodesAnalysis(analyzer)
	}

	if runAll || checkCoupling {
		printCouplingAnalysis(analyzer)
	}

	if runAll || checkLayers {
		printLayersAnalysis(analyzer)
	}

	return nil
}

func runGraphStats(cmd *cobra.Command, args []string) error {
	log := GetLogger()
	ctx := context.Background()

	log.Info("📈 Calculating graph statistics")

	// Load configuration
	cfg, err := loadConfig(log)
	if err != nil {
		log.Warn("Failed to load config, using defaults", logger.Any("error", err))
		cfg = getDefaultConfig()
	}

	// Build dependency graph
	depGraph, err := buildDependencyGraph(ctx, log, cfg)
	if err != nil {
		return fmt.Errorf("failed to build dependency graph: %w", err)
	}

	// Build enhanced graph
	logAdapter := builder.NewLoggerAdapter(log)
	graphBuilder := graph.NewBuilder(logAdapter)
	g, err := graphBuilder.BuildFromDependencyGraph(depGraph, graph.ScopeFile)
	if err != nil {
		return fmt.Errorf("failed to build graph: %w", err)
	}

	return runGraphStatsInternal(g)
}

func runGraphStatsInternal(g *graph.Graph) error {
	stats := graph.CalculateStats(g)

	fmt.Println()
	fmt.Println("📊 Graph Statistics")
	fmt.Println("═══════════════════════════════════════")
	fmt.Println()

	fmt.Printf("  Total Nodes:            %d\n", stats.TotalNodes)
	fmt.Printf("  Total Edges:            %d\n", stats.TotalEdges)
	fmt.Println()

	fmt.Println("  Nodes by Type:")
	for nodeType, count := range stats.NodesByType {
		fmt.Printf("    %-20s  %d\n", nodeType+":", count)
	}
	fmt.Println()

	fmt.Println("  Metrics:")
	fmt.Printf("    Max Depth:            %d\n", stats.MaxDepth)
	fmt.Printf("    Avg Depth:            %.2f\n", stats.AvgDepth)
	fmt.Printf("    Max Fan-Out:          %d\n", stats.MaxFanOut)
	fmt.Printf("    Avg Fan-Out:          %.2f\n", stats.AvgFanOut)
	fmt.Printf("    Max Fan-In:           %d\n", stats.MaxFanIn)
	fmt.Printf("    Avg Fan-In:           %.2f\n", stats.AvgFanIn)
	fmt.Printf("    Density:              %.4f\n", stats.Density)
	fmt.Printf("    Connected Components: %d\n", stats.ConnectedComponents)
	fmt.Println()

	// Top nodes by fan-out
	fmt.Println("  Top 5 by Fan-Out (most dependencies):")
	topFanOut := graph.GetTopByFanOut(g, 5)
	for i, m := range topFanOut {
		if m.Value > 0 {
			fmt.Printf("    %d. %s (%d)\n", i+1, m.NodeID, m.Value)
		}
	}
	fmt.Println()

	// Top nodes by fan-in
	fmt.Println("  Top 5 by Fan-In (most dependents):")
	topFanIn := graph.GetTopByFanIn(g, 5)
	for i, m := range topFanIn {
		if m.Value > 0 {
			fmt.Printf("    %d. %s (%d)\n", i+1, m.NodeID, m.Value)
		}
	}
	fmt.Println()

	return nil
}

func buildDependencyGraph(ctx context.Context, log *logger.Logger, cfg *config.Config) (*builder.DependencyGraph, error) {
	logAdapter := builder.NewLoggerAdapter(log)

	// Find proto files
	var protoFiles []string
	for _, path := range cfg.Proto.Paths {
		fileInfos, err := utils.FindFiles(path, utils.FindFilesOptions{
			Pattern:   "*.proto",
			Recursive: true,
		})
		if err != nil {
			log.Warn("Failed to find proto files", logger.String("path", path), logger.Any("error", err))
			continue
		}
		for _, fi := range fileInfos {
			protoFiles = append(protoFiles, fi.Path)
		}
	}

	if len(protoFiles) == 0 {
		return nil, fmt.Errorf("no proto files found in paths: %v", cfg.Proto.Paths)
	}

	// Parse proto files
	parser := builder.NewProtoParser(logAdapter)
	parsedFiles, err := parser.ParseFiles(ctx, protoFiles, cfg.Proto.Paths)
	if err != nil {
		return nil, fmt.Errorf("failed to parse proto files: %w", err)
	}

	// Resolve dependencies
	resolver := builder.NewDependencyResolver(logAdapter)
	depGraph, err := resolver.Resolve(ctx, parsedFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	return depGraph, nil
}

func parseScope(s string) graph.Scope {
	switch strings.ToLower(s) {
	case "file":
		return graph.ScopeFile
	case "package":
		return graph.ScopePackage
	case "message":
		return graph.ScopeMessage
	case "service":
		return graph.ScopeService
	case "full":
		return graph.ScopeFull
	default:
		return graph.ScopeFile
	}
}

func parseFormat(f string) graph.Format {
	switch strings.ToLower(f) {
	case "tree":
		return graph.FormatTree
	case "dot":
		return graph.FormatDOT
	case "mermaid":
		return graph.FormatMermaid
	case "json":
		return graph.FormatJSON
	case "plantuml":
		return graph.FormatPlantUML
	default:
		return graph.FormatTree
	}
}

func filterGraphForFile(g *graph.Graph, filePath string) *graph.Graph {
	filtered := graph.NewGraph()

	// Normalize path
	filePath = strings.TrimPrefix(filePath, "./")

	// Add the target node
	targetNode := g.GetNode(filePath)
	if targetNode == nil {
		// Try to find by name
		for id, node := range g.Nodes {
			if strings.HasSuffix(id, filePath) || strings.HasSuffix(id, "/"+filePath) {
				targetNode = node
				filePath = id
				break
			}
		}
	}

	if targetNode == nil {
		return g // Return full graph if file not found
	}

	filtered.AddNode(targetNode)

	// Add dependencies
	deps := g.GetTransitiveDependencies(filePath)
	for _, dep := range deps {
		if node := g.GetNode(dep); node != nil {
			filtered.AddNode(node)
		}
	}

	// Add dependents
	dependents := g.GetTransitiveDependents(filePath)
	for _, dep := range dependents {
		if node := g.GetNode(dep); node != nil {
			filtered.AddNode(node)
		}
	}

	// Add relevant edges
	for _, edge := range g.Edges {
		if filtered.GetNode(edge.From) != nil && filtered.GetNode(edge.To) != nil {
			filtered.AddEdge(edge)
		}
	}

	return filtered
}

func printCycleAnalysis(analyzer *graph.Analyzer) {
	fmt.Println("🔄 Cycle Analysis")
	fmt.Println("─────────────────────────────────────")

	cycles := analyzer.DetectCycles()
	if len(cycles) == 0 {
		fmt.Println("✅ No circular dependencies detected")
	} else {
		fmt.Printf("❌ Found %d circular dependency(ies):\n", len(cycles))
		for i, cycle := range cycles {
			fmt.Printf("  %d. %s\n", i+1, strings.Join(cycle.Nodes, " → "))
		}
	}
	fmt.Println()
}

func printOrphanAnalysis(analyzer *graph.Analyzer) {
	fmt.Println("📦 Orphan Analysis")
	fmt.Println("─────────────────────────────────────")

	unused := analyzer.FindUnusedFiles()
	orphans := analyzer.FindOrphans()

	if len(unused) == 0 && len(orphans) == 0 {
		fmt.Println("✅ No orphan or unused files detected")
	} else {
		if len(unused) > 0 {
			fmt.Printf("⚠️  Files not imported by others (%d):\n", len(unused))
			for _, f := range unused {
				fmt.Printf("  - %s\n", f)
			}
		}
		if len(orphans) > 0 {
			fmt.Printf("⚠️  Completely isolated files (%d):\n", len(orphans))
			for _, f := range orphans {
				fmt.Printf("  - %s\n", f)
			}
		}
	}
	fmt.Println()
}

func printHeavyNodesAnalysis(analyzer *graph.Analyzer) {
	fmt.Println("🏋️ Heavy Nodes Analysis")
	fmt.Println("─────────────────────────────────────")

	heavy := analyzer.FindHeavyNodes(10)
	if len(heavy) == 0 {
		fmt.Println("No nodes found")
	} else {
		fmt.Println("Top files by total connections (dependencies + dependents):")
		for i, node := range heavy {
			if i >= 10 {
				break
			}
			total := node.TransitiveDependencies + node.TransitiveDependents
			fmt.Printf("  %d. %s\n", i+1, node.NodeID)
			fmt.Printf("     Dependencies: %d, Dependents: %d, Total: %d\n",
				node.TransitiveDependencies, node.TransitiveDependents, total)
		}
	}
	fmt.Println()
}

func printCouplingAnalysis(analyzer *graph.Analyzer) {
	fmt.Println("🔗 Coupling Analysis")
	fmt.Println("─────────────────────────────────────")
	fmt.Println("Ca = Afferent (incoming), Ce = Efferent (outgoing)")
	fmt.Println("I = Instability (Ce/(Ca+Ce)), 0=stable, 1=unstable")
	fmt.Println()

	metrics := analyzer.CalculateCoupling()

	// Sort by instability (descending)
	for i := 0; i < len(metrics)-1; i++ {
		for j := i + 1; j < len(metrics); j++ {
			if metrics[j].Instability > metrics[i].Instability {
				metrics[i], metrics[j] = metrics[j], metrics[i]
			}
		}
	}

	// Show top unstable
	fmt.Println("Most unstable (many deps, few dependents):")
	count := 0
	for _, m := range metrics {
		if m.AfferentCoupling+m.EfferentCoupling > 0 && count < 5 {
			fmt.Printf("  %s: Ca=%d, Ce=%d, I=%.2f\n",
				m.NodeID, m.AfferentCoupling, m.EfferentCoupling, m.Instability)
			count++
		}
	}

	fmt.Println()

	// Show top stable
	fmt.Println("Most stable (few deps, many dependents):")
	count = 0
	for i := len(metrics) - 1; i >= 0 && count < 5; i-- {
		m := metrics[i]
		if m.AfferentCoupling+m.EfferentCoupling > 0 {
			fmt.Printf("  %s: Ca=%d, Ce=%d, I=%.2f\n",
				m.NodeID, m.AfferentCoupling, m.EfferentCoupling, m.Instability)
			count++
		}
	}
	fmt.Println()
}

func printLayersAnalysis(analyzer *graph.Analyzer) {
	fmt.Println("📚 Layer Analysis")
	fmt.Println("─────────────────────────────────────")
	fmt.Println("Suggested architectural layers (based on dependency depth):")
	fmt.Println()

	layers := analyzer.SuggestLayers()

	for _, layer := range layers {
		fmt.Printf("Level %d (%s): %d file(s)\n", layer.Level, layer.Name, len(layer.Nodes))
		for _, node := range layer.Nodes {
			fmt.Printf("  - %s\n", node)
		}
		fmt.Println()
	}
}
