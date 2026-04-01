package builder

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	"github.com/massonsky/buffalo/internal/config"
	pkglogger "github.com/massonsky/buffalo/pkg/logger"
	"github.com/massonsky/buffalo/pkg/metrics"
)

type stubParser struct{}

func (stubParser) ParseFiles(ctx context.Context, files []string, importPaths []string) ([]*ProtoFile, error) {
	result := make([]*ProtoFile, 0, len(files))
	for _, file := range files {
		result = append(result, &ProtoFile{Path: file, Package: "demo", Syntax: "proto3"})
	}
	return result, nil
}

func (stubParser) ParseFile(ctx context.Context, path string, importPaths []string) (*ProtoFile, error) {
	return &ProtoFile{Path: path, Package: "demo", Syntax: "proto3"}, nil
}

type stubResolver struct{}

func (stubResolver) Resolve(ctx context.Context, files []*ProtoFile) (*DependencyGraph, error) {
	nodes := make(map[string]*ProtoFile, len(files))
	order := make([]string, 0, len(files))
	for _, file := range files {
		nodes[file.Path] = file
		order = append(order, file.Path)
	}
	return &DependencyGraph{Nodes: nodes, Edges: map[string][]string{}, CompilationOrder: order}, nil
}

type failingExecutor struct{}

func (failingExecutor) Execute(ctx context.Context, plan *ExecutionPlan) (*ExecutionResult, error) {
	return &ExecutionResult{FilesGenerated: 0, Warnings: []string{"boom"}, Metrics: map[string]interface{}{}}, stderrors.New("executor failed")
}

func TestBuilderBuildReturnsExecutorError(t *testing.T) {
	b := &builder{
		parser:   stubParser{},
		resolver: stubResolver{},
		executor: failingExecutor{},
		log:      pkglogger.New(),
		metrics:  metrics.NewCollector(),
		config: &config.Config{
			Build: config.BuildConfig{Cache: config.CacheConfig{Directory: ".buffalo-cache"}},
		},
	}

	result, err := b.Build(context.Background(), &BuildPlan{
		ProtoFiles:  []string{"demo.proto"},
		ImportPaths: []string{"."},
		OutputDir:   "generated",
		Languages:   []string{"go"},
		Options:     BuildOptions{},
	})
	if err == nil {
		t.Fatal("expected builder to return executor error")
	}
	if result == nil {
		t.Fatal("expected build result even on failure")
	}
	if result.Success {
		t.Fatal("expected result.Success=false when executor fails")
	}
	if len(result.Errors) == 0 {
		t.Fatal("expected build result to retain executor error")
	}
	if result.Duration <= 0*time.Nanosecond {
		t.Fatal("expected duration to be recorded")
	}
}
