package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/massonsky/buffalo/internal/version"
	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	verbose bool
	log     *logger.Logger
	rootCmd = &cobra.Command{
		Use:   "buffalo",
		Short: "🦬 Buffalo - Protobuf/gRPC Multi-Language Builder",
		Long: `Buffalo is a cross-platform, multi-language builder for protobuf and gRPC files.

It supports Python, Go, Rust, and C++ code generation with intelligent caching,
parallel compilation, and incremental builds.`,
		Version: version.Version,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			initLogger()
		},
	}
)

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./buffalo.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Version template
	rootCmd.SetVersionTemplate(fmt.Sprintf(
		"Buffalo version %s\nCommit: %s\nGo: %s\nPlatform: %s\n",
		version.Version,
		version.GitCommit,
		version.GoVersion,
		version.Platform,
	))
}

// initConfig reads in config file and ENV variables
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config in current directory
		viper.AddConfigPath(".")
		viper.SetConfigName("buffalo")
		viper.SetConfigType("yaml")
	}

	// Read in environment variables that match
	viper.SetEnvPrefix("BUFFALO")
	// Map nested keys (foo.bar.baz) to env vars (BUFFALO_FOO_BAR_BAZ).
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Explicit binds for nested keys that AutomaticEnv cannot discover until
	// the YAML is loaded. This makes overrides like
	//   BUFFALO_LANGUAGES_GO_ENABLED=true
	//   BUFFALO_LANGUAGES_PYTHON_PACKAGE=myproto
	// work even when no buffalo.yaml is present.
	for _, k := range envBindKeys {
		_ = viper.BindEnv(k)
	}

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
}

// initLogger initializes the global logger based on flags and environment.
//
// Honored env vars (case-insensitive):
//
//	BUFFALO_LOG_FORMAT  text | json | colored | color  (default: colored)
//	BUFFALO_LOG_LEVEL   debug | info | warn | error             (default: info; -v forces debug)
func initLogger() {
	level := logger.INFO
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("BUFFALO_LOG_LEVEL"))); v != "" {
		switch v {
		case "debug":
			level = logger.DEBUG
		case "info":
			level = logger.INFO
		case "warn", "warning":
			level = logger.WARN
		case "error":
			level = logger.ERROR
		}
	}
	if verbose {
		level = logger.DEBUG
	}

	var fmt logger.Formatter = logger.NewColoredFormatter()
	switch strings.ToLower(strings.TrimSpace(os.Getenv("BUFFALO_LOG_FORMAT"))) {
	case "text":
		fmt = logger.NewTextFormatter()
	case "json":
		fmt = logger.NewJSONFormatter()
	case "", "colored", "color":
		// keep default
	}

	log = logger.New(
		logger.WithLevel(level),
		logger.WithFormatter(fmt),
		logger.WithOutput(logger.NewStdoutOutput()),
	)
}

// GetLogger returns the global logger
func GetLogger() *logger.Logger {
	if log == nil {
		initLogger()
	}
	return log
}

// envBindKeys lists nested viper keys that should be bound to BUFFALO_*
// environment variables eagerly (before any YAML is loaded). Keep in sync
// with internal/config.Config when new fields are added.
var envBindKeys = []string{
	"project.name",
	"project.version",
	"output.base_dir",
	"build.workers",
	"build.cache.enabled",
	"build.cache.directory",
	"logging.level",
	"logging.format",
	"logging.output",

	"languages.go.enabled",
	"languages.go.module",
	"languages.go.generator",

	"languages.python.enabled",
	"languages.python.package",
	"languages.python.generator",
	"languages.python.workdir",

	"languages.rust.enabled",
	"languages.rust.generator",

	"languages.cpp.enabled",
	"languages.cpp.namespace",

	"languages.typescript.enabled",
	"languages.typescript.generator",
	"languages.typescript.output",

	"bazel.enabled",
	"bazel.bazel_path",
	"bazel.auto_detect",
}
