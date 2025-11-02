package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// Service manages start, stop, and restart operations for Dexter services.
func Service(action, serviceName string) error {
	// Load the service map
	serviceMap, err := config.LoadServiceMap()
	if err != nil {
		return fmt.Errorf("failed to load service map: %w", err)
	}

	// Find the service entry
	var serviceEntry *config.ServiceEntry
	for _, services := range serviceMap.Services {
		for _, s := range services {
			if s.ID == serviceName {
				serviceEntry = &s
				break
			}
		}
		if serviceEntry != nil {
			break
		}
	}

	if serviceEntry == nil {
		return fmt.Errorf("service '%s' not found in service-map.json", serviceName)
	}

	// Check if this is a manageable service (dex-*-service only)
	if !strings.HasPrefix(serviceEntry.ID, "dex-") || !strings.HasSuffix(serviceEntry.ID, "-service") {
		return fmt.Errorf("service '%s' cannot be managed with start/stop/restart commands", serviceName)
	}

	// Perform the action using systemctl --user
	switch action {
	case "start":
		return startService(serviceEntry)
	case "stop":
		return stopService(serviceEntry)
	case "restart":
		return restartService(serviceEntry)
	default:
		return fmt.Errorf("unknown service action: %s", action)
	}
}

func startService(service *config.ServiceEntry) error {
	ui.PrintInfo(fmt.Sprintf("Starting %s...", service.ID))

	cmd := exec.Command("systemctl", "--user", "start", service.ID+".service")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start %s: %w\n%s", service.ID, err, string(output))
	}

	ui.PrintSuccess(fmt.Sprintf("%s started successfully", service.ID))
	return nil
}

func stopService(service *config.ServiceEntry) error {
	ui.PrintInfo(fmt.Sprintf("Stopping %s...", service.ID))

	cmd := exec.Command("systemctl", "--user", "stop", service.ID+".service")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop %s: %w\n%s", service.ID, err, string(output))
	}

	ui.PrintSuccess(fmt.Sprintf("%s stopped successfully", service.ID))
	return nil
}

func restartService(service *config.ServiceEntry) error {
	ui.PrintInfo(fmt.Sprintf("Restarting %s...", service.ID))

	cmd := exec.Command("systemctl", "--user", "restart", service.ID+".service")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to restart %s: %w\n%s", service.ID, err, string(output))
	}

	ui.PrintSuccess(fmt.Sprintf("%s restarted successfully", service.ID))
	return nil
}
