// Package tools provides language tools installation for Buffalo.
package tools

import "runtime"

// Tool represents a build tool that can be installed.
type Tool struct {
	Name           string                 // Tool name (e.g., "protoc-gen-go")
	Description    string                 // Human-readable description
	Language       string                 // Associated language (go, python, rust, cpp, all)
	CheckCommand   string                 // Command to check if installed
	CheckArgs      []string               // Arguments for check command
	CheckFunc      func() (string, error) // Custom check function (returns version)
	InstallMethods map[string]string      // Platform-specific install commands (linux, darwin, windows)
	PostInstall    string                 // Post-installation message or command
	Critical       bool                   // Is this tool required for the language
	URL            string                 // Official documentation URL
}

// InstallResult represents the result of a tool installation.
type InstallResult struct {
	Tool      Tool
	Success   bool
	Version   string
	Message   string
	Error     error
	Skipped   bool
	AlreadyOK bool
}

// InstallOptions configures tool installation.
type InstallOptions struct {
	Languages   []string // Languages to install tools for (empty = all enabled)
	Force       bool     // Force reinstall even if already installed
	DryRun      bool     // Only show what would be installed
	Verbose     bool     // Verbose output
	Interactive bool     // Ask for confirmation before install
	IncludeAll  bool     // Install all tools, not just critical
}

// GetPlatform returns the current platform identifier.
func GetPlatform() string {
	switch runtime.GOOS {
	case "darwin":
		return "darwin"
	case "windows":
		return "windows"
	default:
		return "linux"
	}
}

