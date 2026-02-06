package builder

import (
	"context"
	"path/filepath"
	"sync"

	"github.com/massonsky/buffalo/internal/compiler"
	"github.com/massonsky/buffalo/internal/compiler/cpp"
	"github.com/massonsky/buffalo/internal/compiler/golang"
	"github.com/massonsky/buffalo/internal/compiler/python"
	"github.com/massonsky/buffalo/internal/compiler/rust"
	"github.com/massonsky/buffalo/internal/config"
	"github.com/massonsky/buffalo/internal/versioning"
	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/massonsky/buffalo/pkg/metrics"
	"github.com/massonsky/buffalo/pkg/utils"
)

// ExecutionPlan describes how to execute the build
type ExecutionPlan struct {
	// Graph is the dependency graph
	Graph *DependencyGraph

	// OutputDir is the output directory
	OutputDir string

	// ImportPaths are the import paths for protoc
	ImportPaths []string

	// Languages are the target languages
	Languages []string

	// Options are build options
	Options BuildOptions
}

// ExecutionResult contains execution results
type ExecutionResult struct {
	// FilesGenerated is the number of generated files
	FilesGenerated int

	// Warnings contains execution warnings
	Warnings []string

	// Metrics contains execution metrics
	Metrics map[string]interface{}
}

// Executor executes the build plan
type Executor interface {
	// Execute runs the build execution
	Execute(ctx context.Context, plan *ExecutionPlan) (*ExecutionResult, error)
}

// executor implements Executor
type executor struct {
	log             Logger
	metrics         *metrics.Collector
	config          *config.Config // Add config field
	compilers       map[string]compiler.Compiler
	versionManager  *versioning.Manager
	languageConfigs map[string]interface{} // Language-specific configs
}

// NewExecutor creates a new Executor
func NewExecutor(log Logger, m *metrics.Collector, cfg *config.Config) Executor {
	// Create logger for compilers
	var compilerLog *logger.Logger
	if logAdapter, ok := log.(*loggerAdapter); ok {
		compilerLog = logAdapter.log
	} else {
		// Fallback to a new logger
		compilerLog = logger.New()
	}

	// Initialize versioning manager
	versionMgr := versioning.New(versioning.Options{
		Enabled:      cfg.Versioning.Enabled,
		Strategy:     versioning.Strategy(cfg.Versioning.Strategy),
		OutputFormat: versioning.OutputFormat(cfg.Versioning.OutputFormat),
		KeepVersions: cfg.Versioning.KeepVersions,
		StateDir:     filepath.Join(cfg.Build.Cache.Directory, "versions"),
	})

	// Initialize compilers
	compilers := make(map[string]compiler.Compiler)

	// Python compiler
	if cfg.Languages.Python.Enabled {
		pythonOpts := python.DefaultOptions()
		pythonOpts.WorkDir = cfg.Languages.Python.WorkDir
		// Use custom exclude imports if configured, otherwise keep defaults
		if len(cfg.Languages.Python.ExcludeImports) > 0 {
			pythonOpts.ExcludeImports = cfg.Languages.Python.ExcludeImports
		}
		compilers["python"] = python.New(compilerLog, pythonOpts)
	}

	// Go compiler
	if cfg.Languages.Go.Enabled {
		goOpts := golang.DefaultOptions()
		goOpts.GoModule = cfg.Languages.Go.Module
		compilers["go"] = golang.New(compilerLog, &goOpts)
	}

	// Rust compiler
	if cfg.Languages.Rust.Enabled {
		rustOpts := rust.DefaultOptions()
		rustOpts.Generator = cfg.Languages.Rust.Generator
		if rustOpts.Generator == "" {
			rustOpts.Generator = "prost"
		}
		compilers["rust"] = rust.New(compilerLog, &rustOpts)
	}

	// C++ compiler
	if cfg.Languages.Cpp.Enabled {
		cppOpts := cpp.DefaultOptions()
		cppOpts.Namespace = cfg.Languages.Cpp.Namespace
		compilers["cpp"] = cpp.New(compilerLog, &cppOpts)
	}

	return &executor{
		log:            log,
		metrics:        m,
		config:         cfg, // Store config
		compilers:      compilers,
		versionManager: versionMgr,
	}
}

