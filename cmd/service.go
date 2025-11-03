package cmd

import (
	"fmt"
	"os/exec"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// Service manages start, stop, and restart operations for Dexter services.
func Service(action, serviceShortName string) error {
	logFile, err := config.LogFile()
	if err != nil {
		return fmt.Errorf("failed to get log file: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	log := func(message string) {
		_, _ = fmt.Fprintln(logFile, message)
	}

	log(fmt.Sprintf("Service command called with action '%s' for service '%s'", action, serviceShortName))

	def, err := config.Resolve(serviceShortName)
	if err != nil {
		return fmt.Errorf("failed to resolve service '%s': %w", serviceShortName, err)
	}

	if !def.IsManageable() {
		return fmt.Errorf("service '%s' (type: %s) is not manageable", def.ShortName, def.Type)
	}
	if def.SystemdName == "" {
		return fmt.Errorf("service '%s' has no systemd service name defined", def.ShortName)
	}

	systemdServiceName := def.SystemdName

	// Check if the service file exists
	serviceExists, err := def.CheckSystemdService()
	if err != nil {
		return fmt.Errorf("failed to check for systemd service: %w", err)
	}
	if !serviceExists {
		return fmt.Errorf("systemd service '%s' not found. Run 'dex build %s' to install it", systemdServiceName, def.ShortName)
	}

	// Perform the action using systemctl --user
	switch action {
	case "start":
		return startService(systemdServiceName, log)
	case "stop":
		return stopService(systemdServiceName, log)
	case "restart":
		return restartService(systemdServiceName, log)
	default:
		log(fmt.Sprintf("Unknown service action: %s", action))
		return fmt.Errorf("unknown service action: %s", action)
	}
}

func startService(systemdServiceName string, log func(string)) error {
	ui.PrintInfo(fmt.Sprintf("Starting %s...", systemdServiceName))
	log(fmt.Sprintf("Starting %s...", systemdServiceName))

	cmd := exec.Command("systemctl", "--user", "start", systemdServiceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log(fmt.Sprintf("Failed to start %s: %v\n%s", systemdServiceName, err, string(output)))
		return fmt.Errorf("failed to start %s: %w\n%s", systemdServiceName, err, string(output))
	}

	ui.PrintSuccess(fmt.Sprintf("%s started", systemdServiceName))
	log(fmt.Sprintf("%s started successfully", systemdServiceName))
	return nil
}

func stopService(systemdServiceName string, log func(string)) error {
	ui.PrintInfo(fmt.Sprintf("Stopping %s...", systemdServiceName))
	log(fmt.Sprintf("Stopping %s...", systemdServiceName))

	cmd := exec.Command("systemctl", "--user", "stop", systemdServiceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log(fmt.Sprintf("Failed to stop %s: %v\n%s", systemdServiceName, err, string(output)))
		return fmt.Errorf("failed to stop %s: %w\n%s", systemdServiceName, err, string(output))
	}

	ui.PrintSuccess(fmt.Sprintf("%s stopped", systemdServiceName))
	log(fmt.Sprintf("%s stopped successfully", systemdServiceName))
	return nil
}

func restartService(systemdServiceName string, log func(string)) error {
	ui.PrintInfo(fmt.Sprintf("Restarting %s...", systemdServiceName))
	log(fmt.Sprintf("Restarting %s...", systemdServiceName))

	cmd := exec.Command("systemctl", "--user", "restart", systemdServiceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log(fmt.Sprintf("Failed to restart %s: %v\n%s", systemdServiceName, err, string(output)))
		return fmt.Errorf("failed to restart %s: %w\n%s", systemdServiceName, err, string(output))
	}

	ui.PrintSuccess(fmt.Sprintf("%s restarted", systemdServiceName))
	log(fmt.Sprintf("%s restarted successfully", systemdServiceName))
	return nil
}
