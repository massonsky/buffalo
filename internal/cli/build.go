package cli

import (
	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/spf13/cobra"
)

var (
	buildOutputDir string
	buildLanguages []string
	buildProtoPath []string
	buildDryRun    bool

	buildCmd = &cobra.Command{
		Use:   "build",
		Short: "Build protobuf files",
		Long: `Build protobuf files and generate code for specified languages.

Examples:
  # Build with default config
  buffalo build

  # Build for specific languages
  buffalo build --lang python,go

  # Build with custom output directory
  buffalo build --output ./generated

  # Dry run to see what would be built
  buffalo build --dry-run`,
		RunE: runBuild,
	}
)

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().StringVarP(&buildOutputDir, "output", "o", "./generated", "output directory for generated code")
	buildCmd.Flags().StringSliceVarP(&buildLanguages, "lang", "l", []string{}, "target languages (python,go,rust,cpp)")
	buildCmd.Flags().StringSliceVarP(&buildProtoPath, "proto-path", "p", []string{"."}, "paths to search for proto files")
	buildCmd.Flags().BoolVar(&buildDryRun, "dry-run", false, "show what would be built without building")
}

func runBuild(cmd *cobra.Command, args []string) error {
	log := GetLogger()

	log.Info("🔨 Starting build process")
	log.Info("Configuration",
		logger.String("output", buildOutputDir),
		logger.Any("languages", buildLanguages),
		logger.Any("proto_paths", buildProtoPath),
		logger.Bool("dry_run", buildDryRun),
	)

	if buildDryRun {
		log.Warn("🏃 Dry run mode - no files will be generated")
	}

	// TODO: Implement actual build logic in v0.3.0
	log.Warn("⚠️  Build functionality coming in v0.3.0")
	log.Info("Current status: CLI interface ready, awaiting core builder implementation")

	return nil
}
