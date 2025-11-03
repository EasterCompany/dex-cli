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
	if len(args) > 0 {
		return fmt.Errorf("add command takes no arguments")
	}

	reader := bufio.NewReader(os.Stdin)

	// Load existing service map
	serviceMap, err := config.LoadServiceMapConfig()
	if err != nil {
		if os.IsNotExist(err) {
			serviceMap = config.DefaultServiceMapConfig()
		} else {
			return fmt.Errorf("failed to load service map: %w", err)
		}
	}

	// Create a map of already-added services for quick lookup
	addedServices := make(map[string]bool)
	for _, services := range serviceMap.Services {
		for _, service := range services {
			addedServices[service.ID] = true
		}
	}

	// Find services that are defined but not yet added
	var availableServices []config.ServiceDefinition
	for _, def := range config.ServiceDefinitions {
		// Only show services that are manageable (not 'cli' or 'os')
		if !def.IsManageable() {
			continue
		}
		if _, ok := addedServices[def.ID]; !ok {
			availableServices = append(availableServices, def)
		}
	}

	if len(availableServices) == 0 {
		ui.PrintInfo("All manageable services are already added.")
		return nil
	}

	// Sort for consistent display
	sort.Slice(availableServices, func(i, j int) bool {
		return availableServices[i].ShortName < availableServices[j].ShortName
	})

	fmt.Println("Available services to add:")
	for i, def := range availableServices {
		fmt.Printf("  %d: %s (Port: %s, Type: %s)\n", i+1, def.ShortName, def.Port, def.Type)
	}

	fmt.Print("Enter number(s) to add (e.g., '1' or '1, 2'): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		fmt.Println("No selection made.")
		return nil
	}

	selections := strings.Split(input, ",")
	servicesAdded := 0

	for _, s := range selections {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}

		idx, err := strconv.Atoi(s)
		if err != nil || idx < 1 || idx > len(availableServices) {
			ui.PrintError(fmt.Sprintf("Invalid selection: '%s'. Skipping.", s))
			continue
		}

		def := availableServices[idx-1]

		// Create the new ServiceEntry
		service := config.ServiceEntry{
			ID:     def.ID,
			Repo:   def.Repo,
			Source: def.Source,
		}

		// Add HTTP/Socket info if it's not a CLI/OS service
		if def.Type != "cli" && def.Type != "os" {
			address := "0.0.0.0" // Default address
			service.HTTP = fmt.Sprintf("%s:%s", address, def.Port)
			service.Socket = fmt.Sprintf("ws://%s:%s", address, def.Port)
		}

		// Add the new service
		serviceMap.Services[def.Type] = append(serviceMap.Services[def.Type], service)
		ui.PrintInfo(fmt.Sprintf("Adding '%s' to service map...", def.ShortName))
		servicesAdded++
	}

	if servicesAdded == 0 {
		return nil
	}

	// Save the updated service map
	if err := config.SaveServiceMapConfig(serviceMap); err != nil {
		return fmt.Errorf("failed to save service map: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Successfully added %d service(s).", servicesAdded))
	ui.PrintInfo("Run 'dex build all' or 'dex build <service>' to build and install.")
	return nil
}
