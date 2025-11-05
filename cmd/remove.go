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
	if len(args) > 0 {
		return fmt.Errorf("remove command takes no arguments")
	}

	serviceMap, err := config.LoadServiceMapConfig()
	if err != nil {
		return fmt.Errorf("failed to load service-map.json: %w", err)
	}

	// Collect all installed services
	var installedServices []config.ServiceDefinition
	for _, serviceList := range serviceMap.Services {
		for _, serviceEntry := range serviceList {
			def := config.GetServiceDefinition(serviceEntry.ID)
			if def.ID != "" {
				installedServices = append(installedServices, def)
			}
		}
	}

	if len(installedServices) == 0 {
		ui.PrintInfo("No services are currently installed.")
		return nil
	}

	// Prompt user to select services to remove
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
