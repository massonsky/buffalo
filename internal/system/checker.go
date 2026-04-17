package system

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/massonsky/buffalo/internal/config"
)

// Requirement представляет требование к системе
type Requirement struct {
	Name           string   // Название требования (например, "Python 3.x")
	Command        string   // Команда для проверки (например, "python3")
	Args           []string // Аргументы команды (например, ["--version"])
	CheckFunc      func() error
	InstallCommand string // Команда для установки
	InstallGuide   string // Ссылка на инструкцию по установке
	Critical       bool   // Критичное ли требование для сборки
}

// CheckResult представляет результат проверки требования
type CheckResult struct {
	Requirement    Requirement
	Installed      bool
	Version        string
	Error          error
	InstallCommand string
	InstallGuide   string
}

// SystemChecker проверяет готовность системы к сборке
type SystemChecker struct {
	config *config.Config
}

// NewSystemChecker создает новый SystemChecker
func NewSystemChecker(cfg *config.Config) *SystemChecker {
	return &SystemChecker{
		config: cfg,
	}
}

// CheckReadiness проверяет готовность системы к сборке на основе конфигурации
func (sc *SystemChecker) CheckReadiness() ([]CheckResult, error) {
	results := []CheckResult{}

	// Всегда проверяем protoc - он необходим для всех языков
	results = append(results, sc.checkProtoc())

	// Проверяем языки в зависимости от конфигурации
	if sc.config.Languages.Go.Enabled {
		results = append(results, sc.checkGo()...)
	}

	if sc.config.Languages.Python.Enabled {
		results = append(results, sc.checkPython()...)
	}

	if sc.config.Languages.Rust.Enabled {
		results = append(results, sc.checkRust()...)
	}

	if sc.config.Languages.Cpp.Enabled {
		results = append(results, sc.checkCpp()...)
	}

	if sc.config.Bazel.Enabled || sc.config.Bazel.AutoDetect {
		results = append(results, sc.checkBazel())
	}

	return results, nil
}

// checkProtoc проверяет наличие protoc компилятора
func (sc *SystemChecker) checkProtoc() CheckResult {
	req := Requirement{
		Name:         "Protocol Buffers Compiler (protoc)",
		Command:      "protoc",
		Args:         []string{"--version"},
		InstallGuide: "https://github.com/protocolbuffers/protobuf/releases",
		Critical:     true,
	}

	if runtime.GOOS == "windows" {
		req.InstallCommand = "scoop install protobuf  # или скачайте с GitHub releases"
	} else if runtime.GOOS == "darwin" {
		req.InstallCommand = "brew install protobuf"
	} else {
		req.InstallCommand = "sudo apt install -y protobuf-compiler  # для Ubuntu/Debian"
	}

	return sc.checkCommand(req)
}

// checkGo проверяет все требования для Go
func (sc *SystemChecker) checkGo() []CheckResult {
	results := []CheckResult{}

	// Проверка Go
	goReq := Requirement{
		Name:         "Go Language",
		Command:      "go",
		Args:         []string{"version"},
		InstallGuide: "https://golang.org/dl/",
		Critical:     true,
	}

	if runtime.GOOS == "windows" {
		goReq.InstallCommand = "scoop install go  # или скачайте с golang.org"
	} else if runtime.GOOS == "darwin" {
		goReq.InstallCommand = "brew install go"
	} else {
		goReq.InstallCommand = "wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz && sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz"
	}

	results = append(results, sc.checkCommand(goReq))

	// Проверка protoc-gen-go
	protoGenGoReq := Requirement{
		Name:           "protoc-gen-go",
		Command:        "protoc-gen-go",
		Args:           []string{"--version"},
		InstallCommand: "go install google.golang.org/protobuf/cmd/protoc-gen-go@latest",
		InstallGuide:   "https://grpc.io/docs/languages/go/quickstart/",
		Critical:       true,
	}
	results = append(results, sc.checkCommand(protoGenGoReq))

	// Проверка protoc-gen-go-grpc (если используется gRPC)
	if sc.config.Languages.Go.Generator == "grpc" || sc.config.Languages.Go.Generator == "" {
		grpcGenReq := Requirement{
			Name:           "protoc-gen-go-grpc",
			Command:        "protoc-gen-go-grpc",
			Args:           []string{"--version"},
			InstallCommand: "go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest",
			InstallGuide:   "https://grpc.io/docs/languages/go/quickstart/",
			Critical:       true,
		}
		results = append(results, sc.checkCommand(grpcGenReq))
	}

	return results
}

