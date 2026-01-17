package plugin

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/massonsky/buffalo/pkg/logger"
)

// Registry manages plugin registration, lifecycle, and execution
type Registry struct {
	plugins     map[string]*RegisteredPlugin
	hooks       map[HookPoint][]*RegisteredPlugin
	mu          sync.RWMutex
	log         *logger.Logger
	initialized bool
}

// RegisteredPlugin wraps a plugin with metadata and state
type RegisteredPlugin struct {
	Plugin   Plugin
	Config   Config
	Metadata Metadata
	Status   Status
	mu       sync.RWMutex
}

// NewRegistry creates a new plugin registry
func NewRegistry(log *logger.Logger) *Registry {
	return &Registry{
		plugins: make(map[string]*RegisteredPlugin),
		hooks:   make(map[HookPoint][]*RegisteredPlugin),
		log:     log,
	}
}

// Register registers a plugin with the registry
func (r *Registry) Register(plugin Plugin, config Config) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := plugin.Name()

	// Check if plugin already registered
	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin %s is already registered", name)
	}

	// Create registered plugin wrapper
	registered := &RegisteredPlugin{
		Plugin: plugin,
		Config: config,
		Metadata: Metadata{
			Name:        plugin.Name(),
			Version:     plugin.Version(),
			Type:        plugin.Type(),
			Description: plugin.Description(),
		},
		Status: StatusLoaded,
	}

	// Store in registry
	r.plugins[name] = registered

	// Register for hook points if specified
	if config.Enabled {
		for _, hookPoint := range config.HookPoints {
			r.hooks[hookPoint] = append(r.hooks[hookPoint], registered)
		}
	}

	r.log.Debug("Plugin registered",
		logger.String("name", name),
		logger.String("version", plugin.Version()),
		logger.String("type", string(plugin.Type())),
		logger.Bool("enabled", config.Enabled),
	)

	return nil
}

// Unregister removes a plugin from the registry
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	registered, exists := r.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Shutdown the plugin
	if err := registered.Plugin.Shutdown(); err != nil {
		r.log.Warn("Error shutting down plugin", logger.String("name", name), logger.Any("error", err))
	}

	// Remove from hooks
	for hookPoint, plugins := range r.hooks {
		filtered := make([]*RegisteredPlugin, 0, len(plugins))
		for _, p := range plugins {
			if p.Plugin.Name() != name {
				filtered = append(filtered, p)
			}
		}
		r.hooks[hookPoint] = filtered
	}

	// Remove from registry
	delete(r.plugins, name)

	r.log.Debug("Plugin unregistered", logger.String("name", name))
	return nil
}

// Get retrieves a registered plugin by name
func (r *Registry) Get(name string) (*RegisteredPlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	registered, exists := r.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	return registered, nil
}

// List returns all registered plugins
func (r *Registry) List() []*RegisteredPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugins := make([]*RegisteredPlugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		plugins = append(plugins, p)
	}

	return plugins
}

// ListByType returns plugins of a specific type
func (r *Registry) ListByType(pluginType PluginType) []*RegisteredPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugins := make([]*RegisteredPlugin, 0)
	for _, p := range r.plugins {
		if p.Plugin.Type() == pluginType {
			plugins = append(plugins, p)
		}
	}

	return plugins
}

// InitAll initializes all enabled plugins
func (r *Registry) InitAll() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.initialized {
		return nil
	}

	r.log.Info("Initializing plugins", logger.Int("count", len(r.plugins)))

	for name, registered := range r.plugins {
		if !registered.Config.Enabled {
			registered.setStatus(StatusDisabled)
			continue
		}

		r.log.Debug("Initializing plugin", logger.String("name", name))

		if err := registered.Plugin.Init(registered.Config); err != nil {
			registered.setStatus(StatusError)
			return fmt.Errorf("failed to initialize plugin %s: %w", name, err)
		}

		registered.setStatus(StatusInitialized)
		r.log.Debug("Plugin initialized", logger.String("name", name))
	}

	r.initialized = true
	r.log.Info("All plugins initialized successfully")

	return nil
}

