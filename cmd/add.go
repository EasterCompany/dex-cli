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

// Add prompts the user to create a new service in the new service map.
func Add(args []string) error {
	// Load existing service map
	serviceMap, err := config.LoadServiceMapConfig()
	if err != nil {
		// If the file doesn't exist, create a new map
		if os.IsNotExist(err) {
			serviceMap = config.DefaultServiceMapConfig()
		} else {
			return fmt.Errorf("failed to load service map: %w", err)
		}
	}

	// Figure out which services are already in the map
	existingServices := make(map[string]bool)
	for _, serviceList := range serviceMap.Services {
		for _, service := range serviceList {
			existingServices[service.ID] = true
		}
	}

	// Find available services (defined in ServiceDefinitions but not in service-map.json)
	available := []config.ServiceDefinition{}
	for _, def := range config.ServiceDefinitions {
		// Only "manageable" services can be added
		if !config.IsManageable(def.ShortName) {
			continue
		}
		if _, exists := existingServices[def.ID]; !exists {
			available = append(available, def)
		}
	}

	if len(available) == 0 {
		ui.PrintInfo("All available services are already added to service-map.json.")
		return nil
	}

	// Sort for consistent order
	sort.Slice(available, func(i, j int) bool {
		return available[i].ShortName < available[j].ShortName
	})

	fmt.Println("Available services to add:")
	for i, def := range available {
		fmt.Printf("  %d: %s (Port: %s, Type: %s)\n", i+1, def.ShortName, def.Port, def.Type)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter number(s) to add (e.g., 1 or 1,2): ")
	input, _ := reader.ReadString('\n')

	// --- New Parsing Logic ---
	servicesToAdd := []config.ServiceDefinition{}
	invalidInputs := []string{}
	seenIndices := make(map[int]bool) // To prevent adding the same service twice from "1,1"

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
		if num <= 0 || num > len(available) {
			invalidInputs = append(invalidInputs, fmt.Sprintf("'%d' (out of range)", num))
			continue
		}

		// 4. Get the 0-based index
		index := num - 1

		// 5. Check if already added
		if seenIndices[index] {
			continue
		}

		// 6. Add to list
		servicesToAdd = append(servicesToAdd, available[index])
		seenIndices[index] = true
	}

	// 7. Report errors
	if len(invalidInputs) > 0 {
		return fmt.Errorf("invalid inputs: %s", strings.Join(invalidInputs, ", "))
	}

	// 8. Check if any valid services were selected
	if len(servicesToAdd) == 0 {
		fmt.Println("No valid services selected.")
		return nil
	}

	// 9. Add all selected services to the map
	servicesAddedNames := []string{}
	for _, serviceDef := range servicesToAdd {
		address := "0.0.0.0"
		service := config.ServiceEntry{
			ID:     serviceDef.ID,
			Repo:   serviceDef.Repo,
			Source: serviceDef.Source,
			HTTP:   fmt.Sprintf("%s:%s", address, serviceDef.Port),
			Socket: fmt.Sprintf("ws://%s:%s", address, serviceDef.Port),
		}

		serviceMap.Services[serviceDef.Type] = append(serviceMap.Services[serviceDef.Type], service)
		servicesAddedNames = append(servicesAddedNames, serviceDef.ShortName)
	}

	// 10. Save the map
	if err := config.SaveServiceMapConfig(serviceMap); err != nil {
		return fmt.Errorf("failed to save service map: %w", err)
	}

	// 11. Final report
	if len(servicesAddedNames) > 0 {
		ui.PrintSuccess(fmt.Sprintf("Successfully added: %s", strings.Join(servicesAddedNames, ", ")))
		ui.PrintInfo("Run 'dex build all' or 'dex build <alias>' to install them.")
	}

	return nil
}
