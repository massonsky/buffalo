# Buffalo Installation Script for Windows (PowerShell)
# Usage: 
#   Invoke-WebRequest -Uri "https://raw.githubusercontent.com/massonsky/buffalo/main/install.ps1" -OutFile install.ps1; .\install.ps1
#   Or: .\install.ps1

param(
    [string]$Version = "latest",
    [string]$InstallDir = "$env:ProgramFiles\buffalo",
    [switch]$AddToPath = $true
)

$ErrorActionPreference = "Stop"

# Configuration
$BinaryName = "buffalo"
$Repo = "massonsky/buffalo"

# Colors
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

function Test-Administrator {
    $currentUser = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
    return $currentUser.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Get-Platform {
    Write-Info "Detecting platform..."
    $arch = $env:PROCESSOR_ARCHITECTURE
    
    switch ($arch) {
        "AMD64" { $script:Arch = "amd64" }
        "ARM64" { $script:Arch = "arm64" }
        default { Write-Error "Unsupported architecture: $arch" }
    }
    
    $script:Platform = "windows-$Arch"
    Write-Info "Detected platform: $Platform"
}

function Test-Prerequisites {
    Write-Info "Checking prerequisites..."
    
    # Check for protoc
    try {
        $protocVersion = & protoc --version 2>&1
        Write-Info "protoc version: $protocVersion"
    } catch {
        Write-Warning "protoc is not installed. Buffalo requires protoc to work."
        Write-Info "Install protoc from: https://github.com/protocolbuffers/protobuf/releases"
    }
    
    # Check if Go is available (for building from source if needed)
    try {
        $goVersion = & go version 2>&1
        Write-Info "Go version: $goVersion"
    } catch {
        Write-Warning "Go is not installed. Required only if building from source."
    }
    
    Write-Success "Prerequisites check completed"
}

function Get-LatestVersion {
    if ($Version -eq "latest") {
        Write-Info "Fetching latest version..."
        try {
            $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
            $script:Version = $release.tag_name
            Write-Info "Latest version: $Version"
        } catch {
            Write-Warning "Could not fetch latest version, will attempt to build from source"
            $script:Version = "main"
        }
    }
}

function Get-Binary {
    Write-Info "Downloading Buffalo $Version for $Platform..."
    
    $downloadUrl = "https://github.com/$Repo/releases/download/$Version/$BinaryName-$Version-$Platform.zip"
    $tempDir = New-Item -ItemType Directory -Path "$env:TEMP\buffalo-install-$(Get-Random)" -Force
    $tempFile = Join-Path $tempDir "$BinaryName.zip"
    
    try {
        Invoke-WebRequest -Uri $downloadUrl -OutFile $tempFile -ErrorAction Stop
        Write-Success "Downloaded successfully"
        
        Expand-Archive -Path $tempFile -DestinationPath $tempDir -Force
        $script:BinaryPath = Join-Path $tempDir "$BinaryName-$Platform.exe"
        
        if (-not (Test-Path $BinaryPath)) {
            # Try without platform suffix
            $script:BinaryPath = Get-ChildItem -Path $tempDir -Filter "*.exe" | Select-Object -First 1 -ExpandProperty FullName
        }
    } catch {
        Write-Warning "Release not found, attempting to build from source..."
        Build-FromSource
    }
}

function Build-FromSource {
    Write-Info "Building Buffalo from source..."
    
    # Check for Go
    try {
        $null = & go version 2>&1
    } catch {
        Write-Error "Go is required to build from source. Install from: https://golang.org/dl/"
    }
    
    $tempDir = New-Item -ItemType Directory -Path "$env:TEMP\buffalo-build-$(Get-Random)" -Force
    Set-Location $tempDir
    
    Write-Info "Cloning repository..."
    if ($Version -eq "main") {
        & git clone --depth 1 "https://github.com/$Repo.git" . 2>&1 | Out-Null
    } else {
        $sourceUrl = "https://github.com/$Repo/archive/refs/tags/$Version.zip"
        Invoke-WebRequest -Uri $sourceUrl -OutFile "source.zip"
        Expand-Archive -Path "source.zip" -DestinationPath . -Force
        Set-Location (Get-ChildItem -Directory | Select-Object -First 1).FullName
    }
    
    Write-Info "Building binary..."
    & go build -ldflags "-s -w" -o "$BinaryName.exe" ./cmd/buffalo
    
    if (-not (Test-Path "$BinaryName.exe")) {
        Write-Error "Build failed"
    }
    
    $script:BinaryPath = Join-Path (Get-Location) "$BinaryName.exe"
    Write-Success "Built successfully"
}

function Install-Binary {
    Write-Info "Installing to $InstallDir..."
    
    # Create install directory
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }
    
    # Copy binary
    $targetPath = Join-Path $InstallDir "$BinaryName.exe"
    Copy-Item -Path $BinaryPath -Destination $targetPath -Force
    
    Write-Success "Installed to $targetPath"
    
    # Add to PATH if requested
    if ($AddToPath) {
        Add-ToPath
    }
}

