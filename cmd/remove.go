package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// Remove a service from the service map and systemd
func Remove(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("service alias required (e.g., 'dex remove event')")
	}
	serviceAlias := args[0]

	// 1. Resolve alias
	def, err := config.Resolve(serviceAlias)
	if err != nil {
		return err
	}
	serviceID := def.ID

	// 2. Remove from service-map.json
	serviceMap, err := config.LoadServiceMapConfig()
	if err != nil {
		return fmt.Errorf("failed to load service map: %w", err)
	}

	found := false
	for serviceType, services := range serviceMap.Services {
		for i, service := range services {
			if service.ID == serviceID {
				serviceMap.Services[serviceType] = append(services[:i], services[i+1:]...)
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		ui.PrintWarning(fmt.Sprintf("Service '%s' (%s) not found in service-map.json. Checking systemd...", serviceAlias, serviceID))
	} else {
		if err := config.SaveServiceMapConfig(serviceMap); err != nil {
			return fmt.Errorf("failed to save service map: %w", err)
		}
		ui.PrintSuccess(fmt.Sprintf("Service '%s' removed from service-map.json.", serviceAlias))
	}

	// 3. Remove from systemd
	if config.IsManageable(serviceAlias) {
		ui.PrintInfo(fmt.Sprintf("Attempting to remove systemd service for '%s'...", serviceAlias))
		if err := removeSystemdService(def); err != nil {
			// Don't fail if systemd removal fails, just warn
			ui.PrintWarning(fmt.Sprintf("Could not remove systemd service: %v", err))
		} else {
			ui.PrintSuccess(fmt.Sprintf("Systemd service for '%s' removed.", serviceAlias))
		}
	}

	return nil
}

func removeSystemdService(def config.ServiceDefinition) error {
	systemdName := def.GetSystemdName()

	// 1. Stop the service
	cmdStop := exec.Command("systemctl", "--user", "stop", systemdName)
	if err := cmdStop.Run(); err != nil {
		// Don't fail if already stopped
		fmt.Printf("Service '%s' could not be stopped (may already be stopped).\n", systemdName)
	}

	// 2. Disable the service
	cmdDisable := exec.Command("systemctl", "--user", "disable", systemdName)
	if err := cmdDisable.Run(); err != nil {
		return fmt.Errorf("failed to disable service: %w", err)
	}

	// 3. Remove the service file
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home dir: %w", err)
	}
	serviceFilePath := filepath.Join(homeDir, ".config", "systemd", "user", systemdName)
	if err := os.Remove(serviceFilePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("service file not found at %s", serviceFilePath)
		}
		return fmt.Errorf("failed to remove service file: %w", err)
	}

	// 4. Reload daemon
	cmdReload := exec.Command("systemctl", "--user", "daemon-reload")
	if err := cmdReload.Run(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	return nil
}
