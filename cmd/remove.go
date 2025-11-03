package cmd

import (
	"fmt"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// Remove a service from the service map
func Remove(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("service ID required")
	}
	serviceID := args[0]

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
		return fmt.Errorf("service '%s' not found", serviceID)
	}

	if err := config.SaveServiceMapConfig(serviceMap); err != nil {
		return fmt.Errorf("failed to save service map: %w", err)
	}

	ui.PrintInfo(fmt.Sprintf("Service '%s' removed successfully.", serviceID))
	return nil
}
