package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/massonsky/buffalo/internal/config"
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

It supports Python, Go, Rust, C++ ant TypeScript code generation with intelligent caching,
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

	// Read in environment variables that match. ApplyEnv sets prefix,
	// the dotted-key replacer and binds every nested key declared in
	// internal/config so overrides like BUFFALO_LANGUAGES_GO_ENABLED work
	// even without a buffalo.yaml on disk.
	config.ApplyEnv(viper.GetViper())

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
	formatEnv := strings.ToLower(strings.TrimSpace(os.Getenv("BUFFALO_LOG_FORMAT")))
	if formatEnv == "" && isCIEnvironment() {
		// CI runners typically don't render ANSI well and structured logs
		// are easier to ingest.
		formatEnv = "json"
	}
	switch formatEnv {
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

// isCIEnvironment returns true when the process runs under a typical CI
// system. It checks the de-facto-standard CI=true variable plus common
// vendor-specific markers so that pipelines without CI= still get JSON logs.
func isCIEnvironment() bool {
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("CI"))); v == "true" || v == "1" {
		return true
	}
	for _, k := range []string{
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"BUILDKITE",
		"CIRCLECI",
		"TF_BUILD", // Azure Pipelines
		"TEAMCITY_VERSION",
		"JENKINS_URL",
		"BITBUCKET_BUILD_NUMBER",
	} {
		if os.Getenv(k) != "" {
			return true
		}
	}
	return false
}

// GetLogger returns the global logger
func GetLogger() *logger.Logger {
	if log == nil {
		initLogger()
	}
	return log
}
