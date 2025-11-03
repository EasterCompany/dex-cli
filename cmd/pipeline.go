// cmd/pipeline.go
package cmd

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
func gitUpdateService(def config.ServiceDefinition) error {
	sourcePath, err := config.ExpandPath(def.Source)
	if err != nil {
		return fmt.Errorf("failed to expand source path: %w", err)
	}

	if !checkFileExists(sourcePath) {
		// Source doesn't exist, so let's clone it
		return gitCloneService(def)
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

// gitCloneService clones a service repository from its definition.
func gitCloneService(def config.ServiceDefinition) error {
	sourcePath, err := config.ExpandPath(def.Source)
	if err != nil {
		return fmt.Errorf("failed to expand source path: %w", err)
	}

	if checkFileExists(sourcePath) {
		return fmt.Errorf("source directory %s already exists", sourcePath)
	}

	ui.PrintInfo(fmt.Sprintf("Source for %s not found, cloning from %s...", def.ShortName, def.Repo))
	// Clone with --depth 1 to save space, only main branch
	cmd := exec.Command("git", "clone", "--depth", "1", "--branch", "main", def.Repo, sourcePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone %s: %w", def.ShortName, err)
	}
	return nil
}

// runServicePipelineStep executes a single step (format, lint, test, build)
func runServicePipelineStep(def config.ServiceDefinition, step string) error {
	sourcePath, err := config.ExpandPath(def.Source)
	if err != nil {
		return fmt.Errorf("failed to expand source path: %w", err)
	}

	if !checkFileExists(filepath.Join(sourcePath, "go.mod")) {
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

// installSystemdService creates and enables the systemd service file.
func installSystemdService(def config.ServiceDefinition) error {
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

// fullServiceUninstall stops, disables, and removes all artifacts for a service.
func fullServiceUninstall(def config.ServiceDefinition, serviceMap *config.ServiceMapConfig) error {
	ui.PrintInfo(fmt.Sprintf("Uninstalling all artifacts for %s...", def.ShortName))
	var lastErr error

	// 1. Stop and disable systemd service
	if def.IsManageable() {
		servicePath, _ := config.ExpandPath(def.GetSystemdPath())
		if checkFileExists(servicePath) {
			ui.PrintInfo(fmt.Sprintf("  Stopping and disabling %s...", def.SystemdName))
			_ = exec.Command("systemctl", "--user", "stop", def.SystemdName).Run()
			_ = exec.Command("systemctl", "--user", "disable", def.SystemdName).Run()

			if err := os.Remove(servicePath); err != nil {
				ui.PrintWarning(fmt.Sprintf("Failed to remove systemd file: %v", err))
				lastErr = err
			}
		}
	}

	// 2. Remove log file
	logPath, _ := config.ExpandPath(def.GetLogPath())
	if checkFileExists(logPath) {
		ui.PrintInfo(fmt.Sprintf("  Removing log file: %s", def.GetLogPath()))
		if err := os.Remove(logPath); err != nil {
			ui.PrintWarning(fmt.Sprintf("Failed to remove log file: %v", err))
			lastErr = err
		}
	}

	// 3. Remove binary
	binPath, _ := config.ExpandPath(def.GetBinaryPath())
	if checkFileExists(binPath) {
		ui.PrintInfo(fmt.Sprintf("  Removing binary: %s", def.GetBinaryPath()))
		if err := os.Remove(binPath); err != nil {
			ui.PrintWarning(fmt.Sprintf("Failed to remove binary: %v", err))
			lastErr = err
		}
	}

	// 4. Remove from service-map.json
	found := false
	if s, ok := serviceMap.Services[def.Type]; ok {
		for i, entry := range s {
			if entry.ID == def.ID {
				serviceMap.Services[def.Type] = append(s[:i], s[i+1:]...)
				found = true
				break
			}
		}
	}
	if found {
		ui.PrintInfo("  Removing from service-map.json...")
		if err := config.SaveServiceMapConfig(serviceMap); err != nil {
			ui.PrintWarning(fmt.Sprintf("Failed to save service map: %v", err))
			lastErr = err
		}
	}

	// 5. Remove source code (optional? for now, let's leave it, as it's the user's dev dir)
	// sourcePath, _ := config.ExpandPath(def.Source)
	// if checkFileExists(sourcePath) {
	// 	ui.PrintInfo(fmt.Sprintf("  Removing source directory: %s", def.Source))
	// 	if err := os.RemoveAll(sourcePath); err != nil {
	// 		ui.PrintWarning(fmt.Sprintf("Failed to remove source: %v", err))
	// 		lastErr = err
	// 	}
	// }

	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	return lastErr
}
