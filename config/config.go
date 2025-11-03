package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Hardcoded Dexter environment layout
const (
	DexterRoot        = "~/Dexter"
	EasterCompanyRoot = "~/EasterCompany"
)

// Required subdirectories in ~/Dexter
var RequiredDexterDirs = []string{
	"bin",
	"config",
	"data",
	"logs",
	"models",
	"run",
}

// ServiceMapConfig represents the structure of service-map.json
type ServiceMapConfig struct {
	Doc          string                    `json:"_doc"`
	ServiceTypes []ServiceType             `json:"service_types"`
	Services     map[string][]ServiceEntry `json:"services"`
}

// serviceMapAlias is an alias for ServiceMapConfig used in custom marshaling
// to avoid an infinite recursion loop. It replaces the map with a struct
// to enforce a specific order on the "services" object keys in the JSON output.
type serviceMapAlias struct {
	Doc          string          `json:"_doc"`
	ServiceTypes []ServiceType   `json:"service_types"`
	Services     orderedServices `json:"services"`
}

// orderedServices is a helper struct to enforce a specific field order
// for the "services" object during JSON marshaling. The Go json package
// respects the order of fields in a struct.
type orderedServices struct {
	CLI []ServiceEntry `json:"cli"`
	FE  []ServiceEntry `json:"fe"`
	CS  []ServiceEntry `json:"cs"`
	BE  []ServiceEntry `json:"be"`
	TH  []ServiceEntry `json:"th"`
	OS  []ServiceEntry `json:"os"`
}

// MarshalJSON provides custom JSON marshaling for the ServiceMapConfig struct.
// It ensures that the keys of the "services" object are always in a predefined,
// logical order (cli, fe, cs, be, th, os) rather than the default alphabetical
// sorting of maps. It also sorts services within each category by port number.
func (s *ServiceMapConfig) MarshalJSON() ([]byte, error) {
	// Helper function to parse port from an address string (e.g., "localhost:8000")
	parsePort := func(addr string) int {
		if !strings.Contains(addr, ":") {
			return 0 // No port found
		}
		parts := strings.Split(addr, ":")
		portStr := parts[len(parts)-1]
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return 0 // Invalid port
		}
		return port
	}

	// Sort services within each category by port
	for _, serviceList := range s.Services {
		sort.SliceStable(serviceList, func(i, j int) bool {
			portI := parsePort(serviceList[i].HTTP)
			portJ := parsePort(serviceList[j].HTTP)
			// Services with no port go to the end
			if portI == 0 {
				return false
			}
			if portJ == 0 {
				return true
			}
			return portI < portJ
		})
	}

	alias := serviceMapAlias{
		Doc:          s.Doc,
		ServiceTypes: s.ServiceTypes,
		Services: orderedServices{
			CLI: s.Services["cli"],
			FE:  s.Services["fe"],
			CS:  s.Services["cs"],
			BE:  s.Services["be"],
			TH:  s.Services["th"],
			OS:  s.Services["os"],
		},
	}
	return json.Marshal(&alias)
}

// ServiceType defines a category of services
type ServiceType struct {
	Type        string `json:"type"`
	Label       string `json:"label"`
	Description string `json:"description"`
	MinPort     int    `json:"min_port"`
	MaxPort     int    `json:"max_port"`
}

// ServiceEntry represents a single service in the service map
type ServiceEntry struct {
	ID          string              `json:"id"`
	Repo        string              `json:"repo"`
	Source      string              `json:"source"`
	HTTP        string              `json:"http,omitempty"`
	Socket      string              `json:"socket,omitempty"`
	Credentials *ServiceCredentials `json:"credentials,omitempty"`
}

// ServiceCredentials holds connection credentials for services (e.g., Redis)
type ServiceCredentials struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

// ExpandPath expands ~ to the user's home directory
func ExpandPath(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	if path == "~" {
		return homeDir, nil
	}

	return filepath.Join(homeDir, path[2:]), nil
}

// EnsureDirectoryStructure creates required directories if they don't exist
func EnsureDirectoryStructure() error {
	// Ensure ~/Dexter exists
	dexterPath, err := ExpandPath(DexterRoot)
	if err != nil {
		return fmt.Errorf("failed to expand Dexter root path: %w", err)
	}

	if err := os.MkdirAll(dexterPath, 0755); err != nil {
		return fmt.Errorf("failed to create Dexter directory: %w", err)
	}

	// Ensure all required subdirectories exist
	for _, dir := range RequiredDexterDirs {
		dirPath := filepath.Join(dexterPath, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create %s directory: %w", dir, err)
		}
	}

	// Ensure ~/EasterCompany exists
	easterCompanyPath, err := ExpandPath(EasterCompanyRoot)
	if err != nil {
		return fmt.Errorf("failed to expand EasterCompany root path: %w", err)
	}

	if err := os.MkdirAll(easterCompanyPath, 0755); err != nil {
		return fmt.Errorf("failed to create EasterCompany directory: %w", err)
	}

	return nil
}

// EnsureConfigFiles creates and validates all config files.
func EnsureConfigFiles() error {
	// Service Map
	_, err := LoadServiceMapConfig()
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Creating default service-map.json...")
			if err := SaveServiceMapConfig(DefaultServiceMapConfig()); err != nil {
				return fmt.Errorf("failed to save default service-map.json: %w", err)
			}
		} else {
			return fmt.Errorf("failed to load service-map.json: %w", err)
		}
	}

	// Options
	_, err = LoadOptionsConfig()
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Creating default options.json...")
			if err := SaveOptionsConfig(DefaultOptionsConfig()); err != nil {
				return fmt.Errorf("failed to save default options.json: %w", err)
			}
		} else {
			return fmt.Errorf("failed to load options.json: %w", err)
		}
	}

	return nil
}

// LogFile returns a file handle to the dex-cli log file.
func LogFile() (*os.File, error) {
	logPath, err := ExpandPath(filepath.Join(DexterRoot, "logs", "dex-cli.log"))
	if err != nil {
		return nil, fmt.Errorf("failed to expand log file path: %w", err)
	}

	// Ensure the directory exists.
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open the file in append mode, create it if it doesn't exist.
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return file, nil
}

// HasSourceServices checks if the EasterCompany source directory exists.
func IsDevMode() bool {
	// Check if the source code directory exists
	path, err := ExpandPath("~/EasterCompany/dex-cli")
	if err != nil {
		return false
	}
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}
