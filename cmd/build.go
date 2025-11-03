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
	servicesToBuild := []config.ServiceEntry{}
	if len(args) == 0 || (len(args) > 0 && args[0] == "all") {
		for _, services := range serviceMap.Services {
			for _, service := range services {
				if strings.HasPrefix(service.ID, "dex-") && service.ID != "dex-cli" {
					servicesToBuild = append(servicesToBuild, service)
				}
			}
		}
	} else {
		for _, arg := range args {
			serviceName := arg
			if !strings.HasPrefix(serviceName, "dex-") {
				serviceName = "dex-" + serviceName + "-service"
			}
			found := false
			for _, services := range serviceMap.Services {
				for _, service := range services {
					if service.ID == serviceName {
						servicesToBuild = append(servicesToBuild, service)
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				return fmt.Errorf("service '%s' not found", arg)
			}
		}
	}

	// Build logic
	for serviceType, services := range serviceMap.Services {
		for _, service := range services {
			// Check if this service is in the list to be built
			buildThisOne := false
			for _, serviceToBuild := range servicesToBuild {
				if service.ID == serviceToBuild.ID {
					buildThisOne = true
					break
				}
			}

			if buildThisOne {
				if err := buildService(service, serviceType, log); err != nil {
					fmt.Printf("Failed to build %s: %v\n", service.ID, err)
					log(fmt.Sprintf("Failed to build %s: %v", service.ID, err))
				}
			}
		}
	}

	fmt.Println("All services built")
	log("All services built.")
	return nil
}

func buildService(service config.ServiceEntry, serviceType string, log func(string)) error {
	if service.Source == "" || service.Source == "system" {
		fmt.Printf("Skipping %s: no source path defined\n", service.ID)
		log(fmt.Sprintf("Skipping %s: no source path defined", service.ID))
		return nil
	}

	sourcePath, err := config.ExpandPath(service.Source)
	if err != nil {
		return fmt.Errorf("failed to expand source path for %s: %w", service.ID, err)
	}

	if _, err := os.Stat(filepath.Join(sourcePath, "go.mod")); os.IsNotExist(err) {
		fmt.Printf("Skipping %s: not a Go project (no go.mod)\n", service.ID)
		log(fmt.Sprintf("Skipping %s: not a Go project (no go.mod)", service.ID))
		return nil
	}

	// Expand paths for Dexter bin and logs
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
		{"Building", exec.Command("go", "build", "-o", filepath.Join(dexterBinPath, service.ID))},
	}

	for _, step := range steps {
		ui.PrintInfo(fmt.Sprintf("[%s] Running %s...", service.ID, step.name))
		log(fmt.Sprintf("[%s] Running %s...", service.ID, step.name))

		step.cmd.Dir = sourcePath
		output, err := step.cmd.CombinedOutput()

		if err != nil {
			log(fmt.Sprintf("[%s] Failed %s:\n%s", service.ID, step.name, string(output)))
			return fmt.Errorf("failed %s for %s:\n%s", step.name, service.ID, string(output))
		}
	}

	// --- Installation Step ---
	ui.PrintInfo(fmt.Sprintf("[%s] Installing systemd service...", service.ID))
	log(fmt.Sprintf("[%s] Installing systemd service...", service.ID))
	if err := installService(service, serviceType, log); err != nil {
		log(fmt.Sprintf("[%s] Failed installation: %v", service.ID, err))
		return fmt.Errorf("failed to install service %s: %w", service.ID, err)
	}

	ui.PrintSuccess(fmt.Sprintf("%s processed successfully!", service.ID))
	log(fmt.Sprintf("%s processed successfully!", service.ID))
	return nil
}

func installService(service config.ServiceEntry, serviceType string, log func(string)) error {
	if serviceType == "cli" {
		log(fmt.Sprintf("Skipping systemd installation for cli service %s", service.ID))
		return nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get user home directory: %w", err)
	}

	serviceFileName := fmt.Sprintf("%s.service", service.ID)
	serviceFilePath := filepath.Join(homeDir, ".config", "systemd", "user", serviceFileName)

	dexterBinPath, err := config.ExpandPath(config.DexterBin)
	if err != nil {
		return fmt.Errorf("could not expand dexter bin path: %w", err)
	}
	executablePath := filepath.Join(dexterBinPath, service.ID)

	dexterLogsPath, err := config.ExpandPath(config.DexterLogs)
	if err != nil {
		return fmt.Errorf("could not expand dexter logs path: %w", err)
	}
	logFilePath := filepath.Join(dexterLogsPath, fmt.Sprintf("%s.log", service.ID))

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
`, service.ID, executablePath, logFilePath, logFilePath)

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

	log(fmt.Sprintf("Successfully enabled and restarted %s", service.ID))
	return nil
}
