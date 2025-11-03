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

// ServiceToRemove is a helper struct to link a service entry with its short alias
type ServiceToRemove struct {
	Entry     config.ServiceEntry
	ShortName string
	Type      string
}

// Remove prompts the user to remove existing services from the service map.
func Remove(args []string) error {
	// Load existing service map
	serviceMap, err := config.LoadServiceMapConfig()
	if err != nil {
		if os.IsNotExist(err) {
			ui.PrintError("service-map.json not found. Nothing to remove.")
			return nil
		}
		return fmt.Errorf("failed to load service map: %w", err)
	}

	// Find services that are in the map and are "manageable"
	removable := []ServiceToRemove{}
	for serviceType, serviceList := range serviceMap.Services {
		for _, service := range serviceList {
			// Get the service's definition to find its alias and check if manageable
			def, err := config.ResolveByID(service.ID)
			if err != nil {
				continue // Skip if it's not in the universal definitions
			}

			if config.IsManageable(def.ShortName) {
				removable = append(removable, ServiceToRemove{
					Entry:     service,
					ShortName: def.ShortName,
					Type:      serviceType,
				})
			}
		}
	}

	if len(removable) == 0 {
		ui.PrintInfo("No manageable services found in service-map.json to remove.")
		return nil
	}

	// Sort for consistent order
	sort.Slice(removable, func(i, j int) bool {
		return removable[i].ShortName < removable[j].ShortName
	})

	fmt.Println("Services available to remove:")
	for i, service := range removable {
		fmt.Printf("  %d: %s (ID: %s, Type: %s)\n", i+1, service.ShortName, service.Entry.ID, service.Type)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter number(s) to remove (e.g., 1 or 1,2): ")
	input, _ := reader.ReadString('\n')

	// --- Parsing Logic ---
	servicesToRemove := []ServiceToRemove{}
	invalidInputs := []string{}
	seenIndices := make(map[int]bool) // To prevent removing the same service twice from "1,1"

	// 1. Clean and split the input
	input = strings.TrimSpace(input)
	input = strings.TrimSuffix(input, ",")
	numberStrings := strings.Split(input, ",")

	if len(numberStrings) == 0 {
		fmt.Println("No input provided.")
		return nil
	}

	// 2. Parse each number string
	for _, numStr := range numberStrings {
		numStr = strings.TrimSpace(numStr)
		if numStr == "" {
			continue // Skip empty entries (e.g., "1,,2")
		}

		num, err := strconv.Atoi(numStr)
		if err != nil {
			invalidInputs = append(invalidInputs, fmt.Sprintf("'%s' (not a number)", numStr))
			continue
		}

		// 3. Validate the number
		if num <= 0 || num > len(removable) {
			invalidInputs = append(invalidInputs, fmt.Sprintf("'%d' (out of range)", num))
			continue
		}

		// 4. Get the 0-based index
		index := num - 1

		// 5. Check if already added to the remove list
		if seenIndices[index] {
			continue
		}

		// 6. Add to list
		servicesToRemove = append(servicesToRemove, removable[index])
		seenIndices[index] = true
	}

	// 7. Report errors
	if len(invalidInputs) > 0 {
		return fmt.Errorf("invalid inputs: %s", strings.Join(invalidInputs, ", "))
	}

	// 8. Check if any valid services were selected
	if len(servicesToRemove) == 0 {
		fmt.Println("No valid services selected.")
		return nil
	}

	// 9. Create a map of IDs to remove for efficient filtering
	removeIDMap := make(map[string]bool)
	removedNames := []string{}
	for _, service := range servicesToRemove {
		removeIDMap[service.Entry.ID] = true
		removedNames = append(removedNames, service.ShortName)
	}

	// 10. Filter the service map
	// Create a new map to hold the services we're keeping
	newServiceMapServices := config.DefaultServiceMapConfig().Services
	// Copy over non-manageable types first
	newServiceMapServices["cli"] = serviceMap.Services["cli"]
	newServiceMapServices["os"] = serviceMap.Services["os"]

	// Iterate through all manageable service types
	manageableTypes := []string{"fe", "cs", "be", "th"}
	for _, serviceType := range manageableTypes {
		for _, service := range serviceMap.Services[serviceType] {
			// If this service is NOT in the remove list, add it to our new map
			if _, shouldRemove := removeIDMap[service.ID]; !shouldRemove {
				newServiceMapServices[serviceType] = append(newServiceMapServices[serviceType], service)
			}
		}
	}

	// 11. Replace the old service map with the filtered one
	serviceMap.Services = newServiceMapServices

	// 12. Save the map
	if err := config.SaveServiceMapConfig(serviceMap); err != nil {
		return fmt.Errorf("failed to save service map: %w", err)
	}

	// 13. Final report
	if len(removedNames) > 0 {
		ui.PrintSuccess(fmt.Sprintf("Successfully removed: %s", strings.Join(removedNames, ", ")))
		ui.PrintInfo("Note: This only updates service-map.json. Run 'dex stop <alias>' to stop running services.")
	}

	return nil
}
