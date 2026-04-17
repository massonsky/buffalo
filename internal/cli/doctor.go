package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/massonsky/buffalo/internal/config"
	"github.com/massonsky/buffalo/internal/system"
	"github.com/massonsky/buffalo/internal/version"
	"github.com/massonsky/buffalo/pkg/logger"
	"github.com/spf13/cobra"
)

var (
	doctorVerbose    bool
	doctorConfigOnly bool
	doctorCmd        = &cobra.Command{
		Use:   "doctor",
		Short: "Check Buffalo environment and dependencies",
		Long: `Check your Buffalo environment for common issues.

This command verifies:
  - Buffalo installation
  - protoc compiler
  - Language-specific code generators (protoc-gen-go, grpc_tools, etc.)
  - Config file validity
  - Dependency installations

By default, checks ALL supported languages. Use --config-only flag to check 
only languages enabled in your buffalo.yaml configuration.

Returns exit code 0 if all checks pass, 1 if any critical checks fail.`,
		RunE: runDoctor,
	}
)

func init() {
	rootCmd.AddCommand(doctorCmd)
	doctorCmd.Flags().BoolVarP(&doctorVerbose, "verbose", "v", false, "show detailed diagnostic information")
	doctorCmd.Flags().BoolVarP(&doctorConfigOnly, "config-only", "c", false, "check only languages enabled in buffalo.yaml")
}

// CheckResult represents the result of a diagnostic check
type CheckResult struct {
	Name     string
	Status   string // "pass", "fail", "warn", "skip"
	Message  string
	Details  string
	Critical bool // If true, failure should result in non-zero exit code
}

func runDoctor(cmd *cobra.Command, args []string) error {
	log := GetLogger()

	log.Info("🏥 Running Buffalo Doctor - Environment Diagnostic")
	log.Info("═══════════════════════════════════════════════════")

	results := []CheckResult{}

	// Check 1: Buffalo Version
	results = append(results, checkBuffaloVersion())

	// Check 2: Operating System
	results = append(results, checkOS())

	// Если используется флаг --config-only, проверяем только включенные языки
	if doctorConfigOnly {
		cfg, err := loadConfigForDoctor()
		if err != nil {
			log.Warn(fmt.Sprintf("⚠️  Не удалось загрузить конфигурацию: %v", err))
			log.Info("Выполняется проверка всех языков...")
		} else {
			log.Info("📋 Проверка готовности для языков из конфигурации...")
			results = append(results, runSystemCheck(cfg)...)

			// Проверка конфига и зависимостей
			results = append(results, checkConfig())
			results = append(results, checkDependencies())

			// Выводим результаты и завершаем
			return printDoctorResults(results, log)
		}
	}

	// Стандартная проверка всех языков
	// Check 3: protoc compiler
	results = append(results, checkProtoc())

	// Check 4: Go installation and protoc-gen-go
	results = append(results, checkGo()...)

	// Check 5: Python installation and grpc_tools
	results = append(results, checkPython()...)

	// Check 6: Rust installation and prost/tonic
	results = append(results, checkRust()...)

	// Check 7: C++ installation and protobuf
	results = append(results, checkCpp()...)

	// Check 8: Config file
	results = append(results, checkConfig())

	// Check 9: Dependencies
	results = append(results, checkDependencies())

	return printDoctorResults(results, log)
}

