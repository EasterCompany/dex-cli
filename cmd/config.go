package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/EasterCompany/dex-cli/config"
)

// Config displays the service-map.json configuration for a given service.
func Config(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("config command requires exactly one argument: the service short name (e.g., 'dex config event')")
	}

	serviceShortName := args[0]

	// 1. Resolve the short name to its full definition to get its ID and Type.
	def, err := config.Resolve(serviceShortName)
	if err != nil {
		return err // Error like "service alias '...' not found"
	}

	// 2. Load the user's service-map.json
	serviceMap, err := config.LoadServiceMapConfig()
	if err != nil {
		return fmt.Errorf("failed to load service-map.json: %w", err)
	}

	// 3. Find the corresponding entry in the map
	serviceEntries, ok := serviceMap.Services[def.Type]
	if !ok {
		// This handles cases where the service type (e.g., "cs") isn't even in the map
		return fmt.Errorf("service '%s' (ID: %s) not found in service-map.json", serviceShortName, def.ID)
	}

	for _, entry := range serviceEntries {
		if entry.ID == def.ID {
			// 4. Found it. Pretty-print the JSON.
			jsonData, err := json.MarshalIndent(entry, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal service config: %w", err)
			}
			fmt.Println(string(jsonData))
			return nil
		}
	}

	// 5. Not found in the map
	return fmt.Errorf("service '%s' (ID: %s) not found in service-map.json", serviceShortName, def.ID)
}
