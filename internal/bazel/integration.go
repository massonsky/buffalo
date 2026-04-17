package bazel

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Integrator is the main entry point for Buffalo ↔ Bazel integration.
// It detects a Bazel workspace, discovers proto_library targets,
// resolves proto files, and after compilation generates BUILD.bazel files.
type Integrator struct {
	workspace  *Workspace
	querier    *BazelQuerier
	generator  *Generator
	useBazelQuery bool // whether to use `bazel query` or file-based parsing
}

// NewIntegrator creates an Integrator for the given directory.
// Returns (nil, nil) if no Bazel workspace is detected.
func NewIntegrator(dir string) (*Integrator, error) {
	ws, err := DetectWorkspace(dir)
	if err != nil {
		return nil, fmt.Errorf("bazel: detect workspace: %w", err)
	}
	if ws == nil {
		return nil, nil
	}

	querier := NewQuerier(ws.Root)
	generator := NewGenerator(ws.Root, ws.ModuleName)

	return &Integrator{
		workspace:  ws,
		querier:    querier,
		generator:  generator,
		useBazelQuery: IsBazelAvailable(),
	}, nil
}

// NewIntegratorFromWorkspace creates an Integrator from an already-detected workspace.
func NewIntegratorFromWorkspace(ws *Workspace) *Integrator {
	return &Integrator{
		workspace:  ws,
		querier:    NewQuerier(ws.Root),
		generator:  NewGenerator(ws.Root, ws.ModuleName),
		useBazelQuery: IsBazelAvailable(),
	}
}

// GetWorkspace returns the detected workspace.
func (it *Integrator) GetWorkspace() *Workspace {
	return it.workspace
}

// SetBazelPath sets a custom bazel binary path.
func (it *Integrator) SetBazelPath(path string) {
	it.querier.SetBazelPath(path)
}

// DiscoverProtoTargets finds all proto_library targets in the workspace.
// Uses `bazel query` when available, otherwise falls back to file parsing.
func (it *Integrator) DiscoverProtoTargets(ctx context.Context, patterns []string) ([]BazelTarget, error) {
	if it.useBazelQuery {
		return it.discoverViaQuery(ctx, patterns)
	}
	return it.discoverViaFiles(ctx)
}

// discoverViaQuery uses `bazel query kind(proto_library, //...)`.
func (it *Integrator) discoverViaQuery(ctx context.Context, patterns []string) ([]BazelTarget, error) {
	if len(patterns) == 0 {
		patterns = []string{"//..."}
	}

	var allTargets []BazelTarget
	for _, pattern := range patterns {
		labels, err := it.querier.FindProtoTargets(ctx, pattern)
		if err != nil {
			return nil, fmt.Errorf("bazel query failed for %q: %w", pattern, err)
		}

		for _, label := range labels {
			pkg, name := splitLabel(label)
			allTargets = append(allTargets, BazelTarget{
				Rule:    "proto_library",
				Package: pkg,
				Name:    name,
			})
		}
	}

	// Enrich targets with srcs/deps from XML query
	for i := range allTargets {
		t := &allTargets[i]
		label := t.Package + ":" + t.Name
		sources, _ := it.querier.GetProtoSources(ctx, label)
		if len(sources) > 0 {
			t.Srcs = sources
		}
		deps, _ := it.querier.GetDeps(ctx, label)
		if len(deps) > 0 {
			t.Deps = deps
		}
	}

	return allTargets, nil
}

// discoverViaFiles scans BUILD files on disk.
func (it *Integrator) discoverViaFiles(ctx context.Context) ([]BazelTarget, error) {
	buildFiles, err := FindBuildFiles(it.workspace.Root)
	if err != nil {
		return nil, fmt.Errorf("bazel: scan BUILD files: %w", err)
	}

	it.workspace.BuildFiles = buildFiles

	var allTargets []BazelTarget
	for pkg, path := range buildFiles {
		targets, err := ParseBuildFile(path, pkg)
		if err != nil {
			continue // skip unparseable files
		}
		protoTargets := FilterProtoTargets(targets)
		allTargets = append(allTargets, protoTargets...)
	}

	return allTargets, nil
}

// ResolveProtoFilePaths resolves all proto_library targets to actual file paths
// on disk, relative to the workspace root.
func (it *Integrator) ResolveProtoFilePaths(targets []BazelTarget) []string {
	seen := make(map[string]bool)
	var files []string

	for _, t := range targets {
		resolved := ResolveProtoFiles(it.workspace, t)
		for _, f := range resolved {
			absPath := filepath.Join(it.workspace.Root, f)
			if _, err := os.Stat(absPath); err != nil {
				continue // skip non-existent files
			}
			if !seen[absPath] {
				seen[absPath] = true
				files = append(files, absPath)
			}
		}
	}

	return files
}

// CreateSyncPlan builds a plan describing what Buffalo will compile
// and what BUILD files it will generate.
func (it *Integrator) CreateSyncPlan(ctx context.Context, targets []BazelTarget, languages []string, outputDir string) (*SyncPlan, error) {
	builds, err := it.generator.GenerateBuildFiles(targets, languages, outputDir)
	if err != nil {
		return nil, fmt.Errorf("bazel: generate BUILD files: %w", err)
	}

	return &SyncPlan{
		TargetsToCompile:     targets,
		BuildFilesToGenerate: builds,
		Languages:            languages,
		OutputDir:            outputDir,
	}, nil
}

// WriteBuildFiles writes the generated BUILD.bazel files to disk.
func (it *Integrator) WriteBuildFiles(builds []GeneratedBuild) error {
	for _, build := range builds {
		dir := filepath.Dir(build.Path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("bazel: mkdir %s: %w", dir, err)
		}
		if err := os.WriteFile(build.Path, []byte(build.Content), 0o644); err != nil {
			return fmt.Errorf("bazel: write %s: %w", build.Path, err)
		}
	}
	return nil
}

// SyncAfterBuild is called after Buffalo compilation completes.
// It generates and writes BUILD.bazel files for the generated code.
func (it *Integrator) SyncAfterBuild(ctx context.Context, targets []BazelTarget, languages []string, outputDir string) error {
	plan, err := it.CreateSyncPlan(ctx, targets, languages, outputDir)
	if err != nil {
		return err
	}

	return it.WriteBuildFiles(plan.BuildFilesToGenerate)
}

// splitLabel splits "//pkg:name" into ("//pkg", "name").
func splitLabel(label string) (string, string) {
	if i := strings.LastIndex(label, ":"); i >= 0 {
		return label[:i], label[i+1:]
	}
	// Shorthand: //pkg → package = //pkg, name = last component
	parts := strings.Split(strings.TrimPrefix(label, "//"), "/")
	name := parts[len(parts)-1]
	return label, name
}
