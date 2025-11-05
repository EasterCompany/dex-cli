// cmd/pipeline.go
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

// gitUpdateService clones or force-pulls a service repository.
func GitUpdateService(def config.ServiceDefinition) error {
	sourcePath, err := config.ExpandPath(def.Source)
	if err != nil {
		return fmt.Errorf("failed to expand source path: %w", err)
	}

	if !CheckFileExists(sourcePath) {
		return fmt.Errorf("source directory not found for %s: %s", def.ShortName, sourcePath)
	}

	// Source exists, force-pull latest changes
	ui.PrintInfo(fmt.Sprintf("[%s] Pulling latest changes...", def.ShortName))

	commands := [][]string{
		{"git", "fetch", "--all"},
		{"git", "reset", "--hard", "origin/main"},
		{"git", "pull", "origin", "main"},
	}

	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = sourcePath
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("command failed: %s\nOutput: %s\nError: %w", strings.Join(args, " "), string(output), err)
		}
	}
	return nil
}

// runServicePipelineStep executes a single step (format, lint, test, build)
func RunServicePipelineStep(def config.ServiceDefinition, step string) error {
	sourcePath, err := config.ExpandPath(def.Source)
	if err != nil {
		return fmt.Errorf("failed to expand source path: %w", err)
	}

	if !CheckFileExists(filepath.Join(sourcePath, "go.mod")) {
		return fmt.Errorf("skipping %s: not a Go project (no go.mod)", step)
	}

	var cmd *exec.Cmd
	switch step {
	case "format":
		ui.PrintInfo(fmt.Sprintf("[%s] Running Formatting...", def.ShortName))
		cmd = exec.Command("go", "fmt", "./...")
	case "lint":
		ui.PrintInfo(fmt.Sprintf("[%s] Running Linting...", def.ShortName))
		cmd = exec.Command("golangci-lint", "run")
	case "test":
		ui.PrintInfo(fmt.Sprintf("[%s] Running Testing...", def.ShortName))
		cmd = exec.Command("go", "test", "./...")
	case "build":
		ui.PrintInfo(fmt.Sprintf("[%s] Running Building...", def.ShortName))
		binPath, err := config.ExpandPath(def.GetBinaryPath())
		if err != nil {
			return fmt.Errorf("could not expand binary path: %w", err)
		}
		cmd = exec.Command("go", "build", "-o", binPath)
	default:
		return fmt.Errorf("unknown pipeline step: %s", step)
	}

	cmd.Dir = sourcePath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed %s for %s:\n%s", step, def.ShortName, string(output))
	}
	return nil
}

// InstallSystemdService creates and enables the systemd service file.
func InstallSystemdService(def config.ServiceDefinition) error {
	if !def.IsManageable() {
		return nil // Skip for cli, os
	}

	ui.PrintInfo(fmt.Sprintf("[%s] Installing systemd service...", def.ShortName))

	serviceFilePath, err := config.ExpandPath(def.GetSystemdPath())
	if err != nil {
		return fmt.Errorf("could not get systemd path: %w", err)
	}

	executablePath, err := config.ExpandPath(def.GetBinaryPath())
	if err != nil {
		return fmt.Errorf("could not expand binary path: %w", err)
	}

	logPath, err := config.ExpandPath(def.GetLogPath())
	if err != nil {
		return fmt.Errorf("could not expand log path: %w", err)
	}

	// Ensure directories exist
	if err := os.MkdirAll(filepath.Dir(serviceFilePath), 0755); err != nil {
		return fmt.Errorf("could not create systemd directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return fmt.Errorf("could not create log directory: %w", err)
	}

	// Create service file content
	serviceFileContent := fmt.Sprintf(`[Unit]
Description=%s
After=network.target

[Service]
ExecStart=%s
Restart=always
RestartSec=3
StandardOutput=append:%s
StandardError=append:%s

[Install]
WantedBy=default.target
`, def.ID, executablePath, logPath, logPath)

	// Write the service file
	if err := os.WriteFile(serviceFilePath, []byte(serviceFileContent), 0644); err != nil {
		return fmt.Errorf("could not write service file: %w", err)
	}

	// Run systemctl commands
	commands := [][]string{
		{"systemctl", "--user", "daemon-reload"},
		{"systemctl", "--user", "enable", def.SystemdName},
		{"systemctl", "--user", "restart", def.SystemdName},
	}

	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("command failed: %s\nOutput: %s\nError: %w", strings.Join(args, " "), string(output), err)
		}
	}
	return nil
}