function Add-ToPath {
    Write-Info "Adding to PATH..."
    
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    
    if ($currentPath -notlike "*$InstallDir*") {
        $newPath = "$currentPath;$InstallDir"
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        $env:Path = "$env:Path;$InstallDir"
        Write-Success "Added $InstallDir to PATH"
        Write-Warning "You may need to restart your terminal for PATH changes to take effect"
    } else {
        Write-Info "$InstallDir is already in PATH"
    }
}

function Test-Installation {
    Write-Info "Verifying installation..."
    
    # Refresh environment
    $env:Path = [Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [Environment]::GetEnvironmentVariable("Path", "User")
    
    try {
        $version = & "$InstallDir\$BinaryName.exe" --version 2>&1
        Write-Success "Installation verified: $version"
        Write-Info "Run '$BinaryName --help' to get started"
    } catch {
        Write-Error "Installation verification failed. Binary not working correctly"
    }
}

function Show-Instructions {
    Write-Host ""
    Write-Success "Buffalo installed successfully!"
    Write-Host ""
    Write-Host "Quick Start:" -ForegroundColor Blue
    Write-Host "  1. Create a buffalo.yaml configuration file"
    Write-Host "  2. Add your .proto files"
    Write-Host "  3. Run: buffalo build"
    Write-Host ""
    Write-Host "Examples:" -ForegroundColor Blue
    Write-Host "  buffalo init              # Initialize new project"
    Write-Host "  buffalo build             # Build proto files"
    Write-Host "  buffalo build --lang go   # Build only for Go"
    Write-Host "  buffalo --help            # Show all commands"
    Write-Host ""
    Write-Host "Documentation:" -ForegroundColor Blue
    Write-Host "  https://github.com/$Repo"
    Write-Host ""
    
    if (-not $AddToPath) {
        Write-Warning "Buffalo was not added to PATH automatically."
        Write-Host "Add manually: " -NoNewline
        Write-Host "$InstallDir" -ForegroundColor Yellow
    }
}

# Main installation flow
function Main {
    Write-Host ""
    Write-Host "╔════════════════════════════════════════╗" -ForegroundColor Blue
    Write-Host "║   Buffalo Installation Script         ║" -ForegroundColor Blue
    Write-Host "╚════════════════════════════════════════╝" -ForegroundColor Blue
    Write-Host ""
    
    # Check admin rights for system-wide installation
    if ($InstallDir -like "$env:ProgramFiles*" -and -not (Test-Administrator)) {
        Write-Warning "Installing to $InstallDir requires Administrator privileges"
        Write-Info "Changing install location to user directory..."
        $script:InstallDir = "$env:LOCALAPPDATA\buffalo"
    }
    
    Get-Platform
    Test-Prerequisites
    Get-LatestVersion
    Get-Binary
    Install-Binary
    Test-Installation
    Show-Instructions
}

# Run installation
Main
