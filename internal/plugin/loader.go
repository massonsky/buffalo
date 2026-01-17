package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"

	"github.com/massonsky/buffalo/pkg/logger"
)

// Loader handles loading plugins from filesystem
type Loader struct {
	log          *logger.Logger
	pluginDirs   []string
	loadedPlugins map[string]*plugin.Plugin
}

// NewLoader creates a new plugin loader
func NewLoader(log *logger.Logger, pluginDirs ...string) *Loader {
	// Default plugin directories if none provided
	if len(pluginDirs) == 0 {
		homeDir, _ := os.UserHomeDir()
		pluginDirs = []string{
			filepath.Join(homeDir, ".buffalo", "plugins"),
			"./plugins",
		}
	}

	return &Loader{
		log:          log,
		pluginDirs:   pluginDirs,
		loadedPlugins: make(map[string]*plugin.Plugin),
	}
}

// LoadAll discovers and loads all plugins from configured directories
func (l *Loader) LoadAll(registry *Registry) error {
	l.log.Info("Loading plugins from directories", logger.Any("dirs", l.pluginDirs))

	for _, dir := range l.pluginDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			l.log.Debug("Plugin directory does not exist", logger.String("dir", dir))
			continue
		}

		if err := l.loadFromDirectory(dir, registry); err != nil {
			return fmt.Errorf("failed to load plugins from %s: %w", dir, err)
		}
	}

	return nil
}

// loadFromDirectory loads all plugins from a specific directory
func (l *Loader) loadFromDirectory(dir string, registry *Registry) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Check for plugin.so in subdirectory
			pluginPath := filepath.Join(dir, entry.Name(), "plugin.so")
			if _, err := os.Stat(pluginPath); err == nil {
				if err := l.loadPlugin(pluginPath, registry); err != nil {
					l.log.Warn("Failed to load plugin",
						logger.String("path", pluginPath),
						logger.Any("error", err),
					)
				}
			}
		} else if filepath.Ext(entry.Name()) == ".so" {
			// Direct .so file in plugin directory
			pluginPath := filepath.Join(dir, entry.Name())
			if err := l.loadPlugin(pluginPath, registry); err != nil {
				l.log.Warn("Failed to load plugin",
					logger.String("path", pluginPath),
					logger.Any("error", err),
				)
			}
		}
	}

	return nil
}

// loadPlugin loads a single plugin from a .so file
func (l *Loader) loadPlugin(path string, registry *Registry) error {
	l.log.Debug("Loading plugin", logger.String("path", path))

	// Open the plugin
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open plugin: %w", err)
	}

	// Look for New function that returns a Plugin
	newFunc, err := p.Lookup("New")
	if err != nil {
		return fmt.Errorf("plugin does not export 'New' function: %w", err)
	}

	// Call New() to get plugin instance
	newPluginFunc, ok := newFunc.(func() Plugin)
	if !ok {
		return fmt.Errorf("New function has wrong signature, expected: func() Plugin")
	}

	pluginInstance := newPluginFunc()

	// Look for DefaultConfig function (optional)
	var config Config
	if configFunc, err := p.Lookup("DefaultConfig"); err == nil {
		if getConfig, ok := configFunc.(func() Config); ok {
			config = getConfig()
		}
	}

	// Set default priority if not set
	if config.Priority == 0 {
		config.Priority = 100
	}

	// Default to enabled
	if config.Name == "" {
		config.Name = pluginInstance.Name()
		config.Enabled = true
	}

	// Register the plugin
	if err := registry.Register(pluginInstance, config); err != nil {
		return fmt.Errorf("failed to register plugin: %w", err)
	}

	l.loadedPlugins[pluginInstance.Name()] = p

	l.log.Info("Plugin loaded successfully",
		logger.String("name", pluginInstance.Name()),
		logger.String("version", pluginInstance.Version()),
		logger.String("path", path),
	)

	return nil
}

// LoadByName loads a specific plugin by name from configured directories
func (l *Loader) LoadByName(name string, registry *Registry) error {
	for _, dir := range l.pluginDirs {
		// Try subdirectory with plugin name
		pluginPath := filepath.Join(dir, name, "plugin.so")
		if _, err := os.Stat(pluginPath); err == nil {
			return l.loadPlugin(pluginPath, registry)
		}

		// Try direct .so file
		pluginPath = filepath.Join(dir, name+".so")
		if _, err := os.Stat(pluginPath); err == nil {
			return l.loadPlugin(pluginPath, registry)
		}
	}

	return fmt.Errorf("plugin %s not found in any plugin directory", name)
}

// Unload unloads a plugin (note: Go plugins cannot truly be unloaded)
func (l *Loader) Unload(name string) error {
	// Note: Go's plugin system doesn't support true unloading
	// We can only remove it from our tracking
	if _, exists := l.loadedPlugins[name]; !exists {
		return fmt.Errorf("plugin %s is not loaded", name)
	}

	delete(l.loadedPlugins, name)
	l.log.Debug("Plugin unloaded from loader", logger.String("name", name))

	return nil
}

// GetLoadedPlugins returns names of all loaded plugins
func (l *Loader) GetLoadedPlugins() []string {
	names := make([]string, 0, len(l.loadedPlugins))
	for name := range l.loadedPlugins {
		names = append(names, name)
	}
	return names
}