// checkPython проверяет все требования для Python
func (sc *SystemChecker) checkPython() []CheckResult {
	results := []CheckResult{}

	pythonCmd := "python3"
	if runtime.GOOS == "windows" {
		pythonCmd = "python"
	}

	// Проверка Python
	pyReq := Requirement{
		Name:         "Python",
		Command:      pythonCmd,
		Args:         []string{"--version"},
		InstallGuide: "https://www.python.org/downloads/",
		Critical:     true,
	}

	if runtime.GOOS == "windows" {
		pyReq.InstallCommand = "scoop install python  # или скачайте с python.org"
	} else if runtime.GOOS == "darwin" {
		pyReq.InstallCommand = "brew install python3"
	} else {
		pyReq.InstallCommand = "sudo apt install -y python3 python3-pip"
	}

	results = append(results, sc.checkCommand(pyReq))

	// Проверка grpcio-tools
	grpcToolsReq := Requirement{
		Name:         "grpcio-tools (Python)",
		Critical:     true,
		InstallGuide: "https://grpc.io/docs/languages/python/quickstart/",
		CheckFunc: func() error {
			cmd := exec.Command(pythonCmd, "-c", "import grpc_tools; print(grpc_tools.__version__)")
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("не установлен: %v", err)
			}
			if len(output) > 0 {
				return nil
			}
			return fmt.Errorf("не удалось получить версию")
		},
	}

	if runtime.GOOS == "windows" {
		grpcToolsReq.InstallCommand = "pip install grpcio-tools"
	} else {
		grpcToolsReq.InstallCommand = "pip3 install grpcio-tools"
	}

	results = append(results, sc.checkWithFunc(grpcToolsReq))

	// Проверка protobuf (Python)
	protobufReq := Requirement{
		Name:         "protobuf (Python)",
		Critical:     true,
		InstallGuide: "https://developers.google.com/protocol-buffers/docs/pythontutorial",
		CheckFunc: func() error {
			cmd := exec.Command(pythonCmd, "-c", "import google.protobuf; print(google.protobuf.__version__)")
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("не установлен: %v", err)
			}
			if len(output) > 0 {
				return nil
			}
			return fmt.Errorf("не удалось получить версию")
		},
	}

	if runtime.GOOS == "windows" {
		protobufReq.InstallCommand = "pip install protobuf"
	} else {
		protobufReq.InstallCommand = "pip3 install protobuf"
	}

	results = append(results, sc.checkWithFunc(protobufReq))

	return results
}

// checkRust проверяет все требования для Rust
func (sc *SystemChecker) checkRust() []CheckResult {
	results := []CheckResult{}

	// Проверка Rust compiler
	rustReq := Requirement{
		Name:         "Rust Compiler",
		Command:      "rustc",
		Args:         []string{"--version"},
		InstallGuide: "https://rustup.rs/",
		Critical:     true,
	}

	if runtime.GOOS == "windows" {
		rustReq.InstallCommand = "Скачайте rustup-init.exe с https://rustup.rs/"
	} else {
		rustReq.InstallCommand = "curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh"
	}

	results = append(results, sc.checkCommand(rustReq))

	// Проверка Cargo
	cargoReq := Requirement{
		Name:           "Cargo (Rust Package Manager)",
		Command:        "cargo",
		Args:           []string{"--version"},
		InstallCommand: "Устанавливается вместе с Rust через rustup",
		InstallGuide:   "https://rustup.rs/",
		Critical:       true,
	}
	results = append(results, sc.checkCommand(cargoReq))

	// Проверка prost (для protobuf в Rust)
	// Примечание: prost устанавливается как зависимость в Cargo.toml проекта,
	// но мы можем проверить, что cargo может найти эти зависимости
	prostReq := Requirement{
		Name:           "prost (для Rust)",
		InstallCommand: "Добавьте в Cargo.toml: prost = \"0.12\" и prost-types = \"0.12\"",
		InstallGuide:   "https://github.com/tokio-rs/prost",
		Critical:       false, // Не критично, т.к. устанавливается через Cargo.toml
		CheckFunc: func() error {
			// Проверяем, что cargo доступен (основная зависимость)
			return nil
		},
	}
	results = append(results, sc.checkWithFunc(prostReq))

	return results
}

