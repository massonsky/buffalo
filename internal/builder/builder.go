package builder

import (
	"context"
	"time"

	"github.com/massonsky/buffalo/internal/config"
	"github.com/massonsky/buffalo/internal/plugin"
	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/massonsky/buffalo/pkg/metrics"
)

// Builder is the main interface for building protobuf files
type Builder interface {
	// Build executes the build process
	Build(ctx context.Context, plan *BuildPlan) (*BuildResult, error)

	// GetMetrics returns build metrics
	GetMetrics() *metrics.Collector
}

// BuildPlan describes what to build
type BuildPlan struct {
	// ProtoFiles is the list of proto files to compile
	ProtoFiles []string

	// ImportPaths are additional import paths
	ImportPaths []string

	// OutputDir is the base output directory
	OutputDir string

	// Languages are the target languages
	Languages []string

	// Options are additional build options
	Options BuildOptions
}

// BuildOptions contains build configuration
type BuildOptions struct {
	// Workers is the number of parallel workers (0 = auto)
	Workers int

	// Incremental enables incremental builds
	Incremental bool

	// Cache enables caching
	Cache bool

	// CacheDir is the cache directory
	CacheDir string

	// DryRun only shows what would be built
	DryRun bool

	// Verbose enables verbose logging
	Verbose bool
}

// BuildResult contains build results
type BuildResult struct {
	// Success indicates if build was successful
	Success bool

	// Duration is the total build time
	Duration time.Duration

	// FilesProcessed is the number of files processed
	FilesProcessed int

	// FilesGenerated is the number of generated files
	FilesGenerated int

	// Errors contains any build errors
	Errors []error

	// Warnings contains build warnings
	Warnings []string

	// CacheHits is the number of cache hits
	CacheHits int

	// CacheMisses is the number of cache misses
	CacheMisses int
}

// builder implements Builder interface
type builder struct {
	parser         ProtoParser
	resolver       DependencyResolver
	cache          CacheManager
	executor       Executor
	log            *logger.Logger
	metrics        *metrics.Collector
	config         *config.Config
	pluginRegistry *plugin.Registry
}

// New creates a new Builder
func New(cfg *config.Config, opts ...Option) (Builder, error) {
	if cfg == nil {
		cfg = &config.Config{
			Build: config.BuildConfig{
				Cache: config.CacheConfig{Directory: ".buffalo-cache"},
			},
		}
	}

	b := &builder{
		log:     logger.New(),
		metrics: metrics.NewCollector(),
		config:  cfg,
	}

	for _, opt := range opts {
		if err := opt(b); err != nil {
			return nil, err
		}
	}

	// Create logger adapter
	logAdapter := NewLoggerAdapter(b.log)

	// Initialize default components if not set
	if b.parser == nil {
		b.parser = NewProtoParser(logAdapter)
	}
	if b.resolver == nil {
		b.resolver = NewDependencyResolver(logAdapter)
	}
	if b.cache == nil {
		b.cache = NewCacheManagerWithTools(logAdapter, pinnedToolVersions(cfg))
	}
	if b.executor == nil {
		b.executor = NewExecutor(logAdapter, b.metrics, cfg)
	}

	return b, nil
}

// Option is a functional option for Builder
type Option func(*builder) error

// WithLogger sets the logger
func WithLogger(log *logger.Logger) Option {
	return func(b *builder) error {
		b.log = log
		return nil
	}
}

// WithMetrics sets the metrics collector
func WithMetrics(m *metrics.Collector) Option {
	return func(b *builder) error {
		b.metrics = m
		return nil
	}
}

// WithParser sets the proto parser
func WithParser(p ProtoParser) Option {
	return func(b *builder) error {
		b.parser = p
		return nil
	}
}

// WithResolver sets the dependency resolver
func WithResolver(r DependencyResolver) Option {
	return func(b *builder) error {
		b.resolver = r
		return nil
	}
}

// WithPluginRegistry sets the plugin registry
func WithPluginRegistry(registry *plugin.Registry) Option {
	return func(b *builder) error {
		b.pluginRegistry = registry
		return nil
	}
}

// WithCache sets the cache manager
func WithCache(c CacheManager) Option {
	return func(b *builder) error {
		b.cache = c
		return nil
	}
}

