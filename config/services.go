// config/services.go
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

// ServiceDefinition is the universal, hardcoded definition for all services.
type ServiceDefinition struct {
	// ID is the full "legal name", e.g., "dex-event-service"
	ID string
	// ShortName is the alias, e.g., "event"
	ShortName string
	// SystemdName is the .service file name, e.g., "dex-event-service.service"
	SystemdName string
	// Type is the category, e.g., "cs"
	Type string
	// Repo is the git repository URL
	Repo string
	// Source is the local filesystem path
	Source string
	// Domain is the address to bind to
	Domain string
	// Port is the port to bind to
	Port string
	// Credentials for OS services like Redis
	Credentials *ServiceCredentials
	// Backup configuration
	Backup *BackupConfig
}

// BackupConfig defines the backup settings for a service.
type BackupConfig struct {
	// Artifacts is a list of paths to back up.
	Artifacts []string
}

// ToServiceEntry converts a hardcoded Definition to a ServiceEntry for saving.
func (def *ServiceDefinition) ToServiceEntry() ServiceEntry {
	return ServiceEntry{
		ID:          def.ID,
		ShortName:   def.ShortName,
		Repo:        def.Repo,
		Source:      def.Source,
		Domain:      def.Domain,
		Port:        def.Port,
		Credentials: def.Credentials,
	}
}

// GetHost returns the "domain:port" combination.
func (def *ServiceDefinition) GetHost() string {
	if def.Port == "" {
		return def.Domain
	}
	return fmt.Sprintf("%s:%s", def.Domain, def.Port)
}

// GetHTTP returns the full HTTP address.
func (def *ServiceDefinition) GetHTTP(path string) string {
	return fmt.Sprintf("http://%s%s", def.GetHost(), path)
}

// GetWS returns the full WebSocket address.
func (def *ServiceDefinition) GetWS(path string) string {
	return fmt.Sprintf("ws://%s%s", def.GetHost(), path)
}

// GetSystemdPath returns the full path to the user's systemd service file.
func (def *ServiceDefinition) GetSystemdPath() string {
	return fmt.Sprintf("~/.config/systemd/user/%s", def.SystemdName)
}

// GetLogPath returns the full path to the service's log file.
func (def *ServiceDefinition) GetLogPath() string {
	return fmt.Sprintf("~/Dexter/logs/%s.log", def.ID)
}

// GetBinaryPath returns the full path to the service's compiled binary.
func (def *ServiceDefinition) GetBinaryPath() string {
	if def.ShortName == "cli" {
		return "~/Dexter/bin/dex"
	}
	return fmt.Sprintf("~/Dexter/bin/%s", def.ID)
}

// IsManageable indicates if a service is managed by add/remove/build/start/stop.
// This excludes "cli" (managed by "update") and "os" (managed externally).
func (def *ServiceDefinition) IsManageable() bool {
	return def.Type != "cli" && def.Type != "os"
}

// IsBuildable indicates if a service is built from source.
func (def *ServiceDefinition) IsBuildable() bool {
	return def.Type == "cli" || def.IsManageable()
}

// serviceDefinitions holds the hardcoded master list of all services.
var serviceDefinitions = []ServiceDefinition{
	// CLI
	{
		ID:          "dex-cli",
		ShortName:   "cli",
		SystemdName: "",
		Type:        "cli",
		Repo:        "git@github.com:EasterCompany/dex-cli.git",
		Source:      "~/EasterCompany/dex-cli",
	},
	// Core Services (cs)
	{
		ID:          "dex-event-service",
		ShortName:   "event",
		SystemdName: "dex-event-service.service",
		Type:        "cs",
		Repo:        "git@github.com:EasterCompany/dex-event-service.git",
		Source:      "~/EasterCompany/dex-event-service",
		Domain:      "127.0.0.1", Port: "8100",
	},
	{
		ID:          "dex-model-service",
		ShortName:   "model",
		SystemdName: "dex-model-service.service",
		Type:        "cs",
		Repo:        "git@github.com:EasterCompany/dex-model-service.git",
		Source:      "~/EasterCompany/dex-model-service",
		Domain:      "127.0.0.1", Port: "8101",
	},
	// Backend Services (be)
	{
		ID:          "dex-chat-service",
		ShortName:   "chat",
		SystemdName: "dex-chat-service.service",
		Type:        "be",
		Repo:        "git@github.com:EasterCompany/dex-chat-service.git",
		Source:      "~/EasterCompany/dex-chat-service",
		Domain:      "127.0.0.1", Port: "8200",
	},
	{
		ID:          "dex-tts-service",
		ShortName:   "tts",
		SystemdName: "dex-tts-service.service",
		Type:        "be",
		Repo:        "git@github.com:EasterCompany/dex-tts-service.git",
		Source:      "~/EasterCompany/dex-tts-service",
		Domain:      "127.0.0.1", Port: "8201",
	},
	// 3rd Party (th)
	{
		ID:          "dex-discord-service",
		ShortName:   "discord",
		SystemdName: "dex-discord-service.service",
		Type:        "th",
		Repo:        "git@github.com:EasterCompany/dex-discord-service.git",
		Source:      "~/EasterCompany/dex-discord-service",
		Domain:      "127.0.0.1", Port: "8300",
	},
	// Frontend (fe)
	{
		ID:          "easter-company",
		ShortName:   "web",
		SystemdName: "easter-company.service", // Frontend now has a systemd service
		Type:        "fe",
		Repo:        "git@github.com:EasterCompany/easter.company.git",
		Source:      "~/EasterCompany/easter.company",
		Domain:      "127.0.0.1", Port: "8000",
	},
	// OS Services (os)
	{
		ID:        "local-cache-0",
		ShortName: "cache0",
		Type:      "os",
		Domain:    "127.0.0.1",
		Port:      "6379",
		Credentials: &ServiceCredentials{
			Username: "default",
			Password: "",
		},
	},
	{
		ID:        "cloud-cache-0",
		ShortName: "cache1",
		Type:      "os",
		Domain:    "redis-18309.c243.eu-west-1-3.ec2.redns.redis-cloud.com",
		Port:      "18309",
		Credentials: &ServiceCredentials{
			Username: "default",
			Password: "Mi9mm7DfzWnP59PiMyCuuMr2AC7OMBLr",
		},
	},
	{
		ID:        "cloud-cache-1",
		ShortName: "cache2",
		Type:      "os",
		Domain:    "redis-13647.c269.eu-west-1-3.ec2.redns.redis-cloud.com",
		Port:      "13647",
		Credentials: &ServiceCredentials{
			Username: "default",
			Password: "J3zfv4SPss8DjxIiefRQwOOokT0yHVqO",
		},
	},
	{
		ID:        "local-ollama-0",
		ShortName: "ollama",
		Type:      "os",
		Domain:    "127.0.0.1",
		Port:      "11434",
		Credentials: &ServiceCredentials{
			Username: "",
			Password: "",
		},
	},
}