// loadConfigForDoctor загружает конфигурацию для команды doctor
func loadConfigForDoctor() (*config.Config, error) {
	cfg, err := loadConfig(GetLogger())
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// runSystemCheck выполняет проверку системы с помощью system.SystemChecker
func runSystemCheck(cfg *config.Config) []CheckResult {
	checker := system.NewSystemChecker(cfg)
	sysResults, err := checker.CheckReadiness()

	results := []CheckResult{}

	if err != nil {
		results = append(results, CheckResult{
			Name:     "System Check",
			Status:   "fail",
			Message:  fmt.Sprintf("Ошибка проверки системы: %v", err),
			Critical: true,
		})
		return results
	}

	// Преобразуем system.CheckResult в cli.CheckResult
	for _, sysResult := range sysResults {
		status := "pass"
		message := sysResult.Version

		if !sysResult.Installed {
			if sysResult.Requirement.Critical {
				status = "fail"
			} else {
				status = "warn"
			}

			if sysResult.Error != nil {
				message = sysResult.Error.Error()
			} else {
				message = "Не установлено"
			}
		}

		details := ""
		if !sysResult.Installed && sysResult.InstallCommand != "" {
			details = fmt.Sprintf("Установка: %s", sysResult.InstallCommand)
			if doctorVerbose && sysResult.InstallGuide != "" {
				details += fmt.Sprintf("\nИнструкция: %s", sysResult.InstallGuide)
			}
		}

		results = append(results, CheckResult{
			Name:     sysResult.Requirement.Name,
			Status:   status,
			Message:  message,
			Details:  details,
			Critical: sysResult.Requirement.Critical,
		})
	}

	return results
}

// printDoctorResults выводит результаты проверки
func printDoctorResults(results []CheckResult, loggerInstance *logger.Logger) error {
	loggerInstance.Info("")
	loggerInstance.Info("📋 Diagnostic Results")
	loggerInstance.Info("═══════════════════════════════════════════════════")

	passCount := 0
	warnCount := 0
	failCount := 0
	criticalFail := false

	for _, result := range results {
		symbol := getStatusSymbol(result.Status)
		fmt.Printf("\n%s %s: %s\n", symbol, result.Name, result.Message)

		if doctorVerbose && result.Details != "" {
			fmt.Printf("  Details: %s\n", result.Details)
		}

		switch result.Status {
		case "pass":
			passCount++
		case "warn":
			warnCount++
		case "fail":
			failCount++
			if result.Critical {
				criticalFail = true
			}
		}
	}

	// Summary
	loggerInstance.Info("")
	loggerInstance.Info("═══════════════════════════════════════════════════")
	fmt.Printf("✅ Passed: %d  ⚠️  Warnings: %d  ❌ Failed: %d\n", passCount, warnCount, failCount)

	if criticalFail {
		loggerInstance.Info("")
		loggerInstance.Error("❌ Critical checks failed. Buffalo may not work correctly.")
		return fmt.Errorf("critical environment checks failed")
	}

	if warnCount > 0 {
		loggerInstance.Info("")
		loggerInstance.Warn("⚠️  Some checks have warnings. Some features may be limited.")
	}

	if failCount == 0 && warnCount == 0 {
		loggerInstance.Info("")
		loggerInstance.Info("🎉 All checks passed! Your Buffalo environment is ready.")
	}

	return nil
}

func checkBuffaloVersion() CheckResult {
	return CheckResult{
		Name:    "Buffalo Version",
		Status:  "pass",
		Message: fmt.Sprintf("v%s", version.Version),
		Details: fmt.Sprintf("Commit: %s, Go: %s, Platform: %s", version.GitCommit, version.GoVersion, version.Platform),
	}
}

func checkOS() CheckResult {
	return CheckResult{
		Name:    "Operating System",
		Status:  "pass",
		Message: fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		Details: fmt.Sprintf("NumCPU: %d, Go version: %s", runtime.NumCPU(), runtime.Version()),
	}
}

func checkProtoc() CheckResult {
	path, err := exec.LookPath("protoc")
	if err != nil {
		return CheckResult{
			Name:     "protoc Compiler",
			Status:   "fail",
			Message:  "Not found in PATH",
			Details:  "Install protoc from https://github.com/protocolbuffers/protobuf/releases",
			Critical: true,
		}
	}

	cmd := exec.Command("protoc", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return CheckResult{
			Name:     "protoc Compiler",
			Status:   "fail",
			Message:  "Found but unable to get version",
			Details:  string(output),
			Critical: true,
		}
	}

	versionStr := strings.TrimSpace(string(output))
	return CheckResult{
		Name:    "protoc Compiler",
		Status:  "pass",
		Message: versionStr,
		Details: fmt.Sprintf("Path: %s", path),
	}
}

func checkGo() []CheckResult {
	results := []CheckResult{}

	// Check Go installation
	goPath, err := exec.LookPath("go")
	if err != nil {
		results = append(results, CheckResult{
			Name:    "Go Language",
			Status:  "warn",
			Message: "Not found in PATH",
			Details: "Install Go from https://golang.org/dl/ to enable Go code generation",
		})
		return results
	}

	cmd := exec.Command("go", "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		results = append(results, CheckResult{
			Name:    "Go Language",
			Status:  "warn",
			Message: "Found but unable to get version",
			Details: string(output),
		})
	} else {
		versionStr := strings.TrimSpace(string(output))
		results = append(results, CheckResult{
			Name:    "Go Language",
			Status:  "pass",
			Message: versionStr,
			Details: fmt.Sprintf("Path: %s", goPath),
		})
	}

	// Check protoc-gen-go
	if path, err := exec.LookPath("protoc-gen-go"); err == nil {
		results = append(results, CheckResult{
			Name:    "protoc-gen-go",
			Status:  "pass",
			Message: "Found",
			Details: fmt.Sprintf("Path: %s", path),
		})
	} else {
		results = append(results, CheckResult{
			Name:    "protoc-gen-go",
			Status:  "warn",
			Message: "Not found",
			Details: "Install: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest",
		})
	}

	// Check protoc-gen-go-grpc
	if path, err := exec.LookPath("protoc-gen-go-grpc"); err == nil {
		results = append(results, CheckResult{
			Name:    "protoc-gen-go-grpc",
			Status:  "pass",
			Message: "Found",
			Details: fmt.Sprintf("Path: %s", path),
		})
	} else {
		results = append(results, CheckResult{
			Name:    "protoc-gen-go-grpc",
			Status:  "warn",
			Message: "Not found",
			Details: "Install: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest",
		})
	}

	return results
}