// Execute runs the build execution
func (e *executor) Execute(ctx context.Context, plan *ExecutionPlan) (*ExecutionResult, error) {
	e.log.Debug("Executing build plan",
		"files", len(plan.Graph.Nodes),
		"languages", len(plan.Languages),
	)

	result := &ExecutionResult{
		Metrics: make(map[string]interface{}),
	}

	// Get worker count
	workers := plan.Options.Workers
	if workers <= 0 {
		workers = 4 // Default
	}

	// Create worker pool
	pool, err := utils.NewWorkerPool(workers)
	if err != nil {
		return nil, err
	}
	defer pool.Close()

	// Prepare tasks
	var tasks []utils.Task
	for _, file := range plan.Graph.CompilationOrder {
		protoFile := plan.Graph.Nodes[file]

		// Create task for each language
		for _, lang := range plan.Languages {
			fileCopy := protoFile
			langCopy := lang

			task := func() error {
				return e.compileFile(ctx, fileCopy, langCopy, plan)
			}
			tasks = append(tasks, task)
		}
	}

	// Execute tasks
	e.log.Debug("Executing tasks", "count", len(tasks), "workers", workers)
	taskResults := pool.Execute(tasks)

	// Process results
	var errors []error
	for _, tr := range taskResults {
		if tr.Error != nil {
			errors = append(errors, tr.Error)
			e.log.Warn("Task failed", "index", tr.Index, "error", tr.Error)
		} else {
			result.FilesGenerated++
		}
	}

	if len(errors) > 0 {
		e.log.Error("Build execution completed with errors", "count", len(errors))
		return result, errors[0] // Return first error
	}

	e.log.Debug("Build execution completed successfully")
	return result, nil
}

// compileFile compiles a single proto file for a language
func (e *executor) compileFile(ctx context.Context, file *ProtoFile, language string, plan *ExecutionPlan) error {
	e.log.Debug("Compiling file",
		"file", file.Path,
		"language", language,
		"package", file.Package,
	)

	if plan.Options.DryRun {
		e.log.Info("DRY RUN: Would compile",
			"file", file.Path,
			"language", language,
		)
		return nil
	}

	// Check versioning
	// Create language-specific output directory: generated/{language}/...
	baseOutputDir := plan.OutputDir
	outputDir := filepath.Join(baseOutputDir, language)

	if e.versionManager.IsEnabled() {
		shouldGenerate, err := e.versionManager.ShouldGenerateNewVersion(file.Path, language)
		if err != nil {
			e.log.Warn("Versioning check failed, proceeding with compilation",
				"file", file.Path,
				"language", language,
				"error", err,
			)
		} else if !shouldGenerate {
			e.log.Info("⏭️  Skipping unchanged file",
				"file", file.Path,
				"language", language,
			)
			return nil
		}

		// Generate version
		version, err := e.versionManager.GenerateVersion(file.Path)
		if err != nil {
			e.log.Warn("Failed to generate version, using default output",
				"file", file.Path,
				"error", err,
			)
		} else {
			e.log.Info("📦 Generating new version",
				"file", file.Path,
				"version", version,
			)

			// Update output directory with version: generated/{language}/{version}/...
			outputDir = e.versionManager.GetVersionedOutputPath(outputDir, version)
		}
	}

	// Track metrics
	counter := e.metrics.Counter("files_compiled_total")
	counter.Inc()

	gauge := e.metrics.Gauge("active_compilations")
	gauge.Inc()
	defer gauge.Dec()

	// Get compiler for the language
	comp, ok := e.compilers[language]
	if !ok {
		e.log.Warn("No compiler available for language, skipping",
			"language", language,
			"file", file.Path,
		)
		return nil
	}

	// Prepare compiler options
	compilerOpts := compiler.CompileOptions{
		OutputDir:              outputDir,
		ImportPaths:            plan.ImportPaths,
		Verbose:                plan.Options.Verbose,
		PreserveProtoStructure: e.config.Output.PreserveProtoStructure,
	}

	// Convert ProtoFile to compiler.ProtoFile
	compilerFile := compiler.ProtoFile{
		Path:        file.Path,
		Package:     file.Package,
		ImportPaths: compilerOpts.ImportPaths,
	}

	// Compile
	result, err := comp.Compile(ctx, []compiler.ProtoFile{compilerFile}, compilerOpts)
	if err != nil {
		return err
	}

	if !result.Success {
		e.log.Warn("Compilation completed with warnings",
			"file", file.Path,
			"language", language,
			"warnings", len(result.Warnings),
		)
	}

	// Save version state if versioning is enabled
	if e.versionManager.IsEnabled() {
		version, _ := e.versionManager.GenerateVersion(file.Path)
		if verErr := e.versionManager.SaveVersion(file.Path, language, version, outputDir); verErr != nil {
			e.log.Warn("Failed to save version state",
				"file", file.Path,
				"language", language,
				"error", verErr,
			)
		}
	}

	e.log.Info("✨ Compiled",
		"file", file.Path,
		"language", language,
		"generated", len(result.GeneratedFiles),
	)

	return nil
}

// parallelExecute executes tasks in parallel with error handling
func (e *executor) parallelExecute(ctx context.Context, tasks []func() error) []error {
	var wg sync.WaitGroup
	errorChan := make(chan error, len(tasks))

	for _, task := range tasks {
		wg.Add(1)
		go func(t func() error) {
			defer wg.Done()
			if err := t(); err != nil {
				errorChan <- err
			}
		}(task)
	}

	wg.Wait()
	close(errorChan)

	var errors []error
	for err := range errorChan {
		errors = append(errors, err)
	}

	return errors
}
