package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// getPythonPaths returns the absolute paths for the dexter-managed python environment for a specific version.
func getPythonPaths(version string) (envDir, pythonExecutable, pipExecutable, poetryExecutable string, err error) {
	dexterPath, err := config.GetDexterPath()
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to get dexter path: %w", err)
	}
	envDir = filepath.Join(dexterPath, "python"+version)
	pythonExecutable = filepath.Join(envDir, "bin", "python")
	pipExecutable = filepath.Join(envDir, "bin", "pip")
	poetryExecutable = filepath.Join(envDir, "bin", "poetry")
	return
}

// bootstrapSpecificVenv sets up a specific python version environment.
func bootstrapSpecificVenv(version string, silent bool) error {
	envDir, pythonExecutable, pipExecutable, _, err := getPythonPaths(version)
	if err != nil {
		return err
	}

	// 1. Check if the environment already exists.
	if _, err := os.Stat(pythonExecutable); err == nil {
		if !silent {
			ui.PrintInfo(fmt.Sprintf("Python %s environment already configured.", version))
		}
		return nil
	}

	if !silent {
		ui.PrintInfo(fmt.Sprintf("Creating virtual environment for Python %s at %s...", version, envDir))
	}

	// 2. Ensure system python version is available
	sysPython := "python" + version
	if _, err := exec.LookPath(sysPython); err != nil {
		return fmt.Errorf("system %s not found: %w", sysPython, err)
	}

	// 3. Create the virtual environment.
	if err := os.MkdirAll(envDir, 0755); err != nil {
		return fmt.Errorf("failed to create python directory: %w", err)
	}
	cmd := exec.Command(sysPython, "-m", "venv", envDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create virtual environment for %s: %w", version, err)
	}

	if !silent {
		ui.PrintSuccess(fmt.Sprintf("Virtual environment for %s created.", version))
		ui.PrintInfo(fmt.Sprintf("Installing poetry in %s environment...", version))
	}

	// 4. Install Poetry into the virtual environment.
	cmd = exec.Command(pipExecutable, "install", "poetry")
	// cmd.Stdout = os.Stdout // Keep silent unless verbose? or maybe verify logs
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install poetry in %s environment: %w", version, err)
	}

	if !silent {
		ui.PrintSuccess(fmt.Sprintf("Poetry installed in %s environment.", version))
	}

	return nil
}

// bootstrapPythonEnvironment sets up the initial python environments managed by dex-cli.
func bootstrapPythonEnvironment(silent bool) error {
	ui.PrintHeader("Setting up Dexter's Python Environments")

	if err := bootstrapSpecificVenv("3.13", silent); err != nil {
		return err
	}
	if err := bootstrapSpecificVenv("3.10", silent); err != nil {
		return err
	}
	return nil
}

// CheckPythonVersion checks if the dexter-managed python is the correct version.
func CheckPythonVersion(version string) error {
	_, pythonExecutable, _, _, err := getPythonPaths(version)
	if err != nil {
		return err
	}

	cmd := exec.Command(pythonExecutable, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("dexter-managed python %s not found or not executable: %w", version, err)
	}

	outStr := strings.TrimSpace(string(output))
	if !strings.Contains(outStr, version) {
		return fmt.Errorf("dex requires Python %s, but found %s", version, outStr)
	}

	return nil
}

// CheckPoetryInstalled checks if poetry is installed in the dexter-managed environment.
func CheckPoetryInstalled(version string) error {
	_, _, _, poetryExecutable, err := getPythonPaths(version)
	if err != nil {
		return err
	}
	cmd := exec.Command(poetryExecutable, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("poetry is not installed or not in PATH for %s: %w", version, err)
	}
	return nil
}

// EnsurePythonVenv ensures the entire dexter-managed python environment is correctly set up.
func EnsurePythonVenv(silent bool) error {
	if err := bootstrapPythonEnvironment(silent); err != nil {
		return err
	}

	if !silent {
		ui.PrintHeader("Verifying Python Environments")
	}

	// Verify 3.13
	if !silent {
		ui.PrintInfo("Checking Python 3.13...")
	}
	if err := CheckPythonVersion("3.13"); err != nil {
		return err
	}
	if err := CheckPoetryInstalled("3.13"); err != nil {
		return err
	}
	if !silent {
		ui.PrintSuccess("Python 3.13 found.")
	}

	// Verify 3.10
	if !silent {
		ui.PrintInfo("Checking Python 3.10...")
	}
	if err := CheckPythonVersion("3.10"); err != nil {
		return err
	}
	if err := CheckPoetryInstalled("3.10"); err != nil {
		return err
	}
	if !silent {
		ui.PrintSuccess("Python 3.10 found.")
	}

	return nil
}

// Python runs a command using the dexter-managed python executable (Defaults to 3.13).
func Python(args []string) error {
	// Default to 3.13 for 'dex python' command
	_, pythonExecutable, _, _, err := getPythonPaths("3.13")
	if err != nil {
		return err
	}

	cmd := exec.Command(pythonExecutable, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// CheckBunInstalled checks if Bun is installed.
func CheckBunInstalled() error {
	cmd := exec.Command("bun", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bun is not installed or not in PATH")
	}
	return nil
}