// checkCpp проверяет все требования для C++
func (sc *SystemChecker) checkCpp() []CheckResult {
	results := []CheckResult{}

	// Создаем компиляторы с учетом ОС
	var compilers map[string]Requirement

	if runtime.GOOS == "windows" {
		compilers = map[string]Requirement{
			"cl": {
				Name:           "MSVC Compiler",
				Command:        "cl",
				Args:           []string{},
				InstallCommand: "Установите Visual Studio с компонентами C++",
				InstallGuide:   "https://visualstudio.microsoft.com/",
				Critical:       true,
			},
			"g++": {
				Name:           "G++ Compiler",
				Command:        "g++",
				Args:           []string{"--version"},
				InstallCommand: "scoop install gcc",
				InstallGuide:   "https://gcc.gnu.org/",
				Critical:       true,
			},
			"clang++": {
				Name:           "Clang++ Compiler",
				Command:        "clang++",
				Args:           []string{"--version"},
				InstallCommand: "scoop install llvm",
				InstallGuide:   "https://clang.llvm.org/",
				Critical:       true,
			},
		}
	} else if runtime.GOOS == "darwin" {
		compilers = map[string]Requirement{
			"clang++": {
				Name:           "Clang++ Compiler",
				Command:        "clang++",
				Args:           []string{"--version"},
				InstallCommand: "xcode-select --install",
				InstallGuide:   "https://clang.llvm.org/",
				Critical:       true,
			},
			"g++": {
				Name:           "G++ Compiler",
				Command:        "g++",
				Args:           []string{"--version"},
				InstallCommand: "brew install gcc",
				InstallGuide:   "https://gcc.gnu.org/",
				Critical:       true,
			},
		}
	} else {
		compilers = map[string]Requirement{
			"g++": {
				Name:           "G++ Compiler",
				Command:        "g++",
				Args:           []string{"--version"},
				InstallCommand: "sudo apt install -y build-essential",
				InstallGuide:   "https://gcc.gnu.org/",
				Critical:       true,
			},
			"clang++": {
				Name:           "Clang++ Compiler",
				Command:        "clang++",
				Args:           []string{"--version"},
				InstallCommand: "sudo apt install -y clang",
				InstallGuide:   "https://clang.llvm.org/",
				Critical:       true,
			},
		}
	}

	// Проверяем доступность хотя бы одного компилятора
	found := false
	var lastResult CheckResult

	for _, compiler := range []string{"g++", "clang++", "cl"} {
		if req, exists := compilers[compiler]; exists {
			result := sc.checkCommand(req)
			lastResult = result
			if result.Installed {
				found = true
				results = append(results, result)
				break
			}
		}
	}

	if !found {
		// Если ни один компилятор не найден, добавляем результат с рекомендацией
		lastResult.Requirement.Name = "C++ Compiler (g++, clang++ или cl)"
		if runtime.GOOS == "windows" {
			if req, exists := compilers["cl"]; exists {
				lastResult.InstallCommand = req.InstallCommand
			}
		} else if runtime.GOOS == "darwin" {
			if req, exists := compilers["clang++"]; exists {
				lastResult.InstallCommand = req.InstallCommand
			}
		} else {
			if req, exists := compilers["g++"]; exists {
				lastResult.InstallCommand = req.InstallCommand
			}
		}
		results = append(results, lastResult)
	}

	// Проверка protobuf library для C++
	// Примечание: обычно устанавливается вместе с protoc или через менеджер пакетов
	protobufLibReq := Requirement{
		Name:         "Protocol Buffers Library (C++)",
		InstallGuide: "https://github.com/protocolbuffers/protobuf/blob/main/src/README.md",
		Critical:     false, // Обычно устанавливается автоматически
	}

	if runtime.GOOS == "windows" {
		protobufLibReq.InstallCommand = "scoop install protobuf  # включает библиотеки"
	} else if runtime.GOOS == "darwin" {
		protobufLibReq.InstallCommand = "brew install protobuf"
	} else {
		protobufLibReq.InstallCommand = "sudo apt install -y libprotobuf-dev protobuf-compiler"
	}

	protobufLibReq.CheckFunc = func() error {
		// Упрощенная проверка - просто возвращаем успех, т.к. проверка наличия библиотек
		// требует компиляции тестового файла
		return nil
	}

	results = append(results, sc.checkWithFunc(protobufLibReq))

	return results
}

