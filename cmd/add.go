package cmd

import (
	"bufio"
	"fmt"
	"os"
	"sort"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// Add prompts the user to create a new service in the new service map.
func Add(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("add does not take any arguments")
	}

	reader := bufio.NewReader(os.Stdin)
	serviceMap, err := config.LoadServiceMapConfig()
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to load service map: %w", err)
		}
		// If map doesn't exist, create a default one
		serviceMap = config.DefaultServiceMapConfig()
	}

	// Find all manageable services that do NOT have artifacts
	availableServices := []config.ServiceDefinition{}
	for _, def := range config.GetAllServices() {
		if !def.IsManageable() {
			continue
		}
		has, err := HasArtifacts(def, serviceMap)
		if err != nil {
			ui.PrintWarning(fmt.Sprintf("Error checking service %s: %v", def.ShortName, err))
			continue
		}
		if !has {
			availableServices = append(availableServices, def)
		}
	}

	if len(availableServices) == 0 {
		ui.PrintInfo("All manageable services are already installed.")
		return nil
	}

	// Sort services alphabetically
	sort.Slice(availableServices, func(i, j int) bool {
		return availableServices[i].ShortName < availableServices[j].ShortName
	})

	fmt.Println("Available services to add:")
	for i, def := range availableServices {
		fmt.Printf("  %d: %s (Port: %s, Type: %s)\n", i+1, def.ShortName, def.Port, def.Type)
	}

	fmt.Print("Enter number(s) to add (e.g., '1' or '1, 2'): ")
	input, _ := reader.ReadString('\n')

	indices, err := parseNumericInput(input)
	if err != nil {
		return err
	}

	servicesToAdd := []config.ServiceDefinition{}
	for _, idx := range indices {
		if idx < 1 || idx > len(availableServices) {
			return fmt.Errorf("invalid number: %d", idx)
		}
		servicesToAdd = append(servicesToAdd, availableServices[idx-1])
	}

	if len(servicesToAdd) == 0 {
		ui.PrintInfo("No services selected.")
		return nil
	}

	// Process selected services
	for _, def := range servicesToAdd {
		fmt.Println()
		ui.PrintInfo(ui.Colorize(fmt.Sprintf("--- Installing %s ---", def.ShortName), ui.ColorCyan))

		// 1. Clone
		if err := gitCloneService(def); err != nil {
			ui.PrintError(fmt.Sprintf("Failed to clone %s: %v", def.ShortName, err))
			ui.PrintInfo("Cleaning up...")
			_ = fullServiceUninstall(def, serviceMap) // Attempt cleanup
			return err
		}

		// 2. Format
		if err := runServicePipelineStep(def, "format"); err != nil {
			ui.PrintError(fmt.Sprintf("Failed to format %s: %v", def.ShortName, err))
			ui.PrintInfo("Cleaning up...")
			_ = fullServiceUninstall(def, serviceMap) // Attempt cleanup
			return err
		}

		// 3. Lint
		if err := runServicePipelineStep(def, "lint"); err != nil {
			ui.PrintError(fmt.Sprintf("Failed to lint %s: %v", def.ShortName, err))
			ui.PrintInfo("Cleaning up...")
			_ = fullServiceUninstall(def, serviceMap) // Attempt cleanup
			return err
		}

		// 4. Test
		if err := runServicePipelineStep(def, "test"); err != nil {
			ui.PrintError(fmt.Sprintf("Failed to test %s: %v", def.ShortName, err))
			ui.PrintInfo("Cleaning up...")
			_ = fullServiceUninstall(def, serviceMap) // Attempt cleanup
			return err
		}

		// 5. Build
		if err := runServicePipelineStep(def, "build"); err != nil {
			ui.PrintError(fmt.Sprintf("Failed to build %s: %v", def.ShortName, err))
			ui.PrintInfo("Cleaning up...")
			_ = fullServiceUninstall(def, serviceMap) // Attempt cleanup
			return err
		}

		// 6. Install
		if err := installSystemdService(def); err != nil {
			ui.PrintError(fmt.Sprintf("Failed to install %s: %v", def.ShortName, err))
			ui.PrintInfo("Cleaning up...")
			_ = fullServiceUninstall(def, serviceMap) // Attempt cleanup
			return err
		}

		// 7. Add to service-map.json
		serviceMap.Services[def.Type] = append(serviceMap.Services[def.Type], def.ToServiceEntry())
		if err := config.SaveServiceMapConfig(serviceMap); err != nil {
			ui.PrintError(fmt.Sprintf("Failed to save service map: %v", err))
			ui.PrintInfo("Cleaning up...")
			_ = fullServiceUninstall(def, serviceMap) // Attempt cleanup
			return err
		}

		ui.PrintSuccess(fmt.Sprintf("Successfully installed %s!", def.ShortName))
	}

	fmt.Println()
	ui.PrintSuccess("All selected services installed successfully.")
	return nil
}
