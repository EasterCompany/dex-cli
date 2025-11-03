// cmd/add.go
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"sort"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// Add prompts the user to clone, build, and install new services.
// It only lists services that do not have any artifacts on the system.
func Add(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("add command takes no arguments")
	}

	serviceMap, err := config.LoadServiceMapConfig()
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load service map: %w", err)
	}
	if serviceMap == nil {
		serviceMap = config.DefaultServiceMapConfig()
	}

	// Find services that are "clean" (no artifacts) and not in the map
	type selectableService struct {
		Index int
		Def   config.ServiceDefinition
	}
	availableServices := []selectableService{}
	idx := 1
	allServices := config.GetAllServices()

	for _, def := range allServices {
		if !def.IsManageable() {
			continue
		}

		hasArtifacts, err := HasArtifacts(def, serviceMap)
		if err != nil {
			ui.PrintWarning(fmt.Sprintf("Could not check status of %s: %v", def.ShortName, err))
			continue
		}

		if !hasArtifacts {
			availableServices = append(availableServices, selectableService{Index: idx, Def: def})
			idx++
		}
	}

	if len(availableServices) == 0 {
		ui.PrintInfo("All manageable services are already installed.")
		return nil
	}

	// Sort by name for consistent display
	sort.Slice(availableServices, func(i, j int) bool {
		return availableServices[i].Def.ShortName < availableServices[j].Def.ShortName
	})

	fmt.Println("Available services to add:")
	for _, s := range availableServices {
		fmt.Printf("  %d: %s (Port: %s, Type: %s)\n", s.Index, s.Def.ShortName, s.Def.Port, s.Def.Type)
	}

	fmt.Print("Enter number(s) to add (e.g., '1' or '1, 2'): ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')

	selectedIndices, err := parseNumericInput(input)
	if err != nil {
		return fmt.Errorf("invalid input: %w", err)
	}

	servicesToAdd := []config.ServiceDefinition{}
	for _, selectedIdx := range selectedIndices {
		found := false
		for _, s := range availableServices {
			if s.Index == selectedIdx {
				servicesToAdd = append(servicesToAdd, s.Def)
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("invalid number: %d", selectedIdx)
		}
	}

	if len(servicesToAdd) == 0 {
		ui.PrintInfo("No services selected.")
		return nil
	}

	servicesAdded := 0
	for _, def := range servicesToAdd {
		fmt.Println(ui.Colorize(fmt.Sprintf("--- Installing %s ---", def.ShortName), ui.ColorCyan))

		// 1. Download
		if err := gitCloneService(def); err != nil {
			ui.PrintError(fmt.Sprintf("Failed to clone %s: %v", def.ShortName, err))
			ui.PrintInfo(fmt.Sprintf("Cleaning up artifacts for %s...", def.ShortName))
			_ = fullServiceUninstall(def, serviceMap) // Attempt cleanup
			continue
		}

		// 2. Add to service map *before* build
		serviceMap.Services[def.Type] = append(serviceMap.Services[def.Type], def.ToServiceEntry())
		if err := config.SaveServiceMapConfig(serviceMap); err != nil {
			ui.PrintError(fmt.Sprintf("Failed to save service map for %s: %v", def.ShortName, err))
			_ = fullServiceUninstall(def, serviceMap) // Attempt cleanup
			continue
		}

		// 3. Run full build pipeline
		pipeline := []string{"format", "lint", "test", "build"}
		pipelineFailed := false
		for _, step := range pipeline {
			if err := runServicePipelineStep(def, step); err != nil {
				ui.PrintError(fmt.Sprintf("Failed pipeline step '%s' for %s: %v", step, def.ShortName, err))
				ui.PrintInfo(fmt.Sprintf("Cleaning up artifacts for %s...", def.ShortName))
				_ = fullServiceUninstall(def, serviceMap) // Attempt cleanup
				pipelineFailed = true
				break
			}
		}
		if pipelineFailed {
			continue
		}

		// 4. Install systemd service
		if err := installSystemdService(def); err != nil {
			ui.PrintError(fmt.Sprintf("Failed to install systemd service for %s: %v", def.ShortName, err))
			ui.PrintInfo(fmt.Sprintf("Cleaning up artifacts for %s...", def.ShortName))
			_ = fullServiceUninstall(def, serviceMap) // Attempt cleanup
			continue
		}

		ui.PrintSuccess(fmt.Sprintf("Successfully installed %s!", def.ShortName))
		servicesAdded++
	}

	fmt.Println()
	ui.PrintSuccess(fmt.Sprintf("Successfully added %d service(s).", servicesAdded))
	return nil
}
