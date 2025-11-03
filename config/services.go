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
