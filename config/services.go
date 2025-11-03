package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// ServiceDefinition is the universal, hardcoded definition for a service.
type ServiceDefinition struct {
	ID          string // e.g., "dex-event-service"
	ShortName   string // e.g., "event"
	Type        string // e.g., "cs"
	Port        string // e.g., "8100"
	Repo        string // e.g., "git@github.com:EasterCompany/dex-event-service"
	Source      string // e.g., "~/EasterCompany/dex-event-service"
	SystemdName string // e.g., "dex-event-service.service"
	LogPath     string // e.g., "~/Dexter/logs/dex-event-service.log"
}

// GetSystemdPath returns the full, absolute path to the user's systemd service file.
func (s *ServiceDefinition) GetSystemdPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback, though this should rarely fail
		homeDir = os.Getenv("HOME")
	}
	return filepath.Join(homeDir, ".config", "systemd", "user", s.SystemdName)
}

// GetLogPath returns the *un-expanded* log path (still contains ~).
// Use config.ExpandPath(def.GetLogPath()) to get the absolute path.
func (s *ServiceDefinition) GetLogPath() string {
	return s.LogPath
}

// GetBinaryPath returns the full, absolute path to where the service binary should be.
func (s *ServiceDefinition) GetBinaryPath() string {
	// DexterBin is a const in config.go
	expandedPath, _ := ExpandPath(DexterBin)
	return filepath.Join(expandedPath, s.ID)
}

// IsManageable returns true if the service is intended to be managed by
// commands like `start`, `stop`, `build`, `logs`.
func (s *ServiceDefinition) IsManageable() bool {
	return s.Type != "cli" && s.Type != "os"
}

// GetAllServices returns the universal list of all known Dexter services.
// This is the single source of truth for service definitions.
func GetAllServices() []ServiceDefinition {
	services := []ServiceDefinition{}
	serviceDefs := map[string]struct {
		Type string
		Port string
	}{
		"event":   {Type: "cs", Port: "8100"},
		"model":   {Type: "cs", Port: "8101"},
		"chat":    {Type: "be", Port: "8200"},
		"tts":     {Type: "be", Port: "8201"},
		"stt":     {Type: "be", Port: "8202"},
		"discord": {Type: "th", Port: "8300"},
		// Add new manageable services here
	}

	for shortName, d := range serviceDefs {
		id := fmt.Sprintf("dex-%s-service", shortName)
		services = append(services, ServiceDefinition{
			ID:          id,
			ShortName:   shortName,
			Type:        d.Type,
			Port:        d.Port,
			Repo:        fmt.Sprintf("git@github.com:EasterCompany/%s.git", id),
			Source:      fmt.Sprintf("~/EasterCompany/%s", id),
			SystemdName: fmt.Sprintf("%s.service", id),
			LogPath:     fmt.Sprintf("~/Dexter/logs/%s.log", id),
		})
	}

	// Add non-manageable services
	services = append(services, ServiceDefinition{
		ID:        "dex-cli",
		ShortName: "cli",
		Type:      "cli",
		Source:    "~/EasterCompany/dex-cli",
	})
	services = append(services, ServiceDefinition{
		ID:        "local-cache-0",
		ShortName: "local-cache",
		Type:      "os",
	})
	services = append(services, ServiceDefinition{
		ID:        "cloud-cache-0",
		ShortName: "cloud-cache",
		Type:      "os",
	})

	return services
}