// GetAllServices returns a copy of the master service list.
func GetAllServices() []ServiceDefinition {
	// Return a copy to prevent modification of the original slice
	defs := make([]ServiceDefinition, len(serviceDefinitions))
	copy(defs, serviceDefinitions)
	return defs
}

// GetManageableServices returns all services that can be managed (not cli or os).
func GetManageableServices() []ServiceDefinition {
	defs := []ServiceDefinition{}
	for _, def := range GetAllServices() {
		if def.IsManageable() {
			defs = append(defs, def)
		}
	}
	return defs
}

// GetBuildableServices returns all services that are built from source.
func GetBuildableServices() []ServiceDefinition {
	defs := []ServiceDefinition{}
	for _, def := range GetAllServices() {
		if def.IsBuildable() {
			defs = append(defs, def)
		}
	}
	return defs
}

// GetServiceDefinition returns a service definition by its ID.
func GetServiceDefinition(id string) ServiceDefinition {
	for _, def := range serviceDefinitions {
		if def.ID == id {
			return def
		}
	}
	return ServiceDefinition{}
}

//
// service-map.json struct definitions and helpers
//

// ServiceMapConfig represents the structure of service-map.json
type ServiceMapConfig struct {
	Doc      string                    `json:"_doc"`
	Services map[string][]ServiceEntry `json:"services"`
}

// serviceMapAlias is for custom JSON marshaling to order keys
type serviceMapAlias struct {
	Doc      string          `json:"_doc"`
	Services orderedServices `json:"services"`
}

// orderedServices helps enforce key order in JSON
type orderedServices struct {
	CLI []ServiceEntry `json:"cli"`
	FE  []ServiceEntry `json:"fe"`
	CS  []ServiceEntry `json:"cs"`
	BE  []ServiceEntry `json:"be"`
	TH  []ServiceEntry `json:"th"`
	OS  []ServiceEntry `json:"os"`
}

// MarshalJSON provides custom marshaling for ServiceMapConfig
func (s *ServiceMapConfig) MarshalJSON() ([]byte, error) {
	parsePort := func(portStr string) int {
		port, _ := strconv.Atoi(portStr)
		return port
	}

	// Sort services within each category by port
	for _, serviceList := range s.Services {
		sort.SliceStable(serviceList, func(i, j int) bool {
			portI := parsePort(serviceList[i].Port)
			portJ := parsePort(serviceList[j].Port)
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
		Doc: s.Doc,
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

// ServiceEntry represents a single service in the service map
type ServiceEntry struct {
	ID          string              `json:"id"`
	ShortName   string              `json:"short_name,omitempty"`
	Repo        string              `json:"repo"`
	Source      string              `json:"source"`
	Domain      string              `json:"domain,omitempty"`
	Port        string              `json:"port,omitempty"`
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
	// Create the map from the hardcoded definitions
	services := make(map[string][]ServiceEntry)
	for _, def := range serviceDefinitions {
		// Only "cli", "fe" and "os" types are in the map by default
		if def.Type == "cli" || def.Type == "fe" || def.Type == "os" {
			services[def.Type] = append(services[def.Type], def.ToServiceEntry())
		} else {
			// Ensure other types have at least an empty slice
			if _, ok := services[def.Type]; !ok {
				services[def.Type] = []ServiceEntry{}
			}
		}
	}

	return &ServiceMapConfig{
		Doc:      "Maps service names to their configurations",
		Services: services,
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

	return os.WriteFile(serviceMapPath, data, 0o644)
}
