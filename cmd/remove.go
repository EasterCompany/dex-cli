// cmd/remove.go
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"sort"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// Remove prompts the user to uninstall and remove services.
// It lists any service that has *any* artifact on the system.
func Remove(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("remove command takes no arguments")
	}

	serviceMap, err := config.LoadServiceMapConfig()
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load service map: %w", err)
	}
	if serviceMap == nil {
		serviceMap = config.DefaultServiceMapConfig()
	}

	// Find services that are "dirty" (have artifacts)
	type selectableService struct {
		Index int
		Def   config.ServiceDefinition
	}
	removableServices := []selectableService{}
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

		if hasArtifacts {
			removableServices = append(removableServices, selectableService{Index: idx, Def: def})
			idx++
		}
	}

	if len(removableServices) == 0 {
		ui.PrintInfo("No manageable services found to remove.")
		return nil
	}

	// Sort by name for consistent display
	sort.Slice(removableServices, func(i, j int) bool {
		return removableServices[i].Def.ShortName < removableServices[j].Def.ShortName
	})

	fmt.Println("Services that can be removed:")
	for _, s := range removableServices {
		fmt.Printf("  %d: %s\n", s.Index, s.Def.ShortName)
	}

	fmt.Print("Enter number(s) to remove (e.g., '1' or '1, 2'): ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')

	selectedIndices, err := parseNumericInput(input)
	if err != nil {
		return fmt.Errorf("invalid input: %w", err)
	}

	servicesToRemove := []config.ServiceDefinition{}
	for _, selectedIdx := range selectedIndices {
		found := false
		for _, s := range removableServices {
			if s.Index == selectedIdx {
				servicesToRemove = append(servicesToRemove, s.Def)
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("invalid number: %d", selectedIdx)
		}
	}

	if len(servicesToRemove) == 0 {
		ui.PrintInfo("No services selected.")
		return nil
	}

	servicesRemoved := 0
	for _, def := range servicesToRemove {
		if err := fullServiceUninstall(def, serviceMap); err != nil {
			ui.PrintError(fmt.Sprintf("Failed to fully remove %s: %v", def.ShortName, err))
			continue
		}
		ui.PrintSuccess(fmt.Sprintf("Successfully removed %s.", def.ShortName))
		servicesRemoved++
	}

	fmt.Println()
	ui.PrintSuccess(fmt.Sprintf("Successfully removed %d service(s).", servicesRemoved))
	return nil
}