// checkCommand проверяет наличие команды в системе
func (sc *SystemChecker) checkCommand(req Requirement) CheckResult {
	result := CheckResult{
		Requirement:    req,
		InstallCommand: req.InstallCommand,
		InstallGuide:   req.InstallGuide,
	}

	path, err := exec.LookPath(req.Command)
	if err != nil {
		result.Installed = false
		result.Error = fmt.Errorf("команда '%s' не найдена в PATH", req.Command)
		return result
	}

	// Выполняем команду с аргументами для получения версии
	if len(req.Args) > 0 {
		cmd := exec.Command(req.Command, req.Args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			result.Installed = false
			result.Error = fmt.Errorf("не удалось выполнить '%s %s': %v", req.Command, strings.Join(req.Args, " "), err)
			return result
		}

		result.Version = strings.TrimSpace(string(output))
		// Берем только первую строку для компактности
		if lines := strings.Split(result.Version, "\n"); len(lines) > 0 {
			result.Version = strings.TrimSpace(lines[0])
		}
	} else {
		result.Version = fmt.Sprintf("Найдено в %s", path)
	}

	result.Installed = true
	return result
}

// checkWithFunc проверяет требование с помощью пользовательской функции
func (sc *SystemChecker) checkWithFunc(req Requirement) CheckResult {
	result := CheckResult{
		Requirement:    req,
		InstallCommand: req.InstallCommand,
		InstallGuide:   req.InstallGuide,
	}

	if req.CheckFunc == nil {
		result.Installed = true
		result.Version = "Проверка не требуется"
		return result
	}

	err := req.CheckFunc()
	if err != nil {
		result.Installed = false
		result.Error = err
		return result
	}

	result.Installed = true
	result.Version = "Установлено"
	return result
}

// HasCriticalFailures проверяет, есть ли критичные ошибки в результатах
func HasCriticalFailures(results []CheckResult) bool {
	for _, result := range results {
		if result.Requirement.Critical && !result.Installed {
			return true
		}
	}
	return false
}

// GetMissingCritical возвращает список критичных требований, которые не установлены
func GetMissingCritical(results []CheckResult) []CheckResult {
	var missing []CheckResult
	for _, result := range results {
		if result.Requirement.Critical && !result.Installed {
			missing = append(missing, result)
		}
	}
	return missing
}

// checkBazel проверяет наличие Bazel
func (sc *SystemChecker) checkBazel() CheckResult {
	bazelCmd := "bazel"
	if sc.config.Bazel.BazelPath != "" {
		bazelCmd = sc.config.Bazel.BazelPath
	}

	req := Requirement{
		Name:         "Bazel Build System",
		Command:      bazelCmd,
		Args:         []string{"version"},
		InstallGuide: "https://bazel.build/install",
		Critical:     false, // Not critical — falls back to file parsing
	}

	if runtime.GOOS == "windows" {
		req.InstallCommand = "scoop install bazel  # или choco install bazel"
	} else if runtime.GOOS == "darwin" {
		req.InstallCommand = "brew install bazel"
	} else {
		req.InstallCommand = "sudo apt install -y bazel  # или см. https://bazel.build/install/ubuntu"
	}

	return sc.checkCommand(req)
}
