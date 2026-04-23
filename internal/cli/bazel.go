package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/massonsky/buffalo/internal/bazel"
	"github.com/massonsky/buffalo/pkg/errors"
	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/spf13/cobra"
)

var (
	bazelGenDryRun  bool
	bazelGenOut     string
	bazelGenPattern []string
)

var (
	bazelCmd = &cobra.Command{
		Use:   "bazel",
		Short: "Bazel integration helpers (BUILD generation, target discovery)",
		Long: `Bazel integration commands.

Use these to drive Buffalo from a Bazel-managed monorepo without running a
full 'buffalo build'. Most flows pair 'buffalo bazel gen-build' (regenerate
BUILD.bazel files for generated code) with the buffalo_proto_compile rule
shipped in //bazel/rules_buffalo.`,
	}

	bazelGenBuildCmd = &cobra.Command{
		Use:   "gen-build",
		Short: "Generate BUILD.bazel files for generated code from buffalo.yaml",
		Long: `Read buffalo.yaml, discover proto_library / proto-bearing filegroup targets
in the current Bazel workspace, and emit (or refresh) BUILD.bazel files
under the configured output.base_dir for every enabled language.

Existing non-Buffalo BUILD files are preserved (only files carrying the
Buffalo header marker are overwritten).`,
		Example: `  # Regenerate BUILD files in place
  buffalo bazel gen-build

  # Preview without touching the filesystem
  buffalo bazel gen-build --dry-run

  # Override patterns and output dir
  buffalo bazel gen-build --pattern //proto/... --out gen/`,
		RunE: runBazelGenBuild,
	}
)

func init() {
	rootCmd.AddCommand(bazelCmd)
	bazelCmd.AddCommand(bazelGenBuildCmd)

	bazelGenBuildCmd.Flags().BoolVar(&bazelGenDryRun, "dry-run", false,
		"print the planned BUILD files to stdout without writing them")
	bazelGenBuildCmd.Flags().StringVar(&bazelGenOut, "out", "",
		"override output.base_dir from buffalo.yaml")
	bazelGenBuildCmd.Flags().StringSliceVar(&bazelGenPattern, "pattern", nil,
		"Bazel patterns to query (e.g. //proto/...); defaults to bazel.patterns from buffalo.yaml")
}

func runBazelGenBuild(cmd *cobra.Command, args []string) error {
	log := GetLogger()

	cfg, cfgPath, err := loadConfigWithPath(log)
	if err != nil {
		return err
	}

	root := cfg.ConfigDir
	if root == "" {
		root = filepath.Dir(cfgPath)
	}
	if root == "" || root == "." {
		root, _ = os.Getwd()
	}

	integrator, err := bazel.NewIntegrator(root)
	if err != nil {
		return errors.Wrap(err, errors.ErrConfig,
			fmt.Sprintf("bazel workspace not detected at %s", root))
	}
	if integrator == nil {
		return errors.New(errors.ErrConfig,
			fmt.Sprintf("bazel workspace not detected at %s (no MODULE.bazel/WORKSPACE)", root))
	}
	if cfg.Bazel.BazelPath != "" {
		integrator.SetBazelPath(cfg.Bazel.BazelPath)
	}

	patterns := bazelGenPattern
	if len(patterns) == 0 {
		patterns = cfg.Bazel.Patterns
	}
	if len(patterns) == 0 {
		patterns = []string{"//..."}
	}

	languages := cfg.GetEnabledLanguages()
	if len(languages) == 0 {
		return errors.New(errors.ErrConfig, "no enabled languages in buffalo.yaml")
	}
	sort.Strings(languages)

	outputDir := cfg.Output.BaseDir
	if bazelGenOut != "" {
		outputDir = bazelGenOut
	}

	ctx := context.Background()
	targets, err := integrator.DiscoverProtoTargets(ctx, patterns)
	if err != nil {
		return errors.Wrap(err, errors.ErrIO, "discover proto targets")
	}
	log.Info("🎯 Discovered proto targets",
		logger.Int("count", len(targets)),
		logger.String("mode", string(integrator.GetSyncMode())),
		logger.Any("languages", languages),
	)

	plan, err := integrator.CreateSyncPlan(ctx, targets, languages, outputDir)
	if err != nil {
		return err
	}

	if len(plan.BuildFilesToGenerate) == 0 {
		log.Info("Nothing to generate (no matching targets / all preserved).")
		return nil
	}

	if bazelGenDryRun {
		writeDryRun(plan.BuildFilesToGenerate, root)
		log.Info("ℹ️  Dry run — no files written",
			logger.Int("planned", len(plan.BuildFilesToGenerate)))
		return nil
	}

	if err := integrator.WriteBuildFiles(plan.BuildFilesToGenerate); err != nil {
		return errors.Wrap(err, errors.ErrFileWrite, "write BUILD.bazel")
	}

	log.Info("✅ BUILD files written",
		logger.Int("count", len(plan.BuildFilesToGenerate)))
	for _, b := range plan.BuildFilesToGenerate {
		rel, err := filepath.Rel(root, b.Path)
		if err != nil {
			rel = b.Path
		}
		log.Info("   " + rel)
	}
	return nil
}

func writeDryRun(builds []bazel.GeneratedBuild, root string) {
	for _, b := range builds {
		rel, err := filepath.Rel(root, b.Path)
		if err != nil {
			rel = b.Path
		}
		fmt.Fprintf(os.Stdout, "# === %s ===\n%s\n",
			rel, strings.TrimRight(b.Content, "\n"))
		fmt.Fprintln(os.Stdout)
	}
}
