package cli

import (
	"fmt"

	"github.com/massonsky/buffalo/internal/version"
	"github.com/spf13/cobra"
)

var (
	versionShort bool

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  `Print detailed version information about Buffalo`,
		Run:   runVersion,
	}
)

func init() {
	rootCmd.AddCommand(versionCmd)
	versionCmd.Flags().BoolVarP(&versionShort, "short", "s", false, "print only the version number")
}

func runVersion(cmd *cobra.Command, args []string) {
	if versionShort {
		fmt.Println(version.Short())
		return
	}

	fmt.Printf("🦬 Buffalo - Protobuf/gRPC Multi-Language Builder\n\n")
	fmt.Printf("Version:    %s\n", version.FullVersion())
	fmt.Printf("Commit:     %s\n", version.GitCommit)
	fmt.Printf("Build Date: %s\n", version.BuildDate)
	fmt.Printf("Go Version: %s\n", version.GoVersion)
	fmt.Printf("Platform:   %s\n", version.Platform)
	fmt.Printf("\nInstall:    go install github.com/massonsky/buffalo/cmd/buffalo@v%s\n", version.Version)
}
