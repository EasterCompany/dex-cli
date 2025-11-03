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

// Build compiles one or all services
func Build(args []string) error {
	logFile, err := config.LogFile()
	if err != nil {
		return fmt.Errorf("failed to get log file: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	log := func(message string) {
		_, _ = fmt.Fprintln(logFile, message)
	}

	log(fmt.Sprintf("Build command called with args: %v", args))

	// Determine which services to build
	servicesToBuild := []config.ServiceDefinition{}
	if len(args) == 0 || (len(args) > 0 && args[0] == "all") {
		aliases := config.GetBuildableServices()
		for _, alias := range aliases {
			def, _ := config.Resolve(alias)
			servicesToBuild = append(servicesToBuild, def)
		}
	} else {
		for _, alias := range args {
			def, err := config.Resolve(alias)
			if err != nil {
				return err
			}
			if def.Source == "" || def.Type == "cli" {
				return fmt.Errorf("service '%s' is not buildable", alias)
			}
			servicesToBuild = append(servicesToBuild, def)
		}
	}

	if len(servicesToBuild) == 0 {
		ui.PrintInfo("No services to build.")
		return nil
	}

	// Build logic
	for _, serviceDef := range servicesToBuild {
		if err := buildService(serviceDef, log); err != nil {
			ui.PrintError(fmt.Sprintf("Failed to build %s: %v", serviceDef.ShortName, err))
			log(fmt.Sprintf("Failed to build %s: %v", serviceDef.ShortName, err))
		}
	}

	ui.PrintSuccess("Build process complete.")
	log("All services built.")
	return nil
}

func buildService(serviceDef config.ServiceDefinition, log func(string)) error {
	sourcePath, err := config.ExpandPath(serviceDef.Source)
	if err != nil {
		return fmt.Errorf("failed to expand source path for %s: %w", serviceDef.ShortName, err)
	}

	if _, err := os.Stat(filepath.Join(sourcePath, "go.mod")); os.IsNotExist(err) {
		ui.PrintWarning(fmt.Sprintf("Skipping %s: not a Go project (no go.mod)", serviceDef.ShortName))
		log(fmt.Sprintf("Skipping %s: not a Go project (no go.mod)", serviceDef.ShortName))
		return nil
	}

	// Expand paths for Dexter bin
	dexterBinPath, err := config.ExpandPath(config.DexterBin)
	if err != nil {
		return fmt.Errorf("could not expand dexter bin path: %w", err)
	}

	// --- Pipeline Steps ---
	steps := []struct {
		name string
		cmd  *exec.Cmd
	}{
		{"Formatting", exec.Command("go", "fmt", "./...")},
		{"Linting", exec.Command("golangci-lint", "run")},
		{"Testing", exec.Command("go", "test", "./...")},
		{"Building", exec.Command("go", "build", "-o", filepath.Join(dexterBinPath, serviceDef.ID))},
	}

	for _, step := range steps {
		ui.PrintInfo(fmt.Sprintf("[%s] Running %s...", serviceDef.ShortName, step.name))
		log(fmt.Sprintf("[%s] Running %s...", serviceDef.ShortName, step.name))

		step.cmd.Dir = sourcePath
		output, err := step.cmd.CombinedOutput()
		if err != nil {
			log(fmt.Sprintf("[%s] Failed %s:\n%s", serviceDef.ShortName, step.name, string(output)))
			// Don't fail the whole build for lint/test
			if step.name == "Linting" || step.name == "Testing" {
				ui.PrintWarning(fmt.Sprintf("[%s] %s failed, continuing build...", serviceDef.ShortName, step.name))
			} else {
				return fmt.Errorf("failed %s for %s:\n%s", step.name, serviceDef.ShortName, string(output))
			}
		}
	}

	// --- Installation Step ---
	ui.PrintInfo(fmt.Sprintf("[%s] Installing systemd service...", serviceDef.ShortName))
	log(fmt.Sprintf("[%s] Installing systemd service...", serviceDef.ShortName))
	if err := installService(serviceDef, log); err != nil {
		log(fmt.Sprintf("[%s] Failed installation: %v", serviceDef.ShortName, err))
		return fmt.Errorf("failed to install service %s: %w", serviceDef.ShortName, err)
	}

	ui.PrintSuccess(fmt.Sprintf("%s processed successfully!", serviceDef.ShortName))
	log(fmt.Sprintf("%s processed successfully!", serviceDef.ShortName))
	return nil
}

func installService(serviceDef config.ServiceDefinition, log func(string)) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get user home directory: %w", err)
	}

	serviceFileName := serviceDef.GetSystemdName()
	serviceFilePath := filepath.Join(homeDir, ".config", "systemd", "user", serviceFileName)

	dexterBinPath, err := config.ExpandPath(config.DexterBin)
	if err != nil {
		return fmt.Errorf("could not expand dexter bin path: %w", err)
	}
	executablePath := filepath.Join(dexterBinPath, serviceDef.ID)

	logFilePath, err := serviceDef.GetLogPath()
	if err != nil {
		return fmt.Errorf("could not expand dexter logs path: %w", err)
	}

	// Ensure directories exist
	if err := os.MkdirAll(filepath.Dir(serviceFilePath), 0755); err != nil {
		return fmt.Errorf("could not create systemd directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(logFilePath), 0755); err != nil {
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
`, serviceDef.ID, executablePath, logFilePath, logFilePath)

	// Write the service file
	if err := os.WriteFile(serviceFilePath, []byte(serviceFileContent), 0644); err != nil {
		return fmt.Errorf("could not write service file: %w", err)
	}

	log(fmt.Sprintf("Wrote systemd service file to %s", serviceFilePath))

	// Run systemctl commands
	commands := [][]string{
		{"systemctl", "--user", "daemon-reload"},
		{"systemctl", "--user", "enable", serviceFileName},
		{"systemctl", "--user", "restart", serviceFileName},
	}

	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		log(fmt.Sprintf("Executing: %s", strings.Join(args, " ")))
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("command failed: %s\nOutput: %s\nError: %w", strings.Join(args, " "), string(output), err)
		}
	}

	log(fmt.Sprintf("Successfully enabled and restarted %s", serviceDef.ID))
	return nil
}
