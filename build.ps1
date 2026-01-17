# Buffalo Build Script for Windows PowerShell
# This script provides convenient building, testing, and installation for Windows

param(
    [Parameter(Position = 0)]
    [ValidateSet("help", "build", "build-all", "test", "test-coverage", "install", "uninstall", 
                 "clean", "clean-all", "fmt", "vet", "lint", "check", "version", "example")]
    [string]$Target = "help",
    
    [string]$InstallPrefix = "$env:ProgramFiles\buffalo",
    [switch]$ShowDetails = $false
)

$ErrorActionPreference = "Stop"

# Configuration
$BinaryName = "buffalo"
$Module = "github.com/massonsky/buffalo"
$Version = "v0.5.0-dev"
$BinDir = "bin"
$BuildDir = "build"

# Functions
function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "[SUCCESS] $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "[WARNING] $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
    exit 1
}

function Ensure-Directory {
    param([string]$Path)
    if (-not (Test-Path $Path)) {
        New-Item -ItemType Directory -Path $Path -Force | Out-Null
    }
}

function Get-Version {
    try {
        $version = & git describe --tags --always --dirty 2>$null
        if ($LASTEXITCODE -eq 0) { return $version }
    } catch { }
    return $Version
}

function Get-BuildTime {
    return Get-Date -Format "yyyy-MM-dd_HH:mm:ss"
}

function Get-GitCommit {
    try {
        $commit = & git rev-parse --short HEAD 2>$null
        if ($LASTEXITCODE -eq 0) { return $commit }
    } catch { }
    return "unknown"
}

