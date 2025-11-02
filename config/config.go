package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Hardcoded Dexter environment layout
const (
	DexterRoot        = "~/Dexter"
	EasterCompanyRoot = "~/EasterCompany"
)

// Required subdirectories in ~/Dexter
var RequiredDexterDirs = []string{
	"config",
	"models",
	"bin",
	"run",
	"logs",
}

// ServiceMapConfig represents the structure of service-map.json
type ServiceMapConfig struct {
	Doc          string                    `json:"_doc"`
	ServiceTypes []ServiceType             `json:"service_types"`
	Services     map[string][]ServiceEntry `json:"services"`
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
	ID          string             `json:"id"`
	Source      string             `json:"source"`
	Repo        string             `json:"repo"`
	Addr        string             `json:"addr"`
	Socket      string             `json:"socket"`
	Credentials *ServiceCredentials `json:"credentials,omitempty"`
}

// ServiceCredentials holds connection credentials for services (e.g., Redis)
type ServiceCredentials struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
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

// LoadServiceMap reads and parses the service-map.json file
func LoadServiceMap() (*ServiceMapConfig, error) {
	serviceMapPath, err := ExpandPath(filepath.Join(DexterRoot, "config", "service-map.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to expand service-map path: %w", err)
	}

	data, err := os.ReadFile(serviceMapPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read service-map.json: %w", err)
	}

	var serviceMap ServiceMapConfig
	if err := json.Unmarshal(data, &serviceMap); err != nil {
		return nil, fmt.Errorf("failed to parse service-map.json: %w", err)
	}

	return &serviceMap, nil
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
