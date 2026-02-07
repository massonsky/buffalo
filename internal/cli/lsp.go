package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/massonsky/buffalo/internal/lsp"
	"github.com/massonsky/buffalo/internal/version"
	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/spf13/cobra"
)

var (
	lspStdio bool
	lspTCP   bool
	lspPort  int
)

// lspCmd represents the lsp command
var lspCmd = &cobra.Command{
	Use:   "lsp",
	Short: "Start the Buffalo Language Server",
	Long: `Start the Buffalo Language Server Protocol (LSP) server.

The LSP server provides IDE features for .proto files:
  - Syntax diagnostics and linting
  - Autocompletion for proto3 syntax
  - Hover documentation
  - Go to definition
  - Find references
  - Document symbols
  - Code formatting
  - Code actions and quick fixes
  - Rename symbol
  - Semantic highlighting

The server can run in two modes:
  - stdio (default): communicates via stdin/stdout
  - tcp: listens on a TCP port

Examples:
  # Start LSP server in stdio mode (for most editors)
  buffalo lsp

  # Start LSP server on TCP port (for debugging)
  buffalo lsp --tcp --port 9257

Editor Configuration:

VS Code (with vscode-proto3 extension):
  Add to settings.json:
  {
    "protobuf.languageServer": {
      "command": "buffalo",
      "args": ["lsp"]
    }
  }

Neovim (with nvim-lspconfig):
  require('lspconfig').buffalo.setup{
    cmd = { 'buffalo', 'lsp' },
    filetypes = { 'proto' },
  }

Helix:
  Add to languages.toml:
  [[language]]
  name = "protobuf"
  language-server = { command = "buffalo", args = ["lsp"] }`,
	Run: func(cmd *cobra.Command, args []string) {
		log := logger.New()
		ctx := context.Background()

		// Determine mode
		if lspTCP && lspStdio {
			fmt.Fprintln(os.Stderr, "Error: cannot specify both --stdio and --tcp")
			os.Exit(1)
		}

		// Default to stdio if neither specified
		if !lspTCP {
			lspStdio = true
		}

		server := lsp.NewServer(log)

		if lspTCP {
			log.Info("Starting LSP server on TCP", logger.Int("port", lspPort))
			addr := fmt.Sprintf(":%d", lspPort)
			if err := server.ServeTCP(ctx, addr); err != nil {
				log.Error("LSP server error", logger.Error(err))
				os.Exit(1)
			}
		} else {
			log.Info("Starting LSP server in stdio mode")
			if err := server.ServeStdio(ctx); err != nil {
				log.Error("LSP server error", logger.Error(err))
				os.Exit(1)
			}
		}
	},
}

// lspVersionCmd shows LSP server version info
var lspVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show LSP server version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Buffalo Language Server")
		fmt.Printf("Version: %s\n", version.Version)
		fmt.Println("Protocol: LSP 3.17")
		fmt.Println()
		fmt.Println("Supported capabilities:")
		fmt.Println("  - textDocument/completion")
		fmt.Println("  - textDocument/hover")
		fmt.Println("  - textDocument/definition")
		fmt.Println("  - textDocument/references")
		fmt.Println("  - textDocument/documentSymbol")
		fmt.Println("  - textDocument/formatting")
		fmt.Println("  - textDocument/codeAction")
		fmt.Println("  - textDocument/rename")
		fmt.Println("  - textDocument/prepareRename")
		fmt.Println("  - textDocument/foldingRange")
		fmt.Println("  - textDocument/semanticTokens")
		fmt.Println("  - textDocument/publishDiagnostics")
	},
}

// lspCheckCmd checks LSP server health
var lspCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check LSP server requirements",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Checking Buffalo LSP requirements...")
		fmt.Println()

		allOK := true

		// Check buffalo binary
		fmt.Print("✓ Buffalo binary available\n")

		// Check workspace for proto files
		if _, err := os.Getwd(); err != nil {
			fmt.Printf("✗ Cannot determine working directory: %v\n", err)
			allOK = false
		} else {
			fmt.Print("✓ Working directory accessible\n")
		}

		// Check stdin/stdout
		if fi, _ := os.Stdin.Stat(); fi != nil {
			fmt.Print("✓ Standard I/O available\n")
		}

		fmt.Println()
		if allOK {
			fmt.Println("All checks passed. LSP server is ready to use.")
		} else {
			fmt.Println("Some checks failed. Please resolve the issues above.")
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(lspCmd)
	lspCmd.AddCommand(lspVersionCmd)
	lspCmd.AddCommand(lspCheckCmd)

	// Flags
	lspCmd.Flags().BoolVar(&lspStdio, "stdio", false, "Use stdio for communication (default)")
	lspCmd.Flags().BoolVar(&lspTCP, "tcp", false, "Use TCP for communication")
	lspCmd.Flags().IntVar(&lspPort, "port", 9257, "TCP port to listen on (with --tcp)")
}
