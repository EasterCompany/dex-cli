package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/EasterCompany/dex-cli/config"
)

// Service manages start, stop, and restart operations for Dexter services.
func Service(action, serviceName string) error {
	logFile, err := config.LogFile()
	if err != nil {
		return fmt.Errorf("failed to get log file: %w", err)
	}
	defer func() { _ = logFile.Close() }()

	log := func(message string) {
		_, _ = fmt.Fprintln(logFile, message)
	}

	log(fmt.Sprintf("Service command called with action '%s' for service '%s'", action, serviceName))

	// Load the service map
	serviceMap, err := config.LoadServiceMapConfig()
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
		log(fmt.Sprintf("Service '%s' not found in service-map.json", serviceName))
		return fmt.Errorf("service '%s' not found in service-map.json", serviceName)
	}

	// Check if this is a manageable service (dex-*-service only)
	if !strings.HasPrefix(serviceEntry.ID, "dex-") || !strings.HasSuffix(serviceEntry.ID, "-service") {
		log(fmt.Sprintf("Service '%s' cannot be managed with start/stop/restart commands", serviceName))
		return fmt.Errorf("service '%s' cannot be managed with start/stop/restart commands", serviceName)
	}

	// Perform the action using systemctl --user
	switch action {
	case "start":
		return startService(serviceEntry, log)
	case "stop":
		return stopService(serviceEntry, log)
	case "restart":
		return restartService(serviceEntry, log)
	default:
		log(fmt.Sprintf("Unknown service action: %s", action))
		return fmt.Errorf("unknown service action: %s", action)
	}
}

func startService(service *config.ServiceEntry, log func(string)) error {
	fmt.Printf("Starting %s...\n", service.ID)
	log(fmt.Sprintf("Starting %s...", service.ID))

	cmd := exec.Command("systemctl", "--user", "start", service.ID+".service")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log(fmt.Sprintf("Failed to start %s: %v\n%s", service.ID, err, string(output)))
		return fmt.Errorf("failed to start %s: %w\n%s", service.ID, err, string(output))
	}

	fmt.Printf("%s started successfully\n", service.ID)
	log(fmt.Sprintf("%s started successfully", service.ID))
	return nil
}

func stopService(service *config.ServiceEntry, log func(string)) error {
	fmt.Printf("Stopping %s...\n", service.ID)
	log(fmt.Sprintf("Stopping %s...", service.ID))

	cmd := exec.Command("systemctl", "--user", "stop", service.ID+".service")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log(fmt.Sprintf("Failed to stop %s: %v\n%s", service.ID, err, string(output)))
		return fmt.Errorf("failed to stop %s: %w\n%s", service.ID, err, string(output))
	}

	fmt.Printf("%s stopped successfully\n", service.ID)
	log(fmt.Sprintf("%s stopped successfully", service.ID))
	return nil
}

func restartService(service *config.ServiceEntry, log func(string)) error {
	fmt.Printf("Restarting %s...\n", service.ID)
	log(fmt.Sprintf("Restarting %s...", service.ID))

	cmd := exec.Command("systemctl", "--user", "restart", service.ID+".service")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log(fmt.Sprintf("Failed to restart %s: %v\n%s", service.ID, err, string(output)))
		return fmt.Errorf("failed to restart %s: %w\n%s", service.ID, err, string(output))
	}

	fmt.Printf("%s restarted successfully\n", service.ID)
	log(fmt.Sprintf("%s restarted successfully", service.ID))
	return nil
}
