package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ServiceDefinition is the new universal source of truth for all services.
// It is keyed by the shortName (e.g., "event") in the ServiceDefinitions map.
type ServiceDefinition struct {
	ShortName   string // "event"
	ID          string // "dex-event-service"
	SystemdName string // "dex-event.service"
	Type        string // "cs"
	Port        string // "8100"
	Repo        string // "git@github.com:EasterCompany/dex-event-service"
	Source      string // "~/EasterCompany/dex-event-service"
}

// GetLogPath returns the expected log file path for the service.
func (def *ServiceDefinition) GetLogPath() (string, error) {
	return ExpandPath(filepath.Join(DexterLogs, fmt.Sprintf("%s.log", def.ID)))
}

// ServiceDefinitions is the new hardcoded map that drives all service commands.
// All services MUST be defined here by their shortName.
var ServiceDefinitions = map[string]ServiceDefinition{
	// CLI
	"cli": {
		ShortName:   "cli",
		ID:          "dex-cli",
		SystemdName: "", // No systemd service
		Type:        "cli",
		Port:        "",
		Repo:        "git@github.com:EasterCompany/dex-cli",
		Source:      "~/EasterCompany/dex-cli",
	},
	// Core Services (cs)
	"event": {
		ShortName:   "event",
		ID:          "dex-event-service",
		SystemdName: "dex-event.service",
		Type:        "cs",
		Port:        "8100",
		Repo:        "git@github.com:EasterCompany/dex-event-service",
		Source:      "~/EasterCompany/dex-event-service",
	},
	"model": {
		ShortName:   "model",
		ID:          "dex-model-service",
		SystemdName: "dex-model.service",
		Type:        "cs",
		Port:        "8101",
		Repo:        "git@github.com:EasterCompany/dex-model-service",
		Source:      "~/EasterCompany/dex-model-service",
	},
	// Back Ends (be)
	"chat": {
		ShortName:   "chat",
		ID:          "dex-chat-service",
		SystemdName: "dex-chat.service",
		Type:        "be",
		Port:        "8200",
		Repo:        "git@github.com:EasterCompany/dex-chat-service",
		Source:      "~/EasterCompany/dex-chat-service",
	},
	"tts": {
		ShortName:   "tts",
		ID:          "dex-tts-service",
		SystemdName: "dex-tts.service",
		Type:        "be",
		Port:        "8201",
		Repo:        "git@github.com:EasterCompany/dex-tts-service",
		Source:      "~/EasterCompany/dex-tts-service",
	},
	"stt": {
		ShortName:   "stt",
		ID:          "dex-stt-service",
		SystemdName: "dex-stt.service",
		Type:        "be",
		Port:        "8202",
		Repo:        "git@github.com:EasterCompany/dex-stt-service",
		Source:      "~/EasterCompany/dex-stt-service",
	},
	// 3rd Party (th)
	"discord": {
		ShortName:   "discord",
		ID:          "dex-discord-service",
		SystemdName: "dex-discord.service",
		Type:        "th",
		Port:        "8300",
		Repo:        "git@github.com:EasterCompany/dex-discord-service",
		Source:      "~/EasterCompany/dex-discord-service",
	},
	// Other Services (os)
	"local-cache": {
		ShortName:   "local-cache",
		ID:          "local-cache-0",
		SystemdName: "redis.service", // Assumes a system-wide redis, or user-level
		Type:        "os",
		Port:        "6379",
		Repo:        "N/A",
		Source:      "N/A",
	},
	"cloud-cache": {
		ShortName:   "cloud-cache",
		ID:          "cloud-cache-0",
		SystemdName: "", // No systemd service
		Type:        "os",
		Port:        "6379", // Example port, address is different
		Repo:        "N/A",
		Source:      "N/A",
	},
}

// GetManageableServices returns a list of all service definitions that are not 'cli' or 'os'.
func GetManageableServices() []ServiceDefinition {
	var services []ServiceDefinition
	for _, def := range ServiceDefinitions {
		if def.IsManageable() {
			services = append(services, def)
		}
	}
	return services
}

// GetBuildableServices returns a list of all service definitions that are buildable.
func GetBuildableServices() []ServiceDefinition {
	var services []ServiceDefinition
	for _, def := range ServiceDefinitions {
		if def.IsBuildable() {
			services = append(services, def)
		}
	}
	return services
}

// --- service-map.json struct definitions ---
// These are still needed to read/write the service-map.json config file,
// which represents the *user's specific installation*.

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
	ID          string              `json:"id"` // "dex-event-service"
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
	// Find the definitions for cli, local-cache, and cloud-cache to add them by default
	cliDef := ServiceDefinitions["cli"]
	localCacheDef := ServiceDefinitions["local-cache"]
	cloudCacheDef := ServiceDefinitions["cloud-cache"]

	return &ServiceMapConfig{
		Doc: "Maps service names to their configurations",
		ServiceTypes: []ServiceType{
			{Type: "cli", Label: "CLI", Description: "CLI applications", MinPort: 0, MaxPort: 0},
			{Type: "fe", Label: "Front Ends", Description: "Frontend services", MinPort: 8000, MaxPort: 8099},
			{Type: "cs", Label: "Core Services", Description: "Essential Dexter services", MinPort: 8100, MaxPort: 8199},
			{Type: "be", Label: "Back Ends", Description: "Backend services", MinPort: 8200, MaxPort: 8299},
			{Type: "th", Label: "3rd Party", Description: "3rd party integration services", MinPort: 8300, MaxPort: 8399},
			{Type: "os", Label: "Other Services", Description: "Other services", MinPort: 0, MaxPort: 0},
		},
		Services: map[string][]ServiceEntry{
			"fe": {},
			"cs": {},
			"be": {},
			"th": {},
			"os": {
				{
					ID:     localCacheDef.ID,
					Repo:   localCacheDef.Repo,
					Source: localCacheDef.Source,
					HTTP:   "localhost:6379",
					Socket: "N/A",
					Credentials: &ServiceCredentials{
						Username: "default",
						Password: "",
						DB:       0,
					},
				},
				{
					ID:     cloudCacheDef.ID,
					Repo:   cloudCacheDef.Repo,
					Source: cloudCacheDef.Source,
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
					ID:     cliDef.ID,
					Repo:   cliDef.Repo,
					Source: cliDef.Source,
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
