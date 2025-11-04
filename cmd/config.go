package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/EasterCompany/dex-cli/config"
)

// Config displays the service-map.json configuration for a given service.
func Config(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("config command requires at least one argument: the service short name (e.g., 'dex config event')")
	}

	serviceShortName := args[0]
	fieldPath := args[1:]

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
			// 4. Found it.
			// If no field path, pretty-print the whole JSON.
			if len(fieldPath) == 0 {
				jsonData, err := json.MarshalIndent(entry, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal service config: %w", err)
				}
				fmt.Println(string(jsonData))
				return nil
			}

			// 5. Field path provided. Convert struct to map to traverse it.
			var data map[string]interface{}
			jsonData, err := json.Marshal(entry)
			if err != nil {
				return fmt.Errorf("failed to marshal service config: %w", err)
			}
			if err := json.Unmarshal(jsonData, &data); err != nil {
				return fmt.Errorf("failed to unmarshal config to map: %w", err)
			}

			// 6. Traverse the map to find the value
			value, err := findJSONValue(data, fieldPath)
			if err != nil {
				return err
			}

			// 7. Print the final value
			return printJSONValue(value)
		}
	}

	// 8. Not found in the map
	return fmt.Errorf("service '%s' (ID: %s) not found in service-map.json", serviceShortName, def.ID)
}

// findJSONValue recursively traverses a map to find a value at a given path.
// Note: JSON keys are lowercase due to the `json:"..."` tags in the struct.
func findJSONValue(data map[string]interface{}, path []string) (interface{}, error) {
	if len(path) == 0 {
		return data, nil // Should not happen if called from Config()
	}

	key := path[0]
	value, ok := data[key]
	if !ok {
		return nil, fmt.Errorf("field '%s' not found", key)
	}

	if len(path) == 1 {
		return value, nil // Reached the end of the path
	}

	// Go deeper
	nextMap, ok := value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("field '%s' is not an object, cannot access '%s'", key, path[1])
	}
	return findJSONValue(nextMap, path[1:])
}

// printJSONValue formats and prints the interface{} value.
func printJSONValue(value interface{}) error {
	if value == nil {
		fmt.Println("null")
		return nil
	}

	switch v := value.(type) {
	case string:
		fmt.Println(v)
	case float64:
		// Check if it's actually an int
		if v == float64(int64(v)) {
			fmt.Println(int64(v))
		} else {
			fmt.Println(v)
		}
	case bool:
		fmt.Println(v)
	default:
		// For nested objects or arrays, pretty-print them
		jsonData, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal field value: %w", err)
		}
		// Don't print empty objects/arrays
		s := string(jsonData)
		if s == "{}" || s == "[]" {
			return nil
		}
		fmt.Println(s)
	}
	return nil
}
