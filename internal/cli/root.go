package cli

import (
	"fmt"
	"os"

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
	viper.AutomaticEnv()

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
}

// initLogger initializes the global logger based on flags
func initLogger() {
	level := logger.INFO
	if verbose {
		level = logger.DEBUG
	}

	log = logger.New(
		logger.WithLevel(level),
		logger.WithFormatter(logger.NewColoredFormatter()),
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
