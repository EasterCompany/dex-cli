package cmd

import (
	"bufio"
	"fmt"
	"os"
	"sort"
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
	var serviceDef config.ServiceDefinition
	for {
		fmt.Print("Enter service alias to add (e.g., 'event'): ")
		input, _ := reader.ReadString('\n')
		shortName := strings.TrimSpace(input)

		// Check if the input is one of the available services
		found := false
		for _, def := range available {
			if def.ShortName == shortName {
				serviceDef = def
				found = true
				break
			}
		}

		if found {
			break
		}
		fmt.Println("Invalid alias. Please choose from the list above.")
	}

	// Construct the new ServiceEntry from the definition
	address := "0.0.0.0"
	service := config.ServiceEntry{
		ID:     serviceDef.ID,
		Repo:   serviceDef.Repo,
		Source: serviceDef.Source,
		HTTP:   fmt.Sprintf("%s:%s", address, serviceDef.Port),
		Socket: fmt.Sprintf("ws://%s:%s", address, serviceDef.Port),
	}

	// Add the new service to the correct type list
	serviceMap.Services[serviceDef.Type] = append(serviceMap.Services[serviceDef.Type], service)

	// Save the updated service map
	if err := config.SaveServiceMapConfig(serviceMap); err != nil {
		return fmt.Errorf("failed to save service map: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Service '%s' (%s) added successfully.", serviceDef.ShortName, serviceDef.ID))
	ui.PrintInfo("Run 'dex build all' or 'dex build " + serviceDef.ShortName + "' to install and start it.")
	return nil
}
