package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ServiceDefinition defines the static properties for a known Dexter service.
// This is the new "source of truth" for all commands.
type ServiceDefinition struct {
	ShortName string // The alias, e.g., "event"
	ID        string // The full ID, e.g., "dex-event-service"
	Type      string // Service type, e.g., "cs", "be", "th"
	Port      string // Default port
	Repo      string // Full git repo URL
	Source    string // Full source code path
}

// ServiceDefinitions is the hardcoded map of all known Dexter services,
// keyed by their shorthand alias.
var ServiceDefinitions = map[string]ServiceDefinition{
	"event": {
		ShortName: "event",
		ID:        "dex-event-service",
		Type:      "cs",
		Port:      "8100",
		Repo:      "git@github.com:EasterCompany/dex-event-service",
		Source:    "~/EasterCompany/dex-event-service",
	},
	"model": {
		ShortName: "model",
		ID:        "dex-model-service",
		Type:      "cs",
		Port:      "8101",
		Repo:      "git@github.com:EasterCompany/dex-model-service",
		Source:    "~/EasterCompany/dex-model-service",
	},
	"chat": {
		ShortName: "chat",
		ID:        "dex-chat-service",
		Type:      "be",
		Port:      "8200",
		Repo:      "git@github.com:EasterCompany/dex-chat-service",
		Source:    "~/EasterCompany/dex-chat-service",
	},
	"tts": {
		ShortName: "tts",
		ID:        "dex-tts-service",
		Type:      "be",
		Port:      "8201",
		Repo:      "git@github.com:EasterCompany/dex-tts-service",
		Source:    "~/EasterCompany/dex-tts-service",
	},
	"stt": {
		ShortName: "stt",
		ID:        "dex-stt-service",
		Type:      "be",
		Port:      "8202",
		Repo:      "git@github.com:EasterCompany/dex-stt-service",
		Source:    "~/EasterCompany/dex-stt-service",
	},
	"discord": {
		ShortName: "discord",
		ID:        "dex-discord-service",
		Type:      "th",
		Port:      "8300",
		Repo:      "git@github.com:EasterCompany/dex-discord-service",
		Source:    "~/EasterCompany/dex-discord-service",
	},
	// --- Non-buildable/manageable services ---
	"cli": {
		ShortName: "cli",
		ID:        "dex-cli",
		Type:      "cli",
		Repo:      "git@github.com:EasterCompany/dex-cli",
		Source:    "~/EasterCompany/dex-cli",
	},
	"local-cache": {
		ShortName: "local-cache",
		ID:        "local-cache-0",
		Type:      "os",
		Port:      "6379",
	},
	"cloud-cache": {
		ShortName: "cloud-cache",
		ID:        "cloud-cache-0",
		Type:      "os",
		Port:      "6379",
	},
}

// DefaultServiceMapConfig returns the default service map configuration
func DefaultServiceMapConfig() *ServiceMapConfig {
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
					ID:     ServiceDefinitions["local-cache"].ID,
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
					ID:     ServiceDefinitions["cloud-cache"].ID,
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
					ID:     ServiceDefinitions["cli"].ID,
					Repo:   ServiceDefinitions["cli"].Repo,
					Source: ServiceDefinitions["cli"].Source,
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
