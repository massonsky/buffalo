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
//
// Two modes of operation:
//
//  1. proto_library mode: Bazel defines proto_library rules, Buffalo discovers them
//     and generates language-specific BUILD targets (go_proto_library, etc.).
//
//  2. filegroup mode: Bazel declares proto sources via filegroup(glob()),
//     Buffalo compiles them and generates filegroup/library BUILD targets
//     for downstream services (compile_data, deps).
type Integrator struct {
	workspace     *Workspace
	querier       *BazelQuerier
	generator     *Generator
	useBazelQuery bool     // whether to use `bazel query` or file-based parsing
	syncMode      SyncMode // detected mode of collaboration
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
		workspace:     ws,
		querier:       querier,
		generator:     generator,
		useBazelQuery: IsBazelAvailable(),
		syncMode:      SyncModeFilegroup, // default, auto-detected later
	}, nil
}

// NewIntegratorFromWorkspace creates an Integrator from an already-detected workspace.
func NewIntegratorFromWorkspace(ws *Workspace) *Integrator {
	return &Integrator{
		workspace:     ws,
		querier:       NewQuerier(ws.Root),
		generator:     NewGenerator(ws.Root, ws.ModuleName),
		useBazelQuery: IsBazelAvailable(),
		syncMode:      SyncModeFilegroup,
	}
}

// GetWorkspace returns the detected workspace.
func (it *Integrator) GetWorkspace() *Workspace {
	return it.workspace
}

// GetSyncMode returns the detected sync mode.
func (it *Integrator) GetSyncMode() SyncMode {
	return it.syncMode
}

// SetBazelPath sets a custom bazel binary path.
func (it *Integrator) SetBazelPath(path string) {
	it.querier.SetBazelPath(path)
}

// DiscoverProtoTargets finds all proto-providing targets in the workspace.
// Uses `bazel query` when available, otherwise falls back to file parsing.
// Auto-detects the sync mode based on discovered targets.
func (it *Integrator) DiscoverProtoTargets(ctx context.Context, patterns []string) ([]BazelTarget, error) {
	var targets []BazelTarget
	var err error

	if it.useBazelQuery {
		targets, err = it.discoverViaQuery(ctx, patterns)
	} else {
		targets, err = it.discoverViaFiles(ctx)
	}

	if err != nil {
		return nil, err
	}

	// Auto-detect sync mode based on what we found
	it.syncMode = it.detectSyncMode(targets)

	return targets, nil
}

// detectSyncMode determines the collaboration mode based on discovered targets.
func (it *Integrator) detectSyncMode(targets []BazelTarget) SyncMode {
	hasProtoLibrary := false
	hasFilegroup := false

	for _, t := range targets {
		if t.Rule == "proto_library" {
			hasProtoLibrary = true
		}
		if t.Rule == "filegroup" && t.IsProtoSource() {
			hasFilegroup = true
		}
	}

	// proto_library takes priority (more structured)
	if hasProtoLibrary {
		return SyncModeProtoLibrary
	}
	if hasFilegroup {
		return SyncModeFilegroup
	}
	// Default to filegroup if we found proto files via file scan
	return SyncModeFilegroup
}

// discoverViaQuery uses `bazel query kind(proto_library, //...)`.
// Falls back to file-based parsing if query fails.
func (it *Integrator) discoverViaQuery(ctx context.Context, patterns []string) ([]BazelTarget, error) {
	if len(patterns) == 0 {
		patterns = []string{"//..."}
	}

	var allTargets []BazelTarget
	var queryFailed bool
	for _, pattern := range patterns {
		labels, err := it.querier.FindProtoTargets(ctx, pattern)
		if err != nil {
			// bazel query failed — fall back to file parsing
			queryFailed = true
			break
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

	// If bazel query failed or found nothing, fall back to file parsing
	if queryFailed || len(allTargets) == 0 {
		return it.discoverViaFiles(ctx)
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

// ResolveProtoFilePaths resolves all proto-providing targets to actual file paths
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
	var builds []GeneratedBuild
	var err error

	switch it.syncMode {
	case SyncModeProtoLibrary:
		builds, err = it.generator.GenerateBuildFiles(targets, languages, outputDir)
	case SyncModeFilegroup:
		builds, err = it.generator.GenerateFilegroupBuilds(languages, outputDir)
	default:
		builds, err = it.generator.GenerateFilegroupBuilds(languages, outputDir)
	}

	if err != nil {
		return nil, fmt.Errorf("bazel: generate BUILD files: %w", err)
	}

	return &SyncPlan{
		TargetsToCompile:     targets,
		BuildFilesToGenerate: builds,
		Languages:            languages,
		OutputDir:            outputDir,
		Mode:                 it.syncMode,
	}, nil
}

// WriteBuildFiles writes the generated BUILD.bazel files to disk.
// Existing non-Buffalo BUILD files are preserved.
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
