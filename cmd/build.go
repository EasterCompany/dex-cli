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

	// Load the service map
	serviceMap, err := config.LoadServiceMapConfig()
	if err != nil {
		return fmt.Errorf("failed to load service map: %w", err)
	}

	// Determine which services to build
	servicesToBuild := []config.ServiceDefinition{}
	if len(args) == 0 || (len(args) > 0 && args[0] == "all") {
		servicesToBuild = config.GetBuildableServices()
	} else {
		for _, arg := range args {
			serviceDef, err := config.Resolve(arg)
			if err != nil {
				return fmt.Errorf("service '%s' not found: %w", arg, err)
			}
			if !serviceDef.IsBuildable() {
				return fmt.Errorf("service '%s' is not buildable (type: %s)", arg, serviceDef.Type)
			}
			servicesToBuild = append(servicesToBuild, *serviceDef)
		}
	}

	// Check if these services are in the user's service-map.json
	var buildableInMap []config.ServiceDefinition
	for _, serviceDef := range servicesToBuild {
		foundInMap := false
		for _, services := range serviceMap.Services {
			for _, serviceEntry := range services {
				if serviceEntry.ID == serviceDef.ID {
					foundInMap = true
					break
				}
			}
			if foundInMap {
				break
			}
		}

		if !foundInMap {
			ui.PrintWarning(fmt.Sprintf("Service '%s' is defined but not in your service-map.json. Run 'dex add' to add it.", serviceDef.ShortName))
			continue
		}
		buildableInMap = append(buildableInMap, serviceDef)
	}

	// Build logic
	for _, serviceDef := range buildableInMap {
		if err := buildService(serviceDef, log); err != nil {
			ui.PrintError(fmt.Sprintf("Failed to build %s: %v", serviceDef.ID, err))
			log(fmt.Sprintf("Failed to build %s: %v", serviceDef.ID, err))
		}
	}

	ui.PrintSuccess("All requested services built.")
	log("Build command complete.")
	return nil
}

func buildService(serviceDef config.ServiceDefinition, log func(string)) error {
	if serviceDef.Source == "" || serviceDef.Source == "N/A" {
		ui.PrintInfo(fmt.Sprintf("Skipping %s: no source path defined", serviceDef.ID))
		log(fmt.Sprintf("Skipping %s: no source path defined", serviceDef.ID))
		return nil
	}

	sourcePath, err := config.ExpandPath(serviceDef.Source)
	if err != nil {
		return fmt.Errorf("failed to expand source path for %s: %w", serviceDef.ID, err)
	}

	if _, err := os.Stat(filepath.Join(sourcePath, "go.mod")); os.IsNotExist(err) {
		ui.PrintInfo(fmt.Sprintf("Skipping %s: not a Go project (no go.mod)", serviceDef.ID))
		log(fmt.Sprintf("Skipping %s: not a Go project (no go.mod)", serviceDef.ID))
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
		ui.PrintInfo(fmt.Sprintf("[%s] Running %s...", serviceDef.ID, step.name))
		log(fmt.Sprintf("[%s] Running %s...", serviceDef.ID, step.name))

		step.cmd.Dir = sourcePath
		output, err := step.cmd.CombinedOutput()
		if err != nil {
			log(fmt.Sprintf("[%s] Failed %s:\n%s", serviceDef.ID, step.name, string(output)))
			return fmt.Errorf("failed %s for %s:\n%s", step.name, serviceDef.ID, string(output))
		}
	}

	// --- Installation Step ---
	ui.PrintInfo(fmt.Sprintf("[%s] Installing systemd service...", serviceDef.ID))
	log(fmt.Sprintf("[%s] Installing systemd service...", serviceDef.ID))
	if err := installService(serviceDef, log); err != nil {
		log(fmt.Sprintf("[%s] Failed installation: %v", serviceDef.ID, err))
		return fmt.Errorf("failed to install service %s: %w", serviceDef.ID, err)
	}

	ui.PrintSuccess(fmt.Sprintf("%s processed successfully!", serviceDef.ID))
	log(fmt.Sprintf("%s processed successfully!", serviceDef.ID))
	return nil
}

func installService(serviceDef config.ServiceDefinition, log func(string)) error {
	if !serviceDef.IsManageable() {
		log(fmt.Sprintf("Skipping systemd installation for non-manageable service %s", serviceDef.ID))
		return nil
	}
	if serviceDef.SystemdName == "" {
		log(fmt.Sprintf("Skipping systemd installation for service %s: no systemd name defined", serviceDef.ID))
		return nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get user home directory: %w", err)
	}

	serviceFilePath := filepath.Join(homeDir, ".config", "systemd", "user", serviceDef.SystemdName)

	dexterBinPath, err := config.ExpandPath(config.DexterBin)
	if err != nil {
		return fmt.Errorf("could not expand dexter bin path: %w", err)
	}
	executablePath := filepath.Join(dexterBinPath, serviceDef.ID)

	logFilePath, err := serviceDef.GetLogPath()
	if err != nil {
		return fmt.Errorf("could not get log path: %w", err)
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
		{"systemctl", "--user", "enable", serviceDef.SystemdName},
		{"systemctl", "--user", "restart", serviceDef.SystemdName},
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
