package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DefaultServiceMapConfig returns the default service map configuration
func DefaultServiceMapConfig() *ServiceMapConfig {
	return &ServiceMapConfig{
		Doc: "Maps service names to their configurations",
		ServiceTypes: []ServiceType{
			{
				Type:        "cli",
				Label:       "CLI",
				Description: "CLI applications",
				MinPort:     0,
				MaxPort:     0,
			},
			{
				Type:        "fe",
				Label:       "Front Ends",
				Description: "Frontend services",
				MinPort:     8000,
				MaxPort:     8099,
			},
			{
				Type:        "cs",
				Label:       "Core Services",
				Description: "Essential Dexter services",
				MinPort:     8100,
				MaxPort:     8199,
			},
			{
				Type:        "be",
				Label:       "Back Ends",
				Description: "Backend services",
				MinPort:     8200,
				MaxPort:     8299,
			},
			{
				Type:        "th",
				Label:       "3rd Party",
				Description: "3rd party integration services",
				MinPort:     8300,
				MaxPort:     8399,
			},
			{
				Type:        "os",
				Label:       "Other Services",
				Description: "Other services",
				MinPort:     0,
				MaxPort:     0,
			},
		},
		Services: map[string][]ServiceEntry{
			"fe": {},
			"cs": {},
			"be": {},
			"th": {},
			"os": {
				{
					ID:     "local-cache-0",
					Repo:   "N/A",
					Source: "N/A",
					HTTP:   "localhost:6379",
					Socket: "N/A",
					Credentials: &ServiceCredentials{
						Username: "default",
						Password: "",
						DB:       0,
					},
				},
				{
					ID:     "cloud-cache-0",
					Repo:   "N/A",
					Source: "N/A",
					HTTP:   "cloud.easter.company:6379",
					Socket: "N/A",
					Credentials: &ServiceCredentials{
						Username: "user",
						Password: "password",
						DB:       0,
					},
				},
			},
			"cli": {
				{
					ID:     "dex-cli",
					Repo:   "https://github.com/EasterCompany/dex-cli",
					Source: "~/EasterCompany/dex-cli",
				},
			},
		},
	}
}

// LoadServiceMapConfig loads the service-map.json file
func LoadServiceMapConfig() (*ServiceMapConfig, error) {
	serviceMapPath, err := ExpandPath(filepath.Join(DexterRoot, "config", "service-map.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to expand service-map.json path: %w", err)
	}

	data, err := os.ReadFile(serviceMapPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("failed to read service-map.json: %w", err)
	}

	var serviceMap ServiceMapConfig
	if err := json.Unmarshal(data, &serviceMap); err != nil {
		return nil, fmt.Errorf("failed to parse service-map.json: %w", err)
	}

	return &serviceMap, nil
}

// SaveServiceMapConfig saves the service-map.json file
func SaveServiceMapConfig(serviceMap *ServiceMapConfig) error {
	serviceMapPath, err := ExpandPath(filepath.Join(DexterRoot, "config", "service-map.json"))
	if err != nil {
		return fmt.Errorf("failed to expand service-map.json path: %w", err)
	}

	data, err := json.MarshalIndent(serviceMap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal service-map.json: %w", err)
	}

	return os.WriteFile(serviceMapPath, data, 0644)
}
