package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/EasterCompany/dex-cli/config"
)

var pythonRequirements = []string{
	"black",
	"flake8",
	"pytest",
}

// Python manages the Python virtual environment for Dexter.
func Python(args []string) error {
	venvPath, err := config.ExpandPath("~/Dexter/python")
	if err != nil {
		return err
	}
	pythonPath := filepath.Join(venvPath, "bin", "python")

	// If no arguments, launch interactive python console
	if len(args) == 0 {
		fmt.Println("Launching Python interactive console...")
		cmd := exec.Command(pythonPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	subcommand := args[0]

	switch subcommand {
	case "init":
		return pythonInit()
	case "remove":
		return pythonRemove()
	case "upgrade":
		return pythonUpgrade()
	case "version", "help":
		// Pass version/help directly to python executable
		cmd := exec.Command(pythonPath, args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	default:
		// Pass all arguments directly to python executable
		cmd := exec.Command(pythonPath, args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
}

func pythonInit() error {
	venvPath, err := config.ExpandPath("~/Dexter/python")
	if err != nil {
		return err
	}

	if _, err := os.Stat(venvPath); err == nil {
		fmt.Println("Python virtual environment already exists.")
		return nil
	}

	fmt.Println("Creating Python virtual environment...")
	cmd := exec.Command("python3", "-m", "venv", venvPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create virtual environment: %w", err)
	}

	fmt.Println("Installing Python requirements...")
	pipPath := filepath.Join(venvPath, "bin", "pip")
	for _, req := range pythonRequirements {
		cmd := exec.Command(pipPath, "install", req)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install %s: %w", req, err)
		}
	}

	fmt.Println("Python virtual environment initialized successfully.")
	return nil
}

func pythonRemove() error {
	venvPath, err := config.ExpandPath("~/Dexter/python")
	if err != nil {
		return err
	}

	if _, err := os.Stat(venvPath); os.IsNotExist(err) {
		fmt.Println("Python virtual environment not found.")
		return nil
	}

	fmt.Println("Removing Python virtual environment...")
	if err := os.RemoveAll(venvPath); err != nil {
		return fmt.Errorf("failed to remove virtual environment: %w", err)
	}

	fmt.Println("Python virtual environment removed successfully.")
	return nil
}

func pythonUpgrade() error {
	venvPath, err := config.ExpandPath("~/Dexter/python")
	if err != nil {
		return err
	}

	if _, err := os.Stat(venvPath); os.IsNotExist(err) {
		return fmt.Errorf("python virtual environment not found. Run 'dex python init' first")
	}

	fmt.Println("Upgrading Python requirements...")
	pipPath := filepath.Join(venvPath, "bin", "pip")
	args := append([]string{"install", "--upgrade"}, pythonRequirements...)
	cmd := exec.Command(pipPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to upgrade requirements: %w", err)
	}

	fmt.Println("Python requirements upgraded successfully.")
	return nil
}

func EnsurePythonVenv(currentVersion string) error {
	versionFilePath, err := config.ExpandPath("~/Dexter/.dex-cli-version")
	if err != nil {
		return err
	}

	venvPath, err := config.ExpandPath("~/Dexter/python")
	if err != nil {
		return err
	}

	backupVenvPath, err := config.ExpandPath("~/Dexter/.python.backup")
	if err != nil {
		return err
	}

	data, err := os.ReadFile(versionFilePath)
	if err != nil || string(data) != currentVersion {
		fmt.Println("Python environment is outdated or missing. Re-initializing...")

		// Backup existing venv
		if _, err := os.Stat(venvPath); err == nil {
			fmt.Println("Backing up existing Python environment...")
			if err := os.RemoveAll(backupVenvPath); err != nil {
				return fmt.Errorf("failed to remove old backup: %w", err)
			}
			if err := os.Rename(venvPath, backupVenvPath); err != nil {
				return fmt.Errorf("failed to backup python environment: %w", err)
			}
		}

		// Initialize new venv
		if err := pythonInit(); err != nil {
			return err
		}

		// Upgrade packages
		if err := pythonUpgrade(); err != nil {
			return err
		}

		// Write new version
		if err := os.WriteFile(versionFilePath, []byte(currentVersion), 0644); err != nil {
			return fmt.Errorf("failed to write version file: %w", err)
		}
		fmt.Println("Python environment is up to date.")
	}

	return nil
}
