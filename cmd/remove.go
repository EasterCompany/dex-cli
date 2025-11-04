package cmd

import (
	"bufio"
	"fmt"
	"os"
	"sort"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// Remove prompts the user to uninstall services that have artifacts on the system.
func Remove(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("remove does not take any arguments")
	}

	reader := bufio.NewReader(os.Stdin)
	serviceMap, err := config.LoadServiceMapConfig()
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to load service map: %w", err)
		}
		// If map doesn't exist, create a default one (it will be empty of manageable services)
		serviceMap = config.DefaultServiceMapConfig()
	}

	// Find all services that have artifacts
	removableServices := []config.ServiceDefinition{}
	for _, def := range config.GetAllServices() {
		if !def.IsManageable() {
			continue
		}
		has, err := HasArtifacts(def, serviceMap)
		if err != nil {
			ui.PrintWarning(fmt.Sprintf("Error checking service %s: %v", def.ShortName, err))
			continue
		}
		if has {
			removableServices = append(removableServices, def)
		}
	}

	if len(removableServices) == 0 {
		ui.PrintInfo("No services found to remove.")
		return nil
	}

	// Sort alphabetically
	sort.Slice(removableServices, func(i, j int) bool {
		return removableServices[i].ShortName < removableServices[j].ShortName
	})

	fmt.Println("Services that can be removed:")
	for i, def := range removableServices {
		fmt.Printf("  %d: %s\n", i+1, def.ShortName)
	}

	fmt.Print("Enter number(s) to remove (e.g., '1' or '1, 2'): ")
	input, _ := reader.ReadString('\n')

	indices, err := parseNumericInput(input)
	if err != nil {
		return err
	}

	servicesToRemove := []config.ServiceDefinition{}
	for _, idx := range indices {
		if idx < 1 || idx > len(removableServices) {
			return fmt.Errorf("invalid number: %d", idx)
		}
		servicesToRemove = append(servicesToRemove, removableServices[idx-1])
	}

	if len(servicesToRemove) == 0 {
		ui.PrintInfo("No services selected.")
		return nil
	}

	// Uninstall selected services
	for _, def := range servicesToRemove {
		if err := fullServiceUninstall(def, serviceMap); err != nil {
			return fmt.Errorf("failed to remove %s: %w", def.ShortName, err)
		}
		ui.PrintSuccess(fmt.Sprintf("Successfully removed %s.", def.ShortName))
	}

	ui.PrintSuccess("All selected services removed.")
	return nil
}