// ExecuteHook executes all plugins registered for a specific hook point
func (r *Registry) ExecuteHook(ctx context.Context, hookPoint HookPoint, input *Input) error {
	r.mu.RLock()
	plugins := r.hooks[hookPoint]
	r.mu.RUnlock()

	if len(plugins) == 0 {
		r.log.Debug("No plugins for hook", logger.String("hook", string(hookPoint)))
		return nil
	}

	// Sort by priority (descending)
	sortedPlugins := make([]*RegisteredPlugin, len(plugins))
	copy(sortedPlugins, plugins)
	sort.Slice(sortedPlugins, func(i, j int) bool {
		return sortedPlugins[i].Config.Priority > sortedPlugins[j].Config.Priority
	})

	r.log.Debug("Executing hook",
		logger.String("hook", string(hookPoint)),
		logger.Int("plugins", len(sortedPlugins)),
	)

	for _, registered := range sortedPlugins {
		if registered.Status != StatusInitialized {
			r.log.Warn("Skipping plugin (not initialized)",
				logger.String("plugin", registered.Plugin.Name()),
				logger.String("status", string(registered.Status)),
			)
			continue
		}

		r.log.Debug("Executing plugin",
			logger.String("plugin", registered.Plugin.Name()),
			logger.String("hook", string(hookPoint)),
		)

		registered.setStatus(StatusRunning)

		output, err := registered.Plugin.Execute(ctx, input)
		if err != nil {
			registered.setStatus(StatusError)
			return fmt.Errorf("plugin %s failed: %w", registered.Plugin.Name(), err)
		}

		registered.setStatus(StatusInitialized)

		// Log output
		if output != nil {
			if len(output.Messages) > 0 {
				for _, msg := range output.Messages {
					r.log.Info(msg, logger.String("plugin", registered.Plugin.Name()))
				}
			}
			if len(output.Warnings) > 0 {
				for _, warn := range output.Warnings {
					r.log.Warn(warn, logger.String("plugin", registered.Plugin.Name()))
				}
			}
			if len(output.Errors) > 0 {
				for _, errMsg := range output.Errors {
					r.log.Error(errMsg, logger.String("plugin", registered.Plugin.Name()))
				}
			}

			// Check if plugin execution was successful
			if !output.Success {
				registered.setStatus(StatusError)
				return fmt.Errorf("plugin %s validation failed with %d error(s)",
					registered.Plugin.Name(), len(output.Errors))
			}

			// Update input with generated files for next plugin
			if len(output.GeneratedFiles) > 0 {
				input.GeneratedFiles = append(input.GeneratedFiles, output.GeneratedFiles...)
			}
		}
	}

	return nil
}

// ExecuteCompiler executes a compiler plugin for a specific language
func (r *Registry) ExecuteCompiler(ctx context.Context, language string, input *Input) (*Output, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Find compiler plugin for this language
	for _, registered := range r.plugins {
		if registered.Plugin.Type() != PluginTypeCompiler {
			continue
		}

		if !registered.Config.Enabled || registered.Status != StatusInitialized {
			continue
		}

		// Check if this is a compiler plugin with the right language
		if compilerPlugin, ok := registered.Plugin.(CompilerPlugin); ok {
			if compilerPlugin.SupportedLanguage() == language {
				r.log.Debug("Executing compiler plugin",
					logger.String("plugin", registered.Plugin.Name()),
					logger.String("language", language),
				)

				registered.setStatus(StatusRunning)
				output, err := registered.Plugin.Execute(ctx, input)
				registered.setStatus(StatusInitialized)

				if err != nil {
					registered.setStatus(StatusError)
					return nil, fmt.Errorf("compiler plugin %s failed: %w", registered.Plugin.Name(), err)
				}

				return output, nil
			}
		}
	}

	// No compiler plugin found for this language
	return nil, fmt.Errorf("no compiler plugin found for language: %s", language)
}

// ShutdownAll shuts down all plugins
func (r *Registry) ShutdownAll() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.log.Info("Shutting down plugins", logger.Int("count", len(r.plugins)))

	var errs []error
	for name, registered := range r.plugins {
		r.log.Debug("Shutting down plugin", logger.String("name", name))
		if err := registered.Plugin.Shutdown(); err != nil {
			errs = append(errs, fmt.Errorf("plugin %s: %w", name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errs)
	}

	r.initialized = false
	return nil
}

// setStatus safely updates the plugin status
func (rp *RegisteredPlugin) setStatus(status Status) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	rp.Status = status
}

// GetStatus safely retrieves the plugin status
func (rp *RegisteredPlugin) GetStatus() Status {
	rp.mu.RLock()
	defer rp.mu.RUnlock()
	return rp.Status
}
