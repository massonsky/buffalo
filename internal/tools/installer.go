// Package tools provides language tools installation for Buffalo.
package tools

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/massonsky/buffalo/pkg/logger"
)

// Installer handles tool installation.
type Installer struct {
	log      *logger.Logger
	platform string
}

// NewInstaller creates a new tool installer.
func NewInstaller(log *logger.Logger) *Installer {
	return &Installer{
		log:      log,
		platform: GetPlatform(),
	}
}

// Check checks if a tool is installed and returns its version.
func (i *Installer) Check(tool Tool) (installed bool, version string, err error) {
	// Use custom check function if available
	if tool.CheckFunc != nil {
		version, err = tool.CheckFunc()
		if err != nil {
			return false, "", err
		}
		return true, version, nil
	}

	// Use command check
	if tool.CheckCommand == "" {
		// For pip packages, check via pip
		if tool.Language == "python" && tool.CheckCommand == "" {
			return i.checkPythonPackage(tool.Name)
		}
		return false, "", fmt.Errorf("no check method for tool %s", tool.Name)
	}

	cmd := exec.Command(tool.CheckCommand, tool.CheckArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, "", err
	}

	version = strings.TrimSpace(string(output))
	// Extract version from output (usually first line or after colon)
	lines := strings.Split(version, "\n")
	if len(lines) > 0 {
		version = strings.TrimSpace(lines[0])
	}

	return true, version, nil
}

// checkPythonPackage checks if a Python package is installed.
func (i *Installer) checkPythonPackage(pkg string) (bool, string, error) {
	pythonCmd := "python3"
	if runtime.GOOS == "windows" {
		pythonCmd = "python"
	}

	// Check with pip show
	pipCmd := "pip3"
	if runtime.GOOS == "windows" {
		pipCmd = "pip"
	}

	cmd := exec.Command(pipCmd, "show", pkg)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try alternative check via import
		importCmd := exec.Command(pythonCmd, "-c", fmt.Sprintf("import %s; print('installed')", strings.ReplaceAll(pkg, "-", "_")))
		_, err := importCmd.CombinedOutput()
		if err != nil {
			return false, "", fmt.Errorf("not installed")
		}
		return true, "installed", nil
	}

	// Parse version from pip show output
	version := "installed"
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "Version:") {
			version = strings.TrimSpace(strings.TrimPrefix(line, "Version:"))
			break
		}
	}

	return true, version, nil
}

// Install installs a single tool.
func (i *Installer) Install(tool Tool, opts InstallOptions) InstallResult {
	result := InstallResult{Tool: tool}

	// Check if already installed
	if !opts.Force {
		installed, version, _ := i.Check(tool)
		if installed {
			result.AlreadyOK = true
			result.Success = true
			result.Version = version
			result.Message = fmt.Sprintf("Already installed: %s", version)
			return result
		}
	}

	// Get install command for platform
	installCmd, ok := tool.InstallMethods[i.platform]
	if !ok {
		result.Error = fmt.Errorf("no install method for platform %s", i.platform)
		result.Message = result.Error.Error()
		return result
	}

	// Dry run mode
	if opts.DryRun {
		result.Skipped = true
		result.Message = fmt.Sprintf("Would run: %s", installCmd)
		return result
	}

	// Interactive confirmation
	if opts.Interactive {
		fmt.Printf("Install %s? [y/N] ", tool.Name)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			result.Skipped = true
			result.Message = "Skipped by user"
			return result
		}
	}

	// Log what we're doing
	i.log.Info("Installing tool",
		logger.String("name", tool.Name),
		logger.String("command", installCmd))

	// Execute installation
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", installCmd)
	} else {
		cmd = exec.Command("sh", "-c", installCmd)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		result.Error = err
		result.Message = fmt.Sprintf("Installation failed: %v", err)
		return result
	}

	// Verify installation
	installed, version, _ := i.Check(tool)
	if installed {
		result.Success = true
		result.Version = version
		result.Message = fmt.Sprintf("Successfully installed: %s", version)

		if tool.PostInstall != "" {
			result.Message += fmt.Sprintf("\nNote: %s", tool.PostInstall)
		}
	} else {
		result.Error = fmt.Errorf("installation completed but verification failed")
		result.Message = "Installation completed but tool not found in PATH. You may need to restart your terminal or update PATH."
	}

	return result
}

