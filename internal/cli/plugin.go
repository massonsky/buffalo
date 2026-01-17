package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/massonsky/buffalo/internal/config"
	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var (
	// Plugin flags
	pluginURL    string
	pluginName   string
	pluginForce  bool
	pluginGlobal bool
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage Buffalo plugins",
	Long: `Manage Buffalo plugins: list, install, remove, enable, and disable plugins.

Plugins can be installed from:
  - Local .so files
  - Remote URLs
  - Built-in plugins

Plugins are stored in:
  - Global: ~/.buffalo/plugins/
  - Local: ./plugins/`,
	Example: `  # List all plugins
  buffalo plugin list

  # Install plugin from URL
  buffalo plugin install https://example.com/myplugin.so --name myplugin

  # Install plugin from local file
  buffalo plugin install ./myplugin.so --name myplugin

  # Remove plugin
  buffalo plugin remove myplugin

  # Enable plugin in config
  buffalo plugin enable myplugin

  # Disable plugin in config
  buffalo plugin disable myplugin`,
}

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available plugins",
	Long:  `List all plugins found in ~/.buffalo/plugins/ and ./plugins/ directories, plus built-in plugins.`,
	RunE:  runPluginList,
}

var pluginInstallCmd = &cobra.Command{
	Use:   "install [source]",
	Short: "Install a plugin from URL or local file",
	Long: `Install a plugin from a URL or local .so file.

The plugin will be installed to:
  - Global: ~/.buffalo/plugins/ (with --global flag)
  - Local: ./plugins/ (default)`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginInstall,
}

var pluginRemoveCmd = &cobra.Command{
	Use:     "remove [name]",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove an installed plugin",
	Long:    `Remove a plugin from ~/.buffalo/plugins/ or ./plugins/ directory.`,
	Args:    cobra.ExactArgs(1),
	RunE:    runPluginRemove,
}

var pluginEnableCmd = &cobra.Command{
	Use:   "enable [name]",
	Short: "Enable a plugin in buffalo.yaml",
	Long:  `Enable a plugin by setting enabled: true in the buffalo.yaml configuration file.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runPluginEnable,
}

var pluginDisableCmd = &cobra.Command{
	Use:   "disable [name]",
	Short: "Disable a plugin in buffalo.yaml",
	Long:  `Disable a plugin by setting enabled: false in the buffalo.yaml configuration file.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runPluginDisable,
}

func init() {
	// Add subcommands
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginInstallCmd)
	pluginCmd.AddCommand(pluginRemoveCmd)
	pluginCmd.AddCommand(pluginEnableCmd)
	pluginCmd.AddCommand(pluginDisableCmd)

	// Install flags
	pluginInstallCmd.Flags().StringVarP(&pluginName, "name", "n", "", "plugin name (required)")
	pluginInstallCmd.Flags().BoolVarP(&pluginGlobal, "global", "g", false, "install globally to ~/.buffalo/plugins/")
	pluginInstallCmd.Flags().BoolVarP(&pluginForce, "force", "f", false, "force overwrite if plugin exists")
	pluginInstallCmd.MarkFlagRequired("name")

	// Remove flags
	pluginRemoveCmd.Flags().BoolVarP(&pluginGlobal, "global", "g", false, "remove from global ~/.buffalo/plugins/")

	rootCmd.AddCommand(pluginCmd)
}

// runPluginList lists all available plugins
func runPluginList(cmd *cobra.Command, args []string) error {
	log := GetLogger()
	log.Info("Listing available plugins")

	// Built-in plugins
	builtins := []string{"naming-validator"}
	log.Info("Built-in plugins:")
	for _, name := range builtins {
		fmt.Printf("  ✓ %s (built-in)\n", name)
	}

	// Scan global plugins directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		globalPluginDir := filepath.Join(homeDir, ".buffalo", "plugins")
		if err := listPluginsInDir(globalPluginDir, "global"); err != nil {
			log.Debug("No global plugins found", logger.String("dir", globalPluginDir))
		}
	}

	// Scan local plugins directory
	localPluginDir := "./plugins"
	if err := listPluginsInDir(localPluginDir, "local"); err != nil {
		log.Debug("No local plugins found", logger.String("dir", localPluginDir))
	}

	// List plugins from config
	if cfgFile != "" || viper.ConfigFileUsed() != "" {
		log.Info("\nPlugins in configuration:")
		cfg, err := loadPluginConfig()
		if err != nil {
			log.Warn("Could not load config", logger.String("error", err.Error()))
		} else {
			if len(cfg.Plugins) == 0 {
				fmt.Println("  (none configured)")
			}
			for _, p := range cfg.Plugins {
				status := "disabled"
				if p.Enabled {
					status = "enabled"
				}
				fmt.Printf("  %s (%s)\n", p.Name, status)
			}
		}
	}

	return nil
}

// listPluginsInDir lists .so files in a directory
func listPluginsInDir(dir, location string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	found := false
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".so") {
			if !found {
				fmt.Printf("\n%s plugins (%s):\n", strings.Title(location), dir)
				found = true
			}
			name := strings.TrimSuffix(entry.Name(), ".so")
			fmt.Printf("  ✓ %s\n", name)
		}
	}

	if !found {
		return fmt.Errorf("no .so files found in %s", dir)
	}
	return nil
}