// WithExecutor sets the executor
func WithExecutor(e Executor) Option {
	return func(b *builder) error {
		b.executor = e
		return nil
	}
}

// Build executes the build process
func (b *builder) Build(ctx context.Context, plan *BuildPlan) (*BuildResult, error) {
	startTime := time.Now()

	b.log.Info("🔨 Starting build process",
		logger.Int("proto_files", len(plan.ProtoFiles)),
		logger.Any("languages", plan.Languages),
	)

	// Initialize result
	result := &BuildResult{
		Success: true,
	}

	// Execute pre-build hooks
	if b.pluginRegistry != nil {
		b.log.Debug("Executing pre-build hooks...")
		pluginInput := &plugin.Input{
			ProtoFiles:  plan.ProtoFiles,
			OutputDir:   plan.OutputDir,
			ImportPaths: plan.ImportPaths,
			Metadata:    make(map[string]interface{}),
		}
		if err := b.pluginRegistry.ExecuteHook(ctx, plugin.HookPointPreBuild, pluginInput); err != nil {
			return nil, err
		}
	}

	// Parse proto files
	b.log.Debug("Parsing proto files...")
	protoFiles, err := b.parser.ParseFiles(ctx, plan.ProtoFiles, plan.ImportPaths)
	if err != nil {
		return nil, err
	}
	result.FilesProcessed = len(protoFiles)

	// Execute post-parse hooks
	if b.pluginRegistry != nil {
		b.log.Debug("Executing post-parse hooks...")
		pluginInput := &plugin.Input{
			ProtoFiles:  plan.ProtoFiles,
			OutputDir:   plan.OutputDir,
			ImportPaths: plan.ImportPaths,
			Metadata: map[string]interface{}{
				"parsed_files": protoFiles,
			},
		}
		if err := b.pluginRegistry.ExecuteHook(ctx, plugin.HookPointPostParse, pluginInput); err != nil {
			return nil, err
		}
	}

	// Resolve dependencies
	b.log.Debug("Resolving dependencies...")
	graph, err := b.resolver.Resolve(ctx, protoFiles)
	if err != nil {
		return nil, err
	}

	// Check cache if enabled
	if plan.Options.Cache {
		b.log.Debug("Checking cache...")
		hits, misses := b.cache.Check(ctx, protoFiles)
		result.CacheHits = hits
		result.CacheMisses = misses
	}

	// Execute build
	b.log.Debug("Executing build...")
	execResult, err := b.executor.Execute(ctx, &ExecutionPlan{
		Graph:       graph,
		OutputDir:   plan.OutputDir,
		ImportPaths: plan.ImportPaths,
		Languages:   plan.Languages,
		Options:     plan.Options,
	})
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, err)
	}

	if execResult != nil {
		result.FilesGenerated = execResult.FilesGenerated
		result.Warnings = execResult.Warnings
	}

	// Execute post-build hooks
	if b.pluginRegistry != nil {
		b.log.Debug("Executing post-build hooks...")
		pluginInput := &plugin.Input{
			ProtoFiles:     plan.ProtoFiles,
			OutputDir:      plan.OutputDir,
			ImportPaths:    plan.ImportPaths,
			GeneratedFiles: []string{}, // TODO: populate from execResult
			Metadata: map[string]interface{}{
				"files_generated": result.FilesGenerated,
				"duration":        time.Since(startTime),
			},
		}
		if err := b.pluginRegistry.ExecuteHook(ctx, plugin.HookPointPostBuild, pluginInput); err != nil {
			b.log.Warn("Post-build hook failed", logger.Any("error", err))
			// Don't fail the build, just log the warning
		}
	}

	result.Duration = time.Since(startTime)

	b.log.Info("✅ Build completed",
		logger.String("duration", result.Duration.String()),
		logger.Int("files_processed", result.FilesProcessed),
		logger.Int("files_generated", result.FilesGenerated),
		logger.Bool("success", result.Success),
	)

	if len(result.Errors) > 0 {
		return result, result.Errors[0]
	}

	return result, nil
}

// GetMetrics returns build metrics
func (b *builder) GetMetrics() *metrics.Collector {
	return b.metrics
}