function Build-Binary {
    param(
        [string]$GOOS = "",
        [string]$GOARCH = "",
        [string]$OutputName = $BinaryName
    )
    
    $version = Get-Version
    $buildTime = Get-BuildTime
    $gitCommit = Get-GitCommit
    
    $ldflags = @"
    -X ${Module}/internal/version.Version=${version} `
    -X ${Module}/internal/version.BuildDate=${buildTime} `
    -X ${Module}/internal/version.GitCommit=${gitCommit} `
    -s -w
"@

    $args = @("build", "-ldflags", $ldflags)
    
    if ($OutputName -match "\.exe$") {
        $args += @("-o", $OutputName)
    } else {
        $args += @("-o", "$OutputName.exe")
    }
    
    $args += "./cmd/buffalo"
    
    $env = @{}
    if ($GOOS) { $env["GOOS"] = $GOOS }
    if ($GOARCH) { $env["GOARCH"] = $GOARCH }
    
    Write-Info "Building $OutputName..."
    
    # Set environment variables
    $oldEnv = @{}
    foreach ($key in $env.Keys) {
        $oldEnv[$key] = [Environment]::GetEnvironmentVariable($key, "Process")
        [Environment]::SetEnvironmentVariable($key, $env[$key], "Process")
    }
    
    try {
        & go $args
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Build failed"
        }
        Write-Success "Built: $OutputName"
    } finally {
        # Restore environment
        foreach ($key in $oldEnv.Keys) {
            [Environment]::SetEnvironmentVariable($key, $oldEnv[$key], "Process")
        }
    }
}

function Target-Help {
    Write-Host ""
    Write-Host "======================================================" -ForegroundColor Blue
    Write-Host "   Buffalo - Protocol Buffer Compiler                 " -ForegroundColor Blue
    Write-Host "   Windows Build Script                               " -ForegroundColor Blue
    Write-Host "======================================================" -ForegroundColor Blue
    Write-Host ""
    Write-Host "Usage: .\build.ps1 [target] [options]" -ForegroundColor Green
    Write-Host ""
    Write-Host "Targets:" -ForegroundColor Cyan
    Write-Host "  help            Show this help message"
    Write-Host "  build           Build the binary (default)"
    Write-Host "  build-all       Build for all platforms"
    Write-Host "  test            Run tests"
    Write-Host "  test-coverage   Run tests with coverage"
    Write-Host "  fmt             Format code"
    Write-Host "  vet             Run go vet"
    Write-Host "  lint            Run linter (golangci-lint)"
    Write-Host "  check           Run all checks"
    Write-Host "  install         Install to GOPATH/bin"
    Write-Host "  uninstall       Uninstall from system"
    Write-Host "  clean           Clean build artifacts"
    Write-Host "  clean-all       Clean all including caches"
    Write-Host "  version         Show version information"
    Write-Host "  example         Run example build"
    Write-Host ""
    Write-Host "Options:" -ForegroundColor Cyan
    Write-Host "  -InstallPrefix [path]   Installation directory"
    Write-Host "  -ShowDetails            Show detailed output"
    Write-Host ""
    Write-Host "Examples:" -ForegroundColor Green
    Write-Host "  .\build.ps1 build"
    Write-Host "  .\build.ps1 build-all"
    Write-Host "  .\build.ps1 test"
    Write-Host "  .\build.ps1 install -InstallPrefix C:\buffalo"
    Write-Host ""
}

function Target-Build {
    Ensure-Directory $BinDir
    Build-Binary -OutputName "$BinDir\$BinaryName"
}

function Target-BuildAll {
    Ensure-Directory $BuildDir
    
    Write-Info "Building for all platforms..."
    
    Build-Binary -GOOS "linux" -GOARCH "amd64" -OutputName "$BuildDir\$BinaryName-linux-amd64"
    Build-Binary -GOOS "linux" -GOARCH "arm64" -OutputName "$BuildDir\$BinaryName-linux-arm64"
    Build-Binary -GOOS "darwin" -GOARCH "amd64" -OutputName "$BuildDir\$BinaryName-darwin-amd64"
    Build-Binary -GOOS "darwin" -GOARCH "arm64" -OutputName "$BuildDir\$BinaryName-darwin-arm64"
    Build-Binary -GOOS "windows" -GOARCH "amd64" -OutputName "$BuildDir\$BinaryName-windows-amd64"
    
    Write-Success "All platforms built in $BuildDir\"
}

function Target-Test {
    Write-Info "Running tests..."
    & go test -v -race -coverprofile=coverage.out ./...
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Tests failed"
    }
    Write-Success "Tests passed"
}

function Target-TestCoverage {
    Write-Info "Running tests with coverage..."
    & go test -v -race -coverprofile=coverage.out ./...
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Tests failed"
    }
    
    Write-Info "Generating coverage report..."
    & go tool cover -html=coverage.out -o coverage.html
    Write-Success "Coverage report: coverage.html"
}

function Target-Fmt {
    Write-Info "Formatting code..."
    & go fmt ./...
    Write-Success "Code formatted"
}

function Target-Vet {
    Write-Info "Running go vet..."
    & go vet ./...
    Write-Success "Vet passed"
}

function Target-Lint {
    Write-Info "Running linter..."
    
    # Get GOPATH for finding golangci-lint
    $gopath = & go env GOPATH
    $golangciLintPath = Join-Path $gopath "bin\golangci-lint.exe"
    
    # Check if golangci-lint is installed
    $golangciLint = Get-Command golangci-lint -ErrorAction SilentlyContinue
    if (-not $golangciLint -and -not (Test-Path $golangciLintPath)) {
        Write-Warning "golangci-lint not found. Installing..."
        & go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    }
    
    # Use full path if command not found in PATH
    if ($golangciLint) {
        & golangci-lint run ./...
    } else {
        & $golangciLintPath run ./...
    }
    Write-Success "Lint passed"
}

function Target-Check {
    Write-Info "Running all checks..."
    Target-Fmt
    Target-Vet
    Target-Lint
    Target-Test
    Write-Success "All checks passed"
}

function Target-Install {
    Target-Build
    
    Write-Info "Installing to $InstallPrefix..."
    Ensure-Directory $InstallPrefix
    
    Copy-Item -Path "$BinDir\$BinaryName.exe" -Destination "$InstallPrefix\$BinaryName.exe" -Force
    
    Write-Success "Installed to $InstallPrefix\$BinaryName.exe"
    Write-Info "Add '$InstallPrefix' to PATH to use 'buffalo' command"
}

function Target-Uninstall {
    if (Test-Path "$InstallPrefix\$BinaryName.exe") {
        Remove-Item "$InstallPrefix\$BinaryName.exe" -Force
        Write-Success "Uninstalled"
    } else {
        Write-Warning "Not installed at $InstallPrefix"
    }
}

function Target-Clean {
    Write-Info "Cleaning build artifacts..."
    Remove-Item -Recurse -Force $BinDir -ErrorAction SilentlyContinue
    Remove-Item -Recurse -Force $BuildDir -ErrorAction SilentlyContinue
    Remove-Item -Force "coverage.out", "coverage.html" -ErrorAction SilentlyContinue
    & go clean
    Write-Success "Clean complete"
}

function Target-CleanAll {
    Target-Clean
    Write-Info "Cleaning caches..."
    & go clean -cache -testcache -modcache
    Write-Success "All clean"
}

function Target-Version {
    $version = Get-Version
    $buildTime = Get-BuildTime
    $gitCommit = Get-GitCommit
    $goVersion = & go version
    
    Write-Host ""
    Write-Host "Buffalo Protocol Buffer Compiler" -ForegroundColor Green
    Write-Host "Version:    $version" -ForegroundColor Green
    Write-Host "Build Time: $buildTime" -ForegroundColor Green
    Write-Host "Git Commit: $gitCommit" -ForegroundColor Green
    Write-Host "Go Version: $goVersion" -ForegroundColor Green
    Write-Host ""
}

function Target-Example {
    Target-Build
    Write-Info "Running example build..."
    Push-Location test-project
    & ..\$BinDir\$BinaryName.exe build --lang python,go
    Pop-Location
}

# Main execution
switch ($Target.ToLower()) {
    "help"           { Target-Help }
    "build"          { Target-Build }
    "build-all"      { Target-BuildAll }
    "test"           { Target-Test }
    "test-coverage"  { Target-TestCoverage }
    "fmt"            { Target-Fmt }
    "vet"            { Target-Vet }
    "lint"           { Target-Lint }
    "check"          { Target-Check }
    "install"        { Target-Install }
    "uninstall"      { Target-Uninstall }
    "clean"          { Target-Clean }
    "clean-all"      { Target-CleanAll }
    "version"        { Target-Version }
    "example"        { Target-Example }
    default          { Write-Error "Unknown target: $Target. Run '.\build.ps1 help' for usage." }
}