// InstallForLanguage installs all tools for a language.
func (i *Installer) InstallForLanguage(lang string, opts InstallOptions) []InstallResult {
	var tools []Tool
	if opts.IncludeAll {
		tools = GetToolsForLanguage(lang)
	} else {
		tools = GetCriticalTools(lang)
	}

	return i.InstallTools(tools, opts)
}

// InstallTools installs a list of tools.
func (i *Installer) InstallTools(tools []Tool, opts InstallOptions) []InstallResult {
	results := make([]InstallResult, 0, len(tools))

	for _, tool := range tools {
		result := i.Install(tool, opts)
		results = append(results, result)

		if result.Error != nil && !opts.Force {
			// Stop on first error for critical tools
			if tool.Critical {
				i.log.Error("Critical tool installation failed",
					logger.String("tool", tool.Name),
					logger.Any("error", result.Error))
				break
			}
		}
	}

	return results
}

// InstallAll installs tools for all specified languages.
func (i *Installer) InstallAll(languages []string, opts InstallOptions) map[string][]InstallResult {
	results := make(map[string][]InstallResult)

	// Always install core tools first
	coreTools := GetToolsForLanguage("all")
	results["core"] = i.InstallTools(coreTools, opts)

	// Install language-specific tools
	for _, lang := range languages {
		langResults := i.InstallForLanguage(lang, opts)
		results[lang] = langResults
	}

	return results
}

// ListTools returns tools for the specified languages.
func (i *Installer) ListTools(languages []string, includeAll bool) []Tool {
	toolMap := make(map[string]Tool)

	// Always include core tools
	for _, tool := range GetToolsForLanguage("all") {
		if includeAll || tool.Critical {
			toolMap[tool.Name] = tool
		}
	}

	// Add language-specific tools
	for _, lang := range languages {
		var tools []Tool
		if includeAll {
			tools = GetToolsForLanguage(lang)
		} else {
			tools = GetCriticalTools(lang)
		}
		for _, tool := range tools {
			toolMap[tool.Name] = tool
		}
	}

	// Convert to slice
	result := make([]Tool, 0, len(toolMap))
	for _, tool := range toolMap {
		result = append(result, tool)
	}

	return result
}

// CheckAll checks all tools for specified languages.
func (i *Installer) CheckAll(languages []string) map[string][]InstallResult {
	results := make(map[string][]InstallResult)

	// Check core tools
	coreTools := GetToolsForLanguage("all")
	coreResults := make([]InstallResult, 0, len(coreTools))
	for _, tool := range coreTools {
		installed, version, err := i.Check(tool)
		result := InstallResult{
			Tool:      tool,
			Success:   installed,
			Version:   version,
			AlreadyOK: installed,
		}
		if err != nil {
			result.Error = err
			result.Message = err.Error()
		} else if installed {
			result.Message = version
		} else {
			result.Message = "Not installed"
		}
		coreResults = append(coreResults, result)
	}
	results["core"] = coreResults

	// Check language-specific tools
	for _, lang := range languages {
		tools := GetToolsForLanguage(lang)
		langResults := make([]InstallResult, 0, len(tools))
		for _, tool := range tools {
			installed, version, err := i.Check(tool)
			result := InstallResult{
				Tool:      tool,
				Success:   installed,
				Version:   version,
				AlreadyOK: installed,
			}
			if err != nil {
				result.Error = err
				result.Message = err.Error()
			} else if installed {
				result.Message = version
			} else {
				result.Message = "Not installed"
			}
			langResults = append(langResults, result)
		}
		results[lang] = langResults
	}

	return results
}
