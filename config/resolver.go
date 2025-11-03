package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Resolve finds a service's definition by its shortName (alias).
func Resolve(shortName string) (*ServiceDefinition, error) {
	def, ok := ServiceDefinitions[shortName]
	if !ok {
		return nil, fmt.Errorf("service alias '%s' not found", shortName)
	}
	return &def, nil
}

// ResolveByID finds a service's definition by its full ID (e.g., "dex-event-service").
func ResolveByID(id string) (*ServiceDefinition, error) {
	for _, def := range ServiceDefinitions {
		if def.ID == id {
			return &def, nil
		}
	}
	return nil, fmt.Errorf("service with ID '%s' not found", id)
}

// ResolveSystemdName finds a service's definition by its systemd name (e.g., "dex-event.service").
func ResolveSystemdName(systemdName string) (*ServiceDefinition, error) {
	for _, def := range ServiceDefinitions {
		if def.SystemdName == systemdName {
			return &def, nil
		}
	}
	return nil, fmt.Errorf("service with systemd name '%s' not found", systemdName)
}

// CheckSystemdService verifies if the service's .service file exists for the user.
func (def *ServiceDefinition) CheckSystemdService() (bool, error) {
	if def.SystemdName == "" {
		return false, nil // Not a systemd service
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false, fmt.Errorf("failed to get home directory: %w", err)
	}
	systemdServicePath := filepath.Join(homeDir, ".config", "systemd", "user", def.SystemdName)

	_, err = os.Stat(systemdServicePath)
	if err == nil {
		return true, nil // File exists
	}
	if os.IsNotExist(err) {
		return false, nil // File does not exist
	}
	return false, err // Other error (e.g., permissions)
}

// IsManageable returns true if a service is not 'cli' or 'os'.
func (def *ServiceDefinition) IsManageable() bool {
	return def.Type != "cli" && def.Type != "os"
}

// IsBuildable returns true if a service has source code and is not 'cli' or 'os'.
func (def *ServiceDefinition) IsBuildable() bool {
	return def.IsManageable() && def.Source != "N/A" && def.Repo != "N/A"
}
