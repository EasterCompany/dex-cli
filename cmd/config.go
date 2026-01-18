package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/EasterCompany/dex-cli/config"
	"github.com/EasterCompany/dex-cli/ui"
)

// Config displays the service-map.json configuration for a given service, or manages it.
func Config(args []string) error {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			ui.PrintHeader("Config Command Help")
			ui.PrintInfo("Usage:")
			ui.PrintInfo("  dex config <service> [field...]   View service-map.json configuration.")
			ui.PrintInfo("  dex config get <service> <key>    Get a runtime option value.")
			ui.PrintInfo("  dex config set <service> <key> <val> Set a runtime option value.")
			ui.PrintInfo("  dex config reset                  Reset service-map.json to defaults.")
			return nil
		}
	}

	if len(args) == 0 {
		return fmt.Errorf("config command requires arguments")
	}

	subcommand := args[0]

	if subcommand == "reset" {
		return resetConfig()
	}

	if subcommand == "set" {
		if len(args) < 4 {
			return fmt.Errorf("usage: dex config set <service> <key> <value>")
		}
		return setServiceOption(args[1], args[2], args[3])
	}

	if subcommand == "get" {
		if len(args) < 3 {
			return fmt.Errorf("usage: dex config get <service> <key>")
		}
		return getServiceOption(args[1], args[2])
	}

	// Legacy behavior: View service-map.json
	return viewServiceConfig(args)
}

func setServiceOption(service, key, value string) error {
	opts, err := config.LoadOptionsConfig()
	if err != nil {
		return fmt.Errorf("failed to load options: %w", err)
	}

	if opts.Services == nil {
		opts.Services = make(map[string]map[string]interface{})
	}
	if opts.Services[service] == nil {
		opts.Services[service] = make(map[string]interface{})
	}

	opts.Services[service][key] = value

	if err := config.SaveOptionsConfig(opts); err != nil {
		return fmt.Errorf("failed to save options: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Set %s.%s = %s", service, key, value))
	return nil
}

func getServiceOption(service, key string) error {
	opts, err := config.LoadOptionsConfig()
	if err != nil {
		return fmt.Errorf("failed to load options: %w", err)
	}

	if opts.Services != nil && opts.Services[service] != nil {
		if val, ok := opts.Services[service][key]; ok {
			fmt.Println(val)
			return nil
		}
	}

	return fmt.Errorf("option not set")
}

func viewServiceConfig(args []string) error {
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

func resetConfig() error {
	ui.PrintInfo("Resetting service-map.json to default configuration...")
	defaultServiceMap := config.DefaultServiceMapConfig()
	if err := config.SaveServiceMapConfig(defaultServiceMap); err != nil {
		return fmt.Errorf("failed to save default service-map.json: %w", err)
	}
	ui.PrintSuccess("service-map.json reset successfully. Services will use updated defaults on next build/start.")
	return nil
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
