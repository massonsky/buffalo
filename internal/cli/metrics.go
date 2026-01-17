package cli

import (
	"fmt"

	"github.com/massonsky/buffalo/internal/metrics"
	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/spf13/cobra"
)

var (
	metricsCmd = &cobra.Command{
		Use:   "metrics",
		Short: "View build metrics and statistics",
		Long: `View build metrics and statistics collected during Buffalo builds.

Metrics include:
  - Build duration and performance
  - Files processed and generated
  - Cache hit/miss rates
  - Language breakdown
  - Error and warning counts`,
		Example: `  # Show last build metrics
  buffalo metrics show

  # Show build history
  buffalo metrics history

  # Show metrics in JSON format
  buffalo metrics show --format json`,
	}

	metricsShowCmd = &cobra.Command{
		Use:   "show",
		Short: "Show last build metrics",
		Long:  "Display metrics from the most recent build",
		RunE:  runMetricsShow,
	}

	metricsHistoryCmd = &cobra.Command{
		Use:   "history",
		Short: "Show build metrics history",
		Long:  "Display metrics history from recent builds",
		RunE:  runMetricsHistory,
	}

	metricsFormat string
	metricsLimit  int
)

func init() {
	rootCmd.AddCommand(metricsCmd)

	metricsCmd.AddCommand(metricsShowCmd)
	metricsCmd.AddCommand(metricsHistoryCmd)

	// Show command flags
	metricsShowCmd.Flags().StringVar(&metricsFormat, "format", "text", "output format: text, json")

	// History command flags
	metricsHistoryCmd.Flags().IntVarP(&metricsLimit, "limit", "n", 10, "number of builds to show")
	metricsHistoryCmd.Flags().StringVar(&metricsFormat, "format", "text", "output format: text, json")
}

func runMetricsShow(cmd *cobra.Command, args []string) error {
	log := GetLogger()

	// Load config to get metrics directory
	cfg, err := loadConfig(log)
	if err != nil {
		log.Warn("Failed to load config, using default metrics directory", logger.Any("error", err))
	}

	metricsDir := ".buffalo/metrics"
	if cfg != nil && cfg.Build.Cache.Directory != "" {
		metricsDir = cfg.Build.Cache.Directory + "/metrics"
	}

	// Create metrics store
	store, err := metrics.NewStore(metricsDir)
	if err != nil {
		log.Error("Failed to create metrics store", logger.Any("error", err))
		return err
	}

	// Load latest metrics
	m, err := store.LoadLatest()
	if err != nil {
		log.Info("No build metrics found")
		log.Info("\nRun 'buffalo build --metrics' to collect build metrics")
		return nil
	}

	// Display metrics
	switch metricsFormat {
	case "json":
		// TODO: JSON output
		log.Info("JSON format not yet implemented")
	default:
		fmt.Println(metrics.FormatMetrics(m))
	}

	return nil
}

func runMetricsHistory(cmd *cobra.Command, args []string) error {
	log := GetLogger()

	// Load config to get metrics directory
	cfg, err := loadConfig(log)
	if err != nil {
		log.Warn("Failed to load config, using default metrics directory", logger.Any("error", err))
	}

	metricsDir := ".buffalo/metrics"
	if cfg != nil && cfg.Build.Cache.Directory != "" {
		metricsDir = cfg.Build.Cache.Directory + "/metrics"
	}

	// Create metrics store
	store, err := metrics.NewStore(metricsDir)
	if err != nil {
		log.Error("Failed to create metrics store", logger.Any("error", err))
		return err
	}

	// Load metrics history
	history, err := store.List(metricsLimit)
	if err != nil {
		log.Error("Failed to load metrics history", logger.Any("error", err))
		return err
	}

	if len(history) == 0 {
		log.Info("No build metrics history found")
		log.Info("\nRun 'buffalo build --metrics' to collect build metrics")
		return nil
	}

	log.Info(fmt.Sprintf("📊 Build History (last %d builds)\n", len(history)))

	for i, m := range history {
		status := "✅"
		if m.FailedFiles > 0 {
			status = "❌"
		} else if m.WarningCount > 0 {
			status = "⚠️"
		}

		fmt.Printf("%s %d. %s\n", status, i+1, m.BuildID)
		fmt.Printf("   Duration: %.2fs | Files: %d processed, %d generated\n",
			m.Duration, m.ProcessedFiles, m.GeneratedFiles)
		fmt.Printf("   Cache: %.1f%% hit rate | Errors: %d, Warnings: %d\n",
			m.CacheHitRate, m.ErrorCount, m.WarningCount)
		fmt.Println()
	}

	return nil
}