// ToolRegistry holds all available tools.
var ToolRegistry = []Tool{
	// Core tools (all languages)
	{
		Name:         "protoc",
		Description:  "Protocol Buffers Compiler",
		Language:     "all",
		CheckCommand: "protoc",
		CheckArgs:    []string{"--version"},
		InstallMethods: map[string]string{
			"linux":   "sudo apt install -y protobuf-compiler || sudo dnf install -y protobuf-compiler || sudo pacman -S protobuf",
			"darwin":  "brew install protobuf",
			"windows": "scoop install protobuf",
		},
		Critical: true,
		URL:      "https://github.com/protocolbuffers/protobuf/releases",
	},

	// Go tools
	{
		Name:         "go",
		Description:  "Go Programming Language",
		Language:     "go",
		CheckCommand: "go",
		CheckArgs:    []string{"version"},
		InstallMethods: map[string]string{
			"linux":   "wget https://go.dev/dl/go1.24.0.linux-amd64.tar.gz && sudo tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz && export PATH=$PATH:/usr/local/go/bin",
			"darwin":  "brew install go",
			"windows": "scoop install go",
		},
		Critical: true,
		URL:      "https://golang.org/dl/",
	},
	{
		Name:         "protoc-gen-go",
		Description:  "Go Protocol Buffers Generator",
		Language:     "go",
		CheckCommand: "protoc-gen-go",
		CheckArgs:    []string{"--version"},
		InstallMethods: map[string]string{
			"linux":   "go install google.golang.org/protobuf/cmd/protoc-gen-go@latest",
			"darwin":  "go install google.golang.org/protobuf/cmd/protoc-gen-go@latest",
			"windows": "go install google.golang.org/protobuf/cmd/protoc-gen-go@latest",
		},
		PostInstall: "Make sure $GOPATH/bin is in your PATH",
		Critical:    true,
		URL:         "https://pkg.go.dev/google.golang.org/protobuf/cmd/protoc-gen-go",
	},
	{
		Name:         "protoc-gen-go-grpc",
		Description:  "Go gRPC Generator",
		Language:     "go",
		CheckCommand: "protoc-gen-go-grpc",
		CheckArgs:    []string{"--version"},
		InstallMethods: map[string]string{
			"linux":   "go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest",
			"darwin":  "go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest",
			"windows": "go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest",
		},
		PostInstall: "Make sure $GOPATH/bin is in your PATH",
		Critical:    true,
		URL:         "https://pkg.go.dev/google.golang.org/grpc/cmd/protoc-gen-go-grpc",
	},
	{
		Name:         "protoc-gen-grpc-gateway",
		Description:  "gRPC-Gateway Generator (REST API proxy)",
		Language:     "go",
		CheckCommand: "protoc-gen-grpc-gateway",
		CheckArgs:    []string{"--version"},
		InstallMethods: map[string]string{
			"linux":   "go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest",
			"darwin":  "go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest",
			"windows": "go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest",
		},
		Critical: false,
		URL:      "https://github.com/grpc-ecosystem/grpc-gateway",
	},
	{
		Name:         "protoc-gen-openapiv2",
		Description:  "OpenAPI v2 Generator for gRPC-Gateway",
		Language:     "go",
		CheckCommand: "protoc-gen-openapiv2",
		CheckArgs:    []string{"--version"},
		InstallMethods: map[string]string{
			"linux":   "go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest",
			"darwin":  "go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest",
			"windows": "go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest",
		},
		Critical: false,
		URL:      "https://github.com/grpc-ecosystem/grpc-gateway",
	},
	{
		Name:         "protoc-gen-validate",
		Description:  "Protocol Buffer Validation Generator",
		Language:     "go",
		CheckCommand: "protoc-gen-validate",
		CheckArgs:    []string{"--version"},
		InstallMethods: map[string]string{
			"linux":   "go install github.com/bufbuild/protoc-gen-validate@latest",
			"darwin":  "go install github.com/bufbuild/protoc-gen-validate@latest",
			"windows": "go install github.com/bufbuild/protoc-gen-validate@latest",
		},
		Critical: false,
		URL:      "https://github.com/bufbuild/protoc-gen-validate",
	},

	// Python tools
	{
		Name:         "python3",
		Description:  "Python Programming Language",
		Language:     "python",
		CheckCommand: "python3",
		CheckArgs:    []string{"--version"},
		InstallMethods: map[string]string{
			"linux":   "sudo apt install -y python3 python3-pip || sudo dnf install -y python3 python3-pip",
			"darwin":  "brew install python3",
			"windows": "scoop install python",
		},
		Critical: true,
		URL:      "https://www.python.org/downloads/",
	},
	{
		Name:         "pip",
		Description:  "Python Package Installer",
		Language:     "python",
		CheckCommand: "pip3",
		CheckArgs:    []string{"--version"},
		InstallMethods: map[string]string{
			"linux":   "sudo apt install -y python3-pip || python3 -m ensurepip --upgrade",
			"darwin":  "python3 -m ensurepip --upgrade",
			"windows": "python -m ensurepip --upgrade",
		},
		Critical: true,
		URL:      "https://pip.pypa.io/en/stable/installation/",
	},
	{
		Name:        "grpcio-tools",
		Description: "Python gRPC Tools (protoc plugin)",
		Language:    "python",
		InstallMethods: map[string]string{
			"linux":   "pip3 install grpcio-tools",
			"darwin":  "pip3 install grpcio-tools",
			"windows": "pip install grpcio-tools",
		},
		Critical: true,
		URL:      "https://grpc.io/docs/languages/python/quickstart/",
	},
	{
		Name:        "grpcio",
		Description: "Python gRPC Runtime",
		Language:    "python",
		InstallMethods: map[string]string{
			"linux":   "pip3 install grpcio",
			"darwin":  "pip3 install grpcio",
			"windows": "pip install grpcio",
		},
		Critical: true,
		URL:      "https://grpc.io/docs/languages/python/",
	},
	{
		Name:        "protobuf",
		Description: "Python Protocol Buffers Library",
		Language:    "python",
		InstallMethods: map[string]string{
			"linux":   "pip3 install protobuf",
			"darwin":  "pip3 install protobuf",
			"windows": "pip install protobuf",
		},
		Critical: true,
		URL:      "https://developers.google.com/protocol-buffers/docs/pythontutorial",
	},
	{
		Name:        "mypy-protobuf",
		Description: "Python Type Stubs for Protobuf",
		Language:    "python",
		InstallMethods: map[string]string{
			"linux":   "pip3 install mypy-protobuf",
			"darwin":  "pip3 install mypy-protobuf",
			"windows": "pip install mypy-protobuf",
		},
		Critical: false,
		URL:      "https://github.com/nipunn1313/mypy-protobuf",
	},

	// Rust tools
	{
		Name:         "rustc",
		Description:  "Rust Compiler",
		Language:     "rust",
		CheckCommand: "rustc",
		CheckArgs:    []string{"--version"},
		InstallMethods: map[string]string{
			"linux":   "curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y",
			"darwin":  "curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y",
			"windows": "Download and run rustup-init.exe from https://rustup.rs/",
		},
		PostInstall: "Run: source $HOME/.cargo/env",
		Critical:    true,
		URL:         "https://rustup.rs/",
	},
	{
		Name:         "cargo",
		Description:  "Rust Package Manager",
		Language:     "rust",
		CheckCommand: "cargo",
		CheckArgs:    []string{"--version"},
		InstallMethods: map[string]string{
			"linux":   "Installed with rustup",
			"darwin":  "Installed with rustup",
			"windows": "Installed with rustup",
		},
		Critical: true,
		URL:      "https://doc.rust-lang.org/cargo/",
	},
	{
		Name:         "protobuf-codegen",
		Description:  "Rust Protobuf Code Generator",
		Language:     "rust",
		CheckCommand: "protoc-gen-rust",
		CheckArgs:    []string{"--version"},
		InstallMethods: map[string]string{
			"linux":   "cargo install protobuf-codegen",
			"darwin":  "cargo install protobuf-codegen",
			"windows": "cargo install protobuf-codegen",
		},
		Critical: true,
		URL:      "https://crates.io/crates/protobuf-codegen",
	},

	// C++ tools
	{
		Name:         "g++",
		Description:  "GNU C++ Compiler",
		Language:     "cpp",
		CheckCommand: "g++",
		CheckArgs:    []string{"--version"},
		InstallMethods: map[string]string{
			"linux":   "sudo apt install -y g++ build-essential || sudo dnf install -y gcc-c++",
			"darwin":  "xcode-select --install",
			"windows": "scoop install gcc",
		},
		Critical: true,
		URL:      "https://gcc.gnu.org/",
	},
	{
		Name:         "cmake",
		Description:  "CMake Build System",
		Language:     "cpp",
		CheckCommand: "cmake",
		CheckArgs:    []string{"--version"},
		InstallMethods: map[string]string{
			"linux":   "sudo apt install -y cmake || sudo dnf install -y cmake",
			"darwin":  "brew install cmake",
			"windows": "scoop install cmake",
		},
		Critical: true,
		URL:      "https://cmake.org/",
	},
	{
		Name:         "libprotobuf-dev",
		Description:  "Protocol Buffers C++ Development Files",
		Language:     "cpp",
		CheckCommand: "pkg-config",
		CheckArgs:    []string{"--exists", "protobuf"},
		InstallMethods: map[string]string{
			"linux":   "sudo apt install -y libprotobuf-dev || sudo dnf install -y protobuf-devel",
			"darwin":  "brew install protobuf",
			"windows": "vcpkg install protobuf",
		},
		Critical: true,
		URL:      "https://github.com/protocolbuffers/protobuf",
	},
	{
		Name:         "grpc-dev",
		Description:  "gRPC C++ Development Files",
		Language:     "cpp",
		CheckCommand: "pkg-config",
		CheckArgs:    []string{"--exists", "grpc++"},
		InstallMethods: map[string]string{
			"linux":   "sudo apt install -y libgrpc++-dev || Build from source: https://grpc.io/docs/languages/cpp/quickstart/",
			"darwin":  "brew install grpc",
			"windows": "vcpkg install grpc",
		},
		Critical: true,
		URL:      "https://grpc.io/docs/languages/cpp/quickstart/",
	},
}

// GetToolsForLanguage returns tools for a specific language.
func GetToolsForLanguage(lang string) []Tool {
	var result []Tool
	for _, tool := range ToolRegistry {
		if tool.Language == lang || tool.Language == "all" {
			result = append(result, tool)
		}
	}
	return result
}

// GetCriticalTools returns only critical tools for a language.
func GetCriticalTools(lang string) []Tool {
	var result []Tool
	for _, tool := range ToolRegistry {
		if (tool.Language == lang || tool.Language == "all") && tool.Critical {
			result = append(result, tool)
		}
	}
	return result
}

// GetAllCriticalTools returns all critical tools.
func GetAllCriticalTools() []Tool {
	var result []Tool
	for _, tool := range ToolRegistry {
		if tool.Critical {
			result = append(result, tool)
		}
	}
	return result
}
