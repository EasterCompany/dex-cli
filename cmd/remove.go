package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
	"github.com/EasterCompany/dex-cli/utils"
)

// Remove uninstalls a service.
func Remove(args []string) error {
	serviceMap, err := config.LoadServiceMapConfig()
	if err != nil {
		return fmt.Errorf("failed to load service-map.json: %w", err)
	}

	// Collect all installed manageable services (excludes cli and os)
	var installedServices []config.ServiceDefinition
	for _, serviceList := range serviceMap.Services {
		for _, serviceEntry := range serviceList {
			def := config.GetServiceDefinition(serviceEntry.ID)
			// Only include manageable services (not cli or os)
			if def.ID != "" && def.IsManageable() {
				installedServices = append(installedServices, def)
			}
		}
	}

	if len(installedServices) == 0 {
		ui.PrintInfo("No manageable services are currently installed.")
		return nil
	}

	// If arguments provided, remove services by name
	if len(args) > 0 {
		return removeServicesByName(args, installedServices, serviceMap)
	}

	// Otherwise, show interactive menu
	return removeServicesInteractive(installedServices, serviceMap)
}

// removeServicesByName removes services specified by their short names
func removeServicesByName(names []string, installedServices []config.ServiceDefinition, serviceMap *config.ServiceMapConfig) error {
	// Build a map of short names to services for quick lookup
	servicesByName := make(map[string]config.ServiceDefinition)
	for _, service := range installedServices {
		servicesByName[service.ShortName] = service
	}

	// Validate all service names first
	var invalidNames []string
	var notInstalled []string
	var toRemove []config.ServiceDefinition

	for _, name := range names {
		service, exists := servicesByName[name]
		if !exists {
			// Check if it's a valid service but not installed
			allManageable := config.GetManageableServices()
			found := false
			for _, s := range allManageable {
				if s.ShortName == name {
					found = true
					notInstalled = append(notInstalled, name)
					break
				}
			}
			if !found {
				invalidNames = append(invalidNames, name)
			}
		} else {
			toRemove = append(toRemove, service)
		}
	}

	// Report errors
	if len(invalidNames) > 0 {
		ui.PrintError(fmt.Sprintf("Invalid service name(s): %s", strings.Join(invalidNames, ", ")))
	}
	if len(notInstalled) > 0 {
		ui.PrintWarning(fmt.Sprintf("Not installed: %s", strings.Join(notInstalled, ", ")))
	}
	if len(invalidNames) > 0 {
		return fmt.Errorf("some service names are invalid")
	}

	if len(toRemove) == 0 {
		ui.PrintInfo("No services to remove.")
		return nil
	}

	// Remove all valid services
	for _, service := range toRemove {
		ui.PrintInfo(fmt.Sprintf("Removing service: %s", service.ShortName))

		// Remove from service map
		for serviceType, serviceList := range serviceMap.Services {
			for i, serviceEntry := range serviceList {
				if serviceEntry.ID == service.ID {
					serviceMap.Services[serviceType] = append(serviceList[:i], serviceList[i+1:]...)
					break
				}
			}
		}

		// TODO: Remove source code, binaries, systemd service, etc.
	}

	if err := config.SaveServiceMapConfig(serviceMap); err != nil {
		return fmt.Errorf("failed to save service-map.json: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Successfully removed %d service(s).", len(toRemove)))
	return nil
}

// removeServicesInteractive shows an interactive menu to select services to remove
func removeServicesInteractive(installedServices []config.ServiceDefinition, serviceMap *config.ServiceMapConfig) error {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println()
		ui.PrintInfo("Installed services to remove:")
		for i, service := range installedServices {
			fmt.Printf("  %d: %s\n", i+1, service.ShortName)
			if utils.HasArtifacts(service) {
				ui.PrintWarning(fmt.Sprintf("This service has artifacts that will be backed up: %s", strings.Join(service.Backup.Artifacts, ", ")))
			}
		}

		fmt.Println()
		ui.PrintInfo("Enter numbers to remove (e.g., 1, 3-5):")
		input, _ := reader.ReadString('\n')
		selected, err := utils.ParseNumericInput(strings.TrimSpace(input), len(installedServices))
		if err != nil {
			return fmt.Errorf("invalid input: %w", err)
		}

		if len(selected) == 0 {
			ui.PrintWarning("No services selected.")
			continue
		}

		// Remove selected services
		for _, num := range selected {
			serviceToRemove := installedServices[num-1]
			ui.PrintInfo(fmt.Sprintf("Removing service: %s", serviceToRemove.ShortName))

			// Remove from service map
			for serviceType, serviceList := range serviceMap.Services {
				for i, serviceEntry := range serviceList {
					if serviceEntry.ID == serviceToRemove.ID {
						serviceMap.Services[serviceType] = append(serviceList[:i], serviceList[i+1:]...)
						break
					}
				}
			}

			// TODO: Remove source code, binaries, systemd service, etc.
		}

		if err := config.SaveServiceMapConfig(serviceMap); err != nil {
			return fmt.Errorf("failed to save service-map.json: %w", err)
		}

		ui.PrintSuccess(fmt.Sprintf("Successfully removed %d service(s).", len(selected)))
		break
	}

	return nil
}
