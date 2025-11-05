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

// getPythonPaths returns the absolute paths for the dexter-managed python environment.
func getPythonPaths() (envDir, pythonExecutable, pipExecutable, poetryExecutable string, err error) {
	dexterPath, err := config.GetDexterPath()
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to get dexter path: %w", err)
	}
	envDir = filepath.Join(dexterPath, "python")
	pythonExecutable = filepath.Join(envDir, "bin", "python")
	pipExecutable = filepath.Join(envDir, "bin", "pip")
	poetryExecutable = filepath.Join(envDir, "bin", "poetry")
	return
}

// bootstrapPythonEnvironment sets up the initial python environment managed by dex-cli.
func bootstrapPythonEnvironment(silent bool) error {
	envDir, pythonExecutable, pipExecutable, _, err := getPythonPaths()
	if err != nil {
		return err
	}

	// 1. Check if the environment already exists.
	if _, err := os.Stat(pythonExecutable); err == nil {
		if !silent {
			ui.PrintInfo("Python environment already configured.")
		}
		return nil
	}

	ui.PrintHeader("Setting up Dexter's Python Environment")

	// 2. Install system python if not available.
	if _, err := exec.LookPath("python3"); err != nil {
		ui.PrintInfo("System python3 not found. Installing with 'yay'...")
		cmd := exec.Command("yay", "-S", "--noconfirm", "python")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install system python: %w", err)
		}
		ui.PrintSuccess("System python installed.")
	}

	// 3. Create the virtual environment.
	ui.PrintInfo(fmt.Sprintf("Creating virtual environment at %s...", envDir))
	if err := os.MkdirAll(envDir, 0755); err != nil {
		return fmt.Errorf("failed to create python directory: %w", err)
	}
	cmd := exec.Command("python3", "-m", "venv", envDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create virtual environment: %w", err)
	}
	ui.PrintSuccess("Virtual environment created.")

	// 4. Install Poetry into the virtual environment.
	ui.PrintInfo("Installing poetry...")
	cmd = exec.Command(pipExecutable, "install", "poetry")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install poetry: %w", err)
	}
	ui.PrintSuccess("Poetry installed successfully.")

	return nil
}

// CheckPythonVersion checks if the dexter-managed python is the correct version.
func CheckPythonVersion() error {
	_, pythonExecutable, _, _, err := getPythonPaths()
	if err != nil {
		return err
	}

	cmd := exec.Command(pythonExecutable, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("dexter-managed python not found or not executable: %w", err)
	}

	version := strings.TrimSpace(string(output))
	if !strings.Contains(version, "3.13") {
		return fmt.Errorf("dex requires Python 3.13, but found %s", version)
	}

	return nil
}

// CheckPoetryInstalled checks if poetry is installed in the dexter-managed environment.
func CheckPoetryInstalled() error {
	_, _, _, poetryExecutable, err := getPythonPaths()
	if err != nil {
		return err
	}
	cmd := exec.Command(poetryExecutable, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("poetry is not installed or not in PATH: %w", err)
	}
	return nil
}

// EnsurePythonVenv ensures the entire dexter-managed python environment is correctly set up.
func EnsurePythonVenv(silent bool) error {
	if err := bootstrapPythonEnvironment(silent); err != nil {
		return err
	}

	if !silent {
		ui.PrintHeader("Verifying Python Environment")
	}

	if !silent {
		ui.PrintInfo("Checking Python 3.13...")
	}
	if err := CheckPythonVersion(); err != nil {
		return err
	}
	if !silent {
		ui.PrintSuccess("Python 3.13 found.")
	}

	if !silent {
		ui.PrintInfo("Checking Poetry installation...")
	}
	if err := CheckPoetryInstalled(); err != nil {
		return err
	}
	if !silent {
		ui.PrintSuccess("Poetry installed.")
	}

	return nil
}

// Python runs a command using the dexter-managed python executable.
func Python(args []string) error {
	_, pythonExecutable, _, _, err := getPythonPaths()
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