// runPluginInstall installs a plugin from URL or local file
func runPluginInstall(cmd *cobra.Command, args []string) error {
	log := GetLogger()
	source := args[0]

	if pluginName == "" {
		return fmt.Errorf("plugin name is required (use --name flag)")
	}

	// Determine target directory
	var targetDir string
	if pluginGlobal {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("could not get home directory: %w", err)
		}
		targetDir = filepath.Join(homeDir, ".buffalo", "plugins")
	} else {
		targetDir = "./plugins"
	}

	// Create directory if not exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("could not create plugin directory: %w", err)
	}

	targetPath := filepath.Join(targetDir, pluginName+".so")

	// Check if plugin already exists
	if !pluginForce {
		if _, err := os.Stat(targetPath); err == nil {
			return fmt.Errorf("plugin %s already exists (use --force to overwrite)", pluginName)
		}
	}

	// Install from URL or local file
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		log.Info("Downloading plugin from URL", logger.String("url", source))
		if err := downloadFile(targetPath, source); err != nil {
			return fmt.Errorf("could not download plugin: %w", err)
		}
	} else {
		log.Info("Copying plugin from local file", logger.String("source", source))
		if err := copyFile(source, targetPath); err != nil {
			return fmt.Errorf("could not copy plugin: %w", err)
		}
	}

	location := "local"
	if pluginGlobal {
		location = "global"
	}
	log.Info("Plugin installed successfully", logger.String("name", pluginName), logger.String("location", location), logger.String("path", targetPath))
	fmt.Printf("✓ Plugin '%s' installed to %s\n", pluginName, targetPath)
	fmt.Println("\nTo enable this plugin, add it to your buffalo.yaml:")
	fmt.Printf(`
plugins:
  - name: %s
    enabled: true
    hook_points: [pre-build, post-parse, post-build]
    priority: 100
`, pluginName)

	return nil
}

// runPluginRemove removes an installed plugin
func runPluginRemove(cmd *cobra.Command, args []string) error {
	log := GetLogger()
	name := args[0]

	// Determine target directory
	var targetDir string
	if pluginGlobal {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("could not get home directory: %w", err)
		}
		targetDir = filepath.Join(homeDir, ".buffalo", "plugins")
	} else {
		targetDir = "./plugins"
	}

	targetPath := filepath.Join(targetDir, name+".so")

	// Check if plugin exists
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return fmt.Errorf("plugin %s not found at %s", name, targetPath)
	}

	// Remove plugin file
	if err := os.Remove(targetPath); err != nil {
		return fmt.Errorf("could not remove plugin: %w", err)
	}

	location := "local"
	if pluginGlobal {
		location = "global"
	}
	log.Info("Plugin removed successfully", logger.String("name", name), logger.String("location", location))
	fmt.Printf("✓ Plugin '%s' removed from %s\n", name, targetPath)
	fmt.Println("\nNote: Plugin is still in buffalo.yaml config. Use 'buffalo plugin disable' to disable it.")

	return nil
}

// runPluginEnable enables a plugin in config
func runPluginEnable(cmd *cobra.Command, args []string) error {
	return togglePlugin(args[0], true)
}

// runPluginDisable disables a plugin in config
func runPluginDisable(cmd *cobra.Command, args []string) error {
	return togglePlugin(args[0], false)
}

// togglePlugin enables or disables a plugin in buffalo.yaml
func togglePlugin(name string, enable bool) error {
	log := GetLogger()

	// Determine config file path
	configPath := cfgFile
	if configPath == "" {
		configPath = viper.ConfigFileUsed()
	}
	if configPath == "" {
		configPath = "buffalo.yaml"
	}

	// Load config
	cfg, err := loadPluginConfig()
	if err != nil {
		// Create new config if not exists
		cfg = &config.Config{
			Plugins: []config.PluginConfig{},
		}
	}

	// Find or create plugin entry
	found := false
	for i := range cfg.Plugins {
		if cfg.Plugins[i].Name == name {
			cfg.Plugins[i].Enabled = enable
			found = true
			break
		}
	}

	if !found {
		// Add new plugin entry
		cfg.Plugins = append(cfg.Plugins, config.PluginConfig{
			Name:       name,
			Enabled:    enable,
			HookPoints: []string{"pre-build", "post-parse", "post-build"},
			Priority:   100,
			Options:    map[string]interface{}{},
		})
	}

	// Write back to file
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("could not marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("could not write config file: %w", err)
	}

	action := "disabled"
	if enable {
		action = "enabled"
	}
	log.Info("Plugin "+action, logger.String("name", name), logger.String("config", configPath))
	fmt.Printf("✓ Plugin '%s' %s in %s\n", name, action, configPath)

	return nil
}

// loadPluginConfig loads the buffalo.yaml config (renamed to avoid conflict with build.go)
func loadPluginConfig() (*config.Config, error) {
	configPath := cfgFile
	if configPath == "" {
		configPath = viper.ConfigFileUsed()
	}
	if configPath == "" {
		configPath = "buffalo.yaml"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// downloadFile downloads a file from URL
func downloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
