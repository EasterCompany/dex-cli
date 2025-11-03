package cmd

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// Remove prompts the user to remove a service from the service map.
func Remove(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("remove command takes no arguments")
	}

	reader := bufio.NewReader(os.Stdin)

	// Load existing service map
	serviceMap, err := config.LoadServiceMapConfig()
	if err != nil {
		if os.IsNotExist(err) {
			ui.PrintInfo("Service map not found. Nothing to remove.")
			return nil
		}
		return fmt.Errorf("failed to load service map: %w", err)
	}

	// Find services that are in the map and are "manageable" (can be removed)
	type removableService struct {
		Def  config.ServiceDefinition
		Type string // Service type (e.g., "cs", "be")
	}
	var removableServices []removableService

	for serviceType, services := range serviceMap.Services {
		for _, service := range services {
			def, err := config.ResolveByID(service.ID)
			if err != nil {
				continue // Skip services in map not in definitions (e.g., old/custom)
			}
			// Only show services that are manageable (not 'cli' or 'os')
			if !def.IsManageable() {
				continue
			}
			removableServices = append(removableServices, removableService{
				Def:  *def,
				Type: serviceType,
			})
		}
	}

	if len(removableServices) == 0 {
		ui.PrintInfo("No removable services found in your service map.")
		return nil
	}

	// Sort for consistent display
	sort.Slice(removableServices, func(i, j int) bool {
		return removableServices[i].Def.ShortName < removableServices[j].Def.ShortName
	})

	fmt.Println("Services available to remove:")
	for i, rs := range removableServices {
		fmt.Printf("  %d: %s (Type: %s)\n", i+1, rs.Def.ShortName, rs.Type)
	}

	fmt.Print("Enter number(s) to remove (e.g., '1' or '1, 2'): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		fmt.Println("No selection made.")
		return nil
	}

	selections := strings.Split(input, ",")
	servicesRemoved := 0
	servicesToKeep := make(map[string]bool) // Map of ID -> true

	for _, s := range selections {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}

		idx, err := strconv.Atoi(s)
		if err != nil || idx < 1 || idx > len(removableServices) {
			ui.PrintError(fmt.Sprintf("Invalid selection: '%s'. Skipping.", s))
			continue
		}

		rs := removableServices[idx-1]
		ui.PrintInfo(fmt.Sprintf("Marking '%s' for removal...", rs.Def.ShortName))
		// We'll rebuild the map by *not* adding this one
		servicesToKeep[rs.Def.ID] = false // Mark as "do not keep"
		servicesRemoved++
	}

	if servicesRemoved == 0 {
		return nil
	}

	// Rebuild the service map, skipping the ones marked for removal
	newServiceMap := config.DefaultServiceMapConfig() // Start fresh
	// Copy over non-manageable services (cli, os)
	newServiceMap.Services["cli"] = serviceMap.Services["cli"]
	newServiceMap.Services["os"] = serviceMap.Services["os"]

	// Copy over manageable services that we are *keeping*
	for _, rs := range removableServices {
		if keep, ok := servicesToKeep[rs.Def.ID]; !ok || keep {
			// Find the original ServiceEntry to keep
			for _, service := range serviceMap.Services[rs.Type] {
				if service.ID == rs.Def.ID {
					newServiceMap.Services[rs.Type] = append(newServiceMap.Services[rs.Type], service)
					break
				}
			}
		}
	}

	// Save the updated service map
	if err := config.SaveServiceMapConfig(newServiceMap); err != nil {
		return fmt.Errorf("failed to save service map: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Successfully removed %d service(s).", servicesRemoved))
	return nil
}
