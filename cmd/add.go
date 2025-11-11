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

// Add installs a new service.
func Add(args []string) error {
	serviceMap, err := config.LoadServiceMapConfig()
	if err != nil {
		return fmt.Errorf("failed to load service-map.json: %w", err)
	}

	// Find available services to add
	allServices := config.GetManageableServices()
	var availableServices []config.ServiceDefinition
	for _, service := range allServices {
		isInstalled := false
		for _, installedList := range serviceMap.Services {
			for _, installed := range installedList {
				if installed.ID == service.ID {
					isInstalled = true
					break
				}
			}
			if isInstalled {
				break
			}
		}
		if !isInstalled {
			availableServices = append(availableServices, service)
		}
	}

	if len(availableServices) == 0 {
		ui.PrintInfo("All manageable services are already installed.")
		return nil
	}

	// If arguments provided, add services by name
	if len(args) > 0 {
		return addServicesByName(args, availableServices, serviceMap)
	}

	// Otherwise, show interactive menu
	return addServicesInteractive(availableServices, serviceMap)
}

// addServicesByName adds services specified by their short names
func addServicesByName(names []string, availableServices []config.ServiceDefinition, serviceMap *config.ServiceMapConfig) error {
	// Build a map of short names to services for quick lookup
	servicesByName := make(map[string]config.ServiceDefinition)
	for _, service := range availableServices {
		servicesByName[service.ShortName] = service
	}

	// Validate all service names first
	var invalidNames []string
	var alreadyInstalled []string
	var toAdd []config.ServiceDefinition

	for _, name := range names {
		service, exists := servicesByName[name]
		if !exists {
			// Check if it's already installed
			allManageable := config.GetManageableServices()
			found := false
			for _, s := range allManageable {
				if s.ShortName == name {
					found = true
					alreadyInstalled = append(alreadyInstalled, name)
					break
				}
			}
			if !found {
				invalidNames = append(invalidNames, name)
			}
		} else {
			toAdd = append(toAdd, service)
		}
	}

	// Report errors
	if len(invalidNames) > 0 {
		ui.PrintError(fmt.Sprintf("Invalid service name(s): %s", strings.Join(invalidNames, ", ")))
	}
	if len(alreadyInstalled) > 0 {
		ui.PrintWarning(fmt.Sprintf("Already installed: %s", strings.Join(alreadyInstalled, ", ")))
	}
	if len(invalidNames) > 0 {
		return fmt.Errorf("some service names are invalid")
	}

	if len(toAdd) == 0 {
		ui.PrintInfo("No services to add.")
		return nil
	}

	// Add all valid services
	for _, service := range toAdd {
		ui.PrintInfo(fmt.Sprintf("Adding service: %s", service.ShortName))
		if utils.HasArtifacts(service) {
			ui.PrintWarning(fmt.Sprintf("This service has artifacts that will be backed up: %s", strings.Join(service.Backup.Artifacts, ", ")))
		}

		// Add to service map
		serviceMap.Services[service.Type] = append(serviceMap.Services[service.Type], service.ToServiceEntry())

		// TODO: Git clone, etc.
	}

	if err := config.SaveServiceMapConfig(serviceMap); err != nil {
		return fmt.Errorf("failed to save service-map.json: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Successfully added %d service(s).", len(toAdd)))
	return nil
}

// addServicesInteractive shows an interactive menu to select services
func addServicesInteractive(availableServices []config.ServiceDefinition, serviceMap *config.ServiceMapConfig) error {
	reader := bufio.NewReader(os.Stdin)
	for {
		ui.PrintInfo("Available services to add:")
		for i, service := range availableServices {
			fmt.Printf("  %d: %s\n", i+1, service.ShortName)
			if utils.HasArtifacts(service) {
				ui.PrintWarning(fmt.Sprintf("This service has artifacts that will be backed up: %s", strings.Join(service.Backup.Artifacts, ", ")))
			}
		}

		fmt.Println()
		ui.PrintInfo("Enter numbers to add (e.g., 1, 3-5):")
		input, _ := reader.ReadString('\n')
		selected, err := utils.ParseNumericInput(strings.TrimSpace(input), len(availableServices))
		if err != nil {
			return fmt.Errorf("invalid input: %w", err)
		}

		if len(selected) == 0 {
			ui.PrintWarning("No services selected.")
			continue
		}

		// Add selected services
		for _, num := range selected {
			service := availableServices[num-1]
			ui.PrintInfo(fmt.Sprintf("Adding service: %s", service.ShortName))

			// Add to service map
			serviceMap.Services[service.Type] = append(serviceMap.Services[service.Type], service.ToServiceEntry())

			// TODO: Git clone, etc.
		}

		if err := config.SaveServiceMapConfig(serviceMap); err != nil {
			return fmt.Errorf("failed to save service-map.json: %w", err)
		}

		ui.PrintSuccess(fmt.Sprintf("Successfully added %d service(s).", len(selected)))
		break
	}

	return nil
}
