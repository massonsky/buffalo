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
		fmt.Println(version.Version)
		return
	}

	fmt.Printf("🦬 Buffalo - Protobuf/gRPC Multi-Language Builder\n\n")
	fmt.Printf("Version:    %s\n", version.Version)
	fmt.Printf("Commit:     %s\n", version.GitCommit)
	fmt.Printf("Build Date: %s\n", version.BuildDate)
	fmt.Printf("Go Version: %s\n", version.GoVersion)
	fmt.Printf("Platform:   %s\n", version.Platform)
	fmt.Println()
	fmt.Println("✅ Available Infrastructure:")
	fmt.Println("  • Logger:  Structured logging system")
	fmt.Println("  • Errors:  Enhanced error handling")
	fmt.Println("  • Utils:   File operations & validation")
	fmt.Println("  • Metrics: Performance monitoring")
	fmt.Println()
	fmt.Println("✅ Core Builder (v0.3.0):")
	fmt.Println("  • Proto Parser:       Parse .proto files")
	fmt.Println("  • Dependency Resolver: Topological sort")
	fmt.Println("  • Executor:           Parallel compilation")
	fmt.Println("  • Cache Manager:      Incremental builds")
	fmt.Println()
	fmt.Println("✅ Language Compilers (v0.4.0):")
	fmt.Println("  • Python Compiler:    protoc + grpcio-tools")
	fmt.Println("  • Auto __init__.py:   Package generation")
	fmt.Println("  • Protobuf & gRPC:    Full support")
	fmt.Println()
	fmt.Println("🚧 Coming Soon (v0.5.0+):")
	fmt.Println("  • Go Compiler:        protoc-gen-go + grpc")
	fmt.Println("  • Rust Compiler:      prost + tonic")
	fmt.Println("  • C++ Compiler:       protoc + grpc++")
}