func checkPython() []CheckResult {
	results := []CheckResult{}

	// Check Python installation
	pythonCmd := "python3"
	if runtime.GOOS == "windows" {
		pythonCmd = "python"
	}

	pyPath, err := exec.LookPath(pythonCmd)
	if err != nil {
		results = append(results, CheckResult{
			Name:    "Python Language",
			Status:  "warn",
			Message: "Not found in PATH",
			Details: "Install Python from https://www.python.org/downloads/ to enable Python code generation",
		})
		return results
	}

	cmd := exec.Command(pythonCmd, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		results = append(results, CheckResult{
			Name:    "Python Language",
			Status:  "warn",
			Message: "Found but unable to get version",
			Details: string(output),
		})
	} else {
		versionStr := strings.TrimSpace(string(output))
		results = append(results, CheckResult{
			Name:    "Python Language",
			Status:  "pass",
			Message: versionStr,
			Details: fmt.Sprintf("Path: %s", pyPath),
		})
	}

	// Check grpcio-tools
	cmd = exec.Command(pythonCmd, "-c", "import grpc_tools; print(grpc_tools.__version__)")
	output, err = cmd.CombinedOutput()
	if err != nil {
		results = append(results, CheckResult{
			Name:    "grpcio-tools",
			Status:  "warn",
			Message: "Not installed",
			Details: "Install: pip install grpcio-tools",
		})
	} else {
		versionStr := strings.TrimSpace(string(output))
		results = append(results, CheckResult{
			Name:    "grpcio-tools",
			Status:  "pass",
			Message: fmt.Sprintf("v%s", versionStr),
		})
	}

	return results
}

