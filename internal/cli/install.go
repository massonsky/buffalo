package cli

import (
	"context"
	"fmt"

	"github.com/massonsky/buffalo/internal/config"
	"github.com/massonsky/buffalo/internal/dependency"
	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/spf13/cobra"
)

var (
	installForce     bool
	installUpdate    bool
	installDryRun    bool
	installVerbose   bool
	installWorkspace string
	installGitURL    string
	installGitRef    string
	installSubPath   string
)

// installCmd represents the install command.
var installCmd = &cobra.Command{
	Use:   "install [dependency-name]",
	Short: "Install proto dependencies",
	Long: `Install proto dependencies from git repositories or local sources.

Dependencies are installed to .buffalo/depends/ and automatically added to proto_path.

Examples:
  # Install from config
  buffalo install

  # Install specific dependency from git
  buffalo install googleapis --git https://github.com/googleapis/googleapis --ref master

  # Install with subpath
  buffalo install googleapis --git https://github.com/googleapis/googleapis --sub-path google

  # Update all dependencies
  buffalo install --update

  # Force reinstall
  buffalo install --force

Configuration in buffalo.yaml:
  dependencies:
    - name: googleapis
      source:
        type: git
        url: https://github.com/googleapis/googleapis
        ref: master
      sub_path: google
    
    - name: protoc-gen-validate
      source:
        type: git
        url: https://github.com/bufbuild/protoc-gen-validate
      version: v1.0.0
`,
	RunE: runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)

	installCmd.Flags().BoolVar(&installForce, "force", false, "Force reinstall even if already exists")
	installCmd.Flags().BoolVar(&installUpdate, "update", false, "Update to latest version")
	installCmd.Flags().BoolVar(&installDryRun, "dry-run", false, "Show what would be installed without installing")
	installCmd.Flags().BoolVarP(&installVerbose, "verbose", "v", false, "Verbose output")
	installCmd.Flags().StringVar(&installWorkspace, "workspace", ".buffalo", "Buffalo workspace directory")

	// Git source options
	installCmd.Flags().StringVar(&installGitURL, "git", "", "Git repository URL")
	installCmd.Flags().StringVar(&installGitRef, "ref", "", "Git ref (branch, tag, commit)")
	installCmd.Flags().StringVar(&installSubPath, "sub-path", "", "Subdirectory path within repository")
}

func runInstall(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	log := GetLogger()

	// Load config
	var cfg *config.Config
	var err error
	if cfgFile != "" {
		cfg, err = config.LoadFromFile(cfgFile)
	} else {
		cfg, err = config.Load()
	}
	if err != nil {
		// Config not required if installing from command line
		if installGitURL == "" && len(args) == 0 {
			return fmt.Errorf("no config file found and no git URL provided")
		}
		cfg = &config.Config{}
	}

	// Create dependency manager
	manager, err := dependency.NewManager(installWorkspace, log)
	if err != nil {
		return fmt.Errorf("failed to create dependency manager: %w", err)
	}

	opts := dependency.InstallOptions{
		Force:        installForce,
		Update:       installUpdate,
		DryRun:       installDryRun,
		Verbose:      installVerbose,
		WorkspaceDir: installWorkspace,
	}

	// Case 1: Install specific dependency from command line
	if len(args) > 0 && installGitURL != "" {
		return installFromCLI(ctx, manager, args[0], opts, log)
	}

	// Case 2: Install specific dependency from config
	if len(args) > 0 {
		return installFromConfig(ctx, manager, cfg, args[0], opts, log)
	}

	// Case 3: Install all dependencies from config
	return installAll(ctx, manager, cfg, opts, log)
}

func installFromCLI(ctx context.Context, manager *dependency.Manager, name string, opts dependency.InstallOptions, log *logger.Logger) error {
	log.Info("Installing from CLI",
		logger.String("name", name),
		logger.String("url", installGitURL))

	dep := dependency.Dependency{
		Name: name,
		Source: dependency.DependencySource{
			Type: "git",
			URL:  installGitURL,
			Ref:  installGitRef,
		},
		SubPath: installSubPath,
	}

	if installDryRun {
		log.Info("[DRY RUN] Would install",
			logger.String("name", name),
			logger.String("source", installGitURL))
		return nil
	}

	result, err := manager.Install(ctx, dep, opts)
	if err != nil {
		return err
	}

	printInstallResult(result, log)
	return nil
}

func installFromConfig(ctx context.Context, manager *dependency.Manager, cfg *config.Config, name string, opts dependency.InstallOptions, log *logger.Logger) error {
	if cfg.Dependencies == nil || len(cfg.Dependencies) == 0 {
		return fmt.Errorf("no dependencies found in config")
	}

	// Find dependency in config
	var dep *dependency.Dependency
	for _, d := range cfg.Dependencies {
		if d.Name == name {
			dep = &d
			break
		}
	}

	if dep == nil {
		return fmt.Errorf("dependency %s not found in config", name)
	}

	if installDryRun {
		log.Info("[DRY RUN] Would install",
			logger.String("name", name),
			logger.String("source", dep.Source.URL))
		return nil
	}

	result, err := manager.Install(ctx, *dep, opts)
	if err != nil {
		return err
	}

	printInstallResult(result, log)
	return nil
}

func installAll(ctx context.Context, manager *dependency.Manager, cfg *config.Config, opts dependency.InstallOptions, log *logger.Logger) error {
	if cfg.Dependencies == nil || len(cfg.Dependencies) == 0 {
		log.Info("No dependencies to install")
		return nil
	}

	log.Info("Installing dependencies from config",
		logger.Int("count", len(cfg.Dependencies)))

	if installDryRun {
		for _, dep := range cfg.Dependencies {
			log.Info("[DRY RUN] Would install",
				logger.String("name", dep.Name),
				logger.String("source", dep.Source.URL))
		}
		return nil
	}

	results, err := manager.InstallAll(ctx, cfg.Dependencies, opts)
	if err != nil {
		return err
	}

	log.Info("=== Installation Summary ===")
	for _, result := range results {
		printInstallResult(result, log)
	}

	log.Info("All dependencies installed successfully",
		logger.String("location", installWorkspace+"/depends"),
		logger.String("lockfile", installWorkspace+"/buffalo.lock"))

	return nil
}

func printInstallResult(result *dependency.DownloadResult, log *logger.Logger) {
	if result == nil {
		return
	}

	log.Info("✓ Installed",
		logger.String("name", result.Name),
		logger.String("version", result.Version),
		logger.String("path", result.LocalPath))
}