func checkRust() []CheckResult {
	results := []CheckResult{}

	// Check Rust installation
	rustPath, err := exec.LookPath("rustc")
	if err != nil {
		results = append(results, CheckResult{
			Name:    "Rust Language",
			Status:  "warn",
			Message: "Not found in PATH",
			Details: "Install Rust from https://rustup.rs/ to enable Rust code generation",
		})
		return results
	}

	cmd := exec.Command("rustc", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		results = append(results, CheckResult{
			Name:    "Rust Language",
			Status:  "warn",
			Message: "Found but unable to get version",
			Details: string(output),
		})
	} else {
		versionStr := strings.TrimSpace(string(output))
		results = append(results, CheckResult{
			Name:    "Rust Language",
			Status:  "pass",
			Message: versionStr,
			Details: fmt.Sprintf("Path: %s", rustPath),
		})
	}

	// Check cargo
	if path, err := exec.LookPath("cargo"); err == nil {
		results = append(results, CheckResult{
			Name:    "Cargo (Rust Package Manager)",
			Status:  "pass",
			Message: "Found",
			Details: fmt.Sprintf("Path: %s", path),
		})
	} else {
		results = append(results, CheckResult{
			Name:    "Cargo (Rust Package Manager)",
			Status:  "warn",
			Message: "Not found",
			Details: "Cargo should be installed with Rust",
		})
	}

	return results
}

func checkCpp() []CheckResult {
	results := []CheckResult{}

	// Check for C++ compiler (gcc/clang/MSVC)
	compilers := []string{"g++", "clang++", "cl"}
	found := false
	var compilerInfo string

	for _, compiler := range compilers {
		if path, err := exec.LookPath(compiler); err == nil {
			found = true
			cmd := exec.Command(compiler, "--version")
			if compiler == "cl" {
				// MSVC uses different flag
				cmd = exec.Command(compiler)
			}
			output, _ := cmd.CombinedOutput()
			compilerInfo = fmt.Sprintf("%s at %s", compiler, path)
			if len(output) > 0 {
				lines := strings.Split(string(output), "\n")
				if len(lines) > 0 {
					compilerInfo = strings.TrimSpace(lines[0])
				}
			}
			break
		}
	}

	if !found {
		results = append(results, CheckResult{
			Name:    "C++ Compiler",
			Status:  "warn",
			Message: "Not found in PATH",
			Details: "Install g++, clang++, or MSVC to enable C++ code generation",
		})
	} else {
		results = append(results, CheckResult{
			Name:    "C++ Compiler",
			Status:  "pass",
			Message: compilerInfo,
		})
	}

	return results
}

func checkConfig() CheckResult {
	// Try to find buffalo.yaml
	configPaths := []string{
		"buffalo.yaml",
		"buffalo.yml",
		".buffalo.yaml",
		".buffalo.yml",
	}

	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			// File exists, try to parse it
			absPath, _ := filepath.Abs(path)
			return CheckResult{
				Name:    "Configuration File",
				Status:  "pass",
				Message: fmt.Sprintf("Found: %s", path),
				Details: absPath,
			}
		}
	}

	return CheckResult{
		Name:    "Configuration File",
		Status:  "warn",
		Message: "No buffalo.yaml found in current directory",
		Details: "Run 'buffalo init' to create a configuration file",
	}
}

func checkDependencies() CheckResult {
	// Check if .buffalo directory exists
	if _, err := os.Stat(".buffalo"); os.IsNotExist(err) {
		return CheckResult{
			Name:    "Dependencies",
			Status:  "skip",
			Message: "No dependencies installed",
			Details: "Run 'buffalo install' if your project uses dependencies",
		}
	}

	// Count installed dependencies
	protoDir := filepath.Join(".buffalo", "protos")
	if _, err := os.Stat(protoDir); os.IsNotExist(err) {
		return CheckResult{
			Name:    "Dependencies",
			Status:  "warn",
			Message: ".buffalo directory exists but no proto files found",
			Details: "Run 'buffalo install' to install dependencies",
		}
	}

	// Count proto files
	count := 0
	_ = filepath.Walk(protoDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(path, ".proto") {
			count++
		}
		return nil
	})

	return CheckResult{
		Name:    "Dependencies",
		Status:  "pass",
		Message: fmt.Sprintf("Found %d dependency proto files", count),
		Details: fmt.Sprintf("Directory: %s", protoDir),
	}
}

func getStatusSymbol(status string) string {
	switch status {
	case "pass":
		return "✅"
	case "fail":
		return "❌"
	case "warn":
		return "⚠️"
	case "skip":
		return "⏭️"
	default:
		return "❓"
	}
}
